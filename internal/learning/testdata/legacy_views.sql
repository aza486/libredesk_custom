-- Read-only schema contract captured 2026-10-30; no production rows.
CREATE VIEW public.rag_category_learning_view AS  SELECT id,
    embedding,
    retrieval_text AS text,
    jsonb_build_object('rag_item_id', (id)::text, 'source_type', source_type, 'question', customer_question, 'historical_category', category, 'original_categories', COALESCE((learning_data -> 'original_categories'::text), '[]'::jsonb), 'corrected_categories',
        CASE
            WHEN (source_type = 'classification_example'::text) THEN COALESCE((learning_data -> 'corrected_categories'::text), '[]'::jsonb)
            ELSE '[]'::jsonb
        END, 'correction_reason',
        CASE
            WHEN (source_type = 'classification_example'::text) THEN COALESCE((learning_data ->> 'rationale'::text), ''::text)
            ELSE ''::text
        END, 'correction_origin',
        CASE
            WHEN (source_type = 'classification_example'::text) THEN COALESCE((learning_data ->> 'category_correction_origin'::text), 'explicit_reason'::text)
            ELSE 'not_confirmed'::text
        END, 'employee_answer',
        CASE
            WHEN (source_type = 'response_example'::text) THEN "left"(COALESCE(human_answer, ''::text), 350)
            ELSE ''::text
        END, 'evidence_level',
        CASE
            WHEN (source_type = 'classification_example'::text) THEN 'explicit_employee_correction'::text
            ELSE 'historical_ai_label_unverified'::text
        END) AS metadata
   FROM rag_items r
  WHERE ((approved = true) AND (active = true) AND (quality_status = 'approved'::text) AND (embedding IS NOT NULL) AND ((length(btrim(COALESCE(retrieval_text, ''::text))) >= 12) AND (length(btrim(COALESCE(retrieval_text, ''::text))) <= 900)) AND (((source_type = 'classification_example'::text) AND ((learning_data ->> 'target_agent'::text) = 'classifier'::text) AND (jsonb_typeof((learning_data -> 'corrected_categories'::text)) = 'array'::text) AND (jsonb_array_length((learning_data -> 'corrected_categories'::text)) > 0)) OR ((source_type = 'response_example'::text) AND ((learning_data ->> 'target_agent'::text) = 'final_answer'::text) AND ((learning_data ->> 'example_role'::text) = 'style_only'::text) AND (category = ANY (ARRAY['bestellstatus'::text, 'bestellung'::text, 'versandstatus'::text, 'rechnung'::text, 'produktfragen'::text, 'firmenfragen'::text, 'reklamation'::text, 'spam'::text, 'human_needed'::text])))));
CREATE VIEW public.rag_final_style_view AS  SELECT id,
    embedding,
    retrieval_text AS text,
    jsonb_build_object('rag_item_id', (id)::text, 'source_type', source_type, 'example_role', 'style_only', 'factual_authority', false, 'target_agent', 'final_answer', 'question', customer_question, 'human_answer', human_answer, 'category', category) AS metadata
   FROM rag_items r
  WHERE ((source_type = 'response_example'::text) AND (quality_status = 'approved'::text) AND (approved = true) AND (active = true) AND (embedding IS NOT NULL) AND ((learning_data ->> 'target_agent'::text) = 'final_answer'::text) AND ((learning_data ->> 'example_role'::text) = 'style_only'::text) AND ((learning_data ->> 'factual_authority'::text) = 'false'::text) AND (NULLIF(btrim(retrieval_text), ''::text) IS NOT NULL) AND (NULLIF(btrim(human_answer), ''::text) IS NOT NULL));