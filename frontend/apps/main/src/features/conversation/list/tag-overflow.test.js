import { expect, test } from 'vitest'
import { visibleTagCount } from './tag-overflow'

test('reserves overflow space, keeps whole badges and restores all tags when they fit', () => {
  expect(visibleTagCount([], 0, 16, 4)).toBe(0)
  for (const [width, count] of [
    [30, 0],
    [50, 1],
    [89, 1],
    [94, 2],
    [127, 2],
    [128, 3],
    [500, 3]
  ]) {
    expect(visibleTagCount([30, 40, 50], width, 16, 4)).toBe(count)
  }
  expect(visibleTagCount([500, 20, 20], 100, 16, 4)).toBe(0)
  expect(visibleTagCount([30], 30, 16, 4)).toBe(1)
  expect(visibleTagCount([30], 29, 16, 4)).toBe(0)
})

test('handles ten tags, fractional measurements and repeated shrink/grow cycles', () => {
  const widths = Array(10).fill(20.5)
  for (const available of [300, 80, 160, 20, 300, 50, 300]) {
    const count = visibleTagCount(widths, available, 15.5, 4)
    const used = count * 20.5 + Math.max(0, count - 1) * 4
    expect(used + (count < 10 ? 15.5 + (count ? 4 : 0) : 0)).toBeLessThanOrEqual(available)
    expect(count).toBe(available >= 241 ? 10 : Math.floor((available - 15.5) / 24.5))
  }
})
