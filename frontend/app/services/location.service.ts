/**
 * 位置跟踪服务
 * ✅ 重构为纯 Composable 模式，避免在类方法中调用 Vue composables
 */

import { useUserService } from './users.service'

/**
 * ✅ 使用 Composable 模式而非类模式
 * 所有的 store/service 调用都在 composable 函数内部，确保在正确的 Vue 上下文中执行
 */
export const useLocationTracker = () => {
  // ✅ 在 composable 顶层声明所有需要的 stores 和 services
  const authStore = useAuthStore()
  const locationStore = useLocationStore()
  const userService = useUserService()
  
  // 使用 ref 管理状态
  const updateTimer = ref<NodeJS.Timeout | null>(null)
  const isTracking = ref(false)
  const UPDATE_INTERVAL = 5 * 60 * 1000 // 5分钟更新一次

  /**
   * 更新位置到服务器
   */
  const updateLocationToServer = async () => {
    try {
      // 检查用户是否已认证
      if (!authStore.isAuthenticated) {
        return
      }

      // 获取当前位置
      const position = await locationStore.getCurrentPosition()
      
      if (!position) {
        return
      }

      await userService.updateLocation({
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
        location_name: '当前位置'
      })

      if (import.meta.dev) {
        console.log('位置更新成功:', position.coords)
      }
    } catch (error) {
      // 位置更新失败不影响其他功能，只记录日志
      console.warn('位置更新失败:', error)
    }
  }

  /**
   * 开始位置跟踪
   */
  const start = async () => {
    if (isTracking.value) {
      return
    }

    isTracking.value = true
    
    // 立即更新一次位置
    await updateLocationToServer()
    
    // 设置定时更新
    updateTimer.value = setInterval(async () => {
      await updateLocationToServer()
    }, UPDATE_INTERVAL)

    if (import.meta.dev) {
      console.log('位置跟踪已启动')
    }
  }

  /**
   * 停止位置跟踪
   */
  const stop = () => {
    if (!isTracking.value) {
      return
    }

    isTracking.value = false
    
    if (updateTimer.value) {
      clearInterval(updateTimer.value)
      updateTimer.value = null
    }

    if (import.meta.dev) {
      console.log('位置跟踪已停止')
    }
  }

  /**
   * 手动更新位置
   */
  const updateNow = async (): Promise<GeolocationPosition | null> => {
    try {
      const position = await locationStore.getCurrentPosition()
      
      if (position && authStore.isAuthenticated) {
        await userService.updateLocation({
          latitude: position.coords.latitude,
          longitude: position.coords.longitude,
          location_name: '当前位置'
        })
      }
      
      return position
    } catch (error) {
      console.warn('手动位置更新失败:', error)
      return null
    }
  }

  return {
    start,
    stop,
    updateNow,
    tracking: computed(() => isTracking.value)
  }
}

