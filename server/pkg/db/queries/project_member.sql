-- name: ListAccessibleProjectIds :many
SELECT p.id
FROM project_member pm
JOIN project p ON p.id = pm.project_id
WHERE p.workspace_id = $1
  AND pm.user_id = $2;

-- name: GrantProjectMember :one
INSERT INTO project_member (project_id, user_id)
VALUES ($1, $2)
ON CONFLICT (project_id, user_id) DO NOTHING
RETURNING *;

-- name: RevokeProjectMember :exec
DELETE FROM project_member
WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectMembers :many
SELECT pm.id, pm.project_id, pm.user_id, pm.created_at,
       u.name AS user_name, u.email AS user_email, u.avatar_url AS user_avatar_url
FROM project_member pm
JOIN "user" u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at ASC;
