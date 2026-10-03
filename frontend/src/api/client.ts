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

function pick(apiEndpoint: string, publicEndpoint: string): string {
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
 * DeviceId 最长可达 41 字符（mock 前缀形态），截断到 24 保总量 ≤64。
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

interface RequestOptions {
	/** 外部中止信号（如页面卸载/竞态清理）；与内部超时任一触发即中止请求。 */
	signal?: AbortSignal;
	/** 覆盖默认超时（ms）。 */
	timeoutMs?: number;
	/** 跳过主动续期（内部续期后的重学请求防重入）。 */
	skipProactive?: boolean;
}

async function request<T>(
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
  let json: ApiResponse<T>;
  try {
    res = await fetch(url, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', 'X-Request-Id': requestId, ...extraHeaders },
      body: body ? JSON.stringify(body) : '{}',
      signal: controller.signal,
    });
    json = await res.json();
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
    try { msg = (json as unknown as any).message ?? msg; } catch {}
    // A non-200 transport failure is unexpected per spec (business errors come
    // back as 200 + code). Surface it; AuthContext treats a thrown request as
    // a soft failure the caller can handle.
    throw new Error(msg);
  }

  // 10201: account banned — force-drop and show the ban screen with the message.
  if (json.code === 10201) {
    emitAuthEvent('auth-banned', json.message);
    return json;
  }

  // 10100: access token invalid/expired. Try a single deduped refresh + retry.
  if (json.code === 10100 && retryCount < 1) {
    const outcome = await refreshSessionToken();
    if (outcome === 'ok') return request<T>(endpoint, body, extraHeaders, retryCount + 1, requestId, opts);
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

// Auth
export async function register(params: RegisterRequest)           { return request<AuthTokenPair>(Endpoints.Auth.Register, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function login(params: LoginRequest)              { return request<AuthTokenPair>(Endpoints.Auth.Login, params); }
export async function guest(params?: GuestRequest)             { return request<AuthTokenPair>(Endpoints.Auth.Guest, params ?? {}, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function checkAuthStatus()                   { return request<SessionStatus>(Endpoints.Auth.Session); }
export async function logout()                            { return request<void>(Endpoints.Auth.Signout); }
export async function requestPasswordReset(params: RequestPasswordResetRequest) { return request<void>(Endpoints.Auth.RequestPasswordReset, params); }
export async function resetPassword(params: ResetPasswordRequest)      { return request<AuthTokenPair>(Endpoints.Auth.ResetPassword, params); }
// spec: api_spec_public.md → auth/send_register_code（手机号/邮箱型 login_id 的所有权验证，防枚举恒成功）
export async function sendRegisterCode(params: SendRegisterCodeRequest) { return request<void>(Endpoints.Auth.SendRegisterCode, params); }
// spec: api_spec_auth.md → auth/change_password（成功后服务端自动吊销其余会话；10401=旧密码错误）
export async function changePassword(params: ChangePasswordRequest) { return request<void>(Endpoints.Auth.ChangePassword, params); }
// spec: api_spec_auth.md → auth/fetch_sessions（全量活跃会话，当前会话置顶，无分页）
export async function fetchSessions() { return request<SessionList>(Endpoints.Auth.FetchSessions); }
// spec: api_spec_auth.md → auth/revoke_session（不允许撤销当前会话，10400；当前会话走 signout）
export async function revokeSession(params: RevokeSessionRequest) { return request<void>(Endpoints.Auth.RevokeSession, params); }
// spec: api_spec_auth.md → auth/revoke_other_sessions（撤销当前会话之外的所有其他活跃会话）
export async function revokeOtherSessions() { return request<RevokeOtherSessionsResult>(Endpoints.Auth.RevokeOtherSessions); }

// Users
export async function getAccount()                        { return request<AccountData>(Endpoints.Users.GetAccount); }
export async function updateAccount(params: UpdateAccountRequest)       { return request<void>(Endpoints.Users.UpdateAccount, params); }
export async function getSettings()                       { return request<UserSettings>(Endpoints.Users.GetSettings); }
// spec: api_spec_user.md → users/update_settings（request 为 { settings: Partial<UserSettings> }）
export async function updateSettings(params: { settings: Partial<UserSettings> }) {
	return request<void>(Endpoints.Users.UpdateSettings, params);
}
// spec: api_spec_user.md → users/update_location（真实位置上报：用于被他人发现/展示大致位置）。
// 不在 MUST 幂等清单，不加 Idempotency-Key。
export async function updateLocation(params: UpdateLocationRequest) {
	return request<void>(Endpoints.Users.UpdateLocation, params);
}
export async function follow(params: SetFollowRequest)              { return request<void>(Endpoints.Users.Follow, params); }
export async function block(params: SetBlockRequest)               { return request<void>(Endpoints.Users.Block, params); }
// 解除拉黑 = set_block 两态统一端点的 active=false 侧（spec: api_spec_user.md → users/set_block）
export async function unblock(params: Omit<SetBlockRequest, 'active'>) { return block({ ...params, active: false }); }
// spec: api_spec_user.md → users/fetch_blocked（我的拉黑列表，blocked_at 倒序）
export async function fetchBlocked(params?: FetchBlockedRequest)   { return request<PaginatedData<BlockedUserItem>>(Endpoints.Users.ListBlocked, params ?? { pagination: null }); }
// spec: api_spec_user.md → users/upgrade（访客绑定登录凭证升级为正式账号；返回升级后完整账户数据）
export async function upgradeAccount(params: UpgradeAccountRequest) { return request<AccountData>(Endpoints.Users.Upgrade, params); }
export async function deleteAccount(params: object)       { return request<void>(Endpoints.Users.DeleteAccount, params); }
// spec: api_spec_user.md → users/open_the_door（include_recent_moments=true 时返回主页首屏聚合）
export async function openTheDoor(userId: string, includeRecentMoments = false) {
	return request<UserProfileBundle>(pick(Endpoints.Users.GetDetail, Endpoints.Public.Users.GetDetail), { user_id: userId, include_recent_moments: includeRecentMoments });
}
// spec: api_spec_user.md → users/set_remark（需已关注对方；remark 传空串清除备注）
export async function setRemark(params: { user_id: string; remark: string }) { return request<void>(Endpoints.Users.SetRemark, params); }
export async function listFriends(params: { user_id: string; pagination?: PaginationParams })        { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFriends, Endpoints.Public.Users.ListFriends), params); }
export async function listFollowing(params: { user_id: string; pagination?: PaginationParams })      { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFollowing, Endpoints.Public.Users.ListFollowing), params); }
export async function listFollowers(params: { user_id: string; pagination?: PaginationParams })      { return request<PaginatedData<RelatedUserItem>>(pick(Endpoints.Users.ListFollowers, Endpoints.Public.Users.ListFollowers), params); }

// Media（spec: api_spec_media.md）—— presign → PUT 直传 → complete 三步流程的 API 包装
// presign_upload/complete_upload 均为数组语义，天然覆盖单/批，无独立批量端点。
export async function presignUpload(params: PresignUploadRequest) {
	return request<PresignedUploads>(Endpoints.Media.PresignUpload, params);
}
export async function completeUpload(params: CompleteUploadRequest) {
	return request<CompletedUploads>(Endpoints.Media.CompleteUpload, params);
}

// Moments
export async function searchMoments(params: SearchMomentsRequest)       { return request<PaginatedData<MomentItem>>(pick(Endpoints.Moments.Search, Endpoints.Public.Moments.Search), params); }
export async function getMomentDetail(momentId: string)   { return request<MomentItem>(pick(Endpoints.Moments.GetDetail, Endpoints.Public.Moments.GetDetail), { moment_id: momentId }); }
export async function createMoment(params: CreateMomentRequest)        { return request<OwnerMomentItem>(Endpoints.Moments.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateMoment(params: UpdateMomentRequest)        { return request<OwnerMomentItem>(Endpoints.Moments.Update, params); }
export async function deleteMoment(params: { moment_id: string }) { return request<void>(Endpoints.Moments.Delete, params); }
// spec: api_spec_moment.md → moments/fetch_created（拥有者视角，含 private 与软删占位）
export async function listMyMoments(params?: { pagination?: PaginationParams })      { return request<PaginatedData<OwnerMomentItem>>(Endpoints.Moments.ListCreated, params ?? {}); }
export async function getFavoriteMoments(params?: { pagination?: PaginationParams })   { return request<PaginatedData<FavoritedMomentItem>>(Endpoints.Moments.ListFavorites, params ?? {}); }
// spec: api_spec_moment.md → moments/fetch_following（关注动态；依赖查看者关注关系，无匿名版）
export async function fetchFollowingMoments(params?: FetchFollowingMomentsRequest)  { return request<PaginatedData<MomentItem>>(Endpoints.Moments.FetchFollowing, params ?? {}); }
// spec: api_spec_moment.md → moments/fetch_nearby（附近瞬间，原 feed/nearby_moments 迁移；双版本）
export async function fetchNearbyMoments(params?: FetchNearbyMomentsRequest)        { return request<PaginatedData<MomentItem>>(pick(Endpoints.Moments.FetchNearby, Endpoints.Public.Moments.FetchNearby), params ?? {}); }

// Groups
export async function getGroupDetail(groupId: string)     { return request<GroupItem>(pick(Endpoints.Groups.GetDetail, Endpoints.Public.Groups.GetDetail), { group_id: groupId }); }
export async function createGroup(params: CreateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_group.md → groups/fetch_joined（成员视角，viewer_state 必非 null）
export async function listMyGroups(params?: { pagination?: PaginationParams })       { return request<PaginatedData<GroupItem>>(Endpoints.Groups.ListJoined, params ?? {}); }
// spec: api_spec_group.md → groups/enter（direct 直接加入 / approval 创建申请）、groups/exit
export async function joinGroup(groupId: string, extra?: { password?: string; message?: string }) {
	return request<JoinGroupResult>(Endpoints.Groups.Join, { group_id: groupId, ...extra });
}
export async function leaveGroup(groupId: string)          { return request<void>(Endpoints.Groups.Leave, { group_id: groupId }); }

// Search
export async function searchUsers(params: SearchUsersRequest)         { return request<PaginatedData<UserItem>>(pick(Endpoints.Users.Search, Endpoints.Public.Users.Search), params); }

// Notify
export async function getUnreadCount()                    { return request<UnreadSummaryResult>(Endpoints.Notify.GetUnreadCount); }
export async function listNotifies(params?: { pagination?: PaginationParams })  { return request<PaginatedData<NotifyItem>>(Endpoints.Notify.List, params ?? {}); }
// spec: notify.md → mark_read（参数必显式）：scope='ids' 指定 1-50 条；scope='all' 全部已读。
export async function markRead(params: MarkNotifyReadRequest)            { return request<void>(Endpoints.Notify.MarkRead, params); }

// System
export async function getClientConfig()                   { return request<ClientConfig>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: BootstrapRequest)          { return request<BootstrapData>(pick(Endpoints.System.Bootstrap, Endpoints.Public.System.Bootstrap), params ?? {}); }

// Reports
export async function createReport(params: CreateReportRequest)        { return request<ReportCreated>(Endpoints.Reports.Create, params); }

// Chat
export async function getPrivateChats(params?: { pagination?: PaginationParams })    { return request<PaginatedData<PrivateChatItem>>(Endpoints.PrivateChats.List, params ?? {}); }
// spec: api_spec_chat.md → private_chats/delete { target_user_id }（仅删除自己视角的会话）
export async function deletePrivateChat(params: { target_user_id: string }) { return request<void>(Endpoints.PrivateChats.Delete, params); }
export async function getPrivateChatDetail(params: { target_user_id: string }) { return request<PrivateChatOpenResult>(Endpoints.PrivateChats.GetDetail, params); }
export async function getPrivateMessages(params: ListPrivateMessagesRequest)  { return request<PaginatedData<PrivateMessageItem>>(Endpoints.PrivateMessages.List, params); }
export async function sendPrivateMessage(params: SendPrivateMessageRequest)  { return request<MessageSentResult>(Endpoints.PrivateMessages.Send, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function getGroupMessages(params: ListGroupMessagesRequest)    { return request<PaginatedData<GroupMessageItem>>(Endpoints.GroupMessages.List, params); }
export async function sendGroupMessage(params: SendGroupMessageRequest)    { return request<MessageSentResult>(Endpoints.GroupMessages.Send, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_chat.md → 消息撤回/编辑（recall 受窗口限制/群管理员豁免，update 仅 type=text）
export async function recallPrivateMessage(params: { message_id: string }) { return request<void>(Endpoints.PrivateMessages.Recall, params); }
export async function updatePrivateMessage(params: { message_id: string; content: string }) { return request<PrivateMessageItem>(Endpoints.PrivateMessages.Update, params); }
export async function recallGroupMessage(params: { message_id: string }) { return request<void>(Endpoints.GroupMessages.Recall, params); }
export async function updateGroupMessage(params: { message_id: string; content: string }) { return request<GroupMessageItem>(Endpoints.GroupMessages.Update, params); }
export async function markPrivateRead(params: MarkPrivateReadRequest)     { return request<void>(Endpoints.PrivateMessages.MarkRead, params); }
export async function markGroupRead(params: MarkGroupReadRequest)       { return request<void>(Endpoints.GroupMessages.MarkRead, params); }
// spec: api_spec_chat.md → private_chats/update_preferences（静音/置顶）
export async function updatePrivateChatPreferences(params: UpdatePrivateChatPreferencesRequest) { return request<void>(Endpoints.PrivateChats.UpdatePreferences, params); }
// spec: api_spec_group.md → groups/update_preferences（静音/置顶）
export async function updateGroupPreferences(params: UpdateGroupPreferencesRequest) { return request<void>(Endpoints.Groups.UpdatePreferences, params); }

// Reactions
export async function setReaction(params: SetReactionRequest)         { return request<void>(Endpoints.Reactions.Set, params); }

// Comments
export async function createComment(params: CreateCommentRequest)       { return request<CommentItem>(Endpoints.Comments.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_comment.md → comments/delete（仅评论作者；软删除——顶层删后子评论保留，展示占位）
export async function deleteComment(params: DeleteCommentRequest)       { return request<void>(Endpoints.Comments.Delete, params); }

// Moments — favorite (two-state via active per spec)
export async function favoriteMoment(params: FavoriteMomentRequest)      { return request<void>(Endpoints.Moments.Favorite, params); }

// Groups extended
export async function getGroupMembers(groupId: string, pagination?: PaginationParams) {
  return request<PaginatedData<GroupMemberItem>>(pick(Endpoints.Groups.Members.List, Endpoints.Public.Groups.Members.List), { group_id: groupId, pagination });
}
export async function getGroupAnnouncements(groupId: string) {
  return request<{ items: GroupAnnouncementItem[] }>(Endpoints.Groups.Announcements.List, { group_id: groupId });
}

// Groups admin（spec: api_spec_group.md → 群组管理：搜索/更新/解散/转让/申请/邀请/成员/公告）
export async function searchGroups(params: SearchGroupsRequest) { return request<PaginatedData<GroupItem>>(pick(Endpoints.Groups.Search, Endpoints.Public.Groups.Search), params); }
export async function updateGroup(params: UpdateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Admin.Update, params); }
export async function dismissGroup(params: DismissGroupRequest) { return request<void>(Endpoints.Groups.Admin.Dismiss, params); }
export async function transferGroupOwnership(params: TransferOwnershipRequest) { return request<void>(Endpoints.Groups.Admin.TransferOwnership, params); }
export async function listMyApplications(params?: ListSubmittedApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.ListSubmittedApplications, params ?? {}); }
export async function cancelGroupApplication(params: CancelApplicationRequest) { return request<void>(Endpoints.Groups.CancelApplication, params); }
export async function listMyInvites(params?: ListReceivedInvitesRequest) { return request<PaginatedData<GroupInviteItem>>(Endpoints.Groups.ListReceivedInvites, params ?? {}); }
export async function acceptGroupInvite(params: AcceptInviteRequest) { return request<InviteAcceptedResult>(Endpoints.Groups.AcceptInvite, params); }
export async function rejectGroupInvite(params: RejectInviteRequest) { return request<void>(Endpoints.Groups.RejectInvite, params); }

export async function searchGroupMembers(params: SearchGroupMembersRequest) { return request<PaginatedData<GroupMemberItem>>(pick(Endpoints.Groups.Members.Search, Endpoints.Public.Groups.Members.Search), params); }
export async function inviteGroupMember(params: CreateGroupInviteRequest) { return request<Invitation>(Endpoints.Groups.Admin.Invites.Create, params); }
export async function setGroupMemberRole(params: SetMemberRoleRequest) { return request<void>(Endpoints.Groups.Admin.Members.SetRole, params); }
export async function setGroupMemberMute(params: SetMemberMuteRequest) { return request<void>(Endpoints.Groups.Admin.Members.SetMute, params); }
export async function banGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Admin.Members.SetBan, { ...params, active: true }); }
export async function unbanGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Admin.Members.SetBan, { ...params, active: false }); }
export async function listBannedGroupMembers(params: BannedMembersRequest) { return request<PaginatedData<AdminBannedMemberItem>>(Endpoints.Groups.Admin.Members.ListBanned, params); }
export async function removeGroupMember(params: RemoveMemberRequest) { return request<void>(Endpoints.Groups.Admin.Members.Remove, params); }
export async function listSentGroupInvites(params: ListSentInvitesRequest) { return request<PaginatedData<GroupSentInviteItem>>(Endpoints.Groups.Admin.Invites.ListSent, params); }
export async function revokeSentGroupInvite(params: RevokeGroupInviteRequest) { return request<void>(Endpoints.Groups.Admin.Invites.Revoke, params); }

export async function listGroupApplications(params: GroupApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.Admin.Applications.List, params); }
export async function approveGroupApplication(params: ApproveGroupApplicationRequest) { return request<void>(Endpoints.Groups.Admin.Applications.Approve, params); }
export async function rejectGroupApplication(params: RejectGroupApplicationRequest) { return request<void>(Endpoints.Groups.Admin.Applications.Reject, params); }

export async function createGroupAnnouncement(params: CreateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Admin.Announcements.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateGroupAnnouncement(params: UpdateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Admin.Announcements.Update, params); }
export async function deleteGroupAnnouncement(params: DeleteGroupAnnouncementRequest) { return request<void>(Endpoints.Groups.Admin.Announcements.Delete, params); }

// Groups admin（守卫子树新增端点，spec: api_spec_group.md）
export async function getGroupSettings(groupId: string) {
  return request<GroupSettings>(Endpoints.Groups.Admin.GetSettings, { group_id: groupId });
}
export async function listAdminGroupMembers(params: ListAdminGroupMembersRequest) {
  return request<PaginatedData<AdminGroupMemberItem>>(Endpoints.Groups.Admin.Members.List, params);
}
// 群主/群管理员撤回任意成员消息（无时间窗）；撤回自己的消息走 recallGroupMessage
export async function recallGroupMessageAsAdmin(params: RecallGroupMessageAdminRequest) {
  return request<void>(Endpoints.Groups.Admin.Messages.Recall, params);
}

// Users extended
export async function listUserMoments(userId: string, pagination?: PaginationParams) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListMoments, Endpoints.Public.Users.ListMoments), { user_id: userId, pagination });
}
export async function listUserReactedMoments(userId: string, pagination?: PaginationParams) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListReactedMoments, Endpoints.Public.Users.ListReactedMoments), { user_id: userId, pagination });
}

// Comments
export async function listComments(momentId: string, pagination?: PaginationParams) {
  return request<PaginatedData<CommentItem>>(pick(Endpoints.Comments.List, Endpoints.Public.Comments.List), { moment_id: momentId, pagination });
}
// spec: api_spec_comment.md → comments/fetch_replies（子评论列表，时间正序）
export async function listCommentReplies(commentId: string, pagination?: PaginationParams) {
  return request<PaginatedData<CommentItem>>(pick(Endpoints.Comments.ListReplies, Endpoints.Public.Comments.ListReplies), { comment_id: commentId, pagination });
}

// Map
export async function getMapCanvas(params: MapCanvasRequest) { return request<MapCanvasResponse>(Endpoints.Map.Canvas, params); }

// Search
// spec api_spec_discovery.md：prefix 必填，reference_point/types/radius 必填可空
export async function searchSuggest(params: SearchSuggestRequest) { return request<SearchSuggestResponse>(pick(Endpoints.Search.Suggest, Endpoints.Public.Search.Suggest), params); }

export { Endpoints };

// netError —— 网络层失败（fetch 抛错/超时）时的统一兜底响应：
// code=-1 与业务错误码空间区分，data 传调用方的空态回退（null / { items: [] }）。
// 收敛散落各页的 `.catch(() => netError(…))` 样板。
export function netError<T>(fallback: T): ApiResponse<T> {
	return { code: -1, message: '网络异常', data: fallback };
}

// ── 平台管理（spec: api_spec_admin.md，/admin/api/v1；仅平台管理员）──

// RBAC
export async function adminRbacMe()                      { return request<AdminPermissions>(Endpoints.Admin.Rbac.GetMe); }
export async function adminListPermissions()             { return request<AdminPermissionDictionary>(Endpoints.Admin.Rbac.PermissionsFetch); }
export async function adminListRoles()                   { return request<AdminRoleList>(Endpoints.Admin.Rbac.RolesFetch); }
export async function adminCreateRole(params: { name: string; description: string | null })          { return request<AdminRoleItem>(Endpoints.Admin.Rbac.RolesCreate, params); }
export async function adminUpdateRole(params: { role_id: string; name: string | null; description: string | null; clear_description: boolean | null }) { return request<void>(Endpoints.Admin.Rbac.RolesUpdate, params); }
export async function adminDeleteRole(params: { role_id: string })                                   { return request<void>(Endpoints.Admin.Rbac.RolesDelete, params); }
export async function adminSetRolePermissions(params: { role_id: string; permission_keys: string[] }) { return request<void>(Endpoints.Admin.Rbac.RolesSetPermissions, params); }
export async function adminListAdmins(params?: { pagination?: PaginationParams })                    { return request<PaginatedData<AdminOperatorItem>>(Endpoints.Admin.Rbac.AdminsFetch, params ?? {}); }
export async function adminAwardAdmin(params: { user_id: string; award_all_permissions: boolean | null; role_ids: string[] }) { return request<void>(Endpoints.Admin.Rbac.AdminsAward, params); }
export async function adminRevokeAdmin(params: { user_id: string })                                  { return request<void>(Endpoints.Admin.Rbac.AdminsRevoke, params); }

// 审计日志
export async function adminListAuditLogs(params?: {
	action?: AuditAction | null; operator_id?: string | null; target_type?: AuditTargetType | null;
	target_id?: string | null; start_time?: string | null; end_time?: string | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AuditLogItem>>(Endpoints.Admin.Audit.Fetch, params ?? {}); }

// 地理统计（区域排名 + 热门地点 + 全站存量 + 每日新增趋势）
export async function adminStatsGeo(params?: {
	level?: number | null;
	time_range?: StatsTimeRange | null;
	limit?: number | null;
	trend_days?: number | null;
}) { return request<AdminGeoStatsResponse>(Endpoints.Admin.Stats.Geo, params ?? {}); }

// 区域下钻（网格桶内实体明细，keyset 分页）
export async function adminStatsGeoDrilldown(params: {
	cell_token: string;
	level?: number | null;
	entity_type: 'user' | 'group' | 'moment';
	time_range?: StatsTimeRange | null;
	pagination?: { limit?: number; cursor?: string };
}) { return request<AdminGeoDrilldownPage>(Endpoints.Admin.Stats.Drilldown, params); }

// 应用配置
export async function adminGetAppConfig()                { return request<AppConfig>(Endpoints.Admin.AppConfig.Get); }
export async function adminUpdateAppConfig(params: UpdateAppConfigRequest) { return request<void>(Endpoints.Admin.AppConfig.Update, params); }

// 用户管理
export async function adminSearchUsers(params?: {
	user_id?: string | null; nickname?: string | null; login_id?: string | null;
	status?: UserAccountStatus | null; user_type?: UserType | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AdminUserItem>>(Endpoints.Admin.Users.Search, params ?? {}); }
export async function adminOpenUser(params: { user_id: string })              { return request<AdminUserItem>(Endpoints.Admin.Users.OpenTheDoor, params); }
export async function adminSetUserBan(params: { user_id: string; active: boolean; reason: string | null; duration_days: number | null }) { return request<void>(Endpoints.Admin.Users.SetBan, params); }

// 群组管理
export async function adminSearchGroups(params?: {
	group_id?: string | null; name?: string | null; owner_id?: string | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AdminGroupItem>>(Endpoints.Admin.Groups.Search, params ?? {}); }
export async function adminOpenGroup(params: { group_id: string })            { return request<AdminGroupItem>(Endpoints.Admin.Groups.OpenTheDoor, params); }
export async function adminDismissGroup(params: { group_id: string; reason: string | null }) { return request<void>(Endpoints.Admin.Groups.Dismiss, params); }

// 举报审核
export async function adminListReports(params?: {
	status?: ReportStatus | null; target_type?: ReportTargetType | null; pagination?: PaginationParams;
}) { return request<PaginatedData<AdminReportItem>>(Endpoints.Admin.Reports.Fetch, params ?? {}); }
export async function adminJudgeReport(params: { report_id: string; action: ReportAction; note: string | null; ban_duration_days: number | null }) { return request<void>(Endpoints.Admin.Reports.Judge, params); }

// 标签管理
export async function adminSearchTags(params?: { keyword?: string | null; pagination?: PaginationParams }) { return request<PaginatedData<AdminTagItem>>(Endpoints.Admin.Tags.Search, params ?? {}); }
export async function adminSetTagBan(params: { name: string; active: boolean; hard_delete: boolean | null }) { return request<void>(Endpoints.Admin.Tags.SetBan, params); }

// 系统公告（create 为契约幂等端点：必须携带 Idempotency-Key）
export async function adminListAnnouncements(params?: { severity?: AnnounceSeverity | null; pagination?: PaginationParams }) { return request<PaginatedData<AdminAnnouncementItem>>(Endpoints.Admin.Announcements.Fetch, params ?? {}); }
export async function adminCreateAnnouncement(params: { title: string; content: string; severity: AnnounceSeverity; action_url: string | null; send_push: boolean | null }) {
	return request<AdminAnnouncementItem>(Endpoints.Admin.Announcements.Create, params, { 'Idempotency-Key': newIdempotencyKey() });
}
export async function adminUpdateAnnouncement(params: { announcement_id: string; title: string | null; content: string | null; severity: AnnounceSeverity | null; action_url: string | null }) { return request<AdminAnnouncementItem>(Endpoints.Admin.Announcements.Update, params); }
export async function adminDeleteAnnouncement(params: { announcement_id: string }) { return request<void>(Endpoints.Admin.Announcements.Delete, params); }

// 分享码落地（spec api_spec_public.md share/open_the_door；仅 public 版，匿名可调）
export async function openShareDoor(params: { share_code: string }) { return request<ShareDetail>(Endpoints.Share.Get, params); }
// 分享码创建（spec api_spec_share.md share/create；契约幂等端点：必须携带 Idempotency-Key）
export async function createShare(params: CreateShareRequest) { return request<ShareCreated>(Endpoints.Share.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
