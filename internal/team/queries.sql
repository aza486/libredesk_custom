-- name: get-teams
SELECT id, name, uuid from teams where disabled is not true;

-- name: get-team
SELECT id, name, uuid from teams where disabled is not true and uuid = $1;

-- name: get-team-members
SELECT u.id, u.uuid, t.id as team_id
FROM users u
JOIN team_members tm ON tm.user_id = u.id
JOIN teams t ON t.id = tm.team_id
WHERE t.name = $1;

