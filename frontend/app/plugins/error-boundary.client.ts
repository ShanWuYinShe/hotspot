/**
 * 全局错误边界插件
 * 使用 Nuxt 3 内置错误处理
 */

export default defineNuxtPlugin(() => {
  const nuxtApp = useNuxtApp()
  
  // 错误处理函数
  const handleError = (error: Error, context?: Record<string, any>) => {
    console.error('Global error caught:', error, context)
    
    // 在开发环境显示详细错误
    if (process.dev) {
      console.error('Error details:', {
        message: error.message,
        stack: error.stack,
        context
      })
    }
    
    // 使用 Nuxt 的错误处理
    throw createError({
      statusCode: 500,
      statusMessage: 'Internal Server Error',
      message: process.dev ? error.message : 'Something went wrong'
    })
  }

  // 设置全局错误处理
  if (process.client) {
    // JavaScript 错误
    window.addEventListener('error', (event) => {
      const error = event.error || new Error(event.message)
      handleError(error, {
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
    nuxtApp.hook('vue:error', (error, instance, info) => {
      const errorObj = error instanceof Error ? error : new Error(String(error))
      handleError(errorObj, {
        type: 'vue',
        component: instance?.$options.name || 'Unknown',
        info
      })
    })
  }
})
