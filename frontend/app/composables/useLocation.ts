/**
 * 位置管理 Composable - 简洁而强大
 * 提供地理位置获取和管理功能
 */

export interface LocationCoords {
  latitude: number
  longitude: number
  accuracy?: number
  altitude?: number
  altitudeAccuracy?: number
  heading?: number
  speed?: number
}

export interface LocationOptions {
  enableHighAccuracy?: boolean
  timeout?: number
  maximumAge?: number
}

export function useLocation() {
  const currentLocation = ref<LocationCoords | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const supported = ref(false)

  // 检查浏览器支持
  const checkSupport = () => {
    supported.value = 'geolocation' in navigator
    return supported.value
  }

  /**
   * 获取当前位置
   */
  const getCurrentPosition = async (options: LocationOptions = {}): Promise<LocationCoords | null> => {
    if (!checkSupport()) {
      error.value = '浏览器不支持地理位置功能'
      return null
    }

    loading.value = true
    error.value = null

    const defaultOptions: PositionOptions = {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 300000, // 5分钟缓存
      ...options
    }

    try {
      const position = await new Promise<GeolocationPosition>((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, defaultOptions)
      })

      const coords: LocationCoords = {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        accuracy: position.coords.accuracy,
        altitude: position.coords.altitude || undefined,
        altitudeAccuracy: position.coords.altitudeAccuracy || undefined,
        heading: position.coords.heading || undefined,
        speed: position.coords.speed || undefined
      }

      currentLocation.value = coords
      return coords
    } catch (err: any) {
      const errorMessages = {
        1: '用户拒绝了位置访问请求',
        2: '位置信息不可用',
        3: '获取位置信息超时'
      }
      
      error.value = errorMessages[err.code as keyof typeof errorMessages] || '获取位置失败'
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * 监听位置变化
   */
  const watchPosition = (
    callback: (coords: LocationCoords) => void,
    options: LocationOptions = {}
  ) => {
    if (!checkSupport()) {
      error.value = '浏览器不支持地理位置功能'
      return null
    }

    const defaultOptions: PositionOptions = {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 60000, // 1分钟缓存
      ...options
    }

    const watchId = navigator.geolocation.watchPosition(
      (position) => {
        const coords: LocationCoords = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy,
          altitude: position.coords.altitude || undefined,
          altitudeAccuracy: position.coords.altitudeAccuracy || undefined,
          heading: position.coords.heading || undefined,
          speed: position.coords.speed || undefined
        }

        currentLocation.value = coords
        callback(coords)
      },
      (err) => {
        const errorMessages = {
          1: '用户拒绝了位置访问请求',
          2: '位置信息不可用',
          3: '获取位置信息超时'
        }
        
        error.value = errorMessages[err.code as keyof typeof errorMessages] || '获取位置失败'
      },
      defaultOptions
    )

    return watchId
  }

  /**
   * 停止监听位置变化
   */
  const clearWatch = (watchId: number) => {
    if (supported.value) {
      navigator.geolocation.clearWatch(watchId)
    }
  }

  /**
   * 计算两点间距离（米）
   */
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

  /**
   * 格式化距离显示
   */
  const formatDistance = (distance: number): string => {
    if (distance < 1000) {
      return `${Math.round(distance)}m`
    } else if (distance < 10000) {
      return `${(distance / 1000).toFixed(1)}km`
    } else {
      return `${Math.round(distance / 1000)}km`
    }
  }

  /**
   * 检查是否在指定范围内
   */
  const isWithinRange = (
    targetLat: number,
    targetLon: number,
    range: number,
    currentCoords?: LocationCoords
  ): boolean => {
    const coords = currentCoords || currentLocation.value
    if (!coords) return false

    const distance = calculateDistance(
      coords.latitude,
      coords.longitude,
      targetLat,
      targetLon
    )

    return distance <= range
  }

  /**
   * 获取位置名称（逆地理编码）
   */
  const getLocationName = async (
    latitude: number,
    longitude: number
  ): Promise<string | null> => {
    try {
      // 这里可以集成第三方地理编码服务
      // 例如高德地图、百度地图等
      const response = await fetch(
        `https://api.example.com/geocode?lat=${latitude}&lng=${longitude}`
      )
      
      if (response.ok) {
        const data = await response.json()
        return data.address || null
      }
      
      return null
    } catch (error) {
      console.error('获取位置名称失败:', error)
      return null
    }
  }

  // 初始化 - 避免在非组件上下文中使用 onMounted
  if (typeof window !== 'undefined') {
    checkSupport()
  }

  return {
    // 状态
    currentLocation: readonly(currentLocation),
    loading: readonly(loading),
    error: readonly(error),
    supported: readonly(supported),

    // 方法
    getCurrentPosition,
    watchPosition,
    clearWatch,
    calculateDistance,
    formatDistance,
    isWithinRange,
    getLocationName
  }
}
