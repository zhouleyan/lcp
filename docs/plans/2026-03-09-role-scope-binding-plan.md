# Role Scope Binding Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Associate workspace/namespace-level roles with specific workspace/namespace instances, auto-create built-in roles on resource creation, and make scoped role routes full CRUD.

**Architecture:** Add `workspace_id`/`namespace_id` FK columns to the `roles` table with CHECK constraints for mutual exclusivity. Modify the workspace/namespace Create transactions to seed built-in roles. Convert `scopedRoleStorage` from read-only to full CRUD filtered by resource ID.

**Tech Stack:** Go, PostgreSQL, sqlc, pgx

---

### Task 1: Update Database Schema

**Files:**
- Modify: `pkg/db/schema/schema.sql:98-116`

**Step 1: Modify the roles table DDL**

Replace the existing `roles` table definition (lines 98-115) with:

```sql
-- roles table (builtin + user-defined)
CREATE TABLE roles (
    id           BIGSERIAL    PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    description  TEXT         NOT NULL DEFAULT '',
    scope        VARCHAR(20)  NOT NULL,
    builtin      BOOLEAN      NOT NULL DEFAULT false,
    workspace_id BIGINT       REFERENCES workspaces(id) ON DELETE CASCADE,
    namespace_id BIGINT       REFERENCES namespaces(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_role_scope CHECK (
        (scope = 'platform'  AND workspace_id IS NULL AND namespace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
        (scope = 'namespace' AND workspace_id IS NULL AND namespace_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uk_roles_platform ON roles(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_roles_workspace ON roles(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_roles_namespace ON roles(name, namespace_id) WHERE scope = 'namespace';
```

Key changes:
- Remove `UNIQUE` from `name` column
- Add `workspace_id` and `namespace_id` nullable FK columns with `ON DELETE CASCADE`
- Replace `chk_role_scope` with new CHECK enforcing mutual exclusivity
- Add 3 conditional unique indexes

**Step 2: Update table comments**

```sql
COMMENT ON TABLE roles IS '角色表：内置角色 + 用户自定义角色';
COMMENT ON COLUMN roles.name IS '角色名称，在其作用域内唯一';
COMMENT ON COLUMN roles.display_name IS '角色显示名称';
COMMENT ON COLUMN roles.scope IS '角色作用域：platform（全局）/ workspace（属于某工作空间）/ namespace（属于某项目）';
COMMENT ON COLUMN roles.builtin IS '是否为内置角色（内置不可删除）';
COMMENT ON COLUMN roles.workspace_id IS '所属工作空间 ID（workspace scope 时必填）';
COMMENT ON COLUMN roles.namespace_id IS '所属项目 ID（namespace scope 时必填）';
```

**Step 3: Commit**

```bash
git add pkg/db/schema/schema.sql
git commit -m "feat(rbac): add workspace_id/namespace_id columns to roles table"
```

---

### Task 2: Update sqlc Queries for Roles

**Files:**
- Modify: `pkg/db/query/role.sql`

**Step 1: Update CreateRole query**

Add `workspace_id` and `namespace_id` params:

```sql
-- name: CreateRole :one
INSERT INTO roles (name, display_name, description, scope, builtin, workspace_id, namespace_id)
VALUES (@name, @display_name, @description, @scope, @builtin, @workspace_id, @namespace_id)
RETURNING id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at;
```

**Step 2: Update GetRoleByID to return new columns**

```sql
-- name: GetRoleByID :one
SELECT id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at
FROM roles
WHERE id = @id;
```

**Step 3: Update GetRoleByName to only match platform roles**

The old `GetRoleByName` is only used for platform-scope lookups (SeedRBAC, etc.). Restrict it:

```sql
-- name: GetRoleByName :one
SELECT id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at
FROM roles
WHERE name = @name AND scope = 'platform';
```

**Step 4: Add scoped name lookup queries**

```sql
-- name: GetRoleByNameAndWorkspace :one
SELECT id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at
FROM roles
WHERE name = @name AND scope = 'workspace' AND workspace_id = @workspace_id;

-- name: GetRoleByNameAndNamespace :one
SELECT id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at
FROM roles
WHERE name = @name AND scope = 'namespace' AND namespace_id = @namespace_id;
```

**Step 5: Update UpdateRole to return new columns**

```sql
-- name: UpdateRole :one
UPDATE roles
SET display_name = @display_name,
    description = @description,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at;
```

**Step 6: Update UpsertRole for platform-only (name-based conflict)**

Since UpsertRole is only used in SeedRBAC for platform roles, scope the conflict clause:

```sql
-- name: UpsertRole :one
INSERT INTO roles (name, display_name, description, scope, builtin)
VALUES (@name, @display_name, @description, @scope, @builtin)
ON CONFLICT (name) WHERE scope = 'platform'
DO UPDATE SET display_name = EXCLUDED.display_name,
              description = EXCLUDED.description,
              updated_at = now()
RETURNING id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at;
```

**Step 7: Update CountRoles and ListRoles to filter by workspace_id/namespace_id**

```sql
-- name: CountRoles :one
SELECT count(id)
FROM roles
WHERE (sqlc.narg('scope')::VARCHAR IS NULL OR scope = sqlc.narg('scope'))
  AND (sqlc.narg('builtin')::BOOLEAN IS NULL OR builtin = sqlc.narg('builtin'))
  AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR workspace_id = sqlc.narg('workspace_id'))
  AND (sqlc.narg('namespace_id')::BIGINT IS NULL OR namespace_id = sqlc.narg('namespace_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       name ILIKE '%' || sqlc.narg('search') || '%'
       OR display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListRoles :many
SELECT id, name, display_name, description, scope, builtin, workspace_id, namespace_id, created_at, updated_at,
       (SELECT COUNT(*) FROM role_permission_rules WHERE role_id = roles.id)::INT AS rule_count
FROM roles
WHERE (sqlc.narg('scope')::VARCHAR IS NULL OR scope = sqlc.narg('scope'))
  AND (sqlc.narg('builtin')::BOOLEAN IS NULL OR builtin = sqlc.narg('builtin'))
  AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR workspace_id = sqlc.narg('workspace_id'))
  AND (sqlc.narg('namespace_id')::BIGINT IS NULL OR namespace_id = sqlc.narg('namespace_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       name ILIKE '%' || sqlc.narg('search') || '%'
       OR display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'scope' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN scope END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'scope' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN scope END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
```

**Step 8: Add query to list all workspaces/namespaces (for SeedRBAC migration)**

```sql
-- name: ListAllWorkspaceIDs :many
SELECT id FROM workspaces;

-- name: ListAllNamespaceIDsWithWorkspace :many
SELECT id, workspace_id FROM namespaces;
```

**Step 9: Run sqlc generate**

Run: `make sqlc-generate`
Expected: Clean generation with updated types including `WorkspaceID *int64` and `NamespaceID *int64` on `generated.Role`, `generated.ListRolesRow`, etc.

**Step 10: Commit**

```bash
git add pkg/db/query/role.sql pkg/db/generated/
git commit -m "feat(rbac): update role sqlc queries with workspace_id/namespace_id filters"
```

---

### Task 3: Update RoleStore Interface and Implementation

**Files:**
- Modify: `pkg/apis/iam/store.go:73-85`
- Modify: `pkg/apis/iam/store/pg_role.go`

**Step 1: Update RoleStore interface**

Add new methods to `pkg/apis/iam/store.go`:

```go
// RoleStore defines database operations on roles.
type RoleStore interface {
	Create(ctx context.Context, role *DBRole) (*DBRole, error)
	GetByID(ctx context.Context, id int64) (*DBRoleWithRules, error)
	GetByName(ctx context.Context, name string) (*DBRole, error)
	GetByNameAndWorkspace(ctx context.Context, name string, workspaceID int64) (*DBRole, error)
	GetByNameAndNamespace(ctx context.Context, name string, namespaceID int64) (*DBRole, error)
	Update(ctx context.Context, role *DBRole) (*DBRole, error)
	Upsert(ctx context.Context, role *DBRole) (*DBRole, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, query db.ListQuery) (*db.ListResult[DBRoleListRow], error)
	SetPermissionRules(ctx context.Context, roleID int64, patterns []string) error
	CreateBuiltinRolesForWorkspace(ctx context.Context, workspaceID int64) error
	CreateBuiltinRolesForNamespace(ctx context.Context, namespaceID int64) error
	// SeedRBAC upserts platform built-in roles and migrates existing scoped data.
	SeedRBAC(ctx context.Context, roles []BuiltinRoleDef, adminUsername string) error
}
```

**Step 2: Update pgRoleStore.Create**

Add `workspace_id` and `namespace_id` to `CreateRoleParams`:

```go
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
	return &row, nil
}
```

**Step 3: Implement GetByNameAndWorkspace / GetByNameAndNamespace**

```go
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
	return &row, nil
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
	return &row, nil
}
```

**Step 4: Implement CreateBuiltinRolesForWorkspace**

This creates workspace-admin + workspace-viewer roles with their rules, all in a transaction:

```go
func (s *pgRoleStore) CreateBuiltinRolesForWorkspace(ctx context.Context, workspaceID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)

	wsID := &workspaceID
	for _, def := range iam.WorkspaceBuiltinRoles() {
		role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Scope:       "workspace",
			Builtin:     true,
			WorkspaceID: wsID,
		})
		if err != nil {
			return fmt.Errorf("create builtin role %s for workspace %d: %w", def.Name, workspaceID, err)
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

	return tx.Commit(ctx)
}
```

**Step 5: Implement CreateBuiltinRolesForNamespace**

Same pattern, but for namespace-admin + namespace-viewer with `namespace_id`:

```go
func (s *pgRoleStore) CreateBuiltinRolesForNamespace(ctx context.Context, namespaceID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)

	nsID := &namespaceID
	for _, def := range iam.NamespaceBuiltinRoles() {
		role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			Description: def.Description,
			Scope:       "namespace",
			Builtin:     true,
			NamespaceID: nsID,
		})
		if err != nil {
			return fmt.Errorf("create builtin role %s for namespace %d: %w", def.Name, namespaceID, err)
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

	return tx.Commit(ctx)
}
```

**Step 6: Update List to pass new filter params**

In `pgRoleStore.List()`, add workspace_id / namespace_id to both count and list params:

```go
countParams := generated.CountRolesParams{
	Scope:       filterStr(q.Filters, "scope"),
	Builtin:     filterBool(q.Filters, "builtin"),
	WorkspaceID: filterInt64(q.Filters, "workspace_id"),
	NamespaceID: filterInt64(q.Filters, "namespace_id"),
	Search:      filterStr(q.Filters, "search"),
}
// ... and similarly for ListRolesParams
```

**Step 7: Verify build**

Run: `go build ./...`

**Step 8: Commit**

```bash
git add pkg/apis/iam/store.go pkg/apis/iam/store/pg_role.go
git commit -m "feat(rbac): add scoped role store methods and workspace/namespace filtering"
```

---

### Task 4: Update rbac_seed.go — Platform-Only Seeding + Migration

**Files:**
- Modify: `pkg/apis/iam/rbac_seed.go`
- Modify: `pkg/apis/iam/store/pg_role.go` (SeedRBAC method)

**Step 1: Split builtinRoles into scope-specific helpers**

In `rbac_seed.go`, keep the full list but add helper functions:

```go
// PlatformBuiltinRoles returns the platform-scope built-in role definitions.
func PlatformBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RolePlatformAdmin, DisplayName: "Platform Admin", Description: "Full access to all platform resources", Scope: "platform", Rules: []string{"*:*"}},
		{Name: RolePlatformViewer, DisplayName: "Platform Viewer", Description: "Read-only access to all platform resources", Scope: "platform", Rules: []string{"*:list", "*:get"}},
	}
}

// WorkspaceBuiltinRoles returns the workspace-scope built-in role definitions.
func WorkspaceBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RoleWorkspaceAdmin, DisplayName: "Workspace Admin", Description: "Full access to all resources within the workspace", Scope: "workspace", Rules: []string{"*:*"}},
		{Name: RoleWorkspaceViewer, DisplayName: "Workspace Viewer", Description: "Read-only access to all resources within the workspace", Scope: "workspace", Rules: []string{"*:list", "*:get"}},
	}
}

// NamespaceBuiltinRoles returns the namespace-scope built-in role definitions.
func NamespaceBuiltinRoles() []BuiltinRoleDef {
	return []BuiltinRoleDef{
		{Name: RoleNamespaceAdmin, DisplayName: "Namespace Admin", Description: "Full access to all resources within the namespace", Scope: "namespace", Rules: []string{"*:*"}},
		{Name: RoleNamespaceViewer, DisplayName: "Namespace Viewer", Description: "Read-only access to all resources within the namespace", Scope: "namespace", Rules: []string{"*:list", "*:get"}},
	}
}
```

**Step 2: Update SeedRBAC to only seed platform roles + migrate existing data**

In `rbac_seed.go`, change `SeedRBAC()`:

```go
func SeedRBAC(ctx context.Context, roleStore RoleStore) error {
	if err := roleStore.SeedRBAC(ctx, PlatformBuiltinRoles(), "admin"); err != nil {
		return err
	}
	logger.Infof("seeded platform built-in roles with initial bindings")
	return nil
}
```

**Step 3: Rewrite SeedRBAC in pg_role.go**

The new `SeedRBAC` method:
1. Upserts platform roles only
2. Iterates all existing workspaces → creates built-in roles if missing
3. Iterates all existing namespaces → creates built-in roles if missing
4. Migrates existing role_bindings from old global roles to new scoped roles
5. Deletes old global workspace/namespace roles

```go
func (s *pgRoleStore) SeedRBAC(ctx context.Context, roles []iam.BuiltinRoleDef, adminUsername string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)

	// 1. Upsert platform roles
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
				RoleID: role.ID, Pattern: pattern,
			}); err != nil {
				return fmt.Errorf("add rule %q for role %s: %w", pattern, def.Name, err)
			}
		}
	}

	// 2. Create platform-admin binding for admin user
	if adminUsername != "" && platformAdminRoleID != 0 {
		adminUser, err := qtx.GetUserByUsername(ctx, adminUsername)
		if err == nil {
			_ = qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
				UserID: adminUser.ID, RoleID: platformAdminRoleID, Scope: "platform",
			})
		}
	}

	// 3. Migrate: for each existing workspace, ensure built-in roles exist
	wsIDs, err := qtx.ListAllWorkspaceIDs(ctx)
	if err != nil {
		return fmt.Errorf("list workspace ids: %w", err)
	}
	for _, wsID := range wsIDs {
		for _, def := range iam.WorkspaceBuiltinRoles() {
			wsIDPtr := &wsID
			// Check if already exists
			_, err := qtx.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
				Name: def.Name, WorkspaceID: wsIDPtr,
			})
			if err == nil {
				continue // already exists
			}
			role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
				Name: def.Name, DisplayName: def.DisplayName, Description: def.Description,
				Scope: "workspace", Builtin: true, WorkspaceID: wsIDPtr,
			})
			if err != nil {
				return fmt.Errorf("create builtin role %s for workspace %d: %w", def.Name, wsID, err)
			}
			for _, pattern := range def.Rules {
				if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
					RoleID: role.ID, Pattern: pattern,
				}); err != nil {
					return fmt.Errorf("add rule: %w", err)
				}
			}
		}
	}

	// 4. Same for namespaces
	nsRows, err := qtx.ListAllNamespaceIDsWithWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("list namespace ids: %w", err)
	}
	for _, nsRow := range nsRows {
		for _, def := range iam.NamespaceBuiltinRoles() {
			nsIDPtr := &nsRow.ID
			_, err := qtx.GetRoleByNameAndNamespace(ctx, generated.GetRoleByNameAndNamespaceParams{
				Name: def.Name, NamespaceID: nsIDPtr,
			})
			if err == nil {
				continue
			}
			role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
				Name: def.Name, DisplayName: def.DisplayName, Description: def.Description,
				Scope: "namespace", Builtin: true, NamespaceID: nsIDPtr,
			})
			if err != nil {
				return fmt.Errorf("create builtin role %s for namespace %d: %w", def.Name, nsRow.ID, err)
			}
			for _, pattern := range def.Rules {
				if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
					RoleID: role.ID, Pattern: pattern,
				}); err != nil {
					return fmt.Errorf("add rule: %w", err)
				}
			}
		}
	}

	// 5. Migrate existing role_bindings from old global roles to new scoped roles
	// Find old global workspace/namespace roles (those with workspace_id IS NULL AND scope != 'platform')
	// Re-point their bindings, then delete the old roles
	// Use raw SQL since this is a one-time migration
	migrateScopes := []struct {
		scope      string
		scopeCol   string
		roleDefs   []iam.BuiltinRoleDef
	}{
		{"workspace", "workspace_id", iam.WorkspaceBuiltinRoles()},
		{"namespace", "namespace_id", iam.NamespaceBuiltinRoles()},
	}
	for _, ms := range migrateScopes {
		for _, def := range ms.roleDefs {
			// Find old global role
			var oldRoleID int64
			err := tx.QueryRow(ctx,
				"SELECT id FROM roles WHERE name = $1 AND scope = $2 AND workspace_id IS NULL AND namespace_id IS NULL",
				def.Name, ms.scope,
			).Scan(&oldRoleID)
			if err != nil {
				continue // no old global role, skip
			}
			// Re-point bindings to new scoped roles
			rows, _ := tx.Query(ctx,
				fmt.Sprintf("SELECT id, %s FROM role_bindings WHERE role_id = $1", ms.scopeCol),
				oldRoleID,
			)
			for rows.Next() {
				var bindingID, resourceID int64
				var resourceIDPtr *int64
				if err := rows.Scan(&bindingID, &resourceIDPtr); err != nil || resourceIDPtr == nil {
					continue
				}
				resourceID = *resourceIDPtr
				// Find new scoped role
				var newRoleQuery string
				if ms.scope == "workspace" {
					newRoleQuery = "SELECT id FROM roles WHERE name = $1 AND workspace_id = $2"
				} else {
					newRoleQuery = "SELECT id FROM roles WHERE name = $1 AND namespace_id = $2"
				}
				var newRoleID int64
				if err := tx.QueryRow(ctx, newRoleQuery, def.Name, resourceID).Scan(&newRoleID); err != nil {
					continue
				}
				_, _ = tx.Exec(ctx, "UPDATE role_bindings SET role_id = $1 WHERE id = $2", newRoleID, bindingID)
			}
			rows.Close()
			// Delete old global role (cascades to permission rules)
			_, _ = tx.Exec(ctx, "DELETE FROM roles WHERE id = $1", oldRoleID)
		}
	}

	return tx.Commit(ctx)
}
```

**Step 4: Verify build**

Run: `go build ./...`

**Step 5: Commit**

```bash
git add pkg/apis/iam/rbac_seed.go pkg/apis/iam/store/pg_role.go
git commit -m "feat(rbac): update SeedRBAC for platform-only seeding with workspace/namespace migration"
```

---

### Task 5: Update Workspace/Namespace Store Create Transactions

**Files:**
- Modify: `pkg/apis/iam/store/pg_workspace.go:27-120`
- Modify: `pkg/apis/iam/store/pg_namespace.go:29-80`

**Step 1: Update pgWorkspaceStore.Create**

Replace the `GetRoleByName` calls with inline role creation within the same transaction:

```go
func (s *pgWorkspaceStore) Create(ctx context.Context, ws *iam.DBWorkspace) (*iam.DBWorkspaceWithOwner, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)

	// Create workspace
	row, err := qtx.CreateWorkspace(ctx, generated.CreateWorkspaceParams{...})
	// ... existing error handling ...

	// Create workspace built-in roles
	wsID := &row.ID
	var wsAdminRoleID int64
	for _, def := range iam.WorkspaceBuiltinRoles() {
		role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name: def.Name, DisplayName: def.DisplayName, Description: def.Description,
			Scope: "workspace", Builtin: true, WorkspaceID: wsID,
		})
		if err != nil {
			return nil, fmt.Errorf("create builtin role %s: %w", def.Name, err)
		}
		if def.Name == iam.RoleWorkspaceAdmin {
			wsAdminRoleID = role.ID
		}
		for _, pattern := range def.Rules {
			if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
				RoleID: role.ID, Pattern: pattern,
			}); err != nil {
				return nil, fmt.Errorf("add rule %q: %w", pattern, err)
			}
		}
	}

	// Create default namespace
	defaultNS, err := qtx.CreateNamespace(ctx, ...)
	// ... existing ...

	// Create namespace built-in roles for default namespace
	nsID := &defaultNS.ID
	var nsAdminRoleID int64
	for _, def := range iam.NamespaceBuiltinRoles() {
		role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name: def.Name, DisplayName: def.DisplayName, Description: def.Description,
			Scope: "namespace", Builtin: true, NamespaceID: nsID,
		})
		if err != nil {
			return nil, fmt.Errorf("create builtin role %s: %w", def.Name, err)
		}
		if def.Name == iam.RoleNamespaceAdmin {
			nsAdminRoleID = role.ID
		}
		for _, pattern := range def.Rules {
			if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
				RoleID: role.ID, Pattern: pattern,
			}); err != nil {
				return nil, fmt.Errorf("add rule %q: %w", pattern, err)
			}
		}
	}

	// Bind owner as workspace-admin (is_owner=true)
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID: ws.OwnerID, RoleID: wsAdminRoleID,
		Scope: "workspace", WorkspaceID: wsID, IsOwner: true,
	}); err != nil {
		return nil, fmt.Errorf("create workspace owner binding: %w", err)
	}

	// Bind owner as namespace-admin for default namespace (is_owner=true)
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID: ws.OwnerID, RoleID: nsAdminRoleID,
		Scope: "namespace", WorkspaceID: wsID, NamespaceID: nsID, IsOwner: true,
	}); err != nil {
		return nil, fmt.Errorf("create namespace owner binding: %w", err)
	}

	// Commit + fetch
	// ... rest same as before ...
}
```

**Step 2: Update pgNamespaceStore.Create**

Same pattern — create namespace built-in roles within the transaction, bind owner to the new namespace-admin role:

```go
func (s *pgNamespaceStore) Create(ctx context.Context, ns *iam.DBNamespace) (*iam.DBNamespaceWithOwner, error) {
	tx, err := s.db.Begin(ctx)
	// ...
	qtx := s.queries.WithTx(tx)

	row, err := qtx.CreateNamespace(ctx, ...)
	// ...

	// Create namespace built-in roles
	nsID := &row.ID
	var nsAdminRoleID int64
	for _, def := range iam.NamespaceBuiltinRoles() {
		role, err := qtx.CreateRole(ctx, generated.CreateRoleParams{
			Name: def.Name, DisplayName: def.DisplayName, Description: def.Description,
			Scope: "namespace", Builtin: true, NamespaceID: nsID,
		})
		if err != nil {
			return nil, fmt.Errorf("create builtin role %s: %w", def.Name, err)
		}
		if def.Name == iam.RoleNamespaceAdmin {
			nsAdminRoleID = role.ID
		}
		for _, pattern := range def.Rules {
			if err := qtx.AddRolePermissionRule(ctx, generated.AddRolePermissionRuleParams{
				RoleID: role.ID, Pattern: pattern,
			}); err != nil {
				return nil, fmt.Errorf("add rule %q: %w", pattern, err)
			}
		}
	}

	// Bind owner
	if err := qtx.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID: ns.OwnerID, RoleID: nsAdminRoleID,
		Scope: "namespace", WorkspaceID: &ns.WorkspaceID, NamespaceID: nsID, IsOwner: true,
	}); err != nil {
		return nil, fmt.Errorf("create namespace owner binding: %w", err)
	}

	// Commit + fetch ...
}
```

**Step 3: Verify build**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add pkg/apis/iam/store/pg_workspace.go pkg/apis/iam/store/pg_namespace.go
git commit -m "feat(rbac): create built-in roles in workspace/namespace create transactions"
```

---

### Task 6: Update RoleBindingStore — Scoped Role Lookups

**Files:**
- Modify: `pkg/apis/iam/store/pg_role_binding.go`

**Step 1: Update AddWorkspaceMember**

Change from `GetRoleByName` (global lookup) to `GetRoleByNameAndWorkspace`:

```go
func (s *pgRoleBindingStore) AddWorkspaceMember(ctx context.Context, userID, workspaceID int64) error {
	wsIDPtr := &workspaceID
	viewerRole, err := s.queries.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
		Name:        iam.RoleWorkspaceViewer,
		WorkspaceID: wsIDPtr,
	})
	if err != nil {
		return fmt.Errorf("get workspace-viewer role for workspace %d: %w", workspaceID, err)
	}
	if err := s.queries.CreateRoleBindingIfNotExists(ctx, generated.CreateRoleBindingIfNotExistsParams{
		UserID:      userID,
		RoleID:      viewerRole.ID,
		Scope:       "workspace",
		WorkspaceID: wsIDPtr,
	}); err != nil {
		return fmt.Errorf("add workspace member: %w", err)
	}
	return nil
}
```

**Step 2: Update AddNamespaceMember**

Change both role lookups to scoped:

```go
func (s *pgRoleBindingStore) AddNamespaceMember(ctx context.Context, userID, namespaceID int64) error {
	tx, err := s.db.Begin(ctx)
	// ...
	qtx := s.queries.WithTx(tx)

	// Get namespace to find workspace_id
	var wsID int64
	// ... existing query ...

	wsIDPtr := &wsID
	nsIDPtr := &namespaceID

	// Scoped lookup
	wsViewerRole, err := qtx.GetRoleByNameAndWorkspace(ctx, generated.GetRoleByNameAndWorkspaceParams{
		Name: iam.RoleWorkspaceViewer, WorkspaceID: wsIDPtr,
	})
	// ...
	nsViewerRole, err := qtx.GetRoleByNameAndNamespace(ctx, generated.GetRoleByNameAndNamespaceParams{
		Name: iam.RoleNamespaceViewer, NamespaceID: nsIDPtr,
	})
	// ...

	// Auto-add to workspace + namespace (same binding logic as before)
	// ...
}
```

**Step 3: Update TransferOwnership**

Line 417 currently does a global role name lookup:
```go
if err := tx.QueryRow(ctx, "SELECT id FROM roles WHERE name = $1", adminRoleName).Scan(&adminRoleID); err != nil {
```

Change to scoped lookup:
```go
var adminRoleQuery string
if scope == "workspace" {
	adminRoleQuery = "SELECT id FROM roles WHERE name = $1 AND workspace_id = $2"
} else {
	adminRoleQuery = "SELECT id FROM roles WHERE name = $1 AND namespace_id = $2"
}
if err := tx.QueryRow(ctx, adminRoleQuery, adminRoleName, resourceID).Scan(&adminRoleID); err != nil {
```

**Step 4: Verify build**

Run: `go build ./...`

**Step 5: Commit**

```bash
git add pkg/apis/iam/store/pg_role_binding.go
git commit -m "feat(rbac): update role binding store to use scoped role lookups"
```

---

### Task 7: Update scopedRoleStorage — Full CRUD

**Files:**
- Modify: `pkg/apis/iam/storage.go:1891-1952`

**Step 1: Rewrite scopedRoleStorage as full CRUD**

Replace the current read-only `scopedRoleStorage` with full CRUD:

```go
type scopedRoleStorage struct {
	roleStore    RoleStore
	rbStore      RoleBindingStore
	scope        string // "workspace" or "namespace"
}

func NewScopedRoleStorage(roleStore RoleStore, rbStore RoleBindingStore, scope string) rest.Storage {
	return &scopedRoleStorage{roleStore: roleStore, rbStore: rbStore, scope: scope}
}

func (s *scopedRoleStorage) NewObject() runtime.Object { return &Role{} }
```

**Step 2: Implement List — filter by resource ID**

```go
func (s *scopedRoleStorage) List(ctx context.Context, options *rest.ListOptions) (runtime.Object, error) {
	query := restOptionsToListQuery(options)
	query.Filters["scope"] = s.scope

	if s.scope == "workspace" {
		wsID, err := parseID(options.PathParams["workspaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
		}
		query.Filters["workspace_id"] = wsID
	} else {
		nsID, err := parseID(options.PathParams["namespaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
		}
		query.Filters["namespace_id"] = nsID
	}

	result, err := s.roleStore.List(ctx, query)
	if err != nil {
		return nil, err
	}
	items := make([]Role, len(result.Items))
	for i, item := range result.Items {
		items[i] = *roleListRowToAPI(&item)
	}
	return &RoleList{
		TypeMeta: runtime.TypeMeta{Kind: "RoleList"}, Items: items, TotalCount: result.TotalCount,
	}, nil
}
```

**Step 3: Implement Get — verify role belongs to scoped resource**

```go
func (s *scopedRoleStorage) Get(ctx context.Context, options *rest.GetOptions) (runtime.Object, error) {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}
	role, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	// Verify scope and resource ownership
	if role.Scope != s.scope {
		return nil, apierrors.NewNotFound("role", id)
	}
	if s.scope == "workspace" {
		wsID, _ := parseID(options.PathParams["workspaceId"])
		if role.WorkspaceID == nil || *role.WorkspaceID != wsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	} else {
		nsID, _ := parseID(options.PathParams["namespaceId"])
		if role.NamespaceID == nil || *role.NamespaceID != nsID {
			return nil, apierrors.NewNotFound("role", id)
		}
	}
	return roleWithRulesToAPI(role), nil
}
```

**Step 4: Implement Create**

```go
func (s *scopedRoleStorage) Create(ctx context.Context, obj runtime.Object, options *rest.CreateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}
	// Force scope
	role.Spec.Scope = s.scope
	if errs := ValidateRoleCreate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}
	if options.DryRun {
		return role, nil
	}
	dbRole := &DBRole{
		Name: role.Spec.Name, DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description, Scope: s.scope,
	}
	if s.scope == "workspace" {
		wsID, err := parseID(options.PathParams["workspaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid workspace ID", nil)
		}
		dbRole.WorkspaceID = &wsID
	} else {
		nsID, err := parseID(options.PathParams["namespaceId"])
		if err != nil {
			return nil, apierrors.NewBadRequest("invalid namespace ID", nil)
		}
		dbRole.NamespaceID = &nsID
	}

	created, err := s.roleStore.Create(ctx, dbRole)
	if err != nil {
		return nil, err
	}
	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, created.ID, role.Spec.Rules); err != nil {
			return nil, err
		}
	}
	withRules, err := s.roleStore.GetByID(ctx, created.ID)
	if err != nil {
		return nil, err
	}
	return roleWithRulesToAPI(withRules), nil
}
```

**Step 5: Implement Update**

```go
func (s *scopedRoleStorage) Update(ctx context.Context, obj runtime.Object, options *rest.UpdateOptions) (runtime.Object, error) {
	role, ok := obj.(*Role)
	if !ok {
		return nil, fmt.Errorf("expected *Role, got %T", obj)
	}
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}
	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if existing.Builtin {
		return nil, apierrors.NewBadRequest("cannot modify built-in role", nil)
	}
	// Verify ownership (same scope check as Get)
	// ... same verification as Get ...

	if errs := ValidateRoleUpdate(&role.Spec); errs.HasErrors() {
		return nil, apierrors.NewBadRequest("validation failed", errs)
	}
	if options.DryRun {
		return role, nil
	}
	dbRole := &DBRole{ID: rid, DisplayName: role.Spec.DisplayName, Description: role.Spec.Description}
	if _, err := s.roleStore.Update(ctx, dbRole); err != nil {
		return nil, err
	}
	if len(role.Spec.Rules) > 0 {
		if err := s.roleStore.SetPermissionRules(ctx, rid, role.Spec.Rules); err != nil {
			return nil, err
		}
	}
	withRules, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	return roleWithRulesToAPI(withRules), nil
}
```

**Step 6: Implement Delete**

```go
func (s *scopedRoleStorage) Delete(ctx context.Context, options *rest.DeleteOptions) error {
	id := options.PathParams["roleId"]
	rid, err := parseID(id)
	if err != nil {
		return apierrors.NewBadRequest(fmt.Sprintf("invalid role ID: %s", id), nil)
	}
	existing, err := s.roleStore.GetByID(ctx, rid)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return apierrors.NewBadRequest("cannot delete built-in role", nil)
	}
	// Verify ownership
	// ... same verification as Get ...

	// Check no bindings exist
	count, err := s.rbStore.CountByRoleAndScope(ctx, rid, s.scope)
	if err != nil {
		return err
	}
	if count > 0 {
		return apierrors.NewBadRequest("cannot delete role with active bindings", nil)
	}
	if options.DryRun {
		return nil
	}
	return s.roleStore.Delete(ctx, rid)
}
```

**Step 7: Verify build**

Run: `go build ./...`

**Step 8: Commit**

```bash
git add pkg/apis/iam/storage.go
git commit -m "feat(rbac): convert scopedRoleStorage to full CRUD with resource ownership verification"
```

---

### Task 8: Update Route Registration and Wiring

**Files:**
- Modify: `pkg/apis/iam/v1/install.go:54-55`

**Step 1: Update NewScopedRoleStorage calls to pass rbStore**

```go
wsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, "workspace")
nsRoleStorage := iam.NewScopedRoleStorage(p.Role, p.RoleBinding, "namespace")
```

**Step 2: Update RoleBinding validation in workspaceRoleBindingStorage.Create**

In `storage.go`, add workspace_id verification when creating workspace-level role bindings:

```go
// After fetching the role:
if role.WorkspaceID == nil || *role.WorkspaceID != wsID {
	return nil, apierrors.NewBadRequest("role does not belong to this workspace", nil)
}
```

Similarly in `namespaceRoleBindingStorage.Create`:
```go
if role.NamespaceID == nil || *role.NamespaceID != nsID {
	return nil, apierrors.NewBadRequest("role does not belong to this namespace", nil)
}
```

**Step 3: Update platform roleStorage.List**

In `roleStorage.List()`, force platform scope filter so platform role list doesn't show scoped roles:

```go
query.Filters["scope"] = "platform"
```

**Step 4: Update platform roleStorage.Create validation**

Ensure platform role creation forces scope = "platform":

Already validates `scope` in `ValidateRoleCreate`, but add explicit check:

```go
if role.Spec.Scope != "platform" {
	return nil, apierrors.NewBadRequest("platform role storage only accepts scope 'platform'", nil)
}
```

Or simpler: force `role.Spec.Scope = "platform"` before validation.

**Step 5: Verify build**

Run: `go build ./...`

**Step 6: Commit**

```bash
git add pkg/apis/iam/v1/install.go pkg/apis/iam/storage.go
git commit -m "feat(rbac): update route wiring and role binding validation for scoped roles"
```

---

### Task 9: Regenerate sqlc, Build, and Verify

**Step 1: Regenerate sqlc**

Run: `make sqlc-generate`

**Step 2: Build**

Run: `go build ./...`

**Step 3: Run vet**

Run: `make vet`

**Step 4: Run tests**

Run: `make test`

**Step 5: Fix any compilation errors and re-verify**

**Step 6: Commit any remaining fixes**

```bash
git add -A
git commit -m "fix(rbac): fix compilation issues after role scope binding changes"
```

---

### Task 10: Regenerate OpenAPI Spec

**Step 1: Run OpenAPI generator**

Run: `make openapi-gen`

**Step 2: Verify generated spec**

Check that:
- `/workspaces/{workspaceId}/roles` now shows POST, PUT, DELETE operations
- `/namespaces/{namespaceId}/roles` now shows POST, PUT, DELETE operations
- `/roles` still shows only platform CRUD

**Step 3: Commit**

```bash
git add docs/ cmd/openapi-gen/
git commit -m "docs(rbac): regenerate OpenAPI spec with scoped role CRUD"
```

---

### Task 11: Drop Database and End-to-End Verification

**Step 1: Drop and recreate database**

Since schema changed, recreate the database:
```bash
docker exec -i lcp-postgres psql -U lcp -d postgres -c "DROP DATABASE IF EXISTS lcp;" && \
docker exec -i lcp-postgres psql -U lcp -d postgres -c "CREATE DATABASE lcp;" && \
docker exec -i lcp-postgres psql -U lcp -d lcp < pkg/db/schema/schema.sql
```

**Step 2: Start server**

Run: `go run ./app/lcp-server/ -config ./app/lcp-server/config.yaml`

**Step 3: Create test user and workspace, verify roles**

```bash
# Seed will have created platform roles
curl -s localhost:8428/api/iam/v1/roles | jq '.items[].spec.name'
# Expected: platform-admin, platform-viewer

# Create a workspace (auto-creates workspace built-in roles)
curl -s -X POST localhost:8428/api/iam/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"metadata":{"name":"test-ws"},"spec":{"displayName":"Test WS"}}'

# Verify workspace roles exist
curl -s localhost:8428/api/iam/v1/workspaces/1/roles | jq '.items[].spec.name'
# Expected: workspace-admin, workspace-viewer

# Verify platform roles are NOT shown
curl -s localhost:8428/api/iam/v1/workspaces/1/roles | jq '.totalCount'
# Expected: 2

# Verify namespace roles exist for the default namespace
curl -s localhost:8428/api/iam/v1/namespaces/1/roles | jq '.items[].spec.name'
# Expected: namespace-admin, namespace-viewer
```

**Step 4: Test custom role CRUD on workspace**

```bash
# Create custom role in workspace
curl -s -X POST localhost:8428/api/iam/v1/workspaces/1/roles \
  -H 'Content-Type: application/json' \
  -d '{"spec":{"name":"project-lead","displayName":"Project Lead","rules":["*:list","*:get","iam:*"]}}'
# Expected: 201 with role

# List workspace roles again
curl -s localhost:8428/api/iam/v1/workspaces/1/roles | jq '.totalCount'
# Expected: 3

# Other workspace should not see this role
# (create another workspace and check)
```
