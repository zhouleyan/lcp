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

-- name: CountNamespacesByUserIDJoined :one
SELECT count(ns.id)
FROM namespaces ns
JOIN user_namespaces un ON ns.id = un.namespace_id
WHERE un.user_id = @user_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR ns.workspace_id = sqlc.narg('workspace_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL OR (
        ns.name ILIKE '%' || sqlc.narg('search') || '%'
        OR ns.display_name ILIKE '%' || sqlc.narg('search') || '%'
        OR ns.description ILIKE '%' || sqlc.narg('search') || '%'
    ));

-- name: ListNamespacesByUserIDPaginated :many
WITH ns_data AS (
    SELECT
        ns.id, ns.name, ns.display_name, ns.description, ns.workspace_id, ns.owner_id,
        ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
        u.username AS owner_username,
        w.name AS workspace_name,
        (SELECT count(*) FROM user_namespaces un2 WHERE un2.namespace_id = ns.id) AS member_count,
        un.role, un.created_at AS joined_at
    FROM namespaces ns
    JOIN user_namespaces un ON ns.id = un.namespace_id
    JOIN users u ON ns.owner_id = u.id
    JOIN workspaces w ON ns.workspace_id = w.id
    WHERE un.user_id = @user_id
        AND (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
        AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
        AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR ns.workspace_id = sqlc.narg('workspace_id'))
        AND (sqlc.narg('search')::VARCHAR IS NULL OR (
            ns.name ILIKE '%' || sqlc.narg('search') || '%'
            OR ns.display_name ILIKE '%' || sqlc.narg('search') || '%'
            OR ns.description ILIKE '%' || sqlc.narg('search') || '%'
        ))
)
SELECT * FROM ns_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN updated_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN visibility END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN visibility END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN member_count END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN member_count END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN joined_at END DESC,
    joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

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
