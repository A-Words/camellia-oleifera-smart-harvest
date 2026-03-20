import { describe, expect, it } from 'vitest'
import {
  applyDetectionsToSession,
  buildSessionAggregateSummary,
  createSessionAggregateState
} from '../../app/utils/session-aggregator'
import type { Detection } from '../../app/types/infer'

function detection(trackId: number | null): Detection {
  return {
    bbox: [0, 0, 10, 10],
    class_name: 'camellia_oleifera_fruit',
    confidence: 0.9,
    track_id: trackId,
    ripeness: null
  }
}

describe('session aggregator', () => {
  it('deduplicates detections by track_id', () => {
    let state = createSessionAggregateState()
    state = applyDetectionsToSession(state, [
      detection(1),
      detection(2),
      detection(1)
    ])

    const summary = buildSessionAggregateSummary(state)
    expect(summary.total).toBe(2)
  })

  it('counts detections without track_id as new entries', () => {
    let state = createSessionAggregateState()
    state = applyDetectionsToSession(state, [
      detection(null),
      detection(null),
      detection(null)
    ])

    const summary = buildSessionAggregateSummary(state)
    expect(summary.total).toBe(3)
  })
})
