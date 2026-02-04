/**
 * Store 水合安全 Composable
 * 
 * 用途：在 Vue 组件中安全地使用持久化 store 的值，避免 SSR hydration mismatch
 * 
 * 工作原理：
 * 1. 在 onMounted 之前（SSR/客户端首次渲染）：返回默认值
 * 2. 在 onMounted 之后（客户端水合完成）：返回 store 的实际值
 * 
 * 使用场景：
 * - ✅ Vue 组件模板中使用持久化 store 的值
 * - ✅ computed 中使用持久化 store 的值
 * - ❌ 非 Vue 代码（中间件、服务等）：直接使用 store，无需此 composable
 * - ❌ 非持久化 store（如 messages、location）：直接使用，无需此 composable
 * 
 * @example
 * ```vue
 * <script setup>
 * const authStore = useAuthStore()
 * const settingsStore = useSettingsStore()
 * 
 * // 使用水合安全的值
 * const isAuthenticated = useHydratedStore(authStore, 'isAuthenticated', false)
 * const sidebarCollapsed = useHydratedStore(settingsStore, 'sidebarCollapsed', false)
 * </script>
 * 
 * <template>
 *   <div v-if="isAuthenticated">Welcome</div>
 *   <div :class="{ collapsed: sidebarCollapsed }">Sidebar</div>
 * </template>
 * ```
 */

/**
 * 使用水合安全的 store 值
 * 
 * @param store Pinia store 实例
 * @param key store 中的属性名
 * @param defaultValue 默认值（onMounted 前使用）
 * @returns 响应式的值（onMounted 前返回默认值，onMounted 后返回 store 值）
 */
export function useHydratedStore<
  TStore extends object,
  TKey extends keyof TStore
>(
  store: TStore,
  key: TKey,
  defaultValue: (TStore[TKey] extends { value: infer V } ? V : TStore[TKey])
): ComputedRef<(TStore[TKey] extends { value: infer V } ? V : TStore[TKey])> {
  const mounted = ref(false)

  onMounted(() => {
    mounted.value = true
  })

  return computed(() => {
    // onMounted 之前：返回默认值（避免 hydration mismatch）
    if (!mounted.value) {
      return defaultValue
    }

    // onMounted 之后：返回 store 的实际值
    const storeValue = (store as Record<PropertyKey, unknown>)[key as PropertyKey] as TStore[TKey]
    return isRef(storeValue)
      ? (storeValue.value as (TStore[TKey] extends { value: infer V } ? V : TStore[TKey]))
      : (storeValue as (TStore[TKey] extends { value: infer V } ? V : TStore[TKey]))
  })
}

/**
 * 获取多个水合安全的 store 值
 * 
 * @param store Pinia store 实例
 * @param defaults 默认值对象（key 为 store 属性名，value 为默认值）
 * @returns 响应式的值对象（每个 key 对应一个 computed ref）
 * 
 * @example
 * ```vue
 * <script setup>
 * const settingsStore = useSettingsStore()
 * 
 * const { theme, language, sidebarCollapsed } = useHydratedStoreMultiple(
 *   settingsStore,
 *   {
 *     theme: 'system',
 *     language: 'zh-CN',
 *     sidebarCollapsed: false
 *   }
 * )
 * </script>
 * ```
 */
export function useHydratedStoreMultiple<
  TStore extends object,
  TDefaults extends Partial<Record<keyof TStore, unknown>>
>(
  store: TStore,
  defaults: TDefaults
): { [K in keyof TDefaults]: ComputedRef<TDefaults[K]> } {
  const mounted = ref(false)

  onMounted(() => {
    mounted.value = true
  })

  const result = {} as { [K in keyof TDefaults]: ComputedRef<TDefaults[K]> }

  for (const key in defaults) {
    const k = key as keyof TDefaults
    result[k] = computed(() => {
      if (!mounted.value) {
        return defaults[k]
      }
      const storeValue = (store as Record<PropertyKey, unknown>)[k as PropertyKey] as TStore[keyof TStore]
      return isRef(storeValue)
        ? (storeValue.value as unknown as TDefaults[typeof k])
        : (storeValue as unknown as TDefaults[typeof k])
    })
  }

  return result
}

