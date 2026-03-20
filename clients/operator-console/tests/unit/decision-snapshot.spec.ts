import { describe, expect, it } from 'vitest'
import { buildDecisionSnapshot, createDecisionSnapshotSignature } from '../../app/utils/decision-snapshot'
import type { FrameResult } from '../../app/types/infer'

function buildFrameResult(): FrameResult {
  return {
    frame_index: 7,
    timestamp_ms: 123456,
    detections: [
      {
        bbox: [1, 2, 30, 40],
        class_name: 'camellia_oleifera_fruit',
        confidence: 0.93,
        track_id: 5,
        ripeness: 'harvestable'
      }
    ],
    frame_summary: {
      total: 1
    }
  }
}

describe('decision snapshot', () => {
  it('builds a request payload from the latest frame and stage size', () => {
    const snapshot = buildDecisionSnapshot(buildFrameResult(), 1920, 1080)

    expect(snapshot).toEqual({
      frame_index: 7,
      timestamp_ms: 123456,
      frame_width: 1920,
      frame_height: 1080,
      detections: [
        {
          bbox: [1, 2, 30, 40],
          class_name: 'camellia_oleifera_fruit',
          confidence: 0.93,
          track_id: 5,
          ripeness: 'harvestable'
        }
      ]
    })
  })

  it('returns null when frame dimensions are not ready', () => {
    expect(buildDecisionSnapshot(buildFrameResult(), 0, 1080)).toBeNull()
    expect(buildDecisionSnapshot(null, 1920, 1080)).toBeNull()
  })

  it('creates a stable signature for the same payload', () => {
    const snapshot = buildDecisionSnapshot(buildFrameResult(), 1920, 1080)
    expect(snapshot).not.toBeNull()
    expect(createDecisionSnapshotSignature(snapshot!)).toBe(createDecisionSnapshotSignature(snapshot!))
  })
})
