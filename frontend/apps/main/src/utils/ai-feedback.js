import { reactive } from 'vue'
import { UUID_PATTERN, feedbackReasonBytes, isOriginalCustomerMessage } from './ai-note'
import { syncServerDraft } from './draft-sync'

// One optional feedback operation per conversation. No message sending API is
// available here: status checking and retries can never send a customer reply.
export function createAIFeedback({ api, drafts, userID }) {
  const records = reactive({})
  const busy = new Set()
  const hydrated = new Set()
  const persist = async (uuid) => {
    const record = records[uuid]
    if (record && record.agentID !== userID()) throw new Error('Mitarbeitersitzung geändert')
    const draft = drafts.getDraft(uuid, 'reply') || { content: '', meta: {} }
    const meta = { ...draft.meta }
    if (!record || record.phase === 'saved') delete meta.ai_feedback
    else meta.ai_feedback = JSON.parse(JSON.stringify(record))
    if (!draft.content && !Object.keys(meta).length) drafts.removeDraft(uuid, 'reply')
    else drafts.setDraft(uuid, 'reply', { ...draft, meta })
    const owner = userID()
    await syncServerDraft(api, drafts, uuid, 'reply', () => owner === userID())
  }
  const hydrate = (uuid) => {
    if (hydrated.has(uuid)) return
    hydrated.add(uuid)
    const r = drafts.getDraft(uuid, 'reply')?.meta?.ai_feedback
    if (
      r?.version !== 1 ||
      r.agentID !== userID() ||
      !UUID_PATTERN.test(r.originalUUID || '') ||
      !UUID_PATTERN.test(r.noteUUID || '') ||
      typeof r.reason !== 'string' ||
      feedbackReasonBytes(r.reason) > 8000 ||
      !['composing', 'waiting', 'uncertain', 'error', 'failed'].includes(r.phase) ||
      (r.outgoingUUID &&
        (typeof r.outgoingUUID !== 'string' || !UUID_PATTERN.test(r.outgoingUUID))) ||
      (r.echoID && (typeof r.echoID !== 'string' || r.echoID.length > 200)) ||
      (r.phase !== 'composing' && !r.outgoingUUID && !r.echoID)
    )
      return
    records[uuid] = { ...r }
  }
  const arm = async (uuid, note, reason) => {
    if (!UUID_PATTERN.test(note?.noteUUID || '') || !UUID_PATTERN.test(note?.originalUUID || ''))
      throw new Error('Ungültiger Notizbezug')
    if (feedbackReasonBytes(reason) > 8000) throw new Error('Die Begründung ist zu lang.')
    const existing = records[uuid]
    if (existing && !['composing', 'saved'].includes(existing.phase))
      throw new Error('Bitte zuerst das ausstehende Feedback abschließen.')
    records[uuid] = {
      version: 1,
      agentID: userID(),
      noteUUID: note.noteUUID,
      originalUUID: note.originalUUID,
      reason: reason.trim(),
      phase: 'composing',
      outgoingUUID: '',
      echoID: ''
    }
    await persist(uuid)
  }
  const prepare = async (uuid, echoID) => {
    const r = records[uuid]
    if (r?.phase !== 'composing') return null
    if (r.agentID !== userID()) throw new Error('Mitarbeitersitzung geändert')
    const previousDraft = drafts.getDraft(uuid, 'reply')
    // Persist the intent without a resendable editor/attachment snapshot.
    drafts.setDraft(uuid, 'reply', { content: '', meta: {} })
    r.phase = 'waiting'
    r.echoID = echoID
    try {
      await persist(uuid)
    } catch (e) {
      r.phase = 'composing'
      if (previousDraft) drafts.setDraft(uuid, 'reply', previousDraft)
      await persist(uuid).catch(() => {})
      throw e
    }
    return r.originalUUID
  }
  const queued = async (uuid, message) => {
    const r = records[uuid]
    if (
      !r ||
      !UUID_PATTERN.test(message?.uuid || '') ||
      message.meta?.ai_feedback_original_message_uuid !== r.originalUUID
    ) {
      if (r) r.phase = 'uncertain'
      await persist(uuid)
      return
    }
    r.outgoingUUID = message.uuid
    r.phase = 'waiting'
    await persist(uuid)
  }
  const uncertain = async (uuid) => {
    if (records[uuid]) {
      records[uuid].phase = 'uncertain'
      await persist(uuid)
    }
  }
  const commit = async (uuid, r) => {
    r.phase = 'saving'
    const receipt = (await api.saveAIFeedback(uuid, r.outgoingUUID, { feedback_reason: r.reason }))
      ?.data?.data
    if (r.agentID !== userID()) return
    if (receipt?.committed !== true || !UUID_PATTERN.test(receipt.id || ''))
      throw new Error('Commit nicht bestätigt')
    r.phase = 'saved'
    r.reason = ''
    try {
      await persist(uuid)
    } catch {
      /* Old persisted operation is safe to replay idempotently. */
    }
  }
  const check = async (uuid, retry = false) => {
    const r = records[uuid]
    if (
      !r ||
      r.agentID !== userID() ||
      busy.has(uuid) ||
      ['composing', 'saved'].includes(r.phase) ||
      (r.phase === 'error' && !retry)
    )
      return
    busy.add(uuid)
    try {
      if (retry && r.deliveryConfirmed === true) {
        await commit(uuid, r)
        return
      }
      // A lost queue response is recovered solely via the previously persisted
      // echo ID. Bounded pagination, no inference from text or timestamps.
      if (!r.outgoingUUID) {
        if (!r.echoID) throw new Error('Fehlender Versandbezug')
        let found = null
        for (let page = 1; page <= 10; page++) {
          const response = await api.getConversationMessages(uuid, { page, page_size: 100 })
          const result = response?.data?.data
          if (result?.total_pages > 10)
            throw new Error('Versandbezug außerhalb des begrenzten Suchbereichs')
          const messages = result?.results || []
          const matches = messages.filter((m) => m.meta?.echo_id === r.echoID)
          if (matches.length > 1 || (found && matches.length))
            throw new Error('Mehrdeutiger Versandbezug')
          if (matches.length) found = matches[0]
          if (page >= (result?.total_pages || 1)) break
        }
        if (!found) {
          r.phase = 'uncertain'
          return
        }
        r.outgoingUUID = found.uuid
        await persist(uuid)
      }
      const response = await api.getConversationMessage(uuid, r.outgoingUUID)
      const m = response?.data?.data
      if (
        m?.uuid !== r.outgoingUUID ||
        m.conversation_uuid !== uuid ||
        m.sender_id !== userID() ||
        m.type !== 'outgoing' ||
        m.sender_type !== 'agent' ||
        m.private ||
        m.meta?.ai_feedback_original_message_uuid !== r.originalUUID
      )
        throw new Error('Ungültiger Versandbezug')
      if (m.status === 'failed') {
        r.phase = 'failed'
        await persist(uuid)
        return
      }
      if (m.status !== 'sent') {
        r.phase = 'waiting'
        return
      }
      // Backend validates this again. Verify the original independently on resume.
      const original = (await api.getConversationMessage(uuid, r.originalUUID))?.data?.data
      if (!isOriginalCustomerMessage(original, uuid, r.originalUUID))
        throw new Error('Ungültige Kundennachricht')
      r.deliveryConfirmed = true
      await persist(uuid)
      await commit(uuid, r)
    } catch {
      r.phase = 'error'
      try {
        await persist(uuid)
      } catch {
        /* Keep the reason in memory and the prior server record. */
      }
    } finally {
      busy.delete(uuid)
    }
  }
  const reset = () => {
    Object.keys(records).forEach((key) => delete records[key])
    hydrated.clear()
  }
  const cancel = async (uuid) => {
    if (records[uuid]?.phase !== 'composing') return
    delete records[uuid]
    await persist(uuid)
  }
  return { records, hydrate, arm, prepare, queued, uncertain, check, cancel, persist, reset }
}
