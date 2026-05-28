/**
 * API 客户端
 *
 * 特性：
 * - 所有请求使用 POST 方法
 * - 自动注入 CSRF Token (Web 端)
 * - code=10100 时自动刷新 Token 并重试
 * - 幂等性支持 (Idempotency-Key)
 * - 业务成功判断: code === 0
 */

import type { ApiResponse, PaginatedData } from "@/lib/types/api";
import { AUTH } from "./endpoints";

export interface ClientApiResponse<T = void> extends ApiResponse<T> {
  request_id?: string;
}

export type { ApiResponse, PaginatedData };

export const TOKEN_INVALID_CODE = 10100;
const IDEMPOTENCY_CONFLICT_CODE = 22000;
const IDEMPOTENCY_CACHE_EXPIRED_CODE = 22001;
const IDEMPOTENCY_IN_PROGRESS_CODE = 22002;
const MAX_IDEMPOTENCY_RETRIES = 1;
const RATE_LIMIT_CODE = 10500;
const MAX_RATE_LIMIT_RETRIES = 1;
const DEFAULT_RETRY_AFTER_SECONDS = 5;

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const cookies = document.cookie.split(";");
  for (const cookie of cookies) {
    const trimmed = cookie.trim();
    const separatorIndex = trimmed.indexOf("=");
    if (separatorIndex < 0) continue;
    const key = trimmed.slice(0, separatorIndex);
    if (key !== name) continue;
    return trimmed.slice(separatorIndex + 1);
  }
  return null;
}

export interface RequestOptions {
  headers?: Record<string, string>;
  skipAuth?: boolean;
  idempotencyKey?: string;
  timeout?: number;
  signal?: AbortSignal;
}

export interface ApiClientConfig {
  baseUrl: string;
  onRefreshSuccess: () => void;
  onRefreshFail: () => void;
  onAccountBanned?: (message: string) => void;
  canRefresh?: () => boolean;
}

let _clientId: string | null = null;
function getClientId(): string {
  if (!_clientId) {
    _clientId = crypto.randomUUID().replace(/-/g, "").substring(0, 8);
  }
  return _clientId;
}

export function generateIdempotencyKey(): string {
  const clientId = getClientId();
  const timestamp = Date.now();
  const random = crypto.randomUUID().replace(/-/g, "").substring(0, 12);
  return `${clientId}-${timestamp}-${random}`;
}

let config: ApiClientConfig | null = null;
const MEDIA_URL_KEYS = new Set([
  "url",
  "avatar_url",
  "thumbnail_url",
  "cover_url",
]);

export function initApiClient(clientConfig: ApiClientConfig) {
  config = clientConfig;
}

function getConfig(): ApiClientConfig {
  if (!config) {
    throw new Error("API client not initialized. Call initApiClient first.");
  }
  return config;
}

function resolveApiOrigin(baseUrl: string): string {
  if (typeof window === "undefined") return "";
  const trimmed = baseUrl.trim();
  if (!trimmed) return window.location.origin;
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
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
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://"))
    return trimmed;
  if (!trimmed.startsWith("/s3/")) return trimmed;
  if (!origin) return trimmed;
  return `${origin}${trimmed}`;
}

function patchMediaUrlsDeep(value: unknown, origin: string): void {
  if (!value || !origin) return;
  if (Array.isArray(value)) {
    for (const item of value) patchMediaUrlsDeep(item, origin);
    return;
  }
  if (typeof value !== "object") return;
  const record = value as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    const current = record[key];
    if (typeof current === "string" && MEDIA_URL_KEYS.has(key)) {
      record[key] = normalizeMediaUrl(current, origin);
      continue;
    }
    patchMediaUrlsDeep(current, origin);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function refreshToken(): Promise<boolean> {
  const cfg = getConfig();
  if (cfg.canRefresh && !cfg.canRefresh()) {
    return false;
  }
  const refreshController = new AbortController();
  const refreshTimeoutId = setTimeout(() => refreshController.abort(), 10000);

  try {
    const csrfToken = getCookie("hs_csrf_token");
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken;
    }

    const response = await fetch(`${cfg.baseUrl}${AUTH.REFRESH_TOKEN}`, {
      method: "POST",
      headers,
      credentials: "include",
      body: JSON.stringify({ client_type: "web" }),
      signal: refreshController.signal,
    });

    const data = (await response.json()) as ApiResponse<Record<string, never>>;
    if (data.code === 0) {
      cfg.onRefreshSuccess();
      return true;
    }
    if (data.code === 10201) {
      cfg.onAccountBanned?.(data.message || "账号被封禁");
      return false;
    }
    cfg.onRefreshFail();
    return false;
  } catch {
    cfg.onRefreshFail();
    return false;
  } finally {
    clearTimeout(refreshTimeoutId);
  }
}

export async function ensureRefreshToken(): Promise<boolean> {
  if (isRefreshing) {
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

function combineSignals(signals: AbortSignal[]): {
  signal: AbortSignal;
  cleanup: () => void;
} {
  const controller = new AbortController();
  const cleanups: (() => void)[] = [];
  for (const signal of signals) {
    if (signal.aborted) {
      controller.abort();
      break;
    }
    const handler = () => controller.abort();
    signal.addEventListener("abort", handler, { once: true });
    cleanups.push(() => signal.removeEventListener("abort", handler));
  }
  return {
    signal: controller.signal,
    cleanup: () => cleanups.forEach((fn) => fn()),
  };
}

async function request<T>(
  url: string,
  body?: unknown,
  options: RequestOptions = {},
  isRetry = false,
  idempotencyRetryCount = 0,
  rateLimitRetryCount = 0,
): Promise<ClientApiResponse<T>> {
  const cfg = getConfig();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  const csrfToken = getCookie("hs_csrf_token");
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  if (options.idempotencyKey) {
    headers["Idempotency-Key"] = options.idempotencyKey;
  }

  const timeout = options.timeout ?? 30000;
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);
  const combined = options.signal
    ? combineSignals([options.signal, controller.signal])
    : null;
  const signal = combined?.signal ?? controller.signal;

  try {
    const response = await fetch(`${cfg.baseUrl}${url}`, {
      method: "POST",
      headers,
      credentials: "include",
      body: JSON.stringify(body ?? {}),
      signal,
    });

    if (!response.ok) {
      let errorMessage = `HTTP Error: ${response.status}`;
      let requestId = response.headers.get("X-Request-ID") || "";
      try {
        const errorBody = await response.json();
        if (errorBody.message) errorMessage = errorBody.message;
        if (errorBody.request_id) requestId = errorBody.request_id;
      } catch {
        // ignore
      }
      return { code: -1, message: errorMessage, request_id: requestId };
    }

    let data: ApiResponse<T>;
    try {
      data = (await response.json()) as ApiResponse<T>;
    } catch {
      return {
        code: -1,
        message: `响应解析失败: ${response.statusText}`,
        request_id: response.headers.get("X-Request-ID") || "",
      };
    }
    const apiOrigin = resolveApiOrigin(cfg.baseUrl);
    patchMediaUrlsDeep(data, apiOrigin);

    if (data.code === 10201) {
      cfg.onAccountBanned?.(data.message || "账号被封禁");
      return data;
    }

    if (
      data.code === RATE_LIMIT_CODE &&
      rateLimitRetryCount < MAX_RATE_LIMIT_RETRIES
    ) {
      const retryAfterStr = response.headers.get("Retry-After");
      const retryAfterSec = retryAfterStr
        ? parseInt(retryAfterStr, 10) || DEFAULT_RETRY_AFTER_SECONDS
        : DEFAULT_RETRY_AFTER_SECONDS;
      await sleep(retryAfterSec * 1000);
      return request<T>(
        url,
        body,
        options,
        isRetry,
        idempotencyRetryCount,
        rateLimitRetryCount + 1,
      );
    }

    if (!isRetry && !options.skipAuth && data.code === TOKEN_INVALID_CODE) {
      const refreshed = await ensureRefreshToken();
      if (refreshed) {
        return request<T>(
          url,
          body,
          options,
          true,
          idempotencyRetryCount,
          rateLimitRetryCount,
        );
      }
    }

    if (
      options.idempotencyKey &&
      idempotencyRetryCount < MAX_IDEMPOTENCY_RETRIES
    ) {
      if (data.code === IDEMPOTENCY_IN_PROGRESS_CODE) {
        await sleep(1200);
        return request<T>(
          url,
          body,
          options,
          isRetry,
          idempotencyRetryCount + 1,
          rateLimitRetryCount,
        );
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
          rateLimitRetryCount,
        );
      }
    }

    return data;
  } catch (error) {
    if (error instanceof Error) {
      if (error.name === "AbortError") {
        if (options.signal?.aborted) {
          return { code: -1, message: "请求已取消", request_id: "" };
        } else {
          return {
            code: -1,
            message: `请求超时（${timeout / 1000}秒）`,
            request_id: "",
          };
        }
      }
      return { code: -1, message: error.message || "请求失败", request_id: "" };
    }
    return { code: -1, message: "网络请求失败", request_id: "" };
  } finally {
    clearTimeout(timeoutId);
    combined?.cleanup();
  }
}

export async function post<T>(
  url: string,
  body?: unknown,
  options?: RequestOptions,
): Promise<ClientApiResponse<T>> {
  return request<T>(url, body, options);
}

export async function paginate<T>(
  url: string,
  params?: Record<string, unknown>,
  options?: RequestOptions,
): Promise<ClientApiResponse<PaginatedData<T>>> {
  return post<PaginatedData<T>>(url, params, options);
}

export function isSuccess<T>(
  response: ClientApiResponse<T>,
): response is ClientApiResponse<T> & { data: T } {
  return response.code === 0;
}

export function getData<T>(response: ClientApiResponse<T>): T | undefined {
  return isSuccess(response) ? response.data : undefined;
}

export class ApiError extends Error {
  code: number;
  requestId: string;
  original: ClientApiResponse<unknown>;

  constructor(response: ClientApiResponse<unknown>) {
    super(response.message || "操作失败");
    this.name = "ApiError";
    this.code = response.code;
    this.requestId = response.request_id ?? "";
    this.original = response;
  }
}

export function assertSuccess<T>(
  response: ClientApiResponse<T>,
): asserts response is ClientApiResponse<T> & { data: T } {
  if (!isSuccess(response)) {
    throw new ApiError(response);
  }
}
