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
  const config = useRuntimeConfig()
  const refreshTokenValue = authStore.refreshToken
  
  if (!refreshTokenValue) {
    return false
  }

  try {
    const response = await $fetch<ApiResponse<AuthData>>(AUTH.REFRESH_TOKEN, {
      method: 'POST',
      baseURL: config.public.apiBaseUrl as string,
      headers: { 'Content-Type': 'application/json' },
      body: { refresh_token: refreshTokenValue }
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
  } catch {
    return false
  }
}

/**
 * 确保刷新 token（防止并发）
 * 多个请求同时需要刷新时，只执行一次刷新操作
 * @returns 是否刷新成功
 */
export async function ensureRefreshToken(): Promise<boolean> {
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
  if (typeof error === 'object' && error !== null) {
    const err = error as { businessCode?: number; status?: number }
    // 1101: Token无效或过期
    return err.businessCode === 1101 || err.status === 401
  }
  return false
}



