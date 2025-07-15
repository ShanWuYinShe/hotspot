/**
 * 全局应用设置管理
 * 提供跨页面的设置同步功能
 */

import type { NotificationSettings } from '~/types/api/user'
import { useColorMode } from '@vueuse/core'

// 设置变更事件
const SETTINGS_CHANGE_EVENT = 'app-settings-changed'

export interface AppSettings {
  theme: 'auto' | 'light' | 'dark'
  language: 'zh-CN' | 'en-US'
  fontSize: 'small' | 'medium' | 'large'
  showTooltips: boolean
  navigation: {
    position: 'top' | 'bottom' // 导航栏位置
    alwaysVisible: boolean // 是否始终显示导航栏
  }
  notifications: NotificationSettings
  privacy: {
    location: boolean
    online: boolean
  }
  autoRefresh: {
    enabled: boolean
    interval: number // 刷新间隔（秒）
    locationCheck: boolean // 位置变化检测
    locationThreshold: number // 位置变化阈值（米）
    websocketEnabled: boolean // WebSocket 实时更新
  }
}

const defaultSettings: AppSettings = {
  theme: 'auto',
  language: 'zh-CN',
  fontSize: 'medium',
  showTooltips: true,
  navigation: {
    position: 'top', // 默认顶部导航
    alwaysVisible: true // 始终显示导航栏
  },
  notifications: {
    message_notifications: true,
    group_notifications: true,
    system_notifications: true,
    location_notifications: true,
    quiet_hours: {
      enabled: false,
      start_time: "22:00",
      end_time: "08:00"
    },
    notification_radius_km: 5.0,
    sound_enabled: true,
    vibration_enabled: false
  },
  privacy: {
    location: true,
    online: true
  },
  autoRefresh: {
    enabled: true,
    interval: 30, // 30秒
    locationCheck: true,
    locationThreshold: 100, // 100米
    websocketEnabled: true
  }
}

// 全局设置状态
const globalSettings = ref<AppSettings>({ ...defaultSettings })

// ==================== 通知相关计算属性和方法 ====================

// 检查是否在静音时间内
const isInQuietHours = computed((): boolean => {
  if (!globalSettings.value.notifications.quiet_hours.enabled) return false

  const now = new Date()
  const currentTime = now.getHours() * 60 + now.getMinutes()
  
  const [startHour, startMin] = globalSettings.value.notifications.quiet_hours.start_time.split(':').map(Number)
  const [endHour, endMin] = globalSettings.value.notifications.quiet_hours.end_time.split(':').map(Number)
  
  const startTime = startHour * 60 + startMin
  const endTime = endHour * 60 + endMin

  if (startTime <= endTime) {
    // 同一天内的时间范围
    return currentTime >= startTime && currentTime <= endTime
  } else {
    // 跨天的时间范围（如 22:00 到次日 08:00）
    return currentTime >= startTime || currentTime <= endTime
  }
})

// 是否应该播放声音
const shouldPlaySound = computed((): boolean => {
  return globalSettings.value.notifications.sound_enabled && !isInQuietHours.value
})

// 是否应该震动
const shouldVibrate = computed((): boolean => {
  return globalSettings.value.notifications.vibration_enabled && !isInQuietHours.value
})

// 检查是否应该显示通知（核心过滤逻辑）
const shouldShowNotification = (type: 'message' | 'group' | 'system' | 'location'): boolean => {
  // 静音时间内不显示任何通知
  if (isInQuietHours.value) return false

  // 根据通知类型检查用户设置
  switch (type) {
    case 'message':
      return globalSettings.value.notifications.message_notifications
    case 'group':
      return globalSettings.value.notifications.group_notifications
    case 'system':
      return globalSettings.value.notifications.system_notifications
    case 'location':
      return globalSettings.value.notifications.location_notifications
    default:
      return true
  }
}

// 过滤 WebSocket 通知数据的主要方法
const filterNotification = (notificationType: string, notificationData: any) => {
  const type = notificationType as 'message' | 'group' | 'system' | 'location'
  
  if (!shouldShowNotification(type)) {
    console.log(`通知已被过滤: ${type}`, notificationData)
    return null
  }

  return {
    shouldShow: true,
    shouldPlaySound: shouldPlaySound.value,
    shouldVibrate: shouldVibrate.value,
    data: notificationData
  }
}

export const useAppSettings = () => {
  // 加载状态
  const isLoaded = ref<boolean>(false)
  
  // 主题控制
  const colorMode = useColorMode({
    modes: {
      auto: 'auto',
      light: 'light',
      dark: 'dark'
    }
  })

  // ==================== 设置应用方法 ====================
  
  // 应用主题设置
  const applyThemeSettings = () => {
    colorMode.value = globalSettings.value.theme

    // 应用字体大小
    const fontSize = globalSettings.value.fontSize
    const root = document.documentElement

    switch (fontSize) {
      case 'small':
        root.style.fontSize = '14px'
        break
      case 'large':
        root.style.fontSize = '18px'
        break
      default:
        root.style.fontSize = '16px'
    }

    // 应用提示设置
    if (globalSettings.value.showTooltips) {
      root.classList.add('tooltips-enabled')
    } else {
      root.classList.remove('tooltips-enabled')
    }
  }

  // WebSocket设置控制
  const applyWebSocketSettings = () => {
    if (typeof window === 'undefined') return
    
    if (globalSettings.value.autoRefresh.websocketEnabled) {
      // 启用WebSocket - 通过事件通知
      window.dispatchEvent(new CustomEvent('websocket-enable'))
    } else {
      // 禁用WebSocket - 通过事件通知
      window.dispatchEvent(new CustomEvent('websocket-disable'))
    }
  }

  // 自动刷新间隔管理
  let refreshInterval: NodeJS.Timeout | null = null

  const setupAutoRefresh = () => {
    // 清除现有定时器
    if (refreshInterval) {
      clearInterval(refreshInterval)
      refreshInterval = null
    }

    // 如果启用自动刷新，设置新定时器
    if (globalSettings.value.autoRefresh.enabled) {
      const intervalMs = globalSettings.value.autoRefresh.interval * 1000

      refreshInterval = setInterval(() => {
        // 触发自定义事件，让各个页面响应
        window.dispatchEvent(new CustomEvent('app-auto-refresh', {
          detail: {
            source: 'settings',
            interval: globalSettings.value.autoRefresh.interval
          }
        }))
      }, intervalMs)

      console.log(`自动刷新已启用，间隔: ${globalSettings.value.autoRefresh.interval}秒`)
    }
  }

  // 导航栏位置控制
  const applyNavigationSettings = () => {
    if (typeof document === 'undefined') return
    
    const root = document.documentElement

    // 设置导航栏位置
    root.setAttribute('data-nav-position', globalSettings.value.navigation.position)

    // 设置导航栏可见性
    if (globalSettings.value.navigation.alwaysVisible) {
      root.classList.add('nav-always-visible')
    } else {
      root.classList.remove('nav-always-visible')
    }
  }

  // 声音和震动工具方法
  const playNotificationSound = () => {
    if (globalSettings.value.notifications.sound_enabled && !isInQuietHours.value) {
      // 播放通知声音
      const audio = new Audio('/sounds/notification.mp3')
      audio.volume = 0.5
      audio.play().catch(console.error)
    }
  }

  const triggerVibration = () => {
    if (globalSettings.value.notifications.vibration_enabled && !isInQuietHours.value && 'vibrate' in navigator) {
      navigator.vibrate([200, 100, 200])
    }
  }

  // 监听设置变更并应用
  const setupSettingsWatchers = () => {
    if (typeof window === 'undefined') return

    // 监听主题变更
    watch(() => globalSettings.value.theme, applyThemeSettings, { immediate: true })
    watch(() => globalSettings.value.fontSize, applyThemeSettings)
    watch(() => globalSettings.value.showTooltips, applyThemeSettings)

    // 监听WebSocket设置变更
    watch(() => globalSettings.value.autoRefresh.websocketEnabled, applyWebSocketSettings)

    // 监听自动刷新设置变更
    watch(() => globalSettings.value.autoRefresh.enabled, setupAutoRefresh)
    watch(() => globalSettings.value.autoRefresh.interval, setupAutoRefresh)

    // 监听导航设置变更
    watch(() => globalSettings.value.navigation.position, applyNavigationSettings)
    watch(() => globalSettings.value.navigation.alwaysVisible, applyNavigationSettings)
  }

  // 初始化应用所有设置
  const initializeSettings = () => {
    if (typeof window === 'undefined') return
    
    applyThemeSettings()
    applyNavigationSettings()
    setupAutoRefresh()
    applyWebSocketSettings()
  }

  // 加载设置
  const loadSettings = async () => {
    if (typeof window === 'undefined') return

    try {
      const savedSettings = localStorage.getItem('app-settings')
      if (savedSettings) {
        const parsed = JSON.parse(savedSettings)
        // 合并默认设置，确保没有缺失的字段
        Object.assign(globalSettings.value, defaultSettings, parsed)
      }
    } catch (error) {
      console.error('加载设置失败:', error)
      // 使用默认设置
      Object.assign(globalSettings.value, defaultSettings)
    } finally {
      isLoaded.value = true
    }
  }

  // 保存设置
  const saveSettings = (newSettings: Partial<AppSettings>) => {
    if (typeof window === 'undefined') return

    try {
      // 更新内存中的设置
      Object.assign(globalSettings.value, newSettings)
      
      // 保存到本地存储
      localStorage.setItem('app-settings', JSON.stringify(globalSettings.value))
      
      // 广播设置变更事件
      const event = new CustomEvent(SETTINGS_CHANGE_EVENT, {
        detail: { settings: globalSettings.value, changes: newSettings }
      })
      window.dispatchEvent(event)
      
      console.log('设置已保存:', newSettings)
    } catch (error) {
      console.error('保存设置失败:', error)
    }
  }

  // 监听设置变更
  const onSettingsChange = (callback: (settings: AppSettings, changes: Partial<AppSettings>) => void) => {
    if (typeof window === 'undefined') return () => {}

    const handler = (event: CustomEvent) => {
      const { settings, changes } = event.detail
      callback(settings, changes)
    }

    window.addEventListener(SETTINGS_CHANGE_EVENT, handler as EventListener)
    
    // 返回清理函数
    return () => {
      window.removeEventListener(SETTINGS_CHANGE_EVENT, handler as EventListener)
    }
  }

  // 监听其他标签页的设置变更
  const onStorageChange = () => {
    if (typeof window === 'undefined') return () => {}

    const handler = (event: StorageEvent) => {
      if (event.key === 'app-settings' && event.newValue) {
        try {
          const parsed = JSON.parse(event.newValue)
          Object.assign(globalSettings.value, parsed)
        } catch (error) {
          console.error('解析存储设置失败:', error)
        }
      }
    }

    window.addEventListener('storage', handler)
    
    return () => {
      window.removeEventListener('storage', handler)
    }
  }

  // 更新特定设置项
  const updateSetting = <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
    saveSettings({ [key]: value } as Partial<AppSettings>)
  }

  // 重置设置为默认值
  const resetSettings = () => {
    saveSettings(defaultSettings)
  }

  // 手动刷新触发
  const triggerManualRefresh = () => {
    if (typeof window === 'undefined') return
    
    window.dispatchEvent(new CustomEvent('app-manual-refresh', {
      detail: { source: 'user-action' }
    }))
  }

  // 自动初始化（客户端）
  if (typeof window !== 'undefined') {
    loadSettings()
    onStorageChange()
  }

  return {
    // 设置管理
    settings: globalSettings, // 移除 readonly 以支持双向绑定
    isLoaded: readonly(isLoaded),
    loadSettings,
    saveSettings,
    updateSetting,
    resetSettings,
    onSettingsChange,
    onStorageChange,
    
    // 通知过滤相关
    isInQuietHours: readonly(isInQuietHours),
    shouldPlaySound: readonly(shouldPlaySound),
    shouldVibrate: readonly(shouldVibrate),
    shouldShowNotification,
    filterNotification,
    
    // 设置应用方法
    initializeSettings,
    setupSettingsWatchers,
    applyThemeSettings,
    applyWebSocketSettings,
    applyNavigationSettings,
    playNotificationSound,
    triggerVibration,
    triggerManualRefresh
  }
}
