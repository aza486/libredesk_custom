-- Explicit installation in pg-n8n by the database owner. Never application startup.
-- No backfill, no changes to existing views, no grants to existing publishers.
BEGIN;
DO $$ BEGIN
 IF format_type((SELECT atttypid FROM pg_attribute WHERE attrelid='public.rag_items'::regclass AND attname='embedding'),
                (SELECT atttypmod FROM pg_attribute WHERE attrelid='public.rag_items'::regclass AND attname='embedding')) <> 'vector(1536)'
 THEN RAISE EXCEPTION 'v11 requires vector(1536)'; END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS rag_case_feedback_v11_uidx
 ON public.rag_items(feedback_id) WHERE source_type='case_example';
-- Recreate this named CHECK so rerunning SQL005 also relaxes the old nonempty reason contract.
ALTER TABLE public.rag_items DROP CONSTRAINT IF EXISTS rag_case_contract_v11;
ALTER TABLE public.rag_items ADD CONSTRAINT rag_case_contract_v11 CHECK (
   source_type IS DISTINCT FROM 'case_example' OR (
    feedback_id IS NOT NULL AND source_id=feedback_id::text AND scope='global' AND ai_answer IS NULL
    AND length(btrim(customer_question)) BETWEEN 12 AND 6000 AND length(btrim(human_answer)) BETWEEN 1 AND 6000
    AND length(btrim(feedback_reason)) BETWEEN 0 AND 6000 AND length(btrim(retrieval_text)) BETWEEN 12 AND 900
    AND learning_data @> '{"pipeline_version":11,"case_version":11,"target_agent":"case_retrieval","example_role":"historical_resolution","factual_authority":false,"factual_use_approved":false,"verification_status":"not_applicable","requires_quality_review":true}'
    AND learning_data->>'case_status' IN ('staged','approved','published','rejected','blocked','disabled')
    AND (learning_data->>'case_revision')::bigint >= 0
    AND (NOT active OR (approved AND quality_status='approved' AND embedding IS NOT NULL AND learning_data @> '{"case_status":"published","privacy_checked":true,"quality_checked":true,"generalization_checked":true}'))
   ) IS TRUE);
-- Existing global HNSW cosine index remains usable; do not duplicate it.
CREATE TABLE IF NOT EXISTS public.learning_case_audit_v11 (
 item_id uuid NOT NULL REFERENCES public.rag_items(id),
 revision bigint NOT NULL CHECK (revision >= 0),
 action text NOT NULL CHECK (action IN ('stage','approved','rejected','blocked','disabled','publish')),
 actor text NOT NULL CHECK (length(actor) BETWEEN 1 AND 100),
 db_actor name NOT NULL DEFAULT session_user,
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(item_id,revision)
);

-- Fail closed on malformed envelopes. Blocked/insufficient carry only fixed codes,
-- never free text, questions, answers, reasons, categories or arbitrary metadata.
CREATE OR REPLACE FUNCTION public.learning_case_envelope_valid_v11(c jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,public AS $$
DECLARE k text;
BEGIN
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR c->'version' IS DISTINCT FROM '11'::jsonb THEN RETURN false; END IF;
 IF c->>'status' IN ('blocked','insufficient') THEN
  FOR k IN SELECT jsonb_object_keys(c) LOOP
   IF k NOT IN ('version','status','reason_code') THEN RETURN false; END IF;
  END LOOP;
  RETURN COALESCE(NOT c ? 'reason_code' OR c->>'reason_code' IN ('privacy','security','no_resolution','no_explicit_feedback','insufficient_context'),false);
 END IF;
 IF c->>'status' IS DISTINCT FROM 'staged'
  OR c->'has_human_resolution' IS DISTINCT FROM 'true'::jsonb
  OR jsonb_typeof(c->'has_explicit_feedback') IS DISTINCT FROM 'boolean'
  OR c->'requires_quality_review' IS DISTINCT FROM 'true'::jsonb
  OR jsonb_typeof(c->'feedback_type') IS DISTINCT FROM 'string'
  OR c->>'feedback_type' NOT IN ('response_feedback','manual_resolution','category_correction','general_feedback')
  OR NOT c ? 'feedback_type' THEN RETURN false; END IF;
 IF length(btrim(c->>'customer_question')) < 12 THEN RETURN false; END IF;
 FOR k IN SELECT jsonb_object_keys(c) LOOP
  IF k NOT IN ('version','status','customer_question','human_answer','feedback_reason','feedback_type',
   'original_categories','corrected_categories','has_human_resolution','has_explicit_feedback','requires_quality_review') THEN RETURN false; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['customer_question','human_answer','feedback_reason'] LOOP
  IF jsonb_typeof(c->k) IS DISTINCT FROM 'string' OR length(c->>k) > 6000
   OR (k <> 'feedback_reason' AND length(btrim(c->>k))=0) THEN RETURN false; END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['original_categories','corrected_categories'] LOOP
  IF jsonb_typeof(c->k) IS DISTINCT FROM 'array' THEN RETURN false; END IF;
  IF jsonb_array_length(c->k)>20 OR EXISTS(SELECT FROM jsonb_array_elements(c->k) x
   WHERE jsonb_typeof(x) <> 'string' OR (x#>>'{}') !~ '^[A-Za-z][A-Za-z0-9_-]{0,63}$') THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
END $$;

-- Persisted Receiver provenance, not assertions inside the draft/canonical object.
-- LibreDesk additionally verifies actual SENT messages and sender identity on review.
CREATE OR REPLACE FUNCTION public.learning_case_human_source_valid_v11(f public.ai_feedback)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,public AS $$
 SELECT COALESCE(f.feedback_data->>'response_source'='human'
  AND f.agent_id ~ '^[1-9][0-9]*$'
  AND f.outgoing_message_uuid ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  AND f.mail_event_message_uuid ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  AND length(btrim(f.human_answer))>0 AND octet_length(f.human_answer)<=1048576,false)
$$;

CREATE OR REPLACE FUNCTION public.learning_case_feedback_guard_v11() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp AS $$
DECLARE c jsonb;
BEGIN
 c := NEW.feedback_data->'learning_case_v11';
 IF TG_OP='UPDATE' AND OLD.feedback_data ? 'learning_case_v11' THEN
  IF NEW.agent_id IS DISTINCT FROM OLD.agent_id OR NEW.outgoing_message_uuid IS DISTINCT FROM OLD.outgoing_message_uuid
   OR NEW.mail_event_message_uuid IS DISTINCT FROM OLD.mail_event_message_uuid OR NEW.human_answer IS DISTINCT FROM OLD.human_answer
   OR NEW.feedback_type IS DISTINCT FROM OLD.feedback_type
   OR NEW.feedback_data->'response_source' IS DISTINCT FROM OLD.feedback_data->'response_source' THEN
   RAISE EXCEPTION 'canonical case source immutable';
  END IF;
  IF c IS NOT DISTINCT FROM OLD.feedback_data->'learning_case_v11' THEN RETURN NEW; END IF;
  IF c IS NULL OR c->>'status' IS DISTINCT FROM 'blocked' THEN
   RAISE EXCEPTION 'canonical case immutable except blocking';
  END IF;
 END IF;
 IF c IS NOT NULL AND public.learning_case_envelope_valid_v11(c) IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'invalid learning_case_v11 envelope';
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS learning_case_feedback_guard_v11 ON public.ai_feedback;
CREATE TRIGGER learning_case_feedback_guard_v11 BEFORE INSERT OR UPDATE OF feedback_data,agent_id,outgoing_message_uuid,mail_event_message_uuid,human_answer,feedback_type ON public.ai_feedback
 FOR EACH ROW EXECUTE FUNCTION public.learning_case_feedback_guard_v11();

-- All case writes require the trusted function owner; a forgeable session GUC
-- cannot authorize a pipeline connection. Installation owner must be a trusted role.
CREATE OR REPLACE FUNCTION public.learning_case_write_guard_v11() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,public AS $$
BEGIN
 IF TG_TABLE_NAME='rag_items' THEN
  IF TG_OP='INSERT' THEN
   IF NEW.source_type IS DISTINCT FROM 'case_example' THEN RETURN NEW; END IF;
  ELSIF TG_OP='DELETE' THEN
   IF OLD.source_type IS DISTINCT FROM 'case_example' THEN RETURN OLD; END IF;
   RAISE EXCEPTION 'case provenance must be retained';
  ELSE
   IF OLD.source_type IS DISTINCT FROM 'case_example' AND NEW.source_type IS DISTINCT FROM 'case_example' THEN RETURN NEW; END IF;
   IF OLD.source_type IS DISTINCT FROM NEW.source_type OR
    (to_jsonb(NEW)-ARRAY['embedding','active','approved','quality_status','learning_data','updated_at']) IS DISTINCT FROM
    (to_jsonb(OLD)-ARRAY['embedding','active','approved','quality_status','learning_data','updated_at']) THEN
    RAISE EXCEPTION 'case content and origin immutable';
   END IF;
  END IF;
 ELSE
  IF TG_OP IN ('UPDATE','DELETE') THEN RAISE EXCEPTION 'case audit is append only'; END IF;
 END IF;
 IF current_user IS DISTINCT FROM pg_get_userbyid((SELECT proowner FROM pg_proc
   WHERE oid='public.learning_case_stage_v11(uuid)'::regprocedure)) THEN
  RAISE EXCEPTION 'case write requires controlled v11 function';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION public.learning_case_stage_v11(p_feedback uuid) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp AS $$
DECLARE c jsonb; i uuid; ft text; f public.ai_feedback;
BEGIN
 SELECT * INTO f FROM public.ai_feedback WHERE id=p_feedback FOR UPDATE;
 c:=f.feedback_data->'learning_case_v11'; ft:=f.feedback_type;
 IF NOT FOUND THEN RAISE EXCEPTION 'feedback not found'; END IF;
 IF public.learning_case_envelope_valid_v11(c) IS DISTINCT FROM true OR c->>'status' IS DISTINCT FROM 'staged'
  OR ft IS DISTINCT FROM c->>'feedback_type' OR NOT public.learning_case_human_source_valid_v11(f) THEN RAISE EXCEPTION 'case is not staged'; END IF;
 SELECT id INTO i FROM public.rag_items WHERE feedback_id=p_feedback AND source_type='case_example';
 IF FOUND THEN RETURN i; END IF;
 i := gen_random_uuid();
 INSERT INTO public.rag_items(id,source_type,scope,source_id,feedback_id,topic,customer_question,human_answer,feedback_reason,
  retrieval_text,embedding,approved,active,quality_status,learning_data)
 VALUES(i,'case_example','global',p_feedback::text,p_feedback,'Historischer Lösungsfall',c->>'customer_question',c->>'human_answer',c->>'feedback_reason',
  left(c->>'customer_question',900),NULL,false,false,'pending',jsonb_build_object(
   'pipeline_version',11,'case_version',11,'case_revision',0,'case_status','staged','publish_state','unpublished',
   'target_agent','case_retrieval','example_role','historical_resolution','factual_authority',false,'factual_use_approved',false,
   'verification_status','not_applicable','requires_quality_review',true,'privacy_checked',false,'quality_checked',false,
   'generalization_checked',false,'has_explicit_feedback',c->'has_explicit_feedback','feedback_type',ft,'original_categories',c->'original_categories',
   'corrected_categories',c->'corrected_categories','confirmed_categories','[]'::jsonb,'origin_digest',md5(c::text)));
 INSERT INTO public.learning_case_audit_v11(item_id,revision,action,actor) VALUES(i,0,'stage',session_user);
 RETURN i;
END $$;

-- Explicit human quality decision. Categories are confirmed separately; a correction
-- in the staging envelope alone is not confirmation. No embedding is accepted here.
CREATE OR REPLACE FUNCTION public.learning_case_review_v11(p_id uuid,p_revision bigint,p_decision text,p_actor text,
 p_privacy boolean DEFAULT false,p_quality boolean DEFAULT false,p_generalization boolean DEFAULT false,p_categories jsonb DEFAULT '[]')
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp AS $$
DECLARE r public.rag_items; rev bigint;
BEGIN
 SELECT * INTO r FROM public.rag_items WHERE id=p_id AND source_type='case_example' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'case not found'; END IF;
 rev := (r.learning_data->>'case_revision')::bigint;
 IF rev IS DISTINCT FROM p_revision OR p_decision IS NULL OR p_decision NOT IN ('approved','rejected','blocked','disabled')
  OR length(btrim(p_actor)) NOT BETWEEN 1 AND 100 OR p_actor IS NULL THEN RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='invalid case decision or revision'; END IF;
 IF p_decision='approved' THEN
  IF r.learning_data->>'case_status' IS DISTINCT FROM 'staged' OR p_privacy IS DISTINCT FROM true
   OR p_quality IS DISTINCT FROM true OR p_generalization IS DISTINCT FROM true THEN RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='case quality review required'; END IF;
  PERFORM 1 FROM public.ai_feedback f WHERE f.id=r.feedback_id
   AND public.learning_case_human_source_valid_v11(f)
   AND f.feedback_data->'learning_case_v11'->>'status'='staged'
   AND md5((f.feedback_data->'learning_case_v11')::text)=r.learning_data->>'origin_digest' FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='case origin unavailable or blocked'; END IF;
  IF jsonb_typeof(p_categories) IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'invalid confirmed categories'; END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(p_categories) x WHERE jsonb_typeof(x)<>'string'
   OR (x#>>'{}') !~ '^[A-Za-z][A-Za-z0-9_-]{0,63}$') OR jsonb_array_length(p_categories)>20 THEN RAISE EXCEPTION 'invalid confirmed categories'; END IF;
 END IF;
 UPDATE public.rag_items SET active=false,approved=false,embedding=NULL,quality_status=CASE WHEN p_decision='approved' THEN 'approved' ELSE p_decision END,
  updated_at=now(),learning_data=learning_data||jsonb_build_object('case_status',p_decision,'case_revision',rev+1,
  'publish_state',CASE WHEN p_decision='approved' THEN 'unpublished' ELSE 'blocked' END,
  'privacy_checked',p_decision='approved','quality_checked',p_decision='approved','generalization_checked',p_decision='approved',
  'confirmed_categories',CASE WHEN p_decision='approved' THEN p_categories ELSE '[]'::jsonb END,'reviewer',p_actor,'reviewed_at',now()) WHERE id=p_id;
 INSERT INTO public.learning_case_audit_v11(item_id,revision,action,actor) VALUES(p_id,rev+1,p_decision,p_actor);
 RETURN rev+1;
END $$;

CREATE OR REPLACE FUNCTION public.learning_case_publish_v11(p_id uuid,p_revision bigint,p_embedding public.vector)
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public,pg_temp AS $$
DECLARE r public.rag_items; rev bigint;
BEGIN
 SELECT * INTO r FROM public.rag_items WHERE id=p_id AND source_type='case_example' FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'case not found'; END IF;
 rev := (r.learning_data->>'case_revision')::bigint;
 IF rev IS DISTINCT FROM p_revision OR r.learning_data->>'case_status' IS DISTINCT FROM 'approved'
  OR r.learning_data->>'privacy_checked' IS DISTINCT FROM 'true' OR r.learning_data->>'quality_checked' IS DISTINCT FROM 'true'
  OR r.learning_data->>'generalization_checked' IS DISTINCT FROM 'true'
  OR p_embedding IS NULL OR public.vector_dims(p_embedding)<>1536 OR public.vector_norm(p_embedding)=0 THEN
  RAISE EXCEPTION 'case publication requires current review and valid vector';
 END IF;
 PERFORM 1 FROM public.ai_feedback f WHERE id=r.feedback_id AND public.learning_case_human_source_valid_v11(f) AND feedback_data->'learning_case_v11'->>'status'='staged'
  AND md5((feedback_data->'learning_case_v11')::text)=r.learning_data->>'origin_digest' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'case origin unavailable or blocked'; END IF;
 UPDATE public.rag_items SET embedding=p_embedding,approved=true,active=true,quality_status='approved',updated_at=now(),
  learning_data=learning_data||jsonb_build_object('case_status','published','case_revision',rev+1,'publish_state','published',
   'embedding_model','text-embedding-3-small','published_at',now()) WHERE id=p_id;
 INSERT INTO public.learning_case_audit_v11(item_id,revision,action,actor) VALUES(p_id,rev+1,'publish',session_user);
 RETURN rev+1;
END $$;

DROP TRIGGER IF EXISTS learning_case_write_guard_v11 ON public.rag_items;
CREATE TRIGGER learning_case_write_guard_v11 BEFORE INSERT OR UPDATE OR DELETE ON public.rag_items
 FOR EACH ROW EXECUTE FUNCTION public.learning_case_write_guard_v11();
DROP TRIGGER IF EXISTS learning_case_audit_guard_v11 ON public.learning_case_audit_v11;
CREATE TRIGGER learning_case_audit_guard_v11 BEFORE INSERT OR UPDATE OR DELETE ON public.learning_case_audit_v11
 FOR EACH ROW EXECUTE FUNCTION public.learning_case_write_guard_v11();

-- Never let a case enter the existing employee-fact review/publisher queue.
CREATE OR REPLACE FUNCTION public.learning_case_review_separation_v11() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,public AS $$
BEGIN
 IF EXISTS(SELECT FROM public.rag_items WHERE id=NEW.item_id AND source_type='case_example') THEN
  RAISE EXCEPTION 'case examples require separate quality review';
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS learning_case_review_separation_v11 ON public.learning_reviews;
CREATE TRIGGER learning_case_review_separation_v11 BEFORE INSERT OR UPDATE ON public.learning_reviews
 FOR EACH ROW EXECUTE FUNCTION public.learning_case_review_separation_v11();

CREATE OR REPLACE VIEW public.rag_case_example_view AS
SELECT r.id,r.embedding,r.retrieval_text AS text,
 jsonb_build_object('rag_item_id',r.id::text,'feedback_id',r.feedback_id::text,'source_type','case_example',
  'case_version',11,'case_revision',r.learning_data->'case_revision','origin_digest',r.learning_data->>'origin_digest',
  'target_agent','case_retrieval','example_role','historical_resolution','factual_authority',false,'factual_use_approved',false,
  'verification_status','not_applicable','question',r.customer_question,'human_answer',r.human_answer,
  'feedback_reason',r.feedback_reason,'confirmed_categories',r.learning_data->'confirmed_categories') AS metadata
FROM public.rag_items r JOIN public.ai_feedback f ON f.id=r.feedback_id
WHERE r.source_type='case_example' AND r.active=true AND r.approved=true AND r.quality_status='approved'
 AND r.embedding IS NOT NULL AND public.vector_dims(r.embedding)=1536 AND public.vector_norm(r.embedding)>0
 AND r.learning_data @> '{"case_version":11,"case_status":"published","publish_state":"published","privacy_checked":true,"quality_checked":true,"generalization_checked":true,"factual_authority":false,"factual_use_approved":false,"target_agent":"case_retrieval","example_role":"historical_resolution"}'
 AND length(btrim(r.retrieval_text)) BETWEEN 12 AND 900
 AND length(btrim(r.customer_question))>0 AND length(btrim(r.human_answer))>0
 AND public.learning_case_human_source_valid_v11(f)
 AND f.feedback_data->'learning_case_v11'->>'status'='staged'
 AND md5((f.feedback_data->'learning_case_v11')::text)=r.learning_data->>'origin_digest';

-- SECURITY DEFINER entry points are opt-in. Owner/superuser is trusted; pipeline
-- roles must never own these functions, tables, or schema, or inherit the owner.
REVOKE ALL ON FUNCTION public.learning_case_stage_v11(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.learning_case_review_v11(uuid,bigint,text,text,boolean,boolean,boolean,jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.learning_case_publish_v11(uuid,bigint,public.vector) FROM PUBLIC;
REVOKE ALL ON public.learning_case_audit_v11 FROM PUBLIC;
COMMIT;
