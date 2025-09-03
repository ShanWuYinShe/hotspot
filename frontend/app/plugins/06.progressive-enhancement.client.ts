/**
 * 渐进式增强插件 - Composition API 重构版
 * - 基于设备能力动态启用功能，确保最佳用户体验
 * - 自动降级不支持的功能，提供备选方案
 * - 智能预加载和懒加载，优化性能
 */

interface EnhancementConfig {
  // 功能开关
  features: {
    animations: boolean
    hapticFeedback: boolean
    offlineMode: boolean
    backgroundSync: boolean
    pushNotifications: boolean
    advancedGraphics: boolean
    voiceInteraction: boolean
    biometricAuth: boolean
  }
  
  // 性能优化配置
  optimization: {
    imageQuality: 'low' | 'medium' | 'high'
    preloadStrategy: 'aggressive' | 'conservative' | 'disabled'
    lazyLoadThreshold: number
    networkAwareLoading: boolean
  }
  
  // 用户体验增强
  ux: {
    responsiveImages: boolean
    adaptiveUI: boolean
    smartCaching: boolean
    prefetchContent: boolean
  }
}

// 使用 Composition API 的渐进式增强 composable
const useProgressiveEnhancement = () => {
  const config = ref<EnhancementConfig>()
  const performanceProfile = ref<'high' | 'medium' | 'low'>('medium')
  
  /**
   * 检测性能配置文件
   */
  const detectPerformanceProfile = (): 'high' | 'medium' | 'low' => {
    const deviceCapabilities = globalThis.$deviceCapabilities
    
    // 检查硬件指标
    const hardwareScore = 
      (deviceCapabilities?.device?.cores >= 4 ? 2 : 1) +
      (deviceCapabilities?.device?.ram >= 4 ? 2 : 1) +
      (deviceCapabilities?.device?.pixelRatio >= 2 ? 1 : 0)
    
    // 检查网络条件
    const networkScore = 
      deviceCapabilities?.connection?.downlink >= 10 ? 2 :
      deviceCapabilities?.connection?.downlink >= 2 ? 1 : 0
    
    const totalScore = hardwareScore + networkScore
    
    if (totalScore >= 5) return 'high'
    if (totalScore >= 3) return 'medium'
    return 'low'
  }

  /**
   * 基于设备能力生成最优配置
   */
  const generateOptimalConfig = (): EnhancementConfig => {
    const caps = globalThis.$deviceCapabilities
    const isHighPerformanceDevice = performanceProfile.value === 'high'
    const hasGoodNetwork = caps?.connection?.downlink > 2 || caps?.connection?.effectiveType === '4g'
    const isMobile = caps?.isTouchDevice

    return {
      features: {
        animations: !caps?.device?.prefersReducedMotion && isHighPerformanceDevice,
        hapticFeedback: 'vibrate' in navigator && isMobile,
        offlineMode: caps?.serviceWorker && 'caches' in window,
        backgroundSync: caps?.serviceWorker && 'serviceWorker' in navigator,
        pushNotifications: caps?.notification && caps?.webPush,
        advancedGraphics: caps?.device?.pixelRatio >= 2 && isHighPerformanceDevice,
        voiceInteraction: 'webkitSpeechRecognition' in window || 'SpeechRecognition' in window,
        biometricAuth: caps?.supportsBiometrics && window.isSecureContext
      },
      
      optimization: {
        imageQuality: isHighPerformanceDevice && hasGoodNetwork ? 'high' : 
                     hasGoodNetwork ? 'medium' : 'low',
        preloadStrategy: hasGoodNetwork && isHighPerformanceDevice ? 'aggressive' : 
                        hasGoodNetwork ? 'conservative' : 'disabled',
        lazyLoadThreshold: isMobile ? 50 : 200,
        networkAwareLoading: !!caps?.connection
      },
      
      ux: {
        responsiveImages: true,
        adaptiveUI: true,
        smartCaching: caps?.serviceWorker || false,
        prefetchContent: hasGoodNetwork && isHighPerformanceDevice
      }
    }
  }

  /**
   * 应用渐进式增强
   */
  const applyEnhancements = (): void => {
    if (!config.value) return

    // 应用动画增强
    if (config.value.features.animations) {
      document.documentElement.classList.add('enable-animations')
    } else {
      document.documentElement.classList.add('reduce-motion')
    }

    // 应用图像质量设置
    document.documentElement.setAttribute('data-image-quality', config.value.optimization.imageQuality)
    
    // 应用自适应UI
    if (config.value.ux.adaptiveUI) {
      document.documentElement.classList.add('adaptive-ui')
    }

    // 应用预加载策略
    if (config.value.optimization.preloadStrategy !== 'disabled') {
      setupIntelligentPreloading()
    }

    // 应用懒加载
    if (config.value.optimization.lazyLoadThreshold > 0) {
      setupLazyLoading()
    }
  }

  /**
   * 设置智能预加载
   */
  const setupIntelligentPreloading = (): void => {
    if (!config.value) return

    const strategy = config.value.optimization.preloadStrategy
    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const element = entry.target as HTMLElement
          const href = element.getAttribute('data-preload')
          if (href) {
            preloadResource(href, strategy)
            observer.unobserve(element)
          }
        }
      })
    }, {
      rootMargin: strategy === 'aggressive' ? '100px' : '20px'
    })

    // 观察所有需要预加载的元素
    document.querySelectorAll('[data-preload]').forEach(el => {
      observer.observe(el)
    })
  }

  /**
   * 预加载资源
   */
  const preloadResource = (href: string, strategy: string): void => {
    const link = document.createElement('link')
    link.rel = strategy === 'aggressive' ? 'prefetch' : 'dns-prefetch'
    link.href = href
    document.head.appendChild(link)
  }

  /**
   * 设置懒加载
   */
  const setupLazyLoading = (): void => {
    if (!config.value) return

    const threshold = config.value.optimization.lazyLoadThreshold
    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const img = entry.target as HTMLImageElement
          if (img.dataset.src) {
            img.src = img.dataset.src
            img.removeAttribute('data-src')
            observer.unobserve(img)
          }
        }
      })
    }, {
      rootMargin: `${threshold}px`
    })

    // 观察所有懒加载图像
    document.querySelectorAll('img[data-src]').forEach(img => {
      observer.observe(img)
    })
  }

  /**
   * 设置网络感知加载
   */
  const setupNetworkAwareLoading = (): void => {
    if (!config.value?.optimization.networkAwareLoading) return

    const connection = (navigator as any).connection
    if (!connection) return

    // 监听网络变化
    connection.addEventListener('change', () => {
      const newConfig = generateOptimalConfig()
      config.value = newConfig
      applyEnhancements()
    })
  }

  /**
   * 初始化渐进式增强
   */
  const initialize = (): void => {
    performanceProfile.value = detectPerformanceProfile()
    config.value = generateOptimalConfig()
    applyEnhancements()
    setupNetworkAwareLoading()
  }

  return {
    config: readonly(config),
    performanceProfile: readonly(performanceProfile),
    initialize,
    generateOptimalConfig,
    applyEnhancements
  }
}

export default defineNuxtPlugin(async () => {
  const progressiveEnhancement = useProgressiveEnhancement()
  
  // 初始化渐进式增强
  progressiveEnhancement.initialize()

  // 监听页面切换，重新应用增强
  const router = useRouter()
  router.afterEach(() => {
    nextTick(() => {
      progressiveEnhancement.applyEnhancements()
    })
  })

  if (import.meta.dev) {
    console.debug('✅ 渐进式增强插件初始化完成 (Composition API)')
  }
})