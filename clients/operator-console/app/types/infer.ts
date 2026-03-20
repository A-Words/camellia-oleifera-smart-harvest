export type RipenessLabel = 'harvestable' | 'not_ready' | 'occluded_unclear'

export interface Detection {
  bbox: [number, number, number, number]
  class_name: 'camellia_oleifera_fruit'
  confidence: number
  track_id: number | null
  ripeness: RipenessLabel | null
}

export interface FrameSummary {
  total: number
}

export interface FrameResult {
  frame_index: number
  timestamp_ms: number
  detections: Detection[]
  frame_summary: FrameSummary
}

export interface SessionSummary {
  total_detected: number
}

export interface StreamFrameEnvelope {
  type: 'frame'
  model_version: string
  schema_version: string
  result: FrameResult
}

export interface StreamSummaryEnvelope {
  type: 'summary'
  model_version: string
  schema_version: string
  summary: SessionSummary
}

export interface StreamErrorEnvelope {
  type: 'error'
  detail: string
}

export type StreamEnvelope = StreamFrameEnvelope | StreamSummaryEnvelope | StreamErrorEnvelope
