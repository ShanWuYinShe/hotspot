// Token刷新中间件
import { useAuthStore } from '~/stores/auth'

export default defineNuxtRouteMiddleware(async () => {
  // 在服务器端，我们不执行token刷新
  if (typeof window === 'undefined') return
  
  const authStore = useAuthStore()
  
  // 如果用户已登录且token即将过期，尝试刷新
  if (authStore.token && authStore.isTokenExpired()) {
    try {
      await authStore.refreshTokenAction()
    } catch (error) {
      console.error('无法刷新令牌:', error)
      // 令牌刷新失败，清除认证状态
      authStore.logout()
      return navigateTo('/auth')
    }
  }
})
