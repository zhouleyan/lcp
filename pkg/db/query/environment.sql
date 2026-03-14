-- name: CreateEnvironment :one
INSERT INTO environments (name, display_name, description, env_type, scope, workspace_id, namespace_id, status)
VALUES (@name, @display_name, @description, @env_type, @scope, @workspace_id, @namespace_id, @status)
RETURNING *;

-- name: GetEnvironmentByID :one
SELECT
    e.*,
    (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
FROM environments e
WHERE e.id = @id;

-- name: UpdateEnvironment :one
UPDATE environments
SET name = @name,
    display_name = @display_name,
    description = @description,
    env_type = @env_type,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchEnvironment :one
UPDATE environments
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    env_type = COALESCE(sqlc.narg('env_type'), env_type),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteEnvironment :exec
DELETE FROM environments WHERE id = @id;

-- name: DeleteEnvironmentsByIDs :many
DELETE FROM environments WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountEnvironmentsPlatform :one
SELECT count(*)
FROM environments
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('env_type')::VARCHAR IS NULL OR env_type = sqlc.narg('env_type'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEnvironmentsPlatform :many
WITH env_data AS (
    SELECT
        e.*,
        (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
    FROM environments e
    WHERE (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
        AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR e.name ILIKE '%' || sqlc.narg('search') || '%'
             OR e.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM env_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN env_type END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN env_type END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountEnvironmentsByWorkspaceID :one
SELECT count(*)
FROM environments e
WHERE (
        (e.scope = 'workspace' AND e.workspace_id = @workspace_id)
        OR (e.scope = 'namespace' AND e.namespace_id IN (SELECT n.id FROM namespaces n WHERE n.workspace_id = @workspace_id))
    )
    AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
    AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR e.name ILIKE '%' || sqlc.narg('search') || '%'
         OR e.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEnvironmentsByWorkspaceID :many
WITH env_data AS (
    SELECT
        e.*,
        (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
    FROM environments e
    WHERE (
            (e.scope = 'workspace' AND e.workspace_id = @workspace_id)
            OR (e.scope = 'namespace' AND e.namespace_id IN (SELECT n.id FROM namespaces n WHERE n.workspace_id = @workspace_id))
        )
        AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
        AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR e.name ILIKE '%' || sqlc.narg('search') || '%'
             OR e.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM env_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN env_type END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN env_type END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountEnvironmentsByWorkspaceIDInherit :one
SELECT count(*)
FROM environments e
WHERE (
        (e.scope = 'workspace' AND e.workspace_id = @workspace_id)
        OR e.scope = 'platform'
    )
    AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
    AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR e.name ILIKE '%' || sqlc.narg('search') || '%'
         OR e.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEnvironmentsByWorkspaceIDInherit :many
WITH env_data AS (
    SELECT
        e.*,
        (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
    FROM environments e
    WHERE (
            (e.scope = 'workspace' AND e.workspace_id = @workspace_id)
            OR e.scope = 'platform'
        )
        AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
        AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR e.name ILIKE '%' || sqlc.narg('search') || '%'
             OR e.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM env_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN env_type END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN env_type END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountEnvironmentsByNamespaceID :one
SELECT count(*)
FROM environments
WHERE scope = 'namespace' AND namespace_id = @namespace_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('env_type')::VARCHAR IS NULL OR env_type = sqlc.narg('env_type'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEnvironmentsByNamespaceID :many
WITH env_data AS (
    SELECT
        e.*,
        (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
    FROM environments e
    WHERE e.scope = 'namespace' AND e.namespace_id = @namespace_id
        AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
        AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR e.name ILIKE '%' || sqlc.narg('search') || '%'
             OR e.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM env_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN env_type END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN env_type END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: ListHostsByEnvironmentID :many
SELECT
    h.*,
    e.name AS environment_name,
    COALESCE((SELECT json_agg(json_build_object('id', ia.id, 'ip', ia.ip, 'subnetId', ia.subnet_id) ORDER BY ia.created_at) FROM ip_allocations ia WHERE ia.host_id = h.id), '[]'::json) AS allocated_ips
FROM hosts h
LEFT JOIN environments e ON h.environment_id = e.id
WHERE h.environment_id = @environment_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR h.name ILIKE '%' || sqlc.narg('search') || '%'
         OR h.display_name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN h.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN h.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN h.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN h.created_at END DESC,
    h.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountHostsByEnvironmentID :one
SELECT count(*)
FROM hosts
WHERE environment_id = @environment_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: CountEnvironmentsByNamespaceIDInherit :one
SELECT count(*)
FROM environments e
WHERE (
        (e.scope = 'namespace' AND e.namespace_id = @namespace_id)
        OR (e.scope = 'workspace' AND e.workspace_id = (SELECT n.workspace_id FROM namespaces n WHERE n.id = @namespace_id))
        OR e.scope = 'platform'
    )
    AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
    AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR e.name ILIKE '%' || sqlc.narg('search') || '%'
         OR e.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEnvironmentsByNamespaceIDInherit :many
WITH env_data AS (
    SELECT
        e.*,
        (SELECT count(*) FROM hosts h WHERE h.environment_id = e.id) AS host_count
    FROM environments e
    WHERE (
            (e.scope = 'namespace' AND e.namespace_id = @namespace_id)
            OR (e.scope = 'workspace' AND e.workspace_id = (SELECT n.workspace_id FROM namespaces n WHERE n.id = @namespace_id))
            OR e.scope = 'platform'
        )
        AND (sqlc.narg('status')::VARCHAR IS NULL OR e.status = sqlc.narg('status'))
        AND (sqlc.narg('env_type')::VARCHAR IS NULL OR e.env_type = sqlc.narg('env_type'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR e.name ILIKE '%' || sqlc.narg('search') || '%'
             OR e.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM env_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN env_type END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'env_type' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN env_type END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
