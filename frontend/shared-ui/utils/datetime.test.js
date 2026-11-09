import { expect, test } from 'vitest'
import { formatFullTimestamp, formatMessageTimestamp } from './datetime'

for (const formatter of [formatFullTimestamp, formatMessageTimestamp]) {
  test(`${formatter.name} uses German dates and a 24-hour clock in local time`, () => {
    expect(formatter(new Date(2026, 10, 9, 19, 11))).toBe('09.11.2026, 19:11')
    expect(formatter(new Date(2027, 0, 1, 0, 9).toISOString())).toBe('01.01.2027, 00:09')
    expect(formatter(new Date(2027, 0, 1, 9, 9).getTime())).toBe('01.01.2027, 09:09')
  })

  test(`${formatter.name} converts UTC timestamps across date and DST boundaries`, () => {
    for (const input of ['2026-12-31T23:30:00Z', '2026-03-29T01:30:00Z', '2026-10-25T01:30:00Z']) {
      const date = new Date(input)
      const pad = (n) => String(n).padStart(2, '0')
      expect(formatter(input)).toBe(
        `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${date.getFullYear()}, ${pad(date.getHours())}:${pad(date.getMinutes())}`
      )
    }
  })

  test(`${formatter.name} omits missing and invalid timestamps`, () => {
    for (const value of [null, undefined, '', 'invalid', new Date(NaN)]) {
      expect(formatter(value)).toBe('')
    }
  })
}
