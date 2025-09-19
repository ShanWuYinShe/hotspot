/**
 * 错误边界 Composable
 * 提供全局错误捕获、错误恢复、错误报告
 */

interface ErrorInfo {
  error: Error
  errorInfo?: {
    componentStack?: string
    errorBoundary?: string
  }
  timestamp: number
  url: string
  userAgent: string
  userId?: string
  context?: Record<string, any>
}

interface ErrorBoundaryOptions {
  /** 是否启用错误报告 */
  enableReporting?: boolean
  /** 报告端点 */
  reportingEndpoint?: string
  /** 最大错误缓存数量 */
  maxErrorCache?: number
  /** 是否在开发环境显示详细错误 */
  showDetailInDev?: boolean
  /** 错误恢复策略 */
  recoveryStrategy?: 'reload' | 'redirect' | 'retry' | 'ignore'
  /** 自定义错误处理器 */
  customHandler?: (error: ErrorInfo) => void
}

export function useErrorBoundary(options: ErrorBoundaryOptions = {}) {
  const {
    enableReporting = true,
    reportingEndpoint = '/api/errors',
    maxErrorCache = 50,
    showDetailInDev = true,
    recoveryStrategy = 'retry',
    customHandler
  } = options

  const errors = ref<ErrorInfo[]>([])
  const isRecovering = ref(false)
  const recoveryAttempts = ref(0)
  const maxRecoveryAttempts = 3

  // 错误分类
  const categorizeError = (error: Error): string => {
    if (error.name === 'ChunkLoadError') return 'chunk-load'
    if (error.message.includes('Network')) return 'network'
    if (error.message.includes('Permission')) return 'permission'
    if (error.message.includes('Timeout')) return 'timeout'
    if (error.name === 'TypeError') return 'type'
    if (error.name === 'ReferenceError') return 'reference'
    if (error.name === 'SyntaxError') return 'syntax'
    return 'unknown'
  }

  // 创建错误信息
  const createErrorInfo = (error: Error, context?: Record<string, any>): ErrorInfo => {
    return {
      error,
      timestamp: Date.now(),
      url: window.location.href,
      userAgent: navigator.userAgent,
      userId: getCurrentUserId(), // 需要实现获取当前用户ID的函数
      context: {
        category: categorizeError(error),
        recoveryAttempts: recoveryAttempts.value,
        ...context
      }
    }
  }

  // 获取当前用户ID（示例实现）
  const getCurrentUserId = (): string | undefined => {
    try {
      const authStore = useAuthStore()
      return authStore.user?.user_id
    } catch {
      return undefined
    }
  }

  // 错误报告
  const reportError = async (errorInfo: ErrorInfo) => {
    if (!enableReporting) return

    try {
      const api = useApi()
      await api.post(reportingEndpoint, {
        message: errorInfo.error.message,
        stack: errorInfo.error.stack,
        timestamp: errorInfo.timestamp,
        url: errorInfo.url,
        userAgent: errorInfo.userAgent,
        userId: errorInfo.userId,
        context: errorInfo.context
      })
    } catch (reportingError) {
      console.error('Failed to report error:', reportingError)
    }
  }

  // 错误恢复策略
  const recoverFromError = async (errorInfo: ErrorInfo) => {
    if (isRecovering.value) return
    
    isRecovering.value = true
    recoveryAttempts.value++

    try {
      switch (recoveryStrategy) {
        case 'reload':
          if (recoveryAttempts.value <= maxRecoveryAttempts) {
            setTimeout(() => {
              window.location.reload()
            }, 1000)
          }
          break

        case 'redirect':
          await navigateTo('/')
          break

        case 'retry':
          // 重试上一个操作（需要具体实现）
          if (recoveryAttempts.value <= maxRecoveryAttempts) {
            setTimeout(() => {
              isRecovering.value = false
            }, 2000)
          }
          break

        case 'ignore':
          // 忽略错误，继续执行
          break
      }
    } catch (recoveryError) {
      console.error('Recovery failed:', recoveryError)
    } finally {
      if (recoveryStrategy !== 'retry' || recoveryAttempts.value > maxRecoveryAttempts) {
        isRecovering.value = false
      }
    }
  }

  // 处理错误
  const handleError = async (error: Error, context?: Record<string, any>) => {
    const errorInfo = createErrorInfo(error, context)
    
    // 添加到错误列表
    errors.value.unshift(errorInfo)
    
    // 限制错误缓存数量
    if (errors.value.length > maxErrorCache) {
      errors.value = errors.value.slice(0, maxErrorCache)
    }

    // 自定义处理器
    if (customHandler) {
      try {
        customHandler(errorInfo)
      } catch (handlerError) {
        console.error('Custom error handler failed:', handlerError)
      }
    }

    // 错误报告
    await reportError(errorInfo)

    // 错误恢复
    await recoverFromError(errorInfo)

    // 开发环境显示详细错误
    if (import.meta.dev && showDetailInDev) {
      console.group('🚨 Error Boundary Caught Error')
      console.error('Error:', error)
      console.log('Context:', context)
      console.log('Error Info:', errorInfo)
      console.groupEnd()
    }
  }

  // 全局错误处理器
  const setupGlobalErrorHandlers = () => {
    // JavaScript 错误
    window.addEventListener('error', (event) => {
      handleError(event.error || new Error(event.message), {
        type: 'javascript',
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno
      })
    })

    // Promise 拒绝
    window.addEventListener('unhandledrejection', (event) => {
      const error = event.reason instanceof Error 
        ? event.reason 
        : new Error(String(event.reason))
      
      handleError(error, {
        type: 'promise-rejection'
      })
    })

    // Vue 错误处理
    const app = getCurrentInstance()?.appContext.app
    if (app) {
      app.config.errorHandler = (error: Error, instance, info) => {
        handleError(error, {
          type: 'vue',
          component: instance?.$options.name || 'Unknown',
          info
        })
      }
    }
  }

  // 清理错误
  const clearErrors = () => {
    errors.value = []
    recoveryAttempts.value = 0
  }

  // 重试操作
  const retry = () => {
    if (recoveryAttempts.value < maxRecoveryAttempts) {
      isRecovering.value = false
      recoveryAttempts.value = 0
    }
  }

  // 获取错误统计
  const getErrorStats = () => {
    const stats = {
      total: errors.value.length,
      byCategory: {} as Record<string, number>,
      recent: errors.value.slice(0, 10)
    }

    errors.value.forEach(errorInfo => {
      const category = errorInfo.context?.category || 'unknown'
      stats.byCategory[category] = (stats.byCategory[category] || 0) + 1
    })

    return stats
  }

  // 导出错误日志
  const exportErrorLogs = () => {
    const logs = errors.value.map(errorInfo => ({
      timestamp: new Date(errorInfo.timestamp).toISOString(),
      message: errorInfo.error.message,
      stack: errorInfo.error.stack,
      url: errorInfo.url,
      context: errorInfo.context
    }))

    const blob = new Blob([JSON.stringify(logs, null, 2)], {
      type: 'application/json'
    })

    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `error-logs-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // 初始化
  onMounted(() => {
    setupGlobalErrorHandlers()
  })

  return {
    // 状态
    errors: readonly(errors),
    isRecovering: readonly(isRecovering),
    recoveryAttempts: readonly(recoveryAttempts),

    // 方法
    handleError,
    clearErrors,
    retry,
    getErrorStats,
    exportErrorLogs
  }
}

/**
 * 错误边界组件
 */
export const ErrorBoundary = defineComponent({
  name: 'ErrorBoundary',
  props: {
    fallback: {
      type: [String, Object, Function] as PropType<string | Component | (() => VNode)>,
      default: null
    },
    onError: {
      type: Function as PropType<(error: Error, errorInfo: any) => void>,
      default: null
    }
  },
  setup(props, { slots }) {
    const hasError = ref(false)
    const error = ref<Error | null>(null)
    
    const { handleError, retry } = useErrorBoundary({
      customHandler: props.onError
    })

    // 错误捕获
    onErrorCaptured((err: Error, instance, info) => {
      hasError.value = true
      error.value = err
      
      handleError(err, {
        component: instance?.$options.name || 'Unknown',
        info
      })
      
      return false // 阻止错误继续传播
    })

    // 重置错误状态
    const resetError = () => {
      hasError.value = false
      error.value = null
      retry()
    }

    return () => {
      if (hasError.value) {
        // 自定义错误回退
        if (props.fallback) {
          if (typeof props.fallback === 'string') {
            return h('div', { class: 'error-boundary' }, props.fallback)
          } else if (typeof props.fallback === 'function') {
            return props.fallback()
          } else {
            return h(props.fallback as Component, {
              error: error.value,
              resetError
            })
          }
        }

        // 默认错误界面
        return h('div', {
          class: 'flex flex-col items-center justify-center min-h-64 p-8 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800'
        }, [
          h('div', {
            class: 'w-16 h-16 mb-4 rounded-full bg-red-100 dark:bg-red-900/40 flex items-center justify-center'
          }, [
            h('svg', {
              class: 'w-8 h-8 text-red-500',
              fill: 'none',
              stroke: 'currentColor',
              viewBox: '0 0 24 24'
            }, [
              h('path', {
                'stroke-linecap': 'round',
                'stroke-linejoin': 'round',
                'stroke-width': '2',
                d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z'
              })
            ])
          ]),
          h('h3', {
            class: 'text-lg font-semibold text-red-900 dark:text-red-100 mb-2'
          }, '出现了一些问题'),
          h('p', {
            class: 'text-red-700 dark:text-red-300 text-center mb-4 max-w-md'
          }, '应用遇到了意外错误。我们已经记录了这个问题，请稍后重试。'),
          import.meta.dev && h('details', {
            class: 'mb-4 text-sm text-red-600 dark:text-red-400'
          }, [
            h('summary', { class: 'cursor-pointer' }, '错误详情'),
            h('pre', {
              class: 'mt-2 p-2 bg-red-100 dark:bg-red-900/60 rounded text-xs overflow-auto'
            }, error.value?.stack || error.value?.message)
          ]),
          h('button', {
            class: 'px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors',
            onClick: resetError
          }, '重试')
        ])
      }

      return slots.default?.()
    }
  }
})
