import { inject, onUnmounted } from 'vue'

export interface DashboardHeaderAction {
  key: string
  label?: string
  icon?: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  loading?: boolean
  disabled?: boolean
  class?: string
  action: () => void
}

export interface DashboardHeaderContext {
  setHeader: (header: { title?: string; description?: string }) => void
  clearHeader: () => void
  setStats: (stats: unknown[] | null) => void
  clearStats: () => void
  setActions: (actions: DashboardHeaderAction[]) => void
  clearActions: () => void
}

const DASHBOARD_HEADER_CTX_KEY = 'hs:dashboardHeader'

export function useDashboardHeader() {
  const ctx = inject<DashboardHeaderContext | null>(DASHBOARD_HEADER_CTX_KEY, null)

  const setHeader = (header: { title?: string; description?: string }) => {
    ctx?.setHeader(header)
  }

  const clearHeader = () => {
    ctx?.clearHeader()
  }

  const setStats = (stats: unknown[] | null) => {
    ctx?.setStats(stats)
  }

  const clearStats = () => {
    ctx?.clearStats()
  }

  const setActions = (actions: DashboardHeaderAction[]) => {
    ctx?.setActions(actions)
  }

  const clearActions = () => {
    ctx?.clearActions()
  }

  // 页面卸载时自动清理，避免跨页面遗留
  onUnmounted(() => {
    ctx?.clearActions()
    ctx?.clearStats()
    ctx?.clearHeader()
  })

  return { setHeader, clearHeader, setStats, clearStats, setActions, clearActions }
}

export { DASHBOARD_HEADER_CTX_KEY }

