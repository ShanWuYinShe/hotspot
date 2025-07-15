/**
 * 错误处理工具函数
 * 
 * 简化版本，只保留项目中实际使用的功能
 */

/**
 * 安全地从错误对象中提取错误消息
 * @param error 错误对象，可能是任何类型
 * @param fallbackMessage 默认错误消息
 * @returns 错误消息字符串
 */
export function extractErrorMessage(error: any, fallbackMessage: string): string {
  if (!error) {
    return fallbackMessage
  }
  
  // 如果error是字符串，直接返回
  if (typeof error === 'string' && error.trim()) {
    return error
  }
  
  // 如果error是对象，尝试提取信息
  if (typeof error === 'object') {
    // 尝试从不同的字段提取错误消息
    if (typeof error.message === 'string' && error.message.trim()) {
      return error.message
    } else if (typeof error.msg === 'string' && error.msg.trim()) {
      return error.msg
    } else if (error.response?.data?.msg && typeof error.response.data.msg === 'string') {
      return error.response.data.msg
    } else if (error.response?.data?.message && typeof error.response.data.message === 'string') {
      return error.response.data.message
    }
    
    // 根据错误类型提供友好的错误消息
    if (error.name === 'NetworkError' || error.code === 'NETWORK_ERROR' || 
        error.message?.includes('Failed to fetch') || error.message?.includes('Network')) {
      return '网络连接失败，请检查网络设置'
    } else if (error.name === 'AbortError' || error.code === 'TIMEOUT') {
      return '请求超时，请稍后重试'
    } else if (error.code === 'UNAUTHORIZED' || error.response?.status === 401) {
      return '身份验证失败，请重新登录'
    } else if (error.code === 'FORBIDDEN' || error.response?.status === 403) {
      return '权限不足，无法执行此操作'
    } else if (error.response?.status >= 500) {
      return '服务器错误，请稍后重试'
    }
  }
  
  return fallbackMessage
}
