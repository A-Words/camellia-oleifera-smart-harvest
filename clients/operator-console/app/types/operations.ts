import type { PickSequenceItem, SkipItem, ZonePriority } from '~/types/decision'

export type TreeStatus = 'active' | 'disabled'
export type WorkOrderStatus = 'pending' | 'in_progress' | 'completed' | 'skipped'

export interface Plot {
  plot_id: string
  name: string
  code: string
  row_count: number
  notes: string
  created_at: string
  updated_at: string
}

export interface PlotListResponse {
  items: Plot[]
}

export interface CreatePlotRequest {
  name: string
  code: string
  row_count: number
  notes: string
}

export interface TreeArchive {
  tree_id: string
  plot_id: string
  tree_code: string
  row_index: number
  col_index: number
  x: number
  y: number
  status: TreeStatus
  created_at: string
  updated_at: string
}

export interface TreeListResponse {
  items: TreeArchive[]
}

export interface CreateTreeRequest {
  plot_id: string
  tree_code: string
  row_index: number
  col_index: number
  x: number
  y: number
  status: TreeStatus
}

export interface UpdateTreeRequest {
  tree_code?: string
  row_index?: number
  col_index?: number
  x?: number
  y?: number
  status?: TreeStatus
}

export interface WorkOrder {
  work_order_id: string
  plot_id: string
  tree_id: string
  tree_code: string
  plan_id: string
  status: WorkOrderStatus
  zone_priorities: ZonePriority[]
  pick_sequence: PickSequenceItem[]
  skip_items: SkipItem[]
  skip_reason_note: string
  started_at: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
}

export interface WorkOrderListResponse {
  items: WorkOrder[]
}

export interface CreateWorkOrdersRequest {
  plan_id: string
}

export interface UpdateWorkOrderRequest {
  status: WorkOrderStatus
  skip_reason_note?: string
}
