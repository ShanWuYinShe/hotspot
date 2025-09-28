/**
 * 简单的错误处理 Composable
 * 专注于基本的错误处理和用户通知
 */

export function useErrorHandler() {
  const { error: showError } = useNotifications()

  /**
   * 处理错误并显示用户友好的消息
   */
  const handle = (error: Error | unknown, userMessage?: string) => {
    console.error('Error:', error)
    
    // 显示用户友好的错误消息
    if (userMessage) {
      showError(userMessage)
    } else if (error instanceof Error) {
      showError(error.message)
    } else {
      showError('操作失败，请重试')
    }
  }

  /**
   * 包装异步函数，自动处理错误
   */
  const wrap = async <T>(
    operation: () => Promise<T>,
    userMessage?: string
  ): Promise<T | null> => {
    try {
      return await operation()
    } catch (err) {
      handle(err, userMessage)
      return null
    }
  }

  return {
    handle,
    wrap
  }
}