import { describe, expect, it } from 'vitest'
import { buildRecognitionOverlayBoxes } from '../../app/utils/recognition-overlay'
import type { Detection, RipenessLabel } from '../../app/types/infer'

function detection(
  bbox: Detection['bbox'],
  confidence = 0.91,
  ripeness: RipenessLabel | null = 'harvestable'
): Detection {
  return {
    bbox,
    class_name: 'camellia_oleifera_fruit',
    confidence,
    track_id: null,
    ripeness
  }
}

describe('recognition overlay', () => {
  it('maps valid detection boxes into overlay percentages', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([10, 20, 110, 220], 0.91)
    ], 200, 400)

    expect(boxes).toEqual([{
      key: '0-10-20-110-220-0.91',
      label: '可采',
      left: '5.000%',
      top: '5.000%',
      width: '50.000%',
      height: '50.000%'
    }])
  })

  it('clips detections that exceed video bounds', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([-20, 50, 260, 170], 0.73, 'not_ready')
    ], 200, 160)

    expect(boxes).toEqual([{
      key: '0-0-50-200-160-0.73',
      label: '暂不可采',
      left: '0.000%',
      top: '31.250%',
      width: '100.000%',
      height: '68.750%'
    }])
  })

  it('maps ripeness labels into human readable overlay text', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([10, 20, 110, 220], 0.88, 'occluded_unclear')
    ], 200, 400)

    expect(boxes[0]?.label).toBe('遮挡不清')
  })

  it('shows pending text when ripeness is not ready yet', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([10, 20, 110, 220], 0.88, null)
    ], 200, 400)

    expect(boxes[0]?.label).toBe('判定中')
  })

  it('skips zero-area or invalid boxes', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([10, 10, 10, 30]),
      detection([NaN, 0, 10, 10])
    ], 200, 160)

    expect(boxes).toEqual([])
  })

  it('returns no boxes when video dimensions are unavailable', () => {
    const boxes = buildRecognitionOverlayBoxes([
      detection([10, 20, 30, 40])
    ], 0, 160)

    expect(boxes).toEqual([])
  })
})
