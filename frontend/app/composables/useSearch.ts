/**
 * 通用搜索功能 - 统一搜索逻辑和状态管理
 */

interface SearchOptions {
  debounceMs?: number
  minLength?: number
}

export function useSearch<T>(
  searchFunction: (query: string, filters?: any) => Promise<T[]>,
  options: SearchOptions = {}
) {
  const { debounceMs = 300, minLength = 1 } = options

  // 状态管理
  const query = ref('')
  const results = ref<T[]>([])
  const searching = ref(false)
  const filters = ref<any>({})

  const { debounce } = useUtils()

  // 执行搜索
  const performSearch = async () => {
    if (query.value.length < minLength) {
      results.value = []
      return
    }

    searching.value = true
    try {
      results.value = await searchFunction(query.value, filters.value)
    } catch (error) {
      console.error('Search error:', error)
      results.value = []
    } finally {
      searching.value = false
    }
  }

  // 防抖搜索
  const debouncedSearch = debounce(performSearch, debounceMs)

  // 监听查询变化
  watch(query, () => {
    if (query.value.length === 0) {
      results.value = []
      return
    }
    debouncedSearch()
  })

  // 监听筛选条件变化
  watch(filters, () => {
    if (query.value.length >= minLength) {
      performSearch()
    }
  }, { deep: true })

  // 清空搜索
  const clear = () => {
    query.value = ''
    results.value = []
    filters.value = {}
  }

  return {
    query,
    results: readonly(results),
    searching: readonly(searching),
    filters,
    performSearch,
    clear
  }
}
