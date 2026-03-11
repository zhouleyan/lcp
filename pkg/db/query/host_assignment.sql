-- name: AssignHost :one
INSERT INTO host_assignments (host_id, workspace_id, namespace_id)
VALUES (@host_id, @workspace_id, @namespace_id)
RETURNING *;

-- name: UnassignHostWorkspace :exec
DELETE FROM host_assignments
WHERE host_id = @host_id AND workspace_id = @workspace_id;

-- name: UnassignHostNamespace :exec
DELETE FROM host_assignments
WHERE host_id = @host_id AND namespace_id = @namespace_id;

-- name: ListAssignmentsByHostID :many
SELECT
    ha.*,
    h.name AS host_name,
    w.name AS workspace_name,
    n.name AS namespace_name
FROM host_assignments ha
JOIN hosts h ON h.id = ha.host_id
LEFT JOIN workspaces w ON w.id = ha.workspace_id
LEFT JOIN namespaces n ON n.id = ha.namespace_id
WHERE ha.host_id = @host_id
ORDER BY ha.created_at DESC;
