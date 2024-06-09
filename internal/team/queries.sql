-- name: get-teams
SELECT name, uuid from teams where disabled is not true;

-- name: get-team
SELECT name, uuid from teams where disabled is not true and uuid = $1;