// 错误处理 composable
export const useErrorHandler = () => {
  const toast = useToast()

  const handleError = (error: any, context?: string) => {
    console.error(`[ErrorHandler] ${context || 'Unknown'}:`, error)
    
    let title = '操作失败'
    let description = '请稍后重试'
    
    if (error?.message) {
      description = error.message
    } else if (typeof error === 'string') {
      description = error
    }
    
    if (context) {
      title = `${context}失败`
    }
    
    toast.add({
      title,
      description,
      color: 'error'
    })
  }

  const handleApiError = (error: any, operation?: string) => {
    if (error?.status === 401) {
      handleError('登录已过期，请重新登录', operation)
      // 可以在这里触发登出逻辑
      return
    }
    
    if (error?.status === 403) {
      handleError('没有权限执行此操作', operation)
      return
    }
    
    if (error?.status === 404) {
      handleError('请求的资源不存在', operation)
      return
    }
    
    if (error?.status >= 500) {
      handleError('服务器错误，请稍后重试', operation)
      return
    }
    
    handleError(error, operation)
  }

  const handleNetworkError = (error: any, operation?: string) => {
    if (!navigator.onLine) {
      handleError('网络连接已断开，请检查网络设置', operation)
      return
    }
    
    handleError('网络请求失败，请检查网络连接', operation)
  }

  return {
    handleError,
    handleApiError,
    handleNetworkError
  }
}

