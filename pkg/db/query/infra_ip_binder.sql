-- name: GetSubnetByIDForUpdateACL :one
SELECT
    s.id, s.name, s.display_name, s.description, s.network_id,
    s.cidr, s.gateway, s.bitmap,
    s.created_at, s.updated_at
FROM subnets s
WHERE s.id = @id
FOR UPDATE;

-- name: UpdateSubnetBitmapACL :exec
UPDATE subnets SET bitmap = @bitmap, updated_at = now() WHERE id = @id;

-- name: CreateIPAllocationWithHost :one
INSERT INTO ip_allocations (subnet_id, ip, description, is_gateway, host_id)
VALUES (@subnet_id, @ip, @description, @is_gateway, @host_id)
RETURNING id, subnet_id, ip, description, is_gateway, host_id, created_at;

-- name: UnbindIPAllocationFromHost :execrows
UPDATE ip_allocations SET host_id = NULL WHERE id = @id AND host_id = @host_id;

-- name: ListIPAllocationsByHostID :many
SELECT ia.id, ia.subnet_id, ia.ip, ia.description, ia.is_gateway, ia.created_at, ia.host_id,
       s.name AS subnet_name, s.cidr AS subnet_cidr
FROM ip_allocations ia
JOIN subnets s ON ia.subnet_id = s.id
WHERE ia.host_id = @host_id
ORDER BY ia.created_at;

-- name: GetIPAllocationForHost :one
SELECT ia.id, ia.subnet_id, ia.ip, ia.host_id
FROM ip_allocations ia
WHERE ia.id = @id AND ia.host_id = @host_id;
