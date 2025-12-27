export interface InfiniteLoadMoreOptions {
  /** 预加载距离 */
  rootMargin?: string
}

/**
 * 统一的“触底自动加载更多”逻辑（IntersectionObserver）
 * - 自动触发 loadMore
 * - 仅在浏览器支持 IntersectionObserver 时启用
 * - 可用于保留按钮作为兜底：`canAutoLoadMore`
 */
export function useInfiniteLoadMore(
  loadMore: () => void | Promise<void>,
  isBusy: ComputedRef<boolean>,
  hasMore: ComputedRef<boolean>,
  options: InfiniteLoadMoreOptions = {}
) {
  const { rootMargin = '600px 0px' } = options
  const trigger = ref<HTMLElement | null>(null)

  const canAutoLoadMore = computed(() => import.meta.client && 'IntersectionObserver' in window)

  useIntersectionObserver(
    trigger,
    ([entry]) => {
      if (!entry?.isIntersecting) return
      if (!canAutoLoadMore.value) return
      if (!hasMore.value || isBusy.value) return
      void loadMore()
    },
    { rootMargin }
  )

  return {
    trigger,
    canAutoLoadMore,
  }
}

