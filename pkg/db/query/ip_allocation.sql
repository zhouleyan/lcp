-- name: CreateIPAllocation :one
INSERT INTO ip_allocations (subnet_id, ip, description, is_gateway)
VALUES (@subnet_id, @ip, @description, @is_gateway)
RETURNING id, subnet_id, ip, description, is_gateway, created_at, host_id;

-- name: GetIPAllocationByID :one
SELECT id, subnet_id, ip, description, is_gateway, created_at, host_id
FROM ip_allocations
WHERE id = @id;

-- name: GetIPAllocationBySubnetAndIP :one
SELECT id, subnet_id, ip, description, is_gateway, created_at, host_id
FROM ip_allocations
WHERE subnet_id = @subnet_id AND ip = @ip;

-- name: DeleteIPAllocation :exec
DELETE FROM ip_allocations WHERE id = @id;

-- name: DeleteIPAllocationsBySubnetID :exec
DELETE FROM ip_allocations WHERE subnet_id = @subnet_id;

-- name: CountIPAllocations :one
SELECT count(*)
FROM ip_allocations
WHERE subnet_id = @subnet_id
    AND (sqlc.narg('is_gateway')::BOOLEAN IS NULL OR is_gateway = sqlc.narg('is_gateway'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ip ILIKE '%' || sqlc.narg('search') || '%'
         OR description ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListIPAllocations :many
SELECT ia.id, ia.subnet_id, ia.ip, ia.description, ia.is_gateway, ia.created_at, ia.host_id, h.name AS host_name
FROM ip_allocations ia
LEFT JOIN hosts h ON ia.host_id = h.id
WHERE ia.subnet_id = @subnet_id
    AND (sqlc.narg('is_gateway')::BOOLEAN IS NULL OR ia.is_gateway = sqlc.narg('is_gateway'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ia.ip ILIKE '%' || sqlc.narg('search') || '%'
         OR ia.description ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ia.ip END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ia.ip END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ia.created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ia.created_at END DESC,
    ia.created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
