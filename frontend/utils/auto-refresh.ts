/**
 * 自动刷新工具函数
 * 
 * 从 useAutoRefresh composable 迁移而来的简化版本
 * 提供灵活的自动刷新功能，不依赖 UI 库
 */

export interface AutoRefreshOptions {
  /** 刷新间隔（毫秒） */
  interval: number
  /** 是否立即执行一次 */
  immediate?: boolean
  /** 当页面不可见时是否继续刷新 */
  refreshWhenHidden?: boolean
  /** 最大重试次数 */
  maxRetries?: number
  /** 错误回调 */
  onError?: (error: Error, retryCount: number) => void
  /** 成功回调 */
  onSuccess?: (data: any) => void
  /** 开始刷新回调 */
  onStart?: () => void
  /** 停止刷新回调 */
  onStop?: () => void
}

/**
 * 创建自动刷新管理器 - Composition API 版本
 */
export function useAutoRefreshManager(
  refreshFunction: () => Promise<any>,
  options: AutoRefreshOptions
) {
  const timerId = ref<NodeJS.Timeout | null>(null)
  const isRunning = ref(false)
  const retryCount = ref(0)
  let visibilityCleanup: (() => void) | null = null

  const setupVisibilityListener = () => {
    if (!options.refreshWhenHidden) {
      const handleVisibilityChange = () => {
        if (document.hidden) {
          pause()
        } else if (isRunning.value) {
          resume()
        }
      }

      document.addEventListener('visibilitychange', handleVisibilityChange)
      visibilityCleanup = () => {
        document.removeEventListener('visibilitychange', handleVisibilityChange)
      }
    }
  }

  const executeRefresh = async () => {
    try {
      const result = await refreshFunction()
      retryCount.value = 0 // 重置重试计数
      options.onSuccess?.(result)
      return result
    } catch (error) {
      retryCount.value++
      const maxRetries = options.maxRetries ?? 3

      options.onError?.(error as Error, retryCount.value)

      if (retryCount.value >= maxRetries) {
        console.error(`自动刷新失败，已达到最大重试次数 ${maxRetries}`)
        stop()
      }

      throw error
    }
  }

  const scheduleNext = () => {
    if (!isRunning.value) return

    timerId.value = setTimeout(async () => {
      if (!isRunning.value) return

      await executeRefresh()
      scheduleNext()
    }, options.interval)
  }

  const start = async () => {
    if (isRunning.value) return

    isRunning.value = true
    retryCount.value = 0
    options.onStart?.()

    if (options.immediate) {
      await executeRefresh()
    }

    scheduleNext()
  }

  const stop = () => {
    if (!isRunning.value) return

    isRunning.value = false
    if (timerId.value) {
      clearTimeout(timerId.value)
      timerId.value = null
    }
    options.onStop?.()
  }

  const pause = () => {
    if (timerId.value) {
      clearTimeout(timerId.value)
      timerId.value = null
    }
  }

  const resume = () => {
    if (isRunning.value && !timerId.value) {
      scheduleNext()
    }
  }

  const updateInterval = (newInterval: number) => {
    options.interval = newInterval
    if (isRunning.value) {
      pause()
      resume()
    }
  }

  const cleanup = () => {
    stop()
    visibilityCleanup?.()
  }

  // 初始化可见性监听器
  if (process.client) {
    setupVisibilityListener()
  }

  return {
    isRunning: readonly(isRunning),
    retryCount: readonly(retryCount),
    start,
    stop,
    pause,
    resume,
    updateInterval,
    cleanup
  }
}

/**
 * 创建自动刷新实例
 */
export function createAutoRefresh(
  refreshFunction: () => Promise<any>,
  options: AutoRefreshOptions
) {
  return useAutoRefreshManager(refreshFunction, options)
}

/**
 * 简单的自动刷新钩子（适用于在组件中使用）
 */
export function useSimpleAutoRefresh(
  refreshFunction: () => Promise<any>,
  interval: number = 30000,
  options: Partial<AutoRefreshOptions> = {}
) {
  const manager = createAutoRefresh(refreshFunction, {
    interval,
    immediate: true,
    refreshWhenHidden: false,
    maxRetries: 3,
    ...options
  })

  return {
    start: () => manager.start(),
    stop: () => manager.stop(),
    pause: () => manager.pause(),
    resume: () => manager.resume(),
    updateInterval: (newInterval: number) => manager.updateInterval(newInterval),
    cleanup: () => manager.cleanup()
  }
}

/**
 * 页面级自动刷新设置
 */
export function setupPageAutoRefresh(
  refreshFunctions: Array<{
    name: string
    fn: () => Promise<any>
    interval: number
    options?: Partial<AutoRefreshOptions>
  }>,
  globalOptions: Partial<AutoRefreshOptions> = {}
) {
  const managers = new Map<string, ReturnType<typeof useAutoRefreshManager>>()

  // 创建所有管理器
  refreshFunctions.forEach(({ name, fn, interval, options = {} }) => {
    const manager = useAutoRefreshManager(fn, {
      interval,
      immediate: true,
      refreshWhenHidden: false,
      maxRetries: 3,
      ...globalOptions,
      ...options
    })
    managers.set(name, manager)
  })

  return {
    // 启动所有自动刷新
    startAll: () => {
      managers.forEach(manager => manager.start())
    },

    // 停止所有自动刷新
    stopAll: () => {
      managers.forEach(manager => manager.stop())
    },

    // 暂停所有自动刷新
    pauseAll: () => {
      managers.forEach(manager => manager.pause())
    },

    // 恢复所有自动刷新
    resumeAll: () => {
      managers.forEach(manager => manager.resume())
    },

    // 控制单个刷新器
    start: (name: string) => managers.get(name)?.start(),
    stop: (name: string) => managers.get(name)?.stop(),
    pause: (name: string) => managers.get(name)?.pause(),
    resume: (name: string) => managers.get(name)?.resume(),

    // 更新间隔
    updateInterval: (name: string, interval: number) => {
      managers.get(name)?.updateInterval(interval)
    },

    // 清理所有资源
    cleanup: () => {
      managers.forEach(manager => manager.cleanup())
      managers.clear()
    }
  }
}

/**
 * 智能刷新 - 根据网络状态和页面可见性调整刷新策略
 */
export function createSmartRefresh(
  refreshFunction: () => Promise<any>,
  baseInterval: number = 30000
) {
  let currentInterval = baseInterval
  let isOnline = navigator.onLine
  let isVisible = !document.hidden

  const manager = useAutoRefreshManager(refreshFunction, {
    interval: currentInterval,
    immediate: true,
    refreshWhenHidden: false,
    onError: (error, retryCount) => {
      console.warn(`自动刷新失败 (第${retryCount}次重试):`, error.message)
      // 网络错误时增加间隔
      if (retryCount > 1) {
        currentInterval = Math.min(currentInterval * 1.5, 300000) // 最多5分钟
        manager.updateInterval(currentInterval)
      }
    },
    onSuccess: () => {
      // 成功时恢复正常间隔
      if (currentInterval !== baseInterval) {
        currentInterval = baseInterval
        manager.updateInterval(currentInterval)
      }
    }
  })

  // 监听网络状态
  const handleOnline = () => {
    isOnline = true
    if (isVisible) {
      manager.resume()
    }
  }

  const handleOffline = () => {
    isOnline = false
    manager.pause()
  }

  // 监听页面可见性
  const handleVisibilityChange = () => {
    isVisible = !document.hidden
    if (isVisible && isOnline) {
      manager.resume()
    } else {
      manager.pause()
    }
  }

  if (process.client) {
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)
    document.addEventListener('visibilitychange', handleVisibilityChange)
  }

  return {
    start: () => manager.start(),
    stop: () => manager.stop(),
    cleanup: () => {
      manager.cleanup()
      if (process.client) {
        window.removeEventListener('online', handleOnline)
        window.removeEventListener('offline', handleOffline)
        document.removeEventListener('visibilitychange', handleVisibilityChange)
      }
    }
  }
}
