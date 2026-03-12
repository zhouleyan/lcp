-- name: CreateRegion :one
INSERT INTO regions (name, display_name, description, status, latitude, longitude)
VALUES (@name, @display_name, @description, @status, @latitude, @longitude)
RETURNING *;

-- name: GetRegionByID :one
SELECT
    r.*,
    (SELECT count(*) FROM sites s WHERE s.region_id = r.id) AS site_count
FROM regions r
WHERE r.id = @id;

-- name: UpdateRegion :one
UPDATE regions
SET name = @name,
    display_name = @display_name,
    description = @description,
    status = @status,
    latitude = @latitude,
    longitude = @longitude,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchRegion :one
UPDATE regions
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    latitude = COALESCE(sqlc.narg('latitude'), latitude),
    longitude = COALESCE(sqlc.narg('longitude'), longitude),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteRegion :exec
DELETE FROM regions WHERE id = @id;

-- name: DeleteRegionsByIDs :many
DELETE FROM regions WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountRegionChildSites :one
SELECT count(*) FROM sites WHERE region_id = @region_id;

-- name: CountRegions :one
SELECT count(*)
FROM regions
WHERE (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListRegions :many
WITH region_data AS (
    SELECT
        r.*,
        (SELECT count(*) FROM sites s WHERE s.region_id = r.id) AS site_count
    FROM regions r
    WHERE (sqlc.narg('status')::VARCHAR IS NULL OR r.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR r.name ILIKE '%' || sqlc.narg('search') || '%'
             OR r.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM region_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
