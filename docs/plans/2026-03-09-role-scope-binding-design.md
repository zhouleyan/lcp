# Role Scope Binding Design

## Problem

All workspace/namespace-level roles are global — they are not associated with specific workspace/namespace instances. Every workspace sees the same set of workspace-scoped roles, and every namespace sees the same namespace-scoped roles. Users cannot define custom roles per workspace or namespace.

## Decision

Add `workspace_id` and `namespace_id` columns to the `roles` table so that workspace/namespace-level roles belong to specific instances. Platform roles remain global.

## Schema Changes

### `roles` table

Add columns:

```sql
workspace_id BIGINT REFERENCES workspaces(id) ON DELETE CASCADE,
namespace_id BIGINT REFERENCES namespaces(id) ON DELETE CASCADE,
```

Replace the existing `roles_name_key` unique constraint with scope-conditional unique indexes:

```sql
CREATE UNIQUE INDEX uk_roles_platform ON roles(name) WHERE scope = 'platform';
CREATE UNIQUE INDEX uk_roles_workspace ON roles(name, workspace_id) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uk_roles_namespace ON roles(name, namespace_id) WHERE scope = 'namespace';
```

Add CHECK constraint enforcing mutual exclusivity:

```sql
CHECK (
    (scope = 'platform'  AND workspace_id IS NULL AND namespace_id IS NULL) OR
    (scope = 'workspace' AND workspace_id IS NOT NULL AND namespace_id IS NULL) OR
    (scope = 'namespace' AND workspace_id IS NULL AND namespace_id IS NOT NULL)
)
```

`ON DELETE CASCADE` ensures deleting a workspace/namespace automatically cleans up its roles (and cascades through `role_permission_rules` and `role_bindings`).

## Built-in Role Auto-Creation

### Trigger points

- **Create Workspace** → auto-create `workspace-admin` + `workspace-viewer` with `workspace_id` set
- **Create Namespace** → auto-create `namespace-admin` + `namespace-viewer` with `namespace_id` set

### Transaction scope

Built-in role creation happens within the same transaction as the workspace/namespace creation:

- Create workspace → create workspace built-in roles → create default namespace → create namespace built-in roles → bind owner with the new workspace-admin role

### Owner binding adjustment

- Currently binds the creator to a global `workspace-admin` role
- Changed to bind to the **workspace-specific** `workspace-admin` role
- Namespace owner binding follows the same pattern

## API & Route Changes

### Scoped role routes become full CRUD

| Route | Before | After |
|-------|--------|-------|
| `GET /workspaces/{id}/roles` | Read-only (scope filter) | Full CRUD, filtered by workspace_id |
| `GET /namespaces/{id}/roles` | Read-only (scope filter) | Full CRUD, filtered by namespace_id |
| `GET/POST /roles` | Platform roles | No change |

New operations on scoped routes:
- `POST /workspaces/{workspaceId}/roles` — create custom role for this workspace
- `PUT /workspaces/{workspaceId}/roles/{roleId}` — update role
- `DELETE /workspaces/{workspaceId}/roles/{roleId}` — delete (non-builtin, no bindings)
- Same pattern for `/namespaces/{namespaceId}/roles`

### RoleBinding validation

- Creating a workspace-level role binding validates that `role.workspace_id` matches the path `workspaceId`
- Namespace-level binding validates `role.namespace_id` matches the path `namespaceId`

## Store Layer Changes

### RoleStore interface

- `List()` — `Filters` gains `workspace_id` / `namespace_id` keys
- New: `CreateBuiltinRolesForWorkspace(ctx, workspaceID int64) error`
- New: `CreateBuiltinRolesForNamespace(ctx, namespaceID int64) error`
- New: `GetByNameAndWorkspace(ctx, name string, workspaceID int64) (*DBRole, error)`
- New: `GetByNameAndNamespace(ctx, name string, namespaceID int64) (*DBRole, error)`
- `SeedRBAC()` — only seeds platform roles

### RoleBindingStore changes

- `AddWorkspaceMember()` — looks up `workspace-viewer` by name + workspace_id (not global)
- `AddNamespaceMember()` — looks up `namespace-viewer` by name + namespace_id (not global)

### sqlc query changes

- `ListRoles` — add `workspace_id` / `namespace_id` filter params
- `CreateRole` — add `workspace_id` / `namespace_id` input params

## Data Migration

Handled in `SeedRBAC()` at startup (development stage, no separate migration):

1. Upsert platform roles only
2. For each existing workspace: create built-in roles if not present
3. For each existing namespace: create built-in roles if not present
4. Re-point existing role_bindings from old global roles to new scoped roles
5. Delete old global workspace/namespace roles

## scopedRoleStorage changes

Current `scopedRoleStorage` is read-only. Changed to full CRUD:

- `List()` — filter by `workspace_id` or `namespace_id`
- `Get()` — verify role belongs to the scoped resource
- `Create()` — auto-set `workspace_id` / `namespace_id` from path
- `Update()` — disallow scope/owner changes
- `Delete()` — disallow builtin deletion, check no bindings
