-- name: get-rules
select 
    rules
from automation_rules;

-- name: get-all
SELECT id, created_at, updated_at, name, description, type, rules from automation_rules;

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