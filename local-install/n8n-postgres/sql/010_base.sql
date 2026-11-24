-- Reviewed data-free schema. Never run against an existing unmanaged database.
CREATE TABLE public.ai_feedback (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    mail_event_message_uuid text NOT NULL,
    agent_id text NOT NULL,
    feedback_reason text,
    human_answer text,
    feedback_type text,
    quality_status text DEFAULT 'pending'::text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    outgoing_message_uuid text,
    feedback_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    learning_status text DEFAULT 'new'::text NOT NULL,
    learning_processed_at timestamp without time zone,
    CONSTRAINT ai_feedback_data_object_check CHECK ((jsonb_typeof(feedback_data) = 'object'::text)),
    CONSTRAINT ai_feedback_learning_status_check CHECK ((learning_status = ANY (ARRAY['new'::text, 'processing'::text, 'prepared'::text, 'published'::text, 'skipped'::text, 'failed'::text]))),
    CONSTRAINT ai_feedback_quality_status_check CHECK ((quality_status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text])))
);

CREATE TABLE public.agent_style_profiles (
    agent_id text NOT NULL,
    durchschnittslaenge integer,
    emoji_nutzung text,
    foermlichkeit text,
    direktheit text,
    grussformel text,
    verabschiedung text,
    beispiele jsonb,
    sample_size integer,
    updated_at timestamp without time zone DEFAULT now()
);

CREATE TABLE public.cases (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    conversation_uuid text NOT NULL,
    conversation_id bigint,
    reference_number text,
    customer_email text,
    customer_name text,
    inbox_mail text,
    status text DEFAULT 'open'::text,
    category text,
    confidence double precision,
    summary text,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    last_message_at timestamp without time zone,
    subject text,
    mail_type text DEFAULT 'customer'::text NOT NULL,
    CONSTRAINT cases_mail_type_check CHECK ((mail_type = ANY (ARRAY['customer'::text, 'private_mail'::text])))
);

CREATE TABLE public.company_knowledge (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    category text NOT NULL,
    question text NOT NULL,
    answer text NOT NULL,
    embedding public.vector(1536) NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);

CREATE TABLE public.rag_items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_type text NOT NULL,
    scope text DEFAULT 'global'::text NOT NULL,
    agent_id text,
    category text NOT NULL,
    subcategory text,
    topic text,
    customer_question text,
    ai_answer text,
    human_answer text,
    feedback_reason text,
    retrieval_text text NOT NULL,
    embedding public.vector(1536),
    quality_status text DEFAULT 'pending'::text NOT NULL,
    approved boolean DEFAULT false NOT NULL,
    active boolean DEFAULT true NOT NULL,
    source_id text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feedback_id uuid,
    learning_data jsonb DEFAULT '{}'::jsonb NOT NULL
);

CREATE TABLE public.learning_embedding_stage (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    content text,
    metadata jsonb,
    embedding public.vector(1536)
);

CREATE TABLE public.libredesk_sent_receipts_v2 (
    outgoing_message_uuid text NOT NULL,
    source_message_uuid text NOT NULL,
    conversation_uuid text NOT NULL,
    response_source text,
    agent_id text,
    expected_incoming_version text NOT NULL,
    sent_at timestamp with time zone NOT NULL,
    payload_hash text NOT NULL,
    resolve_state text DEFAULT 'pending'::text NOT NULL,
    received_at timestamp with time zone DEFAULT now() NOT NULL,
    finalized_at timestamp with time zone,
    CONSTRAINT libredesk_sent_receipts_v2_resolve_state_check CHECK ((resolve_state = ANY (ARRAY['pending'::text, 'resolved'::text, 'superseded'::text, 'invalid_completion'::text]))),
    CONSTRAINT libredesk_sent_receipts_v2_response_source_check CHECK ((response_source = ANY (ARRAY['ai'::text, 'ai_adapted'::text, 'human'::text])))
);

CREATE TABLE public.mail_events (
    conversation_uuid text NOT NULL,
    conversation_id bigint,
    message_uuid text NOT NULL,
    message_id bigint,
    direction text,
    sender_email text,
    subject text,
    raw_text text,
    clean_text text,
    suggested_message text,
    sent_message text,
    status text DEFAULT 'new'::text,
    processing_stage text DEFAULT 'received'::text,
    received_at timestamp without time zone,
    status_updated_at timestamp without time zone,
    sent_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now(),
    sender_salutation text,
    sender_firstname text,
    sender_lastname text,
    order_ids text[],
    product_queries text[],
    confidence real,
    subject_clean text,
    categories text[] DEFAULT '{}'::text[],
    can_answer boolean,
    needs_human boolean,
    fallback_used boolean DEFAULT false NOT NULL,
    recommendation text,
    is_form boolean DEFAULT false NOT NULL,
    firma text,
    strasse text,
    hausnummer text,
    plz text,
    stadt text,
    telefonnummer text,
    form_category text,
    form_sent_at timestamp with time zone,
    form_message text,
    customer_number text,
    agent_id text,
    faq_candidate boolean DEFAULT false,
    attuned_message text,
    mail_type text DEFAULT 'customer'::text NOT NULL,
    response_source text,
    CONSTRAINT mail_events_mail_type_check CHECK ((mail_type = ANY (ARRAY['customer'::text, 'private_mail'::text]))),
    CONSTRAINT mail_events_response_source_check CHECK (((response_source IS NULL) OR (response_source = ANY (ARRAY['ai'::text, 'ai_adapted'::text, 'human'::text]))))
);

ALTER TABLE ONLY public.agent_style_profiles
    ADD CONSTRAINT agent_style_profiles_pkey PRIMARY KEY (agent_id);

ALTER TABLE ONLY public.ai_feedback
    ADD CONSTRAINT ai_feedback_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.cases
    ADD CONSTRAINT cases_conversation_uuid_key UNIQUE (conversation_uuid);

ALTER TABLE ONLY public.cases
    ADD CONSTRAINT cases_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.company_knowledge
    ADD CONSTRAINT company_knowledge_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.learning_embedding_stage
    ADD CONSTRAINT learning_embedding_stage_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.libredesk_sent_receipts_v2
    ADD CONSTRAINT libredesk_sent_receipts_v2_pkey PRIMARY KEY (outgoing_message_uuid);

ALTER TABLE ONLY public.mail_events
    ADD CONSTRAINT mail_events_pkey PRIMARY KEY (message_uuid);

ALTER TABLE ONLY public.rag_items
    ADD CONSTRAINT rag_items_pkey PRIMARY KEY (id);

CREATE INDEX ai_feedback_agent_idx ON public.ai_feedback USING btree (agent_id);

CREATE INDEX ai_feedback_created_at_idx ON public.ai_feedback USING btree (created_at);

CREATE INDEX ai_feedback_learning_status_idx ON public.ai_feedback USING btree (learning_status, created_at);

CREATE INDEX ai_feedback_mail_event_idx ON public.ai_feedback USING btree (mail_event_message_uuid);

CREATE UNIQUE INDEX ai_feedback_outgoing_message_uuid_uidx ON public.ai_feedback USING btree (outgoing_message_uuid) WHERE (outgoing_message_uuid IS NOT NULL);

CREATE INDEX ai_feedback_quality_status_idx ON public.ai_feedback USING btree (quality_status);

CREATE INDEX company_knowledge_embedding_idx ON public.company_knowledge USING hnsw (embedding public.vector_cosine_ops);

CREATE INDEX idx_cases_conversation_uuid ON public.cases USING btree (conversation_uuid);

CREATE INDEX idx_ld_sent_v2_source_order ON public.libredesk_sent_receipts_v2 USING btree (source_message_uuid, sent_at DESC, outgoing_message_uuid DESC);

CREATE INDEX idx_mail_events_conversation_uuid ON public.mail_events USING btree (conversation_uuid);

CREATE INDEX idx_mail_events_status ON public.mail_events USING btree (status);

CREATE INDEX learning_embedding_stage_rag_item_idx ON public.learning_embedding_stage USING btree (((metadata ->> 'rag_item_id'::text)));

CREATE INDEX rag_items_active_approved_idx ON public.rag_items USING btree (active, approved);

CREATE INDEX rag_items_agent_id_idx ON public.rag_items USING btree (agent_id);

CREATE INDEX rag_items_category_idx ON public.rag_items USING btree (category);

CREATE INDEX rag_items_embedding_idx ON public.rag_items USING hnsw (embedding public.vector_cosine_ops);

CREATE INDEX rag_items_feedback_idx ON public.rag_items USING btree (feedback_id);

CREATE INDEX rag_items_quality_status_idx ON public.rag_items USING btree (quality_status);

CREATE INDEX rag_items_scope_idx ON public.rag_items USING btree (scope);

CREATE INDEX rag_items_source_type_idx ON public.rag_items USING btree (source_type);

ALTER TABLE ONLY public.ai_feedback
    ADD CONSTRAINT ai_feedback_mail_event_fk FOREIGN KEY (mail_event_message_uuid) REFERENCES public.mail_events(message_uuid) ON DELETE RESTRICT;

ALTER TABLE ONLY public.rag_items
    ADD CONSTRAINT rag_items_feedback_fk FOREIGN KEY (feedback_id) REFERENCES public.ai_feedback(id) ON DELETE SET NULL;
