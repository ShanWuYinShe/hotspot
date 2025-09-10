/**
 * 通用异步操作 Composable
 * 提供统一的加载状态管理和错误处理
 */

interface AsyncOperationOptions {
  /** 成功提示消息 */
  successMessage?: string
  /** 错误提示标题 */
  errorTitle?: string
  /** 是否显示成功提示，默认 true */
  showSuccessToast?: boolean
  /** 是否显示错误提示，默认 true */
  showErrorToast?: boolean
  /** 自定义错误处理函数 */
  onError?: (error: any) => void
  /** 自定义成功处理函数 */
  onSuccess?: (result: any) => void
}

/**
 * 异步操作 composable
 * @param operation 异步操作函数
 * @param options 配置选项
 */
export function useAsyncOperation<T = any>(
  operation: () => Promise<T>,
  options: AsyncOperationOptions = {}
) {
  const {
    successMessage,
    errorTitle = "操作失败",
    showSuccessToast = true,
    showErrorToast = true,
    onError,
    onSuccess
  } = options

  const isLoading = ref(false)
  const error = ref<any>(null)
  const data = ref<T | null>(null)

  const toast = useSmartToast()

  const execute = async (): Promise<T | null> => {
    if (isLoading.value) return null

    isLoading.value = true
    error.value = null

    try {
      const result = await operation()
      data.value = result

      // 成功处理
      if (onSuccess) {
        onSuccess(result)
      }
      
      if (showSuccessToast && successMessage) {
        toast.add({ 
          title: successMessage, 
          color: "success" 
        })
      }

      return result
    } catch (err: any) {
      error.value = err

      // 错误处理
      if (onError) {
        onError(err)
      } else if (showErrorToast) {
        toast.add({
          title: errorTitle,
          description: err?.message || "请稍后重试",
          color: "error"
        })
      }

      throw err
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading: readonly(isLoading),
    error: readonly(error),
    data: readonly(data),
    execute
  }
}

/**
 * 简化版异步操作，只处理加载状态
 * @param operation 异步操作函数
 */
export function useSimpleAsync<T = any>(operation: () => Promise<T>) {
  const isLoading = ref(false)

  const execute = async (): Promise<T | null> => {
    if (isLoading.value) return null

    isLoading.value = true
    try {
      return await operation()
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading: readonly(isLoading),
    execute
  }
}

/**
 * 批量异步操作
 * @param operations 异步操作数组
 * @param options 配置选项
 */
export function useBatchAsync<T = any>(
  operations: Array<() => Promise<T>>,
  options: AsyncOperationOptions = {}
) {
  const isLoading = ref(false)
  const errors = ref<any[]>([])
  const results = ref([]) as Ref<T[]>

  const toast = useSmartToast()

  const execute = async (): Promise<T[]> => {
    if (isLoading.value) return []

    isLoading.value = true
    errors.value = []
    results.value = []

    try {
      const promises = operations.map(async (op, index) => {
        try {
          return await op()
        } catch (err) {
          errors.value[index] = err
          throw err
        }
      })

      const allResults = await Promise.allSettled(promises)
      
      allResults.forEach((result, index) => {
        if (result.status === 'fulfilled') {
          results.value[index] = result.value as T
        }
      })

      const successCount = allResults.filter(r => r.status === 'fulfilled').length
      const errorCount = allResults.length - successCount

      if (options.showSuccessToast && successCount > 0) {
        toast.add({
          title: `成功完成 ${successCount} 项操作`,
          color: "success"
        })
      }

      if (options.showErrorToast && errorCount > 0) {
        toast.add({
          title: `${errorCount} 项操作失败`,
          description: "部分操作未能完成",
          color: "warning"
        })
      }

      return results.value as T[]
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading: readonly(isLoading),
    errors: readonly(errors),
    results: readonly(results),
    execute
  }
}

