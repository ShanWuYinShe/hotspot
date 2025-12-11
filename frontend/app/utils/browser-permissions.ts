/**
 * 浏览器权限管理工具
 * 
 * 纯工具函数，不依赖 Vue/Nuxt 响应式系统
 */

export type BrowserPermissionType = 'geolocation' | 'notifications' | 'camera' | 'microphone'
export type BrowserPermissionStatus = 'granted' | 'denied' | 'prompt' | 'unknown'

/**
 * 检查浏览器权限状态
 */
export async function checkBrowserPermission(type: BrowserPermissionType): Promise<BrowserPermissionStatus> {
  // 仅客户端可用
  if (typeof window === 'undefined' || !('permissions' in navigator)) {
    return 'unknown'
  }

  try {
    const result = await navigator.permissions.query({ name: type as PermissionName })
    return result.state as BrowserPermissionStatus
  } catch {
    return 'unknown'
  }
}

/**
 * 获取权限状态描述
 */
export function getPermissionStatusLabel(status: BrowserPermissionStatus): string {
  const labels: Record<BrowserPermissionStatus, string> = {
    granted: '已授权',
    denied: '已拒绝',
    prompt: '待询问',
    unknown: '未知'
  }
  return labels[status]
}

/**
 * 获取权限状态颜色
 */
export function getPermissionStatusColor(status: BrowserPermissionStatus): string {
  const colors: Record<BrowserPermissionStatus, string> = {
    granted: 'success',
    denied: 'error',
    prompt: 'warning',
    unknown: 'neutral'
  }
  return colors[status]
}

// ============================================================
// 位置获取 - 分级策略
// ============================================================

export interface PositionResult {
  coords: {
    latitude: number
    longitude: number
    accuracy: number
  }
  timestamp: number
  source: 'gps' | 'network' | 'cache'
}

export interface GetPositionOptions {
  /** 是否需要高精度（默认 false，快速获取） */
  highAccuracy?: boolean
  /** 超时时间（毫秒），默认快速模式 5 秒，高精度 20 秒 */
  timeout?: number
  /** 可接受的缓存时间（毫秒），默认 5 分钟 */
  maximumAge?: number
}

/**
 * 获取当前位置 - 快速模式
 * 
 * 优化策略：
 * 1. 先尝试使用缓存（5分钟内有效）
 * 2. 使用网络定位（比 GPS 快很多）
 * 3. 合理的超时设置
 */
export function getCurrentPosition(options?: GetPositionOptions): Promise<PositionResult> {
  const {
    highAccuracy = false,
    timeout = highAccuracy ? 20000 : 5000,
    maximumAge = 5 * 60 * 1000 // 5分钟缓存
  } = options ?? {}

  return new Promise((resolve, reject) => {
    if (typeof window === 'undefined' || !('geolocation' in navigator)) {
      reject(new GeolocationError(2, 'Geolocation not supported'))
      return
    }

    navigator.geolocation.getCurrentPosition(
      (position) => {
        resolve({
          coords: {
            latitude: position.coords.latitude,
            longitude: position.coords.longitude,
            accuracy: position.coords.accuracy
          },
          timestamp: position.timestamp,
          source: position.coords.accuracy < 100 ? 'gps' : 'network'
        })
      },
      (error) => {
        reject(new GeolocationError(error.code, error.message))
      },
      {
        enableHighAccuracy: highAccuracy,
        timeout,
        maximumAge
      }
    )
  })
}

/**
 * 智能位置获取 - 分级尝试
 * 
 * 策略：
 * 1. 先快速获取（网络定位，使用缓存）
 * 2. 如果快速获取成功且精度足够，直接返回
 * 3. 如果需要高精度，再尝试 GPS 定位
 */
export async function getPositionSmart(options?: {
  /** 是否需要高精度，默认 false */
  requireHighAccuracy?: boolean
  /** 可接受的精度阈值（米），超过此值才尝试高精度，默认 500米 */
  accuracyThreshold?: number
}): Promise<PositionResult> {
  const {
    requireHighAccuracy = false,
    accuracyThreshold = 500
  } = options ?? {}

  // 第一步：快速获取
  try {
    const fastResult = await getCurrentPosition({ highAccuracy: false })
    
    // 精度足够或不需要高精度，直接返回
    if (!requireHighAccuracy || fastResult.coords.accuracy <= accuracyThreshold) {
      return fastResult
    }
    
    // 需要高精度，继续尝试
    console.log(`[Geolocation] 快速定位精度 ${fastResult.coords.accuracy}m，尝试高精度定位...`)
  } catch (error) {
    // 快速获取失败，如果是权限问题直接抛出
    if (error instanceof GeolocationError && error.code === 1) {
      throw error
    }
    console.warn('[Geolocation] 快速定位失败，尝试高精度定位...')
  }

  // 第二步：高精度获取
  return getCurrentPosition({ highAccuracy: true })
}

/**
 * 监听位置变化（原始 API 包装）
 */
export function watchPosition(
  onSuccess: (position: PositionResult) => void,
  onError?: (error: GeolocationError) => void,
  options?: GetPositionOptions
): number | null {
  if (typeof window === 'undefined' || !('geolocation' in navigator)) {
    return null
  }

  const {
    highAccuracy = false,
    timeout = 10000,
    maximumAge = 60000
  } = options ?? {}

  return navigator.geolocation.watchPosition(
    (position) => {
      onSuccess({
        coords: {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          accuracy: position.coords.accuracy
        },
        timestamp: position.timestamp,
        source: position.coords.accuracy < 100 ? 'gps' : 'network'
      })
    },
    (error) => {
      onError?.(new GeolocationError(error.code, error.message))
    },
    {
      enableHighAccuracy: highAccuracy,
      timeout,
      maximumAge
    }
  )
}

/**
 * 停止监听位置
 */
export function clearPositionWatch(watchId: number): void {
  if (typeof window !== 'undefined' && 'geolocation' in navigator) {
    navigator.geolocation.clearWatch(watchId)
  }
}

// ============================================================
// 错误处理
// ============================================================

/**
 * 自定义地理位置错误类
 */
export class GeolocationError extends Error {
  readonly code: number
  readonly type: 'permission_denied' | 'position_unavailable' | 'timeout' | 'unknown'

  constructor(code: number, message: string) {
    super(message)
    this.name = 'GeolocationError'
    this.code = code
    this.type = this.getErrorType(code)
  }

  private getErrorType(code: number): GeolocationError['type'] {
    switch (code) {
      case 1: return 'permission_denied'
      case 2: return 'position_unavailable'
      case 3: return 'timeout'
      default: return 'unknown'
    }
  }

  get userMessage(): string {
    switch (this.type) {
      case 'permission_denied':
        return '位置权限被拒绝，请在浏览器设置中允许访问位置'
      case 'position_unavailable':
        return '无法获取位置信息，请检查设备定位服务是否开启'
      case 'timeout':
        return '获取位置超时，请稍后重试'
      default:
        return '获取位置失败'
    }
  }
}

/**
 * 解析位置错误（兼容旧代码）
 */
export function parseGeolocationError(error: GeolocationPositionError | GeolocationError): {
  code: number
  message: string
  type: 'permission_denied' | 'position_unavailable' | 'timeout' | 'unknown'
} {
  if (error instanceof GeolocationError) {
    return { code: error.code, message: error.userMessage, type: error.type }
  }

  switch (error.code) {
    case error.PERMISSION_DENIED:
      return { code: error.code, message: '位置权限被拒绝', type: 'permission_denied' }
    case error.POSITION_UNAVAILABLE:
      return { code: error.code, message: '位置信息不可用', type: 'position_unavailable' }
    case error.TIMEOUT:
      return { code: error.code, message: '获取位置超时', type: 'timeout' }
    default:
      return { code: error.code, message: '获取位置失败', type: 'unknown' }
  }
}

// ============================================================
// 通知权限
// ============================================================

/**
 * 检查通知权限并请求
 */
export async function requestNotificationPermission(): Promise<'granted' | 'denied' | 'default'> {
  if (typeof window === 'undefined' || !('Notification' in window)) {
    return 'denied'
  }

  if (Notification.permission === 'granted') {
    return 'granted'
  }

  return await Notification.requestPermission()
}
