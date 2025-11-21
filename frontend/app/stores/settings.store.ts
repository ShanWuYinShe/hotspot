/**
 * 设置状态存储
 * 职责：设置状态管理、持久化 + API 调用
 */

import type {
  UpdateProfileRequest,
  UpdateSettingsRequest
} from '~/services/users.service'

export interface AppSettings {
  // 外观设置
  theme: 'light' | 'dark' | 'system'
  language: string
  fontSize: 'small' | 'medium' | 'large'
  compactMode: boolean
  enableAnimations: boolean
  
  // 通知设置
  notifications: {
    push: boolean
    email: boolean
    sound: boolean
    vibration: boolean
  }
  
  // 隐私设置
  privacy: {
    profileVisibility: 'public' | 'friends' | 'private'
    locationSharing: boolean
    onlineStatus: boolean
    readReceipts: boolean
  }
}

const DEFAULT_SETTINGS: AppSettings = {
  theme: 'system',
  language: 'zh-CN',
  fontSize: 'medium',
  compactMode: false,
  enableAnimations: true,
  notifications: {
    push: true,
    email: true,
    sound: true,
    vibration: true
  },
  privacy: {
    profileVisibility: 'public',
    locationSharing: true,
    onlineStatus: true,
    readReceipts: true
  }
}

export const useSettingsStore = defineStore('settings', () => {
  // ============ 状态 ============
  const settings = ref<AppSettings>({ ...DEFAULT_SETTINGS })
  
  // ============ 计算属性 ============
  const isDarkMode = computed(() => {
    if (settings.value.theme === 'system') {
      if (typeof window !== 'undefined') {
        return window.matchMedia('(prefers-color-scheme: dark)').matches
      }
      return false
    }
    return settings.value.theme === 'dark'
  })
  
  const displaySettings = computed(() => ({
    theme: settings.value.theme,
    fontSize: settings.value.fontSize,
    compactMode: settings.value.compactMode,
    enableAnimations: settings.value.enableAnimations
  }))
  
  const notificationSettings = computed(() => settings.value.notifications)
  const privacySettings = computed(() => settings.value.privacy)
  
  // ============ 状态操作方法 ============
  
  /**
   * 更新设置
   */
  const updateSettings = (updates: Partial<AppSettings>) => {
    settings.value = { ...settings.value, ...updates }
    
    // 应用主题变化
    if (updates.theme !== undefined) {
      applyTheme()
    }
    
    // 应用语言变化
    if (updates.language) {
      applyLanguage(updates.language)
    }
  }
  
  /**
   * 设置完整设置对象
   */
  const setSettings = (newSettings: AppSettings) => {
    settings.value = { ...newSettings }
    applyTheme()
    applyLanguage(newSettings.language)
  }
  
  /**
   * 重置为默认设置
   */
  const resetSettings = () => {
    settings.value = { ...DEFAULT_SETTINGS }
    applyTheme()
    applyLanguage(DEFAULT_SETTINGS.language)
  }
  
  /**
   * 更新主题设置
   */
  const setTheme = (theme: AppSettings['theme']) => {
    updateSettings({ theme })
  }
  
  /**
   * 切换主题
   */
  const toggleTheme = () => {
    const themes: AppSettings['theme'][] = ['light', 'dark', 'system']
    const currentIndex = themes.indexOf(settings.value.theme)
    const nextTheme = themes[(currentIndex + 1) % themes.length]
    setTheme(nextTheme)
  }
  
  /**
   * 设置语言
   */
  const setLanguage = (language: string) => {
    updateSettings({ language })
  }
  
  /**
   * 更新通知设置
   */
  const updateNotifications = (updates: Partial<AppSettings['notifications']>) => {
    updateSettings({
      notifications: { ...settings.value.notifications, ...updates }
    })
  }
  
  /**
   * 切换通知设置
   */
  const toggleNotification = (type: keyof AppSettings['notifications']) => {
    updateNotifications({
      [type]: !settings.value.notifications[type]
    })
  }
  
  /**
   * 更新隐私设置
   */
  const updatePrivacy = (updates: Partial<AppSettings['privacy']>) => {
    updateSettings({
      privacy: { ...settings.value.privacy, ...updates }
    })
  }
  
  // ============ 内部方法 ============
  
  /**
   * 应用主题到 DOM
   */
  const applyTheme = () => {
    if (typeof window === 'undefined') return
    
    const html = document.documentElement
    const theme = isDarkMode.value ? 'dark' : 'light'
    
    html.classList.remove('light', 'dark')
    html.classList.add(theme)
    html.setAttribute('data-theme', theme)
  }
  
  /**
   * 应用语言设置
   */
  const applyLanguage = (language: string) => {
    if (typeof window === 'undefined') return
    
    document.documentElement.lang = language
    
    // 如果使用了 i18n，在这里切换语言
    // const { $i18n } = useNuxtApp()
    // if ($i18n) {
    //   $i18n.locale.value = language
    // }
  }
  
  /**
   * 初始化设置
   */
  const initialize = () => {
    // Pinia 插件会自动恢复持久化设置
    applyTheme()
    applyLanguage(settings.value.language)
    
    // 监听系统主题变化
    if (typeof window !== 'undefined' && settings.value.theme === 'system') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', applyTheme)
    }
  }

  // ============ API 调用方法 ============

  /**
   * 更新用户资料（API）
   */
  const updateProfile = async (data: UpdateProfileRequest) => {
    const userService = useUserService()
    const authStore = useAuthStore()
    const toast = useToast()
    
    try {
      await userService.updateProfile(data)
      
      // 更新 authStore 中的用户信息
      authStore.updateCurrentUser(data)
      
      toast.add({ color: 'success', title: '资料更新成功' })
    } catch (error: any) {
      toast.add({ color: 'error', title: '更新失败', description: error.message })
      throw error
    }
  }

  /**
   * 更新隐私设置（API）
   */
  const updatePrivacySettings = async (data: UpdateSettingsRequest) => {
    const userService = useUserService()
    
    try {
      await userService.updateSettings(data)
      
      // 更新本地设置
      updatePrivacy({
        profileVisibility: data.privacy_level as any || settings.value.privacy.profileVisibility,
        locationSharing: data.is_location_sharing ?? settings.value.privacy.locationSharing,
        onlineStatus: settings.value.privacy.onlineStatus,
        readReceipts: settings.value.privacy.readReceipts
      })
      
      toast.add({ color: 'success', title: '设置更新成功' })
    } catch (error: any) {
      toast.add({ color: 'error', title: '更新失败', description: error.message })
      throw error
    }
  }

  /**
   * 获取用户设置（API）
   */
  const fetchSettings = async () => {
    const userService = useUserService()
    
    try {
      const response = await userService.getSettings()
      
      // 更新本地设置
      updatePrivacy({
        profileVisibility: response.privacy_level as any,
        locationSharing: response.is_location_sharing,
        onlineStatus: settings.value.privacy.onlineStatus,
        readReceipts: settings.value.privacy.readReceipts
      })
    } catch (error: any) {
      console.error('获取设置失败:', error)
    }
  }
  
  // ============ 初始化 ============
  // 客户端自动初始化
  if (typeof window !== 'undefined') {
    initialize()
  }
  
  return {
    // 状态（直接返回，供 Pinia 和插件使用）
    settings,
    isDarkMode,
    displaySettings,
    notificationSettings,
    privacySettings,
    
    // 状态操作方法
    updateSettings,
    setSettings,
    resetSettings,
    setTheme,
    toggleTheme,
    setLanguage,
    updateNotifications,
    toggleNotification,
    updatePrivacy,
    initialize,

    // API 调用方法
    updateProfile,
    updatePrivacySettings,
    fetchSettings
  }
}, {
  persist: {
    key: 'app-settings',
    pick: ['settings']
  }
})
