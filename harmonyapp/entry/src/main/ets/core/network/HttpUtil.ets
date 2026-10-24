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
import { API_BASE_URL, PageNames, StorageKeys } from '../../constants/AppConstants';
import { ApiEndpoints } from '../config/ApiEndpoints';
import { ApiResponse, AuthData, HealthStatus, MediaCategory, UploadedMedia } from '../types/api';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { TokenAssetStore } from '../storage/TokenAssetStore';
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
import { ApiError } from './ApiError';

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

// ── 请求管线基础设施 ────────────────────────────────────────────────

/** 单次尝试的三种结局：成功（result 可用）/ 终态失败（failure 待抛）/ 重试。 */
enum AttemptDirective {
  DONE,
  FAIL,
  RETRY
}

/**
 * 一次请求的可变上下文：跨重试循环各阶段共享。原实现把这些状态用
 * 8 个方法参数 + 3 个递归计数器传递，分支一多便不可读；集中后每段
 * 处理器只改写自己负责的字段。
 */
class RequestContext {
  url: string = '';
  method: http.RequestMethod = http.RequestMethod.POST;
  data: Object | undefined = undefined;
  retried: boolean = false;
  forcedIdempotencyKey: string = '';
  idempotencyRetryCount: number = 0;
  omitAuthHeader: boolean = false;
  rateLimitRetryCount: number = 0;

  // 单次尝试内的派生状态（performAttempt 写入，重试判定读取）
  hasAuthHeader: boolean = false;
  sentToken: string = '';
  isMustIdempotent: boolean = false;
  idempotencyKey: string = '';
  /** 登录/刷新端点标记（performAttempt 写入；重试判定据此跳过 10100 刷新重试） */
  isAuthEndpoint: boolean = false;

  // 终态载荷：DONE 时为成功响应，FAIL 时为待抛 ApiError
  result?: Object;
  failure?: ApiError;
}
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
  // 路径字面量全部收敛到 ApiEndpoints（*_PUBLIC 后缀常量），此处只做「登录版 → 公开版」映射。
  const m = new Map<string, string>();
  m.set(ApiEndpoints.USER.SEARCH, ApiEndpoints.USER.SEARCH_PUBLIC);
  m.set(ApiEndpoints.USER.GET_DETAIL, ApiEndpoints.USER.GET_DETAIL_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_FRIENDS, ApiEndpoints.USER.LIST_FRIENDS_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_FOLLOWING, ApiEndpoints.USER.LIST_FOLLOWING_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_FOLLOWERS, ApiEndpoints.USER.LIST_FOLLOWERS_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_GROUPS, ApiEndpoints.USER.LIST_GROUPS_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_MOMENTS, ApiEndpoints.USER.LIST_MOMENTS_PUBLIC);
  m.set(ApiEndpoints.USER.LIST_REACTED_MOMENTS, ApiEndpoints.USER.LIST_REACTED_MOMENTS_PUBLIC);
  m.set(ApiEndpoints.GROUP.SEARCH, ApiEndpoints.GROUP.SEARCH_PUBLIC);
  m.set(ApiEndpoints.GROUP.GET_DETAIL, ApiEndpoints.GROUP.GET_DETAIL_PUBLIC);
  m.set(ApiEndpoints.GROUP.MEMBERS.LIST, ApiEndpoints.GROUP.MEMBERS.LIST_PUBLIC);
  m.set(ApiEndpoints.GROUP.MEMBERS.SEARCH, ApiEndpoints.GROUP.MEMBERS.SEARCH_PUBLIC);
  m.set(ApiEndpoints.MOMENT.SEARCH, ApiEndpoints.MOMENT.SEARCH_PUBLIC);
  m.set(ApiEndpoints.MOMENT.GET_DETAIL, ApiEndpoints.MOMENT.GET_DETAIL_PUBLIC);
  m.set(ApiEndpoints.MOMENT.FETCH_NEARBY, ApiEndpoints.MOMENT.FETCH_NEARBY_PUBLIC);
  m.set(ApiEndpoints.COMMENT.LIST, ApiEndpoints.COMMENT.LIST_PUBLIC);
  m.set(ApiEndpoints.COMMENT.LIST_REPLIES, ApiEndpoints.COMMENT.LIST_REPLIES_PUBLIC);
  m.set(ApiEndpoints.COMMENT.GET_DETAIL, ApiEndpoints.COMMENT.GET_DETAIL_PUBLIC);
  m.set(ApiEndpoints.SEARCH.SUGGEST, ApiEndpoints.SEARCH.SUGGEST_PUBLIC);
  return m;
}

const PUBLIC_ENDPOINT_ALIASES: Map<string, string> = buildPublicAliases();


/**
 * 刷新结果三态：ok（可重试原请求）/ invalid（会话确已失效，调用方清除并登出）/
 * transport-error（服务端不可达等传输失败，调用方不得清除会话、按可重试处理）。
 * 背景：部署滚动窗口内续期撞上后端不可用时，以往不分青红皂白统一清除会话，
 * 导致“每次发版 App 都掉登录”（与 Web 端同类缺陷，对齐其三态方案）。
 */
export type RefreshOutcome = 'ok' | 'invalid' | 'transport-error';

export class HttpUtil {
  /**
   * 全局刷新锁：避免并发 refresh_token 触发"轮换冲突"
   * Spec 建议：同一时刻只允许一个刷新请求在飞
   */
  private static refreshPromise: Promise<RefreshOutcome> | null = null;

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
      await TokenAssetStore.set(TokenAssetStore.ALIAS_CURRENT_ACCESS, token);
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
      await TokenAssetStore.remove(TokenAssetStore.ALIAS_CURRENT_ACCESS);
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
      const token = await TokenAssetStore.get(TokenAssetStore.ALIAS_CURRENT_ACCESS);
      return token ?? '';
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取访问令牌失败: ${error}`);
      return '';
    }
  }

  private static async getAuthData(): Promise<AuthData | null> {
    try {
      return await TokenAssetStore.getObject<AuthData>(TokenAssetStore.ALIAS_CURRENT_AUTH);
    } catch {
      return null;
    }
  }

  static async clearAuthData(): Promise<void> {
    try {
      // 只锁当前账号会话（lockCurrent 内部清 token/auth_data 两个键），保留
      // accounts 列表里的其它已保存账号——封禁/刷新失效等终态可能是误判（如
      // 部署滚动窗口），不应一次性抹掉全部多账户；删除单个账号走
      // AuthAccountManager.removeAccount，「清空所有账号」仅 clearAllAccounts。
      await AuthAccountManager.lockCurrent();
    } catch {
      // ignore
    }
    // 同步重置全局状态（UI + 全局管理器）
    GlobalStateManager.getInstance().clear();
    hilog.info(DOMAIN_NUMBER, TAG, '当前账号会话已锁定，UI 状态已重置为未登录');
  }

  /**
   * 刷新 token（App 场景：refresh_token 放在 body）
   *
   * Spec 规则：
   * - refresh_token 端点是 token 可选的，即使 access_token 已过期也可调用
   * - App 端必须在 body 中传 refresh_token，不传 access_token
   * - 并发刷新通过 refreshPromise 锁避免竞态
   */
  private static async ensureRefreshToken(): Promise<RefreshOutcome> {
    if (HttpUtil.refreshPromise) return HttpUtil.refreshPromise;

    HttpUtil.refreshPromise = (async (): Promise<RefreshOutcome> => {
      try {
        const authData = await HttpUtil.getAuthData();
        const refreshToken = authData?.refresh_token;
        if (!refreshToken) {
          hilog.warn(DOMAIN_NUMBER, TAG, '无 refresh_token，无法刷新');
          return 'invalid';
        }
        // 竞态防护：刷新在飞期间用户可能已切换/登出账号。以发起时的 user_id
        // 为准，成功后重读当前会话——归属变了就只落库不激活（防旧账号串数据）。
        const refreshingUserId = authData?.user_id;

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
          return 'transport-error';
        } finally {
          httpRequest.destroy();
        }

        const payloadRecord = payload as Record<string, Object>;
        if (!payloadRecord || typeof payloadRecord.code !== 'number') return 'transport-error';

        if (payloadRecord.code === 0 && payloadRecord.data) {
          const newAuth = payloadRecord.data as AuthData;
          const current = await HttpUtil.getAuthData();
          if (refreshingUserId && current?.user_id && current.user_id !== refreshingUserId) {
            // 刷新在飞期间已切到其它账号：只更新该账号的存储，不激活、不覆盖
            // 当前会话（此前无条件 upsertAndActivate 会把旧账号重新激活串数据）。
            await AuthAccountManager.upsertOnly(newAuth);
            hilog.info(DOMAIN_NUMBER, TAG, 'Token 刷新成功（账号已切换，仅落库）');
            return 'ok';
          }
          // upsertAndActivate 内部经 Asset 存储更新该账号 AuthData 与会话快捷键
          await AuthAccountManager.upsertAndActivate(newAuth);
          hilog.info(DOMAIN_NUMBER, TAG, 'Token 刷新成功');
          return 'ok';
        }

        // 刷新失败（10100=refresh_token 无效、10201=账号被封禁等），清除认证状态
        hilog.warn(DOMAIN_NUMBER, TAG, `Token 刷新失败，code=${payloadRecord.code}`);
        await HttpUtil.clearAuthData();
        return 'invalid';
      } catch (error) {
        // 网络层异常（超时/断网/连接失败）≠ 凭证失效：spec 只要求 refresh_token
        // 无效（10100）时清除会话。弱网下走到这里的刷新请求会抛异常，若此时
        // clearAuthData 会把用户误踢回登录页——保留本地会话，仅返回刷新失败，
        // 后续网络恢复后可重试刷新。
        hilog.warn(DOMAIN_NUMBER, TAG, `refresh_token failed (network): ${JSON.stringify(error)}`);
        return 'transport-error';
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

      // 已过期（<=0）同样触发：前台恢复时 token 可能已过期，此前 >0 条件会跳过
      // 主动刷新，只能等首个请求吃 10100 再补刷；NaN（异常时间戳）不触发。
      if (!Number.isNaN(expiresAt) && expiresAt - now < thresholdMs) {
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
   *
   * 实现（管线化）：单次尝试封装在 performAttempt，各重试分支在改写
   * RequestContext 后由 request() 的有界循环重新进入——取代历史上
   * 8 参数 + 3 个递归计数器的自我递归。重试判定的先后顺序与原实现
   * 严格一致（封禁 → 限流 → 认证刷新 → 匿名降级 → 幂等重试 → 业务错误）。
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
    // mock 命中不进入任何重试/降级逻辑（与原实现一致）。
    if (MockInterceptor.isEnabled()) {
      const canned = MockInterceptor.match(url);
      if (canned) {
        hilog.debug(DOMAIN_NUMBER, TAG, `Mock response for ${url}`);
        return canned as TResponse;
      }
    }

    const ctx = new RequestContext();
    ctx.url = url;
    ctx.method = method;
    ctx.data = data as Object;
    ctx.retried = retried;
    ctx.forcedIdempotencyKey = forcedIdempotencyKey;
    ctx.idempotencyRetryCount = idempotencyRetryCount;
    ctx.omitAuthHeader = omitAuthHeader;
    ctx.rateLimitRetryCount = rateLimitRetryCount;

    try {
      // 有界循环：每个重试分支都有独立计数上限（retried/rl<1/idem<1），
      // 不存在无界重入。
      while (true) {
        const directive = await HttpUtil.performAttempt(ctx);
        if (directive === AttemptDirective.DONE) {
          return ctx.result as TResponse;
        }
        if (directive === AttemptDirective.FAIL) {
          throw ctx.failure!;
        }
        // RETRY：ctx 已被对应分支改写（换端点/换键/退避/计数）。
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '未知错误';
      if (errorMsg.includes('Network') || errorMsg.includes('timeout') || errorMsg.includes('connect')) {
        const baseUrl2 = await HttpUtil.getBaseUrl();
        hilog.error(DOMAIN_NUMBER, TAG, `网络连接问题，请检查服务器地址: ${baseUrl2}`);
      }
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(errorMsg);
    }
  }

  /**
   * 单次尝试：发送请求并按业务码判定 DONE / FAIL / RETRY。
   * RETRY 时本函数已完成对应的上下文改写与退避等待。
   */
  private static async performAttempt(ctx: RequestContext): Promise<AttemptDirective> {
    // 匿名请求路由（spec api_spec_base.md 节点身份语义）：/api/v1 必须携带有效
    // token，未登录时对「公开可浏览」端点直接改走 /public/v1 对偶端点，而不是
    // 打 /api/v1 稳定吃 10100。每次尝试顶部重估（换键/换端点重试后依然成立）。
    if (!ctx.omitAuthHeader && !(await HttpUtil.getToken())) {
      const alias = PUBLIC_ENDPOINT_ALIASES.get(ctx.url);
      if (alias) {
        hilog.info(DOMAIN_NUMBER, TAG, `未登录，匿名端点改走 public 节点: ${ctx.url} -> ${alias}`);
        ctx.url = alias;
      }
    }

    const baseUrl = await HttpUtil.getBaseUrl();
    const fullUrl = `${baseUrl}${ctx.url}`;
    const httpRequest = http.createHttp();
    const startTime = Date.now();
    hilog.info(DOMAIN_NUMBER, TAG, `HTTP请求: ${ctx.method} ${fullUrl}`);

    try {
      ctx.hasAuthHeader = !ctx.omitAuthHeader && !!(await HttpUtil.getToken());

      // 主动刷新：如果 token 即将过期，先刷新再发请求。
      // 认证端点（登录/刷新）本身不触发主动刷新——设备上残留的过期 token 会让
      // signin 前先跑一次 refresh，refresh 失败即 clearAuthData 并污染登录流程
      // （实测：账号密码正确却提示「登录已过期」）。
      const isAuthEndpoint = ctx.url === ApiEndpoints.AUTH.LOGIN || ctx.url === ApiEndpoints.AUTH.REFRESH_TOKEN;
      ctx.isAuthEndpoint = isAuthEndpoint;
      // 登录/刷新/访客是预会话端点，**永不携带 Authorization**：设备上残留的旧
      // token 会把登录请求本身变成 10100（后端先校验请求头再验凭据），
      // 用户看到的是「登录已过期」而非真实的凭据错误
      // （实测：curl 无认证头同凭据返回 code=0，app 带残留头同请求返回 10100）。
      if (isAuthEndpoint) {
        ctx.omitAuthHeader = true;
      }
      if (ctx.hasAuthHeader && !ctx.retried && !isAuthEndpoint) {
        await HttpUtil.proactiveRefreshIfNeeded();
      }

      ctx.sentToken = !ctx.omitAuthHeader ? ((await HttpUtil.getToken()) || '') : '';
      ctx.hasAuthHeader = !ctx.omitAuthHeader && !!ctx.sentToken;

      ctx.isMustIdempotent = MUST_IDEMPOTENT_ENDPOINTS.includes(ctx.url);
      ctx.idempotencyKey = ctx.isMustIdempotent
        ? (ctx.forcedIdempotencyKey || generateIdempotencyKey())
        : '';
      const headers = await HttpUtil.buildHeaders(ctx.url, ctx.idempotencyKey, ctx.omitAuthHeader);
      const requestOptions: http.HttpRequestOptions = {
        method: ctx.method,
        header: headers,
        extraData: ctx.data !== undefined ? JSON.stringify(ctx.data) : undefined,
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
        const resultCodeForLog = (result as Record<string, Object>)?.code;
        hilog.debug(DOMAIN_NUMBER, TAG, `响应 code=${resultCodeForLog ?? 'unknown'}`);
      } catch (parseError) {
        throw new Error(`JSON解析失败: ${parseError}`);
      }

      if (!isApiResponse(result)) {
        throw new Error('响应格式不符合 ApiResponse 约定');
      }

      const origin = HttpUtil.extractOrigin(baseUrl);
      HttpUtil.resolveRelativeMediaUrls(result, origin);

      const resultCode = (result as Record<string, Object>).code as number;

      // ── 封禁：强制登出 + 封禁页（终态）──────────────────────────
      if (isAccountBannedCode(resultCode)) {
        await HttpUtil.clearAuthData();
        const param = new LoginRouteParam();
        param.reason = 'banned';
        param.action = ((result as Record<string, Object>).message as string) || '账号已被封禁';
        AppRouter.clear();
        AppRouter.push(PageNames.LOGIN, param);
        ctx.failure = new ApiError('账号已被封禁', resultCode);
        return AttemptDirective.FAIL;
      }

      // ── 限流：按 Retry-After 等待后重试（至多一次）──────────────
      if (isRateLimitCode(resultCode) && ctx.rateLimitRetryCount < 1) {
        // 头名大小写不敏感取值：HTTP/2 下响应头统一小写（retry-after），
        // 精确匹配 'Retry-After' 会拿不到而退默认 5s。
        const headers = (response.header ?? {}) as Record<string, string>;
        const retryAfterHeader = headers['Retry-After'] ?? headers['retry-after'] ?? '5';
        const retryAfterSec = parseInt(retryAfterHeader) || 5;
        hilog.warn(DOMAIN_NUMBER, TAG, `收到 10500 限流，${retryAfterSec}s 后重试`);
        await new Promise<void>((resolve) => setTimeout(resolve, retryAfterSec * 1000));
        ctx.rateLimitRetryCount += 1;
        return AttemptDirective.RETRY;
      }

      // ── 认证失败：刷新 token 后重试（至多一次，键保持 sticky）────
      // 认证端点（登录/刷新）**不参与** 10100 刷新重试：signin 返回 10100 的语义是
      // 凭据错误/账号不存在，刷新旧 token 毫无意义——且「刷新成功后再试一次登录」
      // 会让登录尝试翻倍，实测直接加速触发服务端 10500 登录限流（900s），
      // 真实「凭据错误」错误也被第二次限流响应掩盖。与 2129ccd8（跳过主动刷新）同一根因的被动路径。
      if (!ctx.retried && ctx.hasAuthHeader && !ctx.isAuthEndpoint && isAuthErrorCode(resultCode)) {
        hilog.info(DOMAIN_NUMBER, TAG, '收到 10100 认证失败，尝试刷新 token');

        // 并发竞态防护：如果在此请求在飞期间并发请求已成功刷新了 token，
        // 则本地 token 已经更新，直接使用新 token 重试，避免冗余重复刷新。
        const currentToken = await HttpUtil.getToken();
        if (currentToken && ctx.sentToken && currentToken !== ctx.sentToken) {
          hilog.info(DOMAIN_NUMBER, TAG, 'Token 已被并发请求刷新，直接使用新 token 重试');
          ctx.retried = true;
          ctx.forcedIdempotencyKey = ctx.idempotencyKey;
          return AttemptDirective.RETRY;
        }

        const outcome = await HttpUtil.ensureRefreshToken();
        if (outcome === 'ok') {
          ctx.retried = true;
          ctx.forcedIdempotencyKey = ctx.idempotencyKey;
          return AttemptDirective.RETRY;
        }
        if (outcome === 'transport-error') {
          // 传输失败（部署滚动/弱网）：不得清除会话——抛可重试错误由上层
          // 提示，本地会话保留，网络恢复后自然续上。
          hilog.warn(DOMAIN_NUMBER, TAG, '令牌刷新传输失败，保持会话并上报可重试错误');
          ctx.failure = new ApiError('网络异常，请稍后重试');
          return AttemptDirective.FAIL;
        }
        hilog.warn(DOMAIN_NUMBER, TAG, '令牌刷新失败，已强制登出');
        await HttpUtil.clearAuthData();
        const param = new LoginRouteParam();
        param.reason = 'expired';
        AppRouter.clear();
        AppRouter.push(PageNames.LOGIN, param);
        // 不 return：落入下方匿名降级分支（原实现即如此——刷新失败的
        // 公开端点还有一次匿名重试机会）
      }

      // ── 认证失败：匿名降级（公开对偶节点 / token_optional 同节点）──
      if (isAuthErrorCode(resultCode) && ctx.hasAuthHeader && !ctx.omitAuthHeader) {
        const alias = PUBLIC_ENDPOINT_ALIASES.get(ctx.url);
        if (alias) {
          hilog.info(DOMAIN_NUMBER, TAG, `认证失败，公开端点降级为匿名 public 节点重试: ${ctx.url} -> ${alias}`);
          ctx.url = alias;
          ctx.omitAuthHeader = true;
          ctx.retried = true;
          ctx.forcedIdempotencyKey = ctx.idempotencyKey;
          return AttemptDirective.RETRY;
        }
        if (isTokenOptionalEndpoint(ctx.url)) {
          hilog.info(DOMAIN_NUMBER, TAG, `token_optional 端点认证失败，匿名降级重试: ${ctx.url}`);
          ctx.omitAuthHeader = true;
          ctx.retried = true;
          ctx.forcedIdempotencyKey = ctx.idempotencyKey;
          return AttemptDirective.RETRY;
        }
      }

      // ── 幂等错误自动重试（仅 MUST 幂等端点，至多 MAX 次）─────────
      if (ctx.isMustIdempotent && ctx.idempotencyRetryCount < MAX_IDEMPOTENCY_RETRY_COUNT) {
        if (resultCode === ErrorCode.IDEMPOTENCY_IN_FLIGHT) {
          await new Promise<void>((resolve) => setTimeout(resolve, 1200));
          ctx.idempotencyRetryCount += 1;
          ctx.forcedIdempotencyKey = ctx.idempotencyKey;
          return AttemptDirective.RETRY;
        }
        if (resultCode === ErrorCode.IDEMPOTENCY_KEY_CONFLICT) {
          hilog.warn(DOMAIN_NUMBER, TAG, `收到 ${ErrorCode.IDEMPOTENCY_KEY_CONFLICT} 幂等键冲突，换新键重试`);
          ctx.idempotencyRetryCount += 1;
          ctx.forcedIdempotencyKey = generateIdempotencyKey();
          return AttemptDirective.RETRY;
        }
        if (resultCode === ErrorCode.IDEMPOTENCY_CACHE_EXPIRED) {
          hilog.warn(DOMAIN_NUMBER, TAG, `收到 ${ErrorCode.IDEMPOTENCY_CACHE_EXPIRED} 幂等缓存过期，换新键重试`);
          ctx.idempotencyRetryCount += 1;
          ctx.forcedIdempotencyKey = generateIdempotencyKey();
          return AttemptDirective.RETRY;
        }
      }

      // ── 业务错误：类型化 ApiError（终态）────────────────────────
      if (resultCode !== 0) {
        const snapshot = extractApiErrorSnapshot(result, response.responseCode);
        const userMessage = guessUserMessageFromSnapshot(snapshot, '操作失败');
        hilog.error(DOMAIN_NUMBER, TAG, `业务错误 [${snapshot.code}]: ${userMessage}`);

        let responseData: Object | undefined = undefined;
        let forbiddenReason: string | undefined = undefined;
        const rawData = (result as Record<string, Object>).data;
        if (rawData !== undefined && rawData !== null) {
          responseData = rawData;
          const dataReason = (rawData as Record<string, Object>).forbidden_reason;
          if (typeof dataReason === 'string' && dataReason.length > 0) {
            forbiddenReason = dataReason;
          }
        }
        ctx.failure = new ApiError(userMessage, snapshot.code, snapshot.status, responseData, forbiddenReason);
        return AttemptDirective.FAIL;
      }

      hilog.info(DOMAIN_NUMBER, TAG, `请求成功: ${ctx.method} ${ctx.url} (${responseTime}ms)`);
      ctx.result = result;
      return AttemptDirective.DONE;
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
