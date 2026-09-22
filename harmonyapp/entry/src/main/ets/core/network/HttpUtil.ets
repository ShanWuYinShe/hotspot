/**
 * HTTP 工具类
 *
 * 功能：
 * - 统一的请求/响应处理（ApiResponse 信封解析）
 * - 自动 Token 刷新（主动+被动两层机制）
 * - 幂等键自动注入
 * - 认证错误（10100）自动刷新并重试
 * - 账号封禁（10201）自动登出
 */

import http from '@ohos.net.http';
import request from '@ohos.request';
import common from '@ohos.app.ability.common';
import util from '@ohos.util';
import { API_BASE_URL, StorageKeys } from '../../constants/AppConstants';
import { ApiEndpoints } from '../config/ApiEndpoints';
import { ApiResponse, AuthData, HealthStatus, MediaCategory, UploadedMedia } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { DeviceIdUtil } from '../util/DeviceIdUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { ErrorCode } from '../constants/ErrorCodes';
import { extractApiErrorSnapshot,
  guessUserMessageFromSnapshot,
  isAuthErrorCode,
  isAccountBannedCode,
  isRateLimitCode
} from './ApiErrorUtil';
import { AppRouter, LoginRouteParam } from '../router/AppRouter';
import { MediaUrlUtil } from '../utils/MediaUrlUtil';
import { AuthAccountManager } from '../state/AuthAccountManager';
import { GlobalStateManager } from '../state/GlobalStateManager';
import { MockInterceptor } from '../mock/MockInterceptor';

/**
 * Spec MUST 幂等端点列表
 * 这些端点的每次请求必须携带 Idempotency-Key，缺少则会被后端拒绝
 * 对齐 spec/api_spec_base.md「需要实现幂等的接口」（自然去重接口无需 key，
 * 如 reports/create、groups/enter、groups/admin/invites/create、reset/change_password、
 * media complete_upload 等均不在列表内）。
 */
const MUST_IDEMPOTENT_ENDPOINTS: string[] = [
  // 认证
  ApiEndpoints.AUTH.REGISTER,
  ApiEndpoints.AUTH.GUEST,
  // 消息
  ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND,
  ApiEndpoints.CHAT.GROUP_MESSAGES.SEND,
  // 创建类
  ApiEndpoints.MOMENT.CREATE,
  ApiEndpoints.COMMENT.CREATE,
  ApiEndpoints.SHARE.CREATE,
  // 群组
  ApiEndpoints.GROUP.CREATE,
  ApiEndpoints.GROUP.ANNOUNCEMENTS.CREATE,
];

/**
 * token_optional 端点列表
 * 当携带过期/无效 token 调用这些接口时，允许降级为匿名重试一次，避免“可选认证接口被强制报错”
 */
const TOKEN_OPTIONAL_ENDPOINTS: string[] = [
  ApiEndpoints.AUTH.GET_STATUS,
  ApiEndpoints.SEARCH.SUGGEST,
  ApiEndpoints.USER.SEARCH,
  ApiEndpoints.USER.GET_DETAIL,
  ApiEndpoints.USER.LIST_FRIENDS,
  ApiEndpoints.USER.LIST_FOLLOWING,
  ApiEndpoints.USER.LIST_FOLLOWERS,
  ApiEndpoints.USER.LIST_GROUPS,
  ApiEndpoints.USER.LIST_MOMENTS,
  ApiEndpoints.USER.LIST_REACTED_MOMENTS,
  ApiEndpoints.GROUP.SEARCH,
  ApiEndpoints.GROUP.GET_DETAIL,
  ApiEndpoints.GROUP.MEMBERS.LIST,
  ApiEndpoints.GROUP.MEMBERS.SEARCH,
  ApiEndpoints.GROUP.ANNOUNCEMENTS.LIST,
  ApiEndpoints.MOMENT.SEARCH,
  ApiEndpoints.MOMENT.GET_DETAIL,
  // 双版本端点：匿名版 /public/v1/moments/fetch_nearby 为拉新路径（spec moment.md/public.md）
  ApiEndpoints.MOMENT.FETCH_NEARBY,
  ApiEndpoints.REACTIONS.LIST,
  ApiEndpoints.COMMENT.LIST,
  ApiEndpoints.COMMENT.LIST_REPLIES,
  ApiEndpoints.SHARE.GET,
  ApiEndpoints.MAP.CANVAS,
];

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'HttpUtil';
const MAX_IDEMPOTENCY_RETRY_COUNT = 1;
const MEDIA_URL_FIELDS: string[] = ['url', 'avatar_url', 'thumbnail_url', 'cover_url'];

/**
 * 幂等键的 client_id 段。spec（api_spec_base.md「幂等性」）要求
 * client_id = DeviceId；取持久化 device_id 全值，仅过滤非法字符并截断到
 * 24 位——服务端按 ^[A-Za-z0-9_-]{32,64}$ 校验整键，超长会被当作无 key
 * 静默跳过幂等（与 Web 端同策略）。device_id 尚未初始化时退化为进程内
 * 稳定值。
 */
function getIdempotencyClientId(): string {
  const raw = DeviceIdUtil.getDeviceId();
  const sanitized = raw.replace(/[^a-zA-Z0-9_-]/g, '').substring(0, 24);
  return sanitized.length > 0 ? sanitized : 'harmonyapp';
}

/**
 * 生成 spec 要求格式的幂等键：
 * {client_id}-{timestamp_ms}-{random}，长度 32-64，字符集 [a-zA-Z0-9-_]
 */
function generateIdempotencyKey(): string {
  const timestampMs = Date.now().toString();
  const randomPart = util.generateRandomUUID(false).split('-').join('').substring(0, 12);
  return `${getIdempotencyClientId()}-${timestampMs}-${randomPart}`;
}

interface DebugInfo {
  baseUrl: string;
  hasToken: boolean;
  tokenPreview?: string;
}

function isRecord(v: Object): boolean {
  return v !== null && typeof v === 'object';
}

function isApiResponse(v: Object): boolean {
  if (!isRecord(v)) return false;
  const record = v as Record<string, Object>;
  return typeof record.code === 'number';
}

function isTokenOptionalEndpoint(url: string): boolean {
  return TOKEN_OPTIONAL_ENDPOINTS.includes(url);
}

/**
 * /api/v1 → /public/v1 对偶端点映射（仅收录 spec public.md 中存在公开镜像的端点）。
 * spec api_spec_base.md 节点身份语义：/api/v1 必须携带有效 token，无一例外；
 * 匿名/未登录对公开内容的访问必须走 /public/v1。已天然是 public 路径的端点
 * （MAP.CANVAS、SHARE.GET、GROUP.ANNOUNCEMENTS.LIST）不在此表。
 */
function buildPublicAliases(): Map<string, string> {
  const m = new Map<string, string>();
  m.set(ApiEndpoints.USER.SEARCH, '/public/v1/users/search');
  m.set(ApiEndpoints.USER.GET_DETAIL, '/public/v1/users/open_the_door');
  m.set(ApiEndpoints.USER.LIST_FRIENDS, '/public/v1/users/fetch_friends');
  m.set(ApiEndpoints.USER.LIST_FOLLOWING, '/public/v1/users/fetch_following');
  m.set(ApiEndpoints.USER.LIST_FOLLOWERS, '/public/v1/users/fetch_followers');
  m.set(ApiEndpoints.USER.LIST_GROUPS, '/public/v1/users/fetch_groups');
  m.set(ApiEndpoints.USER.LIST_MOMENTS, '/public/v1/users/fetch_moments');
  m.set(ApiEndpoints.USER.LIST_REACTED_MOMENTS, '/public/v1/users/fetch_reacted_moments');
  m.set(ApiEndpoints.GROUP.SEARCH, '/public/v1/groups/search');
  m.set(ApiEndpoints.GROUP.GET_DETAIL, '/public/v1/groups/open_the_door');
  m.set(ApiEndpoints.GROUP.MEMBERS.LIST, '/public/v1/groups/members/fetch');
  m.set(ApiEndpoints.GROUP.MEMBERS.SEARCH, '/public/v1/groups/members/search');
  m.set(ApiEndpoints.MOMENT.SEARCH, '/public/v1/moments/search');
  m.set(ApiEndpoints.MOMENT.GET_DETAIL, '/public/v1/moments/open_the_door');
  m.set(ApiEndpoints.MOMENT.FETCH_NEARBY, '/public/v1/moments/fetch_nearby');
  m.set(ApiEndpoints.COMMENT.LIST, '/public/v1/comments/fetch');
  m.set(ApiEndpoints.COMMENT.LIST_REPLIES, '/public/v1/comments/fetch_replies');
  m.set(ApiEndpoints.COMMENT.GET_DETAIL, '/public/v1/comments/open_the_door');
  m.set(ApiEndpoints.SEARCH.SUGGEST, '/public/v1/search/suggest');
  return m;
}

const PUBLIC_ENDPOINT_ALIASES: Map<string, string> = buildPublicAliases();


export class HttpUtil {
  /**
   * 全局刷新锁：避免并发 refresh_token 触发"轮换冲突"
   * Spec 建议：同一时刻只允许一个刷新请求在飞
   */
  private static refreshPromise: Promise<boolean> | null = null;

  private static async getBaseUrl(): Promise<string> {
    try {
      const url = await PreferencesUtil.get<string>(StorageKeys.SERVER_BASE_URL, API_BASE_URL);
      return typeof url === 'string' && url.trim().length > 0 ? url.trim() : API_BASE_URL;
    } catch {
      return API_BASE_URL;
    }
  }

  private static isMediaUrlField(key: string): boolean {
    return MEDIA_URL_FIELDS.indexOf(key) >= 0;
  }

  private static toAbsoluteMediaUrl(raw: string, origin: string): string {
    return MediaUrlUtil.normalize(raw, origin);
  }

  private static resolveRelativeMediaUrls(value: Object, origin: string): void {
    if (!origin || !value) {
      return;
    }
    if (Array.isArray(value)) {
      const arr = value as Object[];
      for (let i = 0; i < arr.length; i++) {
        const item = arr[i];
        if (item && typeof item === 'object') {
          HttpUtil.resolveRelativeMediaUrls(item, origin);
        }
      }
      return;
    }
    if (typeof value !== 'object') {
      return;
    }

    const record = value as Record<string, string | number | boolean | Object | null | undefined>;
    const keys = Object.keys(record);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      const current = record[key];
      if (typeof current === 'string' && HttpUtil.isMediaUrlField(key)) {
        record[key] = HttpUtil.toAbsoluteMediaUrl(current, origin);
      } else if (current && typeof current === 'object') {
        HttpUtil.resolveRelativeMediaUrls(current as Object, origin);
      }
    }
  }

  /**
   * 设置访问令牌并持久化存储
   */
  static async setToken(token: string): Promise<void> {
    try {
      await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, token);
      hilog.info(DOMAIN_NUMBER, TAG, '访问令牌已保存');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存访问令牌失败: ${error}`);
    }
  }

  /**
   * 清除访问令牌
   */
  static async clearToken(): Promise<void> {
    try {
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
      hilog.info(DOMAIN_NUMBER, TAG, '访问令牌已清除');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清除访问令牌失败: ${error}`);
    }
  }

  /**
   * 获取访问令牌
   */
  static async getToken(): Promise<string> {
    try {
      return await PreferencesUtil.get<string>(StorageKeys.ACCESS_TOKEN, '');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return '';
    }
  }

  private static async getAuthData(): Promise<AuthData | null> {
    try {
      return await PreferencesUtil.getObject<AuthData>(StorageKeys.AUTH_DATA);
    } catch {
      return null;
    }
  }

  static async clearAuthData(): Promise<void> {
    try {
      await PreferencesUtil.remove(StorageKeys.ACCESS_TOKEN);
      await PreferencesUtil.remove(StorageKeys.AUTH_DATA);
      await AuthAccountManager.clearAllAccounts();
    } catch {
      // ignore
    }
    // 同步重置全局状态（UI + 全局管理器）
    GlobalStateManager.getInstance().clear();
    hilog.info(DOMAIN_NUMBER, TAG, '认证数据已清除，UI 状态已重置为未登录');
  }

  /**
   * 刷新 token（App 场景：refresh_token 放在 body）
   *
   * Spec 规则：
   * - refresh_token 端点是 token 可选的，即使 access_token 已过期也可调用
   * - App 端必须在 body 中传 refresh_token，不传 access_token
   * - 并发刷新通过 refreshPromise 锁避免竞态
   */
  private static async ensureRefreshToken(): Promise<boolean> {
    if (HttpUtil.refreshPromise) return HttpUtil.refreshPromise;

    HttpUtil.refreshPromise = (async () => {
      try {
        const authData = await HttpUtil.getAuthData();
        const refreshToken = authData?.refresh_token;
        if (!refreshToken) {
          hilog.warn(DOMAIN_NUMBER, TAG, '无 refresh_token，无法刷新');
          return false;
        }

        const baseUrl = await HttpUtil.getBaseUrl();
        const fullUrl = `${baseUrl}${ApiEndpoints.AUTH.REFRESH_TOKEN}`;
        const httpRequest = http.createHttp();

        // 刷新请求不携带 Authorization header（access_token 可能已过期）
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        const requestOptions: http.HttpRequestOptions = {
          method: http.RequestMethod.POST,
          header: headers,
          extraData: JSON.stringify({ refresh_token: refreshToken, client_type: 'app', device_id: DeviceIdUtil.getDeviceId() }),
          connectTimeout: 15000,
          readTimeout: 15000
        };

        const response = await httpRequest.request(fullUrl, requestOptions);
        let payload: Object;
        try {
          payload = JSON.parse(response.result as string) as Object;
        } catch {
          return false;
        } finally {
          httpRequest.destroy();
        }

        const payloadRecord = payload as Record<string, Object>;
        if (!payloadRecord || typeof payloadRecord.code !== 'number') return false;

        if (payloadRecord.code === 0 && payloadRecord.data) {
          const newAuth = payloadRecord.data as AuthData;
          await PreferencesUtil.set(StorageKeys.ACCESS_TOKEN, newAuth.access_token);
          await PreferencesUtil.setObject(StorageKeys.AUTH_DATA, newAuth);
          await AuthAccountManager.upsertAndActivate(newAuth);
          hilog.info(DOMAIN_NUMBER, TAG, 'Token 刷新成功');
          return true;
        }

        // 刷新失败（10100=refresh_token 无效、10201=账号被封禁等），清除认证状态
        hilog.warn(DOMAIN_NUMBER, TAG, `Token 刷新失败，code=${payloadRecord.code}`);
        await HttpUtil.clearAuthData();
        return false;
      } catch (error) {
        // 网络层异常（超时/断网/连接失败）≠ 凭证失效：spec 只要求 refresh_token
        // 无效（10100）时清除会话。弱网下走到这里的刷新请求会抛异常，若此时
        // clearAuthData 会把用户误踢回登录页——保留本地会话，仅返回刷新失败，
        // 后续网络恢复后可重试刷新。
        hilog.warn(DOMAIN_NUMBER, TAG, `refresh_token failed (network): ${JSON.stringify(error)}`);
        return false;
      } finally {
        HttpUtil.refreshPromise = null;
      }
    })();

    return HttpUtil.refreshPromise;
  }

  /**
   * 主动刷新：检查 access_token 是否即将过期，提前刷新
   * 在发起请求前调用，避免请求失败后再刷新导致额外延迟
   */
  private static async proactiveRefreshIfNeeded(): Promise<void> {
    try {
      const authData = await HttpUtil.getAuthData();
      if (!authData?.access_token_expires_at || !authData?.refresh_token) return;

      const expiresAt = new Date(authData.access_token_expires_at).getTime();
      const now = Date.now();
      const thresholdMs = 3 * 60 * 1000; // 提前 3 分钟刷新

      if (expiresAt - now < thresholdMs && expiresAt - now > 0) {
        hilog.info(DOMAIN_NUMBER, TAG, `Access token 将在 ${Math.round((expiresAt - now) / 1000)}s 后过期，主动刷新`);
        await HttpUtil.ensureRefreshToken();
      }
    } catch {
      // 主动刷新失败不影响正常请求，被动刷新会兜底
    }
  }

  private static async buildHeaders(
    url?: string,
    forcedIdempotencyKey: string = '',
    omitAuthHeader: boolean = false
  ): Promise<Record<string, string>> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json'
    };

    if (!omitAuthHeader) {
      try {
        const token = await HttpUtil.getToken();
        if (token) {
          headers['Authorization'] = `Bearer ${token}`;
        }
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `获取令牌失败，使用无认证请求: ${error}`);
      }
    }

    // 自动为 MUST 幂等端点注入 Idempotency-Key
    if (url && MUST_IDEMPOTENT_ENDPOINTS.includes(url)) {
      headers['Idempotency-Key'] = forcedIdempotencyKey || generateIdempotencyKey();
    }

    return headers;
  }

  /**
   * 通用请求方法 - 支持强类型请求体
   *
   * Token 刷新流程（双层机制）：
   * 1. 主动刷新：请求发出前检查 access_token_expires_at，即将过期则提前刷新
   * 2. 被动刷新：收到 10100 时自动刷新 token 并重试一次
   */
  static async request<TResponse, TRequest>(
    url: string,
    method: http.RequestMethod = http.RequestMethod.POST,
    data?: TRequest,
    retried: boolean = false,
    forcedIdempotencyKey: string = '',
    idempotencyRetryCount: number = 0,
    omitAuthHeader: boolean = false,
    rateLimitRetryCount: number = 0
  ): Promise<TResponse> {
    // DEV mock interceptor — returns canned ApiResponse when enabled.
    if (MockInterceptor.isEnabled()) {
      const canned = MockInterceptor.match(url);
      if (canned) {
        hilog.debug(DOMAIN_NUMBER, TAG, `Mock response for ${url}`);
        return canned as TResponse;
      }
    }

    // 匿名请求路由（spec api_spec_base.md 节点身份语义）：/api/v1 必须携带有效
    // token，未登录时对「公开可浏览」端点直接改走 /public/v1 对偶端点，而不是
    // 打 /api/v1 稳定吃 10100。此前未登录状态下首页/搜索/附近的请求全部失败。
    let effectiveUrl = url;
    if (!omitAuthHeader && !(await HttpUtil.getToken())) {
      const alias = PUBLIC_ENDPOINT_ALIASES.get(url);
      if (alias) {
        effectiveUrl = alias;
        hilog.info(DOMAIN_NUMBER, TAG, `未登录，匿名端点改走 public 节点: ${url} -> ${alias}`);
      }
    }

    const baseUrl = await HttpUtil.getBaseUrl();
    const fullUrl = `${baseUrl}${effectiveUrl}`;
    const httpRequest = http.createHttp();
    const startTime = Date.now();

    hilog.info(DOMAIN_NUMBER, TAG, `HTTP请求: ${method} ${fullUrl}`);

    try {
      const tokenForRetryCheck = omitAuthHeader ? '' : await HttpUtil.getToken();
      const hasAuthHeader = !omitAuthHeader && !!tokenForRetryCheck;

      // 主动刷新：如果 token 即将过期，先刷新再发请求
      if (hasAuthHeader && !retried) {
        await HttpUtil.proactiveRefreshIfNeeded();
      }

      const isMustIdempotentEndpoint = MUST_IDEMPOTENT_ENDPOINTS.includes(url);
      const currentIdempotencyKey = isMustIdempotentEndpoint
        ? (forcedIdempotencyKey || generateIdempotencyKey())
        : '';
      const headers = await HttpUtil.buildHeaders(url, currentIdempotencyKey, omitAuthHeader);
      const requestOptions: http.HttpRequestOptions = {
        method,
        header: headers,
        extraData: data ? JSON.stringify(data) : undefined,
        connectTimeout: 30000,
        readTimeout: 30000
      };

      const response = await httpRequest.request(fullUrl, requestOptions);
      const responseTime = Date.now() - startTime;
      hilog.info(DOMAIN_NUMBER, TAG, `HTTP响应: ${response.responseCode} (${responseTime}ms)`);

      if (response.responseCode < 200 || response.responseCode >= 300) {
        throw new Error(`HTTP ${response.responseCode}`);
      }

      let result: Object;
      try {
        result = JSON.parse(response.result as string);
        const resultCode = (result as Record<string, Object>)?.code;
        hilog.debug(DOMAIN_NUMBER, TAG, `响应 code=${resultCode ?? 'unknown'}`);
      } catch (parseError) {
        throw new Error(`JSON解析失败: ${parseError}`);
      }

      // 业务成功与否仅以 ApiResponse.code 判定（后端统一返回 HTTP 200）
      if (!isApiResponse(result)) {
        throw new Error('响应格式不符合 ApiResponse 约定');
      }

      const origin = HttpUtil.extractOrigin(baseUrl);
      HttpUtil.resolveRelativeMediaUrls(result, origin);

      const resultRecord = result as Record<string, Object>;
      const resultCode = resultRecord.code as number;

      // 10201：账号被封禁 -> 强制登出 + 封禁页
      if (isAccountBannedCode(resultCode)) {
        await HttpUtil.clearAuthData();
        const param = new LoginRouteParam();
        param.reason = 'banned';
        param.action = (resultRecord.message as string) || '账号已被封禁';
        AppRouter.clear();
        AppRouter.push('LoginPage', param);
        const err = new Error('账号已被封禁');
        (err as Object as Record<string, Object>).code = resultCode;
        throw err;
      }

      // 10500：限流 -> 按 Retry-After 等待后重试
      if (isRateLimitCode(resultCode) && rateLimitRetryCount < 1) {
        const retryAfterHeader = (response.header as Record<string, string>)?.['Retry-After'] ?? '5';
        const retryAfterSec = parseInt(retryAfterHeader) || 5;
        hilog.warn(DOMAIN_NUMBER, TAG, `收到 10500 限流，${retryAfterSec}s 后重试`);
        await new Promise<void>((resolve) => setTimeout(resolve, retryAfterSec * 1000));
        return await HttpUtil.request<TResponse, TRequest>(
          url, method, data, retried, forcedIdempotencyKey,
          idempotencyRetryCount, omitAuthHeader, rateLimitRetryCount + 1
        );
      }

      // 10100：认证失败 -> 首次尝试时自动刷新 token 并重试
      if (!retried && hasAuthHeader && isAuthErrorCode(resultCode)) {
        hilog.info(DOMAIN_NUMBER, TAG, '收到 10100 认证失败，尝试刷新 token');
        const refreshed = await HttpUtil.ensureRefreshToken();
        if (refreshed) {
          return await HttpUtil.request<TResponse, TRequest>(
            url, method, data, true, currentIdempotencyKey,
            idempotencyRetryCount, omitAuthHeader, rateLimitRetryCount
          );
        }
        hilog.warn(DOMAIN_NUMBER, TAG, '令牌刷新失败，已强制登出');
        await HttpUtil.clearAuthData();
        const param = new LoginRouteParam();
        param.reason = 'expired';
        AppRouter.clear();
        AppRouter.push('LoginPage', param);
      }

      // 认证失败且本次携带了 Authorization 的公开端点：匿名降级重试必须换到
      // /public/v1 对偶节点——重打 /api/v1（仅去掉 Authorization）仍会 10100，
      // 属于无效重试。无对偶的 token_optional 端点保留原同节点匿名重试。
      if (isAuthErrorCode(resultCode) && hasAuthHeader && !omitAuthHeader) {
        const alias = PUBLIC_ENDPOINT_ALIASES.get(url);
        if (alias) {
          hilog.info(DOMAIN_NUMBER, TAG, `认证失败，公开端点降级为匿名 public 节点重试: ${url} -> ${alias}`);
          return await HttpUtil.request<TResponse, TRequest>(
            alias, method, data, true, currentIdempotencyKey,
            idempotencyRetryCount, true, rateLimitRetryCount
          );
        }
      }
      if (isAuthErrorCode(resultCode) && hasAuthHeader && isTokenOptionalEndpoint(url) && !omitAuthHeader) {
        hilog.info(DOMAIN_NUMBER, TAG, `token_optional 端点认证失败，匿名降级重试: ${url}`);
        return await HttpUtil.request<TResponse, TRequest>(
          url, method, data, true, currentIdempotencyKey,
          idempotencyRetryCount, true, rateLimitRetryCount
        );
      }

      // 幂等错误自动重试（仅 MUST 幂等端点）
      if (isMustIdempotentEndpoint && idempotencyRetryCount < MAX_IDEMPOTENCY_RETRY_COUNT) {
        if (resultCode === ErrorCode.IDEMPOTENCY_IN_FLIGHT) {
          await new Promise<void>((resolve) => setTimeout(resolve, 1200));
          return await HttpUtil.request<TResponse, TRequest>(
            url, method, data, retried, currentIdempotencyKey,
            idempotencyRetryCount + 1, omitAuthHeader, rateLimitRetryCount
          );
        }
        // 22000 CONFLICT：同一个幂等键被复用于不同请求体 → 必须换新键重试
        if (resultCode === ErrorCode.IDEMPOTENCY_KEY_CONFLICT) {
          hilog.warn(DOMAIN_NUMBER, TAG, `收到 ${ErrorCode.IDEMPOTENCY_KEY_CONFLICT} 幂等键冲突，换新键重试`);
          return await HttpUtil.request<TResponse, TRequest>(
            url, method, data, retried, generateIdempotencyKey(),
            idempotencyRetryCount + 1, omitAuthHeader, rateLimitRetryCount
          );
        }
        // 22001 CACHE_EXPIRED：缓存响应过期且无法恢复 → 生成新键重试（spec: api_spec_base.md「幂等性」）
        if (resultCode === ErrorCode.IDEMPOTENCY_CACHE_EXPIRED) {
          hilog.warn(DOMAIN_NUMBER, TAG, `收到 ${ErrorCode.IDEMPOTENCY_CACHE_EXPIRED} 幂等缓存过期，换新键重试`);
          return await HttpUtil.request<TResponse, TRequest>(
            url, method, data, retried, generateIdempotencyKey(),
            idempotencyRetryCount + 1, omitAuthHeader, rateLimitRetryCount
          );
        }
      }

      // 其他业务错误
      if (resultCode !== 0) {
        const snapshot = extractApiErrorSnapshot(result, response.responseCode);
        const userMessage = guessUserMessageFromSnapshot(snapshot, '操作失败');
        hilog.error(DOMAIN_NUMBER, TAG, `业务错误 [${snapshot.code}]: ${userMessage}`);

        const err = new Error(userMessage);
        const errRecord = err as Object as Record<string, Object>;
        if (snapshot.code !== undefined) {
          errRecord.code = snapshot.code;
        }
        if (snapshot.status !== undefined) {
          errRecord.status = snapshot.status;
        }
        // 附带响应 data（10200 权限失败时 data = { forbidden_reason: ForbiddenReason }，见 spec api_spec_errors.md），供调用方读取
        const responseData = resultRecord.data;
        if (responseData !== undefined && responseData !== null) {
          errRecord.data = responseData;
          const dataRecord = responseData as Record<string, Object>;
          const dataReason = dataRecord.forbidden_reason;
          if (typeof dataReason === 'string' && dataReason.length > 0) {
            errRecord.forbidden_reason = dataReason;
          }
        }
        throw err;
      }

      hilog.info(DOMAIN_NUMBER, TAG, `请求成功: ${method} ${url} (${responseTime}ms)`);
      return result as TResponse;

    } catch (error) {
      const responseTime = Date.now() - startTime;
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      hilog.error(DOMAIN_NUMBER, TAG, `请求失败: ${method} ${url} (${responseTime}ms) ${errorMsg}`);

      if (errorMsg.includes('Network') || errorMsg.includes('timeout') || errorMsg.includes('connect')) {
        const baseUrl2 = await HttpUtil.getBaseUrl();
        hilog.error(DOMAIN_NUMBER, TAG, `网络连接问题，请检查服务器地址: ${baseUrl2}`);
      }

      if (error instanceof Error) {
        throw error;
      }
      throw new Error(errorMsg);
    } finally {
      httpRequest.destroy();
    }
  }

  // POST 方法 - 支持强类型请求体
  static post<TResponse, TRequest>(url: string, data: TRequest): Promise<TResponse> {
    return HttpUtil.request<TResponse, TRequest>(url, http.RequestMethod.POST, data);
  }

  /**
   * 上传二进制数据到指定 URL (PUT)
   * 用于 S3 Presigned URL 上传
   */
  static async uploadToUrl(url: string, data: ArrayBuffer, headers: Record<string, string>): Promise<void> {
    const httpRequest = http.createHttp();
    try {
      const resolvedUploadUrl = await HttpUtil.resolveUploadUrl(url);
      hilog.info(DOMAIN_NUMBER, TAG, `开始上传文件到: ${resolvedUploadUrl}`);
      const response = await httpRequest.request(resolvedUploadUrl, {
        method: http.RequestMethod.PUT,
        header: headers,
        extraData: data,
        connectTimeout: 60000,
        readTimeout: 60000
      });

      hilog.info(DOMAIN_NUMBER, TAG, `上传响应: ${response.responseCode}`);

      if (response.responseCode < 200 || response.responseCode >= 300) {
        throw new Error(`Upload failed with status ${response.responseCode}`);
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      hilog.error(DOMAIN_NUMBER, TAG, `上传失败: ${errorMsg}`);
      throw new Error(errorMsg);
    } finally {
      httpRequest.destroy();
    }
  }

  /**
   * 预签名 URL 兼容处理：
   * - 绝对 URL：直接使用
   * - 相对 URL（如 /s3/...）：补全为当前服务 origin，避免 ArkTS 网络层拒绝
   */
  private static async resolveUploadUrl(url: string): Promise<string> {
    const trimmed = url.trim();
    const baseUrl = await HttpUtil.getBaseUrl();
    const origin = HttpUtil.extractOrigin(baseUrl);
    const normalized = MediaUrlUtil.normalize(trimmed, origin);
    if (normalized !== trimmed) {
      return normalized;
    }
    if (!origin) {
      return trimmed;
    }
    if (trimmed.startsWith('/')) {
      return `${origin}${trimmed}`;
    }
    return `${origin}/${trimmed}`;
  }

  private static extractOrigin(fullUrl: string): string {
    return MediaUrlUtil.extractOrigin(fullUrl);
  }

  /**
   * 获取当前配置信息（用于调试）
   */
  static async getDebugInfo(): Promise<DebugInfo> {
    const token = await HttpUtil.getToken();
    const baseUrl = await HttpUtil.getBaseUrl();
    return {
      baseUrl,
      hasToken: !!token,
      tokenPreview: token ? `${token.substring(0, 10)}...` : undefined
    };
  }

  /**
   * 测试服务器连接 - 使用系统健康检查API
   */
  static async testConnection(): Promise<ApiResponse<HealthStatus>> {
    const emptyRequest: Record<string, Object> = {};
    return HttpUtil.request<ApiResponse<HealthStatus>, Record<string, Object>>(
      ApiEndpoints.SYSTEM.HEALTH,
      http.RequestMethod.POST,
      emptyRequest
    );
  }
}
