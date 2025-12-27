/**
 * WebSocket 全局生命周期管理插件
 * 
 * 职责：
 * 1. 监听用户认证状态，自动连接/断开 WebSocket
 * 2. 应用挂载时初始化连接（如果已登录）
 * 3. 配置 WebSocket 的 Token 获取方式和错误处理
 */
import { webSocketService } from '~/services/websocket.service'
import { ensureRefreshToken } from '~/utils/token'

export default defineNuxtPlugin((nuxtApp) => {
  // 仅在客户端运行
  if (import.meta.server) return

  const authStore = useAuthStore()
  let appMounted = false

  const buildWsUrl = (): string => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const origin = `${proto}//${window.location.host}`
    return `${origin}/ws`
  }
  
  // 1. 配置 Token Provider
  // WebSocketClient 在每次重连尝试时都会调用此函数获取最新 Token
  webSocketService.setTokenProvider(async () => {
    // 优先使用当前 Store 中的 Token
    if (authStore.token && authStore.isAccessTokenValid) {
      return authStore.token
    }
    
    // 如果 Token 过期或不存在，尝试刷新（app: body refresh_token；web: HttpOnly cookie）
    if (authStore.canAttemptRefresh) {
      console.log('[WS Plugin] Token expired during connection, attempting refresh...')
      const success = await ensureRefreshToken()
      if (success && authStore.token) {
        return authStore.token
      }
    } else if (authStore.token) {
       // AccessToken 存在但过期，且没有有效的 RefreshToken (或者 RefreshToken 也过期了)
       // 这种情况通常意味着登录失效，应该触发登出。
       // 但为了安全，这里只返回 null，由 WebSocketClient 重试或由其他机制（如 Token 刷新插件）处理登出。
       console.warn('[WS Plugin] Token expired and no valid refresh token available')
    }
    
    // 如果无法获取有效 Token，返回 null。
    // WebSocketClient 会将此视为连接失败并尝试重连（直到达到最大重试次数）。
    // 这对于网络不稳定导致刷新失败的情况非常重要。
    return null
  })

  // 2. 配置全局错误处理
  webSocketService.setErrorHandler(async (error) => {
    if (import.meta.dev) {
      console.error('[WS Plugin] Global Error:', error)
    }
    
    // 处理特定错误码
    if (error.code === 1101) {
      // Token 无效或过期 (服务端返回的错误)
      console.warn('[WS Plugin] Received 1101 Token Invalid/Expired from server')
      
      // 尝试刷新 Token
      const success = authStore.canAttemptRefresh ? await ensureRefreshToken() : false
      if (success) {
        console.log('[WS Plugin] Token refreshed successfully, triggering reconnection')
        // 刷新成功后，WebSocketClient 会在下一次重连尝试时自动使用新 Token
        // 或者我们可以主动触发一次连接
        const url = buildWsUrl()
        webSocketService.connect(url)
      } else {
        console.error('[WS Plugin] Failed to refresh token, logging out')
        // 刷新失败，强制登出
        // authService.logout() // 或者直接操作 store
        authStore.logoutCurrent()
        webSocketService.disconnect()
        navigateTo('/auth?mode=login')
      }
    }
  })
  
  // 获取配置
  const url = buildWsUrl()

  // 3. 监听认证状态变化
  watch(
    () => authStore.isAuthenticated,
    (isAuthenticated) => {
      // 避免 hydration 期间连接/断开导致的状态抖动
      if (!appMounted) return
      if (import.meta.dev) {
        console.log('[WS Plugin] 认证状态变化:', isAuthenticated)
      }
      
      if (isAuthenticated && authStore.token) {
        // 确保在浏览器环境且页面已挂载（或即将挂载）时连接
        nextTick(() => {
          webSocketService.connect(url)
        })
      } else {
        webSocketService.disconnect()
      }
    },
    { immediate: true }
  )

  nuxtApp.hook('app:mounted', () => {
    appMounted = true
    // mounted 后补一次：保证已登录用户能连接
    if (authStore.isAuthenticated && authStore.token) {
      webSocketService.connect(url)
    }
  })
})
