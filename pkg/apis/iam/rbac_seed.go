package iam

import (
	"context"

	"lcp.io/lcp/lib/logger"
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

// builtinRoles defines all built-in roles with wildcard-based permission patterns.
// Wildcard patterns automatically cover new modules and resources.
var builtinRoles = []BuiltinRoleDef{
	{
		Name:        RolePlatformAdmin,
		DisplayName: "Platform Admin",
		Description: "Full access to all platform resources",
		Scope:       "platform",
		Rules:       []string{"*:*"},
	},
	{
		Name:        RolePlatformViewer,
		DisplayName: "Platform Viewer",
		Description: "Read-only access to all platform resources",
		Scope:       "platform",
		Rules:       []string{"*:list", "*:get"},
	},
	{
		Name:        RoleWorkspaceAdmin,
		DisplayName: "Workspace Admin",
		Description: "Full access to all resources within the workspace",
		Scope:       "workspace",
		Rules:       []string{"*:*"},
	},
	{
		Name:        RoleWorkspaceViewer,
		DisplayName: "Workspace Viewer",
		Description: "Read-only access to all resources within the workspace",
		Scope:       "workspace",
		Rules:       []string{"*:list", "*:get"},
	},
	{
		Name:        RoleNamespaceAdmin,
		DisplayName: "Namespace Admin",
		Description: "Full access to all resources within the namespace",
		Scope:       "namespace",
		Rules:       []string{"*:*"},
	},
	{
		Name:        RoleNamespaceViewer,
		DisplayName: "Namespace Viewer",
		Description: "Read-only access to all resources within the namespace",
		Scope:       "namespace",
		Rules:       []string{"*:list", "*:get"},
	},
}

// SeedRBAC upserts all built-in roles, their permission rules, and creates
// the initial platform-admin binding for the admin user — all in a single transaction.
// Idempotent: repeated calls update roles/rules and skip existing bindings.
func SeedRBAC(ctx context.Context, roleStore RoleStore) error {
	if err := roleStore.SeedRBAC(ctx, builtinRoles, "admin"); err != nil {
		return err
	}
	logger.Infof("seeded %d built-in roles with initial bindings", len(builtinRoles))
	return nil
}
