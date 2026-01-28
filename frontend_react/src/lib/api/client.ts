/**
 * HotSpot API 客户端
 * 统一的 HTTP 请求处理
 */

import type { ApiResponse } from '@/types/api';
import { generateIdempotencyKey } from '@/lib/utils';
import { 
  getAccessToken, 
  setAccessToken, 
  getRefreshToken, 
  setRefreshToken, 
  clearTokens 
} from '@/stores/auth';

// API 基础 URL
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || '';

/**
 * 请求配置
 */
interface RequestConfig {
  /** 是否需要认证（默认 true） */
  requireAuth?: boolean;
  /** 是否需要幂等性 key */
  idempotent?: boolean;
  /** 自定义 headers */
  headers?: Record<string, string>;
  /** 请求超时（毫秒） */
  timeout?: number;
}

/**
 * 判断 API 响应是否成功
 * 根据 API 规范，code === 0 表示成功
 */
export function isSuccess<T>(response: ApiResponse<T>): response is ApiResponse<T> & { data: T } {
  return response.code === 0;
}

/**
 * 统一 API 请求 - 需要认证
 * 所有请求使用 POST 方法
 */
export async function apiRequest<T>(
  endpoint: string,
  body?: unknown,
  config: RequestConfig = {}
): Promise<ApiResponse<T>> {
  const {
    requireAuth = true,
    idempotent = false,
    headers = {},
    timeout = 30000,
  } = config;

  // 构建请求头
  const requestHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...headers,
  };

  // 添加认证 token
  if (requireAuth) {
    const token = getAccessToken();
    if (token) {
      requestHeaders['Authorization'] = `Bearer ${token}`;
    }
  }

  // 添加幂等性 key
  if (idempotent) {
    requestHeaders['Idempotency-Key'] = generateIdempotencyKey();
  }

  // 创建 AbortController 用于超时
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      method: 'POST',
      headers: requestHeaders,
      body: body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    // 解析响应
    const data = await response.json() as ApiResponse<T>;

    // 处理 token 过期 (10101: Token无效或过期, 10102: 需要登录)
    if (data.code === 10101 || data.code === 10102) {
      // Token 过期，尝试刷新
      const refreshed = await refreshAccessToken();
      if (refreshed) {
        // 重试请求
        return apiRequest<T>(endpoint, body, config);
      }
      // 刷新失败，清除 token
      clearTokens();
    }

    return data;
  } catch (error) {
    clearTimeout(timeoutId);

    if (error instanceof Error) {
      if (error.name === 'AbortError') {
        return {
          code: -1,
          message: '请求超时',
          request_id: '',
        };
      }
      return {
        code: -1,
        message: error.message || '网络错误',
        request_id: '',
      };
    }

    return {
      code: -1,
      message: '未知错误',
      request_id: '',
    };
  }
}

/**
 * 刷新 access token
 */
async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh_token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        client_type: 'web',
        refresh_token: refreshToken,
      }),
    });

    const data = await response.json() as ApiResponse<{
      access_token: string;
      refresh_token?: string;
      access_token_expires_at: string;
    }>;

    if (data.code === 0 && data.data) {
      setAccessToken(data.data.access_token);
      if (data.data.refresh_token) {
        setRefreshToken(data.data.refresh_token);
      }
      return true;
    }
  } catch {
    // 刷新失败
  }

  return false;
}

/**
 * 公开 API 请求 - 不需要认证（Token 可选）
 * 如果有 token 会自动附加，但不强制要求
 */
export async function apiRequestPublic<T>(
  endpoint: string,
  body?: unknown,
  config: Omit<RequestConfig, 'requireAuth'> = {}
): Promise<ApiResponse<T>> {
  // 公开接口也可以附加 token（获取额外字段）
  const token = getAccessToken();
  const headers = config.headers || {};
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return apiRequest<T>(endpoint, body, { 
    ...config, 
    requireAuth: false,
    headers,
  });
}

/**
 * 需要幂等性的 API 请求
 */
export async function apiRequestIdempotent<T>(
  endpoint: string,
  body?: unknown,
  config: Omit<RequestConfig, 'idempotent'> = {}
): Promise<ApiResponse<T>> {
  return apiRequest<T>(endpoint, body, { ...config, idempotent: true });
}

// 导出 token 操作函数（兼容旧代码）
export { getAccessToken, setAccessToken, getRefreshToken, setRefreshToken, clearTokens };
