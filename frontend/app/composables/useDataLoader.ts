/**
 * 通用数据加载器 - 统一管理加载状态和数据获取
 */

interface LoaderOptions<T> {
  initialData?: T[]
  autoLoad?: boolean
}

interface PaginatedData<T> {
  items: T[]
  hasMore: boolean
  nextCursor?: string
}

export function useDataLoader<T>(
  loadFunction: (cursor?: string) => Promise<PaginatedData<T>>,
  options: LoaderOptions<T> = {}
) {
  const { initialData = [], autoLoad = true } = options

  // 状态管理
  const data = ref<T[]>(initialData)
  const loading = ref(false)
  const loadingMore = ref(false)
  const hasMore = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref<string>()

  // 加载数据
  const load = async (reset = true) => {
    if (reset) {
      loading.value = true
      data.value = []
      nextCursor.value = undefined
    } else {
      loadingMore.value = true
    }

    error.value = null

    try {
      const result = await loadFunction(reset ? undefined : nextCursor.value)
      
      if (reset) {
        data.value = result.items || []
      } else {
        data.value.push(...(result.items || []))
      }
      
      hasMore.value = result.hasMore || false
      nextCursor.value = result.nextCursor
    } catch (err: any) {
      error.value = err.message || '加载失败'
      console.error('Data loading error:', err)
    } finally {
      loading.value = false
      loadingMore.value = false
    }
  }

  // 加载更多
  const loadMore = () => {
    if (!hasMore.value || loadingMore.value) return
    return load(false)
  }

  // 刷新数据
  const refresh = () => load(true)

  // 自动加载
  if (autoLoad) {
    onMounted(() => load())
  }

  return {
    data: readonly(data),
    loading: readonly(loading),
    loadingMore: readonly(loadingMore),
    hasMore: readonly(hasMore),
    error: readonly(error),
    load,
    loadMore,
    refresh
  }
}
