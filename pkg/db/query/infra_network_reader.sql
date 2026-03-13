-- name: ListActiveNetworksWithSubnetCount :many
-- ACL query: lists active networks for host IP allocation (infra module).
SELECT
    n.id, n.name, n.display_name, n.description, n.cidr,
    n.is_public, n.status,
    n.created_at, n.updated_at,
    (SELECT count(*) FROM subnets s WHERE s.network_id = n.id) AS subnet_count
FROM networks n
WHERE n.status = 'active'
ORDER BY n.name ASC;

-- name: ListSubnetsByNetworkIDs :many
-- ACL query: lists subnets for the given network IDs with IP usage stats.
SELECT
    s.id, s.name, s.display_name, s.description,
    s.network_id, s.cidr, s.gateway, s.bitmap,
    s.created_at, s.updated_at
FROM subnets s
WHERE s.network_id = ANY(@network_ids::BIGINT[])
ORDER BY s.network_id, s.name ASC;
