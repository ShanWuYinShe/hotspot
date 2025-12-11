/**
 * 简洁的语言切换 Composable
 * 通过 settings store 统一管理语言偏好
 */
import type { LanguageCode } from '~/stores/settings.store'

export const useLocaleSwitcher = () => {
  const { locale, availableLocales } = useI18n()
  const nuxtApp = useNuxtApp()
  const settingsStore = useSettingsStore()
  
  // 语言名称映射
  const localeNames: Record<LanguageCode, string> = {
    'zh-CN': '中文',
    'en-US': 'English'
  }

  /**
   * 切换语言
   */
  const switchLocale = async (code: LanguageCode): Promise<boolean> => {
    try {
      // 切换 i18n 语言
      await (nuxtApp.$i18n as any).setLocale(code)
      
      // 保存到 settings store（本地持久化）
      settingsStore.setLanguage(code)
      
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
      code: code as LanguageCode,
      name: localeNames[code as LanguageCode] || code
    }))
  })

  return {
    currentLocale: computed(() => locale.value as LanguageCode),
    locales,
    switchLocale
  }
}

