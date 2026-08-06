// Theme store — dark mode + 语义主题（vibrant 星际 / forest 清野）。
// ThemeContext（runes 模块单例）。

export type AppTheme = 'forest' | 'vibrant';

/**
 * 主题选项列表（设置页与外观菜单共用）。
 * swatch 用字面值而非 var(--pr-*)，因为色板需要在主题激活前静态展示
 * 各主题代表色（CSS 变量解析的是"当前"主题）；与 themes/*.css 的 --pr-accent-primary 保持同步。
 */
export const THEME_OPTIONS = [
	{ id: 'vibrant', label: '星际', note: '冷色玻璃', swatch: '#6366F1' },
	{ id: 'forest', label: '清野', note: '自然纸面', swatch: '#3D7A55' }
] as const;

const IS_DARK_KEY = 'isDark';
const THEME_KEY = 'appTheme';

let isDark = $state(false);
// 内部状态变量与导出对象重名会自引用（getter 内 `theme` 解析到导出对象本身），
// 故内部用 appTheme，导出对象命名为 theme（与 auth/composer/presence/ws 一致）。
let appTheme = $state<AppTheme>('vibrant');

/* Apply / remove classes on <html> based on current state */
function applyClasses(dark: boolean, t: AppTheme) {
  const root = document.documentElement;

  // Dark
  dark ? root.classList.add('dark') : root.classList.remove('dark');

  root.classList.remove('forest', 'vibrant');
  root.classList.add(t);
}

function readInitial(): { isDark: boolean; theme: AppTheme } {
  let dark = false;
  let t: AppTheme = 'vibrant';
  try {
    dark = localStorage.getItem(IS_DARK_KEY) === 'true';
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === 'forest' || saved === 'vibrant') t = saved;
  } catch {}
  return { isDark: dark, theme: t };
}

/** 由根布局调用一次：读取持久化初值并应用到 <html> 类。 */
export function initTheme(): void {
  const initial = readInitial();
  isDark = initial.isDark;
  appTheme = initial.theme;
  applyClasses(isDark, appTheme);
}

export function toggleDark(): void {
  isDark = !isDark;
  try { localStorage.setItem(IS_DARK_KEY, String(isDark)); } catch {}
  applyClasses(isDark, appTheme);
}

export function setTheme(t: AppTheme): void {
  appTheme = t;
  try { localStorage.setItem(THEME_KEY, t); } catch {}
  applyClasses(isDark, appTheme);
}

export const theme = {
  get isDark() { return isDark; },
  get theme() { return appTheme; },
};
