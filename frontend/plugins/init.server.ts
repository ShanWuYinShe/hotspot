/**
 * 服务端初始化插件
 * 处理服务端启动时的初始化逻辑
 */

import { useAuthStore } from '~/stores/auth'
import { useUserStore } from '~/stores/user'

export default defineNuxtPlugin(async (nuxtApp) => {
  // 获取存储的状态
  const authStore = useAuthStore()
  const userStore = useUserStore()
  
  console.log('Server plugin initialized')
  
  // 这里可以添加服务端专用的初始化代码
  
  return {
    provide: {
      // 提供给应用程序的服务端工具
      serverInitialized: true,
    }
  }
})
