-- name: CreateWorkspace :one
INSERT INTO workspaces (name, display_name, description, owner_id, status)
VALUES (@name, @display_name, @description, @owner_id, @status)
RETURNING id, name, display_name, description, owner_id, status,
          created_at, updated_at;

-- name: GetWorkspaceByID :one
SELECT
    ws.id, ws.name, ws.display_name, ws.description, ws.owner_id, ws.status,
    ws.created_at, ws.updated_at,
    u.username AS owner_username,
    (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) AS namespace_count,
    (SELECT count(*) FROM user_workspaces uw WHERE uw.workspace_id = ws.id) AS member_count
FROM workspaces ws
JOIN users u ON ws.owner_id = u.id
WHERE ws.id = @id;

-- name: GetWorkspaceByName :one
SELECT id, name, display_name, description, owner_id, status,
       created_at, updated_at
FROM workspaces
WHERE name = @name;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = @name,
    display_name = @display_name,
    description = @description,
    owner_id = @owner_id,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, owner_id, status,
          created_at, updated_at;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE id = @id;

-- name: DeleteWorkspacesByIDs :many
DELETE FROM workspaces WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountWorkspaces :one
SELECT count(ws.id)
FROM workspaces ws
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR ws.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ws.owner_id = sqlc.narg('owner_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ws.name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.description ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListWorkspaces :many
SELECT
    ws.id, ws.name, ws.display_name, ws.description, ws.owner_id,
    ws.status, ws.created_at, ws.updated_at,
    u.username AS owner_username,
    (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) AS namespace_count,
    (SELECT count(*) FROM user_workspaces uw WHERE uw.workspace_id = ws.id) AS member_count
FROM workspaces ws
JOIN users u ON ws.owner_id = u.id
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR ws.status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR ws.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ws.owner_id = sqlc.narg('owner_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ws.name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR ws.description ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.status END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ws.updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ws.updated_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'namespace_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'namespace_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN (SELECT count(*) FROM namespaces n WHERE n.workspace_id = ws.id) END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN (SELECT count(*) FROM user_workspaces uw WHERE uw.workspace_id = ws.id) END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'member_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN (SELECT count(*) FROM user_workspaces uw WHERE uw.workspace_id = ws.id) END DESC,
    ws.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountNamespacesByWorkspaceID :one
SELECT count(id)
FROM namespaces
WHERE workspace_id = @workspace_id;
