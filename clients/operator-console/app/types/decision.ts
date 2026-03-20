import type { Detection } from '~/types/infer'

export interface DecisionSnapshotRequest {
  frame_index: number
  timestamp_ms: number
  frame_width: number
  frame_height: number
  detections: Detection[]
}

export interface DecisionSummary {
  total_detections: number
  harvestable_count: number
  skipped_count: number
  main_priority_zone: string | null
  estimated_path_length: number
}

export interface ZonePriority {
  order: number
  zone: string
  harvestable_count: number
  distance_to_start: number
  note: string
}

export interface PickSequenceItem {
  order: number
  detection_index: number
  track_id: number | null
  zone: string
  confidence: number
  reason: string
}

export interface SkipItem {
  detection_index: number
  track_id: number | null
  zone: string
  skip_reason: 'not_ready' | 'occluded_unclear'
  confidence: number
  reason: string
}

export interface DecisionRecommendationResponse {
  decision_id: string
  created_at: string
  summary: DecisionSummary
  zone_priorities: ZonePriority[]
  pick_sequence: PickSequenceItem[]
  skip_items: SkipItem[]
}

export interface DecisionHistoryItem extends DecisionRecommendationResponse {
  request: DecisionSnapshotRequest
}

export interface DecisionHistoryResponse {
  items: DecisionHistoryItem[]
}
