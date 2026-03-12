-- name: CreateIPAllocation :one
INSERT INTO ip_allocations (subnet_id, ip, description, is_gateway)
VALUES (@subnet_id, @ip, @description, @is_gateway)
RETURNING id, subnet_id, ip, description, is_gateway, created_at;

-- name: GetIPAllocationByID :one
SELECT id, subnet_id, ip, description, is_gateway, created_at
FROM ip_allocations
WHERE id = @id;

-- name: GetIPAllocationBySubnetAndIP :one
SELECT id, subnet_id, ip, description, is_gateway, created_at
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
SELECT id, subnet_id, ip, description, is_gateway, created_at
FROM ip_allocations
WHERE subnet_id = @subnet_id
    AND (sqlc.narg('is_gateway')::BOOLEAN IS NULL OR is_gateway = sqlc.narg('is_gateway'))
    AND (sqlc.narg('search')::VARCHAR IS NULL
         OR ip ILIKE '%' || sqlc.narg('search') || '%'
         OR description ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN ip END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'ip' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN ip END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
