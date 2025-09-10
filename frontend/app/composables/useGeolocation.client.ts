/**
 * 地理位置 Composable (客户端专用)
 * 职责：位置获取与计算。
 * 权限检查/请求统一迁移到 usePermissions.client.ts。
 */

import { ref, computed } from 'vue'
import { formatDistance as formatDistanceUtil } from '~/utils/formatters'

export interface Coordinates {
  latitude: number
  longitude: number
  accuracy?: number
  timestamp?: number
}

export const useGeolocation = () => {
  const coordinates = ref<Coordinates | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const isSupported = ref(typeof navigator !== 'undefined' && 'geolocation' in navigator)

  // 计算属性
  const hasPermission = computed(() => !!coordinates.value)

  // 获取当前位置
  const getCurrentPosition = async (options?: PositionOptions): Promise<Coordinates> => {
    if (typeof navigator === 'undefined') {
      throw new Error('地理位置API仅在客户端可用')
    }
    
    if (!isSupported.value) {
      throw new Error('您的浏览器不支持地理位置功能')
    }

    isLoading.value = true
    error.value = null

    try {
      // 必须由用户手势触发（由调用方保证）。此处仅封装调用。
      // 若未授权，则直接抛错，避免浏览器弹出权限提示导致违规日志。
      try {
        // @ts-ignore
        const perm: PermissionStatus = await (navigator.permissions?.query?.({ name: 'geolocation' as any }))
        if (perm && perm.state !== 'granted') {
          throw new Error('尚未授权位置权限，请在用户操作中调用权限请求后再获取位置')
        }
      } catch (_) {
        // 无 Permissions API 支持时跳过预检，由调用方确保用户手势
      }

      const position = await new Promise<GeolocationPosition>((resolve, reject) => {
        try {
          navigator.geolocation.getCurrentPosition(
            resolve,
            reject,
            {
              enableHighAccuracy: true,
              timeout: 15000, // 15秒
              maximumAge: 60000, // 1分钟缓存
              ...options
            }
          )
        } catch (e) {
          reject(e)
        }
      })

      const coords: Coordinates = {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        accuracy: position.coords.accuracy,
        timestamp: position.timestamp
      }

      coordinates.value = coords
      return coords

    } catch (err: any) {
      let message = '获取位置失败'
      
      // 检查是否是 GeolocationPositionError
      if (err && typeof err.code === 'number') {
        switch (err.code) {
          case 1: // PERMISSION_DENIED
            message = '位置权限被拒绝。请点击浏览器地址栏的位置图标，选择"允许"位置访问权限'
            break
          case 2: // POSITION_UNAVAILABLE
            message = '位置信息不可用，请检查GPS是否开启'
            break
          case 3: // TIMEOUT
            message = '获取位置超时，请确保网络连接正常并重试'
            break
          default:
            message = err.message || '获取位置失败，请重试'
        }
      } else {
        message = err?.message || '获取位置失败，请重试'
      }

      error.value = message
      throw new Error(message)

    } finally {
      isLoading.value = false
    }
  }

  // 监听位置变化
  const watchPosition = (
    callback: (coords: Coordinates) => void,
    options?: PositionOptions
  ): number | null => {
    if (typeof navigator === 'undefined') {
      error.value = '地理位置API仅在客户端可用'
      return null
    }
    
    if (!isSupported.value) {
      error.value = '您的浏览器不支持地理位置功能'
      return null
    }

    return navigator.geolocation.watchPosition(
      (position) => {
        const coords: Coordinates = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy,
          timestamp: position.timestamp
        }
        
        coordinates.value = coords
        callback(coords)
      },
      (err) => {
        error.value = `监听位置失败: ${err.message}`
      },
      {
        enableHighAccuracy: true,
        timeout: 15000,
        maximumAge: 60000,
        ...options
      }
    )
  }

  // 停止监听位置
  const clearWatch = (watchId: number) => {
    if (typeof navigator !== 'undefined' && isSupported.value) {
      navigator.geolocation.clearWatch(watchId)
    }
  }

  // 计算两点间距离（单位：米）
  const calculateDistance = (
    lat1: number,
    lon1: number,
    lat2: number,
    lon2: number
  ): number => {
    const R = 6371e3 // 地球半径（米）
    const φ1 = (lat1 * Math.PI) / 180
    const φ2 = (lat2 * Math.PI) / 180
    const Δφ = ((lat2 - lat1) * Math.PI) / 180
    const Δλ = ((lon2 - lon1) * Math.PI) / 180

    const a =
      Math.sin(Δφ / 2) * Math.sin(Δφ / 2) +
      Math.cos(φ1) * Math.cos(φ2) * Math.sin(Δλ / 2) * Math.sin(Δλ / 2)
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))

    return R * c
  }

  // 格式化距离显示（使用统一工具函数）
  const formatDistance = formatDistanceUtil

  // 权限相关统一走 usePermissions
  // 移除权限请求方法：请使用 usePermissions().requestGeolocation()

  // 移除权限检查方法：请使用 usePermissions().checkGeolocation()

  // 权限监听不再透传，请直接使用 usePermissions().onGeolocationChange

  // 获取地理编码（坐标转地址）
  const reverseGeocode = async (lat: number, lon: number): Promise<string> => {
    try {
      // 这里可以集成地图服务API，如高德、百度等
      // 暂时返回坐标信息
      return `${lat.toFixed(6)}, ${lon.toFixed(6)}`
    } catch (err) {
      return `${lat.toFixed(6)}, ${lon.toFixed(6)}`
    }
  }

  // 清除位置信息
  const clearLocation = () => {
    coordinates.value = null
    error.value = null
  }

  // 初始化静默获取：遵循“用户手势触发”，默认不在初始化时获取
  const initializeLocation = async () => {}

  return {
    // 状态
    coordinates: readonly(coordinates),
    isLoading: readonly(isLoading),
    error: readonly(error),
    isSupported: readonly(isSupported),
    
    // 计算属性
    hasPermission,
    
    // 方法
    getCurrentPosition,
    watchPosition,
    clearWatch,
    calculateDistance,
    formatDistance,
    // 权限相关请使用 usePermissions()
    reverseGeocode,
    clearLocation,
    initializeLocation
  }
}
