-- name: CreateCertificate :one
INSERT INTO certificates (name, cert_type, common_name, dns_names, ip_addresses, ca_name,
    serial_number, certificate, private_key, not_before, not_after)
VALUES (@name, @cert_type, @common_name, @dns_names, @ip_addresses, @ca_name,
    @serial_number, @certificate, @private_key, @not_before, @not_after)
RETURNING *;

-- name: GetCertificateByID :one
SELECT * FROM certificates WHERE id = @id;

-- name: GetCertificateByName :one
SELECT * FROM certificates WHERE name = @name;

-- name: CountCertificatesByCAName :one
SELECT count(*) FROM certificates WHERE ca_name = @ca_name;

-- name: CountCertificates :one
SELECT count(*) FROM certificates
WHERE (sqlc.narg('cert_type')::VARCHAR IS NULL OR cert_type = sqlc.narg('cert_type'))
  AND (sqlc.narg('ca_name')::VARCHAR IS NULL OR ca_name = sqlc.narg('ca_name'))
  AND (sqlc.narg('search')::VARCHAR IS NULL
       OR name ILIKE '%' || sqlc.narg('search') || '%'
       OR common_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: ListCertificates :many
SELECT * FROM certificates
WHERE (sqlc.narg('cert_type')::VARCHAR IS NULL OR cert_type = sqlc.narg('cert_type'))
  AND (sqlc.narg('ca_name')::VARCHAR IS NULL OR ca_name = sqlc.narg('ca_name'))
  AND (sqlc.narg('search')::VARCHAR IS NULL
       OR name ILIKE '%' || sqlc.narg('search') || '%'
       OR common_name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN name END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'name' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN name END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    created_at DESC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;

-- name: DeleteCertificate :exec
DELETE FROM certificates WHERE id = @id;

-- name: DeleteCertificates :many
DELETE FROM certificates WHERE id = ANY(@ids::BIGINT[])
RETURNING id;
