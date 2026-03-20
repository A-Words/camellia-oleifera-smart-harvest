import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('decision page', () => {
  it('loads plot-scoped plans and archives pending observations before plan generation', () => {
    const source = readFileSync(new URL('../../app/pages/decision/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('/v1/decision/observations')
    expect(source).toContain('/v1/decision/plans')
    expect(source).toContain('/v1/operations/work-orders')
    expect(source).toContain('maybeArchivePendingObservation')
    expect(source).toContain('handleGeneratePlan')
  })

  it('renders route planning, tree detail, and manual override sections in a single decision center page', () => {
    const source = readFileSync(new URL('../../app/pages/decision/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('树优先级路线')
    expect(source).toContain('当前树推荐详情')
    expect(source).toContain('人工调整与下发')
    expect(source).toContain('toggleManualSkip')
    expect(source).toContain('moveTree')
  })

  it('links recognition and operations pages into the full plot-tree workflow', () => {
    const source = readFileSync(new URL('../../app/pages/recognition/index.vue', import.meta.url), 'utf8')
    const operationsSource = readFileSync(new URL('../../app/pages/operations/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('decisionSnapshot.updateSnapshotFromFrame')
    expect(source).toContain('/v1/operations/plots')
    expect(source).toContain('/v1/decision/observations')
    expect(source).toContain('label="进入整块地决策"')

    expect(operationsSource).toContain('/v1/operations/plots')
    expect(operationsSource).toContain('/v1/operations/trees')
    expect(operationsSource).toContain('/v1/operations/work-orders')
    expect(operationsSource).toContain('handleUpdateWorkOrder')
  })
})
