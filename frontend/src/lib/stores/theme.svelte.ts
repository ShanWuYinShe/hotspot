/**
 * 主题族状态管理
 * 管理多主题 family，明暗模式仍由 mode-watcher 负责。
 */

import { browser } from '$app/environment';
import type { TranslationKey } from '$lib/i18n';

export const THEME_FAMILIES = ['mist', 'grove'] as const;

export type ThemeFamily = (typeof THEME_FAMILIES)[number];

export const THEME_FAMILY_OPTIONS: {
  family: ThemeFamily;
  labelKey: TranslationKey;
  descriptionKey: TranslationKey;
}[] = [
  {
    family: 'mist',
    labelKey: 'sidebar.mist',
    descriptionKey: 'sidebar.mistDesc'
  },
  {
    family: 'grove',
    labelKey: 'sidebar.grove',
    descriptionKey: 'sidebar.groveDesc'
  }
];

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'mist';

/** `<html>` 上用于选择当前主题族的属性名 */
export const THEME_FAMILY_ATTRIBUTE = 'data-theme-family';

// 主题重命名(indigo→mist, forest→grove)+ 删除 sand 后,老 localStorage 值
// 一律落回默认(mist),不做语义迁移 —— 见 docs/specs/2026-07-19-theme-rename-and-flatten-design.md §1.A
const THEME_MIGRATION: Record<string, ThemeFamily> = {};

function isThemeFamily(value: string | null | undefined): value is ThemeFamily {
  return THEME_FAMILIES.some((themeFamily) => themeFamily === value);
}

function normalizeThemeFamily(value: string | null | undefined): ThemeFamily {
  if (isThemeFamily(value)) return value;
  if (value && value in THEME_MIGRATION)
    return THEME_MIGRATION[value as keyof typeof THEME_MIGRATION]!;
  return DEFAULT_THEME_FAMILY;
}

/**
 * 纯函数主题族解析器，会被序列化为阻塞式 <head> 脚本，在首屏绘制前应用主题族
 * 以避免主题闪烁（FOUC）。函数体不能引用任何模块作用域变量——所有输入均通过
 * config 传入，以保证 `.toString()` 序列化结果自包含、可独立运行。
 * 解析规则必须与 normalizeThemeFamily 保持一致，避免水合前后主题不一致。
 */
function resolveInitialThemeFamily(config: {
  storageKey: string;
  attribute: string;
  families: readonly string[];
  migration: Record<string, string>;
  defaultFamily: string;
}) {
  try {
    const stored = window.localStorage.getItem(config.storageKey);
    const family =
      stored && config.families.indexOf(stored) !== -1
        ? stored
        : (stored && config.migration[stored]) || config.defaultFamily;
    document.documentElement.setAttribute(config.attribute, family);
  } catch {
    document.documentElement.setAttribute(config.attribute, config.defaultFamily);
  }
}

/**
 * 阻塞式内联脚本源码：在 hydration 之前根据 localStorage 把持久化的主题族写入
 * <html>，避免首屏主题闪烁。src/app.html 中的静态脚本需与这里保持一致。
 */
export const THEME_FAMILY_INIT_SCRIPT = `(${resolveInitialThemeFamily.toString()})(${JSON.stringify(
  {
    storageKey: THEME_FAMILY_STORAGE_KEY,
    attribute: THEME_FAMILY_ATTRIBUTE,
    families: THEME_FAMILIES,
    migration: THEME_MIGRATION,
    defaultFamily: DEFAULT_THEME_FAMILY
  }
)});`;

class ThemeStore {
  #themeFamily = $state<ThemeFamily>(DEFAULT_THEME_FAMILY);

  themeFamily = $derived(this.#themeFamily);

  init() {
    if (!browser) return;
    const stored = normalizeThemeFamily(localStorage.getItem(THEME_FAMILY_STORAGE_KEY));
    this.#themeFamily = stored;
    if (stored !== normalizeThemeFamily(null)) {
      localStorage.setItem(THEME_FAMILY_STORAGE_KEY, stored);
    }
    this.#apply(stored);
  }

  setThemeFamily(themeFamily: ThemeFamily) {
    if (this.#themeFamily === themeFamily) return;
    this.#themeFamily = themeFamily;
    if (!browser) return;
    localStorage.setItem(THEME_FAMILY_STORAGE_KEY, themeFamily);
    this.#apply(themeFamily);
  }

  #apply(themeFamily: ThemeFamily) {
    if (!browser) return;
    document.documentElement.setAttribute(THEME_FAMILY_ATTRIBUTE, themeFamily);
  }
}

export const themeStore = new ThemeStore();
