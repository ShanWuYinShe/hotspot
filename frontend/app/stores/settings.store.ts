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
  const _state = ref<SettingsState>(getDefaultState())
  const _remoteLoaded = ref(false)

  // ============ Getter ============

  // 隐私标签
  const privacy = computed((): UserSettingsData => _state.value.privacy)

  // 通知标签
  const notifications = computed((): SubscriptionsData => _state.value.notifications)

  // 外观标签
  const appearance = computed((): AppearanceSettings => _state.value.appearance)

  // 外观快捷访问
  const theme = computed(() => appearance.value.theme)
  const language = computed(() => appearance.value.language)
  const enableAnimations = computed(() => appearance.value.enableAnimations)
  const compactMode = computed(() => appearance.value.compactMode)
  const sidebarCollapsed = computed(() => appearance.value.sidebarCollapsed)

  // 引导状态
  const guide = computed((): GuideState => _state.value.guide)

  // 引导快捷访问
  const hasVisited = computed(() => guide.value.hasVisited)
  const lastVisitTime = computed(() => guide.value.lastVisitTime)
  const neverShowLoginPrompt = computed(() => guide.value.neverShowLoginPrompt)
  const lastLoginPromptTime = computed(() => guide.value.lastLoginPromptTime)

  const remoteLoaded = computed(() => _remoteLoaded.value)

  // ============ Setter - 隐私标签 ============

  const setPrivacy = (data: UserSettingsData) => {
    _state.value.privacy = { ...data }
  }

  const updatePrivacy = (updates: Partial<UserSettingsData>) => {
    _state.value.privacy = { ..._state.value.privacy, ...updates }
  }

  // ============ Setter - 通知标签 ============

  const setNotifications = (data: SubscriptionsData) => {
    _state.value.notifications = { ...data }
  }

  const updateNotifications = (updates: Partial<SubscriptionsData>) => {
    _state.value.notifications = { ..._state.value.notifications, ...updates }
  }

  // ============ Setter - 外观标签 ============

  const setTheme = (value: ThemeMode) => {
    _state.value.appearance.theme = value
  }

  const setLanguage = (value: LanguageCode) => {
    _state.value.appearance.language = value
  }

  const setEnableAnimations = (value: boolean) => {
    _state.value.appearance.enableAnimations = value
  }

  const setCompactMode = (value: boolean) => {
    _state.value.appearance.compactMode = value
  }

  const setSidebarCollapsed = (value: boolean) => {
    _state.value.appearance.sidebarCollapsed = value
  }

  // ============ Setter - 引导状态 ============

  const markVisited = () => {
    _state.value.guide.hasVisited = true
    _state.value.guide.lastVisitTime = Date.now()
  }

  const updateLastVisitTime = () => {
    _state.value.guide.lastVisitTime = Date.now()
  }

  const setNeverShowLoginPrompt = (value: boolean) => {
    _state.value.guide.neverShowLoginPrompt = value
  }

  const updateLastLoginPromptTime = () => {
    _state.value.guide.lastLoginPromptTime = Date.now()
  }

  // ============ 其他 ============

  const markRemoteLoaded = () => {
    _remoteLoaded.value = true
  }

  const reset = () => {
    _state.value = getDefaultState()
    _remoteLoaded.value = false
  }

  return {
    _state,

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
    reset
  }
}, {
  persist: {
    key: 'app-settings',
    pick: ['_state']
  }
})
