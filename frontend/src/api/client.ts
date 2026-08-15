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
	PaginatedData,
	PrivateChatItem,
	PrivateChatOpenResult,
	PrivateMessageItem,
	ReportCreated,
	TrendingBundle,
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
	RequestPasswordResetRequest,
	ResetPasswordRequest,
	UpdateAccountRequest,
	SetFollowRequest,
	SetBlockRequest,
	SearchUsersRequest,
	NearbyFeedRequest,
	TrendingRequest,
	CreateMomentRequest,
	UpdateMomentRequest,
	SearchMomentsRequest,
	FavoriteMomentRequest,
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
	CreateReportRequest,
	MarkNotifyReadRequest,
	CompleteUploadRequest,
	BootstrapRequest,
	PaginationParams,
} from '@/api/types';

// 分页信封由 types.ts 提供；此处集中重导出供页面直接使用。
export type { PaginatedData };

export interface ApiResponse<T> {
  code: number;
  message?: string;
  data: T;
}

let baseUrl = '';
let refreshLock: Promise<boolean> | null = null;
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
 * 生成幂等键（spec: api_spec_base.md「幂等性（契约级）」）。
 * 格式：{client_id}-{timestamp_ms}-{random}，长度 32-64，字符集 [a-zA-Z0-9-_]。
 * Web 端没有独立 device_id 可读时使用固定 client_id，至少保证同页面重试稳定。
 */
export function newIdempotencyKey(): string {
  const clientId = 'hotspot-web';
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

async function request<T>(
  endpoint: string,
  body?: unknown,
  extraHeaders: Record<string, string> = {},
  retryCount = 0,
  requestId = newRequestId(),
): Promise<ApiResponse<T>> {
  const url = `${baseUrl}${endpoint}`;
  const res = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', 'X-Request-Id': requestId, ...extraHeaders },
    body: body ? JSON.stringify(body) : '{}',
  });

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try { const errBody = await res.json(); msg = (errBody as any).message ?? msg; } catch {}
    // A non-200 transport failure is unexpected per spec (business errors come
    // back as 200 + code). Surface it; AuthContext treats a thrown request as
    // a soft failure the caller can handle.
    throw new Error(msg);
  }

  const json: ApiResponse<T> = await res.json();

  // 10201: account banned — force-drop and show the ban screen with the message.
  if (json.code === 10201) {
    emitAuthEvent('auth-banned', json.message);
    return json;
  }

  // 10100: access token invalid/expired. Try a single deduped refresh + retry.
  if (json.code === 10100 && retryCount < 1) {
    if (!refreshLock) {
      refreshLock = (async () => {
        try {
          const r = await fetch(`${baseUrl}${Endpoints.Auth.RefreshToken}`, {
            method: 'POST', credentials: 'include',
            headers: { 'Content-Type': 'application/json' }, body: '{}',
          });
          if (r.ok) {
            const j = await r.json();
            if (j.code === 0) return true;
          }
        } catch {}
        return false;
      })().finally(() => { refreshLock = null; });
    }
    const ok = await refreshLock;
    if (ok) return request<T>(endpoint, body, extraHeaders, retryCount + 1, requestId);
    // Refresh failed: notify AuthContext so it can drop to anonymous and prompt.
    emitAuthEvent('auth-failed');
  }

  return json;
}

// Auth
export async function register(params: RegisterRequest)           { return request<AuthTokenPair>(Endpoints.Auth.Register, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function login(params: LoginRequest)              { return request<AuthTokenPair>(Endpoints.Auth.Login, params); }
export async function guest(params?: GuestRequest)             { return request<AuthTokenPair>(Endpoints.Auth.Guest, params ?? {}, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function checkAuthStatus()                   { return request<SessionStatus>(Endpoints.Auth.Session); }
export async function logout()                            { return request<void>(Endpoints.Auth.Signout); }
export async function requestPasswordReset(params: RequestPasswordResetRequest) { return request<void>(Endpoints.Auth.RequestPasswordReset, params); }
export async function resetPassword(params: ResetPasswordRequest)      { return request<AuthTokenPair>(Endpoints.Auth.ResetPassword, params); }

// Users
export async function getAccount()                        { return request<AccountData>(Endpoints.Users.GetAccount); }
export async function updateAccount(params: UpdateAccountRequest)       { return request<void>(Endpoints.Users.UpdateAccount, params); }
export async function getSettings()                       { return request<UserSettings>(Endpoints.Users.GetSettings); }
// spec: api_spec_user.md → users/update_settings（request 为 { settings: Partial<UserSettings> }）
export async function updateSettings(params: { settings: Partial<UserSettings> }) {
	return request<void>(Endpoints.Users.UpdateSettings, params);
}
export async function follow(params: SetFollowRequest)              { return request<void>(Endpoints.Users.Follow, params); }
export async function block(params: SetBlockRequest)               { return request<void>(Endpoints.Users.Block, params); }
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

// Feed
export async function getFeedNearby(params?: NearbyFeedRequest)      { return request<PaginatedData<MomentItem>>(Endpoints.Feed.Nearby, params ?? {}); }
export async function getNearbyUsers(params?: NearbyFeedRequest)     { return request<PaginatedData<UserItem>>(pick(Endpoints.Feed.NearbyUsers, Endpoints.Public.Feed.NearbyUsers), params ?? {}); }
export async function getNearbyGroups(params?: NearbyFeedRequest)    { return request<PaginatedData<GroupItem>>(pick(Endpoints.Feed.NearbyGroups, Endpoints.Public.Feed.NearbyGroups), params ?? {}); }
export async function getTrending(params?: TrendingRequest)        { return request<TrendingBundle>(pick(Endpoints.Feed.Trending, Endpoints.Public.Feed.Trending), params ?? {}); }

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

// Moments — favorite (two-state via active per spec)
export async function favoriteMoment(params: FavoriteMomentRequest)      { return request<void>(Endpoints.Moments.Favorite, params); }

// Groups extended
export async function getGroupMembers(groupId: string, pagination?: object) {
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
export async function listUserMoments(userId: string, pagination?: object) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListMoments, Endpoints.Public.Users.ListMoments), { user_id: userId, pagination });
}
export async function listUserReactedMoments(userId: string, pagination?: object) {
  return request<PaginatedData<MomentItem>>(pick(Endpoints.Users.ListReactedMoments, Endpoints.Public.Users.ListReactedMoments), { user_id: userId, pagination });
}

// Comments
export async function listComments(momentId: string, pagination?: object) {
  return request<PaginatedData<CommentItem>>(pick(Endpoints.Comments.List, Endpoints.Public.Comments.List), { moment_id: momentId, pagination });
}

// Map
export async function getMapCanvas(params: MapCanvasRequest) { return request<MapCanvasResponse>(Endpoints.Map.Canvas, params); }

// Search
export async function searchSuggest(params?: SearchSuggestRequest) { return request<SearchSuggestResponse>(pick(Endpoints.Search.Suggest, Endpoints.Public.Search.Suggest), params ?? {}); }

export { Endpoints };
