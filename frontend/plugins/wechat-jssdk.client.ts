/**
 * 微信JSSDK插件
 * 在客户端自动加载微信JSSDK并初始化
 */
import wx from 'weixin-js-sdk'

export default defineNuxtPlugin((nuxtApp) => {
  // 只在客户端运行
  if (typeof window === 'undefined') return

  // 将wx对象挂载到window
  window.wx = wx

  // 获取运行时配置
  const config = useRuntimeConfig()
  
  // 在Vue应用挂载后初始化
  nuxtApp.hook('app:mounted', () => {
    // 检查是否在微信环境
    const isWechat = /MicroMessenger/i.test(navigator.userAgent)
    
    // 保存环境标志到localStorage，便于组件使用
    localStorage.setItem('isWechatBrowser', isWechat ? 'true' : 'false')
    
    if (isWechat) {
      console.log('检测到微信环境，准备初始化JSSDK')
      
      // 这里不主动初始化，由具体使用组件在需要时调用wechatService.initJssdk()
      // 因为每个页面的URL不同，需要获取对应的签名
    }
  })
  
  // 提供一个函数来检查是否在微信环境
  const isWechatBrowser = () => {
    if (typeof window === 'undefined') return false
    return /MicroMessenger/i.test(navigator.userAgent)
  }
  
  // 返回插件对象
  return {
    provide: {
      isWechatBrowser
    }
  }
})
