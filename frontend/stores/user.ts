import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useUserService } from '~/services/user.service'
import { DEFAULT_LATITUDE, DEFAULT_LONGITUDE } from '~/config'
import type { UserProfile, UserListItem } from '~/types/models/user'
import type { UpdateLocationRequest } from '~/types/api/user'
import { extractErrorMessage } from '~/utils/error-handler'

// 定义默认位置常量（南天门）
const DEFAULT_LOCATION_NAME = '南天门'

export const useUserStore = defineStore('user', () => {
  // 获取服务实例
  const userService = useUserService()

  // 状态
  const nearbyUsers = ref<UserListItem[]>([])
  const currentLocation = ref<{ latitude: number; longitude: number; locationName: string } | null>(null)
  const isLoadingNearby = ref<boolean>(false)
  const hasMoreNearby = ref<boolean>(false)
  const nextCursor = ref<string | number | null>(null)
  const searchResults = ref<UserListItem[]>([])
  const currentUser = ref<UserProfile | null>(null)
  const defaultRadius = ref<number>(1000)
  const error = ref<string | null>(null)
  const mapTipShown = ref<boolean>(false)

  // 计算属性
  const getNearbyUsers = computed(() => nearbyUsers.value)
  const getCurrentLocation = computed(() => currentLocation.value)
  const getSearchResults = computed(() => searchResults.value)
  const hasLocation = computed(() => !!currentLocation.value)

  // Action: 设置当前位置
  function setCurrentLocation(latitude: number, longitude: number, locationName: string) {
    currentLocation.value = { latitude, longitude, locationName }
  }

  // Action: 清除当前位置
  function clearCurrentLocation() {
    currentLocation.value = null
  }

  // Action: 更新用户位置到服务器
  async function updateLocationToServer() {
    if (!currentLocation.value) return
    
    isLoadingNearby.value = true
    error.value = null
    const result = await userService.updateUserLocation({
      latitude: currentLocation.value.latitude,
      longitude: currentLocation.value.longitude,
      location_name: currentLocation.value.locationName
    })
    if (!result) {
      error.value = '更新位置失败'
      isLoadingNearby.value = false
      return
    }
    error.value = null
    isLoadingNearby.value = false
  }
  
  // Action: 获取附近用户
  async function fetchNearbyUsers(refresh = false) {
    if (!currentLocation.value) {
      error.value = "无法获取位置信息"
      return
    }
    isLoadingNearby.value = true
    const result = await userService.searchUsersByLocation({
      latitude: currentLocation.value.latitude,
      longitude: currentLocation.value.longitude,
      radius: defaultRadius.value,
      cursor: refresh ? undefined : (typeof nextCursor.value === 'number' ? nextCursor.value : undefined)
    })
    if (!result) {
      error.value = '获取附近用户失败'
      isLoadingNearby.value = false
      return
    }
    if (refresh) {
      nearbyUsers.value = result.items || []
    } else {
      // 合并结果，避免重复
      const newUsers = (result.items || []).filter(
        (newUser: any) => !nearbyUsers.value.some(existingUser => existingUser.user_id === newUser.user_id)
      )
      nearbyUsers.value = [...nearbyUsers.value, ...newUsers]
    }
    hasMoreNearby.value = result.has_more || false
    nextCursor.value = result.next_cursor || null
    error.value = null
    isLoadingNearby.value = false
  }
  
  // Action: 搜索用户
  async function searchUsersByName(keyword: string) {
    if (!keyword.trim()) {
      searchResults.value = []
      return
    }
    isLoadingNearby.value = true
    error.value = null
    const result = await userService.searchUsersByName({ keyword, cursor: 0, limit: 20 })
    if (!result) {
      error.value = '搜索用户失败'
      isLoadingNearby.value = false
      return
    }
    searchResults.value = result.items || []
    error.value = null
    isLoadingNearby.value = false
  }
  
  // Action: 获取用户详情
  async function getUserById(userId: number | string) {
    isLoadingNearby.value = true
    const result = await userService.searchUsersById({ user_id: Number(userId) })
    if (!result) {
      error.value = '获取用户信息失败'
      isLoadingNearby.value = false
      return null
    }
    currentUser.value = result
    error.value = null
    isLoadingNearby.value = false
    return result
  }

  // Action: 设置地图提示显示状态
  function setMapTipShown(val: boolean) {
    mapTipShown.value = val
  }

  return {
    nearbyUsers,
    currentLocation,
    isLoadingNearby,
    hasMoreNearby,
    nextCursor,
    searchResults,
    currentUser,
    defaultRadius,
    error,
    getNearbyUsers,
    getCurrentLocation,
    getSearchResults,
    hasLocation,
    setCurrentLocation,
    clearCurrentLocation,
    updateLocationToServer,
    fetchNearbyUsers,
    searchUsersByName,
    getUserById,
    mapTipShown,
    setMapTipShown
  }
}, {
  persist: true
})