/**
 * 统一的格式化 Composable
 * 支持国际化，严禁在 Vue 文件中实现格式化逻辑
 * 这个 composable 可以在 Vue 组件中使用，正确访问 i18n 实例
 */

import { createI18nFormatters } from '~/utils/formatters'

export const useFormatters = () => {
  const { t, locale } = useI18n()

  return createI18nFormatters(t, locale)
}
