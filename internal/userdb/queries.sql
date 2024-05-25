-- name: get-agents
SELECT first_name, last_name, uuid from agents where disabled is not true;

-- name: get-agent
select id, email, password, avatar_url, first_name, last_name, uuid from agents where email = $1;

-- name: set-agent-password
update agents set password = $1 where id = $2;

-- name: get-teams
SELECT name, uuid from teams where disabled is not true;
