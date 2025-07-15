/**
 * 现代化Tooltip工具系统
 * 基于Vue 3 Composition API和TypeScript
 * 无兼容性代码，完全响应式
 */

import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'

// 类型定义
export interface TooltipPosition {
  x: number
  y: number
}

export interface TooltipOptions {
  position?: 'top' | 'bottom' | 'left' | 'right' | 'auto'
  delay?: number
  duration?: number
  offset?: number
  maxWidth?: number
  className?: string
}

export interface TooltipConfig {
  [category: string]: {
    [key: string]: string
  }
}

// 默认配置
const DEFAULT_OPTIONS: Required<TooltipOptions> = {
  position: 'top',
  delay: 300,
  duration: 0,
  offset: 8,
  maxWidth: 300,
  className: 'unified-tooltip'
}

// 提示内容配置
export const tooltipConfig: TooltipConfig = {
  navigation: {
    home: '首页 - 查看最新动态和附近活动',
    groups: '群组 - 管理和浏览群组',
    messages: '消息 - 查看私信和群组消息',
    map: '地图 - 探索附近的用户和地点',
    dynamic: '动态 - 浏览和发布生活动态',
    profile: '个人 - 查看和编辑个人资料',
    settings: '设置 - 个性化应用配置'
  },
  
  actions: {
    create: '创建新内容',
    edit: '编辑此项目',
    delete: '删除此项目',
    share: '分享给朋友',
    like: '点赞',
    comment: '添加评论',
    refresh: '刷新内容',
    search: '搜索',
    filter: '筛选结果',
    sort: '排序'
  },
  
  status: {
    online: '在线状态',
    offline: '离线状态',
    busy: '忙碌状态',
    away: '暂时离开',
    loading: '正在加载...',
    saving: '正在保存...',
    uploading: '正在上传...',
    processing: '正在处理...'
  }
}

// Composition API - Tooltip管理器
export function useTooltipManager() {
  // 响应式状态
  const activeTooltips = ref(new Map<string, HTMLElement>())
  const timeouts = ref(new Map<string, ReturnType<typeof setTimeout>>())
  
  // 创建tooltip元素
  const createTooltipElement = (content: string, options: Required<TooltipOptions>): HTMLElement => {
    const tooltip = document.createElement('div')
    tooltip.className = `${options.className} unified-tooltip--${options.position}`
    tooltip.innerHTML = content
    
    // 应用样式
    Object.assign(tooltip.style, {
      position: 'fixed',
      background: 'rgba(0, 0, 0, 0.9)',
      color: 'white',
      padding: 'var(--spacing-xs) var(--spacing-sm)',
      borderRadius: 'var(--card-radius)',
      fontSize: 'var(--font-size-xs)',
      maxWidth: `${options.maxWidth}px`,
      whiteSpace: 'pre-wrap',
      wordBreak: 'break-word',
      zIndex: 'var(--z-tooltip)',
      pointerEvents: 'none',
      opacity: '0',
      transform: 'scale(0.95)',
      transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
      backdropFilter: 'blur(8px)',
      border: '1px solid rgba(255, 255, 255, 0.1)',
      boxShadow: 'var(--shadow-lg)'
    })

    return tooltip
  }

  // 计算最佳位置
  const calculatePosition = (
    target: HTMLElement, 
    tooltip: HTMLElement, 
    preferredPosition: string,
    offset: number
  ): { position: string; coords: TooltipPosition } => {
    const targetRect = target.getBoundingClientRect()
    const tooltipRect = tooltip.getBoundingClientRect()
    const viewport = {
      width: window.innerWidth,
      height: window.innerHeight
    }

    const positions = {
      top: {
        x: targetRect.left + (targetRect.width - tooltipRect.width) / 2,
        y: targetRect.top - tooltipRect.height - offset
      },
      bottom: {
        x: targetRect.left + (targetRect.width - tooltipRect.width) / 2,
        y: targetRect.bottom + offset
      },
      left: {
        x: targetRect.left - tooltipRect.width - offset,
        y: targetRect.top + (targetRect.height - tooltipRect.height) / 2
      },
      right: {
        x: targetRect.right + offset,
        y: targetRect.top + (targetRect.height - tooltipRect.height) / 2
      }
    }

    // 检查首选位置是否合适
    const preferred = positions[preferredPosition as keyof typeof positions]
    if (preferred && 
        preferred.x >= 0 && 
        preferred.x + tooltipRect.width <= viewport.width &&
        preferred.y >= 0 && 
        preferred.y + tooltipRect.height <= viewport.height) {
      return { position: preferredPosition, coords: preferred }
    }

    // 自动选择最佳位置
    for (const [pos, coords] of Object.entries(positions)) {
      if (coords.x >= 0 && 
          coords.x + tooltipRect.width <= viewport.width &&
          coords.y >= 0 && 
          coords.y + tooltipRect.height <= viewport.height) {
        return { position: pos, coords }
      }
    }

    // 如果没有完美位置，使用首选位置并调整
    const coords = preferred
    coords.x = Math.max(0, Math.min(coords.x, viewport.width - tooltipRect.width))
    coords.y = Math.max(0, Math.min(coords.y, viewport.height - tooltipRect.height))
    
    return { position: preferredPosition, coords }
  }

  // 显示tooltip
  const show = (
    target: HTMLElement, 
    content: string, 
    options: Partial<TooltipOptions> = {}
  ): void => {
    const mergedOptions = { ...DEFAULT_OPTIONS, ...options }
    const targetId = target.dataset.tooltipId || Math.random().toString(36)
    target.dataset.tooltipId = targetId

    // 清除现有的tooltip和定时器
    hide(target)

    const showTooltip = () => {
      const tooltip = createTooltipElement(content, mergedOptions)
      document.body.appendChild(tooltip)

      // 计算位置
      const { position, coords } = calculatePosition(
        target,
        tooltip,
        mergedOptions.position,
        mergedOptions.offset
      )

      // 应用位置
      tooltip.style.left = `${coords.x}px`
      tooltip.style.top = `${coords.y}px`
      tooltip.className = `${mergedOptions.className} unified-tooltip--${position}`

      // 显示动画
      requestAnimationFrame(() => {
        tooltip.style.opacity = '1'
        tooltip.style.transform = 'scale(1)'
      })

      // 存储引用
      activeTooltips.value.set(targetId, tooltip)

      // 设置自动隐藏
      if (mergedOptions.duration > 0) {
        const hideTimeout = setTimeout(() => {
          hide(target)
        }, mergedOptions.duration)
        timeouts.value.set(`${targetId}-hide`, hideTimeout)
      }
    }

    // 延迟显示
    if (mergedOptions.delay > 0) {
      const showTimeout = setTimeout(showTooltip, mergedOptions.delay)
      timeouts.value.set(`${targetId}-show`, showTimeout)
    } else {
      showTooltip()
    }
  }

  // 隐藏tooltip
  const hide = (target: HTMLElement): void => {
    const targetId = target.dataset.tooltipId
    if (!targetId) return

    // 清除定时器
    const showTimeout = timeouts.value.get(`${targetId}-show`)
    const hideTimeout = timeouts.value.get(`${targetId}-hide`)
    if (showTimeout) clearTimeout(showTimeout)
    if (hideTimeout) clearTimeout(hideTimeout)
    timeouts.value.delete(`${targetId}-show`)
    timeouts.value.delete(`${targetId}-hide`)

    // 移除tooltip
    const tooltip = activeTooltips.value.get(targetId)
    if (tooltip) {
      tooltip.style.opacity = '0'
      tooltip.style.transform = 'scale(0.95)'
      setTimeout(() => {
        tooltip.remove()
      }, 200)
      activeTooltips.value.delete(targetId)
    }
  }

  // 清除所有tooltips
  const clearAll = (): void => {
    timeouts.value.forEach(timeout => clearTimeout(timeout))
    timeouts.value.clear()
    activeTooltips.value.forEach(tooltip => tooltip.remove())
    activeTooltips.value.clear()
  }

  // 获取提示内容
  const getContent = (category: string, key: string): string => {
    return tooltipConfig[category]?.[key] || ''
  }

  // 清理函数
  onUnmounted(() => {
    clearAll()
  })

  return {
    show,
    hide,
    clearAll,
    getContent,
    activeTooltips: computed(() => activeTooltips.value),
    timeouts: computed(() => timeouts.value)
  }
}

// 全局tooltip实例
let globalTooltipManager: ReturnType<typeof useTooltipManager> | null = null

// 获取全局tooltip管理器
export function getGlobalTooltipManager() {
  if (!globalTooltipManager) {
    globalTooltipManager = useTooltipManager()
  }
  return globalTooltipManager
}

// 导出的工具函数（向后兼容）
export function showTooltip(
  target: HTMLElement, 
  content: string, 
  options?: Partial<TooltipOptions>
): void {
  const manager = getGlobalTooltipManager()
  manager.show(target, content, options)
}

export function hideTooltip(target: HTMLElement): void {
  const manager = getGlobalTooltipManager()
  manager.hide(target)
}

export function clearAllTooltips(): void {
  const manager = getGlobalTooltipManager()
  manager.clearAll()
}

export function getTooltipContent(category: string, key: string): string {
  const manager = getGlobalTooltipManager()
  return manager.getContent(category, key)
}

// Vue 3 Composition API Hook
export function useTooltip() {
  const manager = useTooltipManager()

  return {
    show: manager.show,
    hide: manager.hide,
    getContent: manager.getContent,
    clearAll: manager.clearAll,
    activeTooltips: manager.activeTooltips,
    isVisible: computed(() => manager.activeTooltips.value.size > 0)
  }
}

// Vue指令定义
export const vTooltip = {
  mounted(el: HTMLElement, binding: any) {
    const content = binding.value || ''
    if (!content) return

    const options: Partial<TooltipOptions> = {
      position: binding.arg as any || 'top',
      delay: binding.modifiers.fast ? 100 : binding.modifiers.slow ? 500 : 300,
      ...binding.modifiers
    }

    const manager = getGlobalTooltipManager()

    el.addEventListener('mouseenter', () => {
      manager.show(el, content, options)
    })

    el.addEventListener('mouseleave', () => {
      manager.hide(el)
    })

    el.addEventListener('focus', () => {
      manager.show(el, content, { ...options, delay: 0 })
    })

    el.addEventListener('blur', () => {
      manager.hide(el)
    })
  },

  unmounted(el: HTMLElement) {
    const manager = getGlobalTooltipManager()
    manager.hide(el)
  },

  updated(el: HTMLElement, binding: any) {
    if (binding.value !== binding.oldValue) {
      const manager = getGlobalTooltipManager()
      manager.hide(el)
    }
  }
}
