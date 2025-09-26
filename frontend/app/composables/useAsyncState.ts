/**
 * 通用异步状态管理 composable
 */

export interface AsyncStateOptions<T> {
  immediate?: boolean
  resetOnExecute?: boolean
  onSuccess?: (data: T) => void
  onError?: (error: any) => void
  shallow?: boolean
}

export interface AsyncState<T> {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<any>
  execute: (...args: any[]) => Promise<T | null>
  refresh: () => Promise<T | null>
  reset: () => void
  lastArgs?: Ref<any[]>
}

export const useAsyncState = <T>(
  asyncFn: (...args: any[]) => Promise<T>,
  initialState: T | null = null,
  options: AsyncStateOptions<T> = {}
): AsyncState<T> => {
  const {
    immediate = false,
    resetOnExecute = true,
    onSuccess,
    onError,
    shallow = false
  } = options

  const data = shallow ? shallowRef<T | null>(initialState) : ref<T | null>(initialState)
  const loading = ref(false)
  const error = ref<any>(null)
  const lastArgs = ref<any[]>([])

  const execute = async (...args: any[]): Promise<T | null> => {
    loading.value = true
    if (resetOnExecute) {
      error.value = null
    }
    
    lastArgs.value = args

    try {
      const result = await asyncFn(...args)
      data.value = result
      onSuccess?.(result)
      return result
    } catch (err) {
      error.value = err
      onError?.(err)
      return null
    } finally {
      loading.value = false
    }
  }

  const refresh = () => execute(...lastArgs.value)

  const reset = () => {
    data.value = initialState
    loading.value = false
    error.value = null
    lastArgs.value = []
  }

  if (immediate) {
    execute()
  }

  return {
    data: data as Ref<T | null>,
    loading,
    error,
    execute,
    refresh,
    reset,
    lastArgs
  }
}

/**
 * 分页数据状态管理
 */
export interface PaginatedAsyncStateOptions<T> extends AsyncStateOptions<T[]> {
  pageSize?: number
  append?: boolean
}

export interface PaginatedAsyncState<T> extends AsyncState<T[]> {
  hasMore: Ref<boolean>
  currentPage: Ref<number>
  totalCount: Ref<number>
  loadMore: () => Promise<T[] | null>
  reset: () => void
}

export const usePaginatedAsyncState = <T>(
  asyncFn: (page: number, pageSize: number, ...args: any[]) => Promise<{ items: T[], hasMore: boolean, totalCount?: number }>,
  options: PaginatedAsyncStateOptions<T> = {}
): PaginatedAsyncState<T> => {
  const {
    pageSize = 20,
    append = true,
    ...baseOptions
  } = options

  const currentPage = ref(1)
  const hasMore = ref(true)
  const totalCount = ref(0)

  const wrappedAsyncFn = async (page: number, ...args: any[]) => {
    const result = await asyncFn(page, pageSize, ...args)
    hasMore.value = result.hasMore
    if (result.totalCount !== undefined) {
      totalCount.value = result.totalCount
    }
    return result.items
  }

  const baseState = useAsyncState(wrappedAsyncFn, [], {
    ...baseOptions,
    onSuccess: (newData) => {
      if (append && currentPage.value > 1 && baseState.data.value) {
        baseState.data.value = [...baseState.data.value, ...newData]
      }
      baseOptions.onSuccess?.(newData)
    }
  })

  const execute = async (...args: any[]) => {
    currentPage.value = 1
    return baseState.execute(1, ...args)
  }

  const loadMore = async () => {
    if (!hasMore.value || baseState.loading.value) return null
    currentPage.value += 1
    // 使用 baseState 的 lastArgs 引用
    return baseState.execute(currentPage.value, ...baseState.lastArgs!.value.slice(1))
  }

  const reset = () => {
    currentPage.value = 1
    hasMore.value = true
    totalCount.value = 0
    baseState.reset()
  }

  return {
    ...baseState,
    hasMore,
    currentPage,
    totalCount,
    execute,
    loadMore,
    reset
  }
}

/**
 * 列表操作状态管理
 */
export interface ListOperationsOptions<T> {
  keyField?: keyof T
  onItemAdded?: (item: T) => void
  onItemUpdated?: (item: T) => void
  onItemRemoved?: (item: T) => void
}

export const useListOperations = <T extends Record<string, any>>(
  list: Ref<T[]>,
  options: ListOperationsOptions<T> = {}
) => {
  const { keyField = 'id' as keyof T, onItemAdded, onItemUpdated, onItemRemoved } = options

  const addItem = (item: T) => {
    list.value.unshift(item)
    onItemAdded?.(item)
  }

  const updateItem = (updatedItem: T) => {
    const index = list.value.findIndex(item => item[keyField] === updatedItem[keyField])
    if (index !== -1) {
      list.value[index] = { ...list.value[index], ...updatedItem }
      onItemUpdated?.(updatedItem)
    }
  }

  const removeItem = (itemOrKey: T | T[typeof keyField]) => {
    const key = typeof itemOrKey === 'object' ? itemOrKey[keyField] : itemOrKey
    const index = list.value.findIndex(item => item[keyField] === key)
    if (index !== -1) {
      const removedItem = list.value.splice(index, 1)[0]
      onItemRemoved?.(removedItem)
      return removedItem
    }
    return null
  }

  const findItem = (key: T[typeof keyField]): T | undefined => {
    return list.value.find(item => item[keyField] === key)
  }

  const toggleItem = (item: T, condition?: (item: T) => boolean) => {
    const exists = condition ? condition(item) : findItem(item[keyField])
    if (exists) {
      removeItem(item)
      return false
    } else {
      addItem(item)
      return true
    }
  }

  return {
    addItem,
    updateItem,
    removeItem,
    findItem,
    toggleItem
  }
}
