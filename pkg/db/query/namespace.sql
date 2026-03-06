-- name: CreateNamespace :one
INSERT INTO namespaces (name, display_name, description, workspace_id, owner_id, visibility, max_members, status)
VALUES (@name, @display_name, @description, @workspace_id, @owner_id, @visibility, @max_members, @status)
RETURNING id, name, display_name, description, workspace_id, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: GetNamespaceByID :one
SELECT
    ns.id, ns.name, ns.display_name, ns.description, ns.workspace_id, ns.owner_id,
    ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
    u.username AS owner_username,
    w.name AS workspace_name,
    (SELECT count(*) FROM user_namespaces un WHERE un.namespace_id = ns.id) AS member_count
FROM namespaces ns
JOIN users u ON ns.owner_id = u.id
JOIN workspaces w ON ns.workspace_id = w.id
WHERE ns.id = @id;

-- name: GetNamespaceByName :one
SELECT id, name, display_name, description, workspace_id, owner_id, visibility, max_members, status,
       created_at, updated_at
FROM namespaces
WHERE name = @name;

-- name: UpdateNamespace :one
UPDATE namespaces
SET name = @name,
    display_name = @display_name,
    description = @description,
    workspace_id = @workspace_id,
    owner_id = @owner_id,
    visibility = @visibility,
    max_members = @max_members,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, workspace_id, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: DeleteNamespace :exec
DELETE FROM namespaces WHERE id = @id;

-- name: DeleteNamespacesByIDs :many
DELETE FROM namespaces WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountNamespaces :one
SELECT count(ns.id)
FROM namespaces ns
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR ns.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ns.owner_id = sqlc.narg('owner_id'))
    AND (sqlc.narg('workspace_id')::BIGINT IS NULL OR ns.workspace_id = sqlc.narg('workspace_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL OR (
        ns.name ILIKE '%' || sqlc.narg('search') || '%'
        OR ns.display_name ILIKE '%' || sqlc.narg('search') || '%'
        OR ns.description ILIKE '%' || sqlc.narg('search') || '%'
    ));

-- name: ListNamespaces :many
WITH ns_data AS (
    SELECT
        ns.id, ns.name, ns.display_name, ns.description, ns.workspace_id, ns.owner_id,
        ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
        u.username AS owner_username,
        w.name AS workspace_name,
        (SELECT count(*) FROM user_namespaces un WHERE un.namespace_id = ns.id) AS member_count
    FROM namespaces ns
    JOIN users u ON ns.owner_id = u.id
    JOIN workspaces w ON ns.workspace_id = w.id
    WHERE
        (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
        AND (sqlc.narg('name')::VARCHAR IS NULL OR ns.name ILIKE '%' || sqlc.narg('name') || '%')
        AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
        AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ns.owner_id = sqlc.narg('owner_id'))
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
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountUsersByNamespaceID :one
SELECT count(user_id)
FROM user_namespaces
WHERE namespace_id = @namespace_id;
