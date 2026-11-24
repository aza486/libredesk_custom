-- Structure only. No relation contents, sequence values, login secrets or ACLs.
SELECT jsonb_build_object(
 'columns',COALESCE((SELECT jsonb_object_agg(k,v) FROM (
  SELECT c.relname||'.'||a.attname k,jsonb_build_object('position',row_number() OVER(PARTITION BY c.oid ORDER BY a.attnum),
   'type',format_type(a.atttypid,a.atttypmod),'not_null',a.attnotnull,
   'default',pg_get_expr(d.adbin,d.adrelid),'identity',a.attidentity) v
  FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid
  JOIN pg_namespace n ON n.oid=c.relnamespace
  LEFT JOIN pg_attrdef d ON d.adrelid=c.oid AND d.adnum=a.attnum
  WHERE n.nspname='public' AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped) q),'{}'),
 'constraints',COALESCE((SELECT jsonb_object_agg(k,v) FROM (
  SELECT c.relname||'.'||p.conname k,pg_get_constraintdef(p.oid) v FROM pg_constraint p
  JOIN pg_class c ON c.oid=p.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='public') q),'{}'),
 'indexes',COALESCE((SELECT jsonb_object_agg(tablename||'.'||indexname,indexdef)
  FROM pg_indexes WHERE schemaname='public'),'{}'),
 'views',COALESCE((SELECT jsonb_object_agg(c.relname,jsonb_build_object('definition',pg_get_viewdef(c.oid),'options',c.reloptions))
  FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='v'),'{}'),
 'functions',COALESCE((SELECT jsonb_object_agg(p.proname||'('||pg_get_function_identity_arguments(p.oid)||')',pg_get_functiondef(p.oid))
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='public'
  AND NOT EXISTS(SELECT FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid AND d.deptype='e')),'{}'),
 'triggers',COALESCE((SELECT jsonb_object_agg(c.relname||'.'||t.tgname,jsonb_build_object('definition',pg_get_triggerdef(t.oid),'enabled',t.tgenabled))
  FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname='public' AND NOT t.tgisinternal),'{}'),
 'sequences',COALESCE((SELECT jsonb_object_agg(c.relname,jsonb_build_object('start',s.seqstart,'increment',s.seqincrement,
  'max',s.seqmax,'min',s.seqmin,'cache',s.seqcache,'cycle',s.seqcycle)) FROM pg_sequence s
  JOIN pg_class c ON c.oid=s.seqrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'),'{}'),
 'extensions',(SELECT jsonb_object_agg(extname,extversion) FROM pg_extension));
