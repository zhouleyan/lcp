-- name: AddUserToNamespace :one
INSERT INTO user_namespaces (user_id, namespace_id, role)
VALUES (@user_id, @namespace_id, @role)
ON CONFLICT (user_id, namespace_id) DO NOTHING
RETURNING user_id, namespace_id, role, created_at;

-- name: RemoveUserFromNamespace :exec
DELETE FROM user_namespaces
WHERE user_id = @user_id AND namespace_id = @namespace_id;

-- name: UpdateUserNamespaceRole :one
UPDATE user_namespaces
SET role = @role
WHERE user_id = @user_id AND namespace_id = @namespace_id
RETURNING user_id, namespace_id, role, created_at;

-- name: GetUserNamespace :one
SELECT user_id, namespace_id, role, created_at
FROM user_namespaces
WHERE user_id = @user_id AND namespace_id = @namespace_id;

-- name: ListNamespacesByUserID :many
SELECT
    n.id, n.name, n.display_name, n.description, n.workspace_id, n.owner_id,
    n.visibility, n.max_members, n.status, n.created_at, n.updated_at,
    un.role, un.created_at AS joined_at
FROM namespaces n
JOIN user_namespaces un ON n.id = un.namespace_id
WHERE un.user_id = @user_id
ORDER BY un.created_at DESC;

-- name: ListUsersByNamespaceID :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    un.role, un.created_at AS joined_at
FROM users u
JOIN user_namespaces un ON u.id = un.user_id
WHERE un.namespace_id = @namespace_id
ORDER BY un.created_at DESC;

-- name: CountNamespacesByUserID :one
SELECT count(namespace_id)
FROM user_namespaces
WHERE user_id = @user_id;

-- name: CountUsersByNamespaceIDFiltered :one
SELECT count(u.id)
FROM users u
JOIN user_namespaces un ON u.id = un.user_id
WHERE un.namespace_id = @namespace_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR u.username ILIKE '%' || sqlc.narg('search') || '%'
         OR u.email ILIKE '%' || sqlc.narg('search') || '%'
         OR u.phone ILIKE '%' || sqlc.narg('search') || '%'
         OR u.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListUsersByNamespaceIDPaginated :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    un.role, un.created_at AS joined_at
FROM users u
JOIN user_namespaces un ON u.id = un.user_id
WHERE un.namespace_id = @namespace_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR u.username ILIKE '%' || sqlc.narg('search') || '%'
         OR u.email ILIKE '%' || sqlc.narg('search') || '%'
         OR u.phone ILIKE '%' || sqlc.narg('search') || '%'
         OR u.display_name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.email END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.email END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.updated_at END DESC,
    un.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
