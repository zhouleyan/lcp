-- name: CreateNetwork :one
INSERT INTO networks (name, display_name, description, cidr, max_subnets, is_public, status)
VALUES (@name, @display_name, @description, @cidr, @max_subnets, @is_public, @status)
RETURNING id, name, display_name, description, cidr, max_subnets, is_public, status, created_at, updated_at;

-- name: GetNetworkByID :one
SELECT
    n.id, n.name, n.display_name, n.description, n.cidr, n.max_subnets, n.is_public, n.status,
    n.created_at, n.updated_at,
    (SELECT count(*) FROM subnets s WHERE s.network_id = n.id) AS subnet_count
FROM networks n
WHERE n.id = @id;

-- name: UpdateNetwork :one
UPDATE networks
SET name = @name,
    display_name = @display_name,
    description = @description,
    status = @status,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, cidr, max_subnets, is_public, status, created_at, updated_at;

-- name: PatchNetwork :one
UPDATE networks
SET name = CASE WHEN sqlc.narg('name')::VARCHAR IS NOT NULL THEN sqlc.narg('name') ELSE name END,
    display_name = CASE WHEN sqlc.narg('display_name')::VARCHAR IS NOT NULL THEN sqlc.narg('display_name') ELSE display_name END,
    description = CASE WHEN sqlc.narg('description')::TEXT IS NOT NULL THEN sqlc.narg('description') ELSE description END,
    status = CASE WHEN sqlc.narg('status')::VARCHAR IS NOT NULL THEN sqlc.narg('status') ELSE status END,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, cidr, max_subnets, is_public, status, created_at, updated_at;

-- name: DeleteNetwork :exec
DELETE FROM networks WHERE id = @id;

-- name: DeleteNetworksByIDs :many
DELETE FROM networks WHERE id = ANY(@ids::BIGINT[])
RETURNING id;

-- name: CountSubnetsByNetworkID :one
SELECT count(*) FROM subnets WHERE network_id = @network_id;

-- name: CountNetworks :one
SELECT count(*)
FROM networks
WHERE
    (sqlc.narg('status')::VARCHAR IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('name')::VARCHAR IS NULL OR name = sqlc.narg('name'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR description ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListNetworks :many
WITH net_data AS (
    SELECT
        n.id, n.name, n.display_name, n.description, n.cidr, n.max_subnets, n.is_public, n.status,
        n.created_at, n.updated_at,
        (SELECT count(*) FROM subnets s WHERE s.network_id = n.id) AS subnet_count
    FROM networks n
    WHERE
        (sqlc.narg('status')::VARCHAR IS NULL OR n.status = sqlc.narg('status'))
        AND (sqlc.narg('name')::VARCHAR IS NULL OR n.name = sqlc.narg('name'))
        AND (sqlc.narg('search')::VARCHAR IS NULL
             OR n.name ILIKE '%' || sqlc.narg('search') || '%'
             OR n.display_name ILIKE '%' || sqlc.narg('search') || '%'
             OR n.description ILIKE '%' || sqlc.narg('search') || '%')
)
SELECT * FROM net_data
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN display_name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'display_name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN display_name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'cidr' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN cidr END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'cidr' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN cidr END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'subnet_count' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN subnet_count END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'subnet_count' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN subnet_count END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN updated_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
