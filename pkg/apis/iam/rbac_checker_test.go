package iam

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func newTestChecker(rules []UserPermissionRuleRow) (*RBACChecker, *atomic.Int32) {
	var loadCount atomic.Int32
	store := &mockRoleBindingStore{
		LoadUserPermissionRulesFn: func(_ context.Context, _ int64) ([]UserPermissionRuleRow, error) {
			loadCount.Add(1)
			return rules, nil
		},
	}
	cache := NewPermissionCache(1 * time.Minute)
	return NewRBACChecker(store, cache), &loadCount
}

func ptr[T any](v T) *T { return &v }

func TestRBACChecker_PlatformAdmin(t *testing.T) {
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "*:*"},
	})
	ctx := context.Background()

	isAdmin, err := checker.IsPlatformAdmin(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isAdmin {
		t.Error("expected platform admin")
	}

	// Platform admin should match any permission at any scope
	tests := []struct {
		code  string
		scope string
		wsID  int64
		nsID  int64
	}{
		{"iam:users:list", "platform", 0, 0},
		{"iam:workspaces:get", "workspace", 10, 0},
		{"iam:namespaces:delete", "namespace", 10, 100},
		{"infra:hosts:create", "platform", 0, 0},
	}
	for _, tt := range tests {
		ok, err := checker.CheckPermission(ctx, 1, tt.code, tt.scope, tt.wsID, tt.nsID)
		if err != nil {
			t.Fatalf("CheckPermission(%s): %v", tt.code, err)
		}
		if !ok {
			t.Errorf("platform admin should have %s at %s scope", tt.code, tt.scope)
		}
	}
}

func TestRBACChecker_PlatformMember(t *testing.T) {
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "iam:workspaces:list"},
		{Scope: "platform", Pattern: "iam:namespaces:list"},
		{Scope: "platform", Pattern: "iam:users:change-password"},
	})
	ctx := context.Background()

	isAdmin, _ := checker.IsPlatformAdmin(ctx, 1)
	if isAdmin {
		t.Error("should not be platform admin")
	}

	ok, _ := checker.CheckPermission(ctx, 1, "iam:workspaces:list", "platform", 0, 0)
	if !ok {
		t.Error("should have iam:workspaces:list")
	}

	ok, _ = checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if ok {
		t.Error("should not have iam:users:list")
	}
}

func TestRBACChecker_WorkspaceScope(t *testing.T) {
	var wsID int64 = 10
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: "workspace", WorkspaceID: &wsID, Pattern: "iam:namespaces:*"},
		{Scope: "workspace", WorkspaceID: &wsID, Pattern: "iam:workspaces:get"},
	})
	ctx := context.Background()

	// Has permission within workspace 10
	ok, _ := checker.CheckPermission(ctx, 1, "iam:namespaces:list", "workspace", 10, 0)
	if !ok {
		t.Error("should match wildcard iam:namespaces:* in ws 10")
	}

	// Workspace rule does NOT apply to a different workspace
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:list", "workspace", 20, 0)
	if ok {
		t.Error("should not match in ws 20")
	}

	// Workspace rules inherit to namespace scope (scope chain)
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", "namespace", 10, 100)
	if !ok {
		t.Error("workspace rule should inherit to namespace scope")
	}
}

func TestRBACChecker_NamespaceScope(t *testing.T) {
	var nsID int64 = 100
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: "namespace", NamespaceID: &nsID, Pattern: "iam:namespaces:get"},
		{Scope: "namespace", NamespaceID: &nsID, Pattern: "iam:namespaces:users:list"},
	})
	ctx := context.Background()

	ok, _ := checker.CheckPermission(ctx, 1, "iam:namespaces:get", "namespace", 10, 100)
	if !ok {
		t.Error("should match namespace rule for ns 100")
	}

	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", "namespace", 10, 200)
	if ok {
		t.Error("should not match for ns 200")
	}

	// Namespace rules don't apply to workspace scope
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", "workspace", 10, 0)
	if ok {
		t.Error("namespace rules should not apply to workspace scope")
	}
}

func TestRBACChecker_NoBindings(t *testing.T) {
	checker, _ := newTestChecker(nil)
	ctx := context.Background()

	ok, _ := checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if ok {
		t.Error("user without bindings should have no permissions")
	}

	isAdmin, _ := checker.IsPlatformAdmin(ctx, 1)
	if isAdmin {
		t.Error("user without bindings should not be admin")
	}
}

func TestRBACChecker_CacheHit(t *testing.T) {
	checker, loadCount := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "iam:users:list"},
	})
	ctx := context.Background()

	// First call: cache miss → loads from DB
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 1 {
		t.Errorf("expected 1 DB load, got %d", loadCount.Load())
	}

	// Second call: cache hit → no DB load
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 1 {
		t.Errorf("expected still 1 DB load after cache hit, got %d", loadCount.Load())
	}
}

func TestRBACChecker_InvalidateCache(t *testing.T) {
	checker, loadCount := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "iam:users:list"},
	})
	ctx := context.Background()

	// Load once
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 1 {
		t.Fatalf("expected 1 load, got %d", loadCount.Load())
	}

	// Invalidate and check again → should reload
	checker.InvalidateCache(1)
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 2 {
		t.Errorf("expected 2 loads after invalidation, got %d", loadCount.Load())
	}
}

func TestRBACChecker_InvalidateCacheAll(t *testing.T) {
	checker, loadCount := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "*:*"},
	})
	ctx := context.Background()

	// Load for two users
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	checker.CheckPermission(ctx, 2, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 2 {
		t.Fatalf("expected 2 loads, got %d", loadCount.Load())
	}

	// InvalidateAll → both should reload
	checker.InvalidateCacheAll()
	checker.CheckPermission(ctx, 1, "iam:users:list", "platform", 0, 0)
	checker.CheckPermission(ctx, 2, "iam:users:list", "platform", 0, 0)
	if loadCount.Load() != 4 {
		t.Errorf("expected 4 loads after invalidateAll, got %d", loadCount.Load())
	}
}

func TestRBACChecker_ScopeChainInheritance(t *testing.T) {
	var wsID int64 = 10
	var nsID int64 = 100
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: "platform", Pattern: "iam:workspaces:list"},
		{Scope: "workspace", WorkspaceID: &wsID, Pattern: "iam:namespaces:list"},
		{Scope: "namespace", NamespaceID: &nsID, Pattern: "iam:namespaces:users:list"},
	})
	ctx := context.Background()

	// Platform rule available at namespace level
	ok, _ := checker.CheckPermission(ctx, 1, "iam:workspaces:list", "namespace", 10, 100)
	if !ok {
		t.Error("platform rule should be available at namespace level")
	}

	// Workspace rule available at namespace level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:list", "namespace", 10, 100)
	if !ok {
		t.Error("workspace rule should be available at namespace level")
	}

	// Namespace rule only at namespace level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:users:list", "namespace", 10, 100)
	if !ok {
		t.Error("namespace rule should match at namespace level")
	}

	// Namespace rule NOT at platform level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:users:list", "platform", 0, 0)
	if ok {
		t.Error("namespace rule should not apply at platform level")
	}
}
