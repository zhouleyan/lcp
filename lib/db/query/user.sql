-- name: CreateUser :one
INSERT INTO users (username, email, display_name, phone, avatar_url, status)
VALUES (@username, @email, @display_name, @phone, @avatar_url, @status)
RETURNING id, username, email, display_name, phone, avatar_url, status,
          last_login_at, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE id = @id;

-- name: GetUserByUsername :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE username = @username;

-- name: GetUserByEmail :one
SELECT id, username, email, display_name, phone, avatar_url, status,
       last_login_at, created_at, updated_at
FROM users
WHERE email = @email;

-- name: UpdateUser :one
UPDATE users
SET username = @username,
    email = @email,
    display_name = @display_name,
    phone = @phone,
    avatar_url = @avatar_url,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, username, email, display_name, phone, avatar_url, status,
          last_login_at, created_at, updated_at;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = @id;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id;

-- name: CountUsers :one
SELECT count(id)
FROM users
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('username')::VARCHAR IS NULL OR username ILIKE '%' || sqlc.narg('username') || '%')
    AND (sqlc.narg('email')::VARCHAR IS NULL OR email ILIKE '%' || sqlc.narg('email') || '%')
    AND (sqlc.narg('display_name')::VARCHAR IS NULL OR display_name ILIKE '%' || sqlc.narg('display_name') || '%');

-- name: ListUsers :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    COALESCE(
        array_agg(DISTINCT n.name) FILTER (WHERE n.name IS NOT NULL),
        '{}'
    )::TEXT[] AS namespace_names
FROM users u
LEFT JOIN user_namespaces un ON u.id = un.user_id
LEFT JOIN namespaces n ON un.namespace_id = n.id
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
    AND (sqlc.narg('username')::VARCHAR IS NULL OR u.username ILIKE '%' || sqlc.narg('username') || '%')
    AND (sqlc.narg('email')::VARCHAR IS NULL OR u.email ILIKE '%' || sqlc.narg('email') || '%')
    AND (sqlc.narg('display_name')::VARCHAR IS NULL OR u.display_name ILIKE '%' || sqlc.narg('display_name') || '%')
GROUP BY u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
         u.status, u.last_login_at, u.created_at, u.updated_at
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.username END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'username' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.username END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.email END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'email' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.email END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.status END DESC,
    u.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
