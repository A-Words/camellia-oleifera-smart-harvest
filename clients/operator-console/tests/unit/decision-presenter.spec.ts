import { describe, expect, it } from 'vitest'
import {
  formatDecisionSkipReason,
  formatDecisionTimestamp,
  formatDecisionZoneLabel
} from '../../app/utils/decision-presenter'

describe('decision presenter', () => {
  it('formats zone labels for the decision page', () => {
    expect(formatDecisionZoneLabel('lower_center')).toBe('下中区域')
    expect(formatDecisionZoneLabel(null)).toBe('待定')
  })

  it('formats skip reason labels', () => {
    expect(formatDecisionSkipReason('not_ready')).toBe('暂不可采')
    expect(formatDecisionSkipReason('occluded_unclear')).toBe('遮挡不清')
  })

  it('formats timestamps into a readable zh-CN string', () => {
    expect(formatDecisionTimestamp('2026-03-20T10:11:12Z')).toContain('2026/03/20')
  })
})
