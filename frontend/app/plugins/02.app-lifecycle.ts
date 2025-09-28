/**
 * 应用生命周期管理插件 - 简化版本
 * 避免在插件中使用 Vue 响应式 API 和生命周期钩子
 */

interface LifecycleHandler {
  readonly name: string
  readonly priority: number
  onInit?(): Promise<void> | void
  onDestroy?(): Promise<void> | void
}

class LifecycleManager {
  private handlers: LifecycleHandler[] = []
  private initialized = false

  addHandler(handler: LifecycleHandler): void {
    this.handlers.push(handler)
    this.handlers.sort((a, b) => b.priority - a.priority)
  }

  async executePhase(phase: keyof LifecycleHandler): Promise<void> {
    const promises = this.handlers
      .filter(handler => typeof handler[phase] === 'function')
      .map(handler => {
        try {
          return (handler[phase] as Function)()
        } catch (error) {
          console.error(`Lifecycle handler ${handler.name} failed in ${phase}:`, error)
          return Promise.resolve()
        }
      })

    await Promise.allSettled(promises)
  }

  async initialize(): Promise<void> {
    if (this.initialized) return
    
    await this.executePhase('onInit')
    this.initialized = true
  }
}

// 认证状态处理器 - 简化版本
class AuthLifecycleHandler implements LifecycleHandler {
  readonly name = 'AuthLifecycleHandler'
  readonly priority = 100

  async onInit(): Promise<void> {
    const authStore = useAuthStore()
    
    // 尝试刷新认证状态
    if (authStore.refreshToken) {
      try {
        // 使用 useAuth composable 处理令牌刷新
        const { refreshToken } = useAuth()
        await refreshToken()
      } catch (error) {
        if (import.meta.dev) {
          console.warn('Token 刷新失败:', error)
        }
      }
    }
  }
}


// 位置服务处理器
class LocationLifecycleHandler implements LifecycleHandler {
  readonly name = 'LocationLifecycleHandler'
  readonly priority = 70

  async onInit(): Promise<void> {
    if (typeof window === 'undefined') return
    
    try {
      const location = useLocation()
      // 获取当前位置（不需要持续监听）
      await location.getCurrentPosition()
    } catch (error) {
      if (import.meta.dev) {
        console.warn('位置服务启动失败:', error)
      }
    }
  }
}

export default defineNuxtPlugin({
  name: 'app-lifecycle',
  setup(nuxtApp) {
    const lifecycleManager = new LifecycleManager()
    
    // 注册处理器
    lifecycleManager.addHandler(new AuthLifecycleHandler())
    lifecycleManager.addHandler(new LocationLifecycleHandler())
    
    // 初始化
    lifecycleManager.initialize()
    
    // 客户端特定的清理
    if (typeof window !== 'undefined') {
      // 页面卸载清理
      window.addEventListener('beforeunload', () => {
        lifecycleManager.executePhase('onDestroy')
      })
    }

    if (import.meta.dev) {
      console.debug('✅ 应用生命周期管理初始化完成')
    }
  }
})