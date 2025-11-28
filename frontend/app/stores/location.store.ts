/**
 * 位置存储 - 地理位置管理
 */

import { defineStore } from 'pinia'
import { useUserService } from '~/services/users.service'

export interface LocationState {
  currentPosition: GeolocationPosition | null
  currentLocation: {
    latitude: number
    longitude: number
    location_name: string
  } | null
  error: string | null
  isLoading: boolean
  isWatching: boolean
  watchId: number | null
}

export const useLocationStore = defineStore('location', () => {
  // ============ 状态 ============
  const state = ref<LocationState>({
    currentPosition: null,
    currentLocation: null,
    error: null,
    isLoading: false,
    isWatching: false,
    watchId: null
  })

  // ============ 计算属性 ============
  const hasPermission = computed(() => {
    return state.value.currentPosition !== null || state.value.error === null
  })

  const coordinates = computed(() => {
    if (!state.value.currentPosition) return null
    return {
      latitude: state.value.currentPosition.coords.latitude,
      longitude: state.value.currentPosition.coords.longitude
    }
  })

  // ============ 方法 ============

  /**
   * 检查地理位置权限
   */
  const checkPermission = async (): Promise<PermissionStatus | null> => {
    if (!('permissions' in navigator)) {
      return null
    }

    try {
      const result = await navigator.permissions.query({ name: 'geolocation' as PermissionName })
      return result
    } catch (error) {
      console.warn('权限查询失败:', error)
      return null
    }
  }

  /**
   * 请求地理位置权限
   */
  const requestPermission = async (): Promise<boolean> => {
    // 检查是否已经授权
    const permission = await checkPermission()
    if (permission?.state === 'granted') {
      return true
    }

    // 尝试获取位置（会触发权限请求）
    try {
      await getCurrentPosition()
      return true
    } catch (error) {
      return false
    }
  }

  /**
   * 获取当前位置
   * ✅ Store 只负责状态管理，不处理 UI 反馈
   * 由调用方（组件）捕获错误并显示 toast
   */
  const getCurrentPosition = async (): Promise<GeolocationPosition> => {
    state.value.isLoading = true
    state.value.error = null

    try {
      const position = await new Promise<GeolocationPosition>((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(
          resolve,
          reject,
          {
            enableHighAccuracy: true,
            timeout: 10000,
            maximumAge: 0
          }
        )
      })

      state.value.currentPosition = position
      state.value.currentLocation = {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        location_name: '' // 需要通过反向地理编码获取
      }

      return position
    } catch (error: any) {
      let errorMessage = '获取位置失败'

      switch (error.code) {
        case error.PERMISSION_DENIED:
          errorMessage = '位置权限被拒绝'
          break
        case error.POSITION_UNAVAILABLE:
          errorMessage = '位置信息不可用'
          break
        case error.TIMEOUT:
          errorMessage = '获取位置超时'
          break
      }

      state.value.error = errorMessage
      throw error
    } finally {
      state.value.isLoading = false
    }
  }

  /**
   * 开始监听位置变化
   */
  const watchPosition = (
    onSuccess?: (position: GeolocationPosition) => void,
    onError?: (error: GeolocationPositionError) => void
  ) => {
    if (state.value.isWatching) {
      console.warn('已经在监听位置变化')
      return
    }

    const watchId = navigator.geolocation.watchPosition(
      (position) => {
        state.value.currentPosition = position
        state.value.currentLocation = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          location_name: state.value.currentLocation?.location_name || ''
        }
        onSuccess?.(position)
      },
      (error) => {
        state.value.error = error.message
        onError?.(error)
      },
      {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 0
      }
    )

    state.value.watchId = watchId
    state.value.isWatching = true

    return watchId
  }

  /**
   * 停止监听位置变化
   */
  const clearWatch = () => {
    if (state.value.watchId !== null) {
      navigator.geolocation.clearWatch(state.value.watchId)
      state.value.watchId = null
      state.value.isWatching = false
    }
  }

  /**
   * 更新位置名称
   * ✅ Store 只负责状态管理，不处理 UI 反馈
   */
  const updateLocationName = async (latitude: number, longitude: number, name: string) => {
    const userService = useUserService()

    await userService.updateLocation({
      latitude,
      longitude,
      location_name: name
    })

    state.value.currentLocation = {
      latitude,
      longitude,
      location_name: name
    }
  }

  /**
   * 计算两点之间的距离（单位：米）
   */
  const calculateDistance = (
    lat1: number,
    lon1: number,
    lat2: number,
    lon2: number
  ): number => {
    const R = 6371000 // 地球半径（米）
    const dLat = ((lat2 - lat1) * Math.PI) / 180
    const dLon = ((lon2 - lon1) * Math.PI) / 180

    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos((lat1 * Math.PI) / 180) *
        Math.cos((lat2 * Math.PI) / 180) *
        Math.sin(dLon / 2) *
        Math.sin(dLon / 2)

    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
    return R * c
  }

  /**
   * 格式化距离显示
   */
  const formatDistance = (meters: number): string => {
    if (meters < 1000) {
      return `${Math.round(meters)}m`
    } else {
      return `${(meters / 1000).toFixed(1)}km`
    }
  }

  /**
   * 清除位置数据
   */
  const clearLocation = () => {
    clearWatch()
    state.value.currentPosition = null
    state.value.currentLocation = null
    state.value.error = null
  }

  // ============ 注意事项 ============
  // Store 不能使用生命周期钩子（onMounted/onUnmounted 等）
  // 如果需要在组件卸载时清理资源，应在组件中调用 clearWatch()
  // 例如：onUnmounted(() => locationStore.clearWatch())

  return {
    // 状态
    state,

    // 计算属性
    hasPermission,
    coordinates,

    // 方法
    checkPermission,
    requestPermission,
    getCurrentPosition,
    watchPosition,
    clearWatch,
    updateLocationName,
    calculateDistance,
    formatDistance,
    clearLocation
  }
})


