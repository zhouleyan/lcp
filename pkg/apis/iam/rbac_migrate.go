package iam

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"lcp.io/lcp/lib/logger"
)

// legacyRoleMapping maps a legacy join-table role to the target RBAC role name and is_owner flag.
type legacyRoleMapping struct {
	legacyRole string
	rbacRole   string
	isOwner    bool
}

// MigrateJoinTablesToRoleBindings migrates existing user_workspaces and user_namespaces
// records into role_bindings. Uses ON CONFLICT DO NOTHING for idempotency.
func MigrateJoinTablesToRoleBindings(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	wsMappings := []legacyRoleMapping{
		{"owner", RoleWorkspaceAdmin, true},
		{"admin", RoleWorkspaceAdmin, false},
		{"member", RoleWorkspaceViewer, false},
	}

	nsMappings := []legacyRoleMapping{
		{"owner", RoleNamespaceAdmin, true},
		{"admin", RoleNamespaceAdmin, false},
		{"member", RoleNamespaceViewer, false},
	}

	var totalWS, totalNS int64

	for _, m := range wsMappings {
		n, err := migrateWorkspaceRole(ctx, tx, m)
		if err != nil {
			return err
		}
		totalWS += n
	}

	for _, m := range nsMappings {
		n, err := migrateNamespaceRole(ctx, tx, m)
		if err != nil {
			return err
		}
		totalNS += n
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logger.Infof("migrated %d workspace + %d namespace bindings to role_bindings", totalWS, totalNS)
	return nil
}

func migrateWorkspaceRole(ctx context.Context, tx pgx.Tx, m legacyRoleMapping) (int64, error) {
	tag, err := tx.Exec(ctx,
		`INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, is_owner)
		 SELECT uw.user_id, r.id, 'workspace', uw.workspace_id, $3::boolean
		 FROM user_workspaces uw
		 JOIN roles r ON r.name = $1
		 WHERE uw.role = $2
		 ON CONFLICT DO NOTHING`,
		m.rbacRole, m.legacyRole, m.isOwner,
	)
	if err != nil {
		return 0, fmt.Errorf("migrate workspace role %q → %q: %w", m.legacyRole, m.rbacRole, err)
	}
	return tag.RowsAffected(), nil
}

func migrateNamespaceRole(ctx context.Context, tx pgx.Tx, m legacyRoleMapping) (int64, error) {
	tag, err := tx.Exec(ctx,
		`INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, namespace_id, is_owner)
		 SELECT un.user_id, r.id, 'namespace', n.workspace_id, un.namespace_id, $3::boolean
		 FROM user_namespaces un
		 JOIN namespaces n ON n.id = un.namespace_id
		 JOIN roles r ON r.name = $1
		 WHERE un.role = $2
		 ON CONFLICT DO NOTHING`,
		m.rbacRole, m.legacyRole, m.isOwner,
	)
	if err != nil {
		return 0, fmt.Errorf("migrate namespace role %q → %q: %w", m.legacyRole, m.rbacRole, err)
	}
	return tag.RowsAffected(), nil
}
