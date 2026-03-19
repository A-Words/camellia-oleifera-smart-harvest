export type RipenessLabel = 'green' | 'half' | 'red' | 'young'

export interface RecognitionSummary {
  total: number
  green: number
  half: number
  red: number
  young: number
  unripe_count: number
  unripe_ratio: number
  unripe_handling: 'sorted_out'
}
