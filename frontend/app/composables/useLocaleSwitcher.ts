/**
 * 简洁的语言切换 Composable
 * 通过 settings store 统一管理语言偏好
 */
export const useLocaleSwitcher = () => {
  const { locale, availableLocales } = useI18n()
  const nuxtApp = useNuxtApp()
  const settingsStore = useSettingsStore()
  
  // 语言名称映射
  const localeNames = {
    'zh-CN': '中文',
    'en-US': 'English'
  }

  /**
   * 切换语言
   */
  const switchLocale = async (code: string): Promise<boolean> => {
    try {
      // 切换 i18n 语言
      await (nuxtApp.$i18n as any).setLocale(code)
      
      // 保存到 settings store（本地持久化）
      settingsStore.setPreferredLocale(code)
      
      // 如果用户已登录，同步到服务器
      if (settingsStore.settings) {
        try {
          // 注意：当前 UserSettings 接口不包含 language 字段
          // 这里暂时跳过服务器同步，只在本地存储
          console.warn('语言设置暂时只在本地存储，服务器同步功能待实现')
        } catch (error) {
          console.warn('同步语言设置到服务器失败:', error)
          // 不影响本地切换
        }
      }
      
      return true
    } catch (error) {
      console.error('语言切换失败:', error)
      return false
    }
  }

  /**
   * 获取格式化的语言列表
   */
  const locales = computed(() => {
    return availableLocales.map((code: string) => ({
      code,
      name: localeNames[code as keyof typeof localeNames] || code
    }))
  })

  return {
    currentLocale: computed(() => locale.value),
    locales,
    switchLocale
  }
}
