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

class ApiClient {
  private async request<T>(
    path: string,
    options: RequestInit = {},
  ): Promise<ApiResponse<T>> {
    const url = `${BASE_URL}${path}`;

    const res = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
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

  async get<T>(path: string): Promise<ApiResponse<T>> {
    // Backend uses POST only, but for local dev/proxy we map GET too
    return this.post<T>(path);
  }

  isSuccess(res: ApiResponse<unknown>): boolean {
    return res.code === 0;
  }
}

export const api = new ApiClient();
