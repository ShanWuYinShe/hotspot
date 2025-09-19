/**
 * 离线同步 Composable
 * 提供离线数据缓存、同步队列、冲突解决
 */

interface SyncOperation {
  id: string
  type: 'create' | 'update' | 'delete'
  resource: string
  data: any
  timestamp: number
  retryCount: number
  maxRetries: number
}

interface OfflineSyncOptions {
  /** 存储键前缀 */
  storagePrefix?: string
  /** 最大重试次数 */
  maxRetries?: number
  /** 重试间隔 (ms) */
  retryInterval?: number
  /** 数据过期时间 (ms) */
  dataExpiry?: number
  /** 是否启用冲突检测 */
  conflictDetection?: boolean
}

interface CachedData<T = any> {
  data: T
  timestamp: number
  version: number
  etag?: string
}

export function useOfflineSync<T = any>(
  resource: string,
  options: OfflineSyncOptions = {}
) {
  const {
    storagePrefix = 'hotspot_offline',
    maxRetries = 3,
    retryInterval = 5000,
    dataExpiry = 24 * 60 * 60 * 1000, // 24小时
    conflictDetection = true
  } = options

  const isOnline = ref(navigator.onLine)
  const syncQueue = ref<SyncOperation[]>([])
  const cachedData = ref<Map<string, CachedData<T>>>(new Map())
  const syncInProgress = ref(false)
  const lastSyncTime = ref<number>(0)

  // 存储键生成
  const getStorageKey = (key: string) => `${storagePrefix}_${resource}_${key}`
  const getQueueKey = () => `${storagePrefix}_queue_${resource}`

  // 本地存储操作
  const saveToStorage = (key: string, data: any) => {
    try {
      localStorage.setItem(getStorageKey(key), JSON.stringify(data))
    } catch (error) {
      console.warn('Failed to save to localStorage:', error)
    }
  }

  const loadFromStorage = <T>(key: string): T | null => {
    try {
      const data = localStorage.getItem(getStorageKey(key))
      return data ? JSON.parse(data) : null
    } catch (error) {
      console.warn('Failed to load from localStorage:', error)
      return null
    }
  }

  const removeFromStorage = (key: string) => {
    try {
      localStorage.removeItem(getStorageKey(key))
    } catch (error) {
      console.warn('Failed to remove from localStorage:', error)
    }
  }

  // 缓存管理
  const setCachedData = (id: string, data: T, version = 1, etag?: string) => {
    const cached: CachedData<T> = {
      data,
      timestamp: Date.now(),
      version,
      etag
    }
    
    cachedData.value.set(id, cached)
    saveToStorage(`data_${id}`, cached)
  }

  const getCachedData = (id: string): CachedData<T> | null => {
    // 先从内存缓存获取
    let cached = cachedData.value.get(id)
    
    // 如果内存中没有，从本地存储加载
    if (!cached) {
      cached = loadFromStorage(`data_${id}`)
      if (cached) {
        cachedData.value.set(id, cached)
      }
    }
    
    // 检查是否过期
    if (cached && Date.now() - cached.timestamp > dataExpiry) {
      cachedData.value.delete(id)
      removeFromStorage(`data_${id}`)
      return null
    }
    
    return cached || null
  }

  const removeCachedData = (id: string) => {
    cachedData.value.delete(id)
    removeFromStorage(`data_${id}`)
  }

  // 同步队列管理
  const addToSyncQueue = (operation: Omit<SyncOperation, 'id' | 'timestamp' | 'retryCount'>) => {
    const syncOp: SyncOperation = {
      id: `${operation.type}_${operation.resource}_${Date.now()}_${Math.random()}`,
      timestamp: Date.now(),
      retryCount: 0,
      maxRetries,
      ...operation
    }
    
    syncQueue.value.push(syncOp)
    saveSyncQueue()
    
    // 如果在线，立即尝试同步
    if (isOnline.value) {
      processSyncQueue()
    }
  }

  const saveSyncQueue = () => {
    saveToStorage('queue', syncQueue.value)
  }

  const loadSyncQueue = () => {
    const queue = loadFromStorage<SyncOperation[]>('queue')
    if (queue) {
      syncQueue.value = queue
    }
  }

  // 同步处理
  const processSyncQueue = async () => {
    if (syncInProgress.value || !isOnline.value || syncQueue.value.length === 0) {
      return
    }

    syncInProgress.value = true

    try {
      const operations = [...syncQueue.value]
      
      for (const operation of operations) {
        try {
          await processOperation(operation)
          
          // 成功后从队列移除
          const index = syncQueue.value.findIndex(op => op.id === operation.id)
          if (index !== -1) {
            syncQueue.value.splice(index, 1)
          }
        } catch (error) {
          console.error('Sync operation failed:', error)
          
          // 增加重试次数
          operation.retryCount++
          
          // 如果超过最大重试次数，移除操作
          if (operation.retryCount >= operation.maxRetries) {
            const index = syncQueue.value.findIndex(op => op.id === operation.id)
            if (index !== -1) {
              syncQueue.value.splice(index, 1)
            }
            
            // 触发错误事件
            emitSyncError({ operation, error })
          }
        }
      }
      
      saveSyncQueue()
      lastSyncTime.value = Date.now()
      
    } finally {
      syncInProgress.value = false
    }
  }

  const processOperation = async (operation: SyncOperation): Promise<void> => {
    const api = useApi()
    
    switch (operation.type) {
      case 'create':
        await api.post(`/${operation.resource}`, operation.data)
        break
        
      case 'update':
        if (conflictDetection) {
          // 检查版本冲突
          const cached = getCachedData(operation.data.id)
          if (cached?.etag) {
            // 使用 ETag 进行冲突检测
            await api.put(`/${operation.resource}/${operation.data.id}`, operation.data, {
              headers: { 'If-Match': cached.etag }
            })
          } else {
            await api.put(`/${operation.resource}/${operation.data.id}`, operation.data)
          }
        } else {
          await api.put(`/${operation.resource}/${operation.data.id}`, operation.data)
        }
        break
        
      case 'delete':
        await api.delete(`/${operation.resource}/${operation.data.id}`)
        removeCachedData(operation.data.id)
        break
    }
  }

  // 数据操作方法
  const create = async (data: Partial<T>): Promise<T> => {
    if (isOnline.value) {
      try {
        const api = useApi()
        const result = await api.post(`/${resource}`, data)
        
        // 缓存结果
        if (result.data?.id) {
          setCachedData(result.data.id, result.data)
        }
        
        return result.data
      } catch (error) {
        // 离线时添加到同步队列
        addToSyncQueue({
          type: 'create',
          resource,
          data,
          maxRetries
        })
        
        // 返回临时数据
        const tempId = `temp_${Date.now()}`
        const tempData = { ...data, id: tempId } as T
        setCachedData(tempId, tempData)
        
        return tempData
      }
    } else {
      // 离线模式
      addToSyncQueue({
        type: 'create',
        resource,
        data,
        maxRetries
      })
      
      const tempId = `temp_${Date.now()}`
      const tempData = { ...data, id: tempId } as T
      setCachedData(tempId, tempData)
      
      return tempData
    }
  }

  const update = async (id: string, data: Partial<T>): Promise<T> => {
    const cached = getCachedData(id)
    const updatedData = { ...cached?.data, ...data, id } as T
    
    if (isOnline.value) {
      try {
        const api = useApi()
        const result = await api.put(`/${resource}/${id}`, updatedData)
        
        // 更新缓存
        setCachedData(id, result.data, (cached?.version || 0) + 1)
        
        return result.data
      } catch (error) {
        // 冲突检测
        if (error.response?.status === 409) {
          emitSyncConflict({ id, localData: updatedData, error })
          throw error
        }
        
        // 其他错误，添加到同步队列
        addToSyncQueue({
          type: 'update',
          resource,
          data: updatedData,
          maxRetries
        })
        
        // 更新本地缓存
        setCachedData(id, updatedData, (cached?.version || 0) + 1)
        
        return updatedData
      }
    } else {
      // 离线模式
      addToSyncQueue({
        type: 'update',
        resource,
        data: updatedData,
        maxRetries
      })
      
      setCachedData(id, updatedData, (cached?.version || 0) + 1)
      
      return updatedData
    }
  }

  const remove = async (id: string): Promise<void> => {
    if (isOnline.value) {
      try {
        const api = useApi()
        await api.delete(`/${resource}/${id}`)
        
        removeCachedData(id)
      } catch (error) {
        addToSyncQueue({
          type: 'delete',
          resource,
          data: { id },
          maxRetries
        })
        
        // 标记为已删除但保留在缓存中
        const cached = getCachedData(id)
        if (cached) {
          setCachedData(id, { ...cached.data, _deleted: true } as T)
        }
      }
    } else {
      addToSyncQueue({
        type: 'delete',
        resource,
        data: { id },
        maxRetries
      })
      
      const cached = getCachedData(id)
      if (cached) {
        setCachedData(id, { ...cached.data, _deleted: true } as T)
      }
    }
  }

  const get = async (id: string): Promise<T | null> => {
    // 先从缓存获取
    const cached = getCachedData(id)
    
    if (isOnline.value) {
      try {
        const api = useApi()
        const headers: Record<string, string> = {}
        
        // 使用 ETag 进行条件请求
        if (cached?.etag) {
          headers['If-None-Match'] = cached.etag
        }
        
        const result = await api.get(`/${resource}/${id}`, { headers })
        
        // 304 Not Modified - 使用缓存数据
        if (result.status === 304 && cached) {
          return cached.data
        }
        
        // 更新缓存
        const etag = result.headers?.etag
        setCachedData(id, result.data, (cached?.version || 0) + 1, etag)
        
        return result.data
      } catch (error) {
        // 网络错误时返回缓存数据
        if (cached) {
          return cached.data
        }
        throw error
      }
    } else {
      // 离线模式返回缓存数据
      return cached?.data || null
    }
  }

  const list = async (params?: any): Promise<T[]> => {
    const cacheKey = `list_${JSON.stringify(params || {})}`
    const cached = getCachedData(cacheKey)
    
    if (isOnline.value) {
      try {
        const api = useApi()
        const result = await api.get(`/${resource}`, { params })
        
        // 缓存列表数据
        setCachedData(cacheKey, result.data)
        
        // 同时缓存单个项目
        if (Array.isArray(result.data)) {
          result.data.forEach((item: any) => {
            if (item.id) {
              setCachedData(item.id, item)
            }
          })
        }
        
        return result.data
      } catch (error) {
        if (cached) {
          return cached.data
        }
        throw error
      }
    } else {
      return cached?.data || []
    }
  }

  // 网络状态监听
  const handleOnline = () => {
    isOnline.value = true
    processSyncQueue()
  }

  const handleOffline = () => {
    isOnline.value = false
  }

  // 事件回调函数
  const eventCallbacks = {
    onSyncError: (callback: (data: { operation: SyncOperation; error: any }) => void) => {
      // 存储回调函数，在需要时调用
      eventCallbacks._syncErrorCallback = callback
    },
    onSyncConflict: (callback: (data: { id: string; localData: T; error: any }) => void) => {
      eventCallbacks._syncConflictCallback = callback
    },
    onSyncComplete: (callback: () => void) => {
      eventCallbacks._syncCompleteCallback = callback
    },
    _syncErrorCallback: null as ((data: { operation: SyncOperation; error: any }) => void) | null,
    _syncConflictCallback: null as ((data: { id: string; localData: T; error: any }) => void) | null,
    _syncCompleteCallback: null as (() => void) | null
  }

  // 触发事件的辅助函数
  const emitSyncError = (data: { operation: SyncOperation; error: any }) => {
    eventCallbacks._syncErrorCallback?.(data)
  }
  
  const emitSyncConflict = (data: { id: string; localData: T; error: any }) => {
    eventCallbacks._syncConflictCallback?.(data)
  }
  
  const emitSyncComplete = () => {
    eventCallbacks._syncCompleteCallback?.()
  }

  // 初始化函数 - 需要在组件中手动调用
  const initialize = () => {
    if (typeof window === 'undefined') return
    
    loadSyncQueue()
    
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    
    // 定期同步
    const syncInterval = setInterval(() => {
      if (isOnline.value && syncQueue.value.length > 0) {
        processSyncQueue()
      }
    }, retryInterval)
    
    // 返回清理函数
    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      clearInterval(syncInterval)
    }
  }

  return {
    // 状态
    isOnline: readonly(isOnline),
    syncInProgress: readonly(syncInProgress),
    syncQueue: readonly(syncQueue),
    lastSyncTime: readonly(lastSyncTime),
    
    // 数据操作
    create,
    update,
    remove,
    get,
    list,
    
    // 缓存管理
    getCachedData,
    setCachedData,
    removeCachedData,
    
    // 同步控制
    processSyncQueue,
    addToSyncQueue,
    
    // 初始化和事件
    initialize,
    ...eventCallbacks
  }
}
