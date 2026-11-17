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
  const caseChecks = ref({ privacy: false, generalization: false, quality: false })
  watch(
    selected,
    () => {
      caseChecks.value = { privacy: false, generalization: false, quality: false }
    },
    { flush: 'sync' }
  )
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
  const actCase = async (action) => {
    if (
      !allowed() ||
      busy.value ||
      !selected.value ||
      selected.value.content?.source_type !== 'case_example' ||
      !['approve', 'reject', 'block'].includes(action)
    )
      return
    if (
      action === 'approve' &&
      (selected.value.decision !== 'staged' ||
        selected.value.provenance?.case_source_status !== 'staged' ||
        !caseChecks.value.privacy ||
        !caseChecks.value.generalization ||
        !caseChecks.value.quality)
    )
      return
    busy.value = true
    error.value = ''
    const token = selectionGeneration
    try {
      const result = await api.reviewLearningCase(selected.value.id, {
        action,
        revision: selected.value.revision,
        checks: { ...caseChecks.value }
      })
      if (token !== selectionGeneration) return
      selected.value = null
      await refresh()
      return result.data.data.id
    } catch (e) {
      if (token === selectionGeneration)
        error.value =
          e.response?.status === 409
            ? 'Der Fall wurde geändert oder die Aktion ist nicht zulässig. Bitte neu öffnen.'
            : e.response?.status === 400
              ? 'Der Fall hat die Datenschutz- oder Qualitätsprüfung nicht bestanden. Bitte prüfen oder sperren.'
              : 'Fallentscheidung nicht gespeichert. Bitte Verbindung prüfen.'
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
    caseChecks,
    actCase,
    refresh,
    open,
    act,
    allowed
  }
}
