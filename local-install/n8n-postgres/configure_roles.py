#!/usr/bin/env python3
"""Bind pre-created restricted principals; never read or set passwords."""
import argparse
import re
import sys
from install import body, literal, run


def identifier(value):
    if not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", value):
        raise ValueError("Use a simple lower-case PostgreSQL role name")
    return '"' + value + '"'


def script(workflow, case_publisher, knowledge=None, vector=None):
    names = [workflow, case_publisher, "libredesk_learning"]
    if knowledge:
        names.append(knowledge)
    if vector:
        names.append(vector)
    if len(set(names)) != len(names):
        raise ValueError("All runtime roles must differ")
    w, c = identifier(workflow), identifier(case_publisher)
    role_names = ",".join(map(literal, names))
    sql = f"""\\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(1869432081,1501);
SET LOCAL search_path=public;
DO $$ DECLARE r record; BEGIN
 IF (SELECT count(*) FROM pg_roles WHERE rolname IN ({role_names})) <> {len(names)} THEN
  RAISE EXCEPTION 'Create the separate restricted principals first; passwords are configured externally';
 END IF;
 FOR r IN SELECT * FROM pg_roles WHERE rolname IN ({role_names}) LOOP
  IF r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls
    OR EXISTS(SELECT FROM pg_auth_members WHERE member=r.oid)
    OR (has_schema_privilege(r.oid,'public','CREATE') AND r.rolname IS DISTINCT FROM {literal(vector or '')})
    OR EXISTS(SELECT FROM pg_class x JOIN pg_namespace n ON n.oid=x.relnamespace
      WHERE n.nspname='public' AND x.relowner=r.oid)
  THEN RAISE EXCEPTION 'Runtime principal already has ownership, membership or elevated privileges'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT FROM n8n_schema_installation WHERE module='core' AND version=1) THEN
  RAISE EXCEPTION 'Install core schema first'; END IF;
END $$;
"""
    # Existing Learning grant contracts, not a second review implementation.
    sql += body("learning/003_permissions.sql") + "\n"
    sql += body("learning/004_v10_permissions.sql").replace(':"publisher_role"', w) + "\n"
    sql += f"""
GRANT USAGE ON SCHEMA public TO {w},{c};
GRANT SELECT,INSERT,UPDATE ON public.cases,public.mail_events,public.ai_feedback,
 public.rag_items,public.agent_style_profiles,public.libredesk_sent_receipts_v2 TO {w};
GRANT SELECT,INSERT,UPDATE,DELETE ON public.learning_embedding_stage TO {w};
GRANT SELECT ON public.company_knowledge,public.rag_final_style_view,
 public.rag_verified_fact_view,public.rag_category_learning_view,public.rag_case_example_view TO {w};
GRANT SELECT(item_id) ON public.learning_reviews TO {w};
GRANT EXECUTE ON FUNCTION public.commit_ai_feedback(uuid,text,text,text,text,text,text,timestamptz),
 public.record_libredesk_sent_v2(text,text,text,text,text,timestamptz,text,text),
 public.finish_libredesk_sent_v2(text,text), public.learning_case_stage_v11(uuid) TO {w};
GRANT EXECUTE ON FUNCTION public.learning_case_review_v11(uuid,bigint,text,text,boolean,boolean,boolean,jsonb) TO libredesk_learning;
GRANT SELECT ON public.learning_case_publisher_candidates_v11 TO {c};
GRANT EXECUTE ON FUNCTION public.learning_case_publish_v11(uuid,bigint,public.vector) TO {c};
"""
    if knowledge:
        k = identifier(knowledge)
        sql += f"""DO $$ BEGIN IF NOT EXISTS(SELECT FROM n8n_schema_installation WHERE module='knowledge') THEN
          RAISE EXCEPTION 'Install optional knowledge schema first'; END IF; END $$;
        GRANT USAGE ON SCHEMA public TO {k};
        GRANT SELECT,INSERT,UPDATE ON public.kd_sources,public.kd_candidates,public.kd_audit,
          public.company_knowledge TO {k};
        GRANT USAGE,SELECT ON SEQUENCE public.kd_audit_id_seq TO {k};
        """
    if vector:
        v = identifier(vector)
        sql += f"""-- Explicit opt-in for n8n PGVector's mandatory CREATE TABLE IF NOT EXISTS.
        -- This is NOT an owner, review, publication or generic workflow grant.
        GRANT USAGE,CREATE ON SCHEMA public TO {v};
        GRANT SELECT ON public.rag_final_style_view,public.rag_verified_fact_view,
          public.rag_category_learning_view,public.rag_case_example_view TO {v};
        GRANT SELECT,INSERT,UPDATE,DELETE ON public.learning_embedding_stage TO {v};
        DO $$ BEGIN
          IF has_table_privilege({literal(vector)},'public.rag_items','UPDATE')
          OR has_table_privilege({literal(vector)},'public.ai_feedback','SELECT')
          OR has_table_privilege({literal(vector)},'public.learning_reviews','UPDATE')
          OR has_function_privilege({literal(vector)},'public.learning_case_publish_v11(uuid,bigint,public.vector)','EXECUTE')
          OR has_function_privilege({literal(vector)},'public.learning_case_review_v11(uuid,bigint,text,text,boolean,boolean,boolean,jsonb)','EXECUTE')
          OR EXISTS(SELECT FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
            WHERE n.nspname='public' AND p.prorettype='trigger'::regtype
              AND has_function_privilege({literal(vector)},p.oid,'EXECUTE'))
          THEN RAISE EXCEPTION 'Existing role grants violate review/publisher separation'; END IF;
        END $$;
        """
    # Fail rather than silently stripping preexisting privileges or expanding
    # existing trust boundaries. Catalog checks include effective PUBLIC grants.
    sql += f"""DO $$ BEGIN
 IF has_table_privilege({literal(case_publisher)},'public.rag_items','SELECT')
 OR has_table_privilege({literal(case_publisher)},'public.ai_feedback','SELECT')
 OR has_column_privilege('libredesk_learning','public.mail_events','raw_text','SELECT')
 OR has_table_privilege({literal(workflow)},'public.learning_reviews','UPDATE')
 OR has_column_privilege({literal(workflow)},'public.learning_reviews','decision','INSERT')
 OR has_function_privilege({literal(workflow)},'public.learning_case_review_v11(uuid,bigint,text,text,boolean,boolean,boolean,jsonb)','EXECUTE')
 OR has_function_privilege({literal(workflow)},'public.learning_case_publish_v11(uuid,bigint,public.vector)','EXECUTE')
 OR has_table_privilege({literal(workflow)},'public.n8n_schema_installation','UPDATE')
 THEN RAISE EXCEPTION 'Existing role grants violate review/publisher separation'; END IF;
END $$;
COMMIT;
SELECT 'Restricted role bindings verified';
"""
    return sql


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workflow-role", required=True)
    parser.add_argument("--case-publisher-role", required=True)
    parser.add_argument("--knowledge-role")
    parser.add_argument("--vector-role", help="Explicit separate opt-in principal for n8n native PGVector initialization")
    parser.add_argument("--emit-sql", action="store_true")
    args = parser.parse_args()
    sql = script(args.workflow_role, args.case_publisher_role, args.knowledge_role, args.vector_role)
    if args.emit_sql:
        print(sql)
        return 0
    return run(sql)


if __name__ == "__main__":
    sys.exit(main())
