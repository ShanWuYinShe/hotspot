/**
 * 客户端初始化插件
 * 处理客户端启动时的初始化逻辑
 */

import { useAuthStore } from '~/stores/auth'
import { useUserStore } from '~/stores/user'

export default defineNuxtPlugin(async (nuxtApp) => {
  console.log('Client plugin initialized')
  
  // 延迟执行，确保Pinia持久化数据已加载
  await new Promise(resolve => setTimeout(resolve, 100))
  
  // 获取存储的状态
  const authStore = useAuthStore()
  const userStore = useUserStore()
  
  console.log('Auth store状态:', {
    isAuthenticated: authStore.isAuthenticated,
    hasToken: !!authStore.getToken,
    hasUser: !!authStore.getUser
  })
  
  // 处理用户认证状态
  if (authStore.getToken && authStore.isTokenExpired()) {
    console.log('Token已过期，尝试刷新...')
    try {
      await authStore.refreshTokenAction()
      console.log('Token刷新成功')
    } catch (error) {
      console.error('Failed to refresh token:', error)
      // 令牌刷新失败，清除身份验证状态
      authStore.logout()
    }
  }
  
  // 如果用户没有登录，自动创建临时用户
  if (!authStore.isAuthenticated) {
    console.log('用户未登录，开始创建临时用户...')
    try {
      // 生成一个随机的临时用户昵称
      const tempNickname = `游客${Math.random().toString(36).substr(2, 6)}`
      await authStore.createTempUser(tempNickname)
      console.log('临时用户创建成功:', tempNickname)
    } catch (error) {
      console.error('创建临时用户失败:', error)
      // 即使临时用户创建失败，也不阻止应用程序运行
    }
  }
  
  return {
    provide: {
      // 提供给应用程序的客户端工具
      clientInitialized: true,
    }
  }
})
