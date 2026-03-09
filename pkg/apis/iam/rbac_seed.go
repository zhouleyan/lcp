package iam

import (
	"context"
	"fmt"

	"lcp.io/lcp/lib/logger"
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
		Name:        "platform-admin",
		DisplayName: "Platform Admin",
		Description: "Full access to all platform resources",
		Scope:       "platform",
		Rules:       []string{"*:*"},
	},
	{
		Name:        "platform-viewer",
		DisplayName: "Platform Viewer",
		Description: "Read-only access to all platform resources",
		Scope:       "platform",
		Rules:       []string{"*:list", "*:get"},
	},
	{
		Name:        "workspace-admin",
		DisplayName: "Workspace Admin",
		Description: "Full access to all resources within the workspace",
		Scope:       "workspace",
		Rules:       []string{"*:*"},
	},
	{
		Name:        "workspace-viewer",
		DisplayName: "Workspace Viewer",
		Description: "Read-only access to all resources within the workspace",
		Scope:       "workspace",
		Rules:       []string{"*:list", "*:get"},
	},
	{
		Name:        "namespace-admin",
		DisplayName: "Namespace Admin",
		Description: "Full access to all resources within the namespace",
		Scope:       "namespace",
		Rules:       []string{"*:*"},
	},
	{
		Name:        "namespace-viewer",
		DisplayName: "Namespace Viewer",
		Description: "Read-only access to all resources within the namespace",
		Scope:       "namespace",
		Rules:       []string{"*:list", "*:get"},
	},
}

// SeedBuiltinRoles upserts all built-in roles and their permission rule patterns
// in a single transaction. It is idempotent: repeated calls update roles and rules
// to match the definitions above.
func SeedBuiltinRoles(ctx context.Context, roleStore RoleStore) error {
	if err := roleStore.SeedBuiltinRoles(ctx, builtinRoles); err != nil {
		return err
	}
	logger.Infof("seeded %d built-in roles", len(builtinRoles))
	return nil
}

// SeedInitialBindings creates the platform-admin binding for the admin user.
// It is idempotent via the unique constraint on role_bindings.
func SeedInitialBindings(ctx context.Context, roleStore RoleStore, rbStore RoleBindingStore, userStore UserStore) error {
	// Look up admin user
	adminUser, err := userStore.GetByUsername(ctx, "admin")
	if err != nil {
		// If no admin user exists, skip initial bindings silently
		logger.Infof("no admin user found, skipping initial role bindings")
		return nil
	}

	// Look up platform-admin role
	adminRole, err := roleStore.GetByName(ctx, "platform-admin")
	if err != nil {
		return fmt.Errorf("get platform-admin role: %w", err)
	}

	// Create platform-admin binding for admin user (idempotent via DB constraint)
	if _, err := rbStore.Create(ctx, &DBRoleBinding{
		UserID: adminUser.ID,
		RoleID: adminRole.ID,
		Scope:  "platform",
	}); err != nil {
		// Conflict means binding already exists — that's fine
		logger.Infof("admin user already has platform-admin binding (or created now)")
	}

	return nil
}
