DROP TYPE IF EXISTS "channels" CASCADE; CREATE TYPE "channels" AS ENUM ('email');
DROP TYPE IF EXISTS "media_store" CASCADE; CREATE TYPE "media_store" AS ENUM ('s3', 'fs');
DROP TYPE IF EXISTS "message_type" CASCADE; CREATE TYPE "message_type" AS ENUM ('incoming','outgoing','activity');
DROP TYPE IF EXISTS "message_sender_type" CASCADE; CREATE TYPE "message_sender_type" AS ENUM ('user','contact');
DROP TYPE IF EXISTS "message_status" CASCADE; CREATE TYPE "message_status" AS ENUM ('received','sent','failed','pending');
DROP TYPE IF EXISTS "content_type" CASCADE; CREATE TYPE "content_type" AS ENUM ('text','html');

DROP TABLE IF EXISTS teams CASCADE;
CREATE TABLE teams (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" VARCHAR(140) NOT NULL,
	disabled bool DEFAULT false NOT NULL,
	auto_assign_conversations bool DEFAULT false NOT NULL,
	CONSTRAINT constraint_teams_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS users CASCADE;
CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	deleted_at TIMESTAMPTZ NULL,
	disabled bool DEFAULT false NOT NULL,
	email VARCHAR(254) NOT NULL,
	first_name VARCHAR(100) NOT NULL,
	last_name VARCHAR(100) NULL,
	"password" VARCHAR(150) NULL,
	avatar_url TEXT NULL,
	roles _text DEFAULT '{}'::text [] NOT NULL,
	reset_password_token TEXT NULL,
	reset_password_token_expiry TIMESTAMPTZ NULL,
	CONSTRAINT constraint_users_on_email_unique UNIQUE (email)
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
    reference_number TEXT UNIQUE NOT NULL,
    contact_id BIGINT NOT NULL,
    assigned_user_id INT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    assigned_team_id INT REFERENCES teams(id) ON DELETE SET NULL ON UPDATE CASCADE,
    inbox_id INT NOT NULL,
    meta JSONB DEFAULT '{}'::JSON NOT NULL,
    assignee_last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    first_reply_at TIMESTAMPTZ NULL,
    closed_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,
    status_id INT REFERENCES conversation_statuses(id),
    priority_id INT REFERENCES conversation_priorities(id)
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
    source_id TEXT NULL,
    sender_id INT NULL,
    sender_type message_sender_type NOT NULL,
    meta JSONB DEFAULT '{}'::JSONB NULL
);

DROP TABLE IF EXISTS automation_rules CASCADE;
CREATE TABLE automation_rules (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    "name" VARCHAR(255) NOT NULL,
    description TEXT NULL,
    "type" VARCHAR NOT NULL,
    rules JSONB NULL,
    events TEXT[] NULL,
    disabled BOOL DEFAULT false NOT NULL,
    CONSTRAINT constraint_automation_rules_on_name CHECK (length("name") <= 100),
    CONSTRAINT constraint_automation_rules_on_description CHECK (length(description) <= 300)
);

DROP TABLE IF EXISTS canned_responses CASCADE;
CREATE TABLE canned_responses (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	title TEXT NOT NULL,
	"content" TEXT NOT NULL,
	CONSTRAINT constraint_canned_responses_on_title CHECK (length(title) <= 100),
	CONSTRAINT constraint_canned_responses_on_content CHECK (length("content") <= 5000)
);

DROP TABLE IF EXISTS contacts CASCADE;
CREATE TABLE contacts (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
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
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	user_id INT NOT NULL,
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
	config jsonb DEFAULT '{}'::jsonb NOT NULL,
	"name" VARCHAR(140) NOT NULL,
	"from" VARCHAR(300) NULL
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
	CONSTRAINT constraint_media_on_content_id CHECK (length(content_id) <= 100)
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

-- Roles.
INSERT INTO roles
(permissions, "name", description)
VALUES('{conversations:read,conversations:read_unassigned,conversations:read_assigned,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write}', 'Agent', 'Role for all agents with limited access to conversations.');
INSERT INTO roles
(permissions, "name", description)
VALUES('{conversations:read_unassigned,teams:delete,users:delete,conversations:read_all,conversations:read,conversations:read_assigned,conversations:update_user_assignee,conversations:update_team_assignee,conversations:update_priority,conversations:update_status,conversations:update_tags,messages:read,messages:write,templates:write,templates:read,roles:delete,roles:write,roles:read,inboxes:delete,inboxes:write,inboxes:read,automations:write,automations:delete,automations:read,teams:write,teams:read,users:write,users:read,dashboard_global:read,canned_responses:delete,tags:delete,canned_responses:write,tags:write,status:delete,status:write,status:read,oidc:delete,oidc:read,oidc:write,settings_notifications:read,settings_notifications:write,settings_general:write,templates:delete,admin:read}', 'Admin', 'Role for users who have complete access to everything.');

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
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	"name" TEXT NOT NULL,
	CONSTRAINT constraint_tags_on_name_unique UNIQUE ("name")
);

DROP TABLE IF EXISTS team_members CASCADE;
CREATE TABLE team_members (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
	user_id INT NOT NULL,
	CONSTRAINT constraint_team_members_on_team_id_and_user_id_unique UNIQUE (team_id, user_id)
);

DROP TABLE IF EXISTS templates CASCADE;
CREATE TABLE templates (
	id SERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	body TEXT NOT NULL,
	is_default bool DEFAULT false NOT NULL,
	"name" TEXT NULL
);
CREATE UNIQUE INDEX unique_index_templates_on_is_default_when_is_default_is_true ON templates USING btree (is_default)
WHERE (is_default = true);

DROP TABLE IF EXISTS contact_methods CASCADE;
CREATE TABLE contact_methods (
	id BIGSERIAL PRIMARY KEY,
	contact_id BIGINT REFERENCES contacts(id) ON DELETE CASCADE ON UPDATE CASCADE,
	"source" TEXT NOT NULL,
	source_id TEXT NOT NULL,
	inbox_id INT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	CONSTRAINT constraint_contact_methods_on_source_and_source_id_unique UNIQUE (contact_id, source_id)
);

DROP TABLE IF EXISTS conversation_tags CASCADE;
CREATE TABLE conversation_tags (
	id BIGSERIAL PRIMARY KEY,
	tag_id INT REFERENCES tags(id) ON DELETE CASCADE ON UPDATE CASCADE,
	conversation_id BIGSERIAL REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	CONSTRAINT constraint_conversation_tags_on_conversation_id_and_tag_id_unique UNIQUE (conversation_id, tag_id)
);

-- Default settings
INSERT INTO settings ("key", value)
VALUES
    ('app.lang', '"en"'::jsonb),
    ('app.root_url', '"http://localhost:9000"'::jsonb),
    ('app.logo_url', '""'::jsonb),
    ('app.site_name', '"Helpdesk"'::jsonb),
    ('app.favicon_url', '"http://localhost:9000/favicon.ico"'::jsonb),
    ('app.max_file_upload_size', '20'::jsonb),
    ('app.allowed_file_upload_extensions', '["*"]'::jsonb),
    ('notification.email.username', '"admin@yourcompany.com"'::jsonb),
    ('notification.email.host', '""'::jsonb),
    ('notification.email.port', '587'::jsonb),
    ('notification.email.password', '""'::jsonb),
    ('notification.email.max_conns', '1'::jsonb),
    ('notification.email.idle_timeout', '"5s"'::jsonb),
    ('notification.email.wait_timeout', '"5s"'::jsonb),
    ('notification.email.auth_protocol', '"Plain"'::jsonb),
    ('notification.email.email_address', '""'::jsonb),
    ('notification.email.max_msg_retries', '3'::jsonb),
    ('notification.email.enabled', 'false'::jsonb);


INSERT INTO conversation_priorities
("name")
VALUES('Low');
INSERT INTO conversation_priorities
("name")
VALUES('Medium');
INSERT INTO conversation_priorities
("name")
VALUES('High');

INSERT INTO conversation_statuses
("name")
VALUES('Open');
INSERT INTO conversation_statuses
("name")
VALUES('Replied');
INSERT INTO conversation_statuses
("name")
VALUES('Resolved');
INSERT INTO conversation_statuses
("name")
VALUES('Closed');