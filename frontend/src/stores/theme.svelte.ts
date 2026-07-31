// Theme store — dark mode + 语义主题（vibrant 星际 / forest 清野）。
// 迁移自旧版 ThemeContext（runes 模块单例）。

export type AppTheme = 'forest' | 'vibrant';

const IS_DARK_KEY = 'isDark';
const THEME_KEY = 'appTheme';

let isDark = $state(false);
let theme = $state<AppTheme>('vibrant');

/* Apply / remove classes on <html> based on current state */
function applyClasses(dark: boolean, t: AppTheme) {
  const root = document.documentElement;

  // Dark
  dark ? root.classList.add('dark') : root.classList.remove('dark');

  // Both surviving themes are explicit classes; this prevents a stale warm-sand class from lingering.
  root.classList.remove('forest', 'vibrant', 'neu');
  root.classList.add(t);
}

function readInitial(): { isDark: boolean; theme: AppTheme } {
  let dark = false;
  let t: AppTheme = 'vibrant';
  try {
    dark = localStorage.getItem(IS_DARK_KEY) === 'true';
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === 'forest' || saved === 'vibrant') t = saved;
    // Migrate retired 暖砂 preference to 清野 so old local storage cannot revive it.
    else if (saved === 'neu') t = 'forest';
  } catch {}
  return { isDark: dark, theme: t };
}

/** 由根布局调用一次：读取持久化初值并应用到 <html> 类。 */
export function initTheme(): void {
  const initial = readInitial();
  isDark = initial.isDark;
  theme = initial.theme;
  applyClasses(isDark, theme);
}

export function toggleDark(): void {
  isDark = !isDark;
  try { localStorage.setItem(IS_DARK_KEY, String(isDark)); } catch {}
  applyClasses(isDark, theme);
}

export function setTheme(t: AppTheme): void {
  theme = t;
  try { localStorage.setItem(THEME_KEY, t); } catch {}
  applyClasses(isDark, theme);
}

export const themeStore = {
  get isDark() { return isDark; },
  get theme() { return theme; },
};
