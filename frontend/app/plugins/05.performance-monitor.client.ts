/**
 * 性能监控插件
 * 使用统一的 usePerformanceMonitor composable
 */

export default defineNuxtPlugin(async () => {
  const performanceMonitor = usePerformanceMonitor()
  
  // 初始化性能监控
  if (performanceMonitor.isSupported) {
    // 获取页面加载指标
    performanceMonitor.getPageLoadMetrics()
    
    if (import.meta.dev) {
      console.debug('✅ 性能监控插件初始化完成', performanceMonitor.metrics)
    }
  }
})