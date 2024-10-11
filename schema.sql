DROP TYPE IF EXISTS "channels" CASCADE; CREATE TYPE "channels" AS ENUM ('email');
DROP TYPE IF EXISTS "media_store" CASCADE; CREATE TYPE "media_store" AS ENUM ('s3', 'fs');
DROP TYPE IF EXISTS "message_type" CASCADE; CREATE TYPE "message_type" AS ENUM ('incoming','outgoing','activity');

DROP TABLE IF EXISTS automation_rules CASCADE;
CREATE TABLE automation_rules (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	"name" VARCHAR(255) NOT NULL,
	description TEXT NULL,
	"type" varchar NOT NULL,
	rules jsonb NULL,
	disabled bool DEFAULT false NOT NULL,
	CONSTRAINT constraint_automation_rules_on_name CHECK (length("name") <= 100),
	CONSTRAINT constraint_automation_rules_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS canned_responses CASCADE;
CREATE TABLE canned_responses (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	title TEXT NOT NULL,
	"content" TEXT NOT NULL,
	CONSTRAINT constraint_canned_responses_on_title CHECK (length(title) <= 100),
	CONSTRAINT constraint_canned_responses_on_content CHECK (length("content") <= 5000)
);

DROP TABLE IF EXISTS contacts CASCADE;
CREATE TABLE contacts (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	first_name TEXT NULL,
	last_name TEXT NULL,
	email VARCHAR(254) NULL,
	phone_number TEXT NULL,
	avatar_url TEXT NULL,
	inbox_id INT NULL,
	source_id TEXT NULL,
	CONSTRAINT constraint_contacts_on_first_name CHECK (length(first_name) <= 100),
	CONSTRAINT constraint_contacts_on_last_name CHECK (length(last_name) <= 100),
	CONSTRAINT constraint_contacts_on_email CHECK (length(email) <= 254),
	CONSTRAINT constraint_contacts_on_phone_number CHECK (length(phone_number) <= 50),
	CONSTRAINT constraint_contacts_on_avatar_url CHECK (length(avatar_url) <= 1000),
	CONSTRAINT constraint_contacts_on_source_id CHECK (length(source_id) <= 5000)
);

DROP TABLE IF EXISTS conversation_participants CASCADE;
CREATE TABLE conversation_participants (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	user_id BIGINT NULL,
	conversation_id BIGINT NULL,
	CONSTRAINT constraint_conversation_participants_conversation_id_and_user_id_unique UNIQUE (conversation_id, user_id)
);

DROP TABLE IF EXISTS inboxes CASCADE;
CREATE TABLE inboxes (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	channel "channels" NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"name" VARCHAR(140) NOT NULL,
	"from" VARCHAR(300) NULL,
	assign_to_team INT NULL,
	soft_delete bool DEFAULT false NOT NULL
);

DROP TABLE IF EXISTS media CASCADE;
CREATE TABLE media (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
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
	CONSTRAINT constraint_media_on_filename CHECK (length(filename) <= 1000)
	CONSTRAINT constraint_media_on_content_id CHECK (length(content_id) <= 100)
);

DROP TABLE IF EXISTS oidc CASCADE;
CREATE TABLE oidc (
	id SERIAL PRIMARY KEY,
	provider_url TEXT NOT NULL,
	client_id TEXT NOT NULL,
	client_secret TEXT NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	provider VARCHAR NULL,
	"name" TEXT NULL
);

DROP TABLE IF EXISTS priority CASCADE;
CREATE TABLE priority (
	id SERIAL PRIMARY KEY,
	"name" TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_priority_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS roles CASCADE;
CREATE TABLE roles (
	id SERIAL PRIMARY KEY,
	permissions _text DEFAULT '{}'::text [] NOT NULL,
	"name" TEXT NULL,
	description TEXT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now()
);
-- Create roles.
INSERT INTO roles
(permissions, "name", description)
VALUES('{conversations:read_unassigned,conversations:read_all,conversations:read,conversations:read_assigned,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,templates:write,templates:read,roles:delete,roles:write,roles:read,inboxes:delete,inboxes:write,inboxes:read,automations:write,automations:delete,automations:read,teams:write,teams:read,users:write,users:read,dashboard_global:read,canned_responses:delete,tags:delete,canned_responses:write,tags:write,status:delete,status:write,status:read,oidc:delete,oidc:read,oidc:write,settings_notifications:read,settings_notifications:write,settings_general:write,templates:delete,admin:read}', 'Admin', 'Role for users who have access to the admin panel.');
INSERT INTO roles
(permissions, "name", description)
VALUES('{conversations:read,conversations:read_unassigned,conversations:read_assigned,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,status:write,status:delete,tags:write,tags:delete,canned_responses:write,canned_responses:delete,dashboard:global,users:write,users:read,teams:read,teams:write,automations:read,automations:write,automations:delete,inboxes:read,inboxes:write,inboxes:delete,roles:read,roles:write,roles:delete,templates:read,templates:write,messages:read,messages:write,dashboard_global:read,oidc:delete,status:read,oidc:write,settings_notifications:read,oidc:read,settings_general:write,settings_notifications:write,conversations:read_all,templates:delete}', 'Agent', 'Role for all agents with limited access.');

DROP TABLE IF EXISTS settings CASCADE;
CREATE TABLE settings (
	"key" TEXT NOT NULL,
	value jsonb DEFAULT '{}'::jsonb NOT NULL,
	updated_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT settings_key_key UNIQUE ("key")
);
CREATE INDEX index_settings_on_key ON settings USING btree ("key");

DROP TABLE IF EXISTS status CASCADE;
CREATE TABLE status (
	id SERIAL PRIMARY KEY,
	"name" TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_status_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS tags CASCADE;
CREATE TABLE tags (
	id BIGSERIAL PRIMARY KEY,
	"name" TEXT NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_tags_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS team_members CASCADE;
CREATE TABLE team_members (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	team_id INT NOT NULL,
	user_id INT NOT NULL,
	CONSTRAINT constraint_team_members_on_team_id_and_user_id_unique UNIQUE (team_id, user_id)
);

DROP TABLE IF EXISTS teams CASCADE;
CREATE TABLE teams (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	"name" VARCHAR(140) NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	auto_assign_conversations bool DEFAULT false NOT NULL,
	CONSTRAINT constraint_teams_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
	id SERIAL PRIMARY KEY,
	body TEXT NOT NULL,
	is_default bool DEFAULT false NOT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	"name" TEXT NULL
);
CREATE UNIQUE INDEX unique_index_templates_on_is_default_when_is_default_is_true ON templates USING btree (is_default)
WHERE (is_default = true);

DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	email VARCHAR(254) NOT NULL,
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	first_name VARCHAR(100) NOT NULL,
	last_name VARCHAR(100) NULL,
	"password" VARCHAR(150) NULL,
	disabled bool DEFAULT false NOT NULL,
	avatar_url TEXT NULL,
	roles _text DEFAULT '{}'::text [] NOT NULL,
	CONSTRAINT constraint_users_on_email_unique UNIQUE (email)
);

DROP TABLE IF EXISTS contact_methods CASCADE;
CREATE TABLE contact_methods (
	id BIGSERIAL PRIMARY KEY,
	contact_id BIGINT REFERENCES contacts(id) ON DELETE CASCADE ON UPDATE CASCADE,
	"source" TEXT NOT NULL,
	source_id TEXT NOT NULL,
	inbox_id INT NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_contact_methods_on_source_and_source_id_unique UNIQUE (contact_id, source_id)
);

DROP TABLE IF EXISTS conversations CASCADE;
CREATE TABLE conversations (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	reference_number TEXT NOT NULL,
	contact_id BIGINT NOT NULL,
	assigned_user_id BIGINT NULL,
	assigned_team_id BIGINT NULL,
	inbox_id INT NOT NULL,
	meta jsonb DEFAULT '{}'::json NOT NULL,
	assignee_last_seen_at TIMESTAMPTZ DEFAULT now(),
	first_reply_at TIMESTAMPTZ NULL,
	closed_at TIMESTAMPTZ NULL,
	resolved_at TIMESTAMPTZ NULL,
	status_id int REFERENCES status(id),
	priority_id int REFERENCES priority(id)
);

DROP TABLE IF EXISTS messages CASCADE;
CREATE TABLE messages (
	id BIGSERIAL PRIMARY KEY,
	updated_at TIMESTAMPTZ DEFAULT now(),
	"uuid" uuid DEFAULT gen_random_uuid() NOT NULL,
	"type" TEXT NOT NULL,
	status TEXT NULL,
	conversation_id BIGSERIAL REFERENCES conversations(id),
	"content" TEXT NULL,
	sender_id INT NULL,
	private bool NULL,
	content_type TEXT,
	source_id TEXT NULL,
	meta jsonb DEFAULT '{}'::jsonb NULL,
	inbox_id INT NULL,
	sender_type varchar NULL,
	created_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_messages_on_content_type CHECK (length(content_type) <= 50)
);

DROP TABLE IF EXISTS conversation_tags CASCADE;
CREATE TABLE conversation_tags (
	id BIGSERIAL PRIMARY KEY,
	tag_id BIGSERIAL REFERENCES tags(id),
	conversation_id BIGSERIAL REFERENCES conversations(id),
	created_at TIMESTAMPTZ DEFAULT now(),
	updated_at TIMESTAMPTZ DEFAULT now(),
	CONSTRAINT constraint_conversation_tags_on_conversation_id_and_tag_id_unique UNIQUE (conversation_id, tag_id)
);


-- Default settings
INSERT INTO settings ("key", value)
VALUES
    ('app.lang', '"en"'::jsonb),
    ('app.root_url', '"http://localhost:9009"'::jsonb),
    ('app.site_name', '"Helpdesk"'::jsonb),
    ('app.favicon_url', '""'::jsonb),
    ('app.max_file_upload_size', '20'::jsonb),
    ('app.allowed_file_upload_extensions', '["*"]'::jsonb),
	('notification.email.username', '""'::jsonb),
    ('notification.email.host', '""'::jsonb),
    ('notification.email.port', '587'::jsonb),
    ('notification.email.password', '""'::jsonb),
    ('notification.email.max_conns', '5'::jsonb),
    ('notification.email.idle_timeout', '"30s"'::jsonb),
    ('notification.email.wait_timeout', '"30s"'::jsonb),
    ('notification.email.auth_protocol', '""'::jsonb),
    ('notification.email.email_address', '""'::jsonb),
    ('notification.email.max_msg_retries', '3'::jsonb),
    ('notification.email.enabled', 'false'::jsonb);


INSERT INTO priority
(id, "name")
VALUES(1, 'Low');
INSERT INTO priority
(id, "name")
VALUES(2, 'Medium');
INSERT INTO priority
(id, "name")
VALUES(3, 'High');

INSERT INTO status
(id, "name")
VALUES(1, 'Open');
INSERT INTO status
(id, "name")
VALUES(2, 'Replied');
INSERT INTO status
(id, "name")
VALUES(3, 'Resolved');
INSERT INTO status
(id, "name")
VALUES(4, 'Closed');