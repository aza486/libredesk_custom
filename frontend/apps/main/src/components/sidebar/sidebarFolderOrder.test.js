import { describe, expect, it } from 'vitest'
import { moveFolder } from './sidebarFolderOrder'

describe('shared sidebar folder ordering', () => {
  it('moves a folder in either direction without changing membership or the original array', () => {
    const ids = [11, 12, 13]
    expect(moveFolder(ids, 13, 0)).toEqual([13, 11, 12])
    expect(moveFolder(ids, 11, 2)).toEqual([12, 13, 11])
    expect(ids).toEqual([11, 12, 13])
  })
  it('preserves the order at keyboard boundaries and for unknown IDs', () => {
    expect(moveFolder([11, 12], 11, -1)).toEqual([11, 12])
    expect(moveFolder([11, 12], 12, 2)).toEqual([11, 12])
    expect(moveFolder([11, 12], 99, 0)).toEqual([11, 12])
  })
})
