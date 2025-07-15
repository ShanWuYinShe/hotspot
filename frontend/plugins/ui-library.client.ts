// Naive UI 客户端配置
export default defineNuxtPlugin(() => {
  // 确保使用Naive UI
  const uiLibrary = 'naive-ui'
  
  // 设置全局UI库标识
  if (process.client) {
    ;(window as any).__UI_LIBRARY__ = uiLibrary
    
    // 确保 DOM API 可用
    if (typeof document !== 'undefined') {
      // 初始化客户端特定的 Naive UI 功能
      console.log('Naive UI 客户端初始化完成')
    }
  }
  
  return {
    provide: {
      uiLibrary
    }
  }
}) 