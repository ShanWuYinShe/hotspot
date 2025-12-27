/**
 * 位置同步 Composable
 * 
 * 自动同步用户位置到服务器：
 * - 登录后自动获取并上传位置
 * - 定时更新位置（默认每5分钟）
 * - 位置变化时自动上传（仅做时间节流 + 坐标去重；严禁在客户端计算距离）
 */

import { useUsersService } from '~/services/users.service'
import type { PositionResult } from '~/composables/useGeolocation'
import { ensureRefreshToken } from '~/utils/token'

export interface LocationSyncOptions {
  /** 同步间隔（毫秒），默认 5 分钟 */
  syncInterval?: number
  /** 是否启用位置监听，默认 false（节省电量） */
  enableWatch?: boolean
  /** 是否在初始化时立即获取位置，默认 true */
  immediateSync?: boolean
}

const DEFAULT_OPTIONS: Required<LocationSyncOptions> = {
  syncInterval: 5 * 60 * 1000, // 5 分钟
  enableWatch: false, // 默认关闭持续监听，节省电量
  immediateSync: true
}

const normalizeCoord = (n: number) => Number(n.toFixed(6))

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
  const pendingUpload = ref<Promise<boolean> | null>(null)
  const lastAuthFailAt = ref<number>(0)
  
  // 定时器控制
  const { pause, resume, isActive } = useIntervalFn(() => {
    if (authStore.hasSession) {
      syncLocation()
    }
  }, opts.syncInterval, { immediate: false })
  
  /**
   * 上传位置到服务器
   */
  const uploadLocation = async (
    latitude: number,
    longitude: number,
    locationName?: string
  ): Promise<boolean> => {
    // 完全未登录：禁止触发任何 protected 调用/刷新
    if (!authStore.hasSession) return false
    // access_token 不可用时，不要直接打 protected API；先尝试刷新一次（失败则短暂降噪）
    if (!authStore.isAccessTokenValid) {
      const now = Date.now()
      if (lastAuthFailAt.value > 0 && now - lastAuthFailAt.value < 30_000) return false
      const ok = await ensureRefreshToken()
      if (!ok) {
        lastAuthFailAt.value = now
        return false
      }
    }
    if (syncing.value) return false
    if (pendingUpload.value) return pendingUpload.value
    
    pendingUpload.value = (async () => {
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
        pendingUpload.value = null
      }
    })()

    return pendingUpload.value
  }
  
  /**
   * 获取当前位置并上传
   */
  const syncLocation = async (): Promise<boolean> => {
    if (!authStore.hasSession) return false
    
    // 使用快速定位模式
    const position = await geolocation.getPosition({ silent: true })
    if (!position) {
      // 定位失败，使用默认位置初始化 store（不上传）
      locationStore.initWithDefault()
      return false
    }
    
    const { latitude, longitude } = position.coords

    // 只更新本地状态（不计算距离）
    locationStore.setFromPosition(position, geolocation.locationName.value ?? undefined)

    const now = Date.now()
    const isTooFrequent = lastSyncTime.value > 0 && now - lastSyncTime.value < opts.syncInterval

    const normalized = { latitude: normalizeCoord(latitude), longitude: normalizeCoord(longitude) }
    const last = lastSyncedCoords.value
      ? {
          latitude: normalizeCoord(lastSyncedCoords.value.latitude),
          longitude: normalizeCoord(lastSyncedCoords.value.longitude)
        }
      : null
    const isSameCoords = !!last && normalized.latitude === last.latitude && normalized.longitude === last.longitude

    if (isTooFrequent || isSameCoords) return true
    
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
    if (!authStore.hasSession) return false
    
    // 使用快速定位，但显示错误提示
    const position = await geolocation.getPosition({ silent: false })
    if (!position) return false

    // 更新本地 store（不计算距离）
    locationStore.setFromPosition(position, geolocation.locationName.value ?? undefined)
    
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
    if (isActive.value) return
    
    resume()
    
    console.log(`[LocationSync] 定时同步已启动 (间隔 ${opts.syncInterval / 1000}秒)`)
  }
  
  /**
   * 停止定时同步
   */
  const stopPeriodicSync = () => {
    if (isActive.value) {
      pause()
      console.log('[LocationSync] 定时同步已停止')
    }
  }
  
  /**
   * 位置变化回调（用于 watchPosition）
   */
  const handlePositionChange = (position: PositionResult) => {
    if (!authStore.hasSession) return
    if (!authStore.isAccessTokenValid) return
    
    const { latitude, longitude } = position.coords
    
    // 更新本地 store
    locationStore.setFromPosition(position, geolocation.locationName.value ?? undefined)

    // 仅做时间节流 + 坐标去重（严禁在客户端计算距离）
    const now = Date.now()
    const isTooFrequent = lastSyncTime.value > 0 && now - lastSyncTime.value < opts.syncInterval

    const normalized = { latitude: normalizeCoord(latitude), longitude: normalizeCoord(longitude) }
    const last = lastSyncedCoords.value
      ? {
          latitude: normalizeCoord(lastSyncedCoords.value.latitude),
          longitude: normalizeCoord(lastSyncedCoords.value.longitude)
        }
      : null
    const isSameCoords = !!last && normalized.latitude === last.latitude && normalized.longitude === last.longitude

    if (isTooFrequent || isSameCoords) return
    uploadLocation(latitude, longitude, geolocation.locationName.value ?? undefined)
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
    geolocation.stopWatching(handlePositionChange)
    lastSyncedCoords.value = null
    lastSyncTime.value = 0
    syncCount.value = 0
    console.log('[LocationSync] 资源已清理')
  }
  
  // 监听登录状态变化
  watch(
    () => [authStore.isAuthenticated, authStore.isAccessTokenValid] as const,
    async ([isAuthenticated, isAccessTokenValid]) => {
      if (isAuthenticated && isAccessTokenValid) {
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
