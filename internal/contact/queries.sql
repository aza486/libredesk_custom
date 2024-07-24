-- name: upsert-contact
-- Check if contact exists
WITH existing_contact AS (
    SELECT contact_id
    FROM public.contact_methods
    WHERE source = $1 AND source_id = $2
)

-- Insert contact if it does not exist
, ins_contact AS (
    INSERT INTO public.contacts (first_name, last_name, email, phone_number, avatar_url)
    SELECT $4, $5, $6, $7, $8
    WHERE NOT EXISTS (SELECT 1 FROM existing_contact)
    RETURNING id
)

-- Determine which contact ID to use
, final_contact AS (
    SELECT contact_id AS id FROM existing_contact
    UNION ALL
    SELECT id FROM ins_contact
    LIMIT 1
)

-- Insert contact method if it does not exist
, ins_contact_method AS (
    INSERT INTO public.contact_methods (contact_id, source, source_id, inbox_id)
    SELECT id, $1, $2, $3
    FROM final_contact
    ON CONFLICT DO NOTHING
)

-- Return the final contact ID
SELECT id AS contact_id FROM final_contact;

