-- name: CreateHost :one
INSERT INTO hosts (name, display_name, description, hostname, ip_address, os, arch, cpu_cores, memory_mb, disk_gb, labels, scope, workspace_id, namespace_id, status)
VALUES (@name, @display_name, @description, @hostname, @ip_address, @os, @arch, @cpu_cores, @memory_mb, @disk_gb, @labels, @scope, @workspace_id, @namespace_id, @status)
RETURNING *;

-- name: GetHostByID :one
SELECT
    h.*,
    e.name AS environment_name
FROM hosts h
LEFT JOIN environments e ON h.environment_id = e.id
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

-- name: BindHostEnvironment :exec
UPDATE hosts SET environment_id = @environment_id, updated_at = now()
WHERE id = @id AND environment_id IS NULL;

-- name: UnbindHostEnvironment :exec
UPDATE hosts SET environment_id = NULL, updated_at = now()
WHERE id = @id;

-- name: CountHostsPlatform :one
SELECT count(*)
FROM hosts
WHERE scope = 'platform'
    AND (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
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
        e.name AS environment_name
    FROM hosts h
    LEFT JOIN environments e ON h.environment_id = e.id
    WHERE h.scope = 'platform'
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

-- name: CountHostsByWorkspaceID :one
SELECT count(*)
FROM (
    SELECT h.id FROM hosts h
    WHERE h.scope = 'workspace' AND h.workspace_id = @workspace_id
    UNION
    SELECT h.id FROM hosts h
    JOIN host_assignments ha ON ha.host_id = h.id
    WHERE ha.workspace_id = @workspace_id
) AS visible
JOIN hosts h ON h.id = visible.id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
    AND (sqlc.narg('environment_id')::BIGINT IS NULL
         OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
         OR h.environment_id = sqlc.narg('environment_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR h.name ILIKE '%' || sqlc.narg('search') || '%'
         OR h.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListHostsByWorkspaceID :many
WITH visible_hosts AS (
    SELECT h.id FROM hosts h
    WHERE h.scope = 'workspace' AND h.workspace_id = @workspace_id
    UNION
    SELECT h.id FROM hosts h
    JOIN host_assignments ha ON ha.host_id = h.id
    WHERE ha.workspace_id = @workspace_id
),
host_data AS (
    SELECT
        h.*,
        e.name AS environment_name,
        CASE WHEN h.scope = 'workspace' AND h.workspace_id = @workspace_id THEN 'owned' ELSE 'assigned' END AS origin
    FROM hosts h
    JOIN visible_hosts vh ON vh.id = h.id
    LEFT JOIN environments e ON h.environment_id = e.id
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

-- name: CountHostsByNamespaceID :one
SELECT count(*)
FROM (
    SELECT h.id FROM hosts h
    WHERE h.scope = 'namespace' AND h.namespace_id = @namespace_id
    UNION
    SELECT h.id FROM hosts h
    JOIN host_assignments ha ON ha.host_id = h.id
    WHERE ha.namespace_id = @namespace_id
) AS visible
JOIN hosts h ON h.id = visible.id
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR h.status = sqlc.narg('status'))
    AND (sqlc.narg('environment_id')::BIGINT IS NULL
         OR (sqlc.narg('environment_id')::BIGINT = 0 AND h.environment_id IS NULL)
         OR h.environment_id = sqlc.narg('environment_id'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR h.name ILIKE '%' || sqlc.narg('search') || '%'
         OR h.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListHostsByNamespaceID :many
WITH visible_hosts AS (
    SELECT h.id FROM hosts h
    WHERE h.scope = 'namespace' AND h.namespace_id = @namespace_id
    UNION
    SELECT h.id FROM hosts h
    JOIN host_assignments ha ON ha.host_id = h.id
    WHERE ha.namespace_id = @namespace_id
),
host_data AS (
    SELECT
        h.*,
        e.name AS environment_name,
        CASE WHEN h.scope = 'namespace' AND h.namespace_id = @namespace_id THEN 'owned' ELSE 'assigned' END AS origin
    FROM hosts h
    JOIN visible_hosts vh ON vh.id = h.id
    LEFT JOIN environments e ON h.environment_id = e.id
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
