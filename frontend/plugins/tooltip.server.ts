/**
 * 提示系统服务端插件 - SSR 兼容性处理
 */
export default defineNuxtPlugin((nuxtApp) => {
  // 在服务端注册空指令，避免 SSR 错误
  nuxtApp.vueApp.directive('tooltip', {
    getSSRProps() {
      return {}
    }
  })
})
