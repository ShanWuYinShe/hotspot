/**
 * 统一的API错误处理工具
 * 提供一致的错误创建、处理和用户提示功能
 */

import { logger } from './logger'

/**
 * API错误类型
 */
export interface ApiError extends Error {
  status?: number
  code?: string
  details?: any
  userFriendlyMessage?: string
  canRetry?: boolean
}

/**
 * 创建标准化的API错误
 */
export const createApiError = (params: {
  status?: number
  code?: string
  message: string
  userFriendlyMessage?: string
  details?: any
  canRetry?: boolean
}): ApiError => {
  const error = new Error(params.message) as ApiError
  error.status = params.status || 400
  error.code = params.code || 'API_ERROR'
  error.details = params.details
  error.userFriendlyMessage = params.userFriendlyMessage || params.message
  error.canRetry = params.canRetry ?? false
  
  return error
}

/**
 * 包装API调用，提供统一的错误处理
 */
export const withApiErrorHandling = async <T>(
  operation: () => Promise<T>,
  context: {
    operation: string
    service: string
    fallbackMessage?: string
  }
): Promise<T> => {
  try {
    return await operation()
  } catch (error: any) {
    // 记录错误日志
    logger.error(
      `API调用失败: ${context.operation}`,
      context.service,
      {
        error: error.message,
        status: error.status,
        code: error.code,
        details: error.details
      }
    )

    // 如果已经是ApiError，直接抛出
    if (error.status !== undefined) {
      throw error
    }

    // 包装为标准ApiError
    throw createApiError({
      status: error.status || 500,
      code: error.code || 'UNKNOWN_ERROR',
      message: error.message || context.fallbackMessage || '操作失败',
      userFriendlyMessage: getUserFriendlyMessage(error, context.operation),
      details: error.details,
      canRetry: isRetryableError(error)
    })
  }
}

/**
 * 获取用户友好的错误消息
 */
const getUserFriendlyMessage = (error: any, operation: string): string => {
  // 网络错误
  if (error.message?.includes('fetch') || error.message?.includes('network')) {
    return '网络连接失败，请检查网络后重试'
  }

  // 认证错误
  if (error.status === 401) {
    return '登录已过期，请重新登录'
  }

  // 权限错误
  if (error.status === 403) {
    return '没有权限执行此操作'
  }

  // 资源不存在
  if (error.status === 404) {
    return '请求的资源不存在'
  }

  // 服务器错误
  if (error.status >= 500) {
    return '服务器暂时不可用，请稍后重试'
  }

  // 根据操作类型提供具体消息
  const operationMessages: Record<string, string> = {
    login: '登录失败，请检查用户名和密码',
    register: '注册失败，请检查输入信息',
    create: '创建失败，请检查输入信息',
    update: '更新失败，请稍后重试',
    delete: '删除失败，请稍后重试',
    search: '搜索失败，请稍后重试',
    upload: '上传失败，请检查文件格式和大小'
  }

  return operationMessages[operation] || '操作失败，请稍后重试'
}

/**
 * 判断错误是否可重试
 */
const isRetryableError = (error: any): boolean => {
  // 网络错误可重试
  if (error.message?.includes('fetch') || error.message?.includes('network')) {
    return true
  }

  // 5xx服务器错误可重试
  if (error.status >= 500) {
    return true
  }

  // 超时错误可重试
  if (error.code === 'TIMEOUT' || error.message?.includes('timeout')) {
    return true
  }

  return false
}

/**
 * 处理API响应错误
 */
export const handleApiResponse = <T>(response: any): T => {
  // 检查响应格式
  if (!response || typeof response !== 'object') {
    throw createApiError({
      code: 'INVALID_RESPONSE',
      message: '服务器响应格式错误',
      userFriendlyMessage: '服务器返回了无效的数据格式'
    })
  }

  // 检查业务错误码
  if (response.code !== undefined && response.code !== 0) {
    throw createApiError({
      code: `BUSINESS_ERROR_${response.code}`,
      message: response.msg || response.message || '业务处理失败',
      userFriendlyMessage: response.msg || '操作失败，请稍后重试',
      details: response.data
    })
  }

  return response.data || response
}

/**
 * 重试机制包装器
 */
export const withRetry = async <T>(
  operation: () => Promise<T>,
  options: {
    maxRetries?: number
    delay?: number
    backoff?: boolean
  } = {}
): Promise<T> => {
  const { maxRetries = 3, delay = 1000, backoff = true } = options
  
  let lastError: any
  
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await operation()
    } catch (error: any) {
      lastError = error
      
      // 如果不可重试或已达到最大重试次数，直接抛出错误
      if (!isRetryableError(error) || attempt === maxRetries) {
        throw error
      }
      
      // 计算延迟时间
      const currentDelay = backoff ? delay * Math.pow(2, attempt) : delay
      
      logger.warn(
        `API调用失败，${currentDelay}ms后重试 (${attempt + 1}/${maxRetries})`,
        'ApiRetry',
        { error: error.message }
      )
      
      // 等待后重试
      await new Promise(resolve => setTimeout(resolve, currentDelay))
    }
  }
  
  throw lastError
}

