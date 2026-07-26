// API client — thin fetch wrapper for JSON POST calls.
// Cookie-based auth (HttpOnly) — no Bearer token needed.

import { Endpoints } from './endpoints';

export interface ApiResponse<T> {
  code: number;
  message?: string;
  data: T;
}

export interface PaginatedData<T> {
  items: T[];
  next_cursor?: string;
}

let baseUrl = '';
let refreshLock: Promise<boolean> | null = null;

export function setApiBaseUrl(url: string) {
  baseUrl = url.replace(/\/$/, '');
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
    throw new Error(msg);
  }

  const json: ApiResponse<T> = await res.json();

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
  }

  return json;
}

// Auth
export async function register(params: object)           { return request<object>(Endpoints.Auth.Register, params); }
export async function login(params: object)              { return request<object>(Endpoints.Auth.Login, params); }
export async function guest(params?: object)             { return request<object>(Endpoints.Auth.Guest, params ?? {}); }
export async function checkAuthStatus()                   { return request<object>(Endpoints.Auth.CheckStatus); }
export async function logout()                            { return request<void>(Endpoints.Auth.Logout); }

// Users
export async function getAccount()                        { return request<object>(Endpoints.Users.GetAccount); }
export async function updateAccount(params: object)       { return request<void>(Endpoints.Users.UpdateAccount, params); }
export async function getSettings()                       { return request<object>(Endpoints.Users.GetSettings); }
export async function updateSettings(params: object)      { return request<void>(Endpoints.Users.UpdateSettings, params); }
export async function updateLocation(params: object)      { return request<void>(Endpoints.Users.UpdateLocation, params); }
export async function follow(params: object)              { return request<void>(Endpoints.Users.Follow, params); }
export async function block(params: object)               { return request<void>(Endpoints.Users.Block, params); }
export async function getProfile(userId: string)          { return request<object>(Endpoints.Users.GetProfile, { user_id: userId }); }
export async function getUserDetail(userId: string)       { return request<object>(Endpoints.Users.GetDetail, { user_id: userId }); }
export async function batchGet(userIds: string[])         { return request<Record<string, object>>(Endpoints.Users.BatchGet, { user_ids: userIds }); }

// Feed
export async function getFeedMoments(params?: object)     { return request<PaginatedData<object>>(Endpoints.Feed.ListMoments, params ?? {}); }
export async function getFeedTrending(params?: object)    { return request<object>(Endpoints.Feed.ListTrending, params ?? {}); }
export async function dismissFeedItem(params: object)     { return request<void>(Endpoints.Feed.Dismiss, params); }

// Moments
export async function searchMoments(params: object)       { return request<PaginatedData<object>>(Endpoints.Moments.Search, params); }
export async function getMomentDetail(momentId: string)   { return request<object>(Endpoints.Moments.GetDetail, { moment_id: momentId }); }
export async function createMoment(params: object)        { return request<object>(Endpoints.Moments.Create, params); }
export async function listMyMoments(params?: object)      { return request<PaginatedData<object>>(Endpoints.Moments.ListMy, params ?? {}); }

// Groups
export async function searchGroups(params: object)        { return request<PaginatedData<object>>(Endpoints.Groups.Search, params); }
export async function getGroupDetail(groupId: string)     { return request<object>(Endpoints.Groups.GetDetail, { group_id: groupId }); }
export async function listMyGroups(params?: object)       { return request<PaginatedData<object>>(Endpoints.Groups.ListMy, params ?? {}); }

// Search
export async function searchAll(params: object)           { return request<object>(Endpoints.Search.All, params); }
export async function searchSuggest(params: object)       { return request<object>(Endpoints.Search.Suggest, params); }

// Notifications
export async function getUnreadCount()                    { return request<object>(Endpoints.Notify.GetUnreadCount); }
export async function listNotifications(params?: object)  { return request<PaginatedData<object>>(Endpoints.Notify.List, params ?? {}); }

// System
export async function getClientConfig()                   { return request<object>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: object)          { return request<object>(Endpoints.System.Bootstrap, params ?? {}); }

// Reports
export async function createReport(params: object)        { return request<object>(Endpoints.Reports.Create, params); }

// Share
export async function getShare(params: object)            { return request<object>(Endpoints.Share.Get, params); }
export async function createShare(params: object)         { return request<object>(Endpoints.Share.Create, params); }

// Chat
export async function getPrivateChats(params?: object)    { return request<PaginatedData<object>>(Endpoints.PrivateChats.List, params ?? {}); }
export async function getPrivateMessages(params: object)  { return request<PaginatedData<object>>(Endpoints.PrivateMessages.GetHistory, params); }
export async function sendPrivateMessage(params: object)  { return request<object>(Endpoints.PrivateMessages.Send, params); }
export async function getGroupMessages(params: object)    { return request<PaginatedData<object>>(Endpoints.GroupMessages.GetHistory, params); }
export async function sendGroupMessage(params: object)    { return request<object>(Endpoints.GroupMessages.Send, params); }
export async function markPrivateRead(params: object)     { return request<void>(Endpoints.PrivateMessages.MarkRead, params); }
export async function markGroupRead(params: object)       { return request<void>(Endpoints.GroupMessages.MarkRead, params); }

// Media
export async function presignUpload(params: object)       { return request<object>(Endpoints.Media.PresignUpload, params); }
export async function completeUpload(params: object)      { return request<object>(Endpoints.Media.CompleteUpload, params); }

// Reactions
export async function setReaction(params: object)         { return request<void>(Endpoints.Reactions.Set, params); }
export async function listReactions(params: object)       { return request<PaginatedData<object>>(Endpoints.Reactions.List, params); }

// Groups extended
export async function getGroupMembers(groupId: string, pagination?: object) {
  return request<PaginatedData<object>>(Endpoints.Groups.Members.List, { group_id: groupId, pagination });
}
export async function getGroupAnnouncements(groupId: string) {
  return request<{ items: object[] }>(Endpoints.Groups.Announcements.List, { group_id: groupId });
}

// Users extended
export async function listUserMoments(userId: string, pagination?: object) {
  return request<PaginatedData<object>>(Endpoints.Users.ListMoments, { user_id: userId, pagination });
}
export async function listUserReactedMoments(userId: string, pagination?: object) {
  return request<PaginatedData<object>>(Endpoints.Users.ListReactedMoments, { user_id: userId, pagination });
}

// Comments
export async function listComments(momentId: string, pagination?: object) {
  return request<PaginatedData<object>>(Endpoints.Comments.List, { moment_id: momentId, pagination });
}

// Map
export async function getMapCanvas(params: object)        { return request<object>(Endpoints.Map.Canvas, params); }

export { Endpoints };
