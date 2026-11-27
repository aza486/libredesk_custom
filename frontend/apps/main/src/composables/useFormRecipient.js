import { ref, watch, onUnmounted } from 'vue'

// The server supplies classification; never infer customers from message text.
export function useFormRecipient(conversation, recipients, load) {
  const state = ref(null)
  let version = 0
  let timer
  watch([() => conversation.value?.uuid, () => conversation.value?.reply_recipient], ([uuid, initial]) => {
    const requestVersion = ++version
    clearTimeout(timer)
    state.value = initial || null
    if (!uuid || !initial?.managed) return
    const refresh = async () => {
      try {
        const next = await load(uuid)
        if (requestVersion !== version) return
        state.value = next?.managed ? next : { managed: true, blocked: true, email: '' }
      } catch {
        if (requestVersion !== version) return
        state.value = { managed: true, blocked: true, email: '' }
      }
      if (requestVersion === version) timer = setTimeout(refresh, 5000)
    }
    timer = setTimeout(refresh, 5000)
  }, { immediate: true })
  watch([state, ...recipients], () => {
    if (!state.value?.managed) return
    const values = [state.value.blocked ? '' : state.value.email, '', '']
    recipients.forEach((field, i) => { if (field.value !== values[i]) field.value = values[i] })
  }, { immediate: true, flush: 'sync' })
  onUnmounted(() => { version++; clearTimeout(timer) })
  return state
}
