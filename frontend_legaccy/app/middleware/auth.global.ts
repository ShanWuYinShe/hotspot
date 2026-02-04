import { ensureRefreshToken } from '~/utils/token'
import { extractApiErrorSnapshot, isAuthErrorCode } from '~/utils/apiError'

/**
 * 认证中间件
 * 
 * 中间件在客户端运行时，localStorage 已经可用，Pinia persist 插件会同步恢复数据。
 * 不需要等待水合，水合问题只存在于 Vue 组件的 SSR 渲染中。
 */
export default defineNuxtRouteMiddleware(async (to) => {
  // SSR 阶段跳过，交由客户端处理
  if (import.meta.server) return

  // 仅在需要认证的页面进行校验
  // 明确检查：requiresAuth 必须显式为 true 才需要认证
  // false、undefined、null 都视为不需要认证
  if (to.meta?.requiresAuth !== true) return

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
    const snapshot = extractApiErrorSnapshot(e)
    if (snapshot.status !== 401 && !isAuthErrorCode(snapshot.code)) {
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
