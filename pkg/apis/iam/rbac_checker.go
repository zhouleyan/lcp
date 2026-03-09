package iam

import (
	"context"
	"fmt"
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

// RBACChecker implements PermissionChecker using a TTL cache backed by RoleBindingStore.
type RBACChecker struct {
	rbStore RoleBindingStore
	cache   *PermissionCache
}

// NewRBACChecker creates a new checker with the given store and cache.
func NewRBACChecker(rbStore RoleBindingStore, cache *PermissionCache) *RBACChecker {
	return &RBACChecker{rbStore: rbStore, cache: cache}
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
	c.cache.Invalidate(userID)
}

func (c *RBACChecker) InvalidateCacheAll() {
	c.cache.InvalidateAll()
}

// getOrLoad returns the cached entry for a user, loading from DB on cache miss.
func (c *RBACChecker) getOrLoad(ctx context.Context, userID int64) (*UserPermissionEntry, error) {
	if entry := c.cache.Get(userID); entry != nil {
		return entry, nil
	}
	entry, err := c.loadUserEntry(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.cache.Set(userID, entry)
	return entry, nil
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
		case "platform":
			entry.PlatformRules = append(entry.PlatformRules, row.Pattern)
			if row.Pattern == "*:*" {
				entry.IsPlatformAdmin = true
			}
		case "workspace":
			if row.WorkspaceID != nil {
				entry.WorkspaceRules[*row.WorkspaceID] = append(entry.WorkspaceRules[*row.WorkspaceID], row.Pattern)
			}
		case "namespace":
			if row.NamespaceID != nil {
				entry.NamespaceRules[*row.NamespaceID] = append(entry.NamespaceRules[*row.NamespaceID], row.Pattern)
			}
		}
	}

	return entry, nil
}
