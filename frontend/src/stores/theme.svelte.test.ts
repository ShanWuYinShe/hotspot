// Theme store 测试：明暗三档（auto/light/dark）+ localStorage 初值与 html 类应用。
import { describe, it, expect, beforeEach, vi } from 'vitest';

beforeEach(async () => {
  // 重置模块实例 + 清空 localStorage，保证用例隔离
  vi.resetModules();
  localStorage.clear();
  document.documentElement.className = '';
  // 清理上个用例的 matchMedia mock（jsdom 默认无 matchMedia）
  try { delete (window as unknown as Record<string, unknown>).matchMedia; } catch {}
});

/** mock window.matchMedia，返回可手动 dispatch change 的 mql。 */
function mockMatchMedia(initialMatches: boolean) {
  const listeners = new Set<(e: { matches: boolean }) => void>();
  const mql = {
    matches: initialMatches,
    media: '(prefers-color-scheme: dark)',
    addEventListener: vi.fn((_type: string, cb: (e: { matches: boolean }) => void) => {
      listeners.add(cb);
    }),
    removeEventListener: vi.fn((_type: string, cb: (e: { matches: boolean }) => void) => {
      listeners.delete(cb);
    }),
    dispatch(next: boolean) {
      mql.matches = next;
      for (const cb of [...listeners]) cb({ matches: next });
    }
  };
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn(() => mql)
  });
  return mql;
}

it('默认 vibrant + 自动档（无系统暗色时浅色）', async () => {
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.theme).toBe('vibrant');
  expect(theme.colorScheme).toBe('auto');
  expect(theme.isDark).toBe(false);
  expect(document.documentElement.classList.contains('vibrant')).toBe(true);
});

it('auto 初值跟随系统暗色', async () => {
  mockMatchMedia(true);
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.colorScheme).toBe('auto');
  expect(theme.isDark).toBe(true);
  expect(document.documentElement.classList.contains('dark')).toBe(true);
});

it('auto 档响应系统 change 事件；清理函数移除监听', async () => {
  const mql = mockMatchMedia(false);
  const { initTheme, theme } = await import('./theme.svelte');
  const cleanup = initTheme();
  expect(theme.isDark).toBe(false);
  mql.dispatch(true);
  expect(theme.isDark).toBe(true);
  expect(document.documentElement.classList.contains('dark')).toBe(true);
  cleanup();
  mql.dispatch(false);
  // 监听已移除：状态不再跟随系统
  expect(theme.isDark).toBe(true);
});

it('老 isDark=true 迁移为手动 dark 并删老键', async () => {
  localStorage.setItem('isDark', 'true');
  localStorage.setItem('appTheme', 'forest');
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.colorScheme).toBe('dark');
  expect(theme.isDark).toBe(true);
  expect(theme.theme).toBe('forest');
  expect(localStorage.getItem('colorScheme')).toBe('dark');
  expect(localStorage.getItem('isDark')).toBeNull();
  const cls = document.documentElement.classList;
  expect(cls.contains('dark')).toBe(true);
  expect(cls.contains('forest')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
});

it('老 isDark=false 迁移为手动 light', async () => {
  localStorage.setItem('isDark', 'false');
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.colorScheme).toBe('light');
  expect(theme.isDark).toBe(false);
  expect(localStorage.getItem('colorScheme')).toBe('light');
  expect(localStorage.getItem('isDark')).toBeNull();
});

it('老键缺失时视为 auto（不写新键）', async () => {
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.colorScheme).toBe('auto');
  expect(localStorage.getItem('colorScheme')).toBeNull();
});

it('setColorScheme 持久化并应用类；手动档不受系统变化影响', async () => {
  const mql = mockMatchMedia(false);
  const { initTheme, setColorScheme, theme } = await import('./theme.svelte');
  initTheme();
  setColorScheme('dark');
  expect(theme.colorScheme).toBe('dark');
  expect(theme.isDark).toBe(true);
  expect(localStorage.getItem('colorScheme')).toBe('dark');
  expect(document.documentElement.classList.contains('dark')).toBe(true);
  mql.dispatch(true);
  mql.dispatch(false);
  // 手动 dark 不受系统变化影响
  expect(theme.isDark).toBe(true);
  setColorScheme('auto');
  expect(theme.isDark).toBe(false);
});

it('toggleDark 在手动档之间切换（兼容入口）', async () => {
  const { initTheme, toggleDark, setTheme, theme } = await import('./theme.svelte');
  initTheme();
  toggleDark();
  expect(theme.colorScheme).toBe('dark');
  expect(theme.isDark).toBe(true);
  expect(localStorage.getItem('colorScheme')).toBe('dark');
  toggleDark();
  expect(theme.colorScheme).toBe('light');
  expect(theme.isDark).toBe(false);
  setTheme('forest');
  expect(theme.theme).toBe('forest');
  expect(localStorage.getItem('appTheme')).toBe('forest');
  expect(document.documentElement.classList.contains('forest')).toBe(true);
});

it('setTheme 切到 inkwash：持久化 + 旧主题类移除', async () => {
  const { initTheme, setTheme, theme } = await import('./theme.svelte');
  initTheme();
  setTheme('inkwash');
  expect(theme.theme).toBe('inkwash');
  expect(localStorage.getItem('appTheme')).toBe('inkwash');
  const cls = document.documentElement.classList;
  expect(cls.contains('inkwash')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
  expect(cls.contains('forest')).toBe(false);
});

it('持久化 inkwash 初值：initTheme 恢复 inkwash（含 dark 类）', async () => {
  localStorage.setItem('appTheme', 'inkwash');
  localStorage.setItem('colorScheme', 'dark');
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.theme).toBe('inkwash');
  expect(theme.isDark).toBe(true);
  const cls = document.documentElement.classList;
  expect(cls.contains('inkwash')).toBe(true);
  expect(cls.contains('dark')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
});

it('非法持久化主题值回退默认 vibrant', async () => {
  localStorage.setItem('appTheme', 'invalid-theme');
  localStorage.setItem('colorScheme', 'invalid-mode');
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  expect(theme.theme).toBe('vibrant');
  expect(theme.colorScheme).toBe('auto');
  expect(theme.isDark).toBe(false);
  expect(document.documentElement.classList.contains('vibrant')).toBe(true);
  expect(document.documentElement.classList.contains('dark')).toBe(false);
});

it('主题切换链 vibrant → inkwash → forest：类列表互斥且只剩当前主题', async () => {
  const { initTheme, setTheme } = await import('./theme.svelte');
  initTheme();
  const cls = document.documentElement.classList;
  setTheme('inkwash');
  expect(cls.contains('inkwash')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
  setTheme('forest');
  expect(cls.contains('forest')).toBe(true);
  expect(cls.contains('inkwash')).toBe(false);
  expect(localStorage.getItem('appTheme')).toBe('forest');
});

it('initTheme 幂等：重复调用不改变已应用的主题', async () => {
  localStorage.setItem('colorScheme', 'dark');
  localStorage.setItem('appTheme', 'forest');
  const { initTheme, theme } = await import('./theme.svelte');
  initTheme();
  initTheme();
  expect(theme.theme).toBe('forest');
  expect(theme.isDark).toBe(true);
  const cls = document.documentElement.classList;
  expect(cls.contains('dark')).toBe(true);
  expect(cls.contains('forest')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
});
