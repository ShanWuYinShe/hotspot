import { ensureRefreshToken } from '~/utils/token'

/**
 * 认证中间件
 * 
 * 中间件在客户端运行时，localStorage 已经可用，Pinia persist 插件会同步恢复数据。
 * 不需要等待水合，水合问题只存在于 Vue 组件的 SSR 渲染中。
 */
export default defineNuxtRouteMiddleware(async (to) => {
  // 仅在需要认证的页面进行校验
  if (!to.meta?.requiresAuth) return

  // SSR 阶段跳过，交由客户端处理
  if (import.meta.server) return

  const authStore = useAuthStore()
  
  // 若已认证则放行（store 的 isAuthenticated 不依赖水合状态）
  if (authStore.isAuthenticated) return

  // 尝试静默刷新 token 一次，避免误跳转
  try {
    // web：refresh_token 在 HttpOnly cookie；app：refresh_token 在 store
    if (authStore.canAttemptRefresh) {
      const success = await ensureRefreshToken()
      if (success && authStore.isAuthenticated) return
    }
  } catch (e: unknown) {
    // 收紧处理：仅对明确未授权的场景进行登录引导，其他错误放行以避免误登出
    const errorObj = e as { status?: number; code?: number } | null
    const authErrorCodes = [1100, 1101, 1102]
    if (errorObj?.status !== 401 && !authErrorCodes.includes(errorObj?.code ?? 0)) {
      console.warn('Auth check failed but not unauthorized:', e)
      return
    }
  }

  // 最终检查：如果仍未认证，跳转到登录页
  if (!authStore.isAuthenticated) {
    const hadCreds = !!authStore.token || authStore.refreshToken != null || authStore.canAttemptRefresh
    const params = new URLSearchParams({
      redirect: to.fullPath || '/',
      reason: hadCreds ? 'expired' : 'login_required'
    }).toString()
    return navigateTo(`/auth?mode=login&${params}`)
  }
})
