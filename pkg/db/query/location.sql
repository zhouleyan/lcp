-- name: CreateLocation :one
INSERT INTO locations (name, display_name, description, site_id, status, floor, rack_capacity, contact_name, contact_phone, contact_email)
VALUES (@name, @display_name, @description, @site_id, @status, @floor, @rack_capacity, @contact_name, @contact_phone, @contact_email)
RETURNING *;

-- name: GetLocationByID :one
SELECT
    l.*,
    s.name AS site_name,
    s.region_id AS region_id,
    r.name AS region_name
FROM locations l
JOIN sites s ON l.site_id = s.id
JOIN regions r ON s.region_id = r.id
WHERE l.id = @id;

-- name: UpdateLocation :one
UPDATE locations
SET name = @name,
    display_name = @display_name,
    description = @description,
    site_id = @site_id,
    status = @status,
    floor = @floor,
    rack_capacity = @rack_capacity,
    contact_name = @contact_name,
    contact_phone = @contact_phone,
    contact_email = @contact_email,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchLocation :one
UPDATE locations
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    site_id = COALESCE(sqlc.narg('site_id'), site_id),
    status = COALESCE(sqlc.narg('status'), status),
    floor = COALESCE(sqlc.narg('floor'), floor),
    rack_capacity = COALESCE(sqlc.narg('rack_capacity'), rack_capacity),
    contact_name = COALESCE(sqlc.narg('contact_name'), contact_name),
    contact_phone = COALESCE(sqlc.narg('contact_phone'), contact_phone),
    contact_email = COALESCE(sqlc.narg('contact_email'), contact_email),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id = @id;

-- name: DeleteLocationsByIDs :many
DELETE FROM locations WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountLocations :one
SELECT count(*)
FROM locations l
JOIN sites s ON l.site_id = s.id
WHERE (sqlc.narg('site_id')::BIGINT IS NULL OR l.site_id = sqlc.narg('site_id'))
    AND (sqlc.narg('region_id')::BIGINT IS NULL OR s.region_id = sqlc.narg('region_id'))
    AND (sqlc.narg('status')::VARCHAR IS NULL OR l.status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR l.name ILIKE '%' || sqlc.narg('search') || '%'
         OR l.display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListLocations :many
WITH location_data AS (
    SELECT
        l.*,
        s.name AS site_name,
        s.region_id AS region_id,
        r.name AS region_name
    FROM locations l
    JOIN sites s ON l.site_id = s.id
    JOIN regions r ON s.region_id = r.id
    WHERE (sqlc.narg('site_id')::BIGINT IS NULL OR l.site_id = sqlc.narg('site_id'))
        AND (sqlc.narg('region_id')::BIGINT IS NULL OR s.region_id = sqlc.narg('region_id'))
        AND (sqlc.narg('status')::VARCHAR IS NULL OR l.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR l.name ILIKE '%' || sqlc.narg('search') || '%'
             OR l.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM location_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: ListLocationsBySiteID :many
WITH location_data AS (
    SELECT
        l.*,
        s.name AS site_name,
        s.region_id AS region_id,
        r.name AS region_name
    FROM locations l
    JOIN sites s ON l.site_id = s.id
    JOIN regions r ON s.region_id = r.id
    WHERE l.site_id = @site_id
        AND (sqlc.narg('status')::VARCHAR IS NULL OR l.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR l.name ILIKE '%' || sqlc.narg('search') || '%'
             OR l.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM location_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: CountLocationsBySiteID :one
SELECT count(*)
FROM locations
WHERE site_id = @site_id
    AND (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');
