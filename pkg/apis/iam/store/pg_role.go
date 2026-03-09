package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"lcp.io/lcp/lib/logger"

	apierrors "lcp.io/lcp/lib/api/errors"
	"lcp.io/lcp/pkg/apis/iam"
	"lcp.io/lcp/pkg/db"
	"lcp.io/lcp/pkg/db/generated"
)

type pgRoleStore struct {
	db      *pgxpool.Pool
	queries *generated.Queries
}

// NewPGRoleStore creates a new PostgreSQL-backed RoleStore.
func NewPGRoleStore(pool *pgxpool.Pool, queries *generated.Queries) iam.RoleStore {
	return &pgRoleStore{db: pool, queries: queries}
}

// roleFromCreateRow converts a CreateRoleRow to a Role model.
func roleFromCreateRow(r generated.CreateRoleRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromGetByIDRow converts a GetRoleByIDRow to a Role model.
func roleFromGetByIDRow(r generated.GetRoleByIDRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromGetByNameRow converts a GetRoleByNameRow to a Role model.
func roleFromGetByNameRow(r generated.GetRoleByNameRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromGetByNameAndWorkspaceRow converts a GetRoleByNameAndWorkspaceRow to a Role model.
func roleFromGetByNameAndWorkspaceRow(r generated.GetRoleByNameAndWorkspaceRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromGetByNameAndNamespaceRow converts a GetRoleByNameAndNamespaceRow to a Role model.
func roleFromGetByNameAndNamespaceRow(r generated.GetRoleByNameAndNamespaceRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromUpdateRow converts an UpdateRoleRow to a Role model.
func roleFromUpdateRow(r generated.UpdateRoleRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// roleFromUpsertRow converts an UpsertRoleRow to a Role model.
func roleFromUpsertRow(r generated.UpsertRoleRow) generated.Role {
	return generated.Role{
		ID:          r.ID,
		Name:        r.Name,
		DisplayName: r.DisplayName,
		Description: r.Description,
		Scope:       r.Scope,
		WorkspaceID: r.WorkspaceID,
		NamespaceID: r.NamespaceID,
		Builtin:     r.Builtin,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (s *pgRoleStore) Create(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.CreateRole(ctx, generated.CreateRoleParams{
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Scope:       role.Scope,
		Builtin:     role.Builtin,
		WorkspaceID: role.WorkspaceID,
		NamespaceID: role.NamespaceID,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return nil, apierrors.NewConflict("role", role.Name)
		}
		return nil, fmt.Errorf("create role: %w", err)
	}
	result := roleFromCreateRow(row)
	return &result, nil
}

func (s *pgRoleStore) GetByID(ctx context.Context, id int64) (*iam.DBRoleWithRules, error) {
	row, err := s.queries.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", fmt.Sprintf("%d", id))
		}
		return nil, fmt.Errorf("get role by id: %w", err)
	}

	rules, err := s.queries.GetRulesByRoleID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get rules for role %d: %w", id, err)
	}

	role := roleFromGetByIDRow(row)
	return &iam.DBRoleWithRules{
		Role:  role,
		Rules: rules,
	}, nil
}

func (s *pgRoleStore) GetByName(ctx context.Context, name string) (*iam.DBRole, error) {
	row, err := s.queries.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", name)
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	result := roleFromGetByNameRow(row)
	return &result, nil
}

func (s *pgRoleStore) GetByNameAndWorkspace(ctx context.Context, name string, workspaceID int64) (*iam.DBRole, error) {
	row, err := s.queries.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
		Name:        name,
		WorkspaceID: &workspaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", name)
		}
		return nil, fmt.Errorf("get role by name and workspace: %w", err)
	}
	result := roleFromGetByNameAndWorkspaceRow(row)
	return &result, nil
}

func (s *pgRoleStore) GetByNameAndNamespace(ctx context.Context, name string, namespaceID int64) (*iam.DBRole, error) {
	row, err := s.queries.GetRoleByNameAndNamespace(ctx, generated.GetRoleByNameAndNamespaceParams{
		Name:        name,
		NamespaceID: &namespaceID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", name)
		}
		return nil, fmt.Errorf("get role by name and namespace: %w", err)
	}
	result := roleFromGetByNameAndNamespaceRow(row)
	return &result, nil
}

func (s *pgRoleStore) Update(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.UpdateRole(ctx, generated.UpdateRoleParams{
		ID:          role.ID,
		DisplayName: role.DisplayName,
		Description: role.Description,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.NewNotFound("role", fmt.Sprintf("%d", role.ID))
		}
		return nil, fmt.Errorf("update role: %w", err)
	}
	result := roleFromUpdateRow(row)
	return &result, nil
}

func (s *pgRoleStore) Upsert(ctx context.Context, role *iam.DBRole) (*iam.DBRole, error) {
	row, err := s.queries.UpsertRole(ctx, generated.UpsertRoleParams{
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		Scope:       role.Scope,
		Builtin:     role.Builtin,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert role: %w", err)
	}
	result := roleFromUpsertRow(row)
	return &result, nil
}

func (s *pgRoleStore) Delete(ctx context.Context, id int64) error {
	if err := s.queries.DeleteRole(ctx, id); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (s *pgRoleStore) List(ctx context.Context, q db.ListQuery) (*db.ListResult[iam.DBRoleListRow], error) {
	offset, limit := db.PaginationToOffsetLimit(q.Pagination)

	countParams := generated.CountRolesParams{
		Scope:       filterStr(q.Filters, "scope"),
		Builtin:     filterBool(q.Filters, "builtin"),
		WorkspaceID: filterInt64(q.Filters, "workspace_id"),
		NamespaceID: filterInt64(q.Filters, "namespace_id"),
		Search:      filterStr(q.Filters, "search"),
	}

	count, err := s.queries.CountRoles(ctx, countParams)
	if err != nil {
		return nil, fmt.Errorf("count roles: %w", err)
	}

	sortOrder := q.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, err := s.queries.ListRoles(ctx, generated.ListRolesParams{
		Scope:       countParams.Scope,
		Builtin:     countParams.Builtin,
		WorkspaceID: countParams.WorkspaceID,
		NamespaceID: countParams.NamespaceID,
		Search:      countParams.Search,
		SortField:   q.SortBy,
		SortOrder:   sortOrder,
		PageOffset:  offset,
		PageSize:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	items := make([]iam.DBRoleListRow, len(rows))
	for i, r := range rows {
		items[i] = r
	}

	return &db.ListResult[iam.DBRoleListRow]{
		Items:      items,
		TotalCount: count,
	}, nil
}

// createBuiltinRolesInTx creates built-in roles with permission rules using the provided transaction-scoped queries.
func createBuiltinRolesInTx(ctx context.Context, qtx *generated.Queries, defs []iam.BuiltinRoleDef, workspaceID *int64, namespaceID *int64) error {
	for _, def := range defs {
		// Check if role already exists to avoid unique constraint violation
		// which aborts the PostgreSQL transaction.
		var exists bool
		if workspaceID != nil {
			_, err := qtx.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
				Name:        def.Name,
				WorkspaceID: workspaceID,
			})
			exists = err == nil
		} else if namespaceID != nil {
			_, err := qtx.GetRoleByNameAndNamespace(ctx, generated.GetRoleByNameAndNamespaceParams{
				Name:        def.Name,
				NamespaceID: namespaceID,
			})
			exists = err == nil
		}
		if exists {
			continue
		}

		row, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Scope:       def.Scope,
			Builtin:     true,
			WorkspaceID: workspaceID,
			NamespaceID: namespaceID,
		})
		if err != nil {
			return fmt.Errorf("create builtin role %s: %w", def.Name, err)
		}
		for _, pattern := range def.Rules {
			if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
				RoleID:  row.ID,
				Pattern: pattern,
			}); err != nil {
				return fmt.Errorf("add rule %q for role %s: %w", pattern, def.Name, err)
			}
		}
	}
	return nil
}

func (s *pgRoleStore) SeedRBAC(ctx context.Context, roles []iam.BuiltinRoleDef, adminUsername string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// 1. Upsert platform built-in roles and their permission rules
	var platformAdminRoleID int64
	for _, def := range roles {
		role, err := qtx.UpsertRole(ctx, generated.UpsertRoleParams{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Scope:       def.Scope,
			Builtin:     true,
		})
		if err != nil {
			return fmt.Errorf("upsert builtin role %s: %w", def.Name, err)
		}

		if def.Name == iam.RolePlatformAdmin {
			platformAdminRoleID = role.ID
		}

		if err := qtx.DeleteRolePermissionRules(ctx, role.ID); err != nil {
			return fmt.Errorf("delete rules for role %s: %w", def.Name, err)
		}

		for _, pattern := range def.Rules {
			if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
				RoleID:  role.ID,
				Pattern: pattern,
			}); err != nil {
				return fmt.Errorf("add rule %q for role %s: %w", pattern, def.Name, err)
			}
		}
	}

	// 2. Create initial platform-admin binding for admin user (if exists)
	if adminUsername != "" && platformAdminRoleID != 0 {
		adminUser, err := qtx.GetUserByUsername(ctx, adminUsername)
		if err == nil {
			_ = qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
				UserID: adminUser.ID,
				RoleID: platformAdminRoleID,
				Scope:  iam.ScopePlatform,
			})
		}
	}

	// 3. Create built-in workspace roles for all existing workspaces
	workspaceIDs, err := qtx.ListAllWorkspaceIDs(ctx)
	if err != nil {
		return fmt.Errorf("list workspace IDs: %w", err)
	}
	for _, wsID := range workspaceIDs {
		if err := createBuiltinRolesInTx(ctx, qtx, iam.WorkspaceBuiltinRoles(), &wsID, nil); err != nil {
			return fmt.Errorf("create workspace roles for workspace %d: %w", wsID, err)
		}
	}

	// 4. Create built-in namespace roles for all existing namespaces
	nsRows, err := qtx.ListAllNamespaceIDsWithWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("list namespace IDs: %w", err)
	}
	for _, nsRow := range nsRows {
		if err := createBuiltinRolesInTx(ctx, qtx, iam.NamespaceBuiltinRoles(), nil, &nsRow.ID); err != nil {
			return fmt.Errorf("create namespace roles for namespace %d: %w", nsRow.ID, err)
		}
	}

	// 5. Migrate existing role_bindings from old global workspace/namespace roles to new scoped roles
	type migrationPair struct {
		roleName string
		scope    string
	}
	migrations := []migrationPair{
		{iam.RoleWorkspaceAdmin, iam.ScopeWorkspace},
		{iam.RoleWorkspaceViewer, iam.ScopeWorkspace},
		{iam.RoleNamespaceAdmin, iam.ScopeNamespace},
		{iam.RoleNamespaceViewer, iam.ScopeNamespace},
	}
	for _, m := range migrations {
		// Find old global role (no workspace_id/namespace_id)
		var oldRoleID int64
		err := tx.QueryRow(ctx,
			`SELECT id FROM roles WHERE name = $1 AND scope = $2 AND workspace_id IS NULL AND namespace_id IS NULL`,
			m.roleName, m.scope,
		).Scan(&oldRoleID)
		if err != nil {
			// No old global role found, nothing to migrate
			continue
		}

		if m.scope == iam.ScopeWorkspace {
			// Find bindings referencing the old global role and re-point them
			rows, err := tx.Query(ctx,
				`SELECT id, workspace_id FROM role_bindings WHERE role_id = $1`,
				oldRoleID,
			)
			if err != nil {
				return fmt.Errorf("list bindings for old role %s: %w", m.roleName, err)
			}
			type bindingInfo struct {
				id          int64
				workspaceID *int64
			}
			var bindings []bindingInfo
			for rows.Next() {
				var b bindingInfo
				if err := rows.Scan(&b.id, &b.workspaceID); err != nil {
					rows.Close()
					return fmt.Errorf("scan binding: %w", err)
				}
				bindings = append(bindings, b)
			}
			rows.Close()

			for _, b := range bindings {
				if b.workspaceID == nil {
					continue
				}
				var newRoleID int64
				err := tx.QueryRow(ctx,
					`SELECT id FROM roles WHERE name = $1 AND workspace_id = $2`,
					m.roleName, *b.workspaceID,
				).Scan(&newRoleID)
				if err != nil {
					logger.Warnf("cannot find scoped role %s for workspace %d, skipping binding %d", m.roleName, *b.workspaceID, b.id)
					continue
				}
				if _, err := tx.Exec(ctx, `UPDATE role_bindings SET role_id = $1 WHERE id = $2`, newRoleID, b.id); err != nil {
					return fmt.Errorf("re-point binding %d: %w", b.id, err)
				}
			}
		} else {
			// namespace scope
			rows, err := tx.Query(ctx,
				`SELECT id, namespace_id FROM role_bindings WHERE role_id = $1`,
				oldRoleID,
			)
			if err != nil {
				return fmt.Errorf("list bindings for old role %s: %w", m.roleName, err)
			}
			type bindingInfo struct {
				id          int64
				namespaceID *int64
			}
			var bindings []bindingInfo
			for rows.Next() {
				var b bindingInfo
				if err := rows.Scan(&b.id, &b.namespaceID); err != nil {
					rows.Close()
					return fmt.Errorf("scan binding: %w", err)
				}
				bindings = append(bindings, b)
			}
			rows.Close()

			for _, b := range bindings {
				if b.namespaceID == nil {
					continue
				}
				var newRoleID int64
				err := tx.QueryRow(ctx,
					`SELECT id FROM roles WHERE name = $1 AND namespace_id = $2`,
					m.roleName, *b.namespaceID,
				).Scan(&newRoleID)
				if err != nil {
					logger.Warnf("cannot find scoped role %s for namespace %d, skipping binding %d", m.roleName, *b.namespaceID, b.id)
					continue
				}
				if _, err := tx.Exec(ctx, `UPDATE role_bindings SET role_id = $1 WHERE id = $2`, newRoleID, b.id); err != nil {
					return fmt.Errorf("re-point binding %d: %w", b.id, err)
				}
			}
		}

		// 6. Delete the old global role (cascade deletes any remaining bindings/rules)
		if _, err := tx.Exec(ctx, `DELETE FROM roles WHERE id = $1`, oldRoleID); err != nil {
			return fmt.Errorf("delete old global role %s: %w", m.roleName, err)
		}
		logger.Infof("migrated role %s from global to scoped", m.roleName)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s *pgRoleStore) SetPermissionRules(ctx context.Context, roleID int64, patterns []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	// Delete all existing rules for this role
	if err := qtx.DeleteRolePermissionRules(ctx, roleID); err != nil {
		return fmt.Errorf("delete existing rules: %w", err)
	}

	// Insert new rules
	for _, pattern := range patterns {
		if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
			RoleID:  roleID,
			Pattern: pattern,
		}); err != nil {
			return fmt.Errorf("add rule %q: %w", pattern, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
