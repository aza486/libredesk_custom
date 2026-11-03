// Shared by pointer and keyboard reordering. Never mutate the displayed order
// until the server has accepted the complete scope-local permutation.
export function moveFolder(ids, id, targetIndex) {
  const currentIndex = ids.indexOf(id)
  if (currentIndex < 0 || targetIndex < 0 || targetIndex >= ids.length) return ids.slice()
  const next = ids.filter((value) => value !== id)
  next.splice(targetIndex, 0, id)
  return next
}
