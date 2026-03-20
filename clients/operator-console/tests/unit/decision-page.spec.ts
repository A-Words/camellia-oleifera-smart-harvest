import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('decision page', () => {
  it('auto-loads history and submits the latest snapshot when available', () => {
    const source = readFileSync(new URL('../../app/pages/decision/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('await loadHistory()')
    expect(source).toContain('if (shouldAutoSubmit.value)')
    expect(source).toContain('/v1/decision/recommendation')
    expect(source).toContain('/v1/decision/history')
  })

  it('renders current result, empty state, and history sections through dedicated components', () => {
    const source = readFileSync(new URL('../../app/pages/decision/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('<DecisionResultPanel')
    expect(source).toContain('<DecisionPlaceholderPanel')
    expect(source).toContain('<DecisionHistoryPanel')
  })

  it('links the recognition page to decision generation', () => {
    const source = readFileSync(new URL('../../app/pages/recognition/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('decisionSnapshot.updateSnapshotFromFrame')
    expect(source).toContain('label="生成采摘决策"')
  })
})
