import { describe, expect, it } from 'vitest'
import {
  formatDecisionSkipReason,
  formatDecisionTimestamp,
  formatDecisionZoneLabel,
  formatTreeStatus,
  formatTreeRecommendationStatus,
  formatWorkOrderStatus
} from '../../app/utils/decision-presenter'

describe('decision presenter', () => {
  it('formats zone labels for the decision page', () => {
    expect(formatDecisionZoneLabel('lower_center')).toBe('下中区域')
    expect(formatDecisionZoneLabel(null)).toBe('待定')
  })

  it('formats skip reason labels', () => {
    expect(formatDecisionSkipReason('not_ready')).toBe('暂不可采')
    expect(formatDecisionSkipReason('occluded_unclear')).toBe('遮挡不清')
    expect(formatDecisionSkipReason('manual_skip')).toBe('人工跳过')
  })

  it('formats timestamps into a readable zh-CN string', () => {
    expect(formatDecisionTimestamp('2026-03-20T10:11:12Z')).toContain('2026/03/20')
  })

  it('formats decision and operations statuses', () => {
    expect(formatTreeRecommendationStatus('ready')).toBe('可进入决策')
    expect(formatTreeStatus('disabled')).toBe('停用')
    expect(formatWorkOrderStatus('in_progress')).toBe('执行中')
  })
})
