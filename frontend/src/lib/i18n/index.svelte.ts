/**
 * 轻量 i18n 模块
 *
 * 基于 Svelte 5 runes 实现，无需第三方依赖。
 * - t() 在模板中使用时自动响应 locale 变化
 * - 数据数组中包含翻译字符串时需用 $derived 包裹
 * - 缺失的翻译自动回退到 zh-CN
 */
import zhCN from './locales/zh-CN';
import en from './locales/en';

export type Locale = 'zh-CN' | 'en';

// 从 zh-CN 推导类型安全的 key
type NestedKeyOf<T> = T extends object
  ? {
      [K in keyof T & string]: T[K] extends string ? K : `${K}.${NestedKeyOf<T[K]>}`;
    }[keyof T & string]
  : never;

export type TranslationKey = NestedKeyOf<typeof zhCN>;

const translations = { 'zh-CN': zhCN, en } as const;

let currentLocale = $state<Locale>('zh-CN');

function flattenTranslations(
  obj: unknown,
  prefix = '',
  result: Record<string, string> = {}
): Record<string, string> {
  if (obj == null || typeof obj !== 'object') return result;

  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;
    if (typeof value === 'string') {
      result[nextKey] = value;
      continue;
    }
    flattenTranslations(value, nextKey, result);
  }

  return result;
}

const flatTranslations: Record<Locale, Record<string, string>> = {
  'zh-CN': flattenTranslations(zhCN),
  en: flattenTranslations(en)
};

function translateWithParams(
  locale: Locale,
  key: TranslationKey,
  params?: Record<string, string | number>
): string {
  const raw = flatTranslations[locale][key] ?? flatTranslations['zh-CN'][key] ?? key;
  if (!params) return raw;
  return raw.replace(/\{(\w+)\}/g, (_, k: string) => String(params[k] ?? ''));
}

/**
 * 翻译函数
 *
 * 在模板中使用时自动追踪 reactive 依赖（currentLocale），
 * locale 变更时组件自动重新渲染。
 */
export function t(key: TranslationKey, params?: Record<string, string | number>): string {
  const locale = currentLocale; // reactive read
  return translateWithParams(locale, key, params);
}

export function translateWithLocale(
  locale: Locale,
  key: TranslationKey,
  params?: Record<string, string | number>
): string {
  return translateWithParams(locale, key, params);
}

export function getLocale(): Locale {
  return currentLocale;
}

export function setLocale(locale: Locale): void {
  currentLocale = locale;
  try {
    localStorage.setItem('locale', locale);
  } catch {
    /* quota exceeded */
  }
}

export function getCurrentLocationLabel(locale: Locale = getLocale()): string {
  return translateWithLocale(locale, 'common.currentLocation');
}

/** 在 onMount 中调用，从 localStorage 恢复用户偏好 */
export function initLocale(): void {
  try {
    const saved = localStorage.getItem('locale');
    if (saved && saved in translations) {
      currentLocale = saved as Locale;
    }
  } catch {
    // ignore
  }
}

export const SUPPORTED_LOCALES: { value: Locale; label: string }[] = [
  { value: 'zh-CN', label: '中文' },
  { value: 'en', label: 'English' }
];
