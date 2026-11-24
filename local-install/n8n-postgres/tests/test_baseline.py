#!/usr/bin/env python3
"""Integration tests for a disposable, labelled, network-isolated PG16 container."""
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import shutil
import tempfile
import unittest
from unittest import mock
import uuid

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
import install
import configure_roles

CONTAINER = os.environ.get("F15_TEST_CONTAINER", "")
DB = "f15_validation_" + uuid.uuid4().hex[:8]
OWNER = "f15_schema_owner"
WORKFLOW = "f15_workflow"
REVIEWER = "libredesk_learning"
PUBLISHER = "f15_case_publisher"
KNOWLEDGE = "f15_knowledge"
VECTOR = "f15_vector"


def sql(text, role=OWNER, db=DB, fail=False):
    proc = subprocess.run(["docker", "exec", "-i", CONTAINER, "psql", "-XqAt",
                           "-v", "ON_ERROR_STOP=1", "-U", role, "-d", db],
                          input=text, text=True, capture_output=True)
    if not fail and proc.returncode:
        raise AssertionError(proc.stderr)
    if fail and not proc.returncode:
        raise AssertionError("Expected SQL failure")
    if fail:
        errors = "\n".join(line for line in proc.stderr.splitlines() if line.startswith("ERROR:"))
        if not errors or "syntax error" in errors or "does not exist" in errors:
            raise AssertionError("Test setup failure instead of expected contract rejection: " + proc.stderr)
        if isinstance(fail, str) and fail not in proc.stderr:
            raise AssertionError("Unexpected rejection: " + proc.stderr)
    return proc.stdout.strip()


def isolated_db(name):
    subprocess.run(["docker", "exec", CONTAINER, "createdb", "-U", "f15_lab_admin",
                    "-O", OWNER, name], check=True, capture_output=True)
    sql((ROOT / "sql/000_extensions.sql").read_text(), "f15_lab_admin", name)


def mail(message="incoming", conversation="conversation"):
    return f"""INSERT INTO mail_events(message_uuid,conversation_uuid,direction,suggested_message,
      clean_text,categories) VALUES('{message}','{conversation}','inbound','AI draft',
      'Synthetic customer question',ARRAY['test']);"""


def record(outgoing="outgoing", sent="2026-01-01T12:00:00Z", text="Synthetic answer"):
    return f"""SELECT row_to_json(x) FROM record_libredesk_sent_v2('conversation','incoming',
      '{outgoing}','human','{text}','{sent}', '73','1') x;"""


def case_source():
    source, outgoing, feedback = [str(uuid.uuid4()) for _ in range(3)]
    envelope = {"version": 11, "status": "staged", "customer_question": "How is a synthetic case reviewed?",
                "human_answer": "A human reviews this synthetic answer.", "feedback_reason": "",
                "feedback_type": "manual_resolution", "original_categories": ["test"],
                "corrected_categories": [], "has_human_resolution": True,
                "has_explicit_feedback": False, "requires_quality_review": True}
    data = json.dumps({"response_source": "human", "learning_case_v11": envelope})
    return feedback, f"""{mail(source)} INSERT INTO ai_feedback(id,mail_event_message_uuid,
      outgoing_message_uuid,agent_id,human_answer,feedback_type,feedback_data)
      VALUES('{feedback}','{source}','{outgoing}','73','A human reviews this synthetic answer.',
       'manual_resolution','{data}');"""


class Baseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not CONTAINER:
            raise RuntimeError("Set F15_TEST_CONTAINER to a NEW disposable labelled test container")
        info = json.loads(subprocess.check_output(["docker", "inspect", CONTAINER]))[0]
        if (info["Config"].get("Labels", {}).get("libredesk.f15-isolated") != "true"
                or info["HostConfig"]["NetworkMode"] != "none"
                or not info["HostConfig"].get("Tmpfs", {}).get("/var/lib/postgresql/data")
                or info["HostConfig"].get("PortBindings")
                or info.get("Mounts")):
            raise RuntimeError("Refusing tests: container is not the isolated tmpfs-only fixture")
        for role in [OWNER, WORKFLOW, REVIEWER, PUBLISHER, KNOWLEDGE, VECTOR]:
            sql(f"DO $$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='{role}') THEN CREATE ROLE {role} LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE; END IF; END $$;", "f15_lab_admin", "f15_lab")
        isolated_db(DB)
        sql(install.script("core"))
        sql(configure_roles.script(WORKFLOW, PUBLISHER))

    def transaction(self, text, role=WORKFLOW, fail=False):
        return sql("BEGIN;\n" + text.rstrip().rstrip(";") + ";\nROLLBACK;", role, fail=fail)

    def test_01_empty_installation(self):
        tables = json.loads((ROOT / "manifest.json").read_text())["objects"]["core"]["tables"]
        for table in tables:
            with self.subTest(table=table):
                self.assertEqual(sql(f"SELECT count(*) FROM {table};"), "0")
        self.assertEqual(sql("SELECT count(*) FROM n8n_schema_installation"), "1")

    def test_02_production_structure_contract(self):
        actual = json.loads(sql((ROOT / "tests/catalog.sql").read_text()))
        expected = json.loads((ROOT / "tests/production-contract.json").read_text())
        for category, values in expected.items():
            with self.subTest(category=category):
                for key, value in values.items():
                    self.assertEqual(actual[category].get(key), value, f"{category}: {key}")

    def test_03_repeat_preserves_data_and_catalog(self):
        sql(mail("persistent_synthetic"))
        try:
            before = sql("SELECT row_to_json(m) FROM mail_events m WHERE message_uuid='persistent_synthetic'")
            registry = sql("SELECT row_to_json(m) FROM n8n_schema_installation m")
            sql(install.script("core"))
            self.assertEqual(sql("SELECT row_to_json(m) FROM mail_events m WHERE message_uuid='persistent_synthetic'"), before)
            self.assertEqual(sql("SELECT row_to_json(m) FROM n8n_schema_installation m"), registry)
        finally:
            sql("DELETE FROM mail_events WHERE message_uuid='persistent_synthetic'")

    def test_04_concurrent_initialization(self):
        name = DB + "_parallel"
        isolated_db(name)
        with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
            results = list(pool.map(lambda _: sql(install.script("core"), db=name), range(4)))
        self.assertEqual(len(results), 4)
        self.assertEqual(sql("SELECT count(*) FROM n8n_schema_installation", db=name), "1")

    def test_05_unmanaged_schema_refused(self):
        name = DB + "_unmanaged"
        isolated_db(name)
        sql("CREATE TABLE existing_business(id int); INSERT INTO existing_business VALUES(17)", db=name)
        sql(install.script("core"), db=name, fail=True)
        self.assertEqual(sql("SELECT * FROM existing_business", db=name), "17")
        self.assertEqual(sql("SELECT to_regclass('public.n8n_schema_installation') IS NULL", db=name), "t")

    def test_06_drift_is_refused(self):
        sql("ALTER TABLE cases ADD COLUMN drift_test integer")
        try:
            sql(install.script("core"), fail=True)
            sql(install.script("core", check=True), fail=True)
        finally:
            sql("ALTER TABLE cases DROP COLUMN drift_test")
        sql(install.script("core", check=True))

    def test_07_readonly_check(self):
        self.assertIn("verified", sql("SET default_transaction_read_only=on;\n" + install.script("core", check=True)))

    def test_08_rollback_after_late_failure(self):
        name = DB + "_rollback"
        isolated_db(name)
        batch = install.script("core").replace("SELECT 'Schema version and structure verified';", "SELECT 1/0;")
        sql(batch, db=name, fail="division by zero")
        self.assertEqual(sql("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'", db=name), "0")

    def test_09_feedback_idempotency_and_conflict(self):
        call = "SELECT row_to_json(x) FROM commit_ai_feedback('11111111-1111-4111-8111-111111111111','incoming','conversation','outgoing','73','Synthetic answer','Reason','2026-01-01T00:00:00Z') x;"
        rows = self.transaction(mail() + call + call + call.replace("'Synthetic answer'", "'different'"))
        values = [json.loads(x) for x in rows.splitlines()]
        self.assertEqual([v['result'] for v in values], ["created", "already_committed", "conflict"])

    def test_10_feedback_invalid_and_foreign_key(self):
        self.assertIn("mail_event_not_found", self.transaction("SELECT * FROM commit_ai_feedback(gen_random_uuid(),'missing','conversation','o','73','text',NULL,now())"))
        self.transaction("INSERT INTO ai_feedback(mail_event_message_uuid,agent_id) VALUES('missing','73')", fail=True)

    def test_11_receipt_idempotency_and_terminal_outcome(self):
        batch = mail() + record() + record() + record(text="different")
        batch += "SELECT row_to_json(x) FROM finish_libredesk_sent_v2('outgoing','resolved') x;" * 2
        batch += "SELECT row_to_json(x) FROM finish_libredesk_sent_v2('outgoing','superseded') x;"
        values = [json.loads(x) for x in self.transaction(batch).splitlines()]
        self.assertEqual([v['outcome'] for v in values], ["recorded", "duplicate_accepted", "duplicate_payload_mismatch", "resolved", "resolved", "terminal_outcome_conflict"])

    def test_12_newer_response_wins(self):
        batch = mail() + record("newer", "2026-01-02T00:00:00Z", "Newer response") + record("older", "2026-01-01T00:00:00Z", "Older response")
        self.assertEqual(self.transaction(batch+"SELECT sent_message FROM mail_events WHERE message_uuid='incoming'").splitlines()[-1], "Newer response")

    def test_13_constraints(self):
        for statement in [mail()+mail(), mail()+"UPDATE mail_events SET mail_type='other'", mail()+"UPDATE mail_events SET response_source='robot'", "INSERT INTO rag_items(source_type,category,retrieval_text,embedding) VALUES('response_example','test','text',array_fill(1::real,ARRAY[3])::vector)"]:
            with self.subTest(statement=statement[:35]):
                self.transaction(statement, fail=True)

    def test_14_rag_vector_views(self):
        batch = """INSERT INTO rag_items(source_type,category,retrieval_text,human_answer,embedding,quality_status,approved,active,learning_data)
          VALUES('response_example','test','Synthetic searchable answer','This is the synthetic human answer',
           array_fill(1::real,ARRAY[1536])::vector,'approved',true,true,
           '{"target_agent":"final_answer","example_role":"style_only","factual_authority":false}');
          SELECT text FROM rag_final_style_view ORDER BY embedding <=> array_fill(1::real,ARRAY[1536])::vector LIMIT 1;"""
        self.assertEqual(self.transaction(batch), "Synthetic searchable answer")

    def test_15_workflow_cannot_approve_reviews(self):
        self.transaction("UPDATE learning_reviews SET decision='approved'", fail="permission denied")
        self.transaction("INSERT INTO learning_reviews(item_id,decision,original) VALUES(gen_random_uuid(),'approved','{}')", fail="permission denied")
        self.transaction("SELECT * FROM learning_case_review_v11(gen_random_uuid(),0,'approved','forged',true,true,true,'[]')", fail="permission denied")

    def test_16_case_staging_and_publisher_boundary(self):
        fid, source = case_source()
        self.assertTrue(self.transaction(source+f"SELECT learning_case_stage_v11('{fid}');"))
        self.transaction("SELECT * FROM rag_items", PUBLISHER, fail="permission denied")
        self.transaction("SELECT * FROM ai_feedback", PUBLISHER, fail="permission denied")
        self.assertEqual(sql("SELECT count(*) FROM learning_case_publisher_candidates_v11", PUBLISHER), "0")

    def test_17_case_full_review_publish(self):
        fid, source = case_source()
        try:
            sql(source, WORKFLOW)
            item = sql(f"SELECT learning_case_stage_v11('{fid}')", WORKFLOW)
            sql(f"SELECT learning_case_review_v11('{item}',0,'approved','human-reviewer',true,true,true,'[]')", REVIEWER)
            self.assertEqual(sql(f"SELECT count(*) FROM learning_case_publisher_candidates_v11 WHERE id='{item}'", PUBLISHER), "1")
            sql(f"SELECT learning_case_publish_v11('{item}',1,array_fill(1::real,ARRAY[1536])::vector)", PUBLISHER)
            self.assertEqual(sql(f"SELECT count(*) FROM rag_case_example_view WHERE id='{item}'", WORKFLOW), "1")
            sql(f"SELECT learning_case_publish_v11('{item}',1,array_fill(1::real,ARRAY[1536])::vector)", PUBLISHER, fail=True)
        finally:
            # Case guards intentionally prevent deletion. Synthetic rows stay in
            # this disposable container until it is removed, never in production.
            pass

    def test_18_forged_guc_does_not_grant_case_authority(self):
        self.transaction("SET LOCAL libredesk.review_writer='true'; INSERT INTO rag_items(source_type,category,retrieval_text) VALUES('case_example','test','forged case')", fail="case write requires controlled v11 function")

    def test_19_reviewer_cannot_read_raw_mail(self):
        self.transaction("SELECT raw_text FROM mail_events", REVIEWER, fail="permission denied")
        self.assertEqual(sql("SELECT count(message_uuid) FROM mail_events", REVIEWER), "1")

    def test_20_optional_knowledge(self):
        self.assertEqual(sql("SELECT to_regclass('kd_sources') IS NULL"), "t")
        sql(install.script("knowledge"))
        sql(configure_roles.script(WORKFLOW, PUBLISHER, KNOWLEDGE))
        sql(install.script("knowledge"))
        batch = "INSERT INTO kd_sources(source_type,title,source_text,content_hash) VALUES('other','Synthetic title',repeat('Synthetic text ',20),'hash'); SELECT count(*) FROM kd_sources;"
        self.assertEqual(self.transaction(batch, KNOWLEDGE), "1")
        self.transaction("SELECT * FROM mail_events", KNOWLEDGE, fail="permission denied")

    def test_21_repeat_role_configuration(self):
        self.assertIn("verified", sql(configure_roles.script(WORKFLOW, PUBLISHER, KNOWLEDGE)))

    def test_22_runtime_cannot_own_schema_or_marker(self):
        self.transaction("CREATE TABLE injected(id int)", fail="permission denied")
        self.transaction("UPDATE n8n_schema_installation SET version=2", fail="permission denied")
        self.transaction(f"SET ROLE {OWNER}", fail="permission denied")

    def test_23_no_public_mutator_privilege(self):
        self.assertEqual(sql("SELECT has_function_privilege('f15_knowledge','commit_ai_feedback(uuid,text,text,text,text,text,text,timestamptz)','EXECUTE')"), "f")

    def test_24_changed_sources_are_not_implicitly_applied(self):
        with tempfile.TemporaryDirectory(prefix="f15-source-check-") as directory:
            copied = Path(directory) / "baseline"
            shutil.copytree(ROOT, copied, ignore=shutil.ignore_patterns("__pycache__"))
            changed = copied / "sql/030_views.sql"
            changed.write_text(changed.read_text() + "\n-- synthetic new source revision\n")
            with mock.patch.object(install, "ROOT", copied):
                with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                    install.script("core")
                manifest = json.loads((copied / "manifest.json").read_text())
                manifest["sha256"]["sql/030_views.sql"] = hashlib.sha256(changed.read_bytes()).hexdigest()
                (copied / "manifest.json").write_text(json.dumps(manifest))
                sql(install.script("core"), fail="Version/source/schema mismatch")

    def test_25_optional_structure_contract(self):
        actual = json.loads(sql((ROOT / "tests/catalog.sql").read_text()))
        expected = json.loads((ROOT / "tests/production-optional-contract.json").read_text())
        for category, values in expected.items():
            for key, value in values.items():
                self.assertEqual(actual[category].get(key), value, f"{category}: {key}")

    def test_26_native_vector_initialization(self):
        sql(configure_roles.script(WORKFLOW, PUBLISHER, KNOWLEDGE, VECTOR))
        statements = "CREATE EXTENSION IF NOT EXISTS vector;"
        for table in ["rag_final_style_view", "rag_category_learning_view", "rag_verified_fact_view", "rag_case_example_view", "learning_embedding_stage"]:
            statements += f"CREATE TABLE IF NOT EXISTS public.{table}(id uuid DEFAULT gen_random_uuid() PRIMARY KEY,text text,metadata jsonb,embedding vector);"
        self.transaction(statements, VECTOR)
        self.transaction("UPDATE rag_items SET approved=true", VECTOR, fail="permission denied")
        self.transaction("SELECT * FROM ai_feedback", VECTOR, fail="permission denied")
        self.transaction("SELECT learning_case_publish_v11(gen_random_uuid(),0,array_fill(1::real,ARRAY[1536])::vector)", VECTOR, fail="permission denied")
        self.transaction(f"SET ROLE {OWNER}", VECTOR, fail="permission denied")
        self.transaction("CREATE TABLE public.guard_forgery(id uuid,active boolean); CREATE TRIGGER forged BEFORE UPDATE ON public.guard_forgery FOR EACH ROW EXECUTE FUNCTION public.learning_review_activation_guard()", VECTOR, fail="permission denied for function")
        sql(configure_roles.script(WORKFLOW, PUBLISHER, KNOWLEDGE, VECTOR))

    def test_27_concurrent_receipt_idempotence(self):
        source, outgoing, conversation = [str(uuid.uuid4()) for _ in range(3)]
        sql(mail(source, conversation), WORKFLOW)
        call = record(outgoing).replace("'incoming'", f"'{source}'").replace("'conversation'", f"'{conversation}'")
        with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:
            results = list(pool.map(lambda _: json.loads(sql(call, WORKFLOW)), range(6)))
        self.assertEqual(sum(v['outcome'] == 'recorded' for v in results), 1)
        self.assertTrue(all(v['committed'] for v in results))
        self.assertEqual(sql(f"SELECT count(*) FROM libredesk_sent_receipts_v2 WHERE outgoing_message_uuid='{outgoing}'"), "1")

    def test_28_second_install_preserves_all_synthetic_data(self):
        objects = json.loads((ROOT / "manifest.json").read_text())["objects"]
        tables = objects['core']['tables'] + objects['knowledge']['tables'] + ['n8n_schema_installation']
        query = "\n".join(f"SELECT '{t}|'||md5(COALESCE((SELECT jsonb_agg(to_jsonb(x) ORDER BY to_jsonb(x)::text)::text FROM public.{t} x),''));" for t in tables)
        before = sql(query)
        catalog = sql((ROOT / 'tests/catalog.sql').read_text())
        sql(install.script('core'))
        sql(install.script('knowledge'))
        self.assertEqual(sql(query), before)
        self.assertEqual(sql((ROOT / 'tests/catalog.sql').read_text()), catalog)

    def test_29_owner_membership_is_rejected(self):
        sql(f"GRANT {OWNER} TO {WORKFLOW}", 'f15_lab_admin')
        try:
            sql(configure_roles.script(WORKFLOW, PUBLISHER), fail="Runtime principal already has ownership")
        finally:
            sql(f"REVOKE {OWNER} FROM {WORKFLOW}", 'f15_lab_admin')

    def test_30_review_audit_is_append_only(self):
        self.transaction("UPDATE learning_case_audit_v11 SET actor='forged'", WORKFLOW, fail="permission denied")
        self.transaction("UPDATE learning_review_audit SET actor='forged'", REVIEWER, fail="permission denied")
        self.transaction("INSERT INTO learning_review_audit(item_id,revision,actor,action,reason,snapshot) SELECT item_id,revision+1,'forged','test','test','{}' FROM learning_reviews", WORKFLOW, fail="permission denied")

    def test_31_version_mismatch_is_rejected(self):
        sql("UPDATE n8n_schema_installation SET version=99 WHERE module='core'")
        try:
            sql(install.script('core'), fail="Version/source/schema mismatch")
        finally:
            sql("UPDATE n8n_schema_installation SET version=1 WHERE module='core'")

    def test_32_extensions_are_repeatable(self):
        before = sql((ROOT / 'tests/catalog.sql').read_text())
        sql((ROOT / 'sql/000_extensions.sql').read_text(), 'f15_lab_admin')
        self.assertEqual(sql((ROOT / 'tests/catalog.sql').read_text()), before)

    def test_33_pending_review_blocks_forged_publication(self):
        item = str(uuid.uuid4())
        seed = f"""INSERT INTO rag_items(id,source_type,category,retrieval_text,active,learning_data)
          VALUES('{item}','response_example','test','Synthetic reviewed example',false,
           '{{"managed_by_review":"true","approval_method":"human_review_v1"}}');
          INSERT INTO learning_reviews(item_id,revision,original) VALUES('{item}',0,'{{}}');
          SET LOCAL libredesk.review_writer='true';
          UPDATE rag_items SET active=true,approved=true,quality_status='approved',
           embedding=array_fill(1::real,ARRAY[1536])::vector WHERE id='{item}';"""
        self.transaction(seed, fail="review decision required")

    def test_34_review_snapshot_and_audit_triggers(self):
        item = str(uuid.uuid4())
        seed = f"""INSERT INTO rag_items(id,source_type,category,retrieval_text,active)
          VALUES('{item}','employee_fact','test','Synthetic fact',false);
          INSERT INTO learning_reviews(item_id,original) VALUES('{item}','{{}}');"""
        self.transaction(seed+f"UPDATE learning_reviews SET original='{{\"forged\":true}}' WHERE item_id='{item}'", OWNER, fail="original review snapshot immutable")
        self.transaction(seed+f"INSERT INTO learning_review_audit(item_id,revision,actor,action,reason,snapshot) VALUES('{item}',1,'test','test','test','{{}}'); UPDATE learning_review_audit SET actor='forged'", OWNER, fail="review audit is append only")

    def test_35_canonical_source_and_case_audit_triggers(self):
        fid, source = case_source()
        self.transaction(source+f"SELECT learning_case_stage_v11('{fid}'); UPDATE ai_feedback SET agent_id='99' WHERE id='{fid}'", fail="canonical case source immutable")
        self.transaction(source+f"SELECT learning_case_stage_v11('{fid}'); UPDATE learning_case_audit_v11 SET actor='forged'", OWNER, fail="case audit is append only")


if __name__ == "__main__":
    unittest.main(verbosity=2)
