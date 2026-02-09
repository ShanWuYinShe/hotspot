/**
 * API 客户端
 *
 * 特性：
 * - 所有请求使用 POST 方法
 * - 自动注入 Authorization Token
 * - 自动注入 CSRF Token (Web 端)
 * - 401 时自动刷新 Token 并重试
 * - 幂等性支持 (Idempotency-Key)
 * - 业务成功判断: code === 0
 */

import { dev } from '$app/environment';
import type { ApiResponse, PaginatedData } from '$lib/types/api';
import { AUTH } from './endpoints';

// Re-export types for external use
export type { ApiResponse, PaginatedData };

// Token 无效或过期的错误码（spec 定义的宽泛分类码）
const TOKEN_INVALID_CODE = 10100;

// 是否正在刷新 Token
let isRefreshing = false;
// 等待刷新 Token 完成的 Promise
let refreshPromise: Promise<boolean> | null = null;

/**
 * 从 Cookie 中读取指定名称的值
 */
function getCookie(name: string): string | null {
	if (typeof document === 'undefined') return null;
	
	const cookies = document.cookie.split(';');
	for (const cookie of cookies) {
		const [key, value] = cookie.trim().split('=');
		if (key === name && value !== undefined) {
			return value;
		}
	}
	return null;
}

export interface RequestOptions {
	/** 自定义请求头 */
	headers?: Record<string, string>;
	/** 是否跳过 Token 注入 */
	skipAuth?: boolean;
	/** 幂等键 */
	idempotencyKey?: string;
	/** 请求超时时间（毫秒），默认 30000ms */
	timeout?: number;
	/** AbortSignal，用于取消请求 */
	signal?: AbortSignal;
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
 * spec 定义: code=10100 表示认证类错误（token 无效/过期/缺失等）
 */
function isTokenInvalidError(code: number): boolean {
	return code === TOKEN_INVALID_CODE;
}

/**
 * 刷新 Token
 */
async function refreshToken(): Promise<boolean> {
	const cfg = getConfig();

	try {
		// 读取 CSRF Token (Web 端需要)
		const csrfToken = getCookie('hs_csrf_token');
		
		// Web 端：不在 body 中传递 refresh_token（从 HttpOnly Cookie 中读取）
		// 严格遵循 @spec/api_spec_auth.mdc 规范：
		// - client_type='web'：请求体不得携带 refresh_token
		const headers: Record<string, string> = {
			'Content-Type': 'application/json'
		};
		
		// 添加 CSRF Token header (Web 端必需)
		if (csrfToken) {
			headers['X-CSRF-Token'] = csrfToken;
		}
		
		const response = await fetch(`${cfg.baseUrl}${AUTH.REFRESH_TOKEN}`, {
			method: 'POST',
			headers,
			credentials: 'include', // 携带 HttpOnly Cookie（包含 hs_refresh_token 和 hs_csrf_token）
			body: JSON.stringify({
				client_type: 'web'
				// ✅ 不添加 refresh_token 字段（Web 端从 Cookie 读取）
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
	} catch (error) {
		if (dev) console.error('[TokenRefresh] Token 刷新异常:', error);
		cfg.clearToken();
		return false;
	}
}

/**
 * 确保刷新 Token（防止并发刷新）
 * 导出供 TokenRefreshManager 共享同一个刷新锁
 */
export async function ensureRefreshToken(): Promise<boolean> {
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

	// 注入 CSRF Token (Web 端需要)
	// 对于需要 CSRF 保护的端点（如 refresh_token），自动添加
	const csrfToken = getCookie('hs_csrf_token');
	if (csrfToken) {
		headers['X-CSRF-Token'] = csrfToken;
	}

	// 幂等键
	if (options.idempotencyKey) {
		headers['Idempotency-Key'] = options.idempotencyKey;
	}

	// 设置超时控制
	const timeout = options.timeout ?? 30000; // 默认 30 秒
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), timeout);

	// 合并 signal（支持外部取消 + 内部超时）
	const signal = options.signal
		? combineSignals([options.signal, controller.signal])
		: controller.signal;

	try {
		const response = await fetch(`${cfg.baseUrl}${url}`, {
			method: 'POST',
			headers,
			credentials: 'include',
			body: body ? JSON.stringify(body) : undefined,
			signal
		});

		// 清除超时定时器
		clearTimeout(timeoutId);

		// 处理 HTTP 错误
		if (!response.ok) {
			let errorMessage = `HTTP Error: ${response.status}`;
			let requestId = response.headers.get('X-Request-ID') || '';
			
			// 尝试读取响应体中的错误信息
			try {
				const errorBody = await response.json();
				if (errorBody.message) errorMessage = errorBody.message;
				if (errorBody.request_id) requestId = errorBody.request_id;
			} catch {
				// 忽略响应体解析错误
			}
			
			return {
				code: -1,
				message: errorMessage,
				request_id: requestId
			};
		}

		// 安全解析 JSON
		let data: ApiResponse<T>;
		try {
			data = (await response.json()) as ApiResponse<T>;
		} catch (parseError) {
			return {
				code: -1,
				message: `响应解析失败: ${response.statusText}`,
				request_id: response.headers.get('X-Request-ID') || ''
			};
		}

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
		// 清除超时定时器
		clearTimeout(timeoutId);

		// 处理取消和超时错误
		if (error instanceof Error) {
			if (error.name === 'AbortError') {
				// 检查是超时还是手动取消
				if (options.signal?.aborted) {
					return {
						code: -1,
						message: '请求已取消',
						request_id: ''
					};
				} else {
					return {
						code: -1,
						message: `请求超时（${timeout / 1000}秒）`,
						request_id: ''
					};
				}
			}
			
			// 其他错误
			if (dev) console.error('[API] Request error:', error);
			return {
				code: -1,
				message: error.message || '请求失败',
				request_id: ''
			};
		}

		// 网络错误或未知错误
		return {
			code: -1,
			message: '网络请求失败',
			request_id: ''
		};
	}
}

/**
 * 合并多个 AbortSignal
 */
function combineSignals(signals: AbortSignal[]): AbortSignal {
	const controller = new AbortController();

	for (const signal of signals) {
		if (signal.aborted) {
			controller.abort();
			break;
		}
		signal.addEventListener('abort', () => controller.abort(), { once: true });
	}

	return controller.signal;
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
	params?: Record<string, unknown>,
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
		this.requestId = response.request_id ?? '';
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
