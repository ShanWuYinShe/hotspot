// Auth store 测试：requireAuth 拦截与 pending-action 重放、事件降级。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { AccountData } from '@/api/types';

vi.mock('@/api', () => ({
  checkAuthStatus: vi.fn(),
  getAccount: vi.fn(),
  guest: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
  setAuthenticated: vi.fn(),
}));

import * as api from '@/api';

const accountMock = { user_id: 'u1', nickname: '测试', avatar_url: '' } as AccountData;

beforeEach(async () => {
  vi.resetModules();
  vi.clearAllMocks();
  sessionStorage.clear();
  localStorage.clear();
  // 默认未认证：session 端点返回 10100（客户端据此判定匿名态）
  vi.mocked(api.checkAuthStatus).mockResolvedValue({ code: 10100, message: '未登录', data: null as never });
  vi.mocked(api.getAccount).mockResolvedValue({ code: 0, data: accountMock });
});

it('匿名时 requireAuth 存储 pending-action 并打开登录弹层', async () => {
  const { initAuth, requireAuth, auth } = await import('./auth.svelte');
  void initAuth();
  // 等 boot 完成（checkAuthStatus → anonymous）
  await new Promise((r) => setTimeout(r, 10));

  const run = vi.fn();
  const allowed = requireAuth({ action: '点赞', targetId: 'm1', run });
  expect(allowed).toBe(false);
  expect(auth.loginPromptOpen).toBe(true);
  expect(sessionStorage.getItem('hotspot:pending-action')).toContain('点赞');
});

it('登录成功后重放 pending action', async () => {
  const { initAuth, requireAuth, login, auth } = await import('./auth.svelte');
  void initAuth();
  await new Promise((r) => setTimeout(r, 10));

  const run = vi.fn();
  requireAuth({ action: '发帖', run });
  vi.mocked(api.login).mockResolvedValue({ code: 0, data: { client_type: 'web' } });
  const res = await login({ login_id: 'u1', password: 'x' });
  expect(res.code).toBe(0);
  expect(auth.status).toBe('authenticated');
  // 重放已发生：弹层关闭、pending 清空、run 被调用
  expect(auth.loginPromptOpen).toBe(false);
  expect(auth.pendingAction).toBeNull();
  await new Promise((r) => setTimeout(r, 10));
  expect(run).toHaveBeenCalledTimes(1);
});

it('已认证时 requireAuth 直接放行', async () => {
  const { initAuth, requireAuth, login, auth } = await import('./auth.svelte');
  void initAuth();
  await new Promise((r) => setTimeout(r, 10));
  vi.mocked(api.login).mockResolvedValue({ code: 0, data: { client_type: 'web' } });
  await login({ login_id: 'u1', password: 'x' });
  expect(auth.status).toBe('authenticated');

  const run = vi.fn();
  expect(requireAuth({ action: '关注', run })).toBe(true);
  expect(auth.loginPromptOpen).toBe(false);
  run();
  expect(run).toHaveBeenCalledTimes(1);
});

it('auth-failed 事件降级为 anonymous', async () => {
  const { initAuth, auth } = await import('./auth.svelte');
  void initAuth();
  await new Promise((r) => setTimeout(r, 10));
  vi.mocked(api.login).mockResolvedValue({ code: 0, data: { client_type: 'web' } });
  const { login } = await import('./auth.svelte');
  await login({ login_id: 'u1', password: 'x' });
  expect(auth.status).toBe('authenticated');

  window.dispatchEvent(new CustomEvent('auth-failed'));
  expect(auth.status).toBe('anonymous');
  expect(auth.user).toBeNull();
});
