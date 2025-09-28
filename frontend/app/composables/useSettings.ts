/**
 * 设置业务逻辑 Composable
 * 职责：处理设置相关的业务逻辑和 API 调用，操作 Settings Store
 */

import type { AppSettings } from '~/stores/settings.store'

export function useSettings() {
  const settingsStore = useSettingsStore()
  const authStore = useAuthStore()
  const { execute } = useAsyncOperation()
  
  /**
   * 从服务器同步设置
   */
  const syncFromServer = () => {
    if (!authStore.isAuthenticated) return Promise.resolve()
    
    return execute(async () => {
      const http = useHttp()
      const response = await http.get('/protected/user/settings', { auth: true })
      
      // 合并服务器设置和本地设置
      const serverSettings = response.data
      settingsStore.updateSettings({
        notifications: serverSettings.notifications || settingsStore.settings.notifications,
        privacy: serverSettings.privacy || settingsStore.settings.privacy
      })
      
      return serverSettings
    })
  }
  
  /**
   * 保存设置到服务器
   */
  const saveToServer = () => {
    if (!authStore.isAuthenticated) return Promise.resolve()
    
    return execute(async () => {
      const http = useHttp()
      await http.put('/protected/user/settings', {
        notifications: settingsStore.settings.notifications,
        privacy: settingsStore.settings.privacy
      }, { auth: true })
    }, {
      successMessage: '设置保存成功'
    })
  }
  
  /**
   * 更新设置（本地 + 服务器）
   */
  const updateSettings = async (updates: Partial<AppSettings>) => {
    // 立即更新本地设置
    settingsStore.updateSettings(updates)
    
    // 如果是需要同步到服务器的设置，保存到服务器
    const needsSync = updates.notifications || updates.privacy
    if (needsSync && authStore.isAuthenticated) {
      await saveToServer()
    }
  }
  
  
  /**
   * 更新语言设置
   */
  const updateLanguage = (language: string) => {
    settingsStore.setLanguage(language)
  }
  
  /**
   * 更新通知设置
   */
  const updateNotifications = async (updates: Partial<AppSettings['notifications']>) => {
    await updateSettings({ notifications: { ...settingsStore.settings.notifications, ...updates } })
  }
  
  /**
   * 切换通知设置
   */
  const toggleNotification = async (type: keyof AppSettings['notifications']) => {
    await updateNotifications({
      [type]: !settingsStore.settings.notifications[type]
    })
  }
  
  /**
   * 更新隐私设置
   */
  const updatePrivacy = async (updates: Partial<AppSettings['privacy']>) => {
    await updateSettings({ privacy: { ...settingsStore.settings.privacy, ...updates } })
  }
  
  /**
   * 重置设置
   */
  const resetSettings = () => {
    return execute(async () => {
      settingsStore.resetSettings()
      
      // 如果已登录，也重置服务器设置
      if (authStore.isAuthenticated) {
        const http = useHttp()
        await http.delete('/protected/user/settings', { auth: true })
      }
    }, {
      successMessage: '设置已重置'
    })
  }
  
  /**
   * 初始化设置
   */
  const initialize = async () => {
    // 初始化本地设置
    settingsStore.initialize()
    
    // 如果已登录，同步服务器设置
    if (authStore.isAuthenticated) {
      await syncFromServer()
    }
  }
  
  return {
    // 状态（来自 store）
    settings: settingsStore.settings,
    displaySettings: settingsStore.displaySettings,
    notificationSettings: settingsStore.notificationSettings,
    privacySettings: settingsStore.privacySettings,
    
    // 业务方法
    syncFromServer,
    saveToServer,
    updateSettings,
    updateLanguage,
    updateNotifications,
    toggleNotification,
    updatePrivacy,
    resetSettings,
    initialize
  }
}
