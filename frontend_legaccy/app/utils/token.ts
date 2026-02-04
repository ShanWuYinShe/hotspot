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
import {
  extractApiErrorSnapshot,
  isSoftTokenRotation,
  isTokenInvalidOrExpiredCode,
  shouldForceLogoutOnAuthError
} from '~/utils/apiError'

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
      const data = response.data
      const refreshToken = 'refresh_token' in data ? data.refresh_token : null

      authStore.updateCurrentTokens(
        data.access_token,
        refreshToken,
        new Date(data.access_token_expires_at).getTime(),
        new Date(data.refresh_token_expires_at).getTime()
      )
      return true
    }
    return false
  } catch (error: unknown) {
    // 关键：refresh_token 真正失效后，不能继续“每分钟刷一次”，必须立刻停止并清理本地会话
    // 否则会形成长期噪音与不必要的后端压力
    const snapshot = extractApiErrorSnapshot(error)
    const { code, message } = snapshot

    // 后端提示“已被轮换，请重试”时，不要直接登出（并发刷新场景的软失败）
    if (isTokenInvalidOrExpiredCode(code) && isSoftTokenRotation(message)) {
      if (import.meta.dev) console.warn('[Auth] refresh_token 并发轮换，稍后重试', error)
      return false
    }

    // 真正无效/过期 or 缺失 or 需要重新登录：清理本地会话，终止后续刷新循环
    if (shouldForceLogoutOnAuthError(code, message)) {
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



