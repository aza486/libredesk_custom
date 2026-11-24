-- Existing workflow function contracts; no production data or account identities.
CREATE FUNCTION public.commit_ai_feedback(p_id uuid, p_message_uuid text, p_conversation_uuid text, p_outgoing_uuid text, p_agent_id text, p_human_answer text, p_feedback_reason text, p_sent_at timestamp with time zone) RETURNS TABLE(id text, committed boolean, result text)
    LANGUAGE plpgsql
    AS $$
DECLARE
    v_event mail_events%ROWTYPE;
    v_existing ai_feedback%ROWTYPE;
    v_reason TEXT := NULLIF(BTRIM(p_feedback_reason), '');
    v_sent_at TIMESTAMP := p_sent_at AT TIME ZONE 'UTC';
BEGIN
    IF p_id IS NULL
       OR NULLIF(BTRIM(p_message_uuid), '') IS NULL
       OR NULLIF(BTRIM(p_conversation_uuid), '') IS NULL
       OR NULLIF(BTRIM(p_outgoing_uuid), '') IS NULL
       OR NULLIF(BTRIM(p_agent_id), '') IS NULL
       OR NULLIF(BTRIM(p_human_answer), '') IS NULL
       OR p_sent_at IS NULL
       OR LENGTH(COALESCE(p_feedback_reason, '')) > 8000
    THEN
        RETURN QUERY SELECT p_id::TEXT, FALSE, 'invalid_input'::TEXT;
        RETURN;
    END IF;

    SELECT *
    INTO v_event
    FROM mail_events m
    WHERE m.message_uuid = p_message_uuid
      AND m.conversation_uuid = p_conversation_uuid
      AND NULLIF(BTRIM(m.suggested_message), '') IS NOT NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RETURN QUERY SELECT p_id::TEXT, FALSE, 'mail_event_not_found'::TEXT;
        RETURN;
    END IF;

    SELECT *
    INTO v_existing
    FROM ai_feedback f
    WHERE f.id = p_id
       OR f.outgoing_message_uuid = p_outgoing_uuid
    LIMIT 1;

    IF FOUND THEN
        IF v_existing.id = p_id
           AND v_existing.mail_event_message_uuid = p_message_uuid
           AND v_existing.outgoing_message_uuid = p_outgoing_uuid
           AND v_existing.agent_id = p_agent_id
           AND v_existing.human_answer = p_human_answer
           AND v_existing.feedback_reason IS NOT DISTINCT FROM v_reason
        THEN
            RETURN QUERY SELECT p_id::TEXT, TRUE, 'already_committed'::TEXT;
        ELSE
            RETURN QUERY SELECT p_id::TEXT, FALSE, 'conflict'::TEXT;
        END IF;
        RETURN;
    END IF;

    INSERT INTO ai_feedback (
        id,
        mail_event_message_uuid,
        outgoing_message_uuid,
        agent_id,
        human_answer,
        feedback_reason,
        feedback_type,
        quality_status
    )
    VALUES (
        p_id,
        p_message_uuid,
        p_outgoing_uuid,
        p_agent_id,
        p_human_answer,
        v_reason,
        NULL,
        'pending'
    );

    UPDATE mail_events
    SET
        sent_message = CASE
            WHEN sent_at IS NULL OR sent_at <= v_sent_at
            THEN p_human_answer
            ELSE sent_message
        END,
        response_source = CASE
            WHEN sent_at IS NULL OR sent_at <= v_sent_at
            THEN 'human'
            ELSE response_source
        END,
        agent_id = CASE
            WHEN sent_at IS NULL OR sent_at <= v_sent_at
            THEN p_agent_id
            ELSE agent_id
        END,
        sent_at = GREATEST(
            COALESCE(sent_at, v_sent_at),
            v_sent_at
        ),
        status_updated_at = NOW()
    WHERE message_uuid = p_message_uuid;

    RETURN QUERY SELECT p_id::TEXT, TRUE, 'created'::TEXT;
END;
$$;

CREATE FUNCTION public.finish_libredesk_sent_v2(p_outgoing_message_uuid text, p_outcome text) RETURNS TABLE(finalized boolean, outgoing_message_uuid text, outcome text)
    LANGUAGE plpgsql
    AS $$
DECLARE
  v_state text;
BEGIN
  IF p_outcome NOT IN ('resolved','superseded','invalid_completion') OR p_outcome IS NULL THEN
    RETURN QUERY SELECT false,p_outgoing_message_uuid,'invalid_resolve_outcome'::text;
    RETURN;
  END IF;
  SELECT r.resolve_state INTO v_state
    FROM public.libredesk_sent_receipts_v2 AS r
    WHERE r.outgoing_message_uuid=p_outgoing_message_uuid FOR UPDATE;
  IF NOT FOUND THEN
    RETURN QUERY SELECT false,p_outgoing_message_uuid,'receipt_not_found'::text;
    RETURN;
  END IF;
  IF v_state='pending' THEN
    UPDATE public.libredesk_sent_receipts_v2 AS r
    SET resolve_state=p_outcome,finalized_at=now()
    WHERE r.outgoing_message_uuid=p_outgoing_message_uuid;
    RETURN QUERY SELECT true,p_outgoing_message_uuid,p_outcome;
  ELSIF v_state=p_outcome THEN
    RETURN QUERY SELECT true,p_outgoing_message_uuid,v_state;
  ELSE
    RETURN QUERY SELECT false,p_outgoing_message_uuid,'terminal_outcome_conflict'::text;
  END IF;
END;
$$;

CREATE FUNCTION public.record_libredesk_sent_v2(p_conversation_uuid text, p_source_message_uuid text, p_outgoing_message_uuid text, p_response_source text, p_sent_message text, p_sent_at timestamp with time zone, p_agent_id text, p_expected_incoming_version text) RETURNS TABLE(committed boolean, outgoing_message_uuid text, needs_resolve boolean, outcome text, conversation_uuid text, expected_incoming_version text)
    LANGUAGE plpgsql
    AS $_$
DECLARE
  v_hash text;
  v_ins_count integer;
  v_existing_hash text;
  v_state text;
  v_latest_uuid text;
BEGIN
  IF nullif(trim(p_conversation_uuid),'') IS NULL
    OR nullif(trim(p_source_message_uuid),'') IS NULL
    OR nullif(trim(p_outgoing_message_uuid),'') IS NULL
    OR p_source_message_uuid = p_outgoing_message_uuid
    OR (p_response_source IS NOT NULL AND p_response_source NOT IN ('ai','ai_adapted','human'))
    OR p_sent_message IS NULL -- empty-string is allowed for attachments
    OR p_sent_at IS NULL
    OR p_expected_incoming_version IS NULL
    OR p_expected_incoming_version !~ '^[1-9][0-9]{0,18}$'
  THEN
    RETURN QUERY SELECT false,p_outgoing_message_uuid,false,'invalid_payload'::text,p_conversation_uuid,p_expected_incoming_version;
    RETURN;
  END IF;

  -- Source row lock serializes multiple outgoing replies to this incoming email.
  PERFORM 1 FROM public.mail_events AS m
  WHERE m.message_uuid=p_source_message_uuid
    AND m.conversation_uuid=p_conversation_uuid
    AND m.direction='inbound'
  FOR UPDATE;
  IF NOT FOUND THEN
    RETURN QUERY SELECT false,p_outgoing_message_uuid,false,'source_not_found'::text,p_conversation_uuid,p_expected_incoming_version;
    RETURN;
  END IF;

  v_hash:=md5(jsonb_build_object(
    'conversation_uuid',p_conversation_uuid,
    'source_message_uuid',p_source_message_uuid,
    'outgoing_message_uuid',p_outgoing_message_uuid,
    'response_source',p_response_source,
    'sent_message',p_sent_message,
    'sent_at',to_char(p_sent_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'agent_id',p_agent_id,
    'expected_incoming_version',p_expected_incoming_version
  )::text);

  INSERT INTO public.libredesk_sent_receipts_v2 AS r
    (outgoing_message_uuid,source_message_uuid,conversation_uuid,response_source,
     agent_id,expected_incoming_version,sent_at,payload_hash)
  VALUES(p_outgoing_message_uuid,p_source_message_uuid,p_conversation_uuid,p_response_source,
     p_agent_id,p_expected_incoming_version,p_sent_at,v_hash)
  ON CONFLICT ON CONSTRAINT libredesk_sent_receipts_v2_pkey DO NOTHING;
  GET DIAGNOSTICS v_ins_count=ROW_COUNT;

  IF v_ins_count=0 THEN
    SELECT r.payload_hash,r.resolve_state INTO v_existing_hash,v_state
    FROM public.libredesk_sent_receipts_v2 AS r
    WHERE r.outgoing_message_uuid=p_outgoing_message_uuid;
    IF v_existing_hash IS DISTINCT FROM v_hash THEN
      RETURN QUERY SELECT false,p_outgoing_message_uuid,false,'duplicate_payload_mismatch'::text,p_conversation_uuid,p_expected_incoming_version;
      RETURN;
    END IF;
  ELSE
    v_state:='pending';
  END IF;

  -- Update the latest outbound response per original inbound mail only.
  SELECT r.outgoing_message_uuid INTO v_latest_uuid
  FROM public.libredesk_sent_receipts_v2 AS r
  WHERE r.source_message_uuid=p_source_message_uuid
  ORDER BY r.sent_at DESC,r.outgoing_message_uuid DESC LIMIT 1;
  IF v_latest_uuid=p_outgoing_message_uuid AND v_ins_count=1 THEN
    UPDATE public.mail_events AS m SET
      sent_message=p_sent_message,
      sent_at=p_sent_at AT TIME ZONE 'UTC',
      status='sent',
      processing_stage='complete',
      status_updated_at=now() AT TIME ZONE 'UTC',
      response_source=p_response_source,
      agent_id=COALESCE(p_agent_id,m.agent_id)
    WHERE m.message_uuid=p_source_message_uuid
      AND m.conversation_uuid=p_conversation_uuid;
    -- Do not change faq_candidate: this is Learning/Review-owned state.
  END IF;

  RETURN QUERY SELECT true,p_outgoing_message_uuid,(v_state='pending'),
    (CASE WHEN v_ins_count=0 THEN 'duplicate_accepted' ELSE 'recorded' END)::text,
    p_conversation_uuid,p_expected_incoming_version;
END;
$_$;
