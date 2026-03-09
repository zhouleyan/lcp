-- name: UpsertPermission :one
INSERT INTO permissions (code, method, path, description)
VALUES (@code, @method, @path, @description)
ON CONFLICT (code) DO UPDATE
SET method = EXCLUDED.method,
    path = EXCLUDED.path,
    updated_at = now()
RETURNING id, code, method, path, description, created_at, updated_at;

-- name: DeletePermissionsByModulePrefix :exec
DELETE FROM permissions
WHERE code LIKE @module_prefix::VARCHAR || '%'
  AND code != ALL(@keep_codes::VARCHAR[]);

-- name: GetPermissionByCode :one
SELECT id, code, method, path, description, created_at, updated_at
FROM permissions
WHERE code = @code;

-- name: ListAllPermissionCodes :many
SELECT code FROM permissions ORDER BY code;

-- name: CountPermissions :one
SELECT count(id)
FROM permissions
WHERE (sqlc.narg('module_prefix')::VARCHAR IS NULL
       OR code LIKE sqlc.narg('module_prefix') || '%')
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       code ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
  ));

-- name: ListPermissions :many
SELECT id, code, method, path, description, created_at, updated_at
FROM permissions
WHERE (sqlc.narg('module_prefix')::VARCHAR IS NULL
       OR code LIKE sqlc.narg('module_prefix') || '%')
  AND (sqlc.narg('search')::VARCHAR IS NULL OR (
       code ILIKE '%' || sqlc.narg('search') || '%'
       OR description ILIKE '%' || sqlc.narg('search') || '%'
  ))
ORDER BY
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'code' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN code END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'code' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN code END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'method' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN method END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'method' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN method END DESC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'asc' THEN created_at END ASC,
    CASE WHEN sqlc.arg('sort_field')::VARCHAR = 'created_at' AND sqlc.arg('sort_order')::VARCHAR = 'desc' THEN created_at END DESC,
    code ASC
LIMIT sqlc.arg('page_size')::INT
OFFSET sqlc.arg('page_offset')::INT;
