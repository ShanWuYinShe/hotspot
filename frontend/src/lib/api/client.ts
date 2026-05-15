const BASE_URL = '/api/v1';

interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedData<T> {
  items: T[];
  next_cursor: string | null;
  total: number;
}

// ── Idempotency Key ──────────────────────────────────────────────
let _clientId: string | null = null;

function getClientId(): string {
  if (!_clientId) {
    _clientId = crypto.randomUUID().replace(/-/g, '').substring(0, 8);
  }
  return _clientId;
}

function generateIdempotencyKey(): string {
  const clientId = getClientId();
  const timestamp = Date.now();
  const random = crypto.randomUUID().replace(/-/g, '').substring(0, 12);
  return `${clientId}-${timestamp}-${random}`;
}

// ── API Client ───────────────────────────────────────────────────
class ApiClient {
  private async request<T>(
    path: string,
    options: RequestInit = {},
  ): Promise<ApiResponse<T>> {
    const url = `${BASE_URL}${path}`;

    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'Idempotency-Key': generateIdempotencyKey(),
      ...(options.headers as Record<string, string> || {}),
    };

    try {
      const res = await fetch(url, {
        method: 'POST',
        credentials: 'include',
        headers,
        ...options,
      });

      // Always try to parse response body as JSON
      const body = await res.json().catch(() => null);

      if (!res.ok) {
        // Return backend's JSON error if available, else synthesize one
        if (body && typeof body.code === 'number') {
          return { code: body.code, message: body.message || `HTTP ${res.status}`, data: null as T };
        }
        return { code: res.status, message: `HTTP ${res.status}`, data: null as T };
      }

      // Backend always returns { code, message, data }
      if (body && typeof body.code === 'number') {
        return body as ApiResponse<T>;
      }

      // Fallback: wrap raw response
      return { code: 0, message: 'ok', data: body as T };
    } catch (err) {
      // Network error, timeout, DNS failure — always return JSON
      return {
        code: -1,
        message: err instanceof Error ? err.message : 'Network error',
        data: null as T,
      };
    }
  }

  async post<T>(path: string, body?: unknown): Promise<ApiResponse<T>> {
    return this.request<T>(path, {
      body: body ? JSON.stringify(body) : '{}',
    });
  }

  isSuccess(res: ApiResponse<unknown>): boolean {
    return res.code === 0;
  }
}

export const api = new ApiClient();
