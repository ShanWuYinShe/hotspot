// 统一权限管理（仅客户端）
// 目的：
// 1) 提供标准化的权限状态检查接口
// 2) 提供标准化的权限请求接口（需在用户手势中调用）
// 3) 统一返回值，避免在页面层重复实现

export type PermissionState = 'granted' | 'denied' | 'prompt' | 'unsupported'
export type PermissionName = 'geolocation' | 'notification'

const queryGeolocationPermission = async (): Promise<PermissionState> => {
  if (typeof navigator === 'undefined') return 'unsupported'
  try {
    if ('permissions' in navigator) {
      const status = await navigator.permissions.query({ name: 'geolocation' as any })
      return status.state as PermissionState
    }
    return 'prompt'
  } catch {
    return 'prompt'
  }
}

const getNotificationPermission = (): PermissionState => {
  if (typeof window === 'undefined') return 'unsupported'
  if (!('Notification' in window)) return 'unsupported'
  const perm = Notification.permission as 'granted' | 'denied' | 'default'
  return perm === 'default' ? 'prompt' : perm
}

export const usePermissions = () => {
  // 检查权限
  const check = async (name: PermissionName): Promise<PermissionState> => {
    switch (name) {
      case 'geolocation':
        return queryGeolocationPermission()
      case 'notification':
        return getNotificationPermission()
      default:
        return 'unsupported'
    }
  }

  // 请求权限（需在用户手势中调用）
  const request = async (name: PermissionName): Promise<boolean> => {
    switch (name) {
      case 'geolocation': {
        if (typeof navigator === 'undefined' || !('geolocation' in navigator)) return false
        try {
          await new Promise<GeolocationPosition>((resolve, reject) => {
            navigator.geolocation.getCurrentPosition(resolve, reject, {
              enableHighAccuracy: true,
              timeout: 15000,
              maximumAge: 60000,
            })
          })
          return true
        } catch {
          return false
        }
      }
      case 'notification': {
        if (typeof window === 'undefined' || !('Notification' in window)) return false
        try {
          const perm = await Notification.requestPermission()
          return perm === 'granted'
        } catch {
          return false
        }
      }
      default:
        return false
    }
  }

  // 便捷方法
  const checkGeolocation = () => check('geolocation')
  const requestGeolocation = () => request('geolocation')
  const checkNotification = () => check('notification')
  const checkNotificationSync = (): PermissionState => getNotificationPermission()
  const requestNotification = () => request('notification')

  // 通用：监听权限变化（支持 geolocation/notification）。返回停止函数。
  const onPermissionChange = async (
    name: PermissionName,
    cb: (state: PermissionState) => void
  ): Promise<() => void> => {
    // geolocation: Permissions API 查询 name: 'geolocation'
    // notification: Permissions API 多数实现用 name: 'notifications'（复数）
    try {
      if (typeof navigator === 'undefined' || !('permissions' in navigator)) {
        // 回调一次当前状态，返回空清理函数
        cb(await check(name))
        return () => {}
      }
      const queryName = name === 'notification' ? 'notifications' : name
      const status: PermissionStatus = await navigator.permissions.query({ name: queryName as any })
      const handler = () => cb(status.state as PermissionState)
      status.addEventListener?.('change', handler)
      cb(status.state as PermissionState)
      return () => status.removeEventListener?.('change', handler)
    } catch {
      // 不支持该权限或浏览器限制：回调一次当前状态
      cb(await check(name))
      return () => {}
    }
  }

  // 专用封装
  const onGeolocationChange = (cb: (state: PermissionState) => void) => onPermissionChange('geolocation', cb)
  const onNotificationChange = (cb: (state: PermissionState) => void) => onPermissionChange('notification', cb)

  return {
    check,
    request,
    checkGeolocation,
    requestGeolocation,
    checkNotification,
    checkNotificationSync,
    requestNotification,
    onPermissionChange,
    onGeolocationChange,
    onNotificationChange,
  }
}


