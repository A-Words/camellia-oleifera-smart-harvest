const ZONE_LABELS: Record<string, string> = {
  upper_left: '上左区域',
  upper_center: '上中区域',
  upper_right: '上右区域',
  middle_left: '中左区域',
  middle_center: '中部区域',
  middle_right: '中右区域',
  lower_left: '下左区域',
  lower_center: '下中区域',
  lower_right: '下右区域'
}

const SKIP_REASON_LABELS: Record<string, string> = {
  not_ready: '暂不可采',
  occluded_unclear: '遮挡不清'
}

export function formatDecisionZoneLabel(zone: string | null | undefined): string {
  if (!zone) {
    return '待定'
  }
  return ZONE_LABELS[zone] || zone
}

export function formatDecisionSkipReason(reason: string): string {
  return SKIP_REASON_LABELS[reason] || reason
}

export function formatDecisionTimestamp(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).format(date)
}
