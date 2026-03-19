export type TopNavKey = 'recognition' | 'decision' | 'operations'

type TopNavItemDefinition = {
  key: TopNavKey
  label: string
  to: string
  icon: string
  activePrefixes: string[]
}

export type TopNavItem = {
  key: TopNavKey
  label: string
  to: string
  icon: string
  active: boolean
}

const TOP_NAV_DEFINITIONS: TopNavItemDefinition[] = [
  {
    key: 'recognition',
    label: '识别',
    to: '/recognition',
    icon: 'i-lucide-camera',
    activePrefixes: ['/recognition']
  },
  {
    key: 'decision',
    label: '决策',
    to: '/decision',
    icon: 'i-lucide-route',
    activePrefixes: ['/decision']
  },
  {
    key: 'operations',
    label: '作业',
    to: '/operations',
    icon: 'i-lucide-briefcase-business',
    activePrefixes: ['/operations']
  }
]

function isNavItemActive(path: string, prefixes: string[]): boolean {
  return prefixes.some((prefix) => path.startsWith(prefix))
}

export function buildTopNavItems(path: string): TopNavItem[] {
  return TOP_NAV_DEFINITIONS.map((item) => ({
    key: item.key,
    label: item.label,
    to: item.to,
    icon: item.icon,
    active: isNavItemActive(path, item.activePrefixes)
  }))
}
