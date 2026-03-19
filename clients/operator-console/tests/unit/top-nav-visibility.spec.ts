import { describe, expect, it } from 'vitest'
import { shouldShowTopNav } from '../../app/utils/top-nav-visibility'

describe('top nav visibility', () => {
  it('always shows nav in the new console shell', () => {
    expect(shouldShowTopNav()).toBe(true)
  })
})
