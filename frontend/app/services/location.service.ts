/**
 * 位置跟踪服务
 * 负责自动更新用户位置到服务器
 */


import { useUserService } from './users.service'

class LocationTracker {
  private updateTimer: NodeJS.Timeout | null = null
  private isTracking = false
  private readonly UPDATE_INTERVAL = 5 * 60 * 1000 // 5分钟更新一次

  /**
   * 开始位置跟踪
   */
  async start() {
    if (this.isTracking) {
      return
    }

    this.isTracking = true
    
    // 立即更新一次位置
    await this.updateLocationToServer()
    
    // 设置定时更新
    this.updateTimer = setInterval(async () => {
      await this.updateLocationToServer()
    }, this.UPDATE_INTERVAL)

    if (import.meta.dev) {
      console.log('位置跟踪已启动')
    }
  }

  /**
   * 停止位置跟踪
   */
  stop() {
    if (!this.isTracking) {
      return
    }

    this.isTracking = false
    
    if (this.updateTimer) {
      clearInterval(this.updateTimer)
      this.updateTimer = null
    }

    if (import.meta.dev) {
      console.log('位置跟踪已停止')
    }
  }

  /**
   * 更新位置到服务器
   */
  private async updateLocationToServer() {
    try {
      const authStore = useAuthStore()
      
      // 检查用户是否已认证
      if (!authStore.isAuthenticated) {
        return
      }

      const locationStore = useLocationStore()
      const location = await getCurrentLocation()
      
      if (!location) {
        return
      }

      const userService = useUserService()
      await userService.updateLocation(
        location.latitude,
        location.longitude,
        '当前位置'
      )

      if (import.meta.dev) {
        console.log('位置更新成功:', location)
      }
    } catch (error) {
      // 位置更新失败不影响其他功能，只记录日志
      console.warn('位置更新失败:', error)
    }
  }

  /**
   * 手动更新位置
   */
  async updateNow(): Promise<Location | null> {
    const locationStore = useLocationStore()
    const location = await getCurrentLocation()
    
    if (location) {
      const authStore = useAuthStore()
      
      if (authStore.isAuthenticated) {
        try {
          const userService = useUserService()
          await userService.updateLocation(
            location.latitude,
            location.longitude,
            '当前位置'
          )
        } catch (error) {
          console.warn('手动位置更新失败:', error)
        }
      }
    }
    
    return location
  }

  /**
   * 是否正在跟踪
   */
  get tracking() {
    return this.isTracking
  }
}

// 单例实例
const locationTracker = new LocationTracker()

export const useLocationTracker = () => {
  return {
    start: () => locationTracker.start(),
    stop: () => locationTracker.stop(),
    updateNow: () => locationTracker.updateNow(),
    get tracking() {
      return locationTracker.tracking
    }
  }
}

