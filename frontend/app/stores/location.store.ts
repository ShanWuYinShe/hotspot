/**
 * 位置状态存储 - 纯状态管理
 * 
 * ✅ 职责：存储和管理位置相关的状态
 * ❌ 不包含：位置获取逻辑（使用 useGeolocation composable）
 * 
 * 使用方式：
 * - 组件中使用 useGeolocation() 获取位置
 * - 获取后使用 locationStore.setLocation() 更新全局状态
 */

import { defineStore } from 'pinia'
import { DEFAULTS } from '~/config/defaults'
import type { BrowserPermissionStatus } from '~/utils/browser-permissions'

export interface StoredLocation {
  latitude: number
  longitude: number
  locationName: string
  accuracy?: number
  source?: 'gps' | 'network' | 'cache' | 'default'
  updatedAt: number
}

export const useLocationStore = defineStore('location', () => {
  // ============ 内部状态 ============
  const _location = ref<StoredLocation | null>(null)
  const _permissionStatus = ref<BrowserPermissionStatus>('unknown')
  const _initialized = ref(false)

  // ============ Getter ============

  /**
   * 当前位置
   */
  const location = computed(() => _location.value)

  /**
   * 坐标（快捷访问）
   */
  const coordinates = computed(() => {
    if (!_location.value) return null
    return {
      latitude: _location.value.latitude,
      longitude: _location.value.longitude
    }
  })

  /**
   * 位置名称
   */
  const locationName = computed(() => _location.value?.locationName ?? '')

  /**
   * 定位精度（米）
   */
  const accuracy = computed(() => _location.value?.accuracy ?? null)

  /**
   * 权限状态
   */
  const permissionStatus = computed(() => _permissionStatus.value)

  /**
   * 是否有权限
   */
  const hasPermission = computed(() => _permissionStatus.value === 'granted')

  /**
   * 是否已初始化
   */
  const isInitialized = computed(() => _initialized.value)

  /**
   * 是否使用的是默认位置
   */
  const isDefaultLocation = computed(() => {
    if (!_location.value) return true
    return _location.value.source === 'default'
  })

  /**
   * 同步获取坐标（有则返回，无则返回默认值）
   */
  const getCoordinatesSync = (): { latitude: number; longitude: number } => {
    return coordinates.value ?? { latitude: DEFAULTS.latitude, longitude: DEFAULTS.longitude }
  }

  /**
   * 同步获取完整位置信息
   */
  const getLocationSync = (): StoredLocation => {
    return _location.value ?? {
      latitude: DEFAULTS.latitude,
      longitude: DEFAULTS.longitude,
      locationName: DEFAULTS.locationName,
      source: 'default',
      updatedAt: Date.now()
    }
  }

  // ============ Setter ============

  /**
   * 设置位置
   */
  const setLocation = (
    latitude: number,
    longitude: number,
    locationName: string = '',
    options?: {
      accuracy?: number
      source?: StoredLocation['source']
    }
  ) => {
    _location.value = {
      latitude,
      longitude,
      locationName,
      accuracy: options?.accuracy,
      source: options?.source ?? 'network',
      updatedAt: Date.now()
    }
    _initialized.value = true
  }

  /**
   * 从 PositionResult 设置位置
   */
  const setFromPosition = (
    position: { coords: { latitude: number; longitude: number; accuracy: number }; source?: string },
    locationName?: string
  ) => {
    setLocation(
      position.coords.latitude,
      position.coords.longitude,
      locationName ?? '',
      {
        accuracy: position.coords.accuracy,
        source: (position.source as StoredLocation['source']) ?? 'network'
      }
    )
  }

  /**
   * 设置位置名称
   */
  const setLocationName = (name: string) => {
    if (_location.value) {
      _location.value.locationName = name
      _location.value.updatedAt = Date.now()
    }
  }

  /**
   * 设置权限状态
   */
  const setPermissionStatus = (status: BrowserPermissionStatus) => {
    _permissionStatus.value = status
  }

  /**
   * 使用默认位置初始化
   */
  const initWithDefault = () => {
    if (!_initialized.value) {
      _location.value = {
        latitude: DEFAULTS.latitude,
        longitude: DEFAULTS.longitude,
        locationName: DEFAULTS.locationName,
        source: 'default',
        updatedAt: Date.now()
      }
      _initialized.value = true
    }
  }

  /**
   * 清除位置
   */
  const clearLocation = () => {
    _location.value = null
    _initialized.value = false
  }

  return {
    // Getter
    location,
    coordinates,
    locationName,
    accuracy,
    permissionStatus,
    hasPermission,
    isInitialized,
    isDefaultLocation,
    getCoordinatesSync,
    getLocationSync,

    // Setter
    setLocation,
    setFromPosition,
    setLocationName,
    setPermissionStatus,
    initWithDefault,
    clearLocation
  }
}, {
  // 不需要持久化
  persist: false
})
