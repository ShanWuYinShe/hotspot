// API client — thin fetch wrapper for JSON POST calls.
// Cookie-based auth (HttpOnly) — no Bearer token needed.

import { Endpoints } from '@/api/endpoints';
import type {
	AccountData,
	ApiVersionMeta,
	SessionStatus,
	AuthTokenPair,
	BootstrapData,
	ClientConfig,
	CommentItem,
	RelatedUserItem,
	GroupAnnouncementItem,
	GroupItem,
	GroupMemberItem,
	AdminGroupMemberItem,
	GroupSettings,
	GroupMessageItem,
	JoinGroupResult,
	GroupApplicationItem,
	GroupInviteItem,
	GroupSentInviteItem,
	AdminBannedMemberItem,
	Invitation,
	InviteAcceptedResult,
	SearchGroupsRequest,
	UpdateGroupRequest,
	DismissGroupRequest,
	TransferOwnershipRequest,
	SearchGroupMembersRequest,
	CreateGroupInviteRequest,
	ListSentInvitesRequest,
	RevokeGroupInviteRequest,
	SetMemberRoleRequest,
	SetMemberMuteRequest,
	SetMemberBanRequest,
	BannedMembersRequest,
	RemoveMemberRequest,
	GroupApplicationsRequest,
	ApproveGroupApplicationRequest,
	RejectGroupApplicationRequest,
	CreateGroupAnnouncementRequest,
	UpdateGroupAnnouncementRequest,
	DeleteGroupAnnouncementRequest,
	ListSubmittedApplicationsRequest,
	CancelApplicationRequest,
	ListReceivedInvitesRequest,
	AcceptInviteRequest,
	RejectInviteRequest,
	PresignedUpload,
	PresignUploadRequest,
	PresignedUploads,
	CompletedUploads,
	UploadedMedia,
	MessageSentResult,
	MomentItem,
	OwnerMomentItem,
	FavoritedMomentItem,
	NotifyItem,
	AdminPermissions,
	AdminPermissionDictionary,
	AdminRoleList,
	AdminRoleItem,
	AdminOperatorItem,
	AuditAction,
	AuditTargetType,
	AuditLogItem,
	AdminGeoStatsResponse,
	AdminGeoDrilldownPage,
	StatsTimeRange,
	AppConfig,
	UpdateAppConfigRequest,
	UserAccountStatus,
	AdminUserItem,
	AdminGroupItem,
	AdminReportItem,
	ReportItem,
	AdminTagItem,
	AdminAnnouncementItem,
	ReportAction,
	ReportStatus,
	ReportTargetType,
	AnnounceSeverity,
	UserType,
	PaginatedData,
	PrivateChatItem,
	PrivateChatOpenResult,
	PrivateMessageItem,
	ReportCreated,
	UnreadSummaryResult,
	UserItem,
	UserProfileBundle,
	UserSettings,
	UpdateSettingsRequest,
	DeleteAccountRequest,
	CreateGroupRequest,
	MapCanvasRequest,
	MapCanvasResponse,
	SearchSuggestRequest,
	SearchSuggestResponse,
	RegisterRequest,
	LoginRequest,
	GuestRequest,
	SendRegisterCodeRequest,
	RequestPasswordResetRequest,
	ResetPasswordRequest,
	ChangePasswordRequest,
	RevokeSessionRequest,
	SessionList,
	RevokeOtherSessionsResult,
	UpgradeAccountRequest,
	FetchBlockedRequest,
	BlockedUserItem,
	CreateShareRequest,
	ShareCreated,
	UpdateAccountRequest,
	UpdateLocationRequest,
	SetFollowRequest,
	SetBlockRequest,
	SearchUsersRequest,
	CreateMomentRequest,
	UpdateMomentRequest,
	SearchMomentsRequest,
	FavoriteMomentRequest,
	FetchFollowingMomentsRequest,
	FetchNearbyMomentsRequest,
	SendPrivateMessageRequest,
	ListPrivateMessagesRequest,
	SendGroupMessageRequest,
	ListGroupMessagesRequest,
	MarkPrivateReadRequest,
	MarkGroupReadRequest,
	UpdatePrivateChatPreferencesRequest,
	UpdateGroupPreferencesRequest,
	GetGroupSettingsRequest,
	ListAdminGroupMembersRequest,
	RecallGroupMessageAdminRequest,
	SetReactionRequest,
	CreateCommentRequest,
	DeleteCommentRequest,
	CreateReportRequest,
	MarkNotifyReadRequest,
	CompleteUploadRequest,
	BootstrapRequest,
	PaginationParams,
	ShareDetail,
} from '@/api/types';

// 分页信封由 types.ts 提供；此处集中重导出供页面直接使用。
export type { PaginatedData };

export interface ApiResponse<T> {
  code: number;
  message?: string;
  data: T;
}

let baseUrl = '';
let refreshLock: Promise<RefreshOutcome> | null = null;
let authenticated = false;

/** 供 auth store 在登录态变化时同步，驱动双版本端点选择（/public/v1 vs /api/v1）。 */
export function setAuthenticated(value: boolean) {
  authenticated = value;
}

export function pick(apiEndpoint: string, publicEndpoint: string): string {
  return authenticated ? apiEndpoint : publicEndpoint;
}

export function setApiBaseUrl(url: string) {
  baseUrl = url.replace(/\/$/, '');
}

/**
 * DeviceId：幂等键的 client_id 段（spec: api_spec_base.md「幂等性」client_id=DeviceId）。
 * 优先读后端下发的 device_id Cookie（非 HttpOnly、10 年长效、登出不清除）；
 * 无 Cookie 可读时（首请求前/非浏览器环境）回落 localStorage 持久化的随机 UUID，
 * 保证同设备重试稳定。
 */
function getDeviceId(): string {
  if (typeof document !== 'undefined') {
    const m = document.cookie.match(/(?:^|;\s*)device_id=([^;]*)/);
    if (m?.[1]) return decodeURIComponent(m[1]);
  }
  if (typeof localStorage !== 'undefined') {
    let id = localStorage.getItem('hotspot:device-id');
    if (!id) {
      id = typeof crypto !== 'undefined' && crypto.randomUUID
        ? crypto.randomUUID()
        : `web-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
      try { localStorage.setItem('hotspot:device-id', id); } catch {}
    }
    return id;
  }
  return 'hotspot-web';
}

/**
 * 生成幂等键（spec: api_spec_base.md「幂等性（契约级）」）。
 * 格式：{client_id}-{timestamp_ms}-{random}，长度 32-64，字符集 [a-zA-Z0-9-_]；
 * 服务端按该正则校验，超长/非法 key 会被当作无 key 静默跳过幂等。
 * DeviceId 长度无固定契约（服务端 bootstrap 下发 UUID v4，自定义客户端标识更长），
 * 截断到 24 保总量 ≤64。
 */
export function newIdempotencyKey(): string {
  const clientId = getDeviceId().replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 24) || 'hotspot-web';
  const timestampMs = Date.now().toString();
  const random = Math.random().toString(36).slice(2, 14).replace(/[^a-zA-Z0-9_-]/g, '');
  const key = `${clientId}-${timestampMs}-${random}`;
  return key.length >= 32 ? key : `${key}-${Math.random().toString(36).slice(2, 10)}`;
}

/** 生成请求链路追踪 ID（spec: api_spec_base.md「请求与响应头」，可选、无格式约束）。 */
export function newRequestId(): string {
  return newIdempotencyKey();
}

/**
 * Notify the app of an auth-state change observed at the transport layer.
 * 由 auth store 监听这些事件，API 层自身不依赖 UI 框架。
 *   'auth-failed' → session invalid / refresh failed → drop to anonymous.
 *   'auth-banned' → 10201 account banned → show ban screen.
 */
function emitAuthEvent(type: 'auth-failed' | 'auth-banned', message?: string) {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(type, { detail: { type, message } }));
}

/** 已知的 access 过期时间（ms epoch；由 checkAuthStatus 习得，供主动续期用）。 */
let knownAccessExpiresAt: number | null = null;

/** 由 auth store 在会话检查成功/登出时同步（成功带服务端下发的 expires_at）。 */
export function setKnownAccessExpiry(iso: string | null | undefined): void {
  if (!iso) {
    knownAccessExpiresAt = null;
    return;
  }
  const t = Date.parse(iso);
  knownAccessExpiresAt = Number.isFinite(t) ? t : null;
}

/** 提前续期阈值（对齐鸿蒙 3 分钟）：临期请求先静默续期，避免 10100 往返撞上服务端不可用窗口。 */
const PROACTIVE_REFRESH_THRESHOLD_MS = 3 * 60 * 1000;

/**
 * 主动续期：access 临期时先静默刷新，成功后重学过期时间。
 * 失败一律忽略（invalid → 后续请求走 10100 正常锁存；transport → 同理不锁存），
 * 绝不代替调用方做登录态判决。
 */
async function proactiveRefreshIfNeeded(): Promise<void> {
  if (knownAccessExpiresAt == null) return;
  if (knownAccessExpiresAt - Date.now() >= PROACTIVE_REFRESH_THRESHOLD_MS) return;
  const outcome = await refreshSessionToken();
  if (outcome !== 'ok') return;
  try {
    const st = await request<SessionStatus>(Endpoints.Auth.Session, {}, {}, 0, newRequestId(), { skipProactive: true });
    if (st.code === 0 && st.data) setKnownAccessExpiry(st.data.expires_at);
  } catch {
    // 重学失败忽略：被动续期链兜底。
  }
}

/** 默认请求超时：挂起请求不再让页面 spinner 永久停留（上传走 XHR 独立通道不受此限）。 */
const DEFAULT_REQUEST_TIMEOUT_MS = 15_000;

export interface RequestOptions {
	/** 外部中止信号（如页面卸载/竞态清理）；与内部超时任一触发即中止请求。 */
	signal?: AbortSignal;
	/** 覆盖默认超时（ms）。 */
	timeoutMs?: number;
	/** 跳过主动续期（内部续期后的重学请求防重入）。 */
	skipProactive?: boolean;
}

// 导出供 ./domains/* 端点包装使用（facade 亦经 export * 透出）。
export async function request<T>(
  endpoint: string,
  body?: unknown,
  extraHeaders: Record<string, string> = {},
  retryCount = 0,
  requestId = newRequestId(),
  opts?: RequestOptions,
): Promise<ApiResponse<T>> {
  const url = `${baseUrl}${endpoint}`;
  // 主动续期（临期静默刷新；跳过标记用于内部重学防重入）。
  if (!opts?.skipProactive) await proactiveRefreshIfNeeded();
  // 超时与外部 signal 合流到内部 controller（不经 AbortSignal.any，兼容测试环境）
  const controller = new AbortController();
  const timer = setTimeout(
    () => controller.abort(),
    opts?.timeoutMs ?? DEFAULT_REQUEST_TIMEOUT_MS,
  );
  const abortFromCaller = () => controller.abort();
  if (opts?.signal) {
    if (opts.signal.aborted) abortFromCaller();
    else opts.signal.addEventListener('abort', abortFromCaller, { once: true });
  }
  let res: Response;
  let json: ApiResponse<T> | null;
  try {
    res = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', 'X-Request-Id': requestId, ...extraHeaders },
      body: body ? JSON.stringify(body) : '{}',
      signal: controller.signal,
    });
    // 解析失败（网关 502/504 常回 HTML）单独兜住为 null：不能让裸 SyntaxError
    // 吞掉下方 !res.ok 的 HTTP 状态错误分支。
    try {
      json = await res.json();
    } catch {
      json = null;
    }
  } catch (err) {
    // 超时/取消在 fetch 层表现为 AbortError——映射为可读文案，让调用方按
    // 网络错误提示/重试，而非裸 "The operation was aborted"。
    if (controller.signal.aborted) {
      throw new Error(opts?.signal?.aborted ? '请求已取消' : '请求超时，请检查网络后重试');
    }
    throw err;
  } finally {
    clearTimeout(timer);
    opts?.signal?.removeEventListener('abort', abortFromCaller);
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    msg = (json as unknown as { message?: string } | null)?.message ?? msg;
    // A non-200 transport failure is unexpected per spec (business errors come
    // back as 200 + code). Surface it; AuthContext treats a thrown request as
    // a soft failure the caller can handle.
    throw new Error(msg);
  }
  if (!json) {
    // 200 但响应体不是 JSON（代理注入页/损坏响应等）——按可重试传输错误处理。
    throw new Error('响应格式异常，请稍后重试');
  }

  // 10201: account banned — force-drop and show the ban screen with the message.
  if (json.code === 10201) {
    emitAuthEvent('auth-banned', json.message);
    return json;
  }

  // 10500: rate limited. Per spec the response carries Retry-After (seconds);
  // wait (capped at 10s to keep UI responsive) and retry once. 与鸿蒙端语义对齐。
  if (json.code === 10500 && retryCount < 1) {
    const retryAfter = Number(res.headers.get('retry-after'));
    const waitSec = Number.isFinite(retryAfter) ? Math.min(Math.max(retryAfter, 0), 10) : 5;
    await new Promise((resolve) => setTimeout(resolve, waitSec * 1000));
    return request<T>(endpoint, body, extraHeaders, retryCount + 1, requestId, { ...opts, skipProactive: true });
  }

  // 10100: access token invalid/expired. Try a single deduped refresh + retry.
  if (json.code === 10100 && retryCount < 1) {
    const outcome = await refreshSessionToken();
    // skipProactive：刚刷新出的新 token knownAccessExpiresAt 未知，重试时若再走
    // proactiveRefreshIfNeeded 会立即触发第二次无谓刷新（轮换竞态）。
    if (outcome === 'ok') return request<T>(endpoint, body, extraHeaders, retryCount + 1, requestId, { ...opts, skipProactive: true });
    if (outcome === 'invalid') {
      // Refresh failed: notify AuthContext so it can drop to anonymous and prompt.
      emitAuthEvent('auth-failed');
      return json;
    }
    // transport-error（后端滚动/网关抖动等不可达）：不得锁存登出——抛可重试
    // 错误由调用方处理（toast/重试），登录态保持，待服务端恢复后自然续上。
    throw new Error('网络异常，请稍后重试');
  }

  return json;
}

/**
 * 刷新结果三态：ok（可重试原请求）/ invalid（会话确已失效，调用方锁存登出）/
 * transport-error（服务端不可达等传输失败，调用方不得锁存登出、按可重试处理）。
 * 背景：部署滚动窗口内续期恰好撞上后端不可用时有大量用户在线，以往不分青红
 * 皂白统一锁存匿名，导致“每次发版都掉登录”且永不自动恢复。
 */
export type RefreshOutcome = 'ok' | 'invalid' | 'transport-error';

/**
 * 主动刷新会话 token（单飞：并发调用共享同一次刷新）。
 * 消费方：HTTP 10100 重试（request 内）与 WS 4001 重连（ws client 注入）。
 */
export async function refreshSessionToken(): Promise<RefreshOutcome> {
  if (!refreshLock) {
    refreshLock = (async (): Promise<RefreshOutcome> => {
      let res: Response;
      try {
        // spec api_spec_public.md → RefreshTokenRequest：client_type 必填；
        // Web 端 device_id 为 null（服务端从 device_id Cookie 读取）、
        // refresh_token 为 null（必须为空值，走 Path 限定的 HttpOnly Cookie）。
        // 曾发 '{}' 导致服务端恒回 10000，所有 Web 会话 15 分钟静默死亡。
        res = await fetch(`${baseUrl}${Endpoints.Auth.RefreshToken}`, {
          method: 'POST', credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ client_type: 'web', device_id: null, refresh_token: null }),
        });
      } catch {
        return 'transport-error';
      }
      if (!res.ok) return 'transport-error';
      let code: number;
      try {
        code = (await res.json()).code;
      } catch {
        return 'transport-error';
      }
      return code === 0 ? 'ok' : 'invalid';
    })().finally(() => { refreshLock = null; });
  }
  return refreshLock;
}


// netError —— 网络层失败（fetch 抛错/超时）时的统一兜底响应：
// code=-1 与业务错误码空间区分，data 传调用方的空态回退（null / { items: [] }）。
export function netError<T>(fallback: T): ApiResponse<T> {
	return { code: -1, message: '网络异常', data: fallback };
}
