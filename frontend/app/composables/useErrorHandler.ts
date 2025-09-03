/**
 * 全局错误处理 Composable
 * 提供统一的错误处理和用户友好的错误提示
 */
export const useErrorHandler = () => {
  const toast = useToast()

  /**
   * 处理 API 错误
   */
  const handleApiError = (error: any, context?: string) => {
    console.error(`[API Error${context ? ` - ${context}` : ''}]:`, error)
    
    let title = '操作失败'
    let description = '请稍后重试'
    
    // 根据错误类型提供更具体的提示
    if (error?.status === 401) {
      title = '登录已过期'
      description = '请重新登录后继续操作'
    } else if (error?.status === 403) {
      title = '权限不足'
      description = '您没有执行此操作的权限'
    } else if (error?.status === 404) {
      title = '内容不存在'
      description = '请求的内容可能已被删除或不存在'
    } else if (error?.status === 429) {
      title = '操作过于频繁'
      description = '请稍后再试'
    } else if (error?.status >= 500) {
      title = '服务器错误'
      description = '服务器暂时无法处理请求，请稍后重试'
    } else if (error?.message) {
      description = error.message
    }
    
    toast.add({
      title,
      description,
      color: 'error',
      timeout: 5000
    })
  }

  /**
   * 处理网络错误
   */
  const handleNetworkError = (error: any, context?: string) => {
    console.error(`[Network Error${context ? ` - ${context}` : ''}]:`, error)
    
    toast.add({
      title: '网络连接失败',
      description: '请检查网络连接后重试',
      color: 'error',
      timeout: 5000
    })
  }

  /**
   * 处理权限错误
   */
  const handlePermissionError = (type: 'location' | 'camera' | 'microphone' | 'notification', context?: string) => {
    const permissionNames = {
      location: '位置',
      camera: '摄像头',
      microphone: '麦克风',
      notification: '通知'
    }
    
    toast.add({
      title: `${permissionNames[type]}权限被拒绝`,
      description: `请在浏览器设置中允许${permissionNames[type]}访问`,
      color: 'warning',
      timeout: 8000
    })
  }

  /**
   * 处理文件上传错误
   */
  const handleFileError = (error: any, context?: string) => {
    console.error(`[File Error${context ? ` - ${context}` : ''}]:`, error)
    
    let description = '文件处理失败'
    
    if (error?.message?.includes('size')) {
      description = '文件大小超出限制'
    } else if (error?.message?.includes('type')) {
      description = '不支持的文件类型'
    } else if (error?.message?.includes('network')) {
      description = '上传失败，请检查网络连接'
    }
    
    toast.add({
      title: '文件错误',
      description,
      color: 'error',
      timeout: 5000
    })
  }

  /**
   * 处理表单验证错误
   */
  const handleValidationError = (errors: Record<string, string[]>) => {
    const firstError = Object.values(errors)[0]?.[0]
    if (firstError) {
      toast.add({
        title: '输入有误',
        description: firstError,
        color: 'warning',
        timeout: 4000
      })
    }
  }

  /**
   * 通用错误处理器
   */
  const handleError = (error: any, context?: string) => {
    if (!error) return

    // 网络错误
    if (error.name === 'NetworkError' || error.code === 'NETWORK_ERROR') {
      handleNetworkError(error, context)
      return
    }

    // API 错误
    if (error.status || error.response) {
      handleApiError(error, context)
      return
    }

    // 权限错误
    if (error.name === 'NotAllowedError' || error.message?.includes('permission')) {
      // 根据上下文推断权限类型
      if (context?.includes('location') || error.message?.includes('location')) {
        handlePermissionError('location', context)
        return
      }
    }

    // 文件错误
    if (context?.includes('file') || context?.includes('upload')) {
      handleFileError(error, context)
      return
    }

    // 默认错误处理
    console.error(`[Error${context ? ` - ${context}` : ''}]:`, error)
    toast.add({
      title: '操作失败',
      description: error?.message || '发生未知错误，请重试',
      color: 'error',
      timeout: 5000
    })
  }

  /**
   * 成功提示
   */
  const showSuccess = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'success',
      timeout: 3000
    })
  }

  /**
   * 警告提示
   */
  const showWarning = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'warning',
      timeout: 4000
    })
  }

  /**
   * 信息提示
   */
  const showInfo = (title: string, description?: string) => {
    toast.add({
      title,
      description,
      color: 'info',
      timeout: 3000
    })
  }

  return {
    handleError,
    handleApiError,
    handleNetworkError,
    handlePermissionError,
    handleFileError,
    handleValidationError,
    showSuccess,
    showWarning,
    showInfo
  }
}