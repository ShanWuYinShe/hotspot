/**
 * Token 刷新工具
 * 
 * 统一的 token 刷新逻辑，供以下场景使用：
 * - base.service.ts: API 请求失败时自动刷新
 * - auth.global.ts: 路由守卫中尝试刷新
 * - token-refresh.client.ts: 定时刷新
 */

import type { ApiResponse, AuthData } from '~/types/api'
import { AUTH } from '~/config/api'

// 刷新锁，防止并发刷新
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

/**
 * 刷新 access_token
 * @returns 是否刷新成功
 */
export async function refreshAccessToken(): Promise<boolean> {
  const authStore = useAuthStore()
  // 统一前置条件：没有会话/refresh 已过期 -> 禁止触发 refresh_token（避免“未登录仍刷新”）
  if (!authStore.canAttemptRefresh) return false

  try {
    // app：用 body refresh_token；web：不传（依赖 HttpOnly cookie）
    const refreshTokenValue = authStore.refreshToken

    const response = await $fetch<ApiResponse<AuthData>>(AUTH.REFRESH_TOKEN, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: refreshTokenValue != null
        ? { refresh_token: refreshTokenValue, client_type: 'app' }
        : { client_type: 'web' }
    })

    if (response.code === 0 && response.data) {
      authStore.updateCurrentTokens(
        response.data.access_token,
        response.data.refresh_token,
        new Date(response.data.access_token_expires_at).getTime(),
        new Date(response.data.refresh_token_expires_at).getTime()
      )
      return true
    }
    return false
  } catch (error: unknown) {
    // 关键：refresh_token 真正失效后，不能继续“每分钟刷一次”，必须立刻停止并清理本地会话
    // 否则会形成长期噪音与不必要的后端压力
    let code: number | undefined
    let message: string | undefined

    if (error && typeof error === 'object') {
      const err = error as Record<string, unknown>
      const data = err.data
      if (data && typeof data === 'object') {
        const d = data as Record<string, unknown>
        if (typeof d.code === 'number') code = d.code
        if (typeof d.message === 'string') message = d.message
        if (!message && typeof d.msg === 'string') message = d.msg
      }
      if (!message && typeof err.message === 'string') message = err.message
      if (!code && typeof err.businessCode === 'number') code = err.businessCode
    }

    // 后端提示“已被轮换，请重试”时，不要直接登出（并发刷新场景的软失败）
    if (code === 1101 && message && message.includes('已被轮换')) {
      if (import.meta.dev) console.warn('[Auth] refresh_token 并发轮换，稍后重试', error)
      return false
    }

    // 真正无效/过期 or 需要重新登录：清理本地会话，终止后续刷新循环
    if (code === 1106 || (code === 1101 && message && message.includes('无效或已过期'))) {
      authStore.logoutCurrent()
    }

    if (import.meta.dev) console.warn('[Auth] refresh_token failed', { code, message, error })
    return false
  }
}

/**
 * 确保刷新 token（防止并发）
 * 多个请求同时需要刷新时，只执行一次刷新操作
 * @returns 是否刷新成功
 */
export async function ensureRefreshToken(): Promise<boolean> {
  const authStore = useAuthStore()
  if (!authStore.canAttemptRefresh) return false

  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }

  isRefreshing = true
  refreshPromise = refreshAccessToken().finally(() => {
    isRefreshing = false
    refreshPromise = null
  })

  return refreshPromise
}

/**
 * 判断错误是否为 token 过期错误
 */
export function isTokenExpiredError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false

  const err = error as Record<string, unknown>

  // 1) 业务错误（BaseService 主动 throw 的 businessCode）
  const businessCode = typeof err.businessCode === 'number' ? err.businessCode : undefined
  if (businessCode === 1101) return true

  // 2) FetchError.data 里通常有后端 ApiResponse 的 code/message
  const data = err.data
  if (data && typeof data === 'object') {
    const d = data as Record<string, unknown>
    if (typeof d.code === 'number' && d.code === 1101) return true
    // 兜底：JWT 解析错误原样透出
    if (typeof d.message === 'string' && d.message.includes('ExpiredSignature')) return true
    if (typeof d.msg === 'string' && d.msg.includes('ExpiredSignature')) return true
  }

  // 3) Error.message 兜底
  if (typeof err.message === 'string' && err.message.includes('ExpiredSignature')) return true

  return false
}



