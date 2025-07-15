import { defineStore } from 'pinia'
import { ref, computed, readonly } from 'vue'
import { useMomentService } from '~/services/moment.service'
import type { CheckInInfo } from '~/types/api/checkin'

/**
 * 时刻动态状态管理
 * 管理用户的时刻动态数据
 */
export const useMomentStore = defineStore('moment', () => {
  // 获取服务实例
  const momentService = useMomentService()

  // 状态
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  // 时刻动态数据
  const moments = ref<CheckInInfo[]>([])
  const nearbyMoments = ref<CheckInInfo[]>([])
  const userMoments = ref<CheckInInfo[]>([])
  const currentMoment = ref<CheckInInfo | null>(null)
  
  // 分页状态
  const hasMore = ref(true)
  const nextCursor = ref<number | undefined>(undefined)
  
  // 计算属性
  const totalMoments = computed(() => moments.value.length)
  const hasNearbyMoments = computed(() => nearbyMoments.value.length > 0)
  const hasUserMoments = computed(() => userMoments.value.length > 0)
  
  // 创建时刻动态
  const createMoment = async (data: {
    description: string
    latitude: number
    longitude: number
    location_name: string
    tags?: string[]
  }) => {
    try {
      isLoading.value = true
      error.value = null
      
      const response = await momentService.createMoment(data)
      
      if (response?.checkin_id) {
        // 获取创建的时刻详情
        const momentDetails = await momentService.getMomentDetails({ checkin_id: response.checkin_id })
        if (momentDetails) {
          moments.value.unshift(momentDetails)
          userMoments.value.unshift(momentDetails)
        }
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '创建时刻失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }
  
  // 获取附近的时刻动态
  const fetchNearbyMoments = async (params: {
    latitude: number
    longitude: number
    radius: number
    limit?: number
    cursor?: number
  }) => {
    try {
      isLoading.value = true
      error.value = null
      
      const response = await momentService.getNearbyMoments(params)
      
      if (response?.items) {
        if (params.cursor) {
          nearbyMoments.value.push(...response.items)
        } else {
          nearbyMoments.value = response.items
        }
        
        hasMore.value = response.has_more
        nextCursor.value = response.next_cursor
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '获取附近时刻失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }
  
  // 获取用户的时刻动态
  const fetchUserMoments = async (params: {
    user_id?: number
    limit?: number
    cursor?: number
  }) => {
    try {
      isLoading.value = true
      error.value = null
      
      const response = await momentService.getUserMoments(params)
      
      if (response?.items) {
        if (params.cursor) {
          userMoments.value.push(...response.items)
        } else {
          userMoments.value = response.items
        }
        
        hasMore.value = response.has_more
        nextCursor.value = response.next_cursor
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '获取用户时刻失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }
  
  // 获取时刻详情
  const fetchMomentDetails = async (checkinId: string) => {
    try {
      isLoading.value = true
      error.value = null
      
      const response = await momentService.getMomentDetails({ checkin_id: checkinId })
      
      if (response) {
        currentMoment.value = response
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '获取时刻详情失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }
  
  // 删除时刻动态
  const deleteMoment = async (checkinId: string) => {
    try {
      isLoading.value = true
      error.value = null
      
      const response = await momentService.deleteMoment({ checkin_id: checkinId })
      
      if (response?.success) {
        // 从各个列表中移除
        nearbyMoments.value = nearbyMoments.value.filter((m: CheckInInfo) => m.checkin_id !== checkinId)
        userMoments.value = userMoments.value.filter((m: CheckInInfo) => m.checkin_id !== checkinId)
        moments.value = moments.value.filter((m: CheckInInfo) => m.checkin_id !== checkinId)
        
        if (currentMoment.value?.checkin_id === checkinId) {
          currentMoment.value = null
        }
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '删除时刻失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }
  
  // 更新时刻动态
  const updateMoment = (moment: CheckInInfo) => {
    const updateInArray = (arr: CheckInInfo[]) => {
      const index = arr.findIndex(m => m.checkin_id === moment.checkin_id)
      if (index !== -1) {
        arr[index] = moment
      }
    }
    
    updateInArray(moments.value)
    updateInArray(nearbyMoments.value)
    updateInArray(userMoments.value)
    
    if (currentMoment.value?.checkin_id === moment.checkin_id) {
      currentMoment.value = moment
    }
  }
  
  // 点赞/取消点赞
  const toggleLike = async (checkinId: string) => {
    try {
      const response = await momentService.toggleMomentLike({ checkin_id: checkinId })
      
      if (response) {
        // 更新所有相关的时刻数据
        const updateLikeStatus = (arr: CheckInInfo[]) => {
          const moment = arr.find(m => m.checkin_id === checkinId)
          if (moment) {
            moment.liked_by_me = response.liked
            moment.likes_count = response.like_count
          }
        }
        
        updateLikeStatus(moments.value)
        updateLikeStatus(nearbyMoments.value)
        updateLikeStatus(userMoments.value)
        
        if (currentMoment.value?.checkin_id === checkinId) {
          currentMoment.value.liked_by_me = response.liked
          currentMoment.value.likes_count = response.like_count
        }
      }
      
      return response
    } catch (err: any) {
      error.value = err.message || '操作失败'
      throw err
    }
  }
  
  // 清空错误
  const clearError = () => {
    error.value = null
  }
  
  // 重置状态
  const reset = () => {
    moments.value = []
    nearbyMoments.value = []
    userMoments.value = []
    currentMoment.value = null
    hasMore.value = true
    nextCursor.value = undefined
    isLoading.value = false
    error.value = null
  }
  
  return {
    // 状态
    isLoading: readonly(isLoading),
    error: readonly(error),
    
    // 数据
    moments: readonly(moments),
    nearbyMoments: readonly(nearbyMoments),
    userMoments: readonly(userMoments),
    currentMoment: readonly(currentMoment),
    
    // 分页
    hasMore: readonly(hasMore),
    nextCursor: readonly(nextCursor),
    
    // 计算属性
    totalMoments,
    hasNearbyMoments,
    hasUserMoments,
    
    // 方法
    createMoment,
    fetchNearbyMoments,
    fetchUserMoments,
    fetchMomentDetails,
    deleteMoment,
    updateMoment,
    toggleLike,
    clearError,
    reset
  }
})

// 导出类型
export type MomentStore = ReturnType<typeof useMomentStore>