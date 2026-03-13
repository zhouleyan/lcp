-- name: CreateEndpoint :one
INSERT INTO o11y_endpoints (name, description, public, metrics_url, logs_url, traces_url, apm_url, status)
VALUES (@name, @description, @public, @metrics_url, @logs_url, @traces_url, @apm_url, @status)
RETURNING *;

-- name: GetEndpointByID :one
SELECT * FROM o11y_endpoints WHERE id = @id;

-- name: UpdateEndpoint :one
UPDATE o11y_endpoints
SET name = @name,
    description = @description,
    public = @public,
    metrics_url = @metrics_url,
    logs_url = @logs_url,
    traces_url = @traces_url,
    apm_url = @apm_url,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchEndpoint :one
UPDATE o11y_endpoints
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    public = COALESCE(sqlc.narg('public'), public),
    metrics_url = COALESCE(sqlc.narg('metrics_url'), metrics_url),
    logs_url = COALESCE(sqlc.narg('logs_url'), logs_url),
    traces_url = COALESCE(sqlc.narg('traces_url'), traces_url),
    apm_url = COALESCE(sqlc.narg('apm_url'), apm_url),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteEndpoint :exec
DELETE FROM o11y_endpoints WHERE id = @id;

-- name: DeleteEndpointsByIDs :execrows
DELETE FROM o11y_endpoints WHERE id = ANY(@ids::BIGINT[]);

-- name: CountEndpoints :one
SELECT count(*) FROM o11y_endpoints
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL
       OR name ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
       OR metrics_url ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListEndpoints :many
SELECT * FROM o11y_endpoints
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('search')::VARCHAR IS NULL
       OR name ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
       OR metrics_url ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN status END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'status' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN status END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'metrics_url' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN metrics_url END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'metrics_url' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN metrics_url END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN updated_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
