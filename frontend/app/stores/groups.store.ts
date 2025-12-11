/**
 * 群组偏好存储 - 纯持久化状态管理
 * 
 * ✅ 职责：仅管理持久化的群组偏好设置
 * ❌ 不包含：群组数据、API 调用、业务逻辑
 * 
 * 使用 pinia-plugin-persistedstate 自动处理水合
 */

import { defineStore } from 'pinia'

export type GroupType = 'public_chat' | 'private_chat' | 'announcement'

export interface GroupNotificationSettings {
  newGroupsNearby: boolean
  invitations: boolean
  memberUpdates: boolean
}

export interface GroupPreferences {
  defaultSearchRadius: number
  preferredGroupTypes: GroupType[]
  autoJoinNearbyGroups: boolean
  notificationSettings: GroupNotificationSettings
}

const DEFAULT_PREFERENCES: GroupPreferences = {
  defaultSearchRadius: 5000,
  preferredGroupTypes: ['public_chat'],
  autoJoinNearbyGroups: false,
  notificationSettings: {
    newGroupsNearby: true,
    invitations: true,
    memberUpdates: false
  }
}

export const useGroupsStore = defineStore('groups', () => {
  // ============ 内部状态 ============
  const _preferences = ref<GroupPreferences>({ ...DEFAULT_PREFERENCES })

  // ============ Getter ============

  const preferences = computed((): GroupPreferences => _preferences.value)
  const defaultSearchRadius = computed(() => preferences.value.defaultSearchRadius)
  const preferredGroupTypes = computed(() => preferences.value.preferredGroupTypes)
  const autoJoinNearbyGroups = computed(() => preferences.value.autoJoinNearbyGroups)
  const notificationSettings = computed(() => preferences.value.notificationSettings)

  // ============ Setter ============

  const setDefaultSearchRadius = (value: number) => {
    _preferences.value.defaultSearchRadius = value
  }

  const setPreferredGroupTypes = (value: GroupType[]) => {
    _preferences.value.preferredGroupTypes = value
  }

  const setAutoJoinNearbyGroups = (value: boolean) => {
    _preferences.value.autoJoinNearbyGroups = value
  }

  const setNotificationSettings = (value: Partial<GroupNotificationSettings>) => {
    _preferences.value.notificationSettings = {
      ..._preferences.value.notificationSettings,
      ...value
    }
  }

  const resetPreferences = () => {
    _preferences.value = { ...DEFAULT_PREFERENCES }
  }

  return {
    // 内部状态（用于持久化）
    _preferences,

    // Getter
    preferences,
    defaultSearchRadius,
    preferredGroupTypes,
    autoJoinNearbyGroups,
    notificationSettings,

    // Setter
    setDefaultSearchRadius,
    setPreferredGroupTypes,
    setAutoJoinNearbyGroups,
    setNotificationSettings,
    resetPreferences
  }
}, {
  persist: {
    key: 'groups-preferences',
    pick: ['_preferences']
  }
})
