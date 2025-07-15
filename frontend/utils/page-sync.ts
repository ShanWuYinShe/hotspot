/**
 * 页面同步工具函数
 * 
 * 从 usePageSync composable 迁移而来的简化版本
 * 处理页面间的状态同步和通信
 */

/**
 * 页面可见性检测
 */
export function createVisibilityObserver(
  onVisible: () => void,
  onHidden: () => void
) {
  const handleVisibilityChange = () => {
    if (document.hidden) {
      onHidden()
    } else {
      onVisible()
    }
  }

  document.addEventListener('visibilitychange', handleVisibilityChange)

  return () => {
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }
}

/**
 * 窗口焦点检测
 */
export function createFocusObserver(
  onFocus: () => void,
  onBlur: () => void
) {
  window.addEventListener('focus', onFocus)
  window.addEventListener('blur', onBlur)

  return () => {
    window.removeEventListener('focus', onFocus)
    window.removeEventListener('blur', onBlur)
  }
}

/**
 * 页面同步状态管理器 - Composition API 版本
 */
export function usePageSyncManager(storage: Storage = localStorage) {
  const listeners = ref<Map<string, Set<Function>>>(new Map())
  
  const initStorageListener = () => {
    const handleStorageChange = (event: StorageEvent) => {
      if (event.key && event.newValue) {
        const keyListeners = listeners.value.get(event.key)
        if (keyListeners) {
          try {
            const data = JSON.parse(event.newValue)
            keyListeners.forEach(listener => listener(data))
          } catch (error) {
            console.error('页面同步数据解析失败:', error)
          }
        }
      }
    }

    if (process.client) {
      window.addEventListener('storage', handleStorageChange)
      
      // 返回清理函数
      return () => {
        window.removeEventListener('storage', handleStorageChange)
      }
    }
    
    return () => {}
  }

  // 发送数据到其他页面
  const send = (key: string, data: any) => {
    try {
      storage.setItem(key, JSON.stringify({
        data,
        timestamp: Date.now(),
        id: Math.random().toString(36).substring(2)
      }))
    } catch (error) {
      console.error('页面同步数据发送失败:', error)
    }
  }

  // 监听来自其他页面的数据
  const listen = (key: string, callback: Function) => {
    if (!listeners.value.has(key)) {
      listeners.value.set(key, new Set())
    }
    listeners.value.get(key)!.add(callback)

    // 返回取消监听的函数
    return () => {
      const keyListeners = listeners.value.get(key)
      if (keyListeners) {
        keyListeners.delete(callback)
        if (keyListeners.size === 0) {
          listeners.value.delete(key)
        }
      }
    }
  }

  // 清理所有监听器
  const cleanup = () => {
    listeners.value.clear()
  }

  // 初始化存储监听器
  const storageCleanup = initStorageListener()

  return {
    listeners: readonly(listeners),
    send,
    listen,
    cleanup: () => {
      cleanup()
      storageCleanup()
    }
  }
}

/**
 * 创建页面同步实例
 */
export function createPageSync(storage?: Storage) {
  return usePageSyncManager(storage)
}

/**
 * 页面生命周期钩子
 */
export function createPageLifecycle(options: {
  onPageShow?: () => void
  onPageHide?: () => void
  onPageFocus?: () => void
  onPageBlur?: () => void
  onBeforeUnload?: (event: BeforeUnloadEvent) => void
}) {
  const cleanupFunctions: (() => void)[] = []

  // 页面可见性
  if (options.onPageShow || options.onPageHide) {
    const cleanup = createVisibilityObserver(
      options.onPageShow || (() => {}),
      options.onPageHide || (() => {})
    )
    cleanupFunctions.push(cleanup)
  }

  // 窗口焦点
  if (options.onPageFocus || options.onPageBlur) {
    const cleanup = createFocusObserver(
      options.onPageFocus || (() => {}),
      options.onPageBlur || (() => {})
    )
    cleanupFunctions.push(cleanup)
  }

  // 页面卸载
  if (options.onBeforeUnload) {
    window.addEventListener('beforeunload', options.onBeforeUnload)
    cleanupFunctions.push(() => {
      window.removeEventListener('beforeunload', options.onBeforeUnload!)
    })
  }

  // 返回清理函数
  return () => {
    cleanupFunctions.forEach(cleanup => cleanup())
  }
}

/**
 * 简单的页面间消息传递
 */
export function sendPageMessage(type: string, payload: any = {}) {
  const message = {
    type,
    payload,
    timestamp: Date.now(),
    source: 'hotspot-app'
  }

  try {
    localStorage.setItem('page_message', JSON.stringify(message))
    // 立即清除，只用于触发事件
    localStorage.removeItem('page_message')
  } catch (error) {
    console.error('发送页面消息失败:', error)
  }
}

/**
 * 监听页面间消息
 */
export function listenPageMessage(
  type: string | string[],
  callback: (payload: any) => void
) {
  const types = Array.isArray(type) ? type : [type]

  const handleStorageChange = (event: StorageEvent) => {
    if (event.key === 'page_message' && event.newValue) {
      try {
        const message = JSON.parse(event.newValue)
        if (message.source === 'hotspot-app' && types.includes(message.type)) {
          callback(message.payload)
        }
      } catch (error) {
        console.error('解析页面消息失败:', error)
      }
    }
  }

  window.addEventListener('storage', handleStorageChange)

  return () => {
    window.removeEventListener('storage', handleStorageChange)
  }
}
