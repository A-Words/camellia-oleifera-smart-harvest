import { describe, expect, it } from 'vitest'
import { buildTopNavItems } from '../../app/utils/top-nav-items'

function getItem(path: string, key: 'recognition' | 'decision' | 'operations') {
  const item = buildTopNavItems(path).find((candidate) => candidate.key === key)
  if (!item) {
    throw new Error(`missing nav item: ${key}`)
  }
  return item
}

describe('top nav items', () => {
  it('links to the new three-layer routes', () => {
    expect(getItem('/recognition', 'recognition').to).toBe('/recognition')
    expect(getItem('/decision', 'decision').to).toBe('/decision')
    expect(getItem('/operations', 'operations').to).toBe('/operations')
  })

  it('marks decision route as active', () => {
    expect(getItem('/decision', 'decision').active).toBe(true)
    expect(getItem('/decision', 'recognition').active).toBe(false)
    expect(getItem('/decision', 'operations').active).toBe(false)
  })

  it('marks recognition route as active', () => {
    expect(getItem('/recognition', 'recognition').active).toBe(true)
    expect(getItem('/recognition', 'decision').active).toBe(false)
    expect(getItem('/recognition', 'operations').active).toBe(false)
  })
})
