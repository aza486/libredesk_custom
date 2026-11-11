import { computed, watch } from 'vue'

// Selection lives in the existing reply draft. Templates wrap the message at
// send time; never insert signature HTML into the editor itself.
export function useReplySignature({ conversation, inboxes, templates, messageType, isLoading,
  selection, hasStoredDraft, htmlContent, attachments }) {
  const fallbackID = computed(() =>
    templates.value.find(template => template.is_default)?.id || templates.value[0]?.id || null
  )
  const inbox = computed(() => inboxes.value.find(inbox => inbox.id === conversation.value?.inbox_id))
  const availableID = id => templates.value.some(template => template.id === id) ? id : fallbackID.value

  watch(
    [() => conversation.value?.uuid, () => inbox.value?.id,
      () => inbox.value?.default_signature_id, templates, isLoading, messageType, hasStoredDraft, selection],
    () => {
      if (!conversation.value?.uuid || conversation.value.inbox_channel !== 'email' ||
          messageType.value !== 'reply' || isLoading.value || !templates.value.length) return
      // Restored drafts (including legacy drafts with no signature metadata)
      // and manual decisions always win over asynchronously arriving defaults.
      if (hasStoredDraft.value || selection.value?.source === 'manual') return
      if (selection.value && (htmlContent.value || attachments.value?.length)) return
      const templateID = availableID(inbox.value?.default_signature_id)
      const inboxID = inbox.value?.id || conversation.value.inbox_id
      if (selection.value?.template_id === templateID && selection.value?.inbox_id === inboxID) return
      selection.value = { source: 'automatic', template_id: templateID, inbox_id: inboxID }
    },
    { immediate: true }
  )

  return computed({
    get: () => messageType.value === 'reply' && conversation.value?.inbox_channel === 'email'
      ? availableID(selection.value?.template_id) : null,
    set: id => {
      selection.value = { source: 'manual', template_id: id, inbox_id: conversation.value?.inbox_id }
    }
  })
}
