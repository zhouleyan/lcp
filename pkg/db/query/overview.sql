-- name: GetPlatformStats :one
SELECT
    (SELECT count(*) FROM workspaces) AS workspace_count,
    (SELECT count(*) FROM namespaces) AS namespace_count,
    (SELECT count(*) FROM users) AS user_count,
    (SELECT count(*) FROM roles WHERE scope = 'platform') AS role_count,
    (SELECT count(*) FROM role_bindings WHERE scope = 'platform') AS role_binding_count;

-- name: GetWorkspaceStats :one
SELECT
    (SELECT count(*) FROM namespaces n WHERE n.workspace_id = @workspace_id) AS namespace_count,
    (SELECT count(DISTINCT rb.user_id) FROM role_bindings rb WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id) AS member_count,
    (SELECT count(*) FROM roles r WHERE r.scope = 'workspace' AND r.workspace_id = @workspace_id) AS role_count,
    (SELECT count(*) FROM role_bindings rb WHERE rb.scope = 'workspace' AND rb.workspace_id = @workspace_id) AS role_binding_count;

-- name: GetNamespaceStats :one
SELECT
    (SELECT count(DISTINCT rb.user_id) FROM role_bindings rb WHERE rb.scope = 'namespace' AND rb.namespace_id = sqlc.arg('namespace_id')::BIGINT) AS member_count,
    (SELECT count(*) FROM roles r WHERE r.scope = 'namespace' AND r.namespace_id = sqlc.arg('namespace_id')::BIGINT) AS role_count,
    (SELECT count(*) FROM role_bindings rb WHERE rb.scope = 'namespace' AND rb.namespace_id = sqlc.arg('namespace_id')::BIGINT) AS role_binding_count;
