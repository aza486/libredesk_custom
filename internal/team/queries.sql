-- name: get-teams
SELECT id, created_at, updated_at, name, auto_assign_conversations, disabled from teams order by updated_at desc;

-- name: get-teams-compact
SELECT id, name from teams order by name;

-- name: get-team
SELECT id, name, auto_assign_conversations from teams where id = $1;

-- name: get-team-members
SELECT u.id, t.id as team_id
FROM users u
JOIN team_members tm ON tm.user_id = u.id
JOIN teams t ON t.id = tm.team_id
WHERE t.name = $1;

-- name: insert-team
INSERT INTO teams (name) values($1);

-- name: update-team
UPDATE teams set name = $2, auto_assign_conversations = $3, updated_at = now() where id = $1;

-- name: upsert-user-teams
WITH delete_old_teams AS (
    DELETE FROM team_members 
    WHERE user_id = $1 
    AND team_id NOT IN (SELECT t.id FROM teams t WHERE t.name = ANY($2))
),
insert_new_teams AS (
    INSERT INTO team_members (user_id, team_id)
    SELECT $1, t.id 
    FROM teams t 
    WHERE t.name = ANY($2)
    ON CONFLICT DO NOTHING
)
SELECT 1;

-- name: delete-team
DELETE FROM teams where id = $1;