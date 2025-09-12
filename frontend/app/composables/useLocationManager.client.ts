/**
 * 位置管理 Composable
 * 负责自动化的位置获取、更新和管理
 */

import { getCurrentInstance, onUnmounted, readonly, ref } from 'vue'
import { useUserService } from '~/services/user.service'
import { useAuthStore } from '~/stores/auth.store'
import { DEFAULTS } from '~/config/defaults'

interface LocationState {
  latitude: number
  longitude: number
  accuracy?: number
  timestamp: number
}

export const useLocationManager = () => {
  const userService = useUserService()
  const authStore = useAuthStore()
  
  // 位置状态
  const currentLocation = ref<LocationState | null>(null)
  const isLocationEnabled = ref(false)
  const locationError = ref<string | null>(null)
  const isUpdatingLocation = ref(false)
  
  // 配置参数
  const UPDATE_INTERVAL = 5 * 60 * 1000 // 5分钟更新一次
  const LOCATION_TIMEOUT = 10000 // 位置获取超时时间
  
  let updateTimer: NodeJS.Timeout | null = null
  let watchId: number | null = null

  /**
   * 获取当前位置
   */
  const getCurrentPosition = (): Promise<LocationState> => {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error('浏览器不支持地理位置服务'))
        return
      }

      navigator.geolocation.getCurrentPosition(
        (position) => {
          const location: LocationState = {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            accuracy: position.coords.accuracy,
            timestamp: Date.now()
          }
          resolve(location)
        },
        (error) => {
          console.warn('获取位置失败:', error.message)
          reject(error)
        },
        {
          enableHighAccuracy: true,
          timeout: LOCATION_TIMEOUT,
          maximumAge: 60000 // 1分钟内的缓存位置可接受
        }
      )
    })
  }


  /**
   * 更新位置到服务器
   */
  const updateLocationToServer = async (location: LocationState) => {
    if (!authStore.isAuthenticated) {
      if (import.meta.dev) console.log('用户未认证，跳过位置更新')
      return
    }

    try {
      isUpdatingLocation.value = true
      await userService.updateLocation(
        location.latitude, 
        location.longitude, 
        '当前位置' // 可以后续优化为反地理编码获取地址
      )
      if (import.meta.dev) console.log('位置更新成功:', location)
    } catch (error) {
      console.warn('位置更新失败:', error)
      // 位置更新失败不影响其他功能，只记录日志
    } finally {
      isUpdatingLocation.value = false
    }
  }

  /**
   * 处理位置更新
   */
  const handleLocationUpdate = async (newLocation: LocationState) => {
    // 更新本地状态
    currentLocation.value = newLocation
    locationError.value = null

    // 更新到服务器
    await updateLocationToServer(newLocation)
  }

  /**
   * 获取位置（带默认值回退）
   */
  const getLocationWithFallback = async (): Promise<LocationState> => {
    try {
      const location = await getCurrentPosition()
      return location
    } catch (error) {
      console.warn('获取真实位置失败，使用默认位置:', error)
      locationError.value = error instanceof Error ? error.message : '位置获取失败'
      
      // 返回默认位置（南天门）
      return {
        latitude: DEFAULTS.latitude,
        longitude: DEFAULTS.longitude,
        timestamp: Date.now()
      }
    }
  }

  /**
   * 启动自动位置更新
   */
  const startLocationTracking = async () => {
    if (import.meta.dev) console.log('启动自动位置跟踪')
    
    // 立即获取一次位置
    try {
      const location = await getLocationWithFallback()
      await handleLocationUpdate(location)
      isLocationEnabled.value = true
    } catch (error) {
      console.error('初始位置获取失败:', error)
    }

    // 设置定时更新
    if (updateTimer) {
      clearInterval(updateTimer)
    }
    
    updateTimer = setInterval(async () => {
      try {
        const location = await getLocationWithFallback()
        await handleLocationUpdate(location)
      } catch (error) {
        console.warn('定时位置更新失败:', error)
      }
    }, UPDATE_INTERVAL)
  }

  /**
   * 停止位置跟踪
   */
  const stopLocationTracking = () => {
    if (import.meta.dev) console.log('停止位置跟踪')
    
    if (updateTimer) {
      clearInterval(updateTimer)
      updateTimer = null
    }
    
    if (watchId !== null) {
      navigator.geolocation.clearWatch(watchId)
      watchId = null
    }
    
    isLocationEnabled.value = false
  }

  /**
   * 手动刷新位置
   */
  const refreshLocation = async (): Promise<LocationState> => {
    const location = await getLocationWithFallback()
    await handleLocationUpdate(location)
    return location
  }

  /**
   * 获取当前位置（不更新到服务器）
   */
  const getLocation = (): LocationState | null => {
    return currentLocation.value
  }

  /**
   * 获取当前位置或默认位置
   */
  const getLocationOrDefault = (): LocationState => {
    return currentLocation.value || {
      latitude: DEFAULTS.latitude,
      longitude: DEFAULTS.longitude,
      timestamp: Date.now()
    }
  }

  // 组件卸载时清理（仅在组件上下文中）
  if (getCurrentInstance()) {
    onUnmounted(() => {
      stopLocationTracking()
    })
  }

  return {
    // 状态
    currentLocation: readonly(currentLocation),
    isLocationEnabled: readonly(isLocationEnabled),
    locationError: readonly(locationError),
    isUpdatingLocation: readonly(isUpdatingLocation),
    
    // 方法
    startLocationTracking,
    stopLocationTracking,
    refreshLocation,
    getLocation,
    getLocationOrDefault,
    getCurrentPosition,
    getLocationWithFallback
  }
}
