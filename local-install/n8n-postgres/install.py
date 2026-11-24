#!/usr/bin/env python3
"""Offline schema installer. Connection settings come only from libpq PG* variables."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parent
CORE = ["sql/010_base.sql", "sql/020_functions.sql", "sql/030_views.sql",
        "learning/001_review.sql", "learning/002_style_view.sql",
        "learning/005_case_v11.sql", "learning/006_case_publisher_v11.sql",
        "learning/007_case_stage_category_fix_v11.sql"]


def literal(value):
    return "'" + str(value).replace("'", "''") + "'"


def run(sql, psql="psql"):
    result = subprocess.run([psql, "-X", "-qAt", "-v", "ON_ERROR_STOP=1"],
                            input=sql, text=True, capture_output=True)
    if result.returncode:
        # SQL context can contain caller data or connection strings. Never echo it.
        reasons = ["Refusing to adopt an existing unmanaged schema",
                   "Version/source/schema mismatch; no automatic overwrite",
                   "Core installation is missing", "Requested module is missing",
                   "Core installation required", "Schema owner required",
                   "Create the separate restricted principals first; passwords are configured externally",
                   "Runtime principal already has ownership, membership or elevated privileges",
                   "Existing role grants violate review/publisher separation",
                   "Install PostgreSQL 16 / pgvector 0.8.6 / pgcrypto prerequisites first"]
        reason = next((r for r in reasons if r in result.stderr), "Database or connection error; details withheld")
        print("Schema operation failed: " + reason + ". No automatic retry.", file=sys.stderr)
        return result.returncode
    print(result.stdout.strip())
    return 0


def body(path):
    # Keep the upstream artifacts byte-identical on disk. Only their top-level
    # transaction delimiters are removed in the single atomic installation batch.
    text = (ROOT / path).read_text()
    return re.sub(r"^(?:BEGIN;|COMMIT;)\s*$", "", text, flags=re.M)


def source_digest(files, manifest):
    values = {p: manifest["sha256"][p] for p in files}
    # Generated guards and ACLs are part of the installed version too. A changed
    # installer must never silently skip new protections on an older installation.
    values["install.py"] = hashlib.sha256((ROOT / "install.py").read_bytes()).hexdigest()
    return hashlib.sha256(json.dumps(values, sort_keys=True).encode()).hexdigest()


def fingerprint(objects):
    tables = ",".join(map(literal, objects["tables"]))
    views = ",".join(map(literal, objects["views"])) or "NULL"
    functions = ",".join(map(literal, objects["functions"])) or "NULL"
    # Ownership/ACL changes belong to the separate role configuration. Everything
    # determining row shape, validation, query behavior or guards is fingerprinted.
    return f"""SELECT md5(COALESCE(string_agg(v,E'\\n' ORDER BY v),'')) FROM (
      SELECT 'column|'||c.relname||'|'||a.attnum||'|'||a.attname||'|'||
        format_type(a.atttypid,a.atttypmod)||'|'||a.attnotnull||'|'||
        COALESCE(pg_get_expr(d.adbin,d.adrelid),'')||'|'||a.attidentity::text AS v
      FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
      JOIN pg_namespace n ON n.oid=c.relnamespace
      LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum
      WHERE n.nspname='public' AND c.relname IN ({tables}) AND a.attnum>0 AND NOT a.attisdropped
      UNION ALL SELECT 'constraint|'||c.relname||'|'||p.conname||'|'||pg_get_constraintdef(p.oid)
      FROM pg_constraint p JOIN pg_class c ON c.oid=p.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='public' AND c.relname IN ({tables})
      UNION ALL SELECT 'index|'||tablename||'|'||indexname||'|'||indexdef
      FROM pg_indexes WHERE schemaname='public' AND tablename IN ({tables})
      UNION ALL SELECT 'view|'||c.relname||'|'||pg_get_viewdef(c.oid)||'|'||COALESCE(c.reloptions::text,'')
      FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='public' AND c.relname IN ({views}) AND c.relkind='v'
      UNION ALL SELECT 'function|'||p.proname||'|'||pg_get_functiondef(p.oid)
      FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname='public' AND p.proname IN ({functions})
      UNION ALL SELECT 'trigger|'||c.relname||'|'||pg_get_triggerdef(t.oid)||'|'||t.tgenabled::text
      FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='public' AND c.relname IN ({tables}) AND NOT t.tgisinternal
    ) required_objects"""


def script(module, check=False):
    manifest = json.loads((ROOT / "manifest.json").read_text())
    files = CORE if module == "core" else ["sql/optional_knowledge.sql"]
    for path, expected in manifest["sha256"].items():
        if hashlib.sha256((ROOT / path).read_bytes()).hexdigest() != expected:
            raise ValueError("Installation source checksum mismatch")
    digest = source_digest(files, manifest)
    obj = manifest["objects"][module]
    fp = fingerprint(obj)
    # Serialize first installation, repeated installation, and optional modules.
    sql = "\\set ON_ERROR_STOP on\nBEGIN;\nSELECT pg_advisory_xact_lock(1869432081,1501);\nSET LOCAL search_path=public;\n"
    if check:
        sql = sql.replace("BEGIN;", "BEGIN READ ONLY;", 1)
    sql += """DO $$ BEGIN
      IF current_setting('server_version_num')::int NOT BETWEEN 160000 AND 169999
        OR NOT EXISTS(SELECT FROM pg_extension WHERE extname='vector' AND extversion='0.8.6')
        OR NOT EXISTS(SELECT FROM pg_extension WHERE extname='pgcrypto') THEN
        RAISE EXCEPTION 'Install PostgreSQL 16 / pgvector 0.8.6 / pgcrypto prerequisites first';
      END IF;
    END $$;
    SELECT to_regclass('public.n8n_schema_installation') IS NOT NULL AS has_registry \\gset
    \\if :has_registry
    \\else
    """
    if check or module != "core":
        sql += "DO $$ BEGIN RAISE EXCEPTION 'Core installation is missing'; END $$;\n"
    else:
        sql += """DO $$ BEGIN
          IF EXISTS(SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
            WHERE n.nspname='public' AND NOT EXISTS(SELECT FROM pg_depend d
              WHERE d.classid='pg_class'::regclass AND d.objid=c.oid AND d.deptype='e'))
          OR EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='public' AND NOT EXISTS(SELECT FROM pg_depend d
              WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid AND d.deptype='e'))
          OR EXISTS(SELECT FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
            WHERE n.nspname='public' AND t.typelem=0 AND NOT EXISTS(SELECT FROM pg_depend d
              WHERE d.classid='pg_type'::regclass AND d.objid=t.oid AND d.deptype='e'))
          THEN RAISE EXCEPTION 'Refusing to adopt an existing unmanaged schema'; END IF;
        END $$;
        CREATE TABLE public.n8n_schema_installation(
          module text PRIMARY KEY, version integer NOT NULL, source_sha256 text NOT NULL,
          structure_md5 text NOT NULL, installed_at timestamptz NOT NULL DEFAULT now());
        REVOKE ALL ON public.n8n_schema_installation FROM PUBLIC;
        """
    sql += "\\endif\n"
    sql += "DO $$ BEGIN IF NOT pg_has_role(current_user,(SELECT relowner FROM pg_class WHERE oid='public.n8n_schema_installation'::regclass),'USAGE') THEN RAISE EXCEPTION 'Schema owner required'; END IF; END $$;\n"
    if module != "core":
        core_digest = source_digest(CORE, manifest)
        sql += f"DO $$ BEGIN IF NOT EXISTS(SELECT FROM n8n_schema_installation WHERE module='core' AND version=1 AND source_sha256={literal(core_digest)} AND structure_md5=({fingerprint(manifest['objects']['core'])})) THEN RAISE EXCEPTION 'Core installation required'; END IF; END $$;\n"
    sql += f"SELECT EXISTS(SELECT FROM n8n_schema_installation WHERE module={literal(module)}) AS installed \\gset\n\\if :installed\n"
    sql += f"""DO $$ BEGIN IF NOT EXISTS(SELECT FROM n8n_schema_installation
      WHERE module={literal(module)} AND version=1 AND source_sha256={literal(digest)}
      AND structure_md5=({fp})) THEN RAISE EXCEPTION 'Version/source/schema mismatch; no automatic overwrite'; END IF; END $$;
    """
    sql += "\\else\n"
    if check:
        sql += "DO $$ BEGIN RAISE EXCEPTION 'Requested module is missing'; END $$;\n"
    else:
        sql += "SET LOCAL check_function_bodies=off;\n" + "\n".join(body(p) for p in files) + "\n"
        if module == "core":
            sql += "REVOKE EXECUTE ON FUNCTION public.commit_ai_feedback(uuid,text,text,text,text,text,text,timestamptz), public.record_libredesk_sent_v2(text,text,text,text,text,timestamptz,text,text),public.finish_libredesk_sent_v2(text,text) FROM PUBLIC;\n"
            # Existing triggers continue to run; runtime accounts must not attach
            # trusted trigger functions to attacker-controlled table row types.
            guards = ["learning_case_feedback_guard_v11", "learning_case_review_separation_v11",
                      "learning_case_write_guard_v11", "learning_review_activation_guard",
                      "learning_review_audit_guard", "learning_review_original_guard"]
            sql += "REVOKE EXECUTE ON FUNCTION " + ",".join("public." + f + "()" for f in guards) + " FROM PUBLIC;\n"
        sql += f"INSERT INTO n8n_schema_installation(module,version,source_sha256,structure_md5) VALUES ({literal(module)},1,{literal(digest)},({fp}));\n"
    sql += "\\endif\n"
    # Deliberately no data, roles, passwords, owner changes or implicit upgrades.
    sql += "SELECT 'Schema version and structure verified';\n" + ("ROLLBACK;\n" if check else "COMMIT;\n")
    return sql


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--module", choices=["core", "knowledge"], default="core")
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--emit-sql", action="store_true", help="Only emit credential-free SQL; do not connect")
    args = parser.parse_args()
    sql = script(args.module, args.check)
    if args.emit_sql:
        print(sql)
        return 0
    return run(sql)


if __name__ == "__main__":
    sys.exit(main())
