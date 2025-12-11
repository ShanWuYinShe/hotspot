/**
 * 统一的格式化 Composable
 * 支持国际化，严禁在 Vue 文件中实现格式化逻辑
 * 这个 composable 可以在 Vue 组件中使用，正确访问 i18n 实例
 */

// 翻译参数类型
type TranslateParams = Record<string, string | number>

export const useFormatters = () => {
  const { $i18n } = useNuxtApp()
  const { t: i18nT } = useI18n()

  // 类型安全的翻译函数
  const t = (key: string, params?: TranslateParams): string => {
    // 使用 useI18n 的 t 函数
    const translated = i18nT(key, params ?? {})
    return typeof translated === 'string' ? translated : String(translated)
  }

  /**
   * 格式化时间为相对时间显示（国际化版本）
   * @param dateString ISO 8601 时间字符串或 Date 对象
   * @returns 格式化后的相对时间字符串
   */
  const formatTime = (dateString: string | Date | null | undefined): string => {
    // 处理空值和无效值
    if (!dateString) return '-'

    let date: Date
    try {
      date = typeof dateString === 'string' ? new Date(dateString) : dateString
      if (isNaN(date.getTime())) return '-'
    } catch {
      return '-'
    }

    const now = new Date()
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    if (diffInSeconds < 60) return t('time.justNow')
    if (diffInSeconds < 3600) return t('time.minutesAgo', { n: Math.floor(diffInSeconds / 60) })
    if (diffInSeconds < 86400) return t('time.hoursAgo', { n: Math.floor(diffInSeconds / 3600) })
    if (diffInSeconds < 2592000) return t('time.daysAgo', { n: Math.floor(diffInSeconds / 86400) })
    if (diffInSeconds < 31536000) {
      const months = Math.floor(diffInSeconds / 2592000)
      return t('time.monthsAgo', { n: months })
    }

    const years = Math.floor(diffInSeconds / 31536000)
    return t('time.yearsAgo', { n: years })
  }

  /**
   * 格式化距离为可读的距离字符串（国际化版本）
   * @param distance 距离（米）
   * @returns 格式化后的距离字符串
   */
  const formatDistance = (distance: number | null | undefined): string => {
    // 处理无效值
    if (typeof distance !== 'number' || isNaN(distance) || distance < 0) return '-'

    if (distance < 1000) {
      return t('distance.meters', { n: Math.round(distance) })
    } else {
      return t('distance.kilometers', { n: (distance / 1000).toFixed(1) })
    }
  }

  /**
   * 格式化日期为可读的日期字符串（国际化版本）
   * @param dateString ISO 8601 时间字符串或 Date 对象
   * @param format 格式类型：'short' | 'long' | 'time'
   * @returns 格式化后的日期字符串
   */
  const formatDate = (dateString: string | Date | null | undefined, format: 'short' | 'long' | 'time' = 'short'): string => {
    // 处理空值和无效值
    if (!dateString) return '-'

    let date: Date
    try {
      date = typeof dateString === 'string' ? new Date(dateString) : dateString
      if (isNaN(date.getTime())) return '-'
    } catch {
      return '-'
    }

    // 获取当前语言环境
    const i18nInstance = $i18n as { locale?: { value?: string } }
    const locale = i18nInstance?.locale?.value || 'zh-CN'

    switch (format) {
      case 'long':
        return date.toLocaleDateString(locale, {
          year: 'numeric',
          month: 'long',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit'
        })
      case 'time':
        return date.toLocaleTimeString(locale, {
          hour: '2-digit',
          minute: '2-digit'
        })
      case 'short':
      default:
        return date.toLocaleDateString(locale, {
          month: 'short',
          day: 'numeric',
          hour: '2-digit',
          minute: '2-digit'
        })
    }
  }

  /**
   * 格式化数字为可读的数量字符串（如点赞数、成员数等）（国际化版本）
   * @param count 数量
   * @returns 格式化后的数量字符串
   */
  const formatCount = (count: number | null | undefined): string => {
    // 处理无效值
    if (typeof count !== 'number' || isNaN(count) || count < 0) return '0'

    if (count < 1000) {
      return count.toString()
    } else if (count < 10000) {
      return t('number.thousands', { n: (count / 1000).toFixed(1) })
    } else if (count < 100000000) {
      return t('number.tenThousands', { n: Math.floor(count / 10000) })
    } else {
      return t('number.hundredMillions', { n: (count / 100000000).toFixed(1) })
    }
  }

  return {
    formatTime,
    formatDistance,
    formatDate,
    formatCount
  }
}
