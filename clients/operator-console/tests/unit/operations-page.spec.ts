import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'

describe('operations page', () => {
  it('shows operator-facing status labels and field copy', () => {
    const source = readFileSync(new URL('../../app/pages/operations/index.vue', import.meta.url), 'utf8')

    expect(source).toContain('地块与树木档案')
    expect(source).toContain('按现场进度持续推进执行状态')
    expect(source).toContain('buildTreeStatusOptions')
    expect(source).toContain('buildWorkOrderStatusOptions')
    expect(source).toContain('placeholder="若需要跳过，请填写现场说明（选填）。"')
    expect(source).toContain('开始时间：')
    expect(source).toContain('完成时间：')
    expect(source).not.toContain('<option value="pending">pending</option>')
    expect(source).not.toContain('<option value="in_progress">in_progress</option>')
    expect(source).not.toContain('placeholder="若需要跳过，请先填写 skip_reason_note。"')
    expect(source).not.toContain('started_at:')
    expect(source).not.toContain('completed_at:')
  })
})
