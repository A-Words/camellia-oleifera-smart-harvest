import type { Detection } from '~/types/infer'
import type { DetectionSummary } from '~/types/recognition'

export interface SessionAggregateState {
  seenTrackIds: Set<number>
  totalUnique: number
}

export interface SessionAggregateSummary extends DetectionSummary {}

export function createSessionAggregateState(): SessionAggregateState {
  return {
    seenTrackIds: new Set<number>(),
    totalUnique: 0
  }
}

export function applyDetectionsToSession(
  state: SessionAggregateState,
  detections: Detection[]
): SessionAggregateState {
  const next: SessionAggregateState = {
    seenTrackIds: new Set<number>(state.seenTrackIds),
    totalUnique: state.totalUnique
  }

  for (const detection of detections) {
    if (detection.track_id !== null) {
      if (next.seenTrackIds.has(detection.track_id)) {
        continue
      }
      next.seenTrackIds.add(detection.track_id)
    }

    next.totalUnique += 1
  }

  return next
}

export function buildSessionAggregateSummary(state: SessionAggregateState): SessionAggregateSummary {
  return {
    total: state.totalUnique
  }
}
