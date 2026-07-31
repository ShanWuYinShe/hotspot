// API client 核心行为测试：envelope 解析、10100 去重刷新重试、10201 封禁事件。
import { describe, it, expect, vi, afterEach } from 'vitest';
import * as client from './client';

const BASE = 'https://test.local';

function mockFetchOnce(json: unknown, ok = true) {
  return vi.fn().mockResolvedValue({
    ok,
    json: async () => json,
  });
}

function dispatchCalls(fn: ReturnType<typeof vi.fn>) {
  return fn.mock.calls.map(([url]) => url);
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('request envelope', () => {
  it('code=0 时返回 data', async () => {
    const fetchMock = mockFetchOnce({ code: 0, data: { ok: true } });
    vi.stubGlobal('fetch', fetchMock);
    client.setApiBaseUrl(BASE);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(0);
    expect(res.data).toEqual({ ok: true });
  });

  it('code!==0 时透传 envelope（不抛异常）', async () => {
    const fetchMock = mockFetchOnce({ code: 10000, message: '参数错误' });
    vi.stubGlobal('fetch', fetchMock);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(10000);
    expect(res.message).toBe('参数错误');
  });

  it('HTTP 非 200 时抛出传输错误', async () => {
    const fetchMock = mockFetchOnce({ message: 'boom' }, false);
    vi.stubGlobal('fetch', fetchMock);

    await expect(client.checkAuthStatus()).rejects.toThrow('boom');
  });
});

describe('10100 刷新重试', () => {
  it('刷新恰好一次并重试原请求成功', async () => {
    // 第一次原请求 → 10100；refresh_token → code 0；重试原请求 → code 0
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100, message: 'auth expired' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { refreshed: true } }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { ok: true } }) });
    vi.stubGlobal('fetch', fetchMock);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(0);
    const urls = dispatchCalls(fetchMock);
    expect(urls).toHaveLength(3);
    // 第二次调用必须是 refresh_token 端点
    expect(urls[1]).toContain('/api/v1/auth/refresh_token');
  });

  it('并发 10100 去重：多个请求共享一次刷新', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100 }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100 }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: {} }), })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: {} }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: {} }) });
    vi.stubGlobal('fetch', fetchMock);

    await Promise.all([client.checkAuthStatus(), client.getClientConfig()]);
    const urls = dispatchCalls(fetchMock);
    // 2 次原请求 + 1 次共享刷新 + 2 次重试
    expect(urls.filter((u) => u.includes('refresh_token'))).toHaveLength(1);
  });

  it('刷新失败时派发 auth-failed 事件', async () => {
    const onFailed = vi.fn();
    window.addEventListener('auth-failed', onFailed);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100 }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100, message: 'refresh expired' }) });
    vi.stubGlobal('fetch', fetchMock);

    await client.checkAuthStatus();
    expect(onFailed).toHaveBeenCalledTimes(1);
    window.removeEventListener('auth-failed', onFailed);
  });
});

describe('10201 封禁事件', () => {
  it('code=10201 时派发 auth-banned 并返回 envelope', async () => {
    const onBanned = vi.fn();
    window.addEventListener('auth-banned', onBanned);
    const fetchMock = mockFetchOnce({ code: 10201, message: '账号已被封禁' });
    vi.stubGlobal('fetch', fetchMock);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(10201);
    expect(onBanned).toHaveBeenCalledTimes(1);
    const detail = (onBanned.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.message).toBe('账号已被封禁');
    window.removeEventListener('auth-banned', onBanned);
  });
});
