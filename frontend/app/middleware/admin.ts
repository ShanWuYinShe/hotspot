/**
 * Admin 路由守卫
 * - 不依赖 user_type（因为登录返回的 AuthData 不保证包含 user_type）
 * - 通过一次轻量的 admin API 探测权限并缓存（失败则拦截）
 */
export default defineNuxtRouteMiddleware(async () => {
  // SSR 阶段跳过，交由客户端处理（与 auth.global.ts 的策略一致）
  if (import.meta.server) return

  const authStore = useAuthStore()
  const adminStore = useAdminStore()

  // 必须已登录（requiresAuth 已在 auth.global.ts 处理，这里再兜底一次）
  if (!authStore.isAuthenticated) {
    return navigateTo('/auth?mode=login&reason=login_required')
  }

  // 已缓存 admin 权限
  if (adminStore.hasAccess) return

  // 通过 admin/rbac/me 探测权限（只要求“是管理员”，并返回权限列表）
  try {
    const { useAdminRbacService } = await import('~/services/admin_rbac.service')
    const rbac = useAdminRbacService()
    const me = await rbac.me()
    adminStore.setAccess(true, { isSuper: me.is_super, permissions: me.permissions })
    return
  } catch {
    adminStore.setAccess(false, { isSuper: false, permissions: [] })
    return navigateTo('/', { replace: true })
  }
})

