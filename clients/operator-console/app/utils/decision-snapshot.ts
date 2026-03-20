import type { DecisionSnapshotRequest } from '~/types/decision'
import type { FrameResult } from '~/types/infer'

export function buildDecisionSnapshot(
  frame: FrameResult | null,
  frameWidth: number,
  frameHeight: number
): DecisionSnapshotRequest | null {
  if (!frame || frameWidth <= 0 || frameHeight <= 0) {
    return null
  }

  return {
    frame_index: frame.frame_index,
    timestamp_ms: frame.timestamp_ms,
    frame_width: Math.round(frameWidth),
    frame_height: Math.round(frameHeight),
    detections: frame.detections.map((detection) => ({
      bbox: [...detection.bbox] as [number, number, number, number],
      class_name: detection.class_name,
      confidence: detection.confidence,
      track_id: detection.track_id,
      ripeness: detection.ripeness
    }))
  }
}

export function createDecisionSnapshotSignature(snapshot: DecisionSnapshotRequest): string {
  return JSON.stringify(snapshot)
}
