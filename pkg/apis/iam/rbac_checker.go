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
	// InvalidateCache removes the cached permission entry for a user.
	// Call this when a user's role bindings change.
	InvalidateCache(userID int64)
	// InvalidateCacheAll removes all cached permission entries.
	// Call this when role permission rules change (affects all users).
	InvalidateCacheAll()
}

// RBACChecker implements PermissionChecker using the shared permission cache backed by RoleBindingStore.
type RBACChecker struct {
	rbStore RoleBindingStore
	sfGroup singleflight.Group
}

// NewRBACChecker creates a new checker with the given store.
func NewRBACChecker(rbStore RoleBindingStore) *RBACChecker {
	return &RBACChecker{rbStore: rbStore}
}

func (c *RBACChecker) CheckPermission(ctx context.Context, userID int64, permCode string, scope string, workspaceID, namespaceID int64) (bool, error) {
	entry, err := c.getOrLoad(ctx, userID)
	if err != nil {
		return false, err
	}
	return entry.HasPermission(permCode, scope, workspaceID, namespaceID), nil
}

func (c *RBACChecker) IsPlatformAdmin(ctx context.Context, userID int64) (bool, error) {
	entry, err := c.getOrLoad(ctx, userID)
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

func (c *RBACChecker) InvalidateCache(userID int64) {
	sharedPermCache.Invalidate(userID)
}

func (c *RBACChecker) InvalidateCacheAll() {
	sharedPermCache.InvalidateAll()
}

// getOrLoad returns the cached entry for a user, loading from DB on cache miss.
// Uses singleflight to deduplicate concurrent loads for the same user.
func (c *RBACChecker) getOrLoad(ctx context.Context, userID int64) (*UserPermissionEntry, error) {
	if entry := sharedPermCache.Get(userID); entry != nil {
		return entry, nil
	}
	key := strconv.FormatInt(userID, 10)
	v, err, _ := c.sfGroup.Do(key, func() (any, error) {
		// Double-check cache after acquiring the singleflight slot
		if entry := sharedPermCache.Get(userID); entry != nil {
			return entry, nil
		}
		entry, err := c.loadUserEntry(ctx, userID)
		if err != nil {
			return nil, err
		}
		sharedPermCache.Set(userID, entry)
		return entry, nil
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
