-- name: CreateSite :one
INSERT INTO sites (name, display_name, description, region_id, status, address, latitude, longitude, contact_name, contact_phone, contact_email)
VALUES (@name, @display_name, @description, @region_id, @status, @address, @latitude, @longitude, @contact_name, @contact_phone, @contact_email)
RETURNING *;

-- name: GetSiteByID :one
SELECT
    s.*,
    r.name AS region_name,
    (SELECT count(*) FROM locations l WHERE l.site_id = s.id) AS location_count
FROM sites s
JOIN regions r ON s.region_id = r.id
WHERE s.id = @id;

-- name: UpdateSite :one
UPDATE sites
SET name = @name,
    display_name = @display_name,
    description = @description,
    region_id = @region_id,
    status = @status,
    address = @address,
    latitude = @latitude,
    longitude = @longitude,
    contact_name = @contact_name,
    contact_phone = @contact_phone,
    contact_email = @contact_email,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: PatchSite :one
UPDATE sites
SET name = COALESCE(sqlc.narg('name'), name),
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    description = COALESCE(sqlc.narg('description'), description),
    region_id = COALESCE(sqlc.narg('region_id'), region_id),
    status = COALESCE(sqlc.narg('status'), status),
    address = COALESCE(sqlc.narg('address'), address),
    latitude = COALESCE(sqlc.narg('latitude'), latitude),
    longitude = COALESCE(sqlc.narg('longitude'), longitude),
    contact_name = COALESCE(sqlc.narg('contact_name'), contact_name),
    contact_phone = COALESCE(sqlc.narg('contact_phone'), contact_phone),
    contact_email = COALESCE(sqlc.narg('contact_email'), contact_email),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteSite :exec
DELETE FROM sites WHERE id = @id;

-- name: DeleteSitesByIDs :many
DELETE FROM sites WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountSiteChildLocations :one
SELECT count(*) FROM locations WHERE site_id = @site_id;

-- name: CountSites :one
SELECT count(*)
FROM sites
WHERE (sqlc.narg('region_id')::BIGINT IS NULL OR region_id = sqlc.narg('region_id'))
    AND (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListSites :many
WITH site_data AS (
    SELECT
        s.*,
        r.name AS region_name,
        (SELECT count(*) FROM locations l WHERE l.site_id = s.id) AS location_count
    FROM sites s
    JOIN regions r ON s.region_id = r.id
    WHERE (sqlc.narg('region_id')::BIGINT IS NULL OR s.region_id = sqlc.narg('region_id'))
        AND (sqlc.narg('status')::VARCHAR IS NULL OR s.status = sqlc.narg('status'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR s.name ILIKE '%' || sqlc.narg('search') || '%'
             OR s.display_name ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM site_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

