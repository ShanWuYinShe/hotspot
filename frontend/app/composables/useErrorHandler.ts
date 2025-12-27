/**
 * 统一错误处理 Composable
 * 
 * 作用：
 * 1. 统一捕获错误并显示 Toast 通知
 * 2. 避免在每个组件中重复编写 try-catch 逻辑
 * 3. 统一错误消息的提取逻辑
 */

import { getErrorMessage } from '~/utils/error'

export function useErrorHandler() {
  const feedback = useFeedbackStore()

  /**
   * 处理错误并显示通知
   */
  const handleError = (error: unknown, options: { 
    title?: string, 
    fallbackMessage?: string,
    showToast?: boolean
  } = {}) => {
    const { 
      title = '操作失败', 
      fallbackMessage = '发生了未知错误，请稍后重试',
      showToast = true
    } = options

    const description = getErrorMessage(error, fallbackMessage)
    
    console.error(`[ErrorHandler] ${title}:`, error)

    if (showToast) {
      // 默认使用顶部 banner（移动端更友好）；逐步替换掉 toast
      feedback.error(title, description)
    }
    
    return description
  }

  /**
   * 包装异步操作，自动处理错误
   * 
   * @example
   * const data = await withErrorHandling(async () => {
   *   return await api.getData()
   * })
   */
  const withErrorHandling = async <T>(
    fn: () => Promise<T>, 
    options: { 
      title?: string, 
      fallbackMessage?: string,
      loading?: Ref<boolean>,
      showToast?: boolean
    } = {}
  ): Promise<T | undefined> => {
    if (options.loading) options.loading.value = true
    
    try {
      return await fn()
    } catch (error) {
      handleError(error, options)
      return undefined
    } finally {
      if (options.loading) options.loading.value = false
    }
  }

  return {
    handleError,
    withErrorHandling
  }
}

