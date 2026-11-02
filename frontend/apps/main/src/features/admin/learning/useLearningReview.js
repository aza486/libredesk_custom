import { ref, watch, onUnmounted } from 'vue'
import api from '@main/api'

export function useLearningReview(user) {
  const tab = ref('pending'),
    type = ref(''),
    status = ref(''),
    page = ref(1)
  const items = ref([]),
    hasMore = ref(false),
    loading = ref(false),
    busy = ref(false),
    error = ref('')
  const selected = ref(null)
  let generation = 0,
    selectionGeneration = 0
  const allowed = () => user.hasAdminRole || user.can('ai_learning:manage')
  const refresh = async () => {
    selectionGeneration++
    const current = ++generation
    items.value = []
    selected.value = null
    error.value = ''
    if (!allowed()) {
      loading.value = false
      return
    }
    loading.value = true
    try {
      const response = await api.getLearningItems({
        tab: tab.value,
        type: type.value,
        status: status.value,
        page: page.value
      })
      if (current !== generation) return
      items.value = response.data.data.items || []
      hasMore.value = response.data.data.has_more
    } catch {
      if (current === generation) error.value = 'Lerninhalte konnten nicht geladen werden.'
    } finally {
      if (current === generation) loading.value = false
    }
  }
  const open = async (item) => {
    if (busy.value || !allowed()) return
    const token = ++selectionGeneration
    selected.value = null
    error.value = ''
    try {
      const data = (await api.getLearningItem(item.id)).data.data
      if (token !== selectionGeneration) return
      selected.value = data
    } catch {
      if (token === selectionGeneration) error.value = 'Details konnten nicht geladen werden.'
    }
  }
  const act = async (action) => {
    if (
      !allowed() ||
      busy.value ||
      !selected.value ||
      !['learn', 'discard'].includes(action) ||
      !selected.value.review_ready ||
      selected.value.decision !== 'pending'
    )
      return
    busy.value = true
    error.value = ''
    const token = selectionGeneration
    try {
      const result = await api.reviewLearningItem(selected.value.id, {
        action,
        revision: selected.value.revision
      })
      if (token !== selectionGeneration) return
      selected.value = null
      await refresh()
      return result.data.data.id
    } catch (e) {
      if (token === selectionGeneration)
        error.value =
          e.response?.status === 409
            ? 'Der Eintrag wurde geändert oder die Aktion ist nicht zulässig. Bitte neu öffnen.'
            : 'Entscheidung nicht gespeichert. Bitte Verbindung prüfen.'
    } finally {
      busy.value = false
    }
  }
  watch([tab, type, status], () => {
    page.value = 1
  })
  watch(
    [tab, type, status, page, () => user.userID],
    () => {
      selectionGeneration++
      refresh()
    },
    { immediate: true }
  )
  onUnmounted(() => {
    generation++
    selectionGeneration++
  })
  return {
    tab,
    type,
    status,
    page,
    items,
    hasMore,
    loading,
    busy,
    error,
    selected,
    refresh,
    open,
    act,
    allowed
  }
}
