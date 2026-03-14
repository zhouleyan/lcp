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
