import type { Detection } from '~/types/infer'

export type DecisionZone =
  | 'upper_left'
  | 'upper_center'
  | 'upper_right'
  | 'middle_left'
  | 'middle_center'
  | 'middle_right'
  | 'lower_left'
  | 'lower_center'
  | 'lower_right'

export type SkipReason = 'not_ready' | 'occluded_unclear' | 'manual_skip'
export type TreeRecommendationStatus = 'ready' | 'pending_observation'

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
  main_priority_zone: DecisionZone | null
  estimated_path_length: number
}

export interface ZonePriority {
  order: number
  zone: DecisionZone
  harvestable_count: number
  distance_to_start: number
  note: string
}

export interface PickSequenceItem {
  order: number
  detection_index: number
  track_id: number | null
  zone: DecisionZone
  confidence: number
  reason: string
}

export interface SkipItem {
  detection_index: number
  track_id: number | null
  zone: DecisionZone
  skip_reason: SkipReason
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

export interface ObservationCreateRequest extends DecisionSnapshotRequest {
  tree_id: string
  captured_at?: string
}

export interface TreeObservation extends DecisionSnapshotRequest {
  observation_id: string
  tree_id: string
  captured_at: string
}

export interface TreeObservationListResponse {
  items: TreeObservation[]
}

export interface PlanSummary {
  total_trees: number
  ready_trees: number
  pending_observation_trees: number
  total_harvestable_count: number
}

export interface PlanTreeSequenceItem {
  tree_id: string
  tree_code: string
  priority_order: number
  harvestable_count: number
  harvestable_confidence_weighted_score: number
  distance_from_previous: number
  status: TreeRecommendationStatus
}

export interface TreeRecommendation {
  tree_id: string
  tree_code: string
  observation_id: string | null
  status: TreeRecommendationStatus
  summary: DecisionSummary
  zone_priorities: ZonePriority[]
  pick_sequence: PickSequenceItem[]
  skip_items: SkipItem[]
}

export interface TreeOverride {
  tree_id: string
  zone_order: DecisionZone[]
  pick_sequence_detection_indices: number[]
  manual_skipped_detection_indices: number[]
}

export interface DecisionPlan {
  plan_id: string
  plot_id: string
  generated_at: string
  manual_override: boolean
  manual_override_state: {
    tree_sequence: string[]
    tree_overrides: TreeOverride[]
  }
  summary: PlanSummary
  tree_sequence: PlanTreeSequenceItem[]
  tree_recommendations: TreeRecommendation[]
}

export interface DecisionPlanListResponse {
  items: DecisionPlan[]
}

export interface CreatePlanRequest {
  plot_id: string
}

export interface UpdatePlanRequest {
  tree_sequence: string[]
  tree_overrides: TreeOverride[]
}
