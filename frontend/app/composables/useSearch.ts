/**
 * 简单的搜索 Composable
 * 专注于基本的搜索功能和防抖
 */

export interface SearchResult<T = any> {
  items: T[]
  total: number
  hasMore: boolean
  cursor?: string
}

export function useSearch<T = any>(
  searchFn: (query: string, options?: any) => Promise<SearchResult<T>>,
  options: { debounceMs?: number; minLength?: number } = {}
) {
  const { debounceMs = 300, minLength = 1 } = options

  // 状态
  const query = ref('')
  const results = ref<T[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(false)
  const cursor = ref<string | undefined>()
  const total = ref(0)

  // 计算属性
  const isEmpty = computed(() => results.value.length === 0)
  const hasResults = computed(() => results.value.length > 0)

  /**
   * 执行搜索
   */
  const executeSearch = async (searchQuery: string, loadMore = false) => {
    if (!searchQuery || searchQuery.length < minLength) {
      if (!loadMore) {
        results.value = []
        total.value = 0
        hasMore.value = false
        cursor.value = undefined
      }
      return
    }

    loading.value = true
    error.value = null

    try {
      const result = await searchFn(searchQuery, {
        cursor: loadMore ? cursor.value : undefined
      })

      if (loadMore) {
        results.value.push(...result.items)
      } else {
        results.value = result.items
      }

      total.value = result.total
      hasMore.value = result.hasMore
      cursor.value = result.cursor
    } catch (err) {
      error.value = err instanceof Error ? err.message : '搜索失败'
      console.error('Search error:', err)
    } finally {
      loading.value = false
    }
  }

  /**
   * 防抖搜索
   */
  const debouncedSearch = useDebounceFn(executeSearch, debounceMs)

  /**
   * 设置查询并搜索
   */
  const search = (searchQuery: string) => {
    query.value = searchQuery
    debouncedSearch(searchQuery, false)
  }

  /**
   * 加载更多
   */
  const loadMore = () => {
    if (hasMore.value && !loading.value) {
      executeSearch(query.value, true)
    }
  }

  /**
   * 清空结果
   */
  const clear = () => {
    query.value = ''
    results.value = []
    total.value = 0
    hasMore.value = false
    cursor.value = undefined
    error.value = null
  }

  return {
    // 状态
    query: readonly(query),
    results: readonly(results),
    loading: readonly(loading),
    error: readonly(error),
    hasMore: readonly(hasMore),
    total: readonly(total),
    
    // 计算属性
    isEmpty,
    hasResults,
    
    // 方法
    search,
    loadMore,
    clear,
    executeSearch
  }
}