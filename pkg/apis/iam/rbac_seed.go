package iam

import (
	"context"

	"lcp.io/lcp/lib/logger"
)

// Scope constants for RBAC role/binding scoping.
const (
	ScopePlatform  = "platform"
	ScopeWorkspace = "workspace"
	ScopeNamespace = "namespace"
)

// Built-in role name constants.
const (
	RolePlatformAdmin   = "platform-admin"
	RolePlatformViewer  = "platform-viewer"
	RoleWorkspaceAdmin  = "workspace-admin"
	RoleWorkspaceViewer = "workspace-viewer"
	RoleNamespaceAdmin  = "namespace-admin"
	RoleNamespaceViewer = "namespace-viewer"
)

// BuiltinRoleDef defines a built-in role with its metadata and permission rule patterns.
type BuiltinRoleDef struct {
	Name        string
	DisplayName string
	Description string
	Scope       string
	Rules       []string
}

// PlatformBuiltinRoles returns the built-in roles for the platform scope.
func PlatformBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RolePlatformAdmin, DisplayName: "Platform Admin", Description: "Full access to all platform resources", Scope: ScopePlatform, Rules: []string{"*:*"}},
		{Name: RolePlatformViewer, DisplayName: "Platform Viewer", Description: "Read-only access to all platform resources", Scope: ScopePlatform, Rules: []string{"*:list", "*:get"}},
	}
}

// WorkspaceBuiltinRoles returns the built-in roles for the workspace scope.
func WorkspaceBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RoleWorkspaceAdmin, DisplayName: "Workspace Admin", Description: "Full access to all resources within the workspace", Scope: ScopeWorkspace, Rules: []string{"*:*"}},
		{Name: RoleWorkspaceViewer, DisplayName: "Workspace Viewer", Description: "Read-only access to all resources within the workspace", Scope: ScopeWorkspace, Rules: []string{"*:list", "*:get"}},
	}
}

// NamespaceBuiltinRoles returns the built-in roles for the namespace scope.
func NamespaceBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RoleNamespaceAdmin, DisplayName: "Namespace Admin", Description: "Full access to all resources within the namespace", Scope: ScopeNamespace, Rules: []string{"*:*"}},
		{Name: RoleNamespaceViewer, DisplayName: "Namespace Viewer", Description: "Read-only access to all resources within the namespace", Scope: ScopeNamespace, Rules: []string{"*:list", "*:get"}},
	}
}

// SeedRBAC upserts platform built-in roles, their permission rules, creates
// the initial platform-admin binding for the admin user, and ensures scoped
// built-in roles exist for all workspaces and namespaces — all in a single transaction.
// Idempotent: repeated calls update roles/rules and skip existing bindings.
func SeedRBAC(ctx context.Context, roleStore RoleStore) error {
	if err := roleStore.SeedRBAC(ctx, PlatformBuiltinRoles(), "admin"); err != nil {
		return err
	}
	logger.Infof("seeded platform built-in roles with initial bindings")
	return nil
}
