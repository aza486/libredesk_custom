-- name: get-default-provider
SELECT id, name, provider, config, is_default FROM ai_providers where is_default is true;

-- name: get-prompt
SELECT id, key, title, content FROM ai_prompts where key = $1;

-- name: get-prompts
SELECT id, key, title FROM ai_prompts order by title;