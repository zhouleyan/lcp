package iam

import (
	"sync"
	"time"
)

// UserPermissionEntry holds the cached permission rules for a single user,
// organized by scope level for efficient matching.
type UserPermissionEntry struct {
	IsPlatformAdmin bool               // true if platformRules contains "*:*" (fast short-circuit)
	PlatformRules   []string           // patterns from platform-scoped bindings
	WorkspaceRules  map[int64][]string // workspaceID → patterns
	NamespaceRules  map[int64][]string // namespaceID → patterns
	expiresAt       time.Time
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
	if (scope == "workspace" || scope == "namespace") && wsID > 0 {
		for _, pattern := range e.WorkspaceRules[wsID] {
			if MatchPermission(pattern, code) {
				return true
			}
		}
	}
	// 3. Namespace-level rules apply to namespace scope only
	if scope == "namespace" && nsID > 0 {
		for _, pattern := range e.NamespaceRules[nsID] {
			if MatchPermission(pattern, code) {
				return true
			}
		}
	}
	return false
}

// PermissionCache is a TTL-based in-memory cache for user permission entries.
type PermissionCache struct {
	mu    sync.RWMutex
	items map[int64]*UserPermissionEntry
	ttl   time.Duration
}

// NewPermissionCache creates a new cache with the given TTL for entries.
func NewPermissionCache(ttl time.Duration) *PermissionCache {
	return &PermissionCache{
		items: make(map[int64]*UserPermissionEntry),
		ttl:   ttl,
	}
}

// Get retrieves the cached entry for a user. Returns nil if not found or expired.
// Expired entries are not eagerly deleted here to avoid a TOCTOU race where a
// concurrent Set for the same user could be incorrectly invalidated. Expired
// entries are naturally replaced on the next Set call.
func (c *PermissionCache) Get(userID int64) *UserPermissionEntry {
	c.mu.RLock()
	entry, ok := c.items[userID]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry
}

// Set stores a permission entry for the given user with TTL expiration.
func (c *PermissionCache) Set(userID int64, entry *UserPermissionEntry) {
	c.mu.Lock()
	entry.expiresAt = time.Now().Add(c.ttl)
	c.items[userID] = entry
	c.mu.Unlock()
}

// Invalidate removes the cached entry for a specific user.
func (c *PermissionCache) Invalidate(userID int64) {
	c.mu.Lock()
	delete(c.items, userID)
	c.mu.Unlock()
}

// InvalidateAll clears all cached entries.
func (c *PermissionCache) InvalidateAll() {
	c.mu.Lock()
	c.items = make(map[int64]*UserPermissionEntry)
	c.mu.Unlock()
}
