-- name: get-enabled-rules
select
    type,
    events,
    rules,
    execution_mode
from automation_rules where disabled is not TRUE ORDER BY weight ASC;

-- name: get-all
SELECT id, created_at, updated_at, name, description, type, events, rules, disabled, execution_mode from automation_rules where type = $1 ORDER BY weight ASC;

-- name: get-rule
SELECT id, created_at, updated_at, name, description, type, events, rules, execution_mode from automation_rules where id = $1;

-- name: update-rule
INSERT INTO automation_rules(id, name, description, type, events, rules)
VALUES($1, $2, $3, $4, $5, $6)
ON CONFLICT (id)
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    events = EXCLUDED.events,
    rules = EXCLUDED.rules,
    updated_at = now()
WHERE $1 > 0;

-- name: insert-rule
INSERT into automation_rules (name, description, type, events, rules) values ($1, $2, $3, $4, $5);

-- name: delete-rule
delete from automation_rules where id = $1;

-- name: toggle-rule
UPDATE automation_rules 
SET disabled = NOT disabled, updated_at = NOW() 
WHERE id = $1;

-- name: update-rule-weight
UPDATE automation_rules
SET weight = $2, updated_at = NOW()
WHERE id = $1;

-- name: update-rule-execution-mode
UPDATE automation_rules
SET execution_mode = $2, updated_at = NOW()
WHERE type = $1;