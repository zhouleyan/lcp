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

-- ===== Member management queries (replacing user_workspaces / user_namespaces join tables) =====

-- name: CountWorkspaceMembers :one
WITH members AS (
    SELECT DISTINCT ON (rb.user_id) rb.user_id
    FROM role_bindings rb
    WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id
    ORDER BY rb.user_id
)
SELECT count(*)
FROM members m
JOIN users u ON u.id = m.user_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR u.email ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListWorkspaceMembers :many
WITH members AS (
    SELECT DISTINCT ON (rb.user_id)
        rb.user_id,
        r.name AS role_name,
        rb.created_at AS joined_at
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id
    ORDER BY rb.user_id, rb.is_owner DESC, r.name ASC
)
SELECT u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url, u.status,
       u.last_login_at, u.created_at, u.updated_at,
       m.role_name, m.joined_at
FROM members m
JOIN users u ON u.id = m.user_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR u.email ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN m.joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN m.joined_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    m.joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountNamespaceMembers :one
WITH members AS (
    SELECT DISTINCT ON (rb.user_id) rb.user_id
    FROM role_bindings rb
    WHERE rb.scope = 'namespace' AND rb.namespace_id = @namespace_id
    ORDER BY rb.user_id
)
SELECT count(*)
FROM members m
JOIN users u ON u.id = m.user_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR u.email ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListNamespaceMembers :many
WITH members AS (
    SELECT DISTINCT ON (rb.user_id)
        rb.user_id,
        r.name AS role_name,
        rb.created_at AS joined_at
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.scope = 'namespace' AND rb.namespace_id = @namespace_id
    ORDER BY rb.user_id, rb.is_owner DESC, r.name ASC
)
SELECT u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url, u.status,
       u.last_login_at, u.created_at, u.updated_at,
       m.role_name, m.joined_at
FROM members m
JOIN users u ON u.id = m.user_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       u.username ILIKE '%' || sqlc.narg('search') || '%'
       OR u.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR u.email ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN m.joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN m.joined_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    m.joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountUserWorkspaces :one
WITH user_ws AS (
    SELECT DISTINCT ON (rb.workspace_id)
        rb.workspace_id,
        r.name AS role_name,
        r.display_name AS role_display_name
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.user_id = @user_id AND rb.scope = 'workspace'
    ORDER BY rb.workspace_id, rb.is_owner DESC, r.name ASC
)
SELECT count(*)
FROM user_ws uw
JOIN workspaces ws ON ws.id = uw.workspace_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       ws.name ILIKE '%' || sqlc.narg('search') || '%'
       OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR uw.role_name ILIKE '%' || sqlc.narg('search') || '%'
       OR uw.role_display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListUserWorkspaces :many
WITH user_ws AS (
    SELECT DISTINCT ON (rb.workspace_id)
        rb.workspace_id,
        r.name AS role_name,
        r.display_name AS role_display_name,
        rb.created_at AS joined_at
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.user_id = @user_id AND rb.scope = 'workspace'
    ORDER BY rb.workspace_id, rb.is_owner DESC, r.name ASC
)
SELECT ws.id, ws.name, ws.display_name, ws.description, ws.owner_id, ws.status,
       ws.created_at, ws.updated_at,
       u.username AS owner_username,
       (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) AS namespace_count,
       (SELECT count(DISTINCT rb2.user_id) FROM role_bindings rb2 WHERE rb2.scope = 'workspace' AND rb2.workspace_id = ws.id) AS member_count,
       uw.role_name, uw.role_display_name, uw.joined_at
FROM user_ws uw
JOIN workspaces ws ON ws.id = uw.workspace_id
JOIN users u ON ws.owner_id = u.id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       ws.name ILIKE '%' || sqlc.narg('search') || '%'
       OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR uw.role_name ILIKE '%' || sqlc.narg('search') || '%'
       OR uw.role_display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN uw.role_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN uw.role_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN uw.joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN uw.joined_at END DESC,
    uw.joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountUserNamespaces :one
WITH user_ns AS (
    SELECT DISTINCT ON (rb.namespace_id)
        rb.namespace_id,
        r.name AS role_name,
        r.display_name AS role_display_name
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.user_id = @user_id AND rb.scope = 'namespace'
    ORDER BY rb.namespace_id, rb.is_owner DESC, r.name ASC
)
SELECT count(*)
FROM user_ns un
JOIN namespaces ns ON ns.id = un.namespace_id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
  AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
  AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR ns.workspace_id = sqlc.narg('workspace_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       ns.name ILIKE '%' || sqlc.narg('search') || '%'
       OR ns.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR un.role_name ILIKE '%' || sqlc.narg('search') || '%'
       OR un.role_display_name ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListUserNamespaces :many
WITH user_ns AS (
    SELECT DISTINCT ON (rb.namespace_id)
        rb.namespace_id,
        r.name AS role_name,
        r.display_name AS role_display_name,
        rb.created_at AS joined_at
    FROM role_bindings rb
    JOIN roles r ON r.id = rb.role_id
    WHERE rb.user_id = @user_id AND rb.scope = 'namespace'
    ORDER BY rb.namespace_id, rb.is_owner DESC, r.name ASC
)
SELECT ns.id, ns.name, ns.display_name, ns.description, ns.workspace_id, ns.owner_id,
       ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
       u.username AS owner_username,
       w.name AS workspace_name,
       (SELECT count(DISTINCT rb2.user_id) FROM role_bindings rb2 WHERE rb2.scope = 'namespace' AND rb2.namespace_id = ns.id) AS member_count,
       un.role_name, un.role_display_name, un.joined_at
FROM user_ns un
JOIN namespaces ns ON ns.id = un.namespace_id
JOIN users u ON ns.owner_id = u.id
JOIN workspaces w ON ns.workspace_id = w.id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
  AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
  AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR ns.workspace_id = sqlc.narg('workspace_id'))
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       ns.name ILIKE '%' || sqlc.narg('search') || '%'
       OR ns.display_name ILIKE '%' || sqlc.narg('search') || '%'
       OR un.role_name ILIKE '%' || sqlc.narg('search') || '%'
       OR un.role_display_name ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN un.role_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'role_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN un.role_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN un.joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN un.joined_at END DESC,
    un.joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: DeleteNonOwnerWorkspaceBindings :execrows
DELETE FROM role_bindings
WHERE user_id = @user_id AND scope = 'workspace' AND workspace_id = @workspace_id AND is_owner = false;

-- name: DeleteNonOwnerNamespaceBindings :execrows
DELETE FROM role_bindings
WHERE user_id = @user_id AND scope = 'namespace' AND namespace_id = @namespace_id AND is_owner = false;

-- name: ListAllWorkspaceIDs :many
SELECT id FROM workspaces;

-- name: ListAllNamespaceIDsWithWorkspace :many
SELECT id, workspace_id FROM namespaces;
