CREATE EXTENSION IF NOT EXISTS pg_trgm;

DROP TYPE IF EXISTS "channels" CASCADE; CREATE TYPE "channels" AS ENUM ('email');
DROP TYPE IF EXISTS "media_store" CASCADE; CREATE TYPE "media_store" AS ENUM ('s3', 'fs');
DROP TYPE IF EXISTS "message_type" CASCADE; CREATE TYPE "message_type" AS ENUM ('incoming','outgoing','activity');
DROP TYPE IF EXISTS "message_sender_type" CASCADE; CREATE TYPE "message_sender_type" AS ENUM ('user','contact');
DROP TYPE IF EXISTS "message_status" CASCADE; CREATE TYPE "message_status" AS ENUM ('received','sent','failed','pending');
DROP TYPE IF EXISTS "content_type" CASCADE; CREATE TYPE "content_type" AS ENUM ('text','html');
DROP TYPE IF EXISTS "sla_status" CASCADE; CREATE TYPE "sla_status" AS ENUM ('active','missed');
DROP TYPE IF EXISTS "conversation_assignment_type" CASCADE; CREATE TYPE "conversation_assignment_type" AS ENUM ('Round robin','Manual');
DROP TYPE IF EXISTS "sla_type" CASCADE; CREATE TYPE "sla_type" AS ENUM ('first_response','resolution');
DROP TYPE IF EXISTS "template_type" CASCADE; CREATE TYPE "template_type" AS ENUM ('email_outgoing', 'email_notification');
DROP TYPE IF EXISTS "user_type" CASCADE; CREATE TYPE "user_type" AS ENUM ('agent', 'contact');
DROP TYPE IF EXISTS "ai_provider" CASCADE; CREATE TYPE "ai_provider" AS ENUM ('openai');
DROP TYPE IF EXISTS "automation_execution_mode" CASCADE; CREATE TYPE "automation_execution_mode" AS ENUM ('all', 'first_match');
DROP TYPE IF EXISTS "macro_visibility" CASCADE; CREATE TYPE "visibility" AS ENUM ('all', 'team', 'user');

DROP TABLE IF EXISTS conversation_slas CASCADE;
CREATE TABLE conversation_slas (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
	updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
	conversation_id BIGINT NOT NULL REFERENCES conversations(id),
	sla_policy_id INT NOT NULL REFERENCES sla_policies(id),
	sla_type sla_type NOT NULL,
	due_at TIMESTAMPTZ NULL,
	met_at TIMESTAMPTZ NULL,
	breached_at TIMESTAMPTZ NULL,
	CONSTRAINT constraint_conversation_slas_unique UNIQUE (sla_policy_id, conversation_id, sla_type)
);

DROP TABLE IF EXISTS teams CASCADE;
CREATE TABLE teams (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	emoji TEXT NULL,
	disabled bool DEFAULT false NOT NULL,
	conversation_assignment_type conversation_assignment_type NOT NULL,
	business_hours_id INT REFERENCES business_hours(id) ON DELETE SET NULL ON UPDATE CASCADE NOT NULL,
	timezone TEXT NULL,
	CONSTRAINT constraint_teams_on_emoji CHECK (length(emoji) <= 1),
	CONSTRAINT constraint_teams_on_name CHECK (length("name") <= 140),
	CONSTRAINT constraint_teams_on_timezone CHECK (length(timezone) <= 50),
	CONSTRAINT constraint_teams_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    type user_type NOT NULL,
    deleted_at TIMESTAMPTZ NULL,
    disabled BOOL DEFAULT FALSE NOT NULL,
    email TEXT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NULL,
    phone_number TEXT NULL,
    country TEXT NULL,
    "password" VARCHAR(150) NULL,
    avatar_url TEXT NULL,
    roles TEXT[] DEFAULT '{}'::TEXT[] NULL,
    reset_password_token TEXT NULL,
    reset_password_token_expiry TIMESTAMPTZ NULL,
    CONSTRAINT constraint_users_on_email_and_type_unique UNIQUE (email, type),
    CONSTRAINT constraint_users_on_country CHECK (LENGTH(country) <= 140),
    CONSTRAINT constraint_users_on_phone_number CHECK (LENGTH(phone_number) <= 20),
    CONSTRAINT constraint_users_on_email_length CHECK (LENGTH(email) <= 320),
    CONSTRAINT constraint_users_on_first_name CHECK (LENGTH(first_name) <= 140),
    CONSTRAINT constraint_users_on_last_name CHECK (LENGTH(last_name) <= 140)
);

DROP TABLE IF EXISTS contact_channels CASCADE;
CREATE TABLE contact_channels (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	contact_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
	inbox_id INT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE,
	identifier TEXT NOT NULL,
	CONSTRAINT constraint_contact_channels_on_identifier CHECK (length(identifier) <= 1000),
	CONSTRAINT constraint_contact_channels_on_inbox_id_and_contact_id_unique UNIQUE (inbox_id, contact_id)
);

DROP TABLE IF EXISTS conversation_statuses CASCADE;
CREATE TABLE conversation_statuses (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	CONSTRAINT constraint_status_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS conversation_priorities CASCADE;
CREATE TABLE conversation_priorities (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	CONSTRAINT constraint_priority_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS conversations CASCADE;
CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "uuid" UUID DEFAULT gen_random_uuid() NOT NULL,
	reference_number BIGSERIAL UNIQUE,
    contact_id BIGINT NOT NULL,
	contact_channel_id INT REFERENCES contact_channels(id) ON DELETE SET NULL ON UPDATE CASCADE,
    assigned_user_id INT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    assigned_team_id INT REFERENCES teams(id) ON DELETE SET NULL ON UPDATE CASCADE,
    inbox_id INT NOT NULL,
    meta JSONB DEFAULT '{}'::jsonb NOT NULL,
	custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL,
    assignee_last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    first_reply_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,
    status_id INT REFERENCES conversation_statuses(id),
    priority_id INT REFERENCES conversation_priorities(id),
	sla_policy_id INT REFERENCES sla_policies(id) ON DELETE SET NULL ON UPDATE CASCADE,
	"subject" TEXT NULL,
	last_message_at TIMESTAMPTZ NULL,
	last_message TEXT NULL,
	next_sla_deadline_at TIMESTAMPTZ NULL,
	snoozed_until TIMESTAMPTZ NULL
);

DROP TABLE IF EXISTS conversation_messages CASCADE;
CREATE TABLE conversation_messages (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "uuid" UUID DEFAULT gen_random_uuid() NOT NULL,
    "type" message_type NOT NULL,
    status message_status NOT NULL,
    private BOOL NULL,
    conversation_id BIGSERIAL REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
    content_type content_type NULL,
    "content" TEXT NULL,
	text_content TEXT NULL,
    source_id TEXT NULL,
 	sender_id INT REFERENCES users(id) NULL,
    sender_type message_sender_type NOT NULL,
    meta JSONB DEFAULT '{}'::JSONB NULL
);
CREATE INDEX idx_conversation_messages_text_content ON conversation_messages 
USING GIN (text_content gin_trgm_ops);

DROP TABLE IF EXISTS automation_rules CASCADE;
CREATE TABLE automation_rules (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "name" TEXT NOT NULL,
    description TEXT NULL,
    "type" VARCHAR NOT NULL,
    rules JSONB NULL,
    events TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
    disabled BOOL DEFAULT false NOT NULL,
	weight INT DEFAULT 0 NOT NULL,
	execution_mode automation_execution_mode DEFAULT 'all' NOT NULL,
    CONSTRAINT constraint_automation_rules_on_name CHECK (length("name") <= 140),
    CONSTRAINT constraint_automation_rules_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS macros CASCADE;
CREATE TABLE macros (
   id SERIAL PRIMARY KEY,
   created_at TIMESTAMPTZ DEFAULT NOW(),
   updated_at TIMESTAMPTZ DEFAULT NOW(),
   title TEXT NOT NULL,
   actions JSONB DEFAULT '{}'::jsonb NOT NULL,
   visibility macro_visibility NOT NULL,
   message_content TEXT NOT NULL,
   user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
   team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
   usage_count INT DEFAULT 0 NOT NULL,
   CONSTRAINT title_length CHECK (length(title) <= 255),
   CONSTRAINT message_content_length CHECK (length(message_content) <= 1000)
);

DROP TABLE IF EXISTS conversation_participants CASCADE;
CREATE TABLE conversation_participants (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT constraint_conversation_participants_conversation_id_and_user_id_unique UNIQUE (conversation_id, user_id)
);

DROP TABLE IF EXISTS inboxes CASCADE;
CREATE TABLE inboxes (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	deleted_at TIMESTAMPTZ NULL,
	channel "channels" NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	csat_enabled bool DEFAULT false NOT NULL,
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"name" VARCHAR(140) NOT NULL,
	"from" VARCHAR(500) NULL,
);

DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	store "media_store" NOT NULL,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	model_id INT NULL,
	model_type TEXT NULL,
	disposition VARCHAR(50) NULL,
	content_id TEXT NULL,
	"size" INT NULL,
	meta jsonb DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT constraint_media_on_filename CHECK (length(filename) <= 1000),
	CONSTRAINT constraint_media_on_content_id CHECK (length(content_id) <= 300)
);

DROP TABLE IF EXISTS oidc CASCADE;
CREATE TABLE oidc (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	provider_url TEXT NOT NULL,
	client_id TEXT NOT NULL,
	client_secret TEXT NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	provider VARCHAR NULL,
	"name" TEXT NULL
);

DROP TABLE IF EXISTS roles CASCADE;
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    permissions TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
    "name" TEXT NULL,
    description TEXT NULL
);

DROP TABLE IF EXISTS settings CASCADE;
CREATE TABLE settings (
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"key" TEXT NOT NULL,
	value jsonb DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT settings_key_key UNIQUE ("key")
);
CREATE INDEX index_settings_on_key ON settings USING btree ("key");

DROP TABLE IF EXISTS tags CASCADE;
CREATE TABLE tags (
	id SERIAL PRIMARY KEY,
	"name" TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	CONSTRAINT constraint_tags_on_name_unique UNIQUE ("name"),
	CONSTRAINT constraint_tags_on_name CHECK (length("name") <= 140)
);

DROP TABLE IF EXISTS team_members CASCADE;
CREATE TABLE team_members (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
	user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
	emoji TEXT NULL,
	CONSTRAINT constraint_team_members_on_team_id_and_user_id_unique UNIQUE (team_id, user_id),
	CONSTRAINT constraint_team_members_on_emoji CHECK (length(emoji) <= 1)
);

DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	type template_type NOT NULL,
	body TEXT NOT NULL,
	is_default bool DEFAULT false NOT NULL,
	"name" TEXT NOT NULL,
	subject TEXT NULL,
	is_builtin bool DEFAULT false NOT NULL,
	CONSTRAINT constraint_templates_on_name CHECK (length("name") <= 140),
	CONSTRAINT constraint_templates_on_subject CHECK (length(subject) <= 1000)
);
CREATE UNIQUE INDEX unique_index_templates_on_is_default_when_is_default_is_true ON templates USING btree (is_default)
WHERE (is_default = true);

DROP TABLE IF EXISTS conversation_tags CASCADE;
CREATE TABLE conversation_tags (
	id BIGSERIAL PRIMARY KEY,
	tag_id INT REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE,
	conversation_id BIGSERIAL REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	CONSTRAINT constraint_conversation_tags_on_conversation_id_and_tag_id_unique UNIQUE (conversation_id, tag_id)
);

DROP TABLE IF EXISTS csat_responses CASCADE;
CREATE TABLE csat_responses (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
	uuid UUID DEFAULT gen_random_uuid(),
    conversation_id BIGSERIAL REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
    assigned_agent_id INT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    rating INT DEFAULT 0 NOT NULL,
    feedback TEXT NULL,
    response_timestamp TIMESTAMPTZ NULL,
    CONSTRAINT constraint_csat_responses_on_rating CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT constraint_csat_responses_on_feedback CHECK (length(feedback) <= 1000),
);

DROP TABLE IF EXISTS business_hours CASCADE;
CREATE TABLE business_hours (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
	name TEXT NOT NULL,
	description TEXT NULL,
	is_always_open BOOL DEFAULT false NOT NULL,
	hours JSONB NOT NULL,
	holidays JSONB DEFAULT '{}'::jsonb NOT NULL,
	CONSTRAINT constraint_business_hours_on_name CHECK (length(name) <= 140)
);

DROP TABLE IF EXISTS sla_policies CASCADE;
CREATE TABLE sla_policies (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	name TEXT NOT NULL,
	description TEXT NULL,
	first_response_time TEXT NOT NULL,
	resolution_time TEXT NOT NULL,
	CONSTRAINT constraint_sla_policies_on_name CHECK (length(name) <= 140)
);

DROP TABLE IF EXISTS views CASCADE;
CREATE TABLE views (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	inbox_type TEXT NOT NULL,
    name TEXT NOT NULL,
    filters JSONB NOT NULL,
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	CONSTRAINT constraint_views_on_name CHECK (length(name) <= 140),
	CONSTRAINT constraint_views_on_inbox_type CHECK (length(inbox_type) <= 140)
);

DROP TABLE IF EXISTS ai_providers CASCADE;
CREATE TABLE ai_providers (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	name TEXT NOT NULL,
	provider ai_provider NOT NULL,
	config JSONB NOT NULL DEFAULT '{}',
	is_default BOOLEAN NOT NULL DEFAULT FALSE,
	CONSTRAINT constraint_ai_providers_on_name CHECK (length(name) <= 140)
);
CREATE UNIQUE INDEX unique_index_ai_providers_on_is_default_when_is_default_is_true ON ai_providers USING btree (is_default)
WHERE (is_default = true);
CREATE INDEX index_ai_providers_on_name ON ai_providers USING btree (name);

CREATE TABLE ai_prompts (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	title TEXT NOT NULL,
    key TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
	CONSTRAINT constraint_prompts_on_title CHECK (length(title) <= 140),
    CONSTRAINT constraint_prompts_on_key CHECK (length(key) <= 140)
);
CREATE INDEX index_ai_prompts_on_key ON ai_prompts USING btree (key);

INSERT INTO ai_providers
("name", provider, config, is_default)
VALUES('openai', 'openai', '{"api_key": ""}'::jsonb, true);

-- Default AI prompts
-- TODO: Narrow down the list of prompts.
INSERT INTO public.ai_prompts ("key", "content", title)
VALUES
('make_friendly', 'Modify the text to make it more friendly and approachable.', 'Make Friendly'),
('make_concise', 'Simplify the text to make it more concise and to the point.', 'Make Concise'),
('add_empathy', 'Add empathy to the text while retaining the original meaning.', 'Add Empathy'),
('adjust_positive_tone', 'Adjust the tone of the text to make it sound more positive and reassuring.', 'Adjust Positive Tone'),
('provide_clear_explanation', 'Rewrite the text to provide a clearer explanation of the issue or solution.', 'Provide Clear Explanation'),
('add_urgency', 'Modify the text to convey a sense of urgency without being rude.', 'Add Urgency'),
('make_actionable', 'Rephrase the text to clearly specify the next steps for the customer.', 'Make Actionable'),
('adjust_neutral_tone', 'Adjust the tone to make it neutral and unbiased.', 'Adjust Neutral Tone'),
('make_professional', 'Rephrase the text to make it sound more formal and professional and to the point.', 'Make Professional');

-- Default settings
INSERT INTO settings ("key", value)
VALUES
    ('app.lang', '"en"'::jsonb),
    ('app.root_url', '"http://localhost:9000"'::jsonb),
    ('app.logo_url', '"http://localhost:9000/logo.png"'::jsonb),
    ('app.site_name', '"My helpdesk"'::jsonb),
    ('app.favicon_url', '"http://localhost:9000/favicon.ico"'::jsonb),
    ('app.max_file_upload_size', '20'::jsonb),
    ('app.allowed_file_upload_extensions', '["*"]'::jsonb),
	('app.timezone', '"Asia/Calcutta"'::jsonb),
	('app.business_hours_id', '""'::jsonb),
    ('notification.email.username', '"admin@yourcompany.com"'::jsonb),
    ('notification.email.host', '"smtp.gmail.com"'::jsonb),
    ('notification.email.port', '587'::jsonb),
    ('notification.email.password', '""'::jsonb),
    ('notification.email.max_conns', '1'::jsonb),
    ('notification.email.idle_timeout', '"5s"'::jsonb),
    ('notification.email.wait_timeout', '"5s"'::jsonb),
    ('notification.email.auth_protocol', '"plain"'::jsonb),
    ('notification.email.email_address', '"admin@yourcompany.com"'::jsonb),
    ('notification.email.max_msg_retries', '3'::jsonb),
    ('notification.email.enabled', 'false'::jsonb);

-- Default conversation priorities
INSERT INTO conversation_priorities
("name")
VALUES('Low');
INSERT INTO conversation_priorities
("name")
VALUES('Medium');
INSERT INTO conversation_priorities
("name")
VALUES('High');

-- Default conversation statuses
INSERT INTO conversation_statuses (name) VALUES
('Open'),          
('In Progress'),
('Waiting'),
('Snoozed'),
('Resolved'),
('Closed');

-- Default roles
INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Agent',
		'Role for all agents with limited access to conversations.',
		'{conversations:read_all,conversations:read_unassigned,conversations:read_assigned,conversations:read_team_inbox,conversations:read,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,view:manage}'
	);

INSERT INTO
	roles ("name", description, permissions)
VALUES
	(
		'Admin',
		'Role for users who have complete access to everything.',
		'{general_settings:manage,notification_settings:manage,oidc:manage,conversations:read_all,conversations:read_unassigned,conversations:read_assigned,conversations:read_team_inbox,conversations:read,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,view:manage,status:manage,tags:manage,macros:manage,users:manage,teams:manage,automations:manage,inboxes:manage,roles:manage,reports:manage,templates:manage,business_hours:manage,sla:manage}'
	);


-- Email notification templates
INSERT INTO public.templates
("type", body, is_default, "name", subject, is_builtin)
VALUES('email_notification'::public."template_type", '<p>Hello {{ .agent.full_name }},</p>

<p>A new conversation has been assigned to you:</p>

<div>
    Reference number: {{.conversation.reference_number }} <br>
    Priority: {{.conversation.priority }}<br>
    Subject: {{.conversation.suject }}
</div>

<p>
<a href="{{ RootURL }}/inboxes/assigned/conversation/{{ .conversation.uuid }}">View Conversation</a>
</p>

<div >
    Best regards,<br>
    LibreDesk
</div>', false, 'Conversation assigned', 'New conversation assigned to you', true);