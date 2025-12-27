/**
 * 设置存储 - 直接按设置页面标签分类
 * 
 * 结构对应：
 * - privacy: 隐私标签（隐私设置、社交设置、资料可见性、搜索设置）
 * - notifications: 通知标签
 * - appearance: 外观标签
 * - guide: 引导状态（不在设置页面，但需要持久化）
 * 
 * 账户标签 → 由 authStore 管理
 * 危险操作/关于 → 无需持久化状态
 */

import { defineStore } from 'pinia'
import type { UserSettingsData, SubscriptionsData } from '~/types/api'

// ============ 类型定义 ============

export type ThemeMode = 'light' | 'dark' | 'system'
export type LanguageCode = 'zh-CN' | 'en-US'

// 外观设置（对应"外观"标签）
export interface AppearanceSettings {
  theme: ThemeMode
  language: LanguageCode
  enableAnimations: boolean
  compactMode: boolean
  sidebarCollapsed: boolean  // 侧边栏折叠状态
}

// 引导状态（不在设置页面）
export interface GuideState {
  hasVisited: boolean
  lastVisitTime: number
  neverShowLoginPrompt: boolean
  lastLoginPromptTime: number
}

// 完整设置状态
export interface SettingsState {
  privacy: UserSettingsData        // 隐私标签（直接复用 API 类型）
  notifications: SubscriptionsData // 通知标签（直接复用 API 类型）
  appearance: AppearanceSettings   // 外观标签
  guide: GuideState                // 引导状态
}

// ============ 默认值 ============

const DEFAULT_PRIVACY: UserSettingsData = {
  privacy_level: 'public',
  location_visible: true,
  online_visible: true,
  last_online_visible: true,
  friends_visible: 'public',
  following_visible: 'public',
  followers_visible: 'public',
  moments_visible: 'public',
  stats_visible: 'public',
  allow_follow: true,
  allow_private_chat: 'everyone',
  searchable: true,
  phone_searchable: true,
  nickname_searchable: true,
  location_searchable: true
}

const DEFAULT_NOTIFICATIONS: SubscriptionsData = {
  new_follower_enabled: true,
  mutual_follow_enabled: true,
  group_invite_enabled: true,
  group_invite_response_enabled: true,
  group_application_enabled: true,
  group_application_response_enabled: true,
  message_mention_enabled: true,
  moment_like_enabled: true,
  moment_comment_enabled: true,
  comment_reply_enabled: true,
  system_announcement_enabled: true
}

const DEFAULT_APPEARANCE: AppearanceSettings = {
  theme: 'system',
  language: 'zh-CN',
  enableAnimations: true,
  compactMode: false,
  sidebarCollapsed: false
}

const DEFAULT_GUIDE: GuideState = {
  hasVisited: false,
  lastVisitTime: 0,
  neverShowLoginPrompt: false,
  lastLoginPromptTime: 0
}

const getDefaultState = (): SettingsState => ({
  privacy: { ...DEFAULT_PRIVACY },
  notifications: { ...DEFAULT_NOTIFICATIONS },
  appearance: { ...DEFAULT_APPEARANCE },
  guide: { ...DEFAULT_GUIDE }
})

export const useSettingsStore = defineStore('settings', () => {
  // ============ 状态 ============
  // 多账号隔离：
  // - appearance/guide 属于“设备级”
  // - privacy/notifications 属于“用户级”，跨账号持久化会串数据，所以不持久化
  const _privacy = ref<UserSettingsData>({ ...DEFAULT_PRIVACY })
  const _notifications = ref<SubscriptionsData>({ ...DEFAULT_NOTIFICATIONS })
  const _appearance = ref<AppearanceSettings>({ ...DEFAULT_APPEARANCE })
  const _guide = ref<GuideState>({ ...DEFAULT_GUIDE })
  const _remoteLoaded = ref(false)

  // ============ Getter ============

  // 隐私标签
  const privacy = computed((): UserSettingsData => _privacy.value)

  // 通知标签
  const notifications = computed((): SubscriptionsData => _notifications.value)

  // 外观标签
  const appearance = computed((): AppearanceSettings => _appearance.value)

  // 外观快捷访问
  const theme = computed(() => appearance.value.theme)
  const language = computed(() => appearance.value.language)
  const enableAnimations = computed(() => appearance.value.enableAnimations)
  const compactMode = computed(() => appearance.value.compactMode)
  const sidebarCollapsed = computed(() => appearance.value.sidebarCollapsed)

  // 引导状态
  const guide = computed((): GuideState => _guide.value)

  // 引导快捷访问
  const hasVisited = computed(() => guide.value.hasVisited)
  const lastVisitTime = computed(() => guide.value.lastVisitTime)
  const neverShowLoginPrompt = computed(() => guide.value.neverShowLoginPrompt)
  const lastLoginPromptTime = computed(() => guide.value.lastLoginPromptTime)

  const remoteLoaded = computed(() => _remoteLoaded.value)

  // ============ Setter - 隐私标签 ============

  const setPrivacy = (data: UserSettingsData) => {
    _privacy.value = { ...data }
  }

  const updatePrivacy = (updates: Partial<UserSettingsData>) => {
    _privacy.value = { ..._privacy.value, ...updates }
  }

  // ============ Setter - 通知标签 ============

  const setNotifications = (data: SubscriptionsData) => {
    _notifications.value = { ...data }
  }

  const updateNotifications = (updates: Partial<SubscriptionsData>) => {
    _notifications.value = { ..._notifications.value, ...updates }
  }

  // ============ Setter - 外观标签 ============

  const setTheme = (value: ThemeMode) => {
    _appearance.value.theme = value
  }

  const setLanguage = (value: LanguageCode) => {
    _appearance.value.language = value
  }

  const setEnableAnimations = (value: boolean) => {
    _appearance.value.enableAnimations = value
  }

  const setCompactMode = (value: boolean) => {
    _appearance.value.compactMode = value
  }

  const setSidebarCollapsed = (value: boolean) => {
    _appearance.value.sidebarCollapsed = value
  }

  // ============ Setter - 引导状态 ============

  const markVisited = () => {
    _guide.value.hasVisited = true
    _guide.value.lastVisitTime = Date.now()
  }

  const updateLastVisitTime = () => {
    _guide.value.lastVisitTime = Date.now()
  }

  const setNeverShowLoginPrompt = (value: boolean) => {
    _guide.value.neverShowLoginPrompt = value
  }

  const updateLastLoginPromptTime = () => {
    _guide.value.lastLoginPromptTime = Date.now()
  }

  // ============ 其他 ============

  const markRemoteLoaded = () => {
    _remoteLoaded.value = true
  }

  const reset = () => {
    _privacy.value = { ...DEFAULT_PRIVACY }
    _notifications.value = { ...DEFAULT_NOTIFICATIONS }
    _appearance.value = { ...DEFAULT_APPEARANCE }
    _guide.value = { ...DEFAULT_GUIDE }
    _remoteLoaded.value = false
  }

  /**
   * 仅重置“用户级”设置（多账号切换/锁定时使用）
   * - 保留设备级外观/引导状态
   */
  const resetUserScoped = () => {
    _privacy.value = { ...DEFAULT_PRIVACY }
    _notifications.value = { ...DEFAULT_NOTIFICATIONS }
    _remoteLoaded.value = false
  }

  return {
    // 内部状态（用于持久化/调试）
    _privacy,
    _notifications,
    _appearance,
    _guide,

    // Getter - 按标签分类
    privacy,
    notifications,
    appearance,
    guide,

    // Getter - 外观快捷访问
    theme,
    language,
    enableAnimations,
    compactMode,
    sidebarCollapsed,

    // Getter - 引导快捷访问
    hasVisited,
    lastVisitTime,
    neverShowLoginPrompt,
    lastLoginPromptTime,

    // 状态标志
    remoteLoaded,

    // Setter - 隐私标签
    setPrivacy,
    updatePrivacy,

    // Setter - 通知标签
    setNotifications,
    updateNotifications,

    // Setter - 外观标签
    setTheme,
    setLanguage,
    setEnableAnimations,
    setCompactMode,
    setSidebarCollapsed,

    // Setter - 引导状态
    markVisited,
    updateLastVisitTime,
    setNeverShowLoginPrompt,
    updateLastLoginPromptTime,

    // 其他
    markRemoteLoaded,
    reset,
    resetUserScoped
  }
}, {
  persist: {
    key: 'app-settings',
    // 只持久化“设备级”设置，避免多账号串数据
    pick: ['_appearance', '_guide']
  }
})
