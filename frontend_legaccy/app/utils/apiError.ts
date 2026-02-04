export type ApiErrorSnapshot = {
  code?: number
  status?: number
  message?: string
}

const isRecord = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null

export function extractApiErrorSnapshot(error: unknown): ApiErrorSnapshot {
  if (!isRecord(error)) return {}

  let status: number | undefined
  let code: number | undefined
  let message: string | undefined

  const rawStatus = error.statusCode ?? error.status
  if (typeof rawStatus === 'number') status = rawStatus

  const data = error.data
  if (isRecord(data)) {
    if (typeof data.code === 'number') code = data.code
    if (typeof data.message === 'string') message = data.message
    if (!message && typeof data.msg === 'string') message = data.msg
  }

  if (!message && typeof error.message === 'string') message = error.message
  if (!code && typeof (error as { businessCode?: unknown }).businessCode === 'number') {
    code = (error as { businessCode: number }).businessCode
  }

  return { code, status, message }
}

export function isAuthErrorCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10100 && code <= 10199
}

export function isTokenInvalidOrExpiredCode(code: number | undefined): boolean {
  return code === 10101
}

export function isTokenMissingCode(code: number | undefined): boolean {
  return code === 10102
}

export function isReauthRequiredCode(code: number | undefined): boolean {
  return code === 10106
}

export function isSoftTokenRotation(message: unknown): boolean {
  return typeof message === 'string' && message.includes('已被轮换')
}

export function shouldForceLogoutOnAuthError(code: number | undefined, message?: unknown): boolean {
  if (!isAuthErrorCode(code)) return false
  if (isReauthRequiredCode(code)) return true
  if (isTokenMissingCode(code)) return true
  if (isTokenInvalidOrExpiredCode(code)) {
    return !isSoftTokenRotation(message)
  }
  // 兜底：未来新增 401xx 子码时，默认要求重新登录
  return true
}

export function isBadRequestCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10000 && code <= 10099
}

export function isForbiddenCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10200 && code <= 10299
}

// 管理员相关错误码常量
export const NOT_ADMIN_CODE = 10204
export const ADMIN_PERMISSION_DENIED_CODE = 10205

export function isNotAdminCode(code: number | undefined): boolean {
  return code === NOT_ADMIN_CODE
}

export function isAdminPermissionDeniedCode(code: number | undefined): boolean {
  return code === ADMIN_PERMISSION_DENIED_CODE
}

export function isNotFoundCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10300 && code <= 10399
}

// 用户不存在错误码（10310）- 复用 isNotFoundCode 或单独使用
export const USER_NOT_FOUND_CODE = 10310
export function isUserNotFoundCode(code: number | undefined): boolean {
  return code === USER_NOT_FOUND_CODE
}

// 私聊不存在错误码（10350）
export const CHAT_NOT_FOUND_CODE = 10350
export function isChatNotFoundCode(code: number | undefined): boolean {
  return code === CHAT_NOT_FOUND_CODE
}

export function isConflictCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10400 && code <= 10499
}

export function isCursorInvalidOrExpiredCode(code: number | undefined): boolean {
  return code === 10006 || code === 10007
}

export function isRateLimitCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10500 && code <= 10599
}

export function isServerErrorCode(code: number | undefined): boolean {
  return typeof code === 'number' && code >= 10600 && code <= 10699
}

function isSafeUserMessage(message: unknown): message is string {
  if (typeof message !== 'string') return false
  const msg = message.trim()
  if (!msg) return false
  // 避免把内部细节（URL/方法/堆栈等）直接展示到 UI
  if (msg.includes('/api/') || msg.includes('baseURL')) return false
  if (/^\[[A-Z]+\]\s+\".*\"\:\s+\d{3}\s+/.test(msg)) return false
  if (msg.includes('Internal Server Error')) return false
  if (msg.includes('Bad Gateway')) return false
  return true
}

export function guessUserMessageFromSnapshot(
  snapshot: ApiErrorSnapshot,
  defaultMessage = '操作失败'
): string {
  const { code, status, message } = snapshot

  if (isTokenInvalidOrExpiredCode(code) || (typeof message === 'string' && message.includes('ExpiredSignature'))) {
    return '登录已过期，请重新登录'
  }

  if (isTokenMissingCode(code) || isAuthErrorCode(code) || status === 401) {
    return '请先登录'
  }

  if (status === 502) return '网关错误，服务暂时不可用'
  if (status === 503) return '服务维护中，请稍后重试'
  if (status === 504) return '请求超时，请检查网络连接'

  if (isBadRequestCode(code)) return isSafeUserMessage(message) ? message : '请求参数错误'

  // 区分管理员相关错误
  if (isNotAdminCode(code)) return '您不是管理员，无法访问此功能'
  if (isAdminPermissionDeniedCode(code)) return isSafeUserMessage(message) ? message : '管理员权限不足，请联系超级管理员'
  if (isForbiddenCode(code)) return '无权限执行此操作'
  if (isUserNotFoundCode(code)) return '该用户不存在或已被注销'
  if (isNotFoundCode(code)) return '资源不存在或已被删除'
  if (isConflictCode(code)) return isSafeUserMessage(message) ? message : '操作冲突，请稍后重试'
  if (isCursorInvalidOrExpiredCode(code)) return '列表已过期，请刷新后重试'
  if (isRateLimitCode(code) || status === 429) return '请求过于频繁，请稍后再试'

  if (isSafeUserMessage(message)) return message

  if (isServerErrorCode(code) || (typeof status === 'number' && status >= 500)) return '服务器内部错误，请稍后重试'

  return defaultMessage
}
