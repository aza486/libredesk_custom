-- Explicit pg-n8n installation after 005_case_v11.sql, never on application startup.
-- Run as the trusted owner of the base tables. No roles or role grants are created.
BEGIN;
CREATE OR REPLACE VIEW public.learning_case_publisher_candidates_v11
 WITH (security_barrier=true) AS
SELECT r.id,(r.learning_data->>'case_revision')::bigint AS revision,
 r.topic,r.customer_question,r.human_answer,r.feedback_reason,r.retrieval_text,r.updated_at
FROM public.rag_items r
WHERE r.source_type='case_example' AND r.quality_status='approved'
 AND r.active=false AND r.approved=false AND r.embedding IS NULL
 AND r.learning_data @> '{"case_version":11,"pipeline_version":11,"case_status":"approved","publish_state":"unpublished","privacy_checked":true,"quality_checked":true,"generalization_checked":true,"requires_quality_review":true,"factual_authority":false,"factual_use_approved":false,"target_agent":"case_retrieval","example_role":"historical_resolution"}'
 AND EXISTS(SELECT 1 FROM public.ai_feedback f WHERE f.id=r.feedback_id
  AND f.feedback_data->'learning_case_v11'->>'status'='staged'
  AND md5((f.feedback_data->'learning_case_v11')::text)=r.learning_data->>'origin_digest');
REVOKE ALL ON public.learning_case_publisher_candidates_v11 FROM PUBLIC;
COMMIT;
