/**
 * 位置获取 Composable
 * 
 * 提供响应式的位置获取功能，含：
 * - 优先使用高德地图定位（更准确）
 * - 回退到浏览器原生定位
 * - 自动逆地理编码（获取位置名称）
 * - 用户友好的错误提示
 */

import {
  checkBrowserPermission,
  watchPosition,
  clearPositionWatch,
  type BrowserPermissionStatus,
  type PositionResult
} from '~/utils/browser-permissions'
import { geocoding } from '~/utils/geocoding'
import { DEFAULTS } from '~/config/defaults'
import { waitForAMap, loadAMapPlugin } from '~/plugins/amap.client'

export interface GeolocationState {
  /** 原始位置数据 */
  position: PositionResult | null
  /** 坐标 */
  coordinates: { latitude: number; longitude: number } | null
  /** 位置名称（逆地理编码结果） */
  locationName: string | null
  /** 定位精度（米） */
  accuracy: number | null
  /** 权限状态 */
  permissionStatus: BrowserPermissionStatus
  /** 是否正在加载 */
  loading: boolean
  /** 是否正在获取位置名称 */
  loadingName: boolean
  /** 错误信息 */
  error: string | null
  /** 定位来源 */
  source: 'amap' | 'browser' | null
}

export interface GeolocationOptions {
  /** 是否静默模式（不显示错误提示） */
  silent?: boolean
  /** 是否自动获取位置名称 */
  fetchLocationName?: boolean
}

export function useGeolocation() {
  const toast = useToast()

  // 响应式状态
  const state = reactive<GeolocationState>({
    position: null,
    coordinates: null,
    locationName: null,
    accuracy: null,
    permissionStatus: 'unknown',
    loading: false,
    loadingName: false,
    error: null,
    source: null
  })

  // 监听 ID
  let watchId: number | null = null
  
  // 高德地图定位实例（延迟初始化）
  let amapGeolocation: AMap.Geolocation | null = null

  /**
   * 检查权限状态
   */
  const checkPermission = async (): Promise<BrowserPermissionStatus> => {
    state.permissionStatus = await checkBrowserPermission('geolocation')
    return state.permissionStatus
  }

  /**
   * 更新内部状态
   */
  const updateState = (position: PositionResult, source: 'amap' | 'browser' = 'browser') => {
    state.position = position
    state.coordinates = {
      latitude: position.coords.latitude,
      longitude: position.coords.longitude
    }
    state.accuracy = position.coords.accuracy
    state.permissionStatus = 'granted'
    state.source = source
  }
  
  /**
   * 初始化高德地图定位
   */
  const initAMapGeolocation = async (): Promise<AMap.Geolocation | null> => {
    if (amapGeolocation) return amapGeolocation
    
    try {
      await waitForAMap()
      await loadAMapPlugin('AMap.Geolocation')
      
      amapGeolocation = new AMap.Geolocation({
        enableHighAccuracy: true,
        timeout: 10000,
        convert: true,
        showButton: false,
        showMarker: false,
        showCircle: false,
        extensions: 'all'
      })
      
      return amapGeolocation
    } catch (error) {
      console.warn('[Geolocation] 高德地图定位初始化失败，将回退到浏览器定位:', error)
      return null
    }
  }
  
  /**
   * 使用高德地图定位
   */
  const getAMapPosition = async (): Promise<PositionResult | null> => {
    const geo = await initAMapGeolocation()
    if (!geo) return null
    
    return new Promise((resolve) => {
      geo.getCurrentPosition((status, result) => {
        if (status === 'complete') {
          const position: PositionResult = {
            coords: {
              latitude: result.position.getLat(),
              longitude: result.position.getLng(),
              accuracy: result.accuracy
            },
            timestamp: Date.now(),
            source: result.location_type === 'html5' ? 'gps' : 'network'
          }
          
          // 如果有地址信息，直接更新
          if (result.formattedAddress) {
            state.locationName = formatAMapAddress(result)
          }
          
          console.log('[Geolocation] 高德定位成功:', {
            type: result.location_type,
            accuracy: result.accuracy,
            address: result.formattedAddress
          })
          
          resolve(position)
        } else {
          console.warn('[Geolocation] 高德定位失败:', result.message)
          resolve(null)
        }
      })
    })
  }
  
  /**
   * 格式化高德地图地址
   */
  const formatAMapAddress = (result: AMap.GeolocationResult): string => {
    const addr = result.addressComponent
    if (addr) {
      if (addr.district && addr.street) {
        return `${addr.district}${addr.street}`
      }
      if (addr.district && addr.township) {
        return `${addr.district}${addr.township}`
      }
      if (addr.district) {
        return addr.district
      }
      if (addr.city) {
        return addr.city
      }
    }
    return result.formattedAddress || '未知位置'
  }

  /**
   * 获取位置名称（逆地理编码）
   */
  const fetchLocationName = async (latitude: number, longitude: number): Promise<string> => {
    state.loadingName = true
    try {
      const name = await geocoding.getLocationName(latitude, longitude)
      state.locationName = name
      return name
    } catch (error) {
      console.warn('[Geolocation] 获取位置名称失败:', error)
      state.locationName = '未知位置'
      return '未知位置'
    } finally {
      state.loadingName = false
    }
  }

  /**
   * 获取当前位置
   * 
   * 定位策略：
   * 1. 仅使用高德地图定位
   * 2. 高德定位失败时使用默认位置并提示用户
   */
  const getPosition = async (options?: GeolocationOptions): Promise<PositionResult | null> => {
    const {
      silent = false,
      fetchLocationName: shouldFetchName = true
    } = options ?? {}

    if (state.loading) return null

    state.loading = true
    state.error = null

    try {
      // 尝试高德地图定位
      const position = await getAMapPosition()
      
      if (position) {
        updateState(position, 'amap')
        // 高德定位已经获取了地址信息，不需要再次获取
        if (!state.locationName && shouldFetchName) {
          fetchLocationName(position.coords.latitude, position.coords.longitude)
        }
        return position
      }

      // 高德定位失败，使用默认位置
      console.warn('[Geolocation] 高德定位失败，使用默认位置')
      
      const defaultPosition: PositionResult = {
        coords: {
          latitude: DEFAULTS.latitude,
          longitude: DEFAULTS.longitude,
          accuracy: 0
        },
        timestamp: Date.now(),
        source: 'default'
      }
      
      state.position = defaultPosition
      state.coordinates = {
        latitude: DEFAULTS.latitude,
        longitude: DEFAULTS.longitude
      }
      state.locationName = DEFAULTS.locationName
      state.accuracy = 0
      state.source = null
      state.error = '无法获取精确位置，已使用默认位置'
      
      if (!silent) {
        toast.add({
          title: '定位失败',
          description: `无法获取您的位置，已使用默认位置：${DEFAULTS.locationName}`,
          color: 'warning',
          duration: 5000
        })
      }
      
      return defaultPosition
    } catch (error) {
      console.error('[Geolocation] 定位异常:', error)
      
      // 异常时也使用默认位置
      const defaultPosition: PositionResult = {
        coords: {
          latitude: DEFAULTS.latitude,
          longitude: DEFAULTS.longitude,
          accuracy: 0
        },
        timestamp: Date.now(),
        source: 'default'
      }
      
      state.position = defaultPosition
      state.coordinates = {
        latitude: DEFAULTS.latitude,
        longitude: DEFAULTS.longitude
      }
      state.locationName = DEFAULTS.locationName
      state.accuracy = 0
      state.source = null
      state.error = '定位服务异常，已使用默认位置'
      
      if (!silent) {
        toast.add({
          title: '定位服务异常',
          description: `已使用默认位置：${DEFAULTS.locationName}`,
          color: 'warning',
          duration: 5000
        })
      }
      
      return defaultPosition
    } finally {
      state.loading = false
    }
  }

  /**
   * 获取位置（高精度模式，现在与普通模式相同）
   */
  const getHighAccuracyPosition = async (options?: Omit<GeolocationOptions, 'highAccuracy'>) => {
    return getPosition(options)
  }

  /**
   * 开始监听位置变化
   */
  const startWatching = (
    onUpdate?: (position: PositionResult) => void,
    options?: { highAccuracy?: boolean }
  ): boolean => {
    if (watchId !== null) return true

    const id = watchPosition(
      (position) => {
        updateState(position)
        onUpdate?.(position)
      },
      (error) => {
        state.error = error.userMessage
        if (error.type === 'permission_denied') {
          state.permissionStatus = 'denied'
        }
      },
      { highAccuracy: options?.highAccuracy ?? false }
    )

    if (id === null) {
      toast.add({
        title: '位置服务不可用',
        description: '您的浏览器不支持位置服务',
        color: 'error'
      })
      return false
    }

    watchId = id
    return true
  }

  /**
   * 停止监听位置变化
   */
  const stopWatching = () => {
    if (watchId !== null) {
      clearPositionWatch(watchId)
      watchId = null
    }
  }

  /**
   * 获取坐标（有则返回，无则获取）
   */
  const getCoordinates = async (): Promise<{ latitude: number; longitude: number }> => {
    if (state.coordinates) return state.coordinates

    const position = await getPosition({ silent: true })
    if (position) {
      return {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude
      }
    }

    // 返回默认位置
    return { latitude: DEFAULTS.latitude, longitude: DEFAULTS.longitude }
  }

  /**
   * 同步获取坐标（有则返回，无则返回默认值）
   */
  const getCoordinatesSync = (): { latitude: number; longitude: number } => {
    return state.coordinates ?? { latitude: DEFAULTS.latitude, longitude: DEFAULTS.longitude }
  }

  /**
   * 获取完整位置信息（坐标 + 名称）
   */
  const getFullLocation = async (): Promise<{
    latitude: number
    longitude: number
    locationName: string
  }> => {
    const position = await getPosition({ fetchLocationName: true })
    
    if (position) {
      // 等待位置名称获取完成（如果还在加载）
      if (state.loadingName) {
        await new Promise<void>(resolve => {
          const unwatch = watch(() => state.loadingName, (loading) => {
            if (!loading) {
              unwatch()
              resolve()
            }
          })
        })
      }
      
      return {
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        locationName: state.locationName || DEFAULTS.locationName
      }
    }

    // 返回默认位置
    return {
      latitude: DEFAULTS.latitude,
      longitude: DEFAULTS.longitude,
      locationName: DEFAULTS.locationName
    }
  }

  // 组件卸载时清理
  onUnmounted(() => {
    stopWatching()
  })

  return {
    // 状态
    state: readonly(state),
    
    // 便捷访问
    coordinates: computed(() => state.coordinates),
    locationName: computed(() => state.locationName),
    accuracy: computed(() => state.accuracy),
    permissionStatus: computed(() => state.permissionStatus),
    loading: computed(() => state.loading),
    loadingName: computed(() => state.loadingName),
    error: computed(() => state.error),
    hasPermission: computed(() => state.permissionStatus === 'granted'),

    // 方法
    checkPermission,
    getPosition,
    getHighAccuracyPosition,
    fetchLocationName,
    startWatching,
    stopWatching,
    getCoordinates,
    getCoordinatesSync,
    getFullLocation
  }
}
