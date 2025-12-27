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

import { ensureRefreshToken } from '~/utils/token'

// 提前刷新时间（毫秒）- token 过期前 5 分钟刷新
const REFRESH_BEFORE_EXPIRY = 5 * 60 * 1000

// 检查间隔（毫秒）- 每 60 秒检查一次
const CHECK_INTERVAL = 60 * 1000

export default defineNuxtPlugin(() => {
  // 只在客户端运行
  if (typeof window === 'undefined') return

  const authStore = useAuthStore()
  const nuxtApp = useNuxtApp()
  
  const { pause, resume } = useIntervalFn(checkAndRefresh, CHECK_INTERVAL, { immediate: false })
  let didOneShotRestore = false
  let appMounted = false

  /**
   * 检查并刷新 Token
   */
  async function checkAndRefresh() {
    // 定时刷新只服务于“已登录且 access_token 仍有效”的场景（保持 SSR/CSR 行为简单可预测）
    if (!authStore.isAuthenticated) return

    const session = authStore.currentSession
    if (!session) return

    const now = Date.now()
    const expiresAt = session.accessTokenExpiresAt
    const timeUntilExpiry = expiresAt - now

    // access_token 已过期：交给 BaseService 的 on-demand refresh（避免后台定时器在失效状态下刷接口）
    if (isNaN(expiresAt) || timeUntilExpiry <= 0) return

    // 如果即将过期，执行刷新
    if (timeUntilExpiry < REFRESH_BEFORE_EXPIRY) {
      if (import.meta.dev) {
        console.log(`[TokenRefresh] Token expires in ${Math.round(timeUntilExpiry / 1000)}s, refreshing...`)
      }
      
      await ensureRefreshToken()
    }
  }

  /**
   * 尝试用 refresh token（web: HttpOnly cookie / app: body）恢复会话：只尝试一次，失败不重试
   * 目的：避免“未登录状态定时刷 refresh_token”导致的噪音与 403 spam。
   */
  const oneShotRestore = async () => {
    if (didOneShotRestore) return
    didOneShotRestore = true

    if (authStore.isAuthenticated) return
    if (!authStore.canAttemptRefresh) return

    if (import.meta.dev) {
      console.log('[TokenRefresh] One-shot restore attempt...')
    }
    await ensureRefreshToken()
  }

  /**
   * 启动定时检查
   */
  const startTokenRefreshTimer = () => {
    if (!appMounted) return
    // 立即检查一次
    void checkAndRefresh()

    // 定时检查
    resume()
    if (import.meta.dev) {
      console.log('[TokenRefresh] 定时刷新已启动')
    }
  }

  /**
   * 停止定时检查
   */
  const stopTokenRefreshTimer = () => {
    pause()
    if (import.meta.dev) {
      console.log('[TokenRefresh] 定时刷新已停止')
    }
  }

  // 监听认证状态变化
  // store 数据由 store-hydration 插件统一恢复，此处直接监听
  watch(
    () => authStore.isAuthenticated,
    (isAuthenticated) => {
      // 避免 hydration 期间启动定时器/触发网络请求
      if (!appMounted) return
      if (isAuthenticated) startTokenRefreshTimer()
      else stopTokenRefreshTimer()
    },
    { immediate: true }
  )

  // 页面可见性变化时检查
  const visibility = useDocumentVisibility()
  watch(visibility, (current, previous) => {
    if (current === 'visible' && previous === 'hidden') {
      // 页面变为可见：若已登录则检查续期；未登录则最多做一次“恢复会话尝试”
      if (!appMounted) return
      if (authStore.isAuthenticated) void checkAndRefresh()
      else void oneShotRestore()
    }
  })

  // 统一延后到 app mounted：避免在 hydration 之前改变认证状态/触发请求
  nuxtApp.hook('app:mounted', () => {
    appMounted = true
    void oneShotRestore()
    if (authStore.isAuthenticated) startTokenRefreshTimer()
  })
})



