/**
 * API 客户端
 *
 * 特性：
 * - 所有请求使用 POST 方法
 * - 自动注入 Authorization Token
 * - 自动注入 CSRF Token (Web 端)
 * - code=10100 时自动刷新 Token 并重试
 * - 幂等性支持 (Idempotency-Key)
 * - 业务成功判断: code === 0
 */

import { dev } from '$app/environment';
import type { ApiResponse, PaginatedData } from '$lib/types/api';
import { AUTH } from './endpoints';

/**
 * HTTP 客户端扩展的响应类型，附加 request_id 用于调试追踪。
 * 线上 JSON 仍为 spec 定义的 ApiResponse，request_id 来自响应头或客户端构造。
 */
export interface ClientApiResponse<T = void> extends ApiResponse<T> {
	request_id?: string;
}

// Re-export types for external use
export type { ApiResponse, PaginatedData };

// Token 无效或过期的错误码（spec 定义的宽泛分类码）
const TOKEN_INVALID_CODE = 10100;
const IDEMPOTENCY_CONFLICT_CODE = 22000;
const IDEMPOTENCY_CACHE_EXPIRED_CODE = 22001;
const IDEMPOTENCY_IN_PROGRESS_CODE = 22002;
const MAX_IDEMPOTENCY_RETRIES = 1;
const RATE_LIMIT_CODE = 10500;
const MAX_RATE_LIMIT_RETRIES = 1;
const DEFAULT_RETRY_AFTER_SECONDS = 5;

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
	/**
	 * Token 刷新成功回调
	 * Web 端 token 通过 Set-Cookie 更新，此回调用于通知应用更新过期时间等元数据
	 */
	onRefreshSuccess: () => void;
	/**
	 * Token 刷新失败回调
	 * 应清除认证状态并引导用户重新登录
	 */
	onRefreshFail: () => void;
	/**
	 * 账号封禁回调（code 10201）
	 * spec 要求：强制登出 + 显示封禁页
	 */
	onAccountBanned?: (message: string) => void;
}

// ============ 幂等键生成（spec: {client_id}-{timestamp_ms}-{random}，32-64 字符） ============

// 会话级客户端 ID（8 字符随机字符串）
let _clientId: string | null = null;
function getClientId(): string {
	if (!_clientId) {
		_clientId = crypto.randomUUID().replace(/-/g, '').substring(0, 8);
	}
	return _clientId;
}

/**
 * 生成 spec 要求的幂等键
 * 格式: {client_id}-{timestamp_ms}-{random}
 * 长度: 8 + 1 + 13 + 1 + 12 = ~35 字符（在 32-64 范围内）
 * 字符集: [a-zA-Z0-9-_]
 */
export function generateIdempotencyKey(): string {
	const clientId = getClientId();
	const timestamp = Date.now();
	const random = crypto.randomUUID().replace(/-/g, '').substring(0, 12);
	return `${clientId}-${timestamp}-${random}`;
}

let config: ApiClientConfig | null = null;
const MEDIA_URL_KEYS = new Set(['url', 'avatar_url', 'thumbnail_url', 'cover_url']);

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

function resolveApiOrigin(baseUrl: string): string {
	if (typeof window === 'undefined') return '';
	const trimmed = baseUrl.trim();
	if (!trimmed) return window.location.origin;
	if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
		try {
			return new URL(trimmed).origin;
		} catch {
			return window.location.origin;
		}
	}
	return window.location.origin;
}

function normalizeMediaUrl(raw: string, origin: string): string {
	const trimmed = raw.trim();
	if (!trimmed) return trimmed;
	if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) return trimmed;
	if (!trimmed.startsWith('/s3/')) return trimmed;
	if (!origin) return trimmed;
	return `${origin}${trimmed}`;
}

function patchMediaUrlsDeep(value: unknown, origin: string): void {
	if (!value || !origin) return;
	if (Array.isArray(value)) {
		for (const item of value) patchMediaUrlsDeep(item, origin);
		return;
	}
	if (typeof value !== 'object') return;
	const record = value as Record<string, unknown>;
	for (const key of Object.keys(record)) {
		const current = record[key];
		if (typeof current === 'string' && MEDIA_URL_KEYS.has(key)) {
			record[key] = normalizeMediaUrl(current, origin);
			continue;
		}
		patchMediaUrlsDeep(current, origin);
	}
}

/**
 * 检查是否为 Token 无效错误
 * spec 定义: code=10100 表示认证类错误（token 无效/过期/缺失等）
 */
function isTokenInvalidError(code: number): boolean {
	return code === TOKEN_INVALID_CODE;
}

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * 刷新 Token (Web 端)
 *
 * spec: client_type='web' 时请求体不含 refresh_token（通过 HttpOnly Cookie 传递）
 * spec: 响应体为 AuthTokenPairWeb = Record<string, never>（空对象），新 token 通过 Set-Cookie 更新
 */
async function refreshToken(): Promise<boolean> {
	const cfg = getConfig();

	try {
		const csrfToken = getCookie('hs_csrf_token');

		const headers: Record<string, string> = {
			'Content-Type': 'application/json'
		};

		if (csrfToken) {
			headers['X-CSRF-Token'] = csrfToken;
		}

		// 超时控制：防止刷新请求挂起导致所有待刷新请求无限等待
		const refreshController = new AbortController();
		const refreshTimeoutId = setTimeout(() => refreshController.abort(), 10000);

		const response = await fetch(`${cfg.baseUrl}${AUTH.REFRESH_TOKEN}`, {
			method: 'POST',
			headers,
			credentials: 'include', // 携带 HttpOnly Cookie（包含 refresh_token）
			body: JSON.stringify({
				client_type: 'web'
				// spec: Web 端不传 refresh_token（从 Cookie 读取）
			}),
			signal: refreshController.signal
		});

		clearTimeout(refreshTimeoutId);

		const data = (await response.json()) as ApiResponse<Record<string, never>>;

		if (data.code === 0) {
			// Web 端: token 已通过 Set-Cookie 更新，body 为空对象
			cfg.onRefreshSuccess();
			return true;
		}

		// 刷新失败，清除认证状态
		cfg.onRefreshFail();
		return false;
	} catch (error) {
		if (dev) console.error('[TokenRefresh] Token 刷新异常:', error);
		cfg.onRefreshFail();
		return false;
	}
}

/**
 * 确保刷新 Token（防止并发刷新）
 * 导出供 TokenRefreshManager 共享同一个刷新锁
 */
export async function ensureRefreshToken(): Promise<boolean> {
	if (isRefreshing) {
		// 等待正在进行的刷新
		try {
			return (await refreshPromise) ?? false;
		} catch {
			return false;
		}
	}

	isRefreshing = true;
	try {
		refreshPromise = refreshToken();
		return await refreshPromise;
	} catch {
		return false;
	} finally {
		isRefreshing = false;
		refreshPromise = null;
	}
}

/**
 * 核心请求方法
 */
async function request<T>(
	url: string,
	body?: unknown,
	options: RequestOptions = {},
	isRetry = false,
	idempotencyRetryCount = 0,
	rateLimitRetryCount = 0
): Promise<ClientApiResponse<T>> {
	const cfg = getConfig();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...options.headers
	};

	// spec: Web 端禁止使用 Authorization Header
	// Token 通过 HttpOnly Cookie 自动传递（credentials: 'include'）

	// 注入 CSRF Token (Web 端需要)
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
	const combined = options.signal
		? combineSignals([options.signal, controller.signal])
		: null;
	const signal = combined?.signal ?? controller.signal;

	try {
		const response = await fetch(`${cfg.baseUrl}${url}`, {
			method: 'POST',
			headers,
			credentials: 'include',
			body: JSON.stringify(body ?? {}),
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
		const apiOrigin = resolveApiOrigin(cfg.baseUrl);
		patchMediaUrlsDeep(data, apiOrigin);

		// 账号封禁拦截（spec 10201：强制登出 + 封禁页）
		if (data.code === 10201) {
			cfg.onAccountBanned?.(data.message || '账号被封禁');
			return data;
		}

		// 10500 限流：按 Retry-After 等待后重试
		if (data.code === RATE_LIMIT_CODE && rateLimitRetryCount < MAX_RATE_LIMIT_RETRIES) {
			const retryAfterStr = response.headers.get('Retry-After');
			const retryAfterSec = retryAfterStr
				? parseInt(retryAfterStr, 10) || DEFAULT_RETRY_AFTER_SECONDS
				: DEFAULT_RETRY_AFTER_SECONDS;
			await sleep(retryAfterSec * 1000);
			return request<T>(url, body, options, isRetry, idempotencyRetryCount, rateLimitRetryCount + 1);
		}

		// Token 过期处理（仅在非重试时尝试刷新）
		if (!isRetry && !options.skipAuth && isTokenInvalidError(data.code)) {
			const refreshed = await ensureRefreshToken();
			if (refreshed) {
				return request<T>(url, body, options, true, idempotencyRetryCount, rateLimitRetryCount);
			}
		}

		// 幂等错误自动重试（仅对携带 Idempotency-Key 的请求生效）
		if (options.idempotencyKey && idempotencyRetryCount < MAX_IDEMPOTENCY_RETRIES) {
			if (data.code === IDEMPOTENCY_IN_PROGRESS_CODE) {
				await sleep(1200);
				return request<T>(url, body, options, isRetry, idempotencyRetryCount + 1, rateLimitRetryCount);
			}
			if (
				data.code === IDEMPOTENCY_CONFLICT_CODE ||
				data.code === IDEMPOTENCY_CACHE_EXPIRED_CODE
			) {
				return request<T>(
					url,
					body,
					{ ...options, idempotencyKey: generateIdempotencyKey() },
					isRetry,
					idempotencyRetryCount + 1,
					rateLimitRetryCount
				);
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
	} finally {
		// 释放 combineSignals 在外部 signal 上注册的监听器，防止累积泄漏
		combined?.cleanup();
	}
}

/**
 * 合并多个 AbortSignal，返回合并后的 signal 和清理函数。
 * 请求完成后必须调用 cleanup() 释放外部 signal 上的监听器。
 */
function combineSignals(signals: AbortSignal[]): { signal: AbortSignal; cleanup: () => void } {
	const controller = new AbortController();
	const cleanups: (() => void)[] = [];

	for (const signal of signals) {
		if (signal.aborted) {
			controller.abort();
			break;
		}
		const handler = () => controller.abort();
		signal.addEventListener('abort', handler, { once: true });
		cleanups.push(() => signal.removeEventListener('abort', handler));
	}

	return {
		signal: controller.signal,
		cleanup: () => cleanups.forEach((fn) => fn())
	};
}

/**
 * POST 请求
 */
export async function post<T>(
	url: string,
	body?: unknown,
	options?: RequestOptions
): Promise<ClientApiResponse<T>> {
	return request<T>(url, body, options);
}

/**
 * 分页查询
 */
export async function paginate<T>(
	url: string,
	params?: Record<string, unknown>,
	options?: RequestOptions
): Promise<ClientApiResponse<PaginatedData<T>>> {
	return post<PaginatedData<T>>(url, params, options);
}

/**
 * 检查响应是否成功
 */
export function isSuccess<T>(response: ClientApiResponse<T>): response is ClientApiResponse<T> & { data: T } {
	return response.code === 0;
}

/**
 * 提取响应数据（如果成功）
 */
export function getData<T>(response: ClientApiResponse<T>): T | undefined {
	return isSuccess(response) ? response.data : undefined;
}

/**
 * API 错误类
 */
export class ApiError extends Error {
	code: number;
	requestId: string;

	constructor(response: ClientApiResponse<unknown>) {
		super(response.message || '操作失败');
		this.name = 'ApiError';
		this.code = response.code;
		this.requestId = response.request_id ?? '';
	}
}

/**
 * 断言响应成功，否则抛出错误
 */
export function assertSuccess<T>(response: ClientApiResponse<T>): asserts response is ClientApiResponse<T> & { data: T } {
	if (!isSuccess(response)) {
		throw new ApiError(response);
	}
}
