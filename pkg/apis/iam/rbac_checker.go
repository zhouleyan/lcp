package iam

import (
	"context"
	"fmt"
	"strconv"

	"golang.org/x/sync/singleflight"
)

// PermissionChecker defines the interface for RBAC permission checking.
type PermissionChecker interface {
	// CheckPermission checks whether a user has the given permission at the specified scope.
	CheckPermission(ctx context.Context, userID int64, permCode string, scope string, workspaceID, namespaceID int64) (bool, error)
	// IsPlatformAdmin checks whether a user has the platform-admin super permission (*:*).
	IsPlatformAdmin(ctx context.Context, userID int64) (bool, error)
	// GetAccessibleWorkspaceIDs returns workspace IDs the user has any role binding for.
	GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error)
	// GetAccessibleNamespaceIDs returns namespace IDs the user has any role binding for.
	GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error)
}

// UserPermissionEntry holds permission rules for a single user,
// organized by scope level for efficient matching.
type UserPermissionEntry struct {
	IsPlatformAdmin bool               // true if platformRules contains "*:*" (fast short-circuit)
	PlatformRules   []string           // patterns from platform-scoped bindings
	WorkspaceRules  map[int64][]string // workspaceID → patterns
	NamespaceRules  map[int64][]string // namespaceID → patterns
}

// HasPermission checks whether this entry grants the given permission code
// at the specified scope/resource level, following scope chain inheritance:
// platform rules apply everywhere, workspace rules apply to workspace and its namespaces.
func (e *UserPermissionEntry) HasPermission(code, scope string, wsID, nsID int64) bool {
	// 1. Platform-level rules apply to all scopes
	for _, pattern := range e.PlatformRules {
		if MatchPermission(pattern, code) {
			return true
		}
	}
	// 2. Workspace-level rules apply to workspace and namespace scopes
	if (scope == ScopeWorkspace || scope == ScopeNamespace) && wsID > 0 {
		for _, pattern := range e.WorkspaceRules[wsID] {
			if MatchPermission(pattern, code) {
				return true
			}
		}
	}
	// 3. Namespace-level rules apply to namespace scope only
	if scope == ScopeNamespace && nsID > 0 {
		for _, pattern := range e.NamespaceRules[nsID] {
			if MatchPermission(pattern, code) {
				return true
			}
		}
	}
	return false
}

// RBACChecker implements PermissionChecker backed by RoleBindingStore.
// Uses singleflight to deduplicate concurrent DB loads for the same user.
type RBACChecker struct {
	rbStore RoleBindingStore
	sfGroup singleflight.Group
}

// NewRBACChecker creates a new checker with the given store.
func NewRBACChecker(rbStore RoleBindingStore) *RBACChecker {
	return &RBACChecker{rbStore: rbStore}
}

func (c *RBACChecker) CheckPermission(ctx context.Context, userID int64, permCode string, scope string, workspaceID, namespaceID int64) (bool, error) {
	entry, err := c.loadEntry(ctx, userID)
	if err != nil {
		return false, err
	}
	return entry.HasPermission(permCode, scope, workspaceID, namespaceID), nil
}

func (c *RBACChecker) IsPlatformAdmin(ctx context.Context, userID int64) (bool, error) {
	entry, err := c.loadEntry(ctx, userID)
	if err != nil {
		return false, err
	}
	return entry.IsPlatformAdmin, nil
}

func (c *RBACChecker) GetAccessibleWorkspaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	return c.rbStore.GetAccessibleWorkspaceIDs(ctx, userID)
}

func (c *RBACChecker) GetAccessibleNamespaceIDs(ctx context.Context, userID int64) ([]int64, error) {
	return c.rbStore.GetAccessibleNamespaceIDs(ctx, userID)
}

// loadEntry loads permission rules for a user from the database.
// Uses singleflight to deduplicate concurrent loads for the same user.
func (c *RBACChecker) loadEntry(ctx context.Context, userID int64) (*UserPermissionEntry, error) {
	key := strconv.FormatInt(userID, 10)
	v, err, _ := c.sfGroup.Do(key, func() (any, error) {
		return c.loadUserEntry(ctx, userID)
	})
	if err != nil {
		return nil, err
	}
	return v.(*UserPermissionEntry), nil
}

// loadUserEntry loads all permission rules for a user from the database
// and organizes them into a UserPermissionEntry by scope.
func (c *RBACChecker) loadUserEntry(ctx context.Context, userID int64) (*UserPermissionEntry, error) {
	rows, err := c.rbStore.LoadUserPermissionRules(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load permission rules for user %d: %w", userID, err)
	}

	entry := &UserPermissionEntry{
		WorkspaceRules: make(map[int64][]string),
		NamespaceRules: make(map[int64][]string),
	}

	for _, row := range rows {
		switch row.Scope {
		case ScopePlatform:
			entry.PlatformRules = append(entry.PlatformRules, row.Pattern)
			if row.Pattern == "*:*" {
				entry.IsPlatformAdmin = true
			}
		case ScopeWorkspace:
			if row.WorkspaceID != nil {
				entry.WorkspaceRules[*row.WorkspaceID] = append(entry.WorkspaceRules[*row.WorkspaceID], row.Pattern)
			}
		case ScopeNamespace:
			if row.NamespaceID != nil {
				entry.NamespaceRules[*row.NamespaceID] = append(entry.NamespaceRules[*row.NamespaceID], row.Pattern)
			}
		}
	}

	return entry, nil
}
