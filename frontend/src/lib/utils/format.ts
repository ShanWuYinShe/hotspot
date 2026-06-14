/**
 * 格式化工具函数
 */

import { getLocale, translateWithLocale } from '$lib/i18n';
import type { Locale } from '$lib/i18n';

/** 校验日期字符串是否有效 */
function isValidDate(dateStr: string): boolean {
  if (!dateStr) return false;
  const ts = new Date(dateStr).getTime();
  return !isNaN(ts);
}

type RelativeTimeOptions = {
  fallback?: 'datetime' | 'date';
  dateOptions?: Intl.DateTimeFormatOptions;
};

const dateFormatters = new Map<string, Intl.DateTimeFormat>();

function getDateFormatter(
  locale: Locale,
  options: Intl.DateTimeFormatOptions
): Intl.DateTimeFormat {
  const cacheKey = `${locale}:${JSON.stringify(options)}`;
  const cached = dateFormatters.get(cacheKey);
  if (cached) return cached;
  const formatter = new Intl.DateTimeFormat(locale, options);
  dateFormatters.set(cacheKey, formatter);
  return formatter;
}

/**
 * 格式化时间为相对时间（刚刚、X分钟前、X小时前等）
 */
export function formatRelativeTime(
  dateStr: string,
  locale: Locale = getLocale(),
  options: RelativeTimeOptions = {}
): string {
  if (!isValidDate(dateStr)) return '—';

  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);
  const { fallback = 'datetime', dateOptions } = options;

  if (minutes < 1) return translateWithLocale(locale, 'relativeTime.justNow');
  if (minutes < 60)
    return translateWithLocale(locale, 'relativeTime.minutesAgo', { count: minutes });
  if (hours < 24) return translateWithLocale(locale, 'relativeTime.hoursAgo', { count: hours });
  if (days < 7) return translateWithLocale(locale, 'relativeTime.daysAgo', { count: days });
  return fallback === 'date'
    ? formatDate(dateStr, locale, dateOptions ?? { month: 'short', day: 'numeric' })
    : formatDateTime(dateStr, locale);
}

/**
 * 格式化时间为完整日期时间
 */
export function formatDateTime(dateStr: string, locale: Locale = getLocale()): string {
  if (!isValidDate(dateStr)) return '—';
  return getDateFormatter(locale, {
    dateStyle: 'medium',
    timeStyle: 'medium',
    hour12: false
  }).format(new Date(dateStr));
}

/**
 * 格式化时间为日期（不含时间）
 */
export function formatDate(
  dateStr: string,
  locale: Locale = getLocale(),
  options: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'numeric', day: 'numeric' }
): string {
  if (!isValidDate(dateStr)) return '—';
  return getDateFormatter(locale, options).format(new Date(dateStr));
}

/**
 * 格式化距离（米 -> 米/公里）
 */
export function formatDistance(meters: number, locale: Locale = getLocale()): string {
  if (!isFinite(meters) || meters < 0) return '—';
  if (meters < 1000) {
    const rounded = Math.round(meters);
    return locale === 'zh-CN' ? `${rounded}米` : `${rounded} m`;
  }
  const kilometers = (meters / 1000).toFixed(1);
  return locale === 'zh-CN' ? `${kilometers}公里` : `${kilometers} km`;
}

/**
 * 生成用户 @handle（展示用，全站单一事实源）。
 *
 * 规则：优先取昵称小写、连续空白折叠为点号；昵称为空时回退到 user_id 前 8 位。
 * 返回值不含前导 `@`，模板里保留字面量 `@`（如 `@{formatHandle(user)}`）。
 *
 * 注意：handle 仅用于展示，不保证全局唯一、不随昵称变化保持稳定，不可作为身份标识。
 */
export function formatHandle(user: { nickname?: string | null; user_id?: string | null }): string {
  const nickname = user.nickname?.trim();
  if (nickname) {
    return nickname.toLowerCase().replace(/\s+/g, '.');
  }
  const id = user.user_id?.trim();
  if (id) return id.slice(0, 8);
  return 'user';
}

/**
 * 解析用户输入的标签字符串（支持逗号、中文逗号、空格分隔，自动去除 # 前缀）
 */
export function parseTags(input: string, maxCount = 10): string[] {
  return input
    .split(/[,，\s]+/)
    .map((tag) => tag.replace(/^#/, '').trim())
    .filter(Boolean)
    .slice(0, maxCount);
}
