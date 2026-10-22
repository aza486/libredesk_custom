import { ref, watch } from 'vue'
import { useDebounceFn, useEventListener } from '@vueuse/core'
import { useConversationStore } from '@main/stores/conversation'
import { MACRO_CONTEXT } from '@main/constants/conversation'
import { getTextFromHTML } from '@shared-ui/utils/string.js'
import api from '@main/api'
import { syncServerDraft } from '@main/utils/draft-sync'

const hasKeys = (obj, keys) => Boolean(obj) && keys.every(key => key in obj)

const validateMacroActions = (actions) => {
  if (!Array.isArray(actions)) return []
  return actions.filter(action =>
    hasKeys(action, ['type', 'value', 'display_value']) &&
    Array.isArray(action.value) &&
    Array.isArray(action.display_value)
  )
}

const validateAttachments = (attachments) => {
  if (!Array.isArray(attachments)) return []
  return attachments.filter(attachment =>
    hasKeys(attachment, ['id', 'size', 'uuid', 'filename', 'content_type'])
  )
}

const isDraftEmpty = (draft) => {
  const content = draft?.content || ''
  const hasText = getTextFromHTML(content).length > 0
  const hasInlineImage = /<img\b/i.test(content)
  const hasAttachments = draft?.meta?.attachments?.length > 0
  const hasMacroActions = draft?.meta?.macro_actions?.length > 0
  return !hasText && !hasInlineImage && !hasAttachments && !hasMacroActions && !draft?.meta?.ai_feedback
}

const draftKey = (uuid, type) => `${uuid}::${type}`

const metaSignature = (meta) =>
  JSON.stringify({
    ai_feedback: meta?.ai_feedback || null,
    ai_response: meta?.ai_response || null,
    macro_id: meta?.macro_id || 0,
    macro_actions: meta?.macro_actions || [],
    attachments: (meta?.attachments || []).map(a => a.uuid)
  })

const sameDraft = (a, b) => {
  if (isDraftEmpty(a) && isDraftEmpty(b)) return true
  return (a?.content || '') === (b?.content || '') && metaSignature(a?.meta) === metaSignature(b?.meta)
}

export function useDraftManager (conversationUUID, messageType, uploadedFiles = null) {
  const conversationStore = useConversationStore()
  const aiResponse = ref(null)
  const htmlContent = ref('')
  const textContent = ref('')
  const isLoading = ref(false)
  const loadedAttachments = ref([])
  const loadedMacroActions = ref([])
  const loadedMacroID = ref(0)

  // Live-key guard: the editor is transiently empty during open/switch and must not clobber a stored draft.
  const loadedKey = ref(null)
  const currentKey = () => draftKey(conversationUUID.value, messageType.value)

  const buildDraft = (uuid = conversationUUID.value, type = messageType.value) => {
    const feedback = conversationStore.getDraft(uuid, 'reply')?.meta?.ai_feedback
    const meta = type === 'reply' && feedback ? { ai_feedback: feedback } : {}
    if (type === 'reply' && aiResponse.value) meta.ai_response = aiResponse.value
    const macro = conversationStore.getMacro(MACRO_CONTEXT.REPLY)
    const macroActions = macro?.actions || []
    if (macro?.id > 0) meta.macro_id = macro.id
    if (macroActions.length > 0) meta.macro_actions = macroActions
    if (uploadedFiles?.value?.length > 0) {
      meta.attachments = uploadedFiles.value.map(file => ({
        id: file.id,
        url: file.url,
        size: file.size,
        uuid: file.uuid,
        filename: file.filename,
        content_type: file.content_type,
        disposition: file.disposition
      }))
    }
    return { content: htmlContent.value, meta }
  }

  const applyDraft = (draft) => {
    aiResponse.value = draft?.meta?.ai_response || null
    htmlContent.value = draft?.content || ''
    textContent.value = ''
    loadedAttachments.value = validateAttachments(draft?.meta?.attachments)
    loadedMacroActions.value = validateMacroActions(draft?.meta?.macro_actions)
    loadedMacroID.value = Number(draft?.meta?.macro_id) > 0 ? Number(draft.meta.macro_id) : 0
  }

  const load = async (uuid, type) => {
    isLoading.value = true
    loadedKey.value = null
    // Prefetch may still be in flight on a fresh page load; reading the store now would apply an empty draft.
    const before = htmlContent.value
    await conversationStore.draftsReady
    // If the user typed while drafts were still loading, keep their input instead of clobbering it.
    if (htmlContent.value === before) {
      applyDraft(conversationStore.getDraft(uuid, type))
    }
    loadedKey.value = draftKey(uuid, type)
    isLoading.value = false
  }

  // Update the store synchronously; server sync runs in the background, never awaited.
  const save = (uuid, type) => {
    const draft = buildDraft(uuid, type)
    if (sameDraft(draft, conversationStore.getDraft(uuid, type))) return
    if (isDraftEmpty(draft)) {
      conversationStore.removeDraft(uuid, type)
      syncServerDraft(api, conversationStore, uuid, type).catch(() => {})
    } else {
      conversationStore.setDraft(uuid, type, draft)
      syncServerDraft(api, conversationStore, uuid, type).catch(() => {})
    }
  }

  const debouncedSave = useDebounceFn(() => {
    if (loadedKey.value === currentKey()) save(conversationUUID.value, messageType.value)
  }, 500)

  const clearDraft = (uuid = conversationUUID.value, type = messageType.value) => {
    if (!uuid) return
    const feedback = type === 'reply' && conversationStore.getDraft(uuid, type)?.meta?.ai_feedback
    if (feedback) conversationStore.setDraft(uuid, type, { content: '', meta: { ai_feedback: feedback } })
    else conversationStore.removeDraft(uuid, type)
    syncServerDraft(api, conversationStore, uuid, type).catch(() => {})
    if (uuid === conversationUUID.value && type === messageType.value) applyDraft(null)
  }

  const watchSources = [
    htmlContent,
    aiResponse,
    textContent,
    () => conversationStore.macros[MACRO_CONTEXT.REPLY]
  ]
  if (uploadedFiles) watchSources.push(uploadedFiles)

  watch(
    watchSources,
    () => {
      if (!isLoading.value && loadedKey.value === currentKey()) debouncedSave()
    },
    { deep: true }
  )

  // Serialize switches: a rapid A->B->A must not interleave save and load.
  let chain = Promise.resolve()
  watch(
    [conversationUUID, messageType],
    ([uuid, type], oldVals) => {
      const [prevUuid, prevType] = oldVals || []
      chain = chain.then(async () => {
        if (prevUuid && loadedKey.value === draftKey(prevUuid, prevType)) save(prevUuid, prevType)
        if (uuid) await load(uuid, type)
        else applyDraft(null)
      }).catch(() => {})
    },
    { immediate: true }
  )

  useEventListener(document, 'visibilitychange', () => {
    if (document.visibilityState === 'hidden' && conversationUUID.value && loadedKey.value === currentKey()) {
      save(conversationUUID.value, messageType.value)
    }
  })

  return {
    htmlContent,
    aiResponse,
    textContent,
    isLoading,
    clearDraft,
    loadedAttachments,
    loadedMacroActions,
    loadedMacroID
  }
}
