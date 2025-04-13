-- name: get-all-custom-attributes
SELECT
    id,
    created_at,
    updated_at,
    applies_to,
    name,
    description,
    key,
    values,
    data_type,
    regex
FROM
    custom_attribute_definitions
WHERE
    CASE WHEN $1 = '' THEN TRUE
         ELSE applies_to = $1
    END
ORDER BY
    updated_at DESC;

-- name: get-custom-attribute
SELECT
    id,
    created_at,
    updated_at,
    applies_to,
    name,
    description,
    key,
    values,
    data_type,
    regex
FROM
    custom_attribute_definitions
WHERE
    id = $1;

-- name: insert-custom-attribute
INSERT INTO
    custom_attribute_definitions (applies_to, name, description, key, values, data_type, regex)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)

-- name: delete-custom-attribute
DELETE FROM
    custom_attribute_definitions
WHERE
    id = $1;

-- name: update-custom-attribute
UPDATE
    custom_attribute_definitions
SET
    applies_to = $2,
    name = $3,
    description = $4,
    values = $5,
    data_type = $6,
    regex = $7,
    updated_at = NOW()
WHERE
    id = $1;