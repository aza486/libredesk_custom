BEGIN;

CREATE OR REPLACE FUNCTION public.learning_case_stage_v11(p_feedback uuid)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path TO 'pg_catalog', 'public', 'pg_temp'
AS $function$
DECLARE
  c jsonb;
  i uuid;
  ft text;
  f public.ai_feedback;
BEGIN
  SELECT * INTO f
  FROM public.ai_feedback
  WHERE id = p_feedback
  FOR UPDATE;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'feedback not found';
  END IF;

  c := f.feedback_data->'learning_case_v11';
  ft := f.feedback_type;

  IF public.learning_case_envelope_valid_v11(c) IS DISTINCT FROM true
     OR c->>'status' IS DISTINCT FROM 'staged'
     OR ft IS DISTINCT FROM c->>'feedback_type'
     OR NOT public.learning_case_human_source_valid_v11(f)
  THEN
    RAISE EXCEPTION 'case is not staged';
  END IF;

  SELECT id INTO i
  FROM public.rag_items
  WHERE feedback_id = p_feedback
    AND source_type = 'case_example';

  IF FOUND THEN
    RETURN i;
  END IF;

  i := gen_random_uuid();

  INSERT INTO public.rag_items (
    id, source_type, scope, source_id, feedback_id,
    category, topic, customer_question, human_answer,
    feedback_reason, retrieval_text, embedding,
    approved, active, quality_status, learning_data
  )
  VALUES (
    i,
    'case_example',
    'global',
    p_feedback::text,
    p_feedback,

    COALESCE(
      NULLIF(btrim(c->'corrected_categories'->>0), ''),
      NULLIF(btrim(c->'original_categories'->>0), ''),
      'anderes'
    ),

    'Historischer Lösungsfall',
    c->>'customer_question',
    c->>'human_answer',
    c->>'feedback_reason',
    left(c->>'customer_question', 900),
    NULL,
    false,
    false,
    'pending',

    jsonb_build_object(
      'pipeline_version', 11,
      'case_version', 11,
      'case_revision', 0,
      'case_status', 'staged',
      'publish_state', 'unpublished',
      'target_agent', 'case_retrieval',
      'example_role', 'historical_resolution',
      'factual_authority', false,
      'factual_use_approved', false,
      'verification_status', 'not_applicable',
      'requires_quality_review', true,
      'privacy_checked', false,
      'quality_checked', false,
      'generalization_checked', false,
      'has_explicit_feedback', c->'has_explicit_feedback',
      'feedback_type', ft,
      'original_categories', c->'original_categories',
      'corrected_categories', c->'corrected_categories',
      'confirmed_categories', '[]'::jsonb,
      'origin_digest', md5(c::text)
    )
  );

  INSERT INTO public.learning_case_audit_v11 (
    item_id, revision, action, actor
  )
  VALUES (i, 0, 'stage', session_user);

  RETURN i;
END
$function$;

COMMIT;
