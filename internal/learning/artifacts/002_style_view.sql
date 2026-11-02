BEGIN;
-- Optional: exposes reviewed style_rule entries through the existing style view.
-- Run after 001_review.sql. No changes to existing response_example semantics.
CREATE OR REPLACE VIEW public.rag_final_style_view AS
SELECT id,embedding,retrieval_text AS text,
 jsonb_build_object('rag_item_id',id::text,'source_type',source_type,
 'example_role','style_only','factual_authority',false,'target_agent','final_answer',
 'question',customer_question,'human_answer',human_answer,'category',category) AS metadata
FROM public.rag_items
WHERE source_type IN ('response_example','style_rule') AND quality_status='approved'
 AND approved=true AND active=true AND embedding IS NOT NULL
 AND learning_data->>'target_agent'='final_answer'
 AND learning_data->>'example_role'='style_only'
 AND learning_data->>'factual_authority'='false'
 AND NULLIF(BTRIM(retrieval_text),'') IS NOT NULL
 AND NULLIF(BTRIM(human_answer),'') IS NOT NULL;
-- A separate fact view avoids treating factual knowledge as a response template.
CREATE OR REPLACE VIEW public.rag_verified_fact_view AS
SELECT id,embedding,retrieval_text AS text,
 jsonb_build_object('rag_item_id',id::text,'source_type',source_type,
 'fact_statement',learning_data->>'fact_statement','evidence',learning_data->>'evidence_quote',
 'limitations',learning_data->>'limitations','verification_status','human_verified',
 'factual_use_approved',true) AS metadata
FROM public.rag_items WHERE source_type='employee_fact'
 AND approved=true AND active=true AND quality_status='approved' AND embedding IS NOT NULL
 AND learning_data->>'verification_status'='human_verified'
 AND learning_data->>'factual_use_approved'='true';

COMMIT;
