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

  it('10500 限流按 Retry-After 等待并重试一次', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, status: 200, headers: new Headers({ 'Retry-After': '0' }), json: async () => ({ code: 10500, message: '请求过于频繁' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { ok: true } }) });
    vi.stubGlobal('fetch', fetchMock);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(0);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('10500 重试后仍限流则透传 envelope（不无限重试）', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200, headers: new Headers({ 'Retry-After': '0' }), json: async () => ({ code: 10500, message: '请求过于频繁' }) });
    vi.stubGlobal('fetch', fetchMock);

    const res = await client.checkAuthStatus();
    expect(res.code).toBe(10500);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('网关 502 返回 HTML（非 JSON）时抛 HTTP 状态错误而非裸解析异常', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => {
        throw new SyntaxError('Unexpected token < in JSON');
      },
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(client.checkAuthStatus()).rejects.toThrow('HTTP 502');
  });

  it('200 但响应体非 JSON 时按可重试传输错误处理', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => {
        throw new SyntaxError('Unexpected token < in JSON');
      },
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(client.checkAuthStatus()).rejects.toThrow('响应格式异常');
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
    expect(urls[1]).toContain('/public/v1/auth/refresh_token');
  });

  it('刷新请求体符合 spec（client_type 必填，Web 的 refresh_token 为空）', async () => {
    // 回归：曾发 body '{}'，服务端恒回 10000，所有 Web 会话 15 分钟静默死亡。
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100, message: 'auth expired' }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: {} }) })
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { ok: true } }) });
    vi.stubGlobal('fetch', fetchMock);

    await client.checkAuthStatus();
    const refreshCall = fetchMock.mock.calls.find(([url]) => String(url).includes('refresh_token'));
    expect(refreshCall).toBeTruthy();
    const body = JSON.parse(String((refreshCall![1] as RequestInit).body));
    expect(body.client_type).toBe('web');
    expect(body.refresh_token ?? null).toBeNull();
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

  it('刷新传输失败（后端不可达）时不锁存、抛可重试错误', async () => {
    const onFailed = vi.fn();
    window.addEventListener('auth-failed', onFailed);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 10100 }) })
      .mockRejectedValueOnce(new Error('fetch failed'));
    vi.stubGlobal('fetch', fetchMock);

    await expect(client.checkAuthStatus()).rejects.toThrow('网络异常');
    expect(onFailed).not.toHaveBeenCalled();
    window.removeEventListener('auth-failed', onFailed);
  });

describe('主动续期', () => {
    afterEach(() => {
      client.setKnownAccessExpiry(null);
    });

    it('临期请求先静默刷新、重学过期后再发原请求', async () => {
      client.setApiBaseUrl(BASE);
      client.setKnownAccessExpiry(new Date(Date.now() + 60_000).toISOString());
      const freshExpiry = new Date(Date.now() + 15 * 60_000).toISOString();
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: {} }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { user_id: 'u1', expires_at: freshExpiry } }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { ok: true } }) });
      vi.stubGlobal('fetch', fetchMock);

      const res = await client.checkAuthStatus();
      expect(res.code).toBe(0);
      const urls = dispatchCalls(fetchMock);
      expect(urls).toHaveLength(3);
      expect(urls[0]).toContain('refresh_token');
      expect(urls[1]).toContain('auth/session');
      expect(urls[2]).toContain('auth/session');
    });

    it('未临期不触发刷新', async () => {
      client.setApiBaseUrl(BASE);
      client.setKnownAccessExpiry(new Date(Date.now() + 15 * 60_000).toISOString());
      const fetchMock = mockFetchOnce({ code: 0, data: { ok: true } });
      vi.stubGlobal('fetch', fetchMock);

      await client.checkAuthStatus();
      const urls = dispatchCalls(fetchMock);
      expect(urls).toHaveLength(1);
      expect(urls[0]).not.toContain('refresh_token');
    });

    it('主动刷新传输失败不阻塞原请求、不派发登出', async () => {
      const onFailed = vi.fn();
      window.addEventListener('auth-failed', onFailed);
      client.setApiBaseUrl(BASE);
      client.setKnownAccessExpiry(new Date(Date.now() + 60_000).toISOString());
      const fetchMock = vi
        .fn()
        .mockRejectedValueOnce(new Error('fetch failed'))
        .mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, data: { ok: true } }) });
      vi.stubGlobal('fetch', fetchMock);

      const res = await client.checkAuthStatus();
      expect(res.code).toBe(0);
      expect(onFailed).not.toHaveBeenCalled();
      window.removeEventListener('auth-failed', onFailed);
    });
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

describe('幂等键与认证头', () => {
  it('newIdempotencyKey 返回非空字符串', () => {
    expect(typeof client.newIdempotencyKey()).toBe('string');
    expect(client.newIdempotencyKey().length).toBeGreaterThan(0);
  });

  it('register/guest/createGroup 携带 Idempotency-Key', async () => {
    for (const call of [client.register, client.guest, client.createGroup]) {
      const fetchMock = mockFetchOnce({ code: 0, data: {} });
      vi.stubGlobal('fetch', fetchMock);
      client.setApiBaseUrl(BASE);
      // createGroup 为强类型签名（spec: groups/create），其余为宽松 object
      const params = call === client.createGroup
        ? { name: '测试群', location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '徐州' } }
        : {};
      await call(params as never);
      const headers = fetchMock.mock.calls[0][1].headers;
      expect(typeof headers['Idempotency-Key']).toBe('string');
      expect(headers['Idempotency-Key'].length).toBeGreaterThan(0);
      vi.unstubAllGlobals();
    }
  });

  it('login 不携带 Idempotency-Key（spec 未要求）', async () => {
    const fetchMock = mockFetchOnce({ code: 0, data: {} });
    vi.stubGlobal('fetch', fetchMock);
    client.setApiBaseUrl(BASE);
    await client.login({ login_id: 'u', password: 'p', client_type: 'web', device_id: 'd' });
    const headers = fetchMock.mock.calls[0][1].headers;
    expect(headers['Idempotency-Key']).toBeUndefined();
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
