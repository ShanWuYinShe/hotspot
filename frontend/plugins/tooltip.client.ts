/**
 * 提示系统插件 - 注册全局指令和组件
 * 仅在客户端执行
 */

import { useTooltip } from '~/utils/tooltip'

export default defineNuxtPlugin((nuxtApp) => {
  const { tooltipDirective } = useTooltip()
  
  // 注册全局指令（覆盖服务端的空指令）
  nuxtApp.vueApp.directive('tooltip', tooltipDirective)
  
  // 提供全局实例
  nuxtApp.vueApp.config.globalProperties.$tooltip = useTooltip()
})
