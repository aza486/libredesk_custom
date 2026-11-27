import { ref, watch, onUnmounted } from 'vue'

const fields = ['to', 'cc', 'bcc']

// A form address is only a suggestion. Track deliberate edits separately from
// programmatic defaults; an intentionally empty field is still an edit.
export function useFormRecipient(conversation, recipients, load, options = {}) {
  const state = ref(null)
  const draft = options.draft || ref(null)
  const edited = { to: false, cc: false, bcc: false }
  let applying = false
  let persisting = false
  let version = 0
  let timer
  const write = (values) => {
    applying = true
    try { fields.forEach((key, i) => { if (key in values) recipients[i].value = values[key] }) }
    finally { applying = false }
  }
  const persist = () => {
    if (Object.values(edited).some(Boolean)) {
      persisting = true
      draft.value = { ...Object.fromEntries(fields.map((key, i) => [key, recipients[i].value])), edited: { ...edited } }
      persisting = false
    }
  }
  const suggest = () => {
    if (!state.value?.managed || edited.to) return
    // Once a default has been shown, an outage doesn't erase that value either.
    if (!state.value.blocked && state.value.email) {
      write({ to: state.value.email })
      persist()
    }
  }
  const restore = (saved) => {
    if (!saved) return
    const values = {}
    fields.forEach(key => {
      if (!edited[key] && typeof saved[key] === 'string') {
        values[key] = saved[key]
        edited[key] = saved.edited?.[key] === true
      }
    })
    write(values)
    suggest()
  }
  const reset = () => {
    fields.forEach(key => { edited[key] = false })
    if (state.value?.managed) { write({ to: '', cc: '', bcc: '' }); suggest() }
  }

  watch(() => conversation.value?.uuid, () => {
    version++
    clearTimeout(timer)
    state.value = conversation.value?.reply_recipient || null
    reset()
    restore(draft.value)
  }, { immediate: true, flush: 'sync' })

  watch([() => conversation.value?.uuid, () => conversation.value?.reply_recipient], ([uuid, initial]) => {
    const requestVersion = ++version
    clearTimeout(timer)
    const wasManaged = state.value?.managed
    state.value = initial || null
    if (initial?.managed && !wasManaged) {
      write(Object.fromEntries(fields.filter(key => !edited[key]).map(key => [key, ''])))
      restore(draft.value)
    }
    suggest()
    if (!uuid || !initial?.managed) return
    const refresh = async () => {
      try {
        const next = await load(uuid)
        if (requestVersion !== version) return
        state.value = next?.managed ? next : { managed: true, blocked: true, email: '' }
        suggest()
      } catch {
        if (requestVersion !== version) return
        state.value = { managed: true, blocked: true, email: '' }
      }
      if (requestVersion === version) timer = setTimeout(refresh, 5000)
    }
    timer = setTimeout(refresh, 5000)
  }, { immediate: true })

  watch(draft, saved => { if (!persisting) restore(saved) }, { flush: 'sync' })
  recipients.forEach((recipient, i) => {
    watch(recipient, () => {
      if (applying || !state.value?.managed) return
      edited[fields[i]] = true
      persist()
    }, { flush: 'sync' })
  })
  const markEdited = (key) => {
    if (!fields.includes(key)) return
    edited[key] = true
    if (state.value?.managed) persist()
  }
  onUnmounted(() => { version++; clearTimeout(timer) })
  return { state, reset, markEdited }
}
