// Theme store — 明暗三档（跟随系统 auto / 手动浅色 light / 手动深色 dark）
// + 语义主题（vibrant 星际 / forest 清野 / inkwash 水墨）。
// ThemeContext（runes 模块单例）。

import type { Component } from 'svelte';
// 图标走子路径导入（@lucide/svelte barrel 会 re-export 全量图标，
// 在 vitest resetModules 场景下每用例重复解析导致超时）
import Brush from '@lucide/svelte/icons/brush';
import Palette from '@lucide/svelte/icons/palette';
import Trees from '@lucide/svelte/icons/trees';

export type AppTheme = 'forest' | 'vibrant' | 'inkwash';

/**
 * 明暗模式三档：
 * - auto：跟随系统（matchMedia('(prefers-color-scheme: dark)') 初值 + change 监听刷新）
 * - light / dark：手动浅色 / 手动深色（忽略系统变化）
 */
export type ColorSchemeMode = 'auto' | 'light' | 'dark';

/** 明暗三档选项（设置页与外观菜单共用，避免两份配置漂移）。 */
export const COLOR_SCHEME_OPTIONS = [
	{ id: 'auto', label: '跟随系统' },
	{ id: 'light', label: '浅色' },
	{ id: 'dark', label: '深色' }
] as const;

/**
 * 主题选项列表（设置页与外观菜单共用）。
 * swatch 用字面值而非 var(--cst-palette-accent-primary)，因为色板需要在主题激活前
 * 静态展示各主题代表色（CSS 变量解析的是"当前"主题）；与 themes/*.css 的
 * --cst-palette-accent-primary 保持同步（已登记为 AGENTS.md 配色硬约束的例外，
 * 由 check-cross-platform-colors.mjs 的 swatch 校验锁定）。
 * icon 为主题图标的单一来源（settings 页 TabBar 消费），禁止在页面里按 id 硬编码映射。
 */
export const THEME_OPTIONS = [
	{ id: 'vibrant', label: '星际', note: '冷色玻璃', icon: Palette, swatch: '#2B5EB2' },
	{ id: 'forest', label: '清野', note: '自然纸面', icon: Trees, swatch: '#39714F' },
	{ id: 'inkwash', label: '水墨', note: '宣纸墨韵', icon: Brush, swatch: '#A63A2E' }
] as const;

const COLOR_SCHEME_KEY = 'colorScheme';
const THEME_KEY = 'appTheme';

let colorScheme = $state<ColorSchemeMode>('auto');
let systemDark = $state(false);
// 内部状态变量与导出对象重名会自引用（getter 内 `theme` 解析到导出对象本身），
// 故内部用 appTheme，导出对象命名为 theme（与 auth/composer/presence/ws 一致）。
let appTheme = $state<AppTheme>('vibrant');

/** 生效的暗色（手动档直接取值，auto 档跟随系统）。 */
const effectiveDark = $derived(colorScheme === 'dark' || (colorScheme === 'auto' && systemDark));

function prefersDark(): boolean {
	try {
		return typeof window !== 'undefined' && typeof window.matchMedia === 'function'
			? window.matchMedia('(prefers-color-scheme: dark)').matches
			: false;
	} catch {
		return false;
	}
}

function prefersReduced(): boolean {
	try {
		return typeof window !== 'undefined' && typeof window.matchMedia === 'function'
			? window.matchMedia('(prefers-reduced-motion: reduce)').matches
			: false;
	} catch {
		return false;
	}
}

/* Apply / remove classes on <html> based on current state */
function applyClasses(dark: boolean, t: AppTheme) {
  const root = document.documentElement;

  // Dark
  dark ? root.classList.add('dark') : root.classList.remove('dark');

  root.classList.remove('forest', 'vibrant', 'inkwash');
  root.classList.add(t);

  // 浏览器 chrome 配色跟随 token（读取当前主题的 --cst-surface-page）
  syncThemeColor(root);
}

/**
 * 应用主题类到 <html>，并在支持的环境使用 View Transition 实现整页平滑过渡（P1-6 收敛大面积重绘）。
 * @param dark 是否为深色
 * @param t 主题
 * @param animated 是否启用过渡动画（首屏 init 时为 false，确保零延迟首屏呈现）
 */
function applyTheme(dark: boolean, t: AppTheme, animated = true): void {
  if (
    animated &&
    typeof document !== 'undefined' &&
    'startViewTransition' in document &&
    !prefersReduced()
  ) {
    try {
      (document as unknown as { startViewTransition: (cb: () => void) => void }).startViewTransition(() => {
        applyClasses(dark, t);
      });
      return;
    } catch {
      // 降级直接生效
    }
  }
  applyClasses(dark, t);
}

/** 将 <meta name="theme-color"> 同步为当前主题画布色（移动端地址栏/任务视图）。 */
function syncThemeColor(root: HTMLElement): void {
  const meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) return;
  const value = getComputedStyle(root).getPropertyValue('--cst-surface-page').trim();
  if (value) meta.setAttribute('content', value);
}

function readColorScheme(): ColorSchemeMode {
	try {
		const saved = localStorage.getItem(COLOR_SCHEME_KEY);
		if (saved === 'auto' || saved === 'light' || saved === 'dark') return saved;
	} catch {}
	return 'auto';
}

function readInitial(): { scheme: ColorSchemeMode; theme: AppTheme } {
  let scheme: ColorSchemeMode = 'auto';
  let t: AppTheme = 'vibrant';
  try {
    scheme = readColorScheme();
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === 'forest' || saved === 'vibrant' || saved === 'inkwash') t = saved;
  } catch {}
  return { scheme, theme: t };
}

/**
 * 由根布局调用一次：读取持久化初值并应用到 <html> 类，同时订阅系统明暗变化。
 * @returns 清理函数（移除 matchMedia 监听），由根布局在销毁时调用。
 */
export function initTheme(): () => void {
  const initial = readInitial();
  colorScheme = initial.scheme;
  appTheme = initial.theme;
  systemDark = prefersDark();
  applyTheme(colorScheme === 'dark' || (colorScheme === 'auto' && systemDark), appTheme, false);

  // auto 档跟随系统：change 监听刷新（手动档下监听无操作，仅保留订阅）。
  let mql: MediaQueryList | null = null;
  const onChange = (e: MediaQueryListEvent) => {
    systemDark = e.matches;
    applyTheme(colorScheme === 'dark' || (colorScheme === 'auto' && e.matches), appTheme, true);
  };
  try {
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      mql = window.matchMedia('(prefers-color-scheme: dark)');
      mql.addEventListener('change', onChange);
    }
  } catch {
    mql = null;
  }
  return () => {
    try { mql?.removeEventListener('change', onChange); } catch {}
  };
}

/** 设置明暗三档（持久化 + 应用）。 */
export function setColorScheme(mode: ColorSchemeMode): void {
  colorScheme = mode;
  try { localStorage.setItem(COLOR_SCHEME_KEY, mode); } catch {}
  applyTheme(colorScheme === 'dark' || (colorScheme === 'auto' && systemDark), appTheme, true);
}

/**
 * 兼容入口：在手动档之间切换（light ↔ dark）。
 * auto 档下调用则落到与当前生效值相反的手动档（显式选择覆盖跟随）。
 */
export function toggleDark(): void {
  setColorScheme(effectiveDark ? 'light' : 'dark');
}

export function setTheme(t: AppTheme): void {
  appTheme = t;
  try { localStorage.setItem(THEME_KEY, t); } catch {}
  applyTheme(effectiveDark, appTheme, true);
}

export const theme = {
  get isDark() { return effectiveDark; },
  get colorScheme() { return colorScheme; },
  get theme() { return appTheme; },
};
