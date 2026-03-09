package iam

import (
	"sync"
	"testing"
	"time"
)

func TestPermissionCache_SetAndGet(t *testing.T) {
	cache := NewPermissionCache(1 * time.Minute)
	entry := &UserPermissionEntry{
		PlatformRules: []string{"iam:users:list"},
	}
	cache.Set(1, entry)

	got := cache.Get(1)
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if len(got.PlatformRules) != 1 || got.PlatformRules[0] != "iam:users:list" {
		t.Errorf("unexpected platform rules: %v", got.PlatformRules)
	}
}

func TestPermissionCache_GetMiss(t *testing.T) {
	cache := NewPermissionCache(1 * time.Minute)
	if got := cache.Get(999); got != nil {
		t.Errorf("expected nil for missing key, got %v", got)
	}
}

func TestPermissionCache_TTLExpiry(t *testing.T) {
	cache := NewPermissionCache(10 * time.Millisecond)
	cache.Set(1, &UserPermissionEntry{PlatformRules: []string{"*:*"}})

	// Should be available immediately
	if got := cache.Get(1); got == nil {
		t.Fatal("expected entry before TTL expiry")
	}

	time.Sleep(20 * time.Millisecond)

	// Should be expired
	if got := cache.Get(1); got != nil {
		t.Error("expected nil after TTL expiry")
	}
}

func TestPermissionCache_Invalidate(t *testing.T) {
	cache := NewPermissionCache(1 * time.Minute)
	cache.Set(1, &UserPermissionEntry{PlatformRules: []string{"*:*"}})
	cache.Set(2, &UserPermissionEntry{PlatformRules: []string{"iam:*"}})

	cache.Invalidate(1)

	if got := cache.Get(1); got != nil {
		t.Error("expected nil after invalidate")
	}
	if got := cache.Get(2); got == nil {
		t.Error("expected entry for user 2 to remain")
	}
}

func TestPermissionCache_InvalidateAll(t *testing.T) {
	cache := NewPermissionCache(1 * time.Minute)
	cache.Set(1, &UserPermissionEntry{PlatformRules: []string{"*:*"}})
	cache.Set(2, &UserPermissionEntry{PlatformRules: []string{"iam:*"}})

	cache.InvalidateAll()

	if got := cache.Get(1); got != nil {
		t.Error("expected nil after invalidateAll for user 1")
	}
	if got := cache.Get(2); got != nil {
		t.Error("expected nil after invalidateAll for user 2")
	}
}

func TestPermissionCache_Concurrent(t *testing.T) {
	cache := NewPermissionCache(1 * time.Minute)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Set(int64(i), &UserPermissionEntry{
				PlatformRules: []string{"iam:*"},
			})
		}()
	}

	// Concurrent reads
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Get(int64(i))
		}()
	}

	// Concurrent invalidations
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Invalidate(int64(i))
		}()
	}

	wg.Wait()
}

func TestUserPermissionEntry_HasPermission(t *testing.T) {
	entry := &UserPermissionEntry{
		IsPlatformAdmin: true,
		PlatformRules:   []string{"*:*"},
	}
	// Platform admin matches everything
	if !entry.HasPermission("iam:users:list", ScopePlatform, 0, 0) {
		t.Error("platform admin should match iam:users:list")
	}
	if !entry.HasPermission("iam:workspaces:get", ScopeWorkspace, 1, 0) {
		t.Error("platform admin should match workspace scope")
	}
	if !entry.HasPermission("iam:namespaces:get", ScopeNamespace, 1, 2) {
		t.Error("platform admin should match namespace scope")
	}
}

func TestUserPermissionEntry_HasPermission_PlatformMember(t *testing.T) {
	entry := &UserPermissionEntry{
		PlatformRules: []string{"iam:workspaces:list", "iam:namespaces:list"},
	}

	if !entry.HasPermission("iam:workspaces:list", ScopePlatform, 0, 0) {
		t.Error("platform member should match iam:workspaces:list")
	}
	if entry.HasPermission("iam:users:list", ScopePlatform, 0, 0) {
		t.Error("platform member should not match iam:users:list")
	}
}

func TestUserPermissionEntry_HasPermission_WorkspaceScope(t *testing.T) {
	entry := &UserPermissionEntry{
		WorkspaceRules: map[int64][]string{
			10: {"iam:namespaces:*", "iam:workspaces:get"},
		},
	}

	// Workspace-level permission in workspace scope
	if !entry.HasPermission("iam:workspaces:get", ScopeWorkspace, 10, 0) {
		t.Error("should match workspace rule for ws 10")
	}
	// Workspace rule doesn't apply to different workspace
	if entry.HasPermission("iam:workspaces:get", ScopeWorkspace, 20, 0) {
		t.Error("should not match workspace rule for ws 20")
	}
	// Workspace rules inherit into namespace scope (scope chain)
	if !entry.HasPermission("iam:namespaces:list", ScopeNamespace, 10, 100) {
		t.Error("workspace rule should inherit to namespace scope")
	}
}

func TestUserPermissionEntry_HasPermission_NamespaceScope(t *testing.T) {
	entry := &UserPermissionEntry{
		NamespaceRules: map[int64][]string{
			100: {"iam:namespaces:get", "iam:namespaces:users:list"},
		},
	}

	if !entry.HasPermission("iam:namespaces:get", ScopeNamespace, 10, 100) {
		t.Error("should match namespace rule for ns 100")
	}
	if entry.HasPermission("iam:namespaces:get", ScopeNamespace, 10, 200) {
		t.Error("should not match namespace rule for ns 200")
	}
	// Namespace rules don't apply to workspace scope
	if entry.HasPermission("iam:namespaces:get", ScopeWorkspace, 10, 0) {
		t.Error("namespace rules should not apply to workspace scope")
	}
}

func TestUserPermissionEntry_HasPermission_ScopeChain(t *testing.T) {
	// User has platform + workspace + namespace rules
	entry := &UserPermissionEntry{
		PlatformRules: []string{"iam:workspaces:list"},
		WorkspaceRules: map[int64][]string{
			10: {"iam:namespaces:list"},
		},
		NamespaceRules: map[int64][]string{
			100: {"iam:namespaces:users:list"},
		},
	}

	// Platform rule available at namespace level
	if !entry.HasPermission("iam:workspaces:list", ScopeNamespace, 10, 100) {
		t.Error("platform rule should be available at namespace level")
	}
	// Workspace rule available at namespace level
	if !entry.HasPermission("iam:namespaces:list", ScopeNamespace, 10, 100) {
		t.Error("workspace rule should be available at namespace level")
	}
	// Namespace-specific rule
	if !entry.HasPermission("iam:namespaces:users:list", ScopeNamespace, 10, 100) {
		t.Error("namespace rule should match at namespace level")
	}
	// Namespace rule NOT available at platform level
	if entry.HasPermission("iam:namespaces:users:list", ScopePlatform, 0, 0) {
		t.Error("namespace rule should not apply at platform level")
	}
}
