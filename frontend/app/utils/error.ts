/**
 * 错误处理工具函数
 * 提供类型安全的错误消息提取
 */

/**
 * 从未知错误中提取错误消息
 * @param error 未知类型的错误
 * @param defaultMessage 默认错误消息
 * @returns 错误消息字符串
 */
export function getErrorMessage(error: unknown, defaultMessage = '操作失败'): string {
  if (error instanceof Error) {
    return error.message
  }
  
  if (typeof error === 'string') {
    return error
  }
  
  if (error && typeof error === 'object') {
    // 处理包含 message 属性的对象
    if ('message' in error && typeof (error as { message: unknown }).message === 'string') {
      return (error as { message: string }).message
    }
    // 处理包含 msg 属性的对象（某些 API 错误格式）
    if ('msg' in error && typeof (error as { msg: unknown }).msg === 'string') {
      return (error as { msg: string }).msg
    }
  }
  
  return defaultMessage
}

/**
 * 检查错误是否为业务错误
 */
export function isBusinessError(error: unknown): error is Error & { isBusinessError: true; businessCode: number } {
  return error instanceof Error && 
    'isBusinessError' in error && 
    (error as { isBusinessError: unknown }).isBusinessError === true
}

/**
 * 获取业务错误码
 */
export function getBusinessErrorCode(error: unknown): number | undefined {
  if (isBusinessError(error)) {
    return error.businessCode
  }
  return undefined
}



