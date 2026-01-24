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
const SERVER_API_ORIGIN = process.env.HOTSPOT_ORIGIN ?? process.env.NEXT_PUBLIC_APP_ORIGIN ?? '';
const AUTH_ERROR_CODE_MIN = 10100;
const AUTH_ERROR_CODE_MAX = 10107;
const NOT_FOUND_CODE_MIN = 10300;
const NOT_FOUND_CODE_MAX = 10399;

function normalizeForwardedHeader(value: string | null): string | null {
  if (!value) return null;
  const first = value.split(',')[0]?.trim();
  return first ? first : null;
}

function normalizeOrigin(value: string): string | null {
  try {
    return new URL(value).origin;
  } catch {
    return null;
  }
}

async function getApiBaseUrl(): Promise<string> {
  const envOrigin = SERVER_API_ORIGIN ? normalizeOrigin(SERVER_API_ORIGIN) : null;
  if (envOrigin) return envOrigin;

  const headerStore = await headers();
  const originHeader = normalizeForwardedHeader(headerStore.get('origin'));
  if (originHeader) {
    const normalized = normalizeOrigin(originHeader);
    if (normalized) return normalized;
  }

  const host =
    normalizeForwardedHeader(headerStore.get('x-forwarded-host')) ??
    normalizeForwardedHeader(headerStore.get('host')) ??
    'localhost';
  const proto = normalizeForwardedHeader(headerStore.get('x-forwarded-proto')) ?? 'http';

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
async function buildUrl(endpoint: string): Promise<string> {
  const url = new URL(endpoint, await getApiBaseUrl());
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
  options: ServerRequestOptions = {}
): Promise<ApiResponse<T>> {
  const { params, body, requireAuth = true, cache, revalidate, tags } = options;

  const url = await buildUrl(endpoint);
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

  const payload = body === undefined ? (params ?? {}) : body;
  const fetchOptions: RequestInit = {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
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
   * POST 请求
   */
  post<T>(
    endpoint: string,
    body?: unknown,
    options?: Omit<ServerRequestOptions, 'body'>
  ): Promise<ApiResponse<T>> {
    return serverRequest<T>(endpoint, { ...options, body });
  },

  /**
   * 分页 POST 请求
   */
  paginate<T>(
    endpoint: string,
    params?: Record<string, string | number | boolean | undefined> & {
      cursor?: string;
      limit?: number;
    },
    options?: Omit<ServerRequestOptions, 'params' | 'body'>
  ): Promise<ApiResponse<PaginatedData<T>>> {
    return serverRequest<PaginatedData<T>>(endpoint, { ...options, params });
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

export function isAuthError(response: ApiResponse<unknown>): boolean {
  return response.code >= AUTH_ERROR_CODE_MIN && response.code <= AUTH_ERROR_CODE_MAX;
}

export function isNotFoundError(response: ApiResponse<unknown>): boolean {
  return response.code >= NOT_FOUND_CODE_MIN && response.code <= NOT_FOUND_CODE_MAX;
}

export function getServerErrorMessage(response: ApiResponse<unknown>): string {
  const message = response.message?.trim();
  if (response.code === -1) {
    if (message?.toLowerCase().includes('timeout')) {
      return '请求超时，请稍后再试。';
    }
    return '网络异常，请检查连接后重试。';
  }
  if (message) {
    return message;
  }
  return '服务暂时不可用，请稍后再试。';
}

/**
 * 提取响应数据，失败时返回 null
 */
export function unwrapOrNull<T>(response: ApiResponse<T>): T | null {
  return response.code === 0 ? (response.data ?? null) : null;
}
