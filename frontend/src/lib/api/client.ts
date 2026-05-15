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
      'Content-Type': 'application/json',
      'Idempotency-Key': generateIdempotencyKey(),
      ...(options.headers as Record<string, string> || {}),
    };

    const res = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers,
      ...options,
    });

    if (!res.ok) {
      return {
        code: res.status,
        message: `HTTP ${res.status}`,
        data: null as T,
      };
    }

    return res.json();
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
