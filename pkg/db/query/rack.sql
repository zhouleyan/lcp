-- name: CreateRack :one
INSERT INTO racks (name, display_name, description, location_id, status, u_height, position, power_capacity)
VALUES (@name, @display_name, @description, @location_id, @status, @u_height, @position, @power_capacity)
RETURNING *;

-- name: GetRackByID :one
SELECT
    rk.*,
    l.name AS location_name,
    l.site_id AS site_id,
    s.name AS site_name,
    s.region_id AS region_id,
    r.name AS region_name
FROM racks rk
JOIN locations l ON rk.location_id = l.id
JOIN sites s ON l.site_id = s.id
JOIN regions r ON s.region_id = r.id
WHERE rk.id = @id;

-- name: UpdateRack :one
UPDATE racks
SET name = @name,
    display_name = @display_name,
    description = @description,
    location_id = @location_id,
    status = @status,
    u_height = @u_height,
    position = @position,
    power_capacity = @power_capacity,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchRack :one
UPDATE racks
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    location_id = COALESCE(sqlc.narg('location_id'), location_id),
    status = COALESCE(sqlc.narg('status'), status),
    u_height = COALESCE(sqlc.narg('u_height'), u_height),
    position = COALESCE(sqlc.narg('position'), position),
    power_capacity = COALESCE(sqlc.narg('power_capacity'), power_capacity),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteRack :exec
DELETE FROM racks WHERE id = @id;

-- name: DeleteRacksByIDs :many
DELETE FROM racks WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountRacks :one
SELECT count(*)
FROM racks rk
JOIN locations l ON rk.location_id = l.id
JOIN sites s ON l.site_id = s.id
WHERE (sqlc.narg('location_id')::BIGINT IS NULL OR rk.location_id = sqlc.narg('location_id'))
    AND (sqlc.narg('site_id')::BIGINT IS NULL OR l.site_id = sqlc.narg('site_id'))
    AND (sqlc.narg('region_id')::BIGINT IS NULL OR s.region_id = sqlc.narg('region_id'))
    AND (sqlc.narg('status')::VARCHAR IS NULL OR rk.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR rk.name ILIKE '%' || sqlc.narg('search') || '%'
         OR rk.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListRacks :many
WITH rack_data AS (
    SELECT
        rk.*,
        l.name AS location_name,
        l.site_id AS site_id,
        s.name AS site_name,
        s.region_id AS region_id,
        r.name AS region_name
    FROM racks rk
    JOIN locations l ON rk.location_id = l.id
    JOIN sites s ON l.site_id = s.id
    JOIN regions r ON s.region_id = r.id
    WHERE (sqlc.narg('location_id')::BIGINT IS NULL OR rk.location_id = sqlc.narg('location_id'))
        AND (sqlc.narg('site_id')::BIGINT IS NULL OR l.site_id = sqlc.narg('site_id'))
        AND (sqlc.narg('region_id')::BIGINT IS NULL OR s.region_id = sqlc.narg('region_id'))
        AND (sqlc.narg('status')::VARCHAR IS NULL OR rk.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR rk.name ILIKE '%' || sqlc.narg('search') || '%'
             OR rk.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM rack_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountLocationChildRacks :one
SELECT count(*) FROM racks WHERE location_id = @location_id;
