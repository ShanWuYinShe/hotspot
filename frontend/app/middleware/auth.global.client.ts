export default defineNuxtRouteMiddleware(async (to) => {
  // 仅在需要认证的页面进行校验
  if (!to.meta?.requiresAuth) return

  // 等待 Pinia 状态已恢复（persistedstate）
  // 等待若干轮，直到 auth 恢复或确认无持久化数据
  const wait = (ms: number) => new Promise((r) => setTimeout(r, ms))
  let attempts = 0
  const maxAttempts = 30 // 增加到600ms
  while (attempts < maxAttempts) {
    const authStore = useAuthStore()
    // 若已认证则放行
    if (authStore.isAuthenticated) return
    
    // 检查是否有持久化数据正在恢复
    try {
      const raw = localStorage.getItem('auth')
      if (raw) {
        const data = JSON.parse(raw)
        // 如果有token但store还没恢复，继续等待
        if ((data.token || data.currentUser) && !authStore.isAuthenticated) {
          attempts++
          await wait(20)
          continue
        }
      }
      // 没有持久化数据且尝试超过10次，跳出
      if (!raw && attempts >= 10) break
    } catch (_) {
      // 可能处于隐私模式，直接跳出
      break
    }
    attempts++
    await wait(20)
  }

  const authStore = useAuthStore()

  // 已登录则放行
  if (authStore.isAuthenticated) return

  // 尝试静默校验一次，避免误跳转
  try {
    if (typeof authStore.checkAuth === 'function') {
      await authStore.checkAuth()
      if (authStore.isAuthenticated) return
    }
  } catch (e: any) {
    // 收紧处理：仅对明确未授权的场景进行登录引导，其他错误放行以避免误登出
    if (e?.status !== 401 && e?.code !== 1002) {
      console.warn('Auth check failed but not unauthorized:', e)
      return
    }
  }

  // 最终检查：如果仍未认证，跳转到登录页
  if (!authStore.isAuthenticated) {
    // 将目标地址作为 redirect 带入登录页，并标注原因
    const hadCreds = !!authStore.token || !!authStore.refreshToken
    const params = new URLSearchParams({
      redirect: to.fullPath || '/',
      reason: hadCreds ? 'expired' : 'login_required'
    }).toString()
    return navigateTo(`/auth?${params}`)
  }
})


