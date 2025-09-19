/**
 * 全局错误边界插件
 * 集成高级错误处理和恢复机制
 */

export default defineNuxtPlugin(() => {
  const { handleError } = useErrorBoundary({
    enableReporting: true,
    reportingEndpoint: '/api/v1/errors',
    recoveryStrategy: 'retry',
    showDetailInDev: true
  })

  // 设置全局错误处理
  if (process.client) {
    // JavaScript 错误
    window.addEventListener('error', (event) => {
      handleError(event.error || new Error(event.message), {
        type: 'javascript',
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno
      })
    })

    // Promise 拒绝
    window.addEventListener('unhandledrejection', (event) => {
      const error = event.reason instanceof Error 
        ? event.reason 
        : new Error(String(event.reason))
      
      handleError(error, {
        type: 'promise-rejection'
      })
    })

    // Vue 错误处理
    const nuxtApp = useNuxtApp()
    nuxtApp.hook('vue:error', (error, instance, info) => {
      handleError(error, {
        type: 'vue',
        component: instance?.$options.name || 'Unknown',
        info
      })
    })
  }
})
