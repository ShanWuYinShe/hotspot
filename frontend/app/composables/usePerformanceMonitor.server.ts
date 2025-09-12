/**
 * 性能监控 Composable (服务端版本)
 * 服务端不支持性能监控，提供占位实现
 */

import { ref, readonly } from 'vue'

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
  
  // 服务端占位实现
  const trackApiCall = async <T>(
    apiCall: () => Promise<T>,
    endpoint: string
  ): Promise<T> => {
    return await apiCall()
  }
  
  const trackWebSocketLatency = (latency: number) => {
    // 服务端不执行
  }
  
  const trackRenderTime = (componentName: string, renderTime: number) => {
    // 服务端不执行
  }
  
  const getPageLoadMetrics = () => {
    return null
  }
  
  const reportMetric = (metricName: string, data: any) => {
    // 服务端不执行
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
