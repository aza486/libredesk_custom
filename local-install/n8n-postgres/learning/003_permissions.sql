BEGIN;
-- Run as database owner only after configuring this role's password externally.
-- No password or production DSN belongs in source control.
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='libredesk_learning') THEN
   CREATE ROLE libredesk_learning LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
 END IF;
END $$;
GRANT USAGE ON SCHEMA public TO libredesk_learning;
GRANT SELECT,INSERT,UPDATE ON public.rag_items TO libredesk_learning;
GRANT SELECT ON public.ai_feedback TO libredesk_learning;
GRANT SELECT(message_uuid,clean_text,categories) ON public.mail_events TO libredesk_learning;
GRANT SELECT,INSERT,UPDATE ON public.learning_reviews TO libredesk_learning;
GRANT SELECT,INSERT ON public.learning_manual_origins,public.learning_derived_origins,public.learning_review_audit TO libredesk_learning;
GRANT USAGE,SELECT ON SEQUENCE public.learning_review_audit_id_seq TO libredesk_learning;
-- No mail_events.raw_text privilege, no feedback writes, no DELETE or audit UPDATE.

COMMIT;
