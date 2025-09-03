/**
 * 客户端认证状态恢复插件
 * 在页面刷新时自动检查和恢复登录状态
 */
export default defineNuxtPlugin(async (nuxtApp) => {
  // 客户端插件，无需检查运行环境

  const authStore = useAuthStore()

  // 如果已经有token，尝试检查认证状态
  if (authStore.token && authStore.refreshToken) {
    try {
      // 检查token是否仍然有效
      await authStore.checkAuth()
      
      console.log('[Auth Restore] 登录状态已恢复:', authStore.user?.nickname)
    } catch (error) {
      console.warn('[Auth Restore] 登录状态检查失败，尝试刷新token')
      
      // 如果检查失败，尝试刷新token
      try {
        await authStore.refreshAuth()
        console.log('[Auth Restore] Token刷新成功，登录状态已恢复')
      } catch (refreshError) {
        console.warn('[Auth Restore] Token刷新失败，用户需要重新登录')
        // 不自动清理，让用户手动重新登录
      }
    }
  } else if (authStore.accounts.length > 0 && !authStore.user) {
    // 如果有保存的账户但没有当前用户，提示用户可以切换账户
    console.log('[Auth Restore] 发现已保存的账户，用户可以通过用户菜单切换')
  }
})
