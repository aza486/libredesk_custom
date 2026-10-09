// All editor and feedback writes share this queue and read the latest draft at
// execution time, so stale autosaves cannot resurrect deleted feedback metadata.
const chains = new Map()
export function syncServerDraft(api, store, uuid, type, isCurrent = () => true) {
  const key = `${uuid}:${type}`
  const previous = chains.get(key) || Promise.resolve()
  const task = previous
    .catch(() => {})
    .then(() => {
      if (!isCurrent()) throw new Error('Mitarbeitersitzung geändert')
      const draft = store.getDraft(uuid, type)
      return draft ? api.saveDraft(uuid, type, draft) : api.deleteDraft(uuid, type)
    })
  chains.set(key, task)
  task
    .finally(() => {
      if (chains.get(key) === task) chains.delete(key)
    })
    .catch(() => {})
  return task
}
