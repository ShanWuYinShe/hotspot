/**
 * 服务端 API 客户端
 * 用于 Server Components 和 Server Actions 的数据获取
 */

import { cookies, headers } from 'next/headers';
import type { ApiResponse, PaginatedData } from '@/types/api';

// ============================================================================
// 配置
// ============================================================================

const ACCESS_TOKEN_KEY = 'access_token';
async function getApiBaseUrl(): Promise<string> {
  const headerStore = await headers();
  const host = headerStore.get('x-forwarded-host') ?? headerStore.get('host') ?? 'localhost';
  const proto = headerStore.get('x-forwarded-proto') ?? 'http';

  return `${proto}://${host}`;
}


// ============================================================================
// 辅助函数
// ============================================================================

/**
 * 从 cookies 获取 access token
 */
async function getAccessToken(): Promise<string | null> {
  const cookieStore = await cookies();
  return cookieStore.get(ACCESS_TOKEN_KEY)?.value ?? null;
}

/**
 * 构建请求 URL
 */
async function buildUrl(
  endpoint: string,
  params?: Record<string, string | number | boolean | undefined>
): Promise<string> {
  const url = new URL(endpoint, await getApiBaseUrl());
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        url.searchParams.append(key, String(value));
      }
    });
  }
  return url.toString();
}

// ============================================================================
// 请求方法
// ============================================================================

interface ServerRequestOptions {
  params?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
  requireAuth?: boolean;
  cache?: RequestCache;
  revalidate?: number | false;
  tags?: string[];
}

/**
 * 服务端 HTTP 请求
 */
async function serverRequest<T>(
  endpoint: string,
  method: string,
  options: ServerRequestOptions = {}
): Promise<ApiResponse<T>> {
  const { params, body, requireAuth = true, cache, revalidate, tags } = options;

  const url = await buildUrl(endpoint, params);
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  // 添加认证 Token
  if (requireAuth) {
    const token = await getAccessToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  const fetchOptions: RequestInit = {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  };

  // Next.js 缓存策略
  if (cache) {
    fetchOptions.cache = cache;
  }
  if (revalidate !== undefined || tags) {
    (fetchOptions as { next?: { revalidate?: number | false; tags?: string[] } }).next = {
      ...(revalidate !== undefined && { revalidate }),
      ...(tags && { tags }),
    };
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10000);

  try {
    const response = await fetch(url, { ...fetchOptions, signal: controller.signal });
    const data: ApiResponse<T> = await response.json();
    return data;
  } catch (error) {
    const message =
      error instanceof Error && error.name === 'AbortError'
        ? 'Request timeout'
        : error instanceof Error
          ? error.message
          : 'Network error';
    return {
      code: -1,
      message,
      request_id: '',
      data: null as T,
    };
  } finally {
    clearTimeout(timeoutId);
  }
}

// ============================================================================
// 导出的服务端 API 方法
// ============================================================================

export const serverApi = {
  /**
   * GET 请求
   */
  get<T>(
    endpoint: string,
    params?: Record<string, string | number | boolean | undefined>,
    options?: Omit<ServerRequestOptions, 'params' | 'body'>
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, 'GET', { ...options, params });
  },

  /**
   * POST 请求
   */
  post<T>(
    endpoint: string,
    body?: unknown,
    options?: Omit<ServerRequestOptions, 'body'>
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, 'POST', { ...options, body });
  },

  /**
   * PUT 请求
   */
  put<T>(
    endpoint: string,
    body?: unknown,
    options?: Omit<ServerRequestOptions, 'body'>
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, 'PUT', { ...options, body });
  },

  /**
   * PATCH 请求
   */
  patch<T>(
    endpoint: string,
    body?: unknown,
    options?: Omit<ServerRequestOptions, 'body'>
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, 'PATCH', { ...options, body });
  },

  /**
   * DELETE 请求
   */
  delete<T>(
    endpoint: string,
    options?: ServerRequestOptions
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, 'DELETE', options);
  },

  /**
   * 分页 GET 请求
   */
  paginate<T>(
    endpoint: string,
    params?: Record<string, string | number | boolean | undefined> & {
      cursor?: string;
      limit?: number;
    },
    options?: Omit<ServerRequestOptions, 'params' | 'body'>
  ): Promise<ApiResponse<PaginatedData<T>>> {
    return serverRequest<PaginatedData<T>>(endpoint, 'GET', { ...options, params });
  },
};

/**
 * 检查 API 响应是否成功
 */
export function isServerSuccess<T>(
  response: ApiResponse<T>
): response is ApiResponse<T> & { data: T } {
  return response.code === 0;
}

/**
 * 提取响应数据，失败时返回 null
 */
export function unwrapOrNull<T>(response: ApiResponse<T>): T | null {
  return response.code === 0 ? (response.data ?? null) : null;
}
