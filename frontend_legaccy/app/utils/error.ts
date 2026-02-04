/**
 * 错误处理工具函数
 * 提供类型安全的错误消息提取
 */

import { extractApiErrorSnapshot, guessUserMessageFromSnapshot } from '~/utils/apiError'

/**
 * 从未知错误中提取错误消息
 * @param error 未知类型的错误
 * @param defaultMessage 默认错误消息
 * @returns 错误消息字符串
 */
export function getErrorMessage(error: unknown, defaultMessage = '操作失败'): string {
  // 优先用统一规则从错误中提取 code/status/message，再转成面向用户的提示
  const snapshot = extractApiErrorSnapshot(error)
  const guessed = guessUserMessageFromSnapshot(snapshot, defaultMessage)
  if (guessed !== defaultMessage) return guessed

  if (error instanceof Error) {
    const msg = error.message || ''
    if (msg.includes('ExpiredSignature')) return '登录已过期，请重新登录'
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



