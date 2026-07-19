/**
 * API client tests — covers the most critical paths.
 *
 * Not covered here (require deeper mock setup):
 *   - Refresh-token single-flight (refreshToken uses global fetch internally)
 *   - 10500 / 22002 retry-with-delay (requires fake timers + sleep sequence;
 *     those paths are implicitly exercised when the retry-loop conditions are met)
 */

import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  initApiClient,
  post,
  isSuccess,
  getData,
  generateIdempotencyKey,
  type ClientApiResponse
} from '$lib/api/client';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function setupClient(overrides?: Partial<Parameters<typeof initApiClient>[0]>) {
  initApiClient({
    baseUrl: 'https://api.example.com',
    onRefreshSuccess: vi.fn(),
    onRefreshFail: vi.fn(),
    ...overrides
  });
}

function jsonResponse(body: unknown, status = 200, headers?: Record<string, string>): Response {
  return new Response(JSON.stringify(body), { status, headers: new Headers(headers) });
}

function okResponse<T>(data: T): Response {
  return jsonResponse({ code: 0, message: 'ok', data });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('post', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('returns data on code 0', async () => {
    setupClient();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(okResponse({ items: [] })));

    const result = await post('/api/v1/test', {});

    expect(result.code).toBe(0);
    expect(result.data).toEqual({ items: [] });
  });

  it('returns error on HTTP 500', async () => {
    setupClient();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 500 })));

    const result = await post('/api/v1/test', {});

    expect(result.code).toBe(-1);
    expect(result.message).toContain('500');
  });

  it('returns error on JSON parse failure', async () => {
    setupClient();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('not json', { status: 200, statusText: 'OK' }))
    );

    const result = await post('/api/v1/test', {});

    // Client returns code=-1 for parse failures; message is i18n'd (中文: "响应解析失败: OK")
    expect(result.code).toBe(-1);
    expect(result.message).toBeTruthy();
  });

  it('returns network error on fetch rejection', async () => {
    setupClient();
    const err = new Error('connect ECONNREFUSED');
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(err));

    const result = await post('/api/v1/test', {});

    expect(result.code).toBe(-1);
    expect(result.message).toBe('connect ECONNREFUSED');
  });

  it('forwards custom headers', async () => {
    setupClient();
    const fetchSpy = vi.fn().mockResolvedValue(okResponse(null));
    vi.stubGlobal('fetch', fetchSpy);

    await post('/api/v1/test', {}, { headers: { 'X-Custom': 'value' } });

    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)['X-Custom']).toBe('value');
    expect(init.credentials).toBe('include');
  });

  it('forwards idempotency key as header', async () => {
    setupClient();
    const fetchSpy = vi.fn().mockResolvedValue(okResponse(null));
    vi.stubGlobal('fetch', fetchSpy);

    await post('/api/v1/test', {}, { idempotencyKey: 'key-123' });

    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)['Idempotency-Key']).toBe('key-123');
  });

  it('retries on 10500 rate-limit (honours retry logic structure)', async () => {
    setupClient();
    let callCount = 0;
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.resolve(
            jsonResponse({ code: 10500, message: 'rate limited' }, 200, { 'Retry-After': '1' })
          );
        }
        return Promise.resolve(okResponse({ done: true }));
      })
    );

    // Fake timers to fast-forward through the 1-second sleep() delay
    vi.useFakeTimers();
    const resultPromise = post('/api/v1/test', {});
    await vi.advanceTimersByTimeAsync(3000);
    const result = await resultPromise;
    vi.useRealTimers();

    expect(result.code).toBe(0);
    expect(result.data).toEqual({ done: true });
    expect(callCount).toBe(2);
  }, 8000); // fake timers need a higher timeout
});

describe('pure helpers', () => {
  it('isSuccess returns true for code 0', () => {
    expect(isSuccess({ code: 0, message: 'ok', data: {} })).toBe(true);
  });

  it('isSuccess returns false for non-zero code', () => {
    expect(isSuccess({ code: 10000, message: 'bad request' })).toBe(false);
  });

  it('isSuccess narrows T on truthy', () => {
    const resp: ClientApiResponse<string> = { code: 0, message: 'ok', data: 'hello' };
    if (isSuccess(resp)) {
      // TypeScript should infer resp.data is string here
      expect(resp.data.length).toBe(5);
    }
  });

  it('getData returns data on success', () => {
    const resp: ClientApiResponse<string> = { code: 0, message: 'ok', data: 'hello' };
    expect(getData(resp)).toBe('hello');
  });

  it('getData returns undefined on failure', () => {
    expect(getData({ code: 10000, message: 'bad request' })).toBeUndefined();
  });

  it('generateIdempotencyKey produces format {client_id}-{ts}-{random}', () => {
    const key = generateIdempotencyKey();
    expect(key).toMatch(/^[a-f0-9]{8}-\d{13}-[a-f0-9]{12}$/);
  });

  it('generateIdempotencyKey produces unique keys in a batch', () => {
    const keys = new Set(Array.from({ length: 50 }, () => generateIdempotencyKey()));
    expect(keys.size).toBe(50);
  });
});
