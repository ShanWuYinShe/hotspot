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
  // ofetch / $fetch 的 FetchError（尽量给用户友好的信息，避免把 URL/方法等内部细节直接暴露到 UI）
  if (error && typeof error === 'object') {
    const anyErr = error as Record<string, unknown>

    // 优先使用后端 envelope 里的 message/msg（即使 fetch 层把它包了一层）
    const data = anyErr.data
    if (data && typeof data === 'object') {
      const anyData = data as Record<string, unknown>
      // JWT 过期/无效：避免把内部错误细节（ExpiredSignature）直接展示给用户
      if (
        (typeof anyData.code === 'number' && anyData.code === 1101) ||
        (typeof anyData.message === 'string' && anyData.message.includes('ExpiredSignature')) ||
        (typeof anyData.msg === 'string' && anyData.msg.includes('ExpiredSignature'))
      ) {
        return '登录已过期，请重新登录'
      }
      if (typeof anyData.code === 'number' && anyData.code === 1102) {
        return '请先登录'
      }
      if (typeof anyData.message === 'string' && anyData.message.trim()) return anyData.message
      if (typeof anyData.msg === 'string' && anyData.msg.trim()) return anyData.msg
    }

    // ofetch 常见字段：statusCode/statusMessage（用于网络/5xx 等非业务错误）
    const statusCode =
      (typeof anyErr.statusCode === 'number' ? anyErr.statusCode : undefined) ??
      (typeof anyErr.status === 'number' ? anyErr.status : undefined)
    if (statusCode === 401) {
      // 仅凭 401 无法区分“未登录(1102)”还是“过期(1101)”，此处给更中性的提示
      return '请先登录'
    }
    if (typeof statusCode === 'number' && statusCode >= 500) {
      return defaultMessage
    }
  }

  if (error instanceof Error) {
    const msg = error.message || ''
    if (msg.includes('ExpiredSignature')) {
      return '登录已过期，请重新登录'
    }
    // 兜底：避免把 `[POST] "/api/..."`、baseURL 等内部细节直出到 toast
    if (
      msg.includes('Internal Server Error') ||
      msg.includes('/api/') ||
      /^\[[A-Z]+\]\s+\".*\"\:\s+\d{3}\s+/.test(msg)
    ) {
      return defaultMessage
    }
    return msg
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



