-- name: CreateRoleBinding :one
INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, namespace_id, is_owner)
VALUES (@user_id, @role_id, @scope, @workspace_id, @namespace_id, @is_owner)
RETURNING id, user_id, role_id, scope, workspace_id, namespace_id, is_owner, created_at;

-- name: CreateRoleBindingIfNotExists :exec
INSERT INTO role_bindings (user_id, role_id, scope, workspace_id, namespace_id, is_owner)
VALUES (@user_id, @role_id, @scope, @workspace_id, @namespace_id, @is_owner)
ON CONFLICT DO NOTHING;

-- name: DeleteRoleBinding :exec
DELETE FROM role_bindings WHERE id = @id;

-- name: GetRoleBindingByID :one
SELECT id, user_id, role_id, scope, workspace_id, namespace_id, is_owner, created_at
FROM role_bindings
WHERE id = @id;

-- name: CountRoleBindingsPlatform :one
SELECT count(rb.id)
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'platform'
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListRoleBindingsPlatform :many
SELECT rb.id, rb.user_id, rb.role_id, rb.scope, rb.workspace_id, rb.namespace_id, rb.is_owner, rb.created_at,
       u.username, u.display_name AS user_display_name,
       r.name AS role_name, r.display_name AS role_display_name
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'platform'
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN r.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN r.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN rb.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN rb.created_at END DESC,
    rb.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountRoleBindingsByWorkspaceID :one
SELECT count(rb.id)
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListRoleBindingsByWorkspaceID :many
SELECT rb.id, rb.user_id, rb.role_id, rb.scope, rb.workspace_id, rb.namespace_id, rb.is_owner, rb.created_at,
       u.username, u.display_name AS user_display_name,
       r.name AS role_name, r.display_name AS role_display_name
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN r.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN r.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN rb.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN rb.created_at END DESC,
    rb.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountRoleBindingsByNamespaceID :one
SELECT count(rb.id)
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'namespace' AND rb.namespace_id = @namespace_id
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListRoleBindingsByNamespaceID :many
SELECT rb.id, rb.user_id, rb.role_id, rb.scope, rb.workspace_id, rb.namespace_id, rb.is_owner, rb.created_at,
       u.username, u.display_name AS user_display_name,
       r.name AS role_name, r.display_name AS role_display_name
FROM role_bindings rb
JOIN users u ON u.id = rb.user_id
JOIN roles r ON r.id = rb.role_id
WHERE rb.scope = 'namespace' AND rb.namespace_id = @namespace_id
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('is_owner')::BOOLEAN IS NULL OR rb.is_owner = sqlc.narg('is_owner'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN r.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN r.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN rb.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN rb.created_at END DESC,
    rb.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountRoleBindingsByUserID :one
SELECT count(rb.id)
FROM role_bindings rb
JOIN roles r ON r.id = rb.role_id
WHERE rb.user_id = @user_id
  AND (sqlc.narg('scope')::VARCHAR IS NULL OR rb.scope = sqlc.narg('scope'))
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListRoleBindingsByUserID :many
SELECT rb.id, rb.user_id, rb.role_id, rb.scope, rb.workspace_id, rb.namespace_id, rb.is_owner, rb.created_at,
       r.name AS role_name, r.display_name AS role_display_name
FROM role_bindings rb
JOIN roles r ON r.id = rb.role_id
WHERE rb.user_id = @user_id
  AND (sqlc.narg('scope')::VARCHAR IS NULL OR rb.scope = sqlc.narg('scope'))
  AND (sqlc.narg('role_id')::BIGINT IS NULL OR rb.role_id = sqlc.narg('role_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       r.name ILIKE '%' || sqlc.narg('search') || '%'
       OR r.display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'scope' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN rb.scope END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'scope' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN rb.scope END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN r.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN r.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN rb.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN rb.created_at END DESC,
    rb.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountRoleBindingsByRoleAndScope :one
SELECT count(id)
FROM role_bindings
WHERE role_id = @role_id AND scope = @scope;

-- name: GetAccessibleWorkspaceIDs :many
SELECT DISTINCT workspace_id
FROM role_bindings
WHERE user_id = @user_id AND workspace_id IS NOT NULL;

-- name: GetAccessibleNamespaceIDs :many
SELECT DISTINCT namespace_id
FROM role_bindings
WHERE user_id = @user_id AND namespace_id IS NOT NULL;

-- name: GetUserIDsByWorkspaceID :many
SELECT DISTINCT user_id
FROM role_bindings
WHERE workspace_id = @workspace_id;

-- name: GetUserIDsByNamespaceID :many
SELECT DISTINCT user_id
FROM role_bindings
WHERE namespace_id = @namespace_id;

-- name: LoadUserPermissionRules :many
SELECT rb.scope, rb.workspace_id, rb.namespace_id, rpr.pattern
FROM role_bindings rb
JOIN role_permission_rules rpr ON rpr.role_id = rb.role_id
WHERE rb.user_id = @user_id;

-- name: GetUserRoleBindingsWithRules :many
SELECT rb.scope, rb.workspace_id, rb.namespace_id, r.name AS role_name, rpr.pattern
FROM role_bindings rb
JOIN roles r ON r.id = rb.role_id
JOIN role_permission_rules rpr ON rpr.role_id = rb.role_id
WHERE rb.user_id = @user_id;
