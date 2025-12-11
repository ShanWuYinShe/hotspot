/**
 * 位置同步 Composable
 * 
 * 自动同步用户位置到服务器：
 * - 登录后自动获取并上传位置
 * - 定时更新位置（默认每5分钟）
 * - 位置变化时自动上传（距离超过阈值）
 */

import { useUsersService } from '~/services/users.service'
import { DEFAULTS } from '~/config/defaults'
import type { PositionResult } from '~/utils/browser-permissions'

export interface LocationSyncOptions {
  /** 同步间隔（毫秒），默认 5 分钟 */
  syncInterval?: number
  /** 位置变化阈值（米），超过此距离才上传，默认 100 米 */
  distanceThreshold?: number
  /** 是否启用位置监听，默认 false（节省电量） */
  enableWatch?: boolean
  /** 是否在初始化时立即获取位置，默认 true */
  immediateSync?: boolean
}

const DEFAULT_OPTIONS: Required<LocationSyncOptions> = {
  syncInterval: 5 * 60 * 1000, // 5 分钟
  distanceThreshold: 100, // 100 米
  enableWatch: false, // 默认关闭持续监听，节省电量
  immediateSync: true
}

/**
 * 计算两点之间的距离（米）
 * 使用 Haversine 公式
 */
function calculateDistance(
  lat1: number, lon1: number,
  lat2: number, lon2: number
): number {
  const R = 6371000 // 地球半径（米）
  const dLat = (lat2 - lat1) * Math.PI / 180
  const dLon = (lon2 - lon1) * Math.PI / 180
  const a = 
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) *
    Math.sin(dLon / 2) * Math.sin(dLon / 2)
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
  return R * c
}

export function useLocationSync(options: LocationSyncOptions = {}) {
  const opts = { ...DEFAULT_OPTIONS, ...options }
  
  const authStore = useAuthStore()
  const locationStore = useLocationStore()
  const geolocation = useGeolocation()
  const userService = useUsersService()
  
  // 状态
  const syncing = ref(false)
  const lastSyncTime = ref<number>(0)
  const lastSyncedCoords = ref<{ latitude: number; longitude: number } | null>(null)
  const syncError = ref<string | null>(null)
  const syncCount = ref(0)
  
  // 定时器 ID
  let syncIntervalId: ReturnType<typeof setInterval> | null = null
  
  /**
   * 上传位置到服务器
   */
  const uploadLocation = async (
    latitude: number,
    longitude: number,
    locationName?: string
  ): Promise<boolean> => {
    if (!authStore.isAuthenticated) return false
    if (syncing.value) return false
    
    syncing.value = true
    syncError.value = null
    
    try {
      await userService.updateLocation({
        latitude,
        longitude,
        location_name: locationName
      })
      
      lastSyncTime.value = Date.now()
      lastSyncedCoords.value = { latitude, longitude }
      syncCount.value++
      
      // 同步更新 locationStore
      locationStore.setLocation(latitude, longitude, locationName ?? '', {
        source: 'network'
      })
      
      console.log(`[LocationSync] 位置已同步 (${syncCount.value}次)`)
      return true
    } catch (error) {
      console.error('[LocationSync] 位置同步失败:', error)
      syncError.value = '位置同步失败'
      return false
    } finally {
      syncing.value = false
    }
  }
  
  /**
   * 获取当前位置并上传
   */
  const syncLocation = async (): Promise<boolean> => {
    if (!authStore.isAuthenticated) return false
    
    // 使用快速定位模式
    const position = await geolocation.getPosition({ silent: true })
    if (!position) {
      // 定位失败，使用默认位置初始化 store（不上传）
      locationStore.initWithDefault()
      return false
    }
    
    const { latitude, longitude } = position.coords
    
    // 检查是否需要上传（距离超过阈值）
    if (lastSyncedCoords.value) {
      const distance = calculateDistance(
        lastSyncedCoords.value.latitude,
        lastSyncedCoords.value.longitude,
        latitude,
        longitude
      )
      
      // 距离未超过阈值，只更新本地状态
      if (distance < opts.distanceThreshold) {
        locationStore.setFromPosition(position, geolocation.locationName.value ?? undefined)
        return true
      }
    }
    
    // 获取位置名称（如果还没有）
    let locationName = geolocation.locationName.value
    if (!locationName) {
      locationName = await geolocation.fetchLocationName(latitude, longitude)
    }
    
    return uploadLocation(latitude, longitude, locationName)
  }
  
  /**
   * 强制同步位置（忽略阈值检查）
   */
  const forceSyncLocation = async (): Promise<boolean> => {
    if (!authStore.isAuthenticated) return false
    
    // 使用快速定位，但显示错误提示
    const position = await geolocation.getPosition({ silent: false })
    if (!position) return false
    
    // 获取位置名称
    const locationName = await geolocation.fetchLocationName(
      position.coords.latitude,
      position.coords.longitude
    )
    
    return uploadLocation(
      position.coords.latitude,
      position.coords.longitude,
      locationName
    )
  }
  
  /**
   * 开始定时同步
   */
  const startPeriodicSync = () => {
    if (syncIntervalId) return
    
    syncIntervalId = setInterval(() => {
      if (authStore.isAuthenticated) {
        syncLocation()
      }
    }, opts.syncInterval)
    
    console.log(`[LocationSync] 定时同步已启动 (间隔 ${opts.syncInterval / 1000}秒)`)
  }
  
  /**
   * 停止定时同步
   */
  const stopPeriodicSync = () => {
    if (syncIntervalId) {
      clearInterval(syncIntervalId)
      syncIntervalId = null
      console.log('[LocationSync] 定时同步已停止')
    }
  }
  
  /**
   * 位置变化回调（用于 watchPosition）
   */
  const handlePositionChange = (position: PositionResult) => {
    if (!authStore.isAuthenticated) return
    
    const { latitude, longitude } = position.coords
    
    // 更新本地 store
    locationStore.setFromPosition(position, geolocation.locationName.value ?? undefined)
    
    // 检查是否需要上传
    if (lastSyncedCoords.value) {
      const distance = calculateDistance(
        lastSyncedCoords.value.latitude,
        lastSyncedCoords.value.longitude,
        latitude,
        longitude
      )
      
      if (distance >= opts.distanceThreshold) {
        uploadLocation(latitude, longitude, geolocation.locationName.value ?? undefined)
      }
    } else {
      // 首次位置，直接上传
      uploadLocation(latitude, longitude, geolocation.locationName.value ?? undefined)
    }
  }
  
  /**
   * 初始化位置同步
   * 应在用户登录后调用
   */
  const initialize = async () => {
    if (!authStore.isAuthenticated) return
    
    console.log('[LocationSync] 初始化位置同步...')
    
    // 1. 先用默认位置初始化 store（确保立即可用）
    locationStore.initWithDefault()
    
    // 2. 异步获取并同步真实位置
    if (opts.immediateSync) {
      syncLocation()
    }
    
    // 3. 启动定时同步
    startPeriodicSync()
    
    // 4. 启用位置监听（如果配置允许）
    if (opts.enableWatch) {
      geolocation.startWatching(handlePositionChange)
    }
  }
  
  /**
   * 清理资源
   */
  const cleanup = () => {
    stopPeriodicSync()
    geolocation.stopWatching()
    lastSyncedCoords.value = null
    lastSyncTime.value = 0
    syncCount.value = 0
    console.log('[LocationSync] 资源已清理')
  }
  
  // 监听登录状态变化
  watch(
    () => authStore.isAuthenticated,
    async (isAuthenticated) => {
      if (isAuthenticated) {
        // 用户登录，初始化位置同步
        await initialize()
      } else {
        // 用户登出，清理资源
        cleanup()
      }
    },
    { immediate: true }
  )
  
  // 组件卸载时清理
  onUnmounted(() => {
    cleanup()
  })
  
  return {
    // 状态
    syncing: readonly(syncing),
    lastSyncTime: readonly(lastSyncTime),
    syncError: readonly(syncError),
    syncCount: readonly(syncCount),
    
    // 方法
    syncLocation,
    forceSyncLocation,
    initialize,
    cleanup
  }
}
