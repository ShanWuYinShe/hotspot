/**
 * 地理位置 Composable (服务端占位)
 * 与客户端版本保持接口一致，但提供安全的占位实现
 */

import { ref, readonly, computed } from 'vue'
import { DEFAULTS } from '~/config/defaults'
import type { Coordinates } from '~/types/geolocation'

// 扩展坐标接口，添加服务端占位属性
export interface ExtendedCoordinates extends Coordinates {
  accuracy?: number
  timestamp?: number
}

export const useGeolocation = () => {
  const coordinates = ref<ExtendedCoordinates | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const isSupported = ref(false)

  const hasPermission = computed(() => false)

  const getCurrentPosition = async (): Promise<Coordinates> => {
    // 服务端返回默认位置，避免发送无效的 0,0 坐标
    const fallback: Coordinates = { 
      latitude: DEFAULTS.latitude, 
      longitude: DEFAULTS.longitude, 
      accuracy: 0, 
      timestamp: Date.now() 
    }
    coordinates.value = fallback
    return fallback
  }

  const watchPosition = () => null
  const clearWatch = () => {}
  const calculateDistance = () => 0
  const formatDistance = (d: number) => `${Math.round(d)}m`
  const reverseGeocode = async (lat: number, lon: number) => `${lat.toFixed(6)}, ${lon.toFixed(6)}`
  const clearLocation = () => { coordinates.value = null; error.value = null }
  const initializeLocation = async () => {}

  return {
    coordinates: readonly(coordinates),
    isLoading: readonly(isLoading),
    error: readonly(error),
    isSupported: readonly(isSupported),
    hasPermission,
    getCurrentPosition,
    watchPosition,
    clearWatch,
    calculateDistance,
    formatDistance,
    reverseGeocode,
    clearLocation,
    initializeLocation
  }
}




