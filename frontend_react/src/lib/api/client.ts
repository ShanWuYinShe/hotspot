/**
 * 统一 HTTP 客户端
 * 负责所有 API 请求的发送、Token 注入、错误处理
 */

import type { ApiResponse, PaginatedData } from '@/types/api';

// ============================================================================
// 类型定义
// ============================================================================

export interface RequestOptions extends Omit<RequestInit, 'body'> {
  params?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
  requireAuth?: boolean;
  idempotencyKey?: string;
}

export interface ApiClientConfig {
  baseUrl?: string;
  getAccessToken: () => string | null;
  onUnauthorized: () => void;
  onRefreshToken: () => Promise<string | null>;
}

// ============================================================================
// API 错误类
// ============================================================================

export class ApiError extends Error {
  constructor(
    public code: number,
    message: string,
    public requestId?: string,
    public data?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }

  static fromResponse(response: ApiResponse<unknown>): ApiError {
    return new ApiError(
      response.code,
      response.message ?? 'Unknown error',
      response.request_id,
      response.data
    );
  }
}

// ============================================================================
// API 客户端类
// ============================================================================

let clientConfig: ApiClientConfig | null = null;

export function configureApiClient(config: ApiClientConfig): void {
  clientConfig = config;
}

function getConfig(): ApiClientConfig {
  if (!clientConfig) {
    throw new Error('API client not configured. Call configureApiClient first.');
  }
  return clientConfig;
}

/**
 * 构建请求 URL
 */
function buildRelativeUrl(
  endpoint: string,
  params?: Record<string, string | number | boolean | undefined>
): string {
  const parts = endpoint.split('?');
  const path = parts[0] ?? endpoint;
  const query = parts[1] ?? '';
  const searchParams = new URLSearchParams(query);

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, String(value));
      }
    });
  }

  const queryString = searchParams.toString();
  return queryString ? `${path}?${queryString}` : path;
}

function buildUrl(endpoint: string, params?: Record<string, string | number | boolean | undefined>): string {
  const config = getConfig();
  const isAbsolute = /^https?:\/\//i.test(endpoint);
  const baseUrl = config.baseUrl ?? '';

  if (isAbsolute) {
    const url = new URL(endpoint);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }
    return url.toString();
  }

  if (baseUrl) {
    const url = new URL(endpoint, baseUrl);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }
    return url.toString();
  }

  return buildRelativeUrl(endpoint, params);
}

/**
 * 构建请求头
 */
function buildHeaders(
  options: RequestOptions,
  accessToken: string | null
): Headers {
  const headers = new Headers(options.headers);

  // Content-Type
  if (options.body && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  // Authorization
  if (options.requireAuth !== false && accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }

  // Idempotency-Key
  if (options.idempotencyKey) {
    headers.set('X-Idempotency-Key', options.idempotencyKey);
  }

  return headers;
}

/**
 * 发送 HTTP 请求
 */
async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<ApiResponse<T>> {
  const config = getConfig();
  const accessToken = config.getAccessToken();

  const url = buildUrl(endpoint, options.params);
  const headers = buildHeaders(options, accessToken);

  const fetchOptions: RequestInit = {
    ...options,
    headers,
    body: options.body instanceof FormData 
      ? options.body 
      : options.body 
        ? JSON.stringify(options.body) 
        : undefined,
  };

  try {
    const response = await fetch(url, fetchOptions);
    const data: ApiResponse<T> = await response.json();

    // 处理 401 未授权
    if (response.status === 401 && options.requireAuth !== false) {
      // 尝试刷新 Token
      const newToken = await config.onRefreshToken();
      if (newToken) {
        // 重试请求
        return request<T>(endpoint, options);
      } else {
        // 刷新失败，触发未授权回调
        config.onUnauthorized();
        throw new ApiError(10102, 'Unauthorized', data.request_id);
      }
    }

    return data;
  } catch (error) {
    if (error instanceof ApiError) {
      throw error;
    }
    // 网络错误或其他错误
    throw new ApiError(
      -1,
      error instanceof Error ? error.message : 'Network error'
    );
  }
}

// ============================================================================
// 便捷方法
// ============================================================================

/**
 * GET 请求
 */
export async function get<T>(
  endpoint: string,
  params?: Record<string, string | number | boolean | undefined>,
  options?: Omit<RequestOptions, 'method' | 'body' | 'params'>
): Promise<ApiResponse<T>> {
  return request<T>(endpoint, {
    ...options,
    method: 'GET',
    params,
  });
}

/**
 * POST 请求
 */
export async function post<T>(
  endpoint: string,
  body?: unknown,
  options?: Omit<RequestOptions, 'method' | 'body'>
): Promise<ApiResponse<T>> {
  return request<T>(endpoint, {
    ...options,
    method: 'POST',
    body,
  });
}

/**
 * PUT 请求
 */
export async function put<T>(
  endpoint: string,
  body?: unknown,
  options?: Omit<RequestOptions, 'method' | 'body'>
): Promise<ApiResponse<T>> {
  return request<T>(endpoint, {
    ...options,
    method: 'PUT',
    body,
  });
}

/**
 * PATCH 请求
 */
export async function patch<T>(
  endpoint: string,
  body?: unknown,
  options?: Omit<RequestOptions, 'method' | 'body'>
): Promise<ApiResponse<T>> {
  return request<T>(endpoint, {
    ...options,
    method: 'PATCH',
    body,
  });
}

/**
 * DELETE 请求
 */
export async function del<T>(
  endpoint: string,
  options?: Omit<RequestOptions, 'method'>
): Promise<ApiResponse<T>> {
  return request<T>(endpoint, {
    ...options,
    method: 'DELETE',
  });
}

// ============================================================================
// 辅助函数
// ============================================================================

/**
 * 检查 API 响应是否成功
 * 业务成功判定：code === 0
 */
export function isSuccess<T>(response: ApiResponse<T>): response is ApiResponse<T> & { data: T } {
  return response.code === 0;
}

/**
 * 提取响应数据，失败时抛出错误
 */
export function unwrap<T>(response: ApiResponse<T>): T {
  if (isSuccess(response)) {
    return response.data;
  }
  throw ApiError.fromResponse(response);
}

/**
 * 分页请求辅助函数
 */
export async function paginate<T>(
  endpoint: string,
  params?: Record<string, string | number | boolean | undefined> & { cursor?: string; limit?: number },
  options?: Omit<RequestOptions, 'method' | 'body' | 'params'>
): Promise<ApiResponse<PaginatedData<T>>> {
  return get<PaginatedData<T>>(endpoint, params, options);
}

/**
 * 生成幂等性 Key
 */
export function generateIdempotencyKey(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}
