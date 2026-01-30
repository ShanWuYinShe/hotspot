/**
 * API 客户端
 *
 * 特性：
 * - 所有请求使用 POST 方法
 * - 自动注入 Authorization Token
 * - 401 时自动刷新 Token 并重试
 * - 幂等性支持 (Idempotency-Key)
 * - 业务成功判断: code === 0
 */

import type { ApiResponse, PaginatedData } from '$lib/types/api';
import { AUTH } from './endpoints';

// Token 无效或过期的错误码
const TOKEN_INVALID_CODES = [10101, 10102, 10103, 10104, 10106];

// 是否正在刷新 Token
let isRefreshing = false;
// 等待刷新 Token 完成的 Promise
let refreshPromise: Promise<boolean> | null = null;

export interface RequestOptions {
	/** 自定义请求头 */
	headers?: Record<string, string>;
	/** 是否跳过 Token 注入 */
	skipAuth?: boolean;
	/** 幂等键 */
	idempotencyKey?: string;
}

export interface ApiClientConfig {
	/** API 基础 URL */
	baseUrl: string;
	/** 获取 Token 的函数 */
	getToken: () => string | null;
	/** 设置 Token 的函数 */
	setToken: (token: string, expiresAt: string) => void;
	/** 清除 Token 的函数 */
	clearToken: () => void;
	/** 获取 Refresh Token 的函数 (App 端使用) */
	getRefreshToken?: () => string | null;
}

let config: ApiClientConfig | null = null;

/**
 * 初始化 API 客户端配置
 */
export function initApiClient(clientConfig: ApiClientConfig) {
	config = clientConfig;
}

/**
 * 获取配置
 */
function getConfig(): ApiClientConfig {
	if (!config) {
		throw new Error('API client not initialized. Call initApiClient first.');
	}
	return config;
}

/**
 * 检查是否为 Token 无效错误
 */
function isTokenInvalidError(code: number): boolean {
	return TOKEN_INVALID_CODES.includes(code);
}

/**
 * 刷新 Token
 */
async function refreshToken(): Promise<boolean> {
	const cfg = getConfig();

	try {
		const response = await fetch(`${cfg.baseUrl}${AUTH.REFRESH_TOKEN}`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			credentials: 'include', // 携带 HttpOnly Cookie
			body: JSON.stringify({
				client_type: 'web',
				refresh_token: cfg.getRefreshToken?.() // App 端需要传递
			})
		});

		const data = (await response.json()) as ApiResponse<{
			access_token: string;
			access_token_expires_at: string;
		}>;

		if (data.code === 0 && data.data) {
			cfg.setToken(data.data.access_token, data.data.access_token_expires_at);
			return true;
		}

		// 刷新失败，清除 Token
		cfg.clearToken();
		return false;
	} catch {
		cfg.clearToken();
		return false;
	}
}

/**
 * 确保刷新 Token（防止并发刷新）
 */
async function ensureRefreshToken(): Promise<boolean> {
	if (isRefreshing) {
		return refreshPromise ?? Promise.resolve(false);
	}

	isRefreshing = true;
	refreshPromise = refreshToken().finally(() => {
		isRefreshing = false;
		refreshPromise = null;
	});

	return refreshPromise;
}

/**
 * 核心请求方法
 */
async function request<T>(
	url: string,
	body?: unknown,
	options: RequestOptions = {},
	isRetry = false
): Promise<ApiResponse<T>> {
	const cfg = getConfig();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...options.headers
	};

	// 注入 Token
	if (!options.skipAuth) {
		const token = cfg.getToken();
		if (token) {
			headers['Authorization'] = `Bearer ${token}`;
		}
	}

	// 幂等键
	if (options.idempotencyKey) {
		headers['Idempotency-Key'] = options.idempotencyKey;
	}

	try {
		const response = await fetch(`${cfg.baseUrl}${url}`, {
			method: 'POST',
			headers,
			credentials: 'include',
			body: body ? JSON.stringify(body) : undefined
		});

		if (!response.ok) {
			return {
				code: -1,
				message: `HTTP Error: ${response.status}`,
				request_id: ''
			};
		}

		const data = (await response.json()) as ApiResponse<T>;

		// Token 过期处理（仅在非重试且有 Token 时）
		if (!isRetry && !options.skipAuth && cfg.getToken() && isTokenInvalidError(data.code)) {
			const refreshed = await ensureRefreshToken();
			if (refreshed) {
				// 使用新 Token 重试
				return request<T>(url, body, options, true);
			}
		}

		return data;
	} catch (error) {
		// 网络错误
		return {
			code: -1,
			message: error instanceof Error ? error.message : '网络请求失败',
			request_id: ''
		};
	}
}

/**
 * POST 请求
 */
export async function post<T>(
	url: string,
	body?: unknown,
	options?: RequestOptions
): Promise<ApiResponse<T>> {
	return request<T>(url, body, options);
}

/**
 * 分页查询
 */
export async function paginate<T>(
	url: string,
	params: { limit?: number; cursor?: string } & Record<string, unknown> = {},
	options?: RequestOptions
): Promise<ApiResponse<PaginatedData<T>>> {
	return post<PaginatedData<T>>(url, params, options);
}

/**
 * 检查响应是否成功
 */
export function isSuccess<T>(response: ApiResponse<T>): response is ApiResponse<T> & { data: T } {
	return response.code === 0;
}

/**
 * 提取响应数据（如果成功）
 */
export function getData<T>(response: ApiResponse<T>): T | undefined {
	return isSuccess(response) ? response.data : undefined;
}

/**
 * API 错误类
 */
export class ApiError extends Error {
	code: number;
	requestId: string;

	constructor(response: ApiResponse<unknown>) {
		super(response.message || '操作失败');
		this.name = 'ApiError';
		this.code = response.code;
		this.requestId = response.request_id;
	}
}

/**
 * 断言响应成功，否则抛出错误
 */
export function assertSuccess<T>(response: ApiResponse<T>): asserts response is ApiResponse<T> & { data: T } {
	if (!isSuccess(response)) {
		throw new ApiError(response);
	}
}
