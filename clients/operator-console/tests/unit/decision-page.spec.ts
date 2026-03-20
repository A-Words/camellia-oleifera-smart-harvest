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

  it('uses operator-facing copy instead of development-phase wording', () => {
    const source = readFileSync(new URL('../../app/pages/decision/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('识别页保存的树级识别记录会自动汇总到这里')
    expect(source).toContain('待补记录树木')
    expect(source).toContain('当前树还缺少识别记录')
    expect(source).toContain('每棵可执行的树都会生成一张作业任务')
    expect(source).not.toContain('Decision Center')
    expect(source).not.toContain('当前计划没有 ready 树可下发作业单。')
    expect(source).not.toContain('active 树木会进入树间路线')
    expect(source).not.toContain('observation 形成路线')
  })

  it('links recognition and operations pages into the full plot-tree workflow', () => {
    const source = readFileSync(new URL('../../app/pages/recognition/index.vue', import.meta.url), 'utf8')
    const operationsSource = readFileSync(new URL('../../app/pages/operations/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('decisionSnapshot.updateSnapshotFromFrame')
    expect(source).toContain('/v1/operations/plots')
    expect(source).toContain('/v1/decision/observations')
    expect(source).toContain('label="进入地块决策"')

    expect(operationsSource).toContain('/v1/operations/plots')
    expect(operationsSource).toContain('/v1/operations/trees')
    expect(operationsSource).toContain('/v1/operations/work-orders')
    expect(operationsSource).toContain('handleUpdateWorkOrder')
  })
})
