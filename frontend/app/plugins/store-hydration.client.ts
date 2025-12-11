/**
 * Store 水合插件（客户端专用）
 * 
 * 执行时机：在 Vue 组件 onMounted 之前
 * 
 * 职责：从 localStorage 恢复所有 store 数据（同步操作）
 * 
 * 使用约定：
 * - 非 Vue 代码（中间件、服务等）：直接使用 store getter
 * - Vue 组件：使用 useHydratedValue() composable 获取值，避免 SSR mismatch
 */

export default defineNuxtPlugin(() => {
  const authStore = useAuthStore()
  const momentsStore = useMomentsStore()
  const groupsStore = useGroupsStore()
  const settingsStore = useSettingsStore()

  const stores = [authStore, momentsStore, groupsStore, settingsStore]
  
  // 从 localStorage 恢复数据
  for (const store of stores) {
    if (typeof store.$hydrate === 'function') {
      store.$hydrate({ runHooks: false })
    }
  }
})
