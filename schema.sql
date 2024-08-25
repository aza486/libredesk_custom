-- public.attachments definition

-- Drop table

-- DROP TABLE public.attachments;

CREATE TABLE public.attachments (
	id int8 DEFAULT nextval('media_id_seq'::regclass) NOT NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NULL,
	created_at timestamp DEFAULT now() NULL,
	store varchar(10) DEFAULT ''::text NOT NULL,
	filename varchar(140) NOT NULL,
	content_type varchar(140) NOT NULL,
	message_id int8 NULL,
	"size" varchar(10) NULL,
	content_disposition varchar(50) NULL,
	CONSTRAINT media_pkey PRIMARY KEY (id)
);


-- public.automation_rules definition

-- Drop table

-- DROP TABLE public.automation_rules;

CREATE TABLE public.automation_rules (
	id int4 DEFAULT nextval('rules_id_seq'::regclass) NOT NULL,
	"name" varchar(255) NOT NULL,
	description text NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	"type" varchar NOT NULL,
	rules jsonb NULL,
	updated_at timestamp DEFAULT now() NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	CONSTRAINT rules_pkey PRIMARY KEY (id)
);


-- public.canned_responses definition

-- Drop table

-- DROP TABLE public.canned_responses;

CREATE TABLE public.canned_responses (
	id serial4 NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	title text NOT NULL,
	"content" text NOT NULL,
	CONSTRAINT canned_responses_pkey PRIMARY KEY (id)
);


-- public.contacts definition

-- Drop table

-- DROP TABLE public.contacts;

CREATE TABLE public.contacts (
	id bigserial NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	first_name text NULL,
	last_name text NULL,
	email varchar NULL,
	phone_number text NULL,
	avatar_url text NULL,
	inbox_id int4 NULL,
	source_id text NULL,
	CONSTRAINT contacts_pkey PRIMARY KEY (id)
);


-- public.conversation_participants definition

-- Drop table

-- DROP TABLE public.conversation_participants;

CREATE TABLE public.conversation_participants (
	id bigserial NOT NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	user_id int8 NULL,
	conversation_id int8 NULL,
	CONSTRAINT conversation_participants_pkey PRIMARY KEY (id),
	CONSTRAINT conversation_participants_unique UNIQUE (conversation_id, user_id)
);


-- public.file_upload_providers definition

-- Drop table

-- DROP TABLE public.file_upload_providers;

CREATE TABLE public.file_upload_providers (
	id serial4 NOT NULL,
	provider_name text NOT NULL,
	region text NULL,
	access_key text NULL,
	access_secret text NULL,
	bucket_name text NULL,
	bucket_type text NULL,
	bucket_path text NULL,
	upload_expiry interval NULL,
	s3_backend_url text NULL,
	custom_public_url text NULL,
	upload_path text NULL,
	upload_uri text NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT file_upload_providers_pkey PRIMARY KEY (id)
);


-- public.inboxes definition

-- Drop table

-- DROP TABLE public.inboxes;

CREATE TABLE public.inboxes (
	id serial4 NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	channel public."channels" NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"name" varchar(140) NOT NULL,
	"from" varchar(200) NULL,
	assign_to_team int4 NULL,
	soft_delete bool DEFAULT false NOT NULL,
	CONSTRAINT inboxes_pkey PRIMARY KEY (id)
);


-- public.media definition

-- Drop table

-- DROP TABLE public.media;

CREATE TABLE public.media (
	id serial4 NOT NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	store public."media_store" NOT NULL,
	filename text NOT NULL,
	content_type text NOT NULL,
	model_id int4 NULL,
	model_type text NULL,
	"size" int4 NULL,
	meta jsonb DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT media_pkey1 PRIMARY KEY (id)
);


-- public.oidc definition

-- Drop table

-- DROP TABLE public.oidc;

CREATE TABLE public.oidc (
	id int4 DEFAULT nextval('social_login_id_seq'::regclass) NOT NULL,
	provider_url text NOT NULL,
	client_id text NOT NULL,
	client_secret text NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	created_at timestamp DEFAULT now() NOT NULL,
	updated_at timestamp DEFAULT now() NOT NULL,
	provider varchar NULL,
	"name" text NULL,
	CONSTRAINT social_login_pkey PRIMARY KEY (id)
);


-- public.priority definition

-- Drop table

-- DROP TABLE public.priority;

CREATE TABLE public.priority (
	id serial4 NOT NULL,
	"name" text NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT priority_pkey PRIMARY KEY (id),
	CONSTRAINT priority_priority_name_key UNIQUE (name)
);


-- public.roles definition

-- Drop table

-- DROP TABLE public.roles;

CREATE TABLE public.roles (
	id serial4 NOT NULL,
	permissions _text DEFAULT '{}'::text[] NOT NULL,
	"name" text NULL,
	description text NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT roles_pkey PRIMARY KEY (id)
);


-- public.settings definition

-- Drop table

-- DROP TABLE public.settings;

CREATE TABLE public.settings (
	"key" text NOT NULL,
	value jsonb DEFAULT '{}'::jsonb NOT NULL,
	updated_at timestamptz DEFAULT now() NULL,
	CONSTRAINT settings_key_key UNIQUE (key)
);
CREATE INDEX idx_settings_key ON public.settings USING btree (key);


-- public.status definition

-- Drop table

-- DROP TABLE public.status;

CREATE TABLE public.status (
	id serial4 NOT NULL,
	"name" text NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT status_pkey PRIMARY KEY (id),
	CONSTRAINT status_status_name_key UNIQUE (name)
);


-- public.tags definition

-- Drop table

-- DROP TABLE public.tags;

CREATE TABLE public.tags (
	id bigserial NOT NULL,
	"name" text NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT tags_pkey PRIMARY KEY (id),
	CONSTRAINT tags_tag_name_key UNIQUE (name)
);


-- public.team_members definition

-- Drop table

-- DROP TABLE public.team_members;

CREATE TABLE public.team_members (
	id serial4 NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	team_id int4 NOT NULL,
	user_id int4 NOT NULL,
	CONSTRAINT team_members_pkey PRIMARY KEY (id),
	CONSTRAINT unique_team_user UNIQUE (team_id, user_id)
);


-- public.teams definition

-- Drop table

-- DROP TABLE public.teams;

CREATE TABLE public.teams (
	id serial4 NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	"name" varchar(140) NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	auto_assign_conversations bool DEFAULT false NOT NULL,
	CONSTRAINT teams_pkey PRIMARY KEY (id),
	CONSTRAINT teams_unique UNIQUE (name)
);


-- public.templates definition

-- Drop table

-- DROP TABLE public.templates;

CREATE TABLE public.templates (
	id serial4 NOT NULL,
	body text NOT NULL,
	is_default bool DEFAULT false NOT NULL,
	created_at timestamptz DEFAULT now() NULL,
	updated_at timestamptz DEFAULT now() NULL,
	"name" text NULL,
	CONSTRAINT email_templates_pkey PRIMARY KEY (id)
);
CREATE UNIQUE INDEX email_templates_is_default_idx ON public.templates USING btree (is_default) WHERE (is_default = true);


-- public.uploads definition

-- Drop table

-- DROP TABLE public.uploads;

CREATE TABLE public.uploads (
	id uuid DEFAULT gen_random_uuid() NOT NULL,
	filename text NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT uploads_pkey PRIMARY KEY (id)
);


-- public.users definition

-- Drop table

-- DROP TABLE public.users;

CREATE TABLE public.users (
	id int4 DEFAULT nextval('agents_id_seq'::regclass) NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	email varchar(255) NOT NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	first_name varchar(100) NOT NULL,
	last_name varchar(100) NULL,
	"password" varchar(150) NULL,
	disabled bool DEFAULT false NOT NULL,
	avatar_url text NULL,
	roles _text DEFAULT '{}'::text[] NOT NULL,
	CONSTRAINT agents_pkey PRIMARY KEY (id),
	CONSTRAINT users_email_unique UNIQUE (email)
);


-- public.contact_methods definition

-- Drop table

-- DROP TABLE public.contact_methods;

CREATE TABLE public.contact_methods (
	id bigserial NOT NULL,
	contact_id int8 NOT NULL,
	"source" text NOT NULL,
	source_id text NOT NULL,
	inbox_id int4 NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT contact_methods_pkey PRIMARY KEY (id),
	CONSTRAINT unique_contact_method UNIQUE (source, source_id),
	CONSTRAINT fk_contact FOREIGN KEY (contact_id) REFERENCES public.contacts(id) ON DELETE CASCADE
);


-- public.conversations definition

-- Drop table

-- DROP TABLE public.conversations;

CREATE TABLE public.conversations (
	id bigserial NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	updated_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	reference_number text NOT NULL,
	closed_at timestamp NULL,
	contact_id int8 NOT NULL,
	assigned_user_id int8 NULL,
	assigned_team_id int8 NULL,
	resolved_at timestamp NULL,
	inbox_id int4 NOT NULL,
	meta jsonb DEFAULT '{}'::json NOT NULL,
	assignee_last_seen_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	first_reply_at timestamp NULL,
	status_id int8 DEFAULT 1 NOT NULL,
	priority_id int8 NULL,
	CONSTRAINT messages_pkey PRIMARY KEY (id),
	CONSTRAINT conversations_status_id_fkey FOREIGN KEY (status_id) REFERENCES public.status(id)
);


-- public.messages definition

-- Drop table

-- DROP TABLE public.messages;

CREATE TABLE public.messages (
	id bigserial NOT NULL,
	updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	"type" text NOT NULL,
	status text NULL,
	conversation_id bigserial NOT NULL,
	"content" text NULL,
	sender_id int4 NULL,
	private bool NULL,
	content_type varchar(50) DEFAULT false NOT NULL,
	source_id text NULL,
	meta jsonb DEFAULT '{}'::jsonb NULL,
	inbox_id int4 NULL,
	sender_type varchar NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
	CONSTRAINT messages__pkey PRIMARY KEY (id),
	CONSTRAINT messages_unique UNIQUE (source_id),
	CONSTRAINT fk_conversation_id FOREIGN KEY (conversation_id) REFERENCES public.conversations(id)
);


-- public.conversation_tags definition

-- Drop table

-- DROP TABLE public.conversation_tags;

CREATE TABLE public.conversation_tags (
	id bigserial NOT NULL,
	conversation_id int8 DEFAULT nextval('conversation_tags_converastion_id_seq'::regclass) NOT NULL,
	tag_id bigserial NOT NULL,
	created_at timestamp DEFAULT CURRENT_TIMESTAMP NULL,
	CONSTRAINT conversation_tags_unique UNIQUE (conversation_id, tag_id),
	CONSTRAINT message_tags_pkey PRIMARY KEY (id),
	CONSTRAINT message_tags_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.conversations(id),
	CONSTRAINT message_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id)
);