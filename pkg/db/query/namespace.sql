-- name: CreateNamespace :one
INSERT INTO namespaces (name, display_name, description, owner_id, visibility, max_members, status)
VALUES (@name, @display_name, @description, @owner_id, @visibility, @max_members, @status)
RETURNING id, name, display_name, description, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: GetNamespaceByID :one
SELECT id, name, display_name, description, owner_id, visibility, max_members, status,
       created_at, updated_at
FROM namespaces
WHERE id = @id;

-- name: GetNamespaceByName :one
SELECT id, name, display_name, description, owner_id, visibility, max_members, status,
       created_at, updated_at
FROM namespaces
WHERE name = @name;

-- name: UpdateNamespace :one
UPDATE namespaces
SET name = @name,
    display_name = @display_name,
    description = @description,
    owner_id = @owner_id,
    visibility = @visibility,
    max_members = @max_members,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, owner_id, visibility, max_members, status,
          created_at, updated_at;

-- name: DeleteNamespace :exec
DELETE FROM namespaces WHERE id = @id;

-- name: CountNamespaces :one
SELECT count(id)
FROM namespaces
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR owner_id = sqlc.narg('owner_id'));

-- name: ListNamespaces :many
SELECT
    ns.id, ns.name, ns.display_name, ns.description, ns.owner_id,
    ns.visibility, ns.max_members, ns.status, ns.created_at, ns.updated_at,
    u.username AS owner_username
FROM namespaces ns
JOIN users u ON ns.owner_id = u.id
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR ns.status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR ns.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('visibility')::VARCHAR IS NULL OR ns.visibility = sqlc.narg('visibility'))
    AND (sqlc.narg('owner_id')::BIGINT IS NULL OR ns.owner_id = sqlc.narg('owner_id'))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.visibility END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'visibility' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.visibility END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ns.status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ns.status END DESC,
    ns.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
