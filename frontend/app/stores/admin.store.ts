import { defineStore } from 'pinia'

/**
 * Admin 权限缓存（前端侧）
 * - 不依赖 user_type（AuthData 不保证包含 user_type）
 * - 通过调用 admin API 探测权限后缓存
 */
export const useAdminStore = defineStore('admin', () => {
  const hasAccess = ref(false)
  const permissions = ref<string[]>([])
  const checkedAt = ref<number | null>(null)

  const setAccess = (value: boolean, opts?: { permissions?: string[] }) => {
    hasAccess.value = value
    permissions.value = opts?.permissions ?? []
    checkedAt.value = Date.now()
  }

  const hasPermission = (key: string) => {
    if (!hasAccess.value) return false
    if (permissions.value.includes('*')) return true
    return permissions.value.includes(key)
  }

  // Derived property for convenience, though internal logic uses hasPermission
  const isSuper = computed(() => permissions.value.includes('*'))

  return {
    hasAccess,
    isSuper,
    permissions,
    checkedAt,
    setAccess,
    hasPermission
  }
})
