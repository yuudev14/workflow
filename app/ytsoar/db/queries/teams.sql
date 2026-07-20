-- name: GetTeamById :one
SELECT * FROM teams WHERE id = $1;

-- name: CreateTeam :one
INSERT INTO teams (name, description) VALUES ($1, $2) RETURNING *;

-- name: UpdateTeam :one
UPDATE teams
SET
    name = CASE
        WHEN sqlc.arg('name_set')::boolean THEN sqlc.narg('name')
        ELSE name
    END,
    description = CASE
        WHEN sqlc.arg('description_set')::boolean THEN sqlc.narg('description')
        ELSE description
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteTeam :execrows
DELETE FROM teams WHERE id = $1;

-- name: InsertTeamMember :exec
INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteTeamMembers :exec
DELETE FROM team_members WHERE team_id = $1;
