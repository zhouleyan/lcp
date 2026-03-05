-- name: AddUserToWorkspace :one
INSERT INTO user_workspaces (user_id, workspace_id, role)
VALUES (@user_id, @workspace_id, @role)
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

-- name: ListWorkspacesByUserID :many
SELECT
    w.id, w.name, w.display_name, w.description, w.owner_id,
    w.status, w.created_at, w.updated_at,
    uw.role, uw.created_at AS joined_at
FROM workspaces w
JOIN user_workspaces uw ON w.id = uw.workspace_id
WHERE uw.user_id = @user_id
ORDER BY uw.created_at DESC;

-- name: ListUsersByWorkspaceID :many
SELECT
    u.id, u.username, u.email, u.display_name, u.phone, u.avatar_url,
    u.status, u.last_login_at, u.created_at, u.updated_at,
    uw.role, uw.created_at AS joined_at
FROM users u
JOIN user_workspaces uw ON u.id = uw.user_id
WHERE uw.workspace_id = @workspace_id
ORDER BY uw.created_at DESC;

-- name: CountUsersByWorkspaceID :one
SELECT count(user_id)
FROM user_workspaces
WHERE workspace_id = @workspace_id;
