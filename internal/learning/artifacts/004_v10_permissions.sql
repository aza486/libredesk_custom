-- pg-n8n only, separately approved installation by database owner.
-- psql -v ON_ERROR_STOP=1 -v publisher_role="$N8N_PUBLISHER_DB_ROLE" -f 004_v10_permissions.sql
-- v10 creates the immutable original atomically with each prepared fact.
-- Existing SELECT(item_id) (v9 prerequisite) remains sufficient for conflict handling.
GRANT INSERT(item_id,revision,original) ON public.learning_reviews TO :"publisher_role";
-- No UPDATE, reviewer/checks/publication privileges, or access to customer raw_text.
