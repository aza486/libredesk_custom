package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Stable baseline required before running any historical migration. Missing
// objects are diagnosed, never recreated over a populated or damaged schema.
var baseInstallationTables = strings.Fields(`settings users roles user_roles teams team_members inboxes conversations conversation_messages tags conversation_tags conversation_statuses conversation_priorities templates sla_policies applied_slas`)

func checkTables(ctx context.Context, db *sqlx.DB, tables []string) error {
	var missing []string
	if err := db.SelectContext(ctx, &missing, `SELECT t FROM unnest($1::text[]) t WHERE NOT EXISTS
 (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname=t AND c.relkind IN ('r','p')) ORDER BY t`, pq.Array(tables)); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete database schema: missing required tables: %s; no schema reset or automatic replay performed", strings.Join(missing, ", "))
	}
	return nil
}

func checkSchema(db *sqlx.DB) (bool, error) {
	var objects int
	// Extension-owned objects (e.g. pg_trgm) do not constitute an application.
	if err := db.Get(&objects, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_%'
 AND c.relkind IN ('r','p','v','m','S','f') AND NOT EXISTS
 (SELECT 1 FROM pg_depend d WHERE d.classid='pg_class'::regclass AND d.objid=c.oid AND d.deptype='e')`); err != nil {
		return false, err
	}
	if objects == 0 {
		// Do not treat orphaned application functions/types as a genuinely empty DB.
		if err := db.Get(&objects, `SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_%' AND t.typtype IN ('e','d') AND NOT EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_type'::regclass AND d.objid=t.oid AND d.deptype='e')`); err != nil {
			return false, err
		}
		if objects == 0 {
			if err := db.Get(&objects, `SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_%' AND NOT EXISTS(SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid AND d.deptype='e')`); err != nil {
				return false, err
			}
			if objects == 0 {
				return false, nil
			}
		}
	}
	if err := checkTables(context.Background(), db, baseInstallationTables); err != nil {
		return false, err
	}
	return true, nil
}

// A deliberately read-only structural contract for the current runtime. Manager
// SQL preparation remains the final validation of its own queries at startup.
func checkRuntimeSchema(ctx context.Context, db *sqlx.DB) error {
	if err := pendingUpgradeError(db); err != nil {
		return err
	}
	tables := strings.Fields(`sla_policies business_hours inboxes teams roles users personal_inbox_owners
 user_roles conversation_statuses conversation_priorities conversations conversation_assignees service_email_addresses conversation_messages
 automation_rules conversation_drafts macros conversation_participants conversation_mentions conversation_last_seen media
 oidc settings tags team_members templates conversation_tags csat_responses
 views applied_slas sla_events scheduled_sla_notifications ai_providers ai_prompts ai_knowledge_base
 embeddings help_centers article_collections help_articles help_article_feedback help_search_queries ai_tools
 ai_assistants ai_assistant_tools ai_agent_events copilot_messages ai_faq_suggestions custom_attribute_definitions contact_notes
 activity_logs webhooks context_links user_notifications conversation_mail_versions manual_resolution_feedback manual_review_annotations
 sidebar_folders message_templates`)
	if err := checkTables(ctx, db, tables); err != nil {
		return err
	}
	columns := strings.Fields(`users.api_key users.api_secret users.work_phone_number users.availability_status users.deleted_at
 inboxes.access_mode inboxes.owner_user_id inboxes.default_signature_id inboxes.linked_email_inbox_id
 conversations.contact_id conversations.custom_attributes conversations.last_continuity_email_sent_at
 conversation_statuses.category conversation_messages.meta conversation_messages.status
 message_templates.uuid message_templates.owner_id message_templates.scope message_templates.revision
 sidebar_folders.scope_type sidebar_folders.scope_id sidebar_folders.filters
 manual_resolution_feedback.feedback_reason manual_resolution_feedback.committed_at
 manual_review_annotations.feedback_reason conversation_mail_versions.version conversation_mail_versions.resolved_version`)
	var missing []string
	if err := db.SelectContext(ctx, &missing, `SELECT k FROM unnest($1::text[]) k WHERE NOT EXISTS(SELECT 1 FROM information_schema.columns c WHERE c.table_schema=current_schema() AND c.table_name=split_part(k,'.',1) AND c.column_name=split_part(k,'.',2)) ORDER BY k`, pq.Array(columns)); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete runtime schema: missing columns: %s; restore/review the schema instead of replaying completed migrations", strings.Join(missing, ", "))
	}
	triggers := strings.Fields(`inboxes.validate_inbox_access inboxes.sync_personal_inbox_primary_owner inboxes.inbox_signature_type
 personal_inbox_owners.validate_personal_inbox_owner personal_inbox_owners.protect_personal_inbox_owner
 users.protect_personal_inbox_owner templates.protect_inbox_signature_type
 conversation_messages.libredesk_mail_input_clock conversation_messages.libredesk_manual_resolution_feedback
 manual_resolution_feedback.libredesk_manual_reason_snapshot manual_resolution_feedback.libredesk_manual_reason_immutable`)
	if err := db.SelectContext(ctx, &missing, `SELECT k FROM unnest($1::text[]) k WHERE NOT EXISTS(SELECT 1 FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname=split_part(k,'.',1) AND t.tgname=split_part(k,'.',2) AND t.tgenabled IN ('O','A') AND NOT t.tgisinternal) ORDER BY k`, pq.Array(triggers)); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete runtime schema: missing/disabled triggers: %s", strings.Join(missing, ", "))
	}
	constraints := strings.Fields(`message_templates.message_templates_owner_check users.users_pkey roles.roles_pkey settings.settings_key_key user_roles.user_roles_user_id_fkey user_roles.user_roles_role_id_fkey conversations.conversations_inbox_id_fkey conversation_messages.conversation_messages_conversation_id_fkey conversations.conversations_contact_required inboxes.inboxes_access_owner_check sidebar_folders.sidebar_folders_scope_type_check sidebar_folders.sidebar_folders_scope_id_check`)
	if err := db.SelectContext(ctx, &missing, `SELECT k FROM unnest($1::text[]) k WHERE NOT EXISTS(SELECT 1 FROM pg_constraint ct JOIN pg_class c ON c.oid=ct.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname=split_part(k,'.',1) AND ct.conname=split_part(k,'.',2) AND ct.convalidated) ORDER BY k`, pq.Array(constraints)); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete runtime schema: missing/unvalidated constraints: %s", strings.Join(missing, ", "))
	}
	indexes := strings.Fields(`applied_slas.index_applied_slas_unique_pending_per_conv team_members.index_unique_team_members_on_team_id_and_user_id`)
	if err := db.SelectContext(ctx, &missing, `SELECT k FROM unnest($1::text[]) k WHERE NOT EXISTS(SELECT 1 FROM pg_index i JOIN pg_class ic ON ic.oid=i.indexrelid JOIN pg_class tc ON tc.oid=i.indrelid JOIN pg_namespace n ON n.oid=tc.relnamespace WHERE n.nspname=current_schema() AND tc.relname=split_part(k,'.',1) AND ic.relname=split_part(k,'.',2) AND i.indisunique AND i.indisvalid AND i.indisready) ORDER BY k`, pq.Array(indexes)); err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete runtime schema: missing/invalid unique indexes: %s; no historical deduplication replayed", strings.Join(missing, ", "))
	}
	var valid bool
	if err := db.GetContext(ctx, &valid, `SELECT (SELECT is_nullable='YES' FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='conversations' AND column_name='contact_id')
 AND EXISTS(SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname=current_schema() AND t.typname='message_type' AND e.enumlabel='internal')
 AND EXISTS(SELECT 1 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname=current_schema() AND t.typname='message_status' AND e.enumlabel='stored')`); err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("incomplete runtime schema: v2.24.0 contactless internal message contract is missing")
	}
	var settings []string
	if err := db.SelectContext(ctx, &settings, `SELECT k FROM unnest($1::text[]) k WHERE NOT EXISTS(SELECT 1 FROM settings WHERE key=k) ORDER BY k`, pq.Array(strings.Fields(`app.lang app.root_url app.site_name app.timezone notification.email.enabled notification.email.email_address notification.email.host notification.email.port notification.email.password`))); err != nil {
		return err
	}
	if len(settings) > 0 {
		return fmt.Errorf("incomplete installation: missing required settings: %s; existing settings were not reset", strings.Join(settings, ", "))
	}
	return nil
}
