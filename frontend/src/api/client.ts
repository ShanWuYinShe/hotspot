// API client — thin fetch wrapper for JSON POST calls.
// Cookie-based auth (HttpOnly) — no Bearer token needed.

import { Endpoints } from '@/api/endpoints';
import type {
	AccountData,
	ApiVersionMeta,
	AuthStatus,
	AuthTokenPair,
	BootstrapData,
	ClientConfig,
	CommentItem,
	RelatedUserItem,
	GroupAnnouncementItem,
	GroupItem,
	GroupMemberItem,
	GroupMessageItem,
	JoinGroupResult,
	GroupApplicationItem,
	GroupInviteItem,
	GroupSentInviteItem,
	BannedMemberItem,
	Invitation,
	InviteAcceptedResult,
	SearchGroupsRequest,
	UpdateGroupRequest,
	DismissGroupRequest,
	TransferOwnershipRequest,
	SearchGroupMembersRequest,
	InviteMemberRequest,
	GroupSentInvitesRequest,
	RevokeSentInviteRequest,
	SetMemberRoleRequest,
	SetMuteRequest,
	SetMemberBanRequest,
	BannedMembersRequest,
	RemoveMemberRequest,
	GroupApplicationsRequest,
	ApproveGroupApplicationRequest,
	RejectGroupApplicationRequest,
	CreateGroupAnnouncementRequest,
	UpdateGroupAnnouncementRequest,
	DeleteGroupAnnouncementRequest,
	MyApplicationsRequest,
	CancelApplicationRequest,
	MyInvitesRequest,
	AcceptInviteRequest,
	RejectInviteRequest,
	PresignedUpload,
	PresignUploadRequest,
	BatchPresignedUpload,
	BatchCompleteUpload,
	BatchPresignUploadRequest,
	BatchCompleteUploadRequest,
	UploadedMedia,
	MessageSent,
	MomentItem,
	FavoritedMomentItem,
	NotifyItem,
	PaginatedData,
	PrivateChatItem,
	PrivateChatOpenResult,
	PrivateMessageItem,
	ReportCreated,
	TrendingBundle,
	UnreadCountResult,
	UserItem,
	UserProfileBundle,
	UserSettings,
	CreateGroupRequest,
	MapCanvasRequest,
	MapCanvasResponse,
	SearchSuggestRequest,
	SearchSuggestResponse,
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

export function setApiBaseUrl(url: string) {
  baseUrl = url.replace(/\/$/, '');
}

/**
 * 生成幂等键（spec: api_spec_base.md「幂等性（契约级）」）。
 * 仅用于 spec 明确要求 Idempotency-Key 的写操作（auth/register、auth/guest、
 * groups/create）；优先 crypto.randomUUID，非安全上下文回退时间戳+随机数。
 */
export function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
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
export async function register(params: object)           { return request<AuthTokenPair>(Endpoints.Auth.Register, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function login(params: object)              { return request<AuthTokenPair>(Endpoints.Auth.Login, params); }
export async function guest(params?: object)             { return request<AuthTokenPair>(Endpoints.Auth.Guest, params ?? {}, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function checkAuthStatus()                   { return request<AuthStatus>(Endpoints.Auth.CheckStatus); }
export async function logout()                            { return request<void>(Endpoints.Auth.Logout); }
export async function requestPasswordReset(params: object) { return request<void>(Endpoints.Auth.RequestPasswordReset, params); }
export async function resetPassword(params: object)      { return request<AuthTokenPair>(Endpoints.Auth.ResetPassword, params); }

// Users
export async function getAccount()                        { return request<AccountData>(Endpoints.Users.GetAccount); }
export async function updateAccount(params: object)       { return request<void>(Endpoints.Users.UpdateAccount, params); }
export async function getSettings()                       { return request<UserSettings>(Endpoints.Users.GetSettings); }
// spec: api_spec_user.md → users/update_settings（request 为 { settings: Partial<UserSettings> }）
export async function updateSettings(params: { settings: Partial<UserSettings> }) {
	return request<void>(Endpoints.Users.UpdateSettings, params);
}
export async function follow(params: object)              { return request<void>(Endpoints.Users.Follow, params); }
export async function block(params: object)               { return request<void>(Endpoints.Users.Block, params); }
export async function deleteAccount(params: object)       { return request<void>(Endpoints.Users.DeleteAccount, params); }
// spec: api_spec_user.md → users/open_the_door（include_recent_moments=true 时返回主页首屏聚合）
export async function openTheDoor(userId: string, includeRecentMoments = false) {
	return request<UserProfileBundle>(Endpoints.Users.GetDetail, { user_id: userId, include_recent_moments: includeRecentMoments });
}
// spec: api_spec_user.md → users/set_remark（需已关注对方；remark 传空串清除备注）
export async function setRemark(params: { user_id: string; remark: string }) { return request<void>(Endpoints.Users.SetRemark, params); }
export async function listFriends(params?: object)        { return request<PaginatedData<RelatedUserItem>>(Endpoints.Users.ListFriends, params ?? {}); }
export async function listFollowing(params?: object)      { return request<PaginatedData<RelatedUserItem>>(Endpoints.Users.ListFollowing, params ?? {}); }
export async function listFollowers(params?: object)      { return request<PaginatedData<RelatedUserItem>>(Endpoints.Users.ListFollowers, params ?? {}); }

// Feed
export async function getFeedNearby(params?: object)      { return request<PaginatedData<MomentItem>>(Endpoints.Feed.Nearby, params ?? {}); }
export async function getNearbyUsers(params?: object)     { return request<PaginatedData<UserItem>>(Endpoints.Feed.NearbyUsers, params ?? {}); }
export async function getNearbyGroups(params?: object)    { return request<PaginatedData<GroupItem>>(Endpoints.Feed.NearbyGroups, params ?? {}); }
export async function getTrending(params?: object)        { return request<TrendingBundle>(Endpoints.Feed.Trending, params ?? {}); }

// Media（spec: api_spec_media.md）—— presign → PUT 直传 → complete 三步流程的 API 包装
export async function presignUpload(params: object) {
	return request<PresignedUpload>(Endpoints.Media.PresignUpload, params);
}
export async function completeUpload(params: object) {
	return request<UploadedMedia>(Endpoints.Media.CompleteUpload, params);
}
// spec: api_spec_media.md → 媒体批量预签名/完成（瞬间多图等，最多 9 个，原子操作）
export async function batchPresignUpload(params: BatchPresignUploadRequest) {
	return request<BatchPresignedUpload>(Endpoints.Media.BatchPresignUpload, params);
}
export async function batchCompleteUpload(params: BatchCompleteUploadRequest) {
	return request<BatchCompleteUpload>(Endpoints.Media.BatchCompleteUpload, params);
}

// Moments
export async function searchMoments(params: object)       { return request<PaginatedData<MomentItem>>(Endpoints.Moments.Search, params); }
export async function getMomentDetail(momentId: string)   { return request<MomentItem>(Endpoints.Moments.GetDetail, { moment_id: momentId }); }
export async function createMoment(params: object)        { return request<MomentItem>(Endpoints.Moments.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateMoment(params: object)        { return request<MomentItem>(Endpoints.Moments.Update, params); }
export async function deleteMoment(params: { moment_id: string }) { return request<void>(Endpoints.Moments.Delete, params); }
export async function listMyMoments(params?: object)      { return request<PaginatedData<MomentItem>>(Endpoints.Moments.ListMy, params ?? {}); }
export async function getFavoriteMoments(params?: object)   { return request<PaginatedData<FavoritedMomentItem>>(Endpoints.Moments.ListFavorites, params ?? {}); }

// Groups
export async function getGroupDetail(groupId: string)     { return request<GroupItem>(Endpoints.Groups.GetDetail, { group_id: groupId }); }
export async function createGroup(params: CreateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function listMyGroups(params?: object)       { return request<PaginatedData<GroupItem>>(Endpoints.Groups.ListMy, params ?? {}); }
// spec: api_spec_group.md → groups/join（direct 直接加入 / approval 创建申请）、groups/leave
export async function joinGroup(groupId: string, extra?: { password?: string; message?: string }) {
	return request<JoinGroupResult>(Endpoints.Groups.Join, { group_id: groupId, ...extra });
}
export async function leaveGroup(groupId: string)          { return request<void>(Endpoints.Groups.Leave, { group_id: groupId }); }

// Search
export async function searchUsers(params: object)         { return request<PaginatedData<UserItem>>(Endpoints.Users.Search, params); }

// Notifications
export async function getUnreadCount()                    { return request<UnreadCountResult>(Endpoints.Notify.GetUnreadCount); }
export async function listNotifications(params?: object)  { return request<PaginatedData<NotifyItem>>(Endpoints.Notify.List, params ?? {}); }
// spec: notify.md → mark_read：notify_ids 指定 1-50 条；不传或空数组 = 全部已读。
export async function markRead(params: object)            { return request<void>(Endpoints.Notify.MarkRead, params); }

// System
export async function getClientConfig()                   { return request<ClientConfig>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: object)          { return request<BootstrapData>(Endpoints.System.Bootstrap, params ?? {}); }

// Reports
export async function createReport(params: object)        { return request<ReportCreated>(Endpoints.Reports.Create, params); }

// Chat
export async function getPrivateChats(params?: object)    { return request<PaginatedData<PrivateChatItem>>(Endpoints.PrivateChats.List, params ?? {}); }
export async function getPrivateChatDetail(params: { target_user_id: string }) { return request<PrivateChatOpenResult>(Endpoints.PrivateChats.GetDetail, params); }
export async function getPrivateMessages(params: object)  { return request<PaginatedData<PrivateMessageItem>>(Endpoints.PrivateMessages.GetHistory, params); }
export async function sendPrivateMessage(params: object)  { return request<MessageSent>(Endpoints.PrivateMessages.Send, params); }
export async function getGroupMessages(params: object)    { return request<PaginatedData<GroupMessageItem>>(Endpoints.GroupMessages.GetHistory, params); }
export async function sendGroupMessage(params: object)    { return request<MessageSent>(Endpoints.GroupMessages.Send, params); }
// spec: api_spec_chat.md → 消息撤回/编辑（recall 受窗口限制/群管理员豁免，update 仅 type=text）
export async function recallPrivateMessage(params: { message_id: string }) { return request<void>(Endpoints.PrivateMessages.Recall, params); }
export async function updatePrivateMessage(params: { message_id: string; content: string }) { return request<PrivateMessageItem>(Endpoints.PrivateMessages.Update, params); }
export async function recallGroupMessage(params: { message_id: string }) { return request<void>(Endpoints.GroupMessages.Recall, params); }
export async function updateGroupMessage(params: { message_id: string; content: string }) { return request<GroupMessageItem>(Endpoints.GroupMessages.Update, params); }
export async function markPrivateRead(params: object)     { return request<void>(Endpoints.PrivateMessages.MarkRead, params); }
export async function markGroupRead(params: object)       { return request<void>(Endpoints.GroupMessages.MarkRead, params); }
// spec: api_spec_chat.md → private_chats/update_preferences（静音/置顶）
export async function updatePrivateChatPreferences(params: object) { return request<void>(Endpoints.PrivateChats.UpdatePreferences, params); }
// spec: api_spec_group.md → groups/update_preferences（静音/置顶）
export async function updateGroupPreferences(params: object) { return request<void>(Endpoints.Groups.UpdatePreferences, params); }

// Reactions
export async function setReaction(params: object)         { return request<void>(Endpoints.Reactions.Set, params); }

// Comments
export async function createComment(params: object)       { return request<CommentItem>(Endpoints.Comments.Create, params); }

// Moments — favorite (two-state via active per spec)
export async function favoriteMoment(params: object)      { return request<void>(Endpoints.Moments.Favorite, params); }

// Groups extended
export async function getGroupMembers(groupId: string, pagination?: object) {
  return request<PaginatedData<GroupMemberItem>>(Endpoints.Groups.Members.List, { group_id: groupId, pagination });
}
export async function getGroupAnnouncements(groupId: string) {
  return request<{ items: GroupAnnouncementItem[] }>(Endpoints.Groups.Announcements.List, { group_id: groupId });
}

// Groups admin（spec: api_spec_group.md → 群组管理：搜索/更新/解散/转让/申请/邀请/成员/公告）
export async function searchGroups(params: SearchGroupsRequest) { return request<PaginatedData<GroupItem>>(Endpoints.Groups.Search, params); }
export async function updateGroup(params: UpdateGroupRequest) { return request<GroupItem>(Endpoints.Groups.Update, params); }
export async function dismissGroup(params: DismissGroupRequest) { return request<void>(Endpoints.Groups.Dismiss, params); }
export async function transferGroupOwnership(params: TransferOwnershipRequest) { return request<void>(Endpoints.Groups.TransferOwnership, params); }
export async function listMyApplications(params?: MyApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.ListApplications, params ?? {}); }
export async function cancelGroupApplication(params: CancelApplicationRequest) { return request<void>(Endpoints.Groups.CancelApplication, params); }
export async function listMyInvites(params?: MyInvitesRequest) { return request<PaginatedData<GroupInviteItem>>(Endpoints.Groups.ListInvites, params ?? {}); }
export async function acceptGroupInvite(params: AcceptInviteRequest) { return request<InviteAcceptedResult>(Endpoints.Groups.AcceptInvite, params); }
export async function rejectGroupInvite(params: RejectInviteRequest) { return request<void>(Endpoints.Groups.RejectInvite, params); }

export async function searchGroupMembers(params: SearchGroupMembersRequest) { return request<PaginatedData<GroupMemberItem>>(Endpoints.Groups.Members.Search, params); }
export async function inviteGroupMember(params: InviteMemberRequest) { return request<Invitation>(Endpoints.Groups.Members.Invite, params); }
export async function setGroupMemberRole(params: SetMemberRoleRequest) { return request<void>(Endpoints.Groups.Members.SetRole, params); }
export async function setGroupMemberMute(params: SetMuteRequest) { return request<void>(Endpoints.Groups.Members.SetMute, params); }
export async function banGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Members.Ban, { ...params, active: true }); }
export async function unbanGroupMember(params: Omit<SetMemberBanRequest, 'active'>) { return request<void>(Endpoints.Groups.Members.Unban, { ...params, active: false }); }
export async function listBannedGroupMembers(params: BannedMembersRequest) { return request<PaginatedData<BannedMemberItem>>(Endpoints.Groups.Members.ListBanned, params); }
export async function removeGroupMember(params: RemoveMemberRequest) { return request<void>(Endpoints.Groups.Members.Remove, params); }
export async function listSentGroupInvites(params: GroupSentInvitesRequest) { return request<PaginatedData<GroupSentInviteItem>>(Endpoints.Groups.Members.Invites.List, params); }
export async function revokeSentGroupInvite(params: RevokeSentInviteRequest) { return request<void>(Endpoints.Groups.Members.Invites.Revoke, params); }

export async function listGroupApplications(params: GroupApplicationsRequest) { return request<PaginatedData<GroupApplicationItem>>(Endpoints.Groups.Applications.List, params); }
export async function approveGroupApplication(params: ApproveGroupApplicationRequest) { return request<void>(Endpoints.Groups.Applications.Approve, params); }
export async function rejectGroupApplication(params: RejectGroupApplicationRequest) { return request<void>(Endpoints.Groups.Applications.Reject, params); }

export async function createGroupAnnouncement(params: CreateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Announcements.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function updateGroupAnnouncement(params: UpdateGroupAnnouncementRequest) { return request<GroupAnnouncementItem>(Endpoints.Groups.Announcements.Update, params); }
export async function deleteGroupAnnouncement(params: DeleteGroupAnnouncementRequest) { return request<void>(Endpoints.Groups.Announcements.Delete, params); }

// Users extended
export async function listUserMoments(userId: string, pagination?: object) {
  return request<PaginatedData<MomentItem>>(Endpoints.Users.ListMoments, { user_id: userId, pagination });
}
export async function listUserReactedMoments(userId: string, pagination?: object) {
  return request<PaginatedData<MomentItem>>(Endpoints.Users.ListReactedMoments, { user_id: userId, pagination });
}

// Comments
export async function listComments(momentId: string, pagination?: object) {
  return request<PaginatedData<CommentItem>>(Endpoints.Comments.List, { moment_id: momentId, pagination });
}

// Map
export async function getMapCanvas(params: MapCanvasRequest) { return request<MapCanvasResponse>(Endpoints.Map.Canvas, params); }

// Search
export async function searchSuggest(params?: SearchSuggestRequest) { return request<SearchSuggestResponse>(Endpoints.Search.Suggest, params ?? {}); }

export { Endpoints };
