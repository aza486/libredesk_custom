import { onBeforeUnmount, onBeforeUpdate, onMounted, onUpdated } from 'vue'

// Measure the actual scrolling viewport, not the page or the sidebar shell.
export function useConversationListPosition(container, store) {
  let renderedKey = store.listKey
  let renderedRevision = store.listRevision
  let resizeObserver

  function rows() {
    return Array.from(container.value?.querySelectorAll('[data-conversation-id]') || [])
  }

  function saveListPosition() {
    const el = container.value
    if (!el?.clientHeight || !renderedKey || renderedRevision !== store.listRevision) return
    const items = rows()
    if (!items.length) return
    const top = el.getBoundingClientRect().top + el.clientTop
    const measured = items.map((row) => {
      const bounds = row.getBoundingClientRect()
      return { id: row.dataset.conversationId, offset: bounds.top - top, bottom: bounds.bottom - top }
    })
    const firstVisible = measured.findIndex((row) => row.bottom > 0)
    const anchor = Math.max(0, firstVisible)
    // Retain neighbours and their old offsets if the anchor leaves the filter.
    const candidates = measured
      .map((row, index) => ({ ...row, index, distance: Math.abs(index - anchor) }))
      .sort((a, b) => a.distance - b.distance || b.index - a.index)
    store.listPositions.set(renderedKey, { scrollTop: el.scrollTop, candidates })
  }

  function restoreListPosition() {
    const el = container.value
    if (!el) return
    renderedKey = store.listKey
    renderedRevision = store.listRevision
    if (!el.clientHeight) return // A hidden mobile list is restored when shown.
    const saved = store.listPositions.get(renderedKey)
    if (!saved) {
      el.scrollTop = 0
      return
    }
    const items = rows()
    if (!items.length) return // Wait for a rendered response, retaining the anchor.
    const byID = new Map(items.map((row) => [row.dataset.conversationId, row]))
    const anchor = saved.candidates.find((candidate) => byID.has(candidate.id))
    if (anchor) {
      const top = el.getBoundingClientRect().top + el.clientTop
      el.scrollTop += byID.get(anchor.id).getBoundingClientRect().top - top - anchor.offset
    } else {
      el.scrollTop = saved.scrollTop
    }
  }

  // Vue runs these around the DOM patch, in the same render cycle before paint.
  onBeforeUpdate(saveListPosition)
  onUpdated(restoreListPosition)
  onMounted(() => {
    restoreListPosition()
    // v-show can reveal the mobile list after its own update hook has run.
    // ResizeObserver restores it as soon as the viewport has layout again.
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(restoreListPosition)
      resizeObserver.observe(container.value)
    }
  })
  onBeforeUnmount(() => {
    saveListPosition()
    resizeObserver?.disconnect()
  })

  return { saveListPosition }
}
