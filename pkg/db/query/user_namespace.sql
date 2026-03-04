-- name: AddUserToNamespace :one
INSERT INTO user_namespaces (user_id, namespace_id, role)
VALUES (@user_id, @namespace_id, @role)
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
    n.id, n.name, n.display_name, n.description, n.owner_id,
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

-- name: CountUsersByNamespaceID :one
SELECT count(user_id)
FROM user_namespaces
WHERE namespace_id = @namespace_id;

-- name: CountNamespacesByUserID :one
SELECT count(namespace_id)
FROM user_namespaces
WHERE user_id = @user_id;
