package iam

import (
	"context"
	"sync/atomic"
	"testing"
)

func newTestChecker(rules []UserPermissionRuleRow) (*RBACChecker, *atomic.Int32) {
	var loadCount atomic.Int32
	store := &mockRoleBindingStore{
		LoadUserPermissionRulesFn: func(_ context.Context, _ int64) ([]UserPermissionRuleRow, error) {
			loadCount.Add(1)
			return rules, nil
		},
	}
	return NewRBACChecker(store), &loadCount
}

func ptr[T any](v T) *T { return &v }

func TestRBACChecker_PlatformAdmin(t *testing.T) {
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: ScopePlatform, Pattern: "*:*"},
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
		{"iam:users:list", ScopePlatform, 0, 0},
		{"iam:workspaces:get", ScopeWorkspace, 10, 0},
		{"iam:namespaces:delete", ScopeNamespace, 10, 100},
		{"infra:hosts:create", ScopePlatform, 0, 0},
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
		{Scope: ScopePlatform, Pattern: "iam:workspaces:list"},
		{Scope: ScopePlatform, Pattern: "iam:namespaces:list"},
		{Scope: ScopePlatform, Pattern: "iam:users:change-password"},
	})
	ctx := context.Background()

	isAdmin, _ := checker.IsPlatformAdmin(ctx, 1)
	if isAdmin {
		t.Error("should not be platform admin")
	}

	ok, _ := checker.CheckPermission(ctx, 1, "iam:workspaces:list", ScopePlatform, 0, 0)
	if !ok {
		t.Error("should have iam:workspaces:list")
	}

	ok, _ = checker.CheckPermission(ctx, 1, "iam:users:list", ScopePlatform, 0, 0)
	if ok {
		t.Error("should not have iam:users:list")
	}
}

func TestRBACChecker_WorkspaceScope(t *testing.T) {
	var wsID int64 = 10
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: ScopeWorkspace, WorkspaceID: &wsID, Pattern: "iam:namespaces:*"},
		{Scope: ScopeWorkspace, WorkspaceID: &wsID, Pattern: "iam:workspaces:get"},
	})
	ctx := context.Background()

	// Has permission within workspace 10
	ok, _ := checker.CheckPermission(ctx, 1, "iam:namespaces:list", ScopeWorkspace, 10, 0)
	if !ok {
		t.Error("should match wildcard iam:namespaces:* in ws 10")
	}

	// Workspace rule does NOT apply to a different workspace
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:list", ScopeWorkspace, 20, 0)
	if ok {
		t.Error("should not match in ws 20")
	}

	// Workspace rules inherit to namespace scope (scope chain)
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", ScopeNamespace, 10, 100)
	if !ok {
		t.Error("workspace rule should inherit to namespace scope")
	}
}

func TestRBACChecker_NamespaceScope(t *testing.T) {
	var nsID int64 = 100
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: ScopeNamespace, NamespaceID: &nsID, Pattern: "iam:namespaces:get"},
		{Scope: ScopeNamespace, NamespaceID: &nsID, Pattern: "iam:namespaces:users:list"},
	})
	ctx := context.Background()

	ok, _ := checker.CheckPermission(ctx, 1, "iam:namespaces:get", ScopeNamespace, 10, 100)
	if !ok {
		t.Error("should match namespace rule for ns 100")
	}

	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", ScopeNamespace, 10, 200)
	if ok {
		t.Error("should not match for ns 200")
	}

	// Namespace rules don't apply to workspace scope
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:get", ScopeWorkspace, 10, 0)
	if ok {
		t.Error("namespace rules should not apply to workspace scope")
	}
}

func TestRBACChecker_NoBindings(t *testing.T) {
	checker, _ := newTestChecker(nil)
	ctx := context.Background()

	ok, _ := checker.CheckPermission(ctx, 1, "iam:users:list", ScopePlatform, 0, 0)
	if ok {
		t.Error("user without bindings should have no permissions")
	}

	isAdmin, _ := checker.IsPlatformAdmin(ctx, 1)
	if isAdmin {
		t.Error("user without bindings should not be admin")
	}
}

func TestRBACChecker_ScopeChainInheritance(t *testing.T) {
	var wsID int64 = 10
	var nsID int64 = 100
	checker, _ := newTestChecker([]UserPermissionRuleRow{
		{Scope: ScopePlatform, Pattern: "iam:workspaces:list"},
		{Scope: ScopeWorkspace, WorkspaceID: &wsID, Pattern: "iam:namespaces:list"},
		{Scope: ScopeNamespace, NamespaceID: &nsID, Pattern: "iam:namespaces:users:list"},
	})
	ctx := context.Background()

	// Platform rule available at namespace level
	ok, _ := checker.CheckPermission(ctx, 1, "iam:workspaces:list", ScopeNamespace, 10, 100)
	if !ok {
		t.Error("platform rule should be available at namespace level")
	}

	// Workspace rule available at namespace level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:list", ScopeNamespace, 10, 100)
	if !ok {
		t.Error("workspace rule should be available at namespace level")
	}

	// Namespace rule only at namespace level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:users:list", ScopeNamespace, 10, 100)
	if !ok {
		t.Error("namespace rule should match at namespace level")
	}

	// Namespace rule NOT at platform level
	ok, _ = checker.CheckPermission(ctx, 1, "iam:namespaces:users:list", ScopePlatform, 0, 0)
	if ok {
		t.Error("namespace rule should not apply at platform level")
	}
}
