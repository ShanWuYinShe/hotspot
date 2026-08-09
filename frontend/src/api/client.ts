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
	FollowerItem,
	FriendItem,
	FollowingItem,
	GroupAnnouncementItem,
	GroupItem,
	GroupMemberItem,
	GroupMessageItem,
	JoinGroupResult,
	PresignedUpload,
	UploadedMedia,
	MessageSent,
	MomentItem,
	NotifyItem,
	PaginatedData,
	PrivateChatItem,
	PrivateMessageItem,
	ReportCreated,
	TrendingBundle,
	UnreadCountResult,
	UserItem,
	UserProfileBundle,
	UserSettings,
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
 * Notify the app of an auth-state change observed at the transport layer.
 * 由 auth store 监听这些事件，API 层自身不依赖 UI 框架。
 *   'auth-failed' → session invalid / refresh failed → drop to anonymous.
 *   'auth-banned' → 10201 account banned → show ban screen.
 */
function emitAuthEvent(type: 'auth-failed' | 'auth-banned', message?: string) {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(type, { detail: { type, message } }));
}

async function request<T>(endpoint: string, body?: unknown, retryCount = 0): Promise<ApiResponse<T>> {
  const url = `${baseUrl}${endpoint}`;
  const res = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
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
    if (ok) return request<T>(endpoint, body, retryCount + 1);
    // Refresh failed: notify AuthContext so it can drop to anonymous and prompt.
    emitAuthEvent('auth-failed');
  }

  return json;
}

// Auth
export async function register(params: object)           { return request<AuthTokenPair>(Endpoints.Auth.Register, params); }
export async function login(params: object)              { return request<AuthTokenPair>(Endpoints.Auth.Login, params); }
export async function guest(params?: object)             { return request<AuthTokenPair>(Endpoints.Auth.Guest, params ?? {}); }
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
export async function getProfile(userId: string)          { return request<UserProfileBundle>(Endpoints.Users.GetProfile, { user_id: userId }); }
export async function listFriends(params?: object)        { return request<PaginatedData<FriendItem>>(Endpoints.Users.ListFriends, params ?? {}); }
export async function listFollowing(params?: object)      { return request<PaginatedData<FollowingItem>>(Endpoints.Users.ListFollowing, params ?? {}); }
export async function listFollowers(params?: object)      { return request<PaginatedData<FollowerItem>>(Endpoints.Users.ListFollowers, params ?? {}); }

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

// Moments
export async function searchMoments(params: object)       { return request<PaginatedData<MomentItem>>(Endpoints.Moments.Search, params); }
export async function getMomentDetail(momentId: string)   { return request<MomentItem>(Endpoints.Moments.GetDetail, { moment_id: momentId }); }
export async function createMoment(params: object)        { return request<MomentItem>(Endpoints.Moments.Create, params); }
export async function listMyMoments(params?: object)      { return request<PaginatedData<MomentItem>>(Endpoints.Moments.ListMy, params ?? {}); }
export async function getFavoriteMoments(params?: object)   { return request<PaginatedData<MomentItem>>(Endpoints.Moments.ListFavorites, params ?? {}); }

// Groups
export async function getGroupDetail(groupId: string)     { return request<GroupItem>(Endpoints.Groups.GetDetail, { group_id: groupId }); }
export async function createGroup(params: object)         { return request<GroupItem>(Endpoints.Groups.Create, params); }
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
// spec: scope='selected' needs exactly one of notification_ids|type; scope='all' clears all.
export async function markRead(params: object)            { return request<void>(Endpoints.Notify.MarkRead, params); }

// System
export async function getClientConfig()                   { return request<ClientConfig>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: object)          { return request<BootstrapData>(Endpoints.System.Bootstrap, params ?? {}); }

// Reports
export async function createReport(params: object)        { return request<ReportCreated>(Endpoints.Reports.Create, params); }

// Chat
export async function getPrivateChats(params?: object)    { return request<PaginatedData<PrivateChatItem>>(Endpoints.PrivateChats.List, params ?? {}); }
export async function getPrivateMessages(params: object)  { return request<PaginatedData<PrivateMessageItem>>(Endpoints.PrivateMessages.GetHistory, params); }
export async function sendPrivateMessage(params: object)  { return request<MessageSent>(Endpoints.PrivateMessages.Send, params); }
export async function getGroupMessages(params: object)    { return request<PaginatedData<GroupMessageItem>>(Endpoints.GroupMessages.GetHistory, params); }
export async function sendGroupMessage(params: object)    { return request<MessageSent>(Endpoints.GroupMessages.Send, params); }
export async function markPrivateRead(params: object)     { return request<void>(Endpoints.PrivateMessages.MarkRead, params); }
export async function markGroupRead(params: object)       { return request<void>(Endpoints.GroupMessages.MarkRead, params); }

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
export async function getMapCanvas(params: object)        { return request<object>(Endpoints.Map.Canvas, params); }

// Search
export async function searchSuggest(params?: object)      { return request<object>(Endpoints.Search.Suggest, params ?? {}); }

export { Endpoints };
