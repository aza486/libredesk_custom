-- Run only against pg-n8n, never against the Libredesk database.
BEGIN;
CREATE TABLE IF NOT EXISTS learning_reviews (
 item_id uuid PRIMARY KEY REFERENCES public.rag_items(id),
 revision bigint NOT NULL DEFAULT 1,
 decision text NOT NULL DEFAULT 'pending' CHECK (decision IN ('pending','approved','rejected','disabled')),
 publication text NOT NULL DEFAULT 'unpublished' CHECK(publication IN ('unpublished','publishing','published','failed')),
 original jsonb NOT NULL,
 reviewer text,
 note text,
 checks jsonb NOT NULL DEFAULT '{}',
 attempts integer NOT NULL DEFAULT 0,
 retry_at timestamptz NOT NULL DEFAULT now(),
 lease uuid,
 started_at timestamptz,
 last_error text,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS learning_review_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 item_id uuid NOT NULL REFERENCES public.learning_reviews(item_id),
 revision bigint NOT NULL, actor text NOT NULL, action text NOT NULL,
 reason text NOT NULL, snapshot jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(item_id,revision)
);
CREATE TABLE IF NOT EXISTS learning_manual_origins (
 feedback_id uuid NOT NULL REFERENCES public.ai_feedback(id),
 source_type text NOT NULL, item_id uuid NOT NULL REFERENCES public.rag_items(id),
 PRIMARY KEY(feedback_id,source_type)
);
CREATE TABLE IF NOT EXISTS learning_derived_origins (
 parent_item_id uuid NOT NULL REFERENCES public.rag_items(id),
 item_id uuid NOT NULL REFERENCES public.rag_items(id),
 PRIMARY KEY(parent_item_id)
);
-- Covers late completion by an old v8 publisher and approval-v1 races as well.
CREATE OR REPLACE FUNCTION learning_review_activation_guard() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp AS $$
DECLARE v public.learning_reviews; managed boolean;
BEGIN
 SELECT * INTO v FROM public.learning_reviews WHERE item_id=NEW.id;
 managed := FOUND;
 IF managed AND current_setting('libredesk.review_writer',true) IS DISTINCT FROM 'true' THEN
   RAISE EXCEPTION 'managed learning content requires review transaction';
 END IF;
 IF NEW.active THEN
  IF NEW.approved IS DISTINCT FROM true OR NEW.quality_status IS DISTINCT FROM 'approved' OR NEW.embedding IS NULL
    OR public.vector_dims(NEW.embedding) <> 1536 THEN RAISE EXCEPTION 'invalid learning publication'; END IF;
  IF NEW.source_type='employee_fact' AND
    (NEW.learning_data->>'verification_status' IS DISTINCT FROM 'human_verified'
     OR NEW.learning_data->>'factual_use_approved' IS DISTINCT FROM 'true') THEN
    RAISE EXCEPTION 'human verification required';
  END IF;
  IF managed AND (v.decision <> 'approved' OR v.publication NOT IN ('publishing','published') OR NEW.learning_data->>'approval_method' IS DISTINCT FROM 'human_review_v1') THEN
    RAISE EXCEPTION 'review decision required';
  END IF;
  IF managed AND OLD.active AND (NEW.retrieval_text IS DISTINCT FROM OLD.retrieval_text
      OR NEW.human_answer IS DISTINCT FROM OLD.human_answer OR NEW.learning_data IS DISTINCT FROM OLD.learning_data) THEN
    RAISE EXCEPTION 'published content immutable';
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS learning_review_activation_guard ON public.rag_items;
CREATE TRIGGER learning_review_activation_guard BEFORE UPDATE ON public.rag_items
 FOR EACH ROW EXECUTE FUNCTION learning_review_activation_guard();
CREATE OR REPLACE FUNCTION learning_review_original_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.original IS DISTINCT FROM OLD.original THEN RAISE EXCEPTION 'original review snapshot immutable'; END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS learning_review_original_guard ON learning_reviews;
CREATE TRIGGER learning_review_original_guard BEFORE UPDATE ON learning_reviews
 FOR EACH ROW EXECUTE FUNCTION learning_review_original_guard();
CREATE OR REPLACE FUNCTION learning_review_audit_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'review audit is append only'; END $$;
DROP TRIGGER IF EXISTS learning_review_audit_guard ON learning_review_audit;
CREATE TRIGGER learning_review_audit_guard BEFORE UPDATE OR DELETE ON learning_review_audit
 FOR EACH ROW EXECUTE FUNCTION learning_review_audit_guard();
COMMIT;
