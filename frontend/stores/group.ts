import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useGroupService } from '~/services'
import type { GroupInfo, GroupMember } from '~/types/models/group'
import type { CreateGroupRequest, JoinGroupRequest, LeaveGroupRequest } from '~/types/api/group'

// 定义本地分页响应类型
interface LocalPaginatedResponse<T> {
  items: T[]
  has_more: boolean
  next_cursor: number | string | null
  total?: number
}

export const useGroupStore = defineStore('group', () => {
  // 获取群组服务实例
  const {
    createGroup: createGroupApi,
    searchGroupsById,
    getUserGroups: getUserGroupsApi,
    searchGroupsByLocation,
    getGroupMembers: getGroupMembersApi,
    joinGroup: joinGroupApi,
    leaveGroup: leaveGroupApi
  } = useGroupService()

  // 状态
  const userGroups = ref<GroupInfo[]>([])
  const nearbyGroups = ref<GroupInfo[]>([])
  const currentGroup = ref<GroupInfo | null>(null)
  const groupMembers = ref<GroupMember[]>([])
  const recentJoinedGroups = ref<GroupInfo[]>([])
  const isLoading = ref<boolean>(false)
  const isLoadingMembers = ref<boolean>(false)
  const isCreating = ref<boolean>(false)
  const hasMoreGroups = ref<boolean>(false)
  const hasMoreMembers = ref<boolean>(false)
  const nextGroupsCursor = ref<string | number | null>(null)
  const nextMembersCursor = ref<string | number | null>(null)
  const error = ref<string | null>(null)

  // 计算属性
  const getUserGroups = computed(() => userGroups.value)
  const getNearbyGroups = computed(() => nearbyGroups.value)
  const getCurrentGroup = computed(() => currentGroup.value)
  const getGroupMembers = computed(() => groupMembers.value)
  const getRecentJoinedGroups = computed(() => recentJoinedGroups.value)

  // Action: 创建群组
  async function createGroup(name: string, description: string, location: { latitude: number; longitude: number; locationName: string }) {
    isCreating.value = true
    try {
      const response = await createGroupApi({
        name,
        description,
        latitude: location.latitude,
        longitude: location.longitude,
        location_name: location.locationName
      })

      error.value = null
      return response.group_id
    } catch (err: any) {
      error.value = err.message || '创建群组失败'
      throw err
    } finally {
      isCreating.value = false
    }
  }

  // Action: 获取用户的群组
  async function fetchUserGroups(refresh = false) {
    if (refresh) {
      userGroups.value = []
      nextGroupsCursor.value = null
    }

    isLoading.value = true
    try {
      const response = await getUserGroupsApi({
        limit: 20,
        offset: refresh ? 0 : userGroups.value.length
      })

      // 处理响应数据 - 根据 PaginatedResponse 格式
      if (refresh) {
        userGroups.value = response.items || []
      } else {
        userGroups.value = [...userGroups.value, ...(response.items || [])]
      }

      hasMoreGroups.value = response.has_more || false
      error.value = null
    } catch (err: any) {
      error.value = err.message || '获取群组失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Action: 获取附近的群组
  async function fetchNearbyGroups(
    latitude: number,
    longitude: number,
    radius: number = 5000,
    refresh = false
  ) {
    if (refresh) {
      nearbyGroups.value = []
      nextGroupsCursor.value = null
    }

    isLoading.value = true
    try {
      const response = await searchGroupsByLocation({
        latitude,
        longitude,
        radius,
        limit: 20,
        cursor: refresh ? 0 : nearbyGroups.value.length
      })

      // 处理响应数据
      if (refresh) {
        nearbyGroups.value = response.items || []
      } else {
        nearbyGroups.value = [...nearbyGroups.value, ...(response.items || [])]
      }

      hasMoreGroups.value = response.has_more || false
      error.value = null
    } catch (err: any) {
      error.value = err.message || '获取附近群组失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Action: 获取群组详情
  async function getGroupDetail(groupId: string) {
    isLoading.value = true
    try {
      const groupInfo = await searchGroupsById({ group_id: groupId })
      currentGroup.value = groupInfo
      error.value = null
      return groupInfo
    } catch (err: any) {
      error.value = err.message || '获取群组详情失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // Action: 获取群组成员
  async function fetchGroupMembers(groupId: string, refresh = false) {
    if (refresh) {
      groupMembers.value = []
      nextMembersCursor.value = null
    }

    isLoadingMembers.value = true
    try {
      const response = await getGroupMembersApi({
        group_id: groupId,
        cursor: typeof nextMembersCursor.value === 'number' ? nextMembersCursor.value : 0,
        limit: 20
      })

      if (refresh) {
        groupMembers.value = response.items || []
      } else {
        groupMembers.value = [...groupMembers.value, ...(response.items || [])]
      }

      hasMoreMembers.value = response.has_more || false
      nextMembersCursor.value = response.next_cursor || null
      error.value = null
    } catch (err: any) {
      error.value = err.message || '获取群组成员失败'
      throw err
    } finally {
      isLoadingMembers.value = false
    }
  }

  // Action: 加入群组
  async function joinGroup(groupId: string, password?: string) {
    try {
      const response = await joinGroupApi({ group_id: groupId, password })

      // 刷新用户群组列表
      fetchUserGroups(true)

      // 如果已经有群组详情，更新成员状态
      if (currentGroup.value && currentGroup.value.group_id === groupId) {
        currentGroup.value.is_member = true
        currentGroup.value.user_role = 'member'
      }

      error.value = null
      return response
    } catch (err: any) {
      error.value = err.message || '加入群组失败'
      throw err
    }
  }

  // Action: 退出群组
  async function leaveGroup(groupId: string) {
    try {
      const response = await leaveGroupApi({ group_id: groupId })

      // 从用户群组列表中移除
      userGroups.value = userGroups.value.filter((group: GroupInfo) => group.group_id !== groupId)

      // 如果已经有群组详情，更新成员状态
      if (currentGroup.value && currentGroup.value.group_id === groupId) {
        currentGroup.value.is_member = false
        currentGroup.value.user_role = ''
      }

      error.value = null
      return response
    } catch (err: any) {
      error.value = err.message || '退出群组失败'
      throw err
    }
  }

  // Action: 获取最近加入的群组
  async function fetchRecentJoinedGroups() {
    try {
      // 使用用户群组接口获取最近的群组
      const groups = await getUserGroupsApi({
        limit: 5,
        offset: 0
      })
      recentJoinedGroups.value = groups.items || []
      error.value = null
      return groups
    } catch (err: any) {
      error.value = err.message || '获取最近加入群组失败'
      throw err
    }
  }

  return {
    userGroups,
    nearbyGroups,
    currentGroup,
    groupMembers,
    recentJoinedGroups,
    isLoading,
    isLoadingMembers,
    isCreating,
    hasMoreGroups,
    hasMoreMembers,
    error,
    getUserGroups,
    getNearbyGroups,
    getCurrentGroup,
    getGroupMembers,
    getRecentJoinedGroups,
    createGroup,
    fetchUserGroups,
    fetchNearbyGroups,
    getGroupDetail,
    fetchGroupMembers,
    joinGroup,
    leaveGroup,
    fetchRecentJoinedGroups
  }
}, {
  persist: true
})