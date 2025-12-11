/**
 * Token 定时刷新 Plugin
 * 
 * 功能：
 * - 在 access_token 过期前自动刷新
 * - 使用 refresh_token 获取新的 token 对
 * - 防止并发刷新
 * 
 * 刷新策略：
 * - 在 token 过期前 5 分钟刷新
 * - 每 60 秒检查一次是否需要刷新
 * - 只有在用户已认证的情况下才刷新
 */

import { refreshAccessToken } from '~/utils/token'

// 提前刷新时间（毫秒）- token 过期前 5 分钟刷新
const REFRESH_BEFORE_EXPIRY = 5 * 60 * 1000

// 检查间隔（毫秒）- 每 60 秒检查一次
const CHECK_INTERVAL = 60 * 1000

// 刷新锁
let isRefreshing = false

export default defineNuxtPlugin(() => {
  // 只在客户端运行
  if (typeof window === 'undefined') return

  const authStore = useAuthStore()
  
  let checkTimer: ReturnType<typeof setInterval> | null = null

  /**
   * 检查并刷新 token
   */
  const checkAndRefresh = async () => {
    // 未认证则跳过
    if (!authStore.isAuthenticated) return

    // 正在刷新则跳过
    if (isRefreshing) return

    const session = authStore.currentSession
    if (!session) return

    const now = Date.now()
    const expiresAt = session.accessTokenExpiresAt

    // 如果 token 即将过期（过期前 5 分钟），则刷新
    if (expiresAt - now < REFRESH_BEFORE_EXPIRY) {
      console.log('[TokenRefresh] Token 即将过期，开始刷新...')
      isRefreshing = true
      try {
        const success = await refreshAccessToken()
        if (success) {
          console.log('[TokenRefresh] Token 刷新成功')
        }
      } finally {
        isRefreshing = false
      }
    }
  }

  /**
   * 启动定时检查
   */
  const startTokenRefreshTimer = () => {
    if (checkTimer) return

    // 立即检查一次
    checkAndRefresh()

    // 定时检查
    checkTimer = setInterval(checkAndRefresh, CHECK_INTERVAL)
    console.log('[TokenRefresh] 定时刷新已启动')
  }

  /**
   * 停止定时检查
   */
  const stopTokenRefreshTimer = () => {
    if (checkTimer) {
      clearInterval(checkTimer)
      checkTimer = null
      console.log('[TokenRefresh] 定时刷新已停止')
    }
  }

  // 监听认证状态变化
  // store 数据由 store-hydration 插件统一恢复，此处直接监听
  watch(
    () => authStore.isAuthenticated,
    (isAuthenticated) => {
      if (isAuthenticated) {
        startTokenRefreshTimer()
      } else {
        stopTokenRefreshTimer()
      }
    },
    { immediate: true }
  )

  // 页面可见性变化时检查
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
        // 页面变为可见时立即检查
        checkAndRefresh()
      }
    })
  }
})



