// Theme store 测试：localStorage 初值、'neu'→'forest' 迁移、html 类应用。
import { describe, it, expect, beforeEach, vi } from 'vitest';

beforeEach(async () => {
  // 重置模块实例 + 清空 localStorage，保证用例隔离
  vi.resetModules();
  localStorage.clear();
  document.documentElement.className = '';
});

it('默认 vibrant + 非暗色', async () => {
  const { initTheme, themeStore } = await import('./theme.svelte');
  initTheme();
  expect(themeStore.theme).toBe('vibrant');
  expect(themeStore.isDark).toBe(false);
  expect(document.documentElement.classList.contains('vibrant')).toBe(true);
});

it('读取持久化初值并应用 html 类', async () => {
  localStorage.setItem('isDark', 'true');
  localStorage.setItem('appTheme', 'forest');
  const { initTheme, themeStore } = await import('./theme.svelte');
  initTheme();
  expect(themeStore.isDark).toBe(true);
  expect(themeStore.theme).toBe('forest');
  const cls = document.documentElement.classList;
  expect(cls.contains('dark')).toBe(true);
  expect(cls.contains('forest')).toBe(true);
  expect(cls.contains('vibrant')).toBe(false);
});

it('废弃的 "neu" 偏好迁移到 forest', async () => {
  localStorage.setItem('appTheme', 'neu');
  const { initTheme, themeStore } = await import('./theme.svelte');
  initTheme();
  expect(themeStore.theme).toBe('forest');
});

it('toggleDark / setTheme 持久化并应用类', async () => {
  const { initTheme, toggleDark, setTheme, themeStore } = await import('./theme.svelte');
  initTheme();
  toggleDark();
  expect(themeStore.isDark).toBe(true);
  expect(localStorage.getItem('isDark')).toBe('true');
  setTheme('forest');
  expect(themeStore.theme).toBe('forest');
  expect(localStorage.getItem('appTheme')).toBe('forest');
  expect(document.documentElement.classList.contains('forest')).toBe(true);
});
