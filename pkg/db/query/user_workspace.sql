-- name: AddUserToWorkspace :one
INSERT INTO user_workspaces (user_id, workspace_id, role)
VALUES (@user_id, @workspace_id, @role)
ON CONFLICT (user_id, workspace_id) DO NOTHING
RETURNING user_id, workspace_id, role, created_at;

-- name: RemoveUserFromWorkspace :exec
DELETE FROM user_workspaces
WHERE user_id = @user_id AND workspace_id = @workspace_id;

-- name: UpdateUserWorkspaceRole :one
UPDATE user_workspaces
SET role = @role
WHERE user_id = @user_id AND workspace_id = @workspace_id
RETURNING user_id, workspace_id, role, created_at;

-- name: GetUserWorkspace :one
SELECT user_id, workspace_id, role, created_at
FROM user_workspaces
WHERE user_id = @user_id AND workspace_id = @workspace_id;

-- name: CountWorkspacesByUserID :one
SELECT count(ws.id)
FROM workspaces ws
JOIN user_workspaces uw ON ws.id = uw.workspace_id
WHERE uw.user_id = @user_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ws.name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.description ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListWorkspacesByUserIDPaginated :many
WITH ws_data AS (
    SELECT
        ws.id, ws.name, ws.display_name, ws.description, ws.owner_id,
        ws.status, ws.created_at, ws.updated_at,
        u.username AS owner_username,
        (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) AS namespace_count,
        (SELECT count(*) FROM user_workspaces uw2 WHERE uw2.workspace_id = ws.id) AS member_count,
        uw.role, uw.created_at AS joined_at
    FROM workspaces ws
    JOIN user_workspaces uw ON ws.id = uw.workspace_id
    JOIN users u ON ws.owner_id = u.id
    WHERE uw.user_id = @user_id
        AND (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR ws.name ILIKE '%' || sqlc.narg('search') || '%'
             OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
             OR ws.description ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM ws_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN updated_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'namespace_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN namespace_count END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'namespace_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN namespace_count END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN member_count END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN member_count END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN joined_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'joined_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN joined_at END DESC,
    joined_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: ListUsersByWorkspaceID :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    uw.role, uw.created_at AS joined_at
FROM users u
JOIN user_workspaces uw ON u.id = uw.user_id
WHERE uw.workspace_id = @workspace_id
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
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'phone' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.phone END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'phone' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.phone END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN u.updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN u.updated_at END DESC,
    uw.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountUsersByWorkspaceID :one
SELECT count(u.id)
FROM users u
JOIN user_workspaces uw ON u.id = uw.user_id
WHERE uw.workspace_id = @workspace_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR u.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR u.username ILIKE '%' || sqlc.narg('search') || '%'
         OR u.email ILIKE '%' || sqlc.narg('search') || '%'
         OR u.phone ILIKE '%' || sqlc.narg('search') || '%'
         OR u.display_name ILIKE '%' || sqlc.narg('search') || '%');
