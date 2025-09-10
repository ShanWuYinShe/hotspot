/**
 * TanStack Virtual 虚拟滚动 Composable
 * 基于 @tanstack/vue-virtual 的封装，提供更好的 TypeScript 支持和 Nuxt UI 集成
 */

import { useVirtualizer } from '@tanstack/vue-virtual'

export interface VirtualScrollConfig {
  // 容器高度
  containerHeight: number
  // 估计项目高度
  estimatedItemHeight: number
  // 缓冲区大小（额外渲染的项目数）
  overscan?: number
  // 是否启用动态高度测量
  enableDynamicSize?: boolean
  // 滚动方向
  horizontal?: boolean
  // 调试模式
  debug?: boolean
}

export interface VirtualScrollReturn<T> {
  // 虚拟化器实例
  virtualizer: ReturnType<typeof useVirtualizer>
  // 容器引用
  containerRef: Ref<HTMLElement | undefined>
  // 可见项目
  visibleItems: ComputedRef<Array<{
    index: number
    item: T
    size: number
    start: number
    end: number
    key: string | number
  }>>
  // 总高度
  totalHeight: ComputedRef<number>
  // 滚动到指定索引
  scrollToIndex: (index: number, options?: { align?: 'start' | 'center' | 'end' | 'auto' }) => void
  // 滚动到顶部
  scrollToTop: () => void
  // 滚动到底部
  scrollToBottom: () => void
  // 测量项目大小（用于动态高度）
  measureElement: (index: number, element: HTMLElement) => void
}

export const useTanStackVirtualScroll = <T = any>(
  items: Ref<T[]>,
  config: VirtualScrollConfig
): VirtualScrollReturn<T> => {
  const {
    containerHeight,
    estimatedItemHeight,
    overscan = 5,
    enableDynamicSize = false,
    horizontal = false,
    debug = false
  } = config

  // 容器引用
  const containerRef = ref<HTMLElement>()

  // 创建虚拟化器
  const virtualizer = useVirtualizer({
    count: items.value.length,
    getScrollElement: () => containerRef.value || null,
    estimateSize: () => estimatedItemHeight,
    overscan,
    horizontal,
    debug,
    // 动态尺寸测量
    ...(enableDynamicSize && {
      measureElement: (element: HTMLElement) => {
        return horizontal ? element.offsetWidth : element.offsetHeight
      }
    })
  })

  // 可见项目（增强版）
  const visibleItems = computed(() => {
    return virtualizer.value.getVirtualItems().map((virtualItem: any) => ({
      index: virtualItem.index as number,
      item: items.value[virtualItem.index] as T,
      size: virtualItem.size as number,
      start: virtualItem.start as number,
      end: virtualItem.end as number,
      key: virtualItem.key as string | number
    }))
  })

  // 总高度
  const totalHeight = computed(() => virtualizer.value.getTotalSize())

  // 滚动方法
  const scrollToIndex = (index: number, options?: { align?: 'start' | 'center' | 'end' | 'auto' }) => {
    virtualizer.value.scrollToIndex(index, options)
  }

  const scrollToTop = () => {
    containerRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const scrollToBottom = () => {
    const element = containerRef.value
    if (element) {
      element.scrollTo({ 
        top: element.scrollHeight, 
        behavior: 'smooth' 
      })
    }
  }

  // 测量元素（用于动态高度）
  const measureElement = (index: number, element: HTMLElement) => {
    if (enableDynamicSize) {
      virtualizer.value.measureElement(element)
    }
  }

  return {
    virtualizer: virtualizer as any,
    containerRef,
    visibleItems,
    totalHeight,
    scrollToIndex,
    scrollToTop,
    scrollToBottom,
    measureElement
  }
}

// 预设配置
export const VIRTUAL_SCROLL_PRESETS = {
  // 聊天消息列表
  chatMessages: {
    containerHeight: 400,
    estimatedItemHeight: 60,
    overscan: 10,
    enableDynamicSize: true
  },
  // 用户列表
  userList: {
    containerHeight: 500,
    estimatedItemHeight: 80,
    overscan: 5,
    enableDynamicSize: false
  },
  // 群组列表
  groupList: {
    containerHeight: 600,
    estimatedItemHeight: 100,
    overscan: 3,
    enableDynamicSize: false
  },
  // 动态列表
  momentList: {
    containerHeight: 700,
    estimatedItemHeight: 200,
    overscan: 2,
    enableDynamicSize: true
  }
} as const
