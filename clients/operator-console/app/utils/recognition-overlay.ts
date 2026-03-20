import type { Detection } from '~/types/infer'

export interface OverlayBox {
  key: string
  label: string
  left: string
  top: string
  width: string
  height: string
}

export function buildRecognitionOverlayBoxes(
  detections: Detection[],
  videoWidth: number,
  videoHeight: number
): OverlayBox[] {
  if (videoWidth <= 0 || videoHeight <= 0) {
    return []
  }

  return detections.flatMap((detection, index) => {
    const normalized = normalizeBbox(detection.bbox, videoWidth, videoHeight)
    if (!normalized) {
      return []
    }

    const [x1, y1, x2, y2] = normalized
    return [{
      key: `${index}-${x1}-${y1}-${x2}-${y2}-${detection.confidence}`,
      label: formatRipenessLabel(detection),
      left: toPercent(x1 / videoWidth),
      top: toPercent(y1 / videoHeight),
      width: toPercent((x2 - x1) / videoWidth),
      height: toPercent((y2 - y1) / videoHeight)
    }]
  })
}

function normalizeBbox(
  bbox: Detection['bbox'],
  videoWidth: number,
  videoHeight: number
): Detection['bbox'] | null {
  const [rawX1, rawY1, rawX2, rawY2] = bbox
  if (![rawX1, rawY1, rawX2, rawY2].every(Number.isFinite)) {
    return null
  }

  const maxX = videoWidth
  const maxY = videoHeight
  const x1 = clamp(Math.min(rawX1, rawX2), 0, maxX)
  const y1 = clamp(Math.min(rawY1, rawY2), 0, maxY)
  const x2 = clamp(Math.max(rawX1, rawX2), 0, maxX)
  const y2 = clamp(Math.max(rawY1, rawY2), 0, maxY)

  if (x2 <= x1 || y2 <= y1) {
    return null
  }

  return [x1, y1, x2, y2]
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

function toPercent(value: number): string {
  return `${(value * 100).toFixed(3)}%`
}

function formatRipenessLabel(detection: Detection): string {
  if (detection.ripeness === 'harvestable') {
    return '可采'
  }
  if (detection.ripeness === 'not_ready') {
    return '暂不可采'
  }
  if (detection.ripeness === 'occluded_unclear') {
    return '遮挡不清'
  }
  return '判定中'
}
