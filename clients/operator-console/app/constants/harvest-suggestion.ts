import type { HarvestSuggestion } from '~/types/infer'

type SuggestionColor = 'success' | 'warning' | 'neutral' | 'error'

export interface HarvestSuggestionMeta {
  label: string
  description: string
  color: SuggestionColor
}

export const HARVEST_SUGGESTION_META: Record<HarvestSuggestion, HarvestSuggestionMeta> = {
  not_ready: {
    label: '暂不建议采摘',
    description: '成熟果比例不足，建议继续观察油茶果成熟进度。',
    color: 'neutral'
  },
  partially_ready: {
    label: '部分可采',
    description: '已进入阶段性采摘窗口，适合结合人工经验分区处理。',
    color: 'warning'
  },
  ready: {
    label: '建议采摘',
    description: '成熟度满足当前识别规则，可进入采摘决策环节。',
    color: 'success'
  },
  overripe_risk: {
    label: '过熟风险',
    description: '存在过熟风险，建议优先处理该区域果实。',
    color: 'error'
  }
}
