/**
 * 群组 Store - 改进版本
 * 使用 readonly + 选择性持久化（只持久化用户偏好）
 */

import type { 
  GroupResponse, 
  GroupMemberResponse,
  PaginatedResponse 
} from '~/types/api'

export interface Group extends GroupResponse {
  // 扩展字段
  isJoining?: boolean
  isLeaving?: boolean
  membersLoaded?: boolean
  optimisticJoin?: boolean
}

export interface GroupMember extends GroupMemberResponse {
  // 扩展字段
  isRemoving?: boolean
  isPromoting?: boolean
}

// 用户偏好设置（需要持久化）
export interface GroupPreferences {
  defaultSearchRadius: number
  preferredGroupTypes: string[]
  autoJoinNearbyGroups: boolean
  notificationSettings: {
    newGroupsNearby: boolean
    invitations: boolean
    memberUpdates: boolean
  }
}

export const useGroupsStore = defineStore('groups', () => {
  // ============ 内部状态 (私有) ============
  
  // 群组数据（不持久化，从服务器获取）
  const _groups = ref<Map<string, Group>>(new Map())
  const _members = ref<Map<string, GroupMember[]>>(new Map())
  const _myGroups = ref<Group[]>([])
  const _nearbyGroups = ref<Group[]>([])
  const _searchResults = ref<Group[]>([])
  
  // 用户偏好（需要持久化）
  const _preferences = ref<GroupPreferences>({
    defaultSearchRadius: 5000,
    preferredGroupTypes: ['public_chat'],
    autoJoinNearbyGroups: false,
    notificationSettings: {
      newGroupsNearby: true,
      invitations: true,
      memberUpdates: false
    }
  })
  
  // 临时状态（不持久化）
  const _loading = ref({
    myGroups: false,
    nearby: false,
    search: false,
    members: false,
    creating: false,
    joining: false
  })
  
  // 分页状态（不持久化）
  const _pagination = ref({
    myGroups: { hasMore: false, cursor: undefined as string | undefined },
    nearby: { hasMore: false, cursor: undefined as string | undefined },
    search: { hasMore: false, cursor: undefined as string | undefined }
  })

  // ============ 只读状态 (公开) ============
  const groups = readonly(_groups)
  const members = readonly(_members)
  const myGroups = readonly(_myGroups)
  const nearbyGroups = readonly(_nearbyGroups)
  const searchResults = readonly(_searchResults)
  const preferences = readonly(_preferences)
  const loading = readonly(_loading)
  const pagination = readonly(_pagination)

  // ============ 计算属性 ============
  const groupsList = computed(() => Array.from(_groups.value.values()))
  
  const ownedGroups = computed(() => 
    _myGroups.value.filter(group => group.user_role === 'owner')
  )
  
  const joinedGroups = computed(() => 
    _myGroups.value.filter(group => group.user_role !== 'owner')
  )
  
  const totalGroupsCount = computed(() => _myGroups.value.length)
  const ownedGroupsCount = computed(() => ownedGroups.value.length)

  // ============ Actions ============
  
  /**
   * 设置我的群组数据（由调用者提供）
   */
  const setMyGroups = (groups: Group[]) => {
    _myGroups.value = groups.map(item => ({ ...item } as Group))
    
    // 同时更新到 groups Map
    _myGroups.value.forEach(group => {
      _groups.value.set(group.group_id, group)
    })
  }

  /**
   * 设置搜索结果（由调用者提供）
   */
  const setSearchResults = (groups: Group[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newGroups = groups.map(item => ({ ...item } as Group))

    if (loadMore) {
      _searchResults.value.push(...newGroups)
    } else {
      _searchResults.value = newGroups
    }
    
    // 更新到 groups Map
    newGroups.forEach(group => {
      _groups.value.set(group.group_id, group)
    })
    
    // 更新分页信息
    _pagination.value.search = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newGroups
  }

  /**
   * 设置附近群组数据（由调用者提供）
   */
  const setNearbyGroups = (groups: Group[], hasMore: boolean, cursor?: string, loadMore = false) => {
    const newGroups = groups.map(item => ({ ...item } as Group))

    if (loadMore) {
      _nearbyGroups.value.push(...newGroups)
    } else {
      _nearbyGroups.value = newGroups
    }
    
    // 更新到 groups Map
    newGroups.forEach(group => {
      _groups.value.set(group.group_id, group)
    })
    
    // 更新分页信息
    _pagination.value.nearby = {
      hasMore,
      cursor: cursor || ''
    }
    
    return newGroups
  }

  /**
   * 设置群组详情（由调用者提供）
   */
  const setGroupDetails = (groupId: string, group: Group) => {
    const groupData = { ...group } as Group
    _groups.value.set(groupId, groupData)
    return groupData
  }

  /**
   * 添加新创建的群组（由调用者提供）
   */
  const addCreatedGroup = (group: Group) => {
    const groupData = { ...group } as Group
    _groups.value.set(group.group_id, groupData)
    _myGroups.value.unshift(groupData)
    return groupData
  }

  /**
   * 设置群组加入状态（由调用者提供）
   */
  const setGroupJoinStatus = (groupId: string, isJoining: boolean, optimisticJoin = false) => {
    const group = _groups.value.get(groupId)
    if (group) {
      group.isJoining = isJoining
      group.optimisticJoin = optimisticJoin
    }
  }

  /**
   * 设置群组退出状态（由调用者提供）
   */
  const setGroupLeaveStatus = (groupId: string, isLeaving: boolean) => {
    const group = _groups.value.get(groupId)
    if (group) {
      group.isLeaving = isLeaving
    }
  }

  /**
   * 从我的群组列表中移除群组
   */
  const removeFromMyGroups = (groupId: string) => {
    _myGroups.value = _myGroups.value.filter(g => g.group_id !== groupId)
  }

  /**
   * 设置群组成员数据（由调用者提供）
   */
  const setGroupMembers = (groupId: string, members: GroupMember[]) => {
    const membersList = members.map(item => ({ ...item } as GroupMember))
    _members.value.set(groupId, membersList)
    
    const group = _groups.value.get(groupId)
    if (group) {
      group.membersLoaded = true
      _groups.value.set(groupId, group)
    }
    
    return membersList
  }

  /**
   * 设置成员移除状态（由调用者提供）
   */
  const setMemberRemoveStatus = (groupId: string, userId: string, isRemoving: boolean) => {
    const membersList = _members.value.get(groupId) || []
    const member = membersList.find(m => m.user_id === userId)
    
    if (member) {
      member.isRemoving = isRemoving
      _members.value.set(groupId, [...membersList])
    }
  }

  /**
   * 从成员列表中移除成员
   */
  const removeMemberFromList = (groupId: string, userId: string) => {
    const membersList = _members.value.get(groupId) || []
    const updatedMembers = membersList.filter(m => m.user_id !== userId)
    _members.value.set(groupId, updatedMembers)
  }

  /**
   * 设置成员角色状态（由调用者提供）
   */
  const setMemberRoleStatus = (
    groupId: string, 
    userId: string, 
    isPromoting: boolean
  ) => {
    const membersList = _members.value.get(groupId) || []
    const member = membersList.find(m => m.user_id === userId)
    
    if (member) {
      member.isPromoting = isPromoting
      _members.value.set(groupId, [...membersList])
    }
  }

  /**
   * 更新成员角色
   */
  const updateMemberRole = (
    groupId: string, 
    userId: string, 
    role: 'owner' | 'admin' | 'member'
  ) => {
    const membersList = _members.value.get(groupId) || []
    const member = membersList.find(m => m.user_id === userId)
    
    if (member) {
      member.role = role
      _members.value.set(groupId, [...membersList])
    }
  }

  /**
   * 更新群组数据（由调用者提供）
   */
  const updateGroup = (groupId: string, updatedGroup: Group) => {
    const groupData = { ...updatedGroup } as Group
    _groups.value.set(groupId, groupData)
    
    // 同时更新我的群组列表
    const myGroupIndex = _myGroups.value.findIndex(g => g.group_id === groupId)
    if (myGroupIndex !== -1) {
      _myGroups.value[myGroupIndex] = groupData
    }
    
    return groupData
  }

  /**
   * 删除群组数据（由调用者提供）
   */
  const removeGroup = (groupId: string) => {
    // 从所有列表中移除
    _groups.value.delete(groupId)
    _members.value.delete(groupId)
    _myGroups.value = _myGroups.value.filter(g => g.group_id !== groupId)
    _nearbyGroups.value = _nearbyGroups.value.filter(g => g.group_id !== groupId)
    _searchResults.value = _searchResults.value.filter(g => g.group_id !== groupId)
  }

  /**
   * 更新用户偏好
   */
  const updatePreferences = (updates: Partial<GroupPreferences>) => {
    Object.assign(_preferences.value, updates)
  }

  /**
   * 获取群组
   */
  const getGroup = (groupId: string): Group | undefined => {
    return _groups.value.get(groupId)
  }

  /**
   * 获取群组成员
   */
  const getGroupMembers = (groupId: string): GroupMember[] => {
    return _members.value.get(groupId) || []
  }

  /**
   * 清除搜索结果
   */
  const clearSearchResults = () => {
    _searchResults.value = []
    _pagination.value.search = { hasMore: false, cursor: undefined }
  }

  /**
   * 清空指定类型的群组数据
   */
  const clearGroups = (type: 'my' | 'nearby' | 'search') => {
    switch (type) {
      case 'my':
        _myGroups.value = []
        break
      case 'nearby':
        _nearbyGroups.value = []
        _pagination.value.nearby = { hasMore: false, cursor: undefined }
        break
      case 'search':
        _searchResults.value = []
        _pagination.value.search = { hasMore: false, cursor: undefined }
        break
    }
  }

  // ============ 初始化 ============
  // 注意：Store 不负责自动加载数据，由调用者负责

    return {
    // ============ 只读状态 ============
    groups,
    members,
    myGroups,
    nearbyGroups,
    searchResults,
    preferences,
    loading,
    pagination,
    
    // ============ 计算属性 ============
    groupsList,
    ownedGroups,
    joinedGroups,
      totalGroupsCount,
      ownedGroupsCount,

    // ============ 方法 ============
    setMyGroups,
    setSearchResults,
    setNearbyGroups,
    setGroupDetails,
    addCreatedGroup,
    setGroupJoinStatus,
    setGroupLeaveStatus,
    removeFromMyGroups,
    setGroupMembers,
    setMemberRemoveStatus,
    removeMemberFromList,
    setMemberRoleStatus,
    updateMemberRole,
    updateGroup,
    removeGroup,
    updatePreferences,
    getGroup,
    getGroupMembers,
    clearSearchResults,
    clearGroups
  }
}, {
  // ============ 选择性持久化配置 ============
    persist: {
    key: 'groups-preferences',
    // 只持久化用户偏好，不持久化群组数据
    pick: ['_preferences']
    // 注意：默认已经使用 JSON.stringify/destr，无需重复配置
  }
})
