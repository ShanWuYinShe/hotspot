// 认证中间件
import { useAuthStore } from '~/stores/auth'
import { storeToRefs } from 'pinia'

export default defineNuxtRouteMiddleware((to) => {
  // 在服务器端，我们可能没有访问 localStorage
  if (typeof window === 'undefined') return
  
  const authStore = useAuthStore()
  const { isAuthenticated } = storeToRefs(authStore)
  
  // 公开路由（不需要认证）
  const publicRoutes = ['/auth', '/']
  
  // 如果没有认证并且不是公开的路由，重定向到登录页面
  if (!isAuthenticated.value && !publicRoutes.includes(to.path)) {
    return navigateTo('/auth')
  }
  
  // 如果已认证并正在尝试访问登录页面，重定向到首页
  if (isAuthenticated.value && publicRoutes.includes(to.path) && to.path === '/auth') {
    return navigateTo('/')
  }
})
