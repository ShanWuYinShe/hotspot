/**
 * 位置获取 Composable (Singleton State)
 * 
 * 提供响应式的位置获取功能，共享全局位置状态。
 * 避免多个组件重复初始化 AMap Geolocation 实例。
 */

import { geocoding } from '~/utils/geocoding'
import { DEFAULTS } from '~/config/defaults'
import { waitForAMap, loadAMapPlugin } from '~/plugins/01.amap.client'

export interface PositionResult {
  coords: {
    latitude: number
    longitude: number
    accuracy: number
  }
  timestamp: number
  source: 'amap' | 'default'
}

export interface GeolocationState {
  position: PositionResult | null
  coordinates: { latitude: number; longitude: number } | null
  locationName: string | null
  accuracy: number | null
  loading: boolean
  loadingName: boolean
  error: string | null
  source: 'amap' | 'default' | null
}

export interface GeolocationOptions {
  silent?: boolean
  fetchLocationName?: boolean
}

// ============ Global Singleton State ============

const state = reactive<GeolocationState>({
  position: null,
  coordinates: null,
  locationName: null,
  accuracy: null,
  loading: false,
  loadingName: false,
  error: null,
  source: null
})

let amapGeolocation: AMap.Geolocation | null = null
let watchId: number | null = null
type PositionUpdateHandler = (position: PositionResult) => void
const watchHandlers = new Set<PositionUpdateHandler>()

// ============ Private Helpers ============

const updateState = (position: PositionResult) => {
  state.position = position
  state.coordinates = {
    latitude: position.coords.latitude,
    longitude: position.coords.longitude
  }
  state.accuracy = position.coords.accuracy
  state.source = position.source
  state.error = null
}

const setDefaultPosition = (errorMsg: string, silent: boolean): PositionResult => {
  const defaultPosition: PositionResult = {
    coords: {
      latitude: DEFAULTS.latitude,
      longitude: DEFAULTS.longitude,
      accuracy: 0
    },
    timestamp: Date.now(),
    source: 'default'
  }
  
  updateState(defaultPosition)
  state.locationName = DEFAULTS.locationName // Reset location name to default
  state.error = errorMsg
  
  if (!silent) {
    const feedback = useFeedbackStore()
    feedback.warning('定位提示', `${errorMsg}，已使用默认位置`)
  }
  
  return defaultPosition
}


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
      extensions: 'all',
      GeoLocationFirst: true, 
    })
    
    return amapGeolocation
  } catch (error) {
    console.warn('[Geolocation] 高德地图定位初始化失败:', error)
    return null
  }
}

// ============ Exported Composable ============

export function useGeolocation() {
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
   */
  const getPosition = async (options?: GeolocationOptions): Promise<PositionResult> => {
    const { silent = false, fetchLocationName: shouldFetchName = true } = options ?? {}

    if (state.loading && state.position) return state.position

    state.loading = true
    state.error = null

    try {
      const geo = await initAMapGeolocation()
      if (!geo) return setDefaultPosition('定位服务初始化失败', silent)
      
      return new Promise((resolve) => {
        geo.getCurrentPosition((status, result) => {
          state.loading = false
          if (status === 'complete') {
            const position: PositionResult = {
              coords: {
                latitude: result.position.getLat(),
                longitude: result.position.getLng(),
                accuracy: result.accuracy
              },
              timestamp: Date.now(),
              source: 'amap'
            }
            
            updateState(position)
            
            // 统一使用逆地理编码获取友好的位置名称（包含POI信息）
            // 即使 shouldFetchName 为 false，也使用 getLocationName 以确保名称友好
            fetchLocationName(position.coords.latitude, position.coords.longitude)
            
            resolve(position)
          } else {
            console.warn('[Geolocation] 高德定位失败:', result.message)
            resolve(setDefaultPosition('无法获取精确位置', silent))
          }
        })
      })
    } catch (error) {
      state.loading = false
      console.error('[Geolocation] 定位异常:', error)
      return setDefaultPosition('定位服务异常', silent)
    }
  }

  /**
   * 开始监听位置变化
   * 使用引用计数，当最后一个监听者移除时才真正停止监听
   */
  const startWatching = async (onUpdate?: PositionUpdateHandler): Promise<boolean> => {
    if (onUpdate) watchHandlers.add(onUpdate)
    if (onUpdate && state.position) onUpdate(state.position)
    if (watchId !== null) return true

    const geo = await initAMapGeolocation()
    if (!geo) {
      setDefaultPosition('无法启动位置监听', false)
      return false
    }

    const onComplete = (result: AMap.GeolocationResult) => {
      const position: PositionResult = {
        coords: {
          latitude: result.position.getLat(),
          longitude: result.position.getLng(),
          accuracy: result.accuracy
        },
        timestamp: Date.now(),
        source: 'amap'
      }

      updateState(position)

      // 使用逆地理编码获取更友好的位置名称（包含POI信息）
      fetchLocationName(position.coords.latitude, position.coords.longitude)

      // ✅ 触发所有注册的回调（比如 useLocationSync）
      watchHandlers.forEach((fn) => {
        try {
          fn(position)
        } catch (e) {
          console.warn('[Geolocation] 监听回调执行失败:', e)
        }
      })
    }

    const onError = (result: unknown) => {
      console.warn('[Geolocation] 监听定位失败', result)
    }

    geo.on('complete', onComplete)
    geo.on('error', onError)
    
    // 保存引用以便清理 (简化：只保存 geo 实例引用清理)
    // 实际 AMap watchPosition 返回 watchId
    watchId = geo.watchPosition() as unknown as number
    
    return true
  }

  /**
   * 停止监听位置变化
   */
  const stopWatching = async (onUpdate?: PositionUpdateHandler) => {
    if (onUpdate) watchHandlers.delete(onUpdate)
    else watchHandlers.clear()

    if (watchHandlers.size > 0) return
    if (watchId !== null && amapGeolocation) {
      amapGeolocation.clearWatch(watchId)
      watchId = null
    }
  }

  return {
    // 状态
    state: readonly(state),
    
    // 便捷访问
    coordinates: computed(() => state.coordinates),
    locationName: computed(() => state.locationName),
    accuracy: computed(() => state.accuracy),
    loading: computed(() => state.loading),
    loadingName: computed(() => state.loadingName),
    error: computed(() => state.error),
    
    // 方法
    getPosition,
    getHighAccuracyPosition: getPosition,
    fetchLocationName,
    startWatching, // 注意：现在 startWatching 主要是为了开启全局定位流
    stopWatching,
    getCoordinates: async () => {
      if (state.coordinates) return state.coordinates
      const pos = await getPosition({ silent: true })
      return { latitude: pos.coords.latitude, longitude: pos.coords.longitude }
    },
    getCoordinatesSync: () => state.coordinates ?? { latitude: DEFAULTS.latitude, longitude: DEFAULTS.longitude },
    getFullLocation: async () => {
      if (!state.locationName) {
        await getPosition({ fetchLocationName: true })
      }
      return {
        latitude: state.coordinates?.latitude ?? DEFAULTS.latitude,
        longitude: state.coordinates?.longitude ?? DEFAULTS.longitude,
        locationName: state.locationName ?? DEFAULTS.locationName
      }
    }
  }
}
