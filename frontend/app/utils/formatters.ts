import type { Ref } from 'vue'

export type I18nT = (key: string, params?: Record<string, string | number>) => string

export type DateFormatStyle = 'short' | 'long' | 'time'

function safeDate(input: string | Date | null | undefined): Date | null {
  if (!input) return null
  try {
    const d = typeof input === 'string' ? new Date(input) : input
    if (Number.isNaN(d.getTime())) return null
    return d
  } catch {
    return null
  }
}

export function formatRelativeTime(t: I18nT, input: string | Date | null | undefined): string {
  const date = safeDate(input)
  if (!date) return '-'

  const now = new Date()
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)

  if (diffInSeconds < 60) return t('time.justNow')
  if (diffInSeconds < 3600) return t('time.minutesAgo', { n: Math.floor(diffInSeconds / 60) })
  if (diffInSeconds < 86400) return t('time.hoursAgo', { n: Math.floor(diffInSeconds / 3600) })
  if (diffInSeconds < 2592000) return t('time.daysAgo', { n: Math.floor(diffInSeconds / 86400) })
  if (diffInSeconds < 31536000) return t('time.monthsAgo', { n: Math.floor(diffInSeconds / 2592000) })
  return t('time.yearsAgo', { n: Math.floor(diffInSeconds / 31536000) })
}

export function formatDistanceMeters(t: I18nT, distance: number | null | undefined): string {
  if (typeof distance !== 'number' || Number.isNaN(distance) || distance < 0) return '-'
  if (distance < 1000) return t('distance.meters', { n: Math.round(distance) })
  return t('distance.kilometers', { n: (distance / 1000).toFixed(1) })
}

export function formatCount(t: I18nT, count: number | null | undefined): string {
  if (typeof count !== 'number' || Number.isNaN(count) || count < 0) return '0'
  if (count < 1000) return count.toString()
  if (count < 10000) return t('number.thousands', { n: (count / 1000).toFixed(1) })
  if (count < 100000000) return t('number.tenThousands', { n: Math.floor(count / 10000) })
  return t('number.hundredMillions', { n: (count / 100000000).toFixed(1) })
}

export function formatDate(
  locale: string,
  input: string | Date | null | undefined,
  format: DateFormatStyle = 'short'
): string {
  const date = safeDate(input)
  if (!date) return '-'

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

export function createI18nFormatters(t: I18nT, localeRef: Ref<string>) {
  const getLocale = () => localeRef.value || 'zh-CN'

  return {
    formatTime: (date: string | Date | null | undefined) => formatRelativeTime(t, date),
    formatDistance: (distance: number | null | undefined) => formatDistanceMeters(t, distance),
    formatDate: (date: string | Date | null | undefined, style: DateFormatStyle = 'short') =>
      formatDate(getLocale(), date, style),
    formatCount: (count: number | null | undefined) => formatCount(t, count)
  }
}

