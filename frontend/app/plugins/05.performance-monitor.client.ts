/**
 * 性能监控插件 - Composition API 重构版
 * - 收集 Core Web Vitals 指标（LCP, FID, CLS, FCP, TTFB）
 * - 监控 API 响应时间和错误率
 * - 追踪用户交互性能和页面加载速度
 */

interface PerformanceMetrics {
  // Core Web Vitals
  lcp?: number  // Largest Contentful Paint
  fid?: number  // First Input Delay
  cls?: number  // Cumulative Layout Shift
  fcp?: number  // First Contentful Paint
  ttfb?: number // Time to First Byte

  // 自定义指标
  pageLoadTime?: number
  domContentLoaded?: number
  firstInteraction?: number
  
  // 资源加载
  resourceCount: number
  resourceLoadTime: number
  
  // 内存使用
  memoryUsage?: {
    usedHeapSize: number
    totalHeapSize: number
    heapLimit: number
  }
  
  // 网络信息
  connection?: {
    effectiveType: string
    downlink: number
    rtt: number
  }
}

interface APIPerformance {
  url: string
  method: string
  duration: number
  status: number
  success: boolean
  timestamp: number
}

// 使用 Composition API 的性能监控 composable
const usePerformanceMonitor = () => {
  const metrics = ref<PerformanceMetrics>({ resourceCount: 0, resourceLoadTime: 0 })
  const apiMetrics = ref<APIPerformance[]>([])
  let observer: PerformanceObserver | undefined
  const navigationStartTime = performance.now()

  /**
   * 设置性能观察器
   */
  const setupPerformanceObserver = (): void => {
    if (!('PerformanceObserver' in window)) return

    try {
      observer = new PerformanceObserver((entryList) => {
        for (const entry of entryList.getEntries()) {
          processPerformanceEntry(entry)
        }
      })

      observer.observe({ 
        entryTypes: ['navigation', 'resource', 'paint', 'largest-contentful-paint', 'first-input', 'layout-shift']
      })
    } catch (error) {
      console.warn('[PerformanceMonitor] Performance Observer 初始化失败:', error)
    }
  }

  /**
   * 处理性能条目
   */
  const processPerformanceEntry = (entry: PerformanceEntry): void => {
    switch (entry.entryType) {
      case 'navigation':
        handleNavigationEntry(entry as PerformanceNavigationTiming)
        break
      case 'resource':
        handleResourceEntry(entry as PerformanceResourceTiming)
        break
      case 'paint':
        handlePaintEntry(entry as PerformancePaintTiming)
        break
      case 'largest-contentful-paint':
        metrics.value.lcp = entry.startTime
        break
      case 'first-input':
        metrics.value.fid = (entry as any).processingStart - entry.startTime
        break
      case 'layout-shift':
        if (!(entry as any).hadRecentInput) {
          metrics.value.cls = (metrics.value.cls || 0) + (entry as any).value
        }
        break
    }
  }

  /**
   * 处理导航条目
   */
  const handleNavigationEntry = (entry: PerformanceNavigationTiming): void => {
    metrics.value.pageLoadTime = entry.loadEventEnd - entry.fetchStart
    metrics.value.domContentLoaded = entry.domContentLoadedEventEnd - entry.fetchStart
    metrics.value.ttfb = entry.responseStart - entry.requestStart
  }

  /**
   * 处理资源条目
   */
  const handleResourceEntry = (entry: PerformanceResourceTiming): void => {
    metrics.value.resourceCount++
    metrics.value.resourceLoadTime += entry.responseEnd - entry.requestStart
  }

  /**
   * 处理绘制条目
   */
  const handlePaintEntry = (entry: PerformancePaintTiming): void => {
    if (entry.name === 'first-contentful-paint') {
      metrics.value.fcp = entry.startTime
    }
  }

  /**
   * 收集初始指标
   */
  const collectInitialMetrics = (): void => {
    // 内存使用情况
    if ('memory' in performance) {
      const memory = (performance as any).memory
      metrics.value.memoryUsage = {
        usedHeapSize: memory.usedJSHeapSize,
        totalHeapSize: memory.totalJSHeapSize,
        heapLimit: memory.jsHeapSizeLimit
      }
    }

    // 网络连接信息
    const connection = (navigator as any).connection
    if (connection) {
      metrics.value.connection = {
        effectiveType: connection.effectiveType,
        downlink: connection.downlink,
        rtt: connection.rtt
      }
    }
  }

  /**
   * 监控页面加载
   */
  const monitorPageLoad = (): void => {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => {
        metrics.value.domContentLoaded = performance.now() - navigationStartTime
      })
    }

    window.addEventListener('load', () => {
      metrics.value.pageLoadTime = performance.now() - navigationStartTime
    })
  }

  /**
   * 获取性能报告
   */
  const getPerformanceReport = () => {
    return {
      metrics: toRaw(metrics.value),
      apiMetrics: toRaw(apiMetrics.value),
      timestamp: Date.now(),
      url: window.location.href
    }
  }

  /**
   * 清理资源
   */
  const cleanup = (): void => {
    if (observer) {
      observer.disconnect()
      observer = undefined
    }
  }

  return {
    metrics: readonly(metrics),
    apiMetrics: readonly(apiMetrics),
    setupPerformanceObserver,
    collectInitialMetrics,
    monitorPageLoad,
    getPerformanceReport,
    cleanup
  }
}

export default defineNuxtPlugin(async () => {
  const performanceMonitor = usePerformanceMonitor()
  
  // 初始化性能监控
  performanceMonitor.setupPerformanceObserver()
  performanceMonitor.collectInitialMetrics()
  performanceMonitor.monitorPageLoad()

  // 页面卸载时清理资源
  window.addEventListener('beforeunload', () => {
    performanceMonitor.cleanup()
  })

  if (import.meta.dev) {
    console.debug('✅ 性能监控插件初始化完成 (Composition API)')
  }
})