-- name: CreateSubnet :one
INSERT INTO subnets (name, display_name, description, network_id, cidr, gateway, bitmap)
VALUES (@name, @display_name, @description, @network_id, @cidr, @gateway, @bitmap)
RETURNING id, name, display_name, description, network_id, cidr, gateway, bitmap, created_at, updated_at;

-- name: GetSubnetByID :one
SELECT
    s.id, s.name, s.display_name, s.description, s.network_id,
    s.cidr, s.gateway, s.bitmap,
    s.created_at, s.updated_at
FROM subnets s
WHERE s.id = @id;

-- name: GetSubnetByIDForUpdate :one
SELECT
    s.id, s.name, s.display_name, s.description, s.network_id,
    s.cidr, s.gateway, s.bitmap,
    s.created_at, s.updated_at
FROM subnets s
WHERE s.id = @id
FOR UPDATE;

-- name: UpdateSubnet :one
UPDATE subnets
SET name = @name,
    display_name = @display_name,
    description = @description,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, network_id, cidr, gateway, bitmap, created_at, updated_at;

-- name: PatchSubnet :one
UPDATE subnets
SET name = CASE WHEN sqlc.narg('name')::VARCHAR IS NOT NULL THEN sqlc.narg('name') ELSE name END,
    display_name = CASE WHEN sqlc.narg('display_name')::VARCHAR IS NOT NULL THEN sqlc.narg('display_name') ELSE display_name END,
    description = CASE WHEN sqlc.narg('description')::TEXT IS NOT NULL THEN sqlc.narg('description') ELSE description END,
    updated_at = now()
WHERE id = @id
RETURNING id, name, display_name, description, network_id, cidr, gateway, bitmap, created_at, updated_at;

-- name: UpdateSubnetBitmap :exec
UPDATE subnets SET bitmap = @bitmap, updated_at = now() WHERE id = @id;

-- name: UpdateSubnetGateway :exec
UPDATE subnets SET gateway = @gateway, updated_at = now() WHERE id = @id;

-- name: DeleteSubnet :exec
DELETE FROM subnets WHERE id = @id;

-- name: DeleteSubnetsByIDs :many
DELETE FROM subnets WHERE id = ANY(@ids::BIGINT[]) AND network_id = @network_id
RETURNING id;

-- name: ListSubnetCIDRsByNetworkID :many
SELECT id, cidr FROM subnets WHERE network_id = @network_id;

-- name: CountNonGatewayAllocationsBySubnetID :one
SELECT count(*) FROM ip_allocations WHERE subnet_id = @subnet_id AND is_gateway = false;

-- name: CountSubnets :one
SELECT count(*)
FROM subnets
WHERE network_id = @network_id
    AND (sqlc.narg('name')::VARCHAR IS NULL OR name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR name ILIKE '%' || sqlc.narg('search') || '%'
         OR display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR description ILIKE '%' || sqlc.narg('search') || '%'
         OR cidr ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListSubnets :many
SELECT
    s.id, s.name, s.display_name, s.description, s.network_id,
    s.cidr, s.gateway, s.bitmap,
    s.created_at, s.updated_at
FROM subnets s
WHERE s.network_id = @network_id
    AND (sqlc.narg('name')::VARCHAR IS NULL OR s.name ILIKE '%' || sqlc.narg('name') || '%')
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR s.name ILIKE '%' || sqlc.narg('search') || '%'
         OR s.display_name ILIKE '%' || sqlc.narg('search') || '%'
         OR s.description ILIKE '%' || sqlc.narg('search') || '%'
         OR s.cidr ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN s.name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN s.name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'cidr' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN s.cidr END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'cidr' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN s.cidr END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN s.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN s.created_at END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN s.updated_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'updated_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN s.updated_at END DESC,
    s.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
