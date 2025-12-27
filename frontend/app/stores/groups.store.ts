/**
 * 群组偏好存储 - 纯持久化状态管理
 * 
 * ✅ 职责：仅管理持久化的群组偏好设置
 * ❌ 不包含：群组数据、API 调用、业务逻辑
 * 
 * 使用 pinia-plugin-persistedstate 自动处理水合
 */

import { defineStore } from 'pinia'
import type { GroupSearchItem } from '~/types/api'

export type GroupType = 'public_chat' | 'private_chat' | 'announcement'

export interface Group extends GroupSearchItem {
  is_member?: boolean
  my_role?: 'owner' | 'admin' | 'member'
  unread_count?: number
}

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
  const _groups = ref<Record<string, Group>>({})
  const _feeds = ref({
    nearby: [] as Group[]
  })
  
  // ============ Getter ============

  const preferences = computed((): GroupPreferences => _preferences.value)
  const defaultSearchRadius = computed(() => preferences.value.defaultSearchRadius)
  const preferredGroupTypes = computed(() => preferences.value.preferredGroupTypes)
  const autoJoinNearbyGroups = computed(() => preferences.value.autoJoinNearbyGroups)
  const notificationSettings = computed(() => preferences.value.notificationSettings)
  
  const groups = computed(() => _groups.value)
  const nearbyGroups = computed(() => _feeds.value.nearby)
  const getGroup = (id: string) => _groups.value[id]

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
  
  const setGroup = (group: GroupSearchItem) => {
    const existing = _groups.value[group.id]
    _groups.value[group.id] = { ...existing, ...group }
  }
  
  const setGroups = (groupsList: GroupSearchItem[]) => {
    groupsList.forEach(group => setGroup(group))
  }
  
  const setNearbyGroups = (groupsList: GroupSearchItem[], loadMore = false) => {
    const newGroups = groupsList.map(item => ({ ...item }))
    
    if (loadMore) {
      _feeds.value.nearby.push(...newGroups)
    } else {
      _feeds.value.nearby = newGroups
    }
    
    newGroups.forEach(group => setGroup(group))
  }

  /**
   * 清空用户相关缓存（列表/feeds），保留偏好（设备级）
   * - 多账号切换/锁定时使用，避免跨账号看到旧列表
   */
  const clearUserCache = () => {
    _groups.value = {}
    _feeds.value = { nearby: [] }
  }

  return {
    // 内部状态（用于持久化）
    _preferences,
    _groups,
    _feeds,

    // Getter
    preferences,
    defaultSearchRadius,
    preferredGroupTypes,
    autoJoinNearbyGroups,
    notificationSettings,
    groups,
    nearbyGroups,
    getGroup,

    // Setter
    setDefaultSearchRadius,
    setPreferredGroupTypes,
    setAutoJoinNearbyGroups,
    setNotificationSettings,
    resetPreferences,
    clearUserCache,
    setGroup,
    setGroups,
    setNearbyGroups
  }
}, {
  persist: {
    key: 'groups-store',
    // 多账号隔离：群组列表/推荐流与用户强相关（包含 is_member/my_role/unread 等）
    // 仅持久化“偏好”避免跨账号显示旧列表数据
    pick: ['_preferences']
  }
})
