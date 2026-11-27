import { onBeforeUnmount, ref, watch } from 'vue'
import api from '@main/api'

// Deliberately component-local and ephemeral: never localStorage or a global
// content cache. Every insertion and edit retrieves fresh, authorized content.
export function useMessageTemplates(context, identity) {
  const templates = ref([])
  const canSupport = ref(false)
  const loading = ref(false)
  const error = ref('')
  const accessDenied = ref(false)
  let generation = 0
  const clear = () => {
    generation++
    templates.value = []
    canSupport.value = false
    loading.value = false
    error.value = ''
    accessDenied.value = false
  }
  const load = async () => {
    const request = ++generation
    templates.value = []
    canSupport.value = false
    error.value = ''
    accessDenied.value = false
    loading.value = true
    try {
      const { data } = await api.listMessageTemplates(context())
      if (request !== generation) return
      templates.value = data.data.templates.map(({ content: _content, ...item }) => item)
      canSupport.value = data.data.can_customer_support
      return true
    } catch (err) {
      if (request === generation) {
        error.value = templateError(err)
        accessDenied.value = [401, 403, 404].includes(err?.response?.status)
      }
      return false
    } finally {
      if (request === generation) loading.value = false
    }
  }
  watch([context, identity], clear)
  onBeforeUnmount(clear)
  return { templates, canSupport, loading, error, accessDenied, load, clear }
}
export function templateError(error) {
  return (
    error?.response?.data?.message ||
    'Die Vorlagen konnten nicht geladen oder gespeichert werden. Bitte erneut versuchen.'
  )
}
