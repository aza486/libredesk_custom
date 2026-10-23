// Fit an ordered prefix of whole badges, reserving the indicator only when needed.
export function visibleTagCount(widths, available, indicator, gap) {
  const total = widths.reduce((sum, width) => sum + width, 0) + Math.max(0, widths.length - 1) * gap
  if (total <= available) return widths.length
  let used = 0
  let count = 0
  for (const width of widths) {
    if (used + width + gap + indicator > available) break
    used += width + gap
    count++
  }
  return count
}
