-- Privileged prerequisite, separate from schema owner and runtime accounts.
\set ON_ERROR_STOP on
BEGIN;
DO $$ BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 160000 AND 169999 THEN
  RAISE EXCEPTION 'This baseline is validated for PostgreSQL 16 only';
 END IF;
END $$;
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public VERSION '0.8.6';
DO $$ BEGIN
 IF NOT EXISTS(SELECT FROM pg_extension WHERE extname='vector' AND extversion='0.8.6') THEN
  RAISE EXCEPTION 'Expected pgvector 0.8.6; do not upgrade extensions implicitly';
 END IF;
END $$;
COMMIT;
