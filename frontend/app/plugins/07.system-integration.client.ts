/**
 * 系统集成插件 - Composition API 重构版
 * - PWA 功能集成（安装提示、更新管理、离线缓存）
 * - 原生分享和剪贴板集成
 * - 系统通知和标记集成
 * - 文件系统访问和拖拽支持
 * - 设备 API 集成（摄像头、麦克风、传感器）
 */

interface SystemIntegration {
  pwa: {
    installable: boolean
    installed: boolean
    updateAvailable: boolean
  }
  
  sharing: {
    webShare: boolean
    clipboard: boolean
  }
  
  notifications: {
    system: boolean
    badges: boolean
  }
  
  fileSystem: {
    access: boolean
    dragDrop: boolean
  }
  
  media: {
    camera: boolean
    microphone: boolean
    screen: boolean
  }
  
  sensors: {
    geolocation: boolean
    deviceMotion: boolean
    ambient: boolean
  }
}

// 使用 Composition API 的系统集成 composable
const useSystemIntegration = () => {
  const integration = ref<SystemIntegration>()
  let installPrompt: any = null
  let updateWorker: ServiceWorker | null = null

  /**
   * 检测系统集成能力
   */
  const detectSystemIntegration = (): SystemIntegration => {
    const caps = globalThis.$deviceCapabilities

    return {
      pwa: {
        installable: false, // 动态检测
        installed: window.matchMedia('(display-mode: standalone)').matches,
        updateAvailable: false // 动态检测
      },
      
      sharing: {
        webShare: 'share' in navigator,
        clipboard: 'clipboard' in navigator
      },
      
      notifications: {
        system: 'Notification' in window,
        badges: 'setAppBadge' in navigator
      },
      
      fileSystem: {
        access: 'showOpenFilePicker' in window,
        dragDrop: 'ondragover' in window
      },
      
      media: {
        camera: !!(navigator.mediaDevices?.getUserMedia),
        microphone: !!(navigator.mediaDevices?.getUserMedia),
        screen: !!(navigator.mediaDevices?.getDisplayMedia)
      },
      
      sensors: {
        geolocation: caps?.geolocation || false,
        deviceMotion: 'DeviceMotionEvent' in window,
        ambient: 'AmbientLightSensor' in window
      }
    }
  }

  /**
   * 设置 PWA 功能
   */
  const setupPWAFeatures = (): void => {
    // 监听安装提示
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault()
      installPrompt = e
      if (integration.value) {
        integration.value.pwa.installable = true
      }
    })

    // 监听应用安装
    window.addEventListener('appinstalled', () => {
      installPrompt = null
      if (integration.value) {
        integration.value.pwa.installable = false
        integration.value.pwa.installed = true
      }
    })

    // 监听 Service Worker 更新
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        if (integration.value) {
          integration.value.pwa.updateAvailable = true
        }
      })

      navigator.serviceWorker.ready.then((registration) => {
        updateWorker = registration.active
      })
    }
  }

  /**
   * 设置分享功能
   */
  const setupSharingFeatures = (): void => {
    if (!integration.value?.sharing.webShare) return

    // 提供全局分享方法
    ;(globalThis as any).$share = async (data: ShareData) => {
      try {
        if (navigator.share) {
          await navigator.share(data)
          return true
        }
        return false
      } catch (error) {
        console.warn('分享失败:', error)
        return false
      }
    }

    // 设置剪贴板功能
    if (integration.value?.sharing.clipboard) {
      ;(globalThis as any).$clipboard = {
        write: async (text: string) => {
          try {
            await navigator.clipboard.writeText(text)
            return true
          } catch (error) {
            console.warn('复制到剪贴板失败:', error)
            return false
          }
        },
        read: async () => {
          try {
            return await navigator.clipboard.readText()
          } catch (error) {
            console.warn('读取剪贴板失败:', error)
            return null
          }
        }
      }
    }
  }

  /**
   * 设置通知功能
   */
  const setupNotificationFeatures = (): void => {
    if (!integration.value?.notifications.system) return

    // 提供全局通知方法
    ;(globalThis as any).$notification = {
      show: async (title: string, options?: NotificationOptions) => {
        try {
          if (Notification.permission === 'granted') {
            return new Notification(title, options)
          } else if (Notification.permission !== 'denied') {
            const permission = await Notification.requestPermission()
            if (permission === 'granted') {
              return new Notification(title, options)
            }
          }
          return null
        } catch (error) {
          console.warn('显示通知失败:', error)
          return null
        }
      },
      badge: async (count: number) => {
        try {
          if (integration.value?.notifications.badges) {
            await (navigator as any).setAppBadge(count)
            return true
          }
          return false
        } catch (error) {
          console.warn('设置徽章失败:', error)
          return false
        }
      }
    }
  }

  /**
   * 设置文件系统功能
   */
  const setupFileSystemFeatures = (): void => {
    if (!integration.value?.fileSystem.access) return

    // 提供全局文件访问方法
    ;(globalThis as any).$fileSystem = {
      pick: async (options?: any) => {
        try {
          return await (window as any).showOpenFilePicker(options)
        } catch (error) {
          console.warn('打开文件失败:', error)
          return null
        }
      },
      save: async (fileHandle: any, data: any) => {
        try {
          const writable = await fileHandle.createWritable()
          await writable.write(data)
          await writable.close()
          return true
        } catch (error) {
          console.warn('保存文件失败:', error)
          return false
        }
      }
    }

    // 设置拖拽上传
    if (integration.value?.fileSystem.dragDrop) {
      setupDragAndDrop()
    }
  }

  /**
   * 设置拖拽上传
   */
  const setupDragAndDrop = (): void => {
    document.addEventListener('dragover', (e) => {
      e.preventDefault()
    })

    document.addEventListener('drop', (e) => {
      e.preventDefault()
      const files = Array.from(e.dataTransfer?.files || [])
      if (files.length > 0) {
        // 触发自定义事件
        const event = new CustomEvent('app:files-dropped', {
          detail: { files, position: { x: e.clientX, y: e.clientY } }
        })
        document.dispatchEvent(event)
      }
    })
  }

  /**
   * 设置媒体功能
   */
  const setupMediaFeatures = (): void => {
    if (!integration.value?.media.camera && !integration.value?.media.microphone) return

    // 提供全局媒体访问方法
    ;(globalThis as any).$media = {
      getUserMedia: async (constraints: MediaStreamConstraints) => {
        try {
          return await navigator.mediaDevices.getUserMedia(constraints)
        } catch (error) {
          console.warn('获取媒体设备失败:', error)
          return null
        }
      },
      getDisplayMedia: async (constraints?: MediaStreamConstraints) => {
        try {
          if (integration.value?.media.screen) {
            return await navigator.mediaDevices.getDisplayMedia(constraints as any)
          }
          return null
        } catch (error) {
          console.warn('获取屏幕共享失败:', error)
          return null
        }
      }
    }
  }

  /**
   * 安装应用
   */
  const installApp = async (): Promise<boolean> => {
    if (!installPrompt) return false

    try {
      const result = await installPrompt.prompt()
      const outcome = await result.userChoice
      
      if (outcome === 'accepted') {
        installPrompt = null
        return true
      }
      return false
    } catch (error) {
      console.warn('安装应用失败:', error)
      return false
    }
  }

  /**
   * 初始化系统集成
   */
  const initialize = (): void => {
    integration.value = detectSystemIntegration()
    setupPWAFeatures()
    setupSharingFeatures()
    setupNotificationFeatures()
    setupFileSystemFeatures()
    setupMediaFeatures()
  }

  return {
    integration: readonly(integration),
    initialize,
    installApp
  }
}

export default defineNuxtPlugin(async () => {
  const systemIntegration = useSystemIntegration()
  
  // 初始化系统集成
  systemIntegration.initialize()

  // 提供给全局使用
  ;(globalThis as any).$systemIntegration = systemIntegration

  if (import.meta.dev) {
    console.debug('✅ 系统集成插件初始化完成 (Composition API)')
  }
})