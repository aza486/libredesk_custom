-- name: get-enabled-rules
select
    type,
    rules
from automation_rules where disabled is not TRUE;

-- name: get-all
SELECT id, created_at, updated_at, name, description, type, rules, disabled from automation_rules where type = $1;

-- name: get-rule
SELECT id, created_at, updated_at, name, description, type, rules from automation_rules where id = $1;

-- name: update-rule
INSERT INTO automation_rules(id, name, description, type, rules)
VALUES($1, $2, $3, $4, $5)
ON CONFLICT (id)
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    rules = EXCLUDED.rules,
    updated_at = now()
WHERE $1 > 0;

-- name: insert-rule
INSERT into automation_rules (name, description, type, rules) VALUES ($1, $2, $3, $4);

-- name: delete-rule
delete from automation_rules where id = $1;

-- name: toggle-rule
UPDATE automation_rules 
SET disabled = NOT disabled, updated_at = NOW() 
WHERE id = $1;
