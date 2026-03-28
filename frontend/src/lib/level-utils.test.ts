import { levelBadgeClass, LEVEL_LABELS } from './level-utils'

describe('levelBadgeClass', () => {
  it('returns neutral classes for level 1', () => {
    const cls = levelBadgeClass(1)
    expect(cls).toMatch(/neutral/)
  })

  it('returns blue classes for level 2', () => {
    const cls = levelBadgeClass(2)
    expect(cls).toMatch(/blue/)
  })

  it('returns green classes for level 3', () => {
    const cls = levelBadgeClass(3)
    expect(cls).toMatch(/green/)
  })

  it('returns purple classes for level 4', () => {
    const cls = levelBadgeClass(4)
    expect(cls).toMatch(/purple/)
  })

  it('returns amber classes for level 5', () => {
    const cls = levelBadgeClass(5)
    expect(cls).toMatch(/amber/)
  })

  it('returns the default (neutral) classes for level 0', () => {
    const cls = levelBadgeClass(0)
    expect(cls).toMatch(/neutral/)
  })

  it('returns the default (neutral) classes for an out-of-range level like 99', () => {
    const cls = levelBadgeClass(99)
    expect(cls).toMatch(/neutral/)
  })
})

describe('LEVEL_LABELS', () => {
  it('defines a label for each of the 5 levels', () => {
    expect(Object.keys(LEVEL_LABELS)).toHaveLength(5)
    for (let level = 1; level <= 5; level++) {
      expect(LEVEL_LABELS[level]).toBeTruthy()
    }
  })
})
