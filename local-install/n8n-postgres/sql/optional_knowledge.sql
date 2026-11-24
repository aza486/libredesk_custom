-- Reviewed data-free schema. Never run against an existing unmanaged database.
CREATE TABLE public.kd_audit (
    id bigint NOT NULL,
    candidate_id uuid NOT NULL,
    action text NOT NULL,
    revision integer NOT NULL,
    note text,
    snapshot jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE SEQUENCE public.kd_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.kd_audit_id_seq OWNED BY public.kd_audit.id;

CREATE TABLE public.kd_candidates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_id uuid NOT NULL,
    knowledge_type text NOT NULL,
    category text DEFAULT 'firmenfragen'::text NOT NULL,
    topic text DEFAULT ''::text NOT NULL,
    question text NOT NULL,
    answer text NOT NULL,
    evidence_quote text NOT NULL,
    limitations text DEFAULT ''::text NOT NULL,
    review_reason text DEFAULT ''::text NOT NULL,
    explanation text DEFAULT ''::text NOT NULL,
    risk_level text DEFAULT 'medium'::text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    publish_state text DEFAULT 'idle'::text NOT NULL,
    revision integer DEFAULT 1 NOT NULL,
    publish_attempts integer DEFAULT 0 NOT NULL,
    publish_lease_until timestamp with time zone,
    last_publish_error text,
    reviewed_at timestamp with time zone,
    reviewer_note text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    published_at timestamp with time zone,
    fingerprint text NOT NULL,
    CONSTRAINT kd_candidates_answer_check CHECK (((length(answer) >= 10) AND (length(answer) <= 3000))),
    CONSTRAINT kd_candidates_knowledge_type_check CHECK ((knowledge_type = ANY (ARRAY['fact'::text, 'process'::text, 'faq'::text]))),
    CONSTRAINT kd_candidates_publish_state_check CHECK ((publish_state = ANY (ARRAY['idle'::text, 'queued'::text, 'publishing'::text, 'published'::text, 'failed'::text, 'revoked'::text]))),
    CONSTRAINT kd_candidates_question_check CHECK (((length(question) >= 10) AND (length(question) <= 500))),
    CONSTRAINT kd_candidates_risk_level_check CHECK ((risk_level = ANY (ARRAY['low'::text, 'medium'::text, 'high'::text]))),
    CONSTRAINT kd_candidates_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'approved'::text, 'rejected'::text, 'published'::text, 'revoked'::text])))
);

CREATE TABLE public.kd_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_type text NOT NULL,
    title text DEFAULT ''::text NOT NULL,
    source_text text NOT NULL,
    content_hash text NOT NULL,
    status text DEFAULT 'queued'::text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    lease_until timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    processed_at timestamp with time zone,
    CONSTRAINT kd_sources_source_text_check CHECK (((length(source_text) >= 20) AND (length(source_text) <= 20000))),
    CONSTRAINT kd_sources_source_type_check CHECK ((source_type = ANY (ARRAY['email'::text, 'blog'::text, 'website'::text, 'employee'::text, 'other'::text]))),
    CONSTRAINT kd_sources_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'processing'::text, 'done'::text, 'failed'::text])))
);

ALTER TABLE ONLY public.kd_audit ALTER COLUMN id SET DEFAULT nextval('public.kd_audit_id_seq'::regclass);

ALTER TABLE ONLY public.kd_audit
    ADD CONSTRAINT kd_audit_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.kd_candidates
    ADD CONSTRAINT kd_candidates_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.kd_candidates
    ADD CONSTRAINT kd_candidates_source_id_fingerprint_key UNIQUE (source_id, fingerprint);

ALTER TABLE ONLY public.kd_sources
    ADD CONSTRAINT kd_sources_content_hash_key UNIQUE (content_hash);

ALTER TABLE ONLY public.kd_sources
    ADD CONSTRAINT kd_sources_pkey PRIMARY KEY (id);

CREATE INDEX kd_candidates_pub_idx ON public.kd_candidates USING btree (publish_state, created_at);

CREATE INDEX kd_candidates_review_idx ON public.kd_candidates USING btree (status, created_at);

CREATE INDEX kd_sources_queue_idx ON public.kd_sources USING btree (status, created_at);

ALTER TABLE ONLY public.kd_audit
    ADD CONSTRAINT kd_audit_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES public.kd_candidates(id);

ALTER TABLE ONLY public.kd_candidates
    ADD CONSTRAINT kd_candidates_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.kd_sources(id) ON DELETE RESTRICT;
