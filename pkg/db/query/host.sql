-- name: CreateHost :one
INSERT INTO hosts (name, display_name, description, hostname, ip_address, os, arch, cpu_cores, memory_mb, disk_gb, labels, scope, workspace_id, namespace_id, status)
VALUES (@name, @display_name, @description, @hostname, @ip_address, @os, @arch, @cpu_cores, @memory_mb, @disk_gb, @labels, @scope, @workspace_id, @namespace_id, @status)
RETURNING *;

-- name: GetHostByID :one
SELECT
    h.*,
    e.name AS environment_name,
    w.name AS workspace_name,
    n.name AS namespace_name
FROM hosts h
LEFT JOIN environments e ON h.environment_id = e.id
LEFT JOIN workspaces w ON h.workspace_id = w.id
LEFT JOIN namespaces n ON h.namespace_id = n.id
WHERE h.id = @id;

-- name: UpdateHost :one
UPDATE hosts
SET name = @name,
    display_name = @display_name,
    description = @description,
    hostname = @hostname,
    ip_address = @ip_address,
    os = @os,
    arch = @arch,
    cpu_cores = @cpu_cores,
    memory_mb = @memory_mb,
    disk_gb = @disk_gb,
    labels = @labels,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchHost :one
UPDATE hosts
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    hostname = COALESCE(sqlc.narg('hostname'), hostname),
    ip_address = COALESCE(sqlc.narg('ip_address'), ip_address),
    os = COALESCE(sqlc.narg('os'), os),
    arch = COALESCE(sqlc.narg('arch'), arch),
    cpu_cores = COALESCE(sqlc.narg('cpu_cores'), cpu_cores),
    memory_mb = COALESCE(sqlc.narg('memory_mb'), memory_mb),
    disk_gb = COALESCE(sqlc.narg('disk_gb'), disk_gb),
    labels = COALESCE(sqlc.narg('labels'), labels),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteHost :exec
DELETE FROM hosts WHERE id = @id;

-- name: DeleteHostsByIDs :many
DELETE FROM hosts WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: GetWorkspaceIDByNamespaceID :one
SELECT workspace_id FROM namespaces WHERE id = @id;

-- name: BindHostEnvironment :exec
UPDATE hosts SET environment_id = @environment_id, updated_at = now()
WHERE id = @id AND environment_id IS NULL;

-- name: UnbindHostEnvironment :exec
UPDATE hosts SET environment_id = NULL, updated_at = now()
WHERE id = @id;

-- name: CountHostsPlatform :one
SELECT count(*)
FROM hosts
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('environment_id')::BIGINT IS NULL
         OR (sqlc.narg('environment_id')::BIGINT = 0 AND environment_id IS NULL)
         OR environment_id = sqlc.narg('environment_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListHostsPlatform :many
WITH host_data AS (
    SELECT
        h.*,
        e.name AS environment_name,
        w.name AS workspace_name,
        n.name AS namespace_name
    FROM hosts h
    LEFT JOIN environments e ON h.environment_id = e.id
    LEFT JOIN workspaces w ON h.workspace_id = w.id
    LEFT JOIN namespaces n ON h.namespace_id = n.id
    WHERE (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
        AND (sqlc.narg('environment_id')::BIGINT IS NULL
             OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
             OR h.environment_id = sqlc.narg('environment_id'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR h.name ILIKE '%' || sqlc.narg('search') || '%'
             OR h.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM host_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ip_address END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ip_address END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountHostsByWorkspaceID :one
SELECT count(*)
FROM hosts h
WHERE (
    (h.scope = 'workspace' AND h.workspace_id = @workspace_id)
    OR (h.scope = 'namespace' AND h.namespace_id IN (SELECT n.id FROM namespaces n WHERE n.workspace_id = @workspace_id))
)
    AND (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
    AND (sqlc.narg('environment_id')::BIGINT IS NULL
         OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
         OR h.environment_id = sqlc.narg('environment_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR h.name ILIKE '%' || sqlc.narg('search') || '%'
         OR h.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListHostsByWorkspaceID :many
WITH host_data AS (
    SELECT
        h.*,
        e.name AS environment_name,
        n.name AS namespace_name
    FROM hosts h
    LEFT JOIN environments e ON h.environment_id = e.id
    LEFT JOIN namespaces n ON h.namespace_id = n.id
    WHERE (
        (h.scope = 'workspace' AND h.workspace_id = @workspace_id)
        OR (h.scope = 'namespace' AND h.namespace_id IN (SELECT n2.id FROM namespaces n2 WHERE n2.workspace_id = @workspace_id))
    )
        AND (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
        AND (sqlc.narg('environment_id')::BIGINT IS NULL
             OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
             OR h.environment_id = sqlc.narg('environment_id'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR h.name ILIKE '%' || sqlc.narg('search') || '%'
             OR h.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM host_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ip_address END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ip_address END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountHostsByNamespaceID :one
SELECT count(*)
FROM hosts h
WHERE h.scope = 'namespace' AND h.namespace_id = @namespace_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
    AND (sqlc.narg('environment_id')::BIGINT IS NULL
         OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
         OR h.environment_id = sqlc.narg('environment_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR h.name ILIKE '%' || sqlc.narg('search') || '%'
         OR h.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListHostsByNamespaceID :many
WITH host_data AS (
    SELECT
        h.*,
        e.name AS environment_name
    FROM hosts h
    LEFT JOIN environments e ON h.environment_id = e.id
    WHERE h.scope = 'namespace' AND h.namespace_id = @namespace_id
        AND (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
        AND (sqlc.narg('environment_id')::BIGINT IS NULL
             OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
             OR h.environment_id = sqlc.narg('environment_id'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR h.name ILIKE '%' || sqlc.narg('search') || '%'
             OR h.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM host_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ip_address END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip_address' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ip_address END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
