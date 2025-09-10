/**
 * 性能监控 Composable
 * 监控关键性能指标和用户体验指标
 */

interface PerformanceMetrics {
  // Core Web Vitals
  LCP?: number  // Largest Contentful Paint
  FID?: number  // First Input Delay
  CLS?: number  // Cumulative Layout Shift
  
  // 自定义指标
  TTI?: number  // Time to Interactive
  FCP?: number  // First Contentful Paint
  
  // 应用特定指标
  apiResponseTime?: number
  websocketLatency?: number
  renderTime?: number
}

export const usePerformanceMonitor = () => {
  const metrics = ref<PerformanceMetrics>({})
  const isSupported = ref(false)
  
  // 检查浏览器支持
  onMounted(() => {
    isSupported.value = 'PerformanceObserver' in window
    if (isSupported.value) {
      initPerformanceObserver()
    }
  })
  
  // 初始化性能观察器
  const initPerformanceObserver = () => {
    // 监控 Core Web Vitals
    if ('PerformanceObserver' in window) {
      // LCP (Largest Contentful Paint)
      const lcpObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries()
        const lastEntry = entries[entries.length - 1]
        if (lastEntry) {
          metrics.value.LCP = lastEntry.startTime
        }
      })
      lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] })
      
      // FID (First Input Delay)
      const fidObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries()
        entries.forEach((entry: any) => {
          metrics.value.FID = entry.processingStart - entry.startTime
        })
      })
      fidObserver.observe({ entryTypes: ['first-input'] })
      
      // CLS (Cumulative Layout Shift)
      let clsValue = 0
      const clsObserver = new PerformanceObserver((entryList) => {
        const entries = entryList.getEntries()
        entries.forEach((entry: any) => {
          if (!entry.hadRecentInput) {
            clsValue += entry.value
            metrics.value.CLS = clsValue
          }
        })
      })
      clsObserver.observe({ entryTypes: ['layout-shift'] })
    }
  }
  
  // 监控API响应时间
  const trackApiCall = async <T>(
    apiCall: () => Promise<T>,
    endpoint: string
  ): Promise<T> => {
    const startTime = performance.now()
    
    try {
      const result = await apiCall()
      const endTime = performance.now()
      const responseTime = endTime - startTime
      
      metrics.value.apiResponseTime = responseTime
      
      // 发送到分析服务（仅在生产环境）
      if (import.meta.env.PROD) {
        reportMetric('api_response_time', {
          endpoint,
          responseTime,
          timestamp: Date.now()
        })
      }
      
      return result
    } catch (error) {
      const endTime = performance.now()
      const responseTime = endTime - startTime
      
      // 记录失败的API调用
      if (import.meta.env.PROD) {
        reportMetric('api_error', {
          endpoint,
          responseTime,
          error: error instanceof Error ? error.message : 'Unknown error',
          timestamp: Date.now()
        })
      }
      
      throw error
    }
  }
  
  // 监控WebSocket延迟
  const trackWebSocketLatency = (latency: number) => {
    metrics.value.websocketLatency = latency
    
    if (import.meta.env.PROD) {
      reportMetric('websocket_latency', {
        latency,
        timestamp: Date.now()
      })
    }
  }
  
  // 监控组件渲染时间
  const trackRenderTime = (componentName: string, renderTime: number) => {
    metrics.value.renderTime = renderTime
    
    if (import.meta.env.PROD) {
      reportMetric('component_render_time', {
        componentName,
        renderTime,
        timestamp: Date.now()
      })
    }
  }
  
  // 发送指标到分析服务
  const reportMetric = (metricName: string, data: any) => {
    // 这里可以集成 Google Analytics、Sentry 等分析服务
    if (import.meta.dev) {
      console.log(`📊 Performance Metric [${metricName}]:`, data)
    }
    
    // 示例：发送到自定义分析端点
    // fetch('/api/analytics/metrics', {
    //   method: 'POST',
    //   headers: { 'Content-Type': 'application/json' },
    //   body: JSON.stringify({ metric: metricName, data })
    // }).catch(() => {}) // 静默失败，不影响用户体验
  }
  
  // 获取页面加载性能
  const getPageLoadMetrics = () => {
    if (!('performance' in window)) return null
    
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    
    return {
      // 页面加载时间
      pageLoadTime: navigation.loadEventEnd - navigation.fetchStart,
      // DNS查询时间
      dnsTime: navigation.domainLookupEnd - navigation.domainLookupStart,
      // TCP连接时间
      tcpTime: navigation.connectEnd - navigation.connectStart,
      // 请求响应时间
      requestTime: navigation.responseEnd - navigation.requestStart,
      // DOM解析时间
      domParseTime: navigation.domContentLoadedEventEnd - ((navigation as any).domLoading || navigation.fetchStart),
      // 资源加载时间
      resourceLoadTime: navigation.loadEventEnd - navigation.domContentLoadedEventEnd
    }
  }
  
  return {
    metrics: readonly(metrics),
    isSupported: readonly(isSupported),
    trackApiCall,
    trackWebSocketLatency,
    trackRenderTime,
    getPageLoadMetrics,
    reportMetric
  }
}

