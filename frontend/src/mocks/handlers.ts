import { http, HttpResponse, type JsonBodyType } from "msw";
import {
  apiOk, paginated,
  mockAuthData, mockUsers, mockAccount, mockSettings,
  mockMoments, mockGroups, mockNotifications, mockChats, mockTrending,
} from "./data";

const API = "/api/v1";

function json(data: JsonBodyType) {
  return HttpResponse.json(data);
}

export const handlers = [
  // ── Auth ───────────────────────────────────────
  http.post(`${API}/auth/login`, () => json(apiOk(mockAuthData))),
  http.post(`${API}/auth/register`, () => json(apiOk(mockAuthData))),
  http.post(`${API}/auth/guest`, () => json(apiOk(mockAuthData))),
  http.post(`${API}/auth/check_status`, () => json(apiOk({ user_id: mockAccount.user_id, user_type: mockAccount.user_type }))),
  http.post(`${API}/auth/logout`, () => json(apiOk(null))),
  http.post(`${API}/auth/refresh_token`, () => json(apiOk(mockAuthData))),
  http.post(`${API}/auth/change_password`, () => json(apiOk(null))),
  http.post(`${API}/auth/list_sessions`, () => json(apiOk({ items: [] }))),

  // ── Users ──────────────────────────────────────
  http.post(`${API}/users/get_account`, () => json(apiOk(mockAccount))),
  http.post(`${API}/users/update_account`, () => json(apiOk(null))),
  http.post(`${API}/users/get_settings`, () => json(apiOk(mockSettings))),
  http.post(`${API}/users/update_settings`, () => json(apiOk(null))),
  http.post(`${API}/users/update_location`, () => json(apiOk(null))),
  http.post(`${API}/users/get_profile`, () => json(apiOk({
    user: mockUsers[1], stats: { moments_count: 24, friends_count: 18, followers_count: 89, following_count: 42, groups_count: 5 },
    relation: { is_following: true, is_followed_by: false, is_blocked: false },
  }))),
  http.post(`${API}/users/get_detail`, () => json(apiOk(mockUsers[1]))),
  http.post(`${API}/users/search`, () => json(paginated(mockUsers))),
  http.post(`${API}/users/batch_get`, () => json(apiOk({ users: Object.fromEntries(mockUsers.map(u => [u.user_id, u])) }))),
  http.post(`${API}/users/list_friends`, () => json(paginated(mockUsers.slice(0, 6)))),
  http.post(`${API}/users/list_following`, () => json(paginated(mockUsers.slice(1, 6)))),
  http.post(`${API}/users/list_followers`, () => json(paginated(mockUsers.slice(2, 7)))),
  http.post(`${API}/users/list_groups`, () => json(paginated(mockGroups))),
  http.post(`${API}/users/list_moments`, () => json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/users/list_reacted_moments`, () => json(paginated(mockMoments.slice(6, 18)))),
  http.post(`${API}/users/follow`, () => json(apiOk(null))),
  http.post(`${API}/users/block`, () => json(apiOk(null))),
  http.post(`${API}/users/list_blocked`, () => json(paginated([]))),

  // ── Moments ────────────────────────────────────
  http.post(`${API}/moments/search`, () => json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/moments/get_detail`, () => json(apiOk(mockMoments[0]))),
  http.post(`${API}/moments/list_my`, () => json(paginated(mockMoments.slice(0, 9)))),
  http.post(`${API}/moments/list_favorites`, () => json(paginated(mockMoments.slice(0, 6)))),
  http.post(`${API}/moments/favorite`, () => json(apiOk(null))),
  http.post(`${API}/moments/create`, () => json(apiOk(mockMoments[0]))),

  // ── Comments ───────────────────────────────────
  http.post(`${API}/comments/list`, () => json(paginated([
    { comment_id: "c1", user: mockUsers[1], content: "拍得真好看！", created_at: "2026-07-26T14:00:00Z", likes_count: 5 },
    { comment_id: "c2", user: mockUsers[2], content: "什么相机拍的？", created_at: "2026-07-26T13:00:00Z", likes_count: 2 },
  ]))),
  http.post(`${API}/comments/list_replies`, () => json(paginated([]))),
  http.post(`${API}/comments/get_detail`, () => json(apiOk({ comment_id: "c1", user: mockUsers[1], content: "拍得真好看！", created_at: "2026-07-26T14:00:00Z" }))),

  // ── Reactions ──────────────────────────────────
  http.post(`${API}/reactions/list`, () => json(paginated([
    { user: mockUsers[1], emoji: "❤️", reacted_at: "2026-07-26T14:00:00Z" },
    { user: mockUsers[2], emoji: "🔥", reacted_at: "2026-07-26T13:30:00Z" },
    { user: mockUsers[3], emoji: "❤️", reacted_at: "2026-07-26T13:00:00Z" },
  ]))),
  http.post(`${API}/reactions/set`, () => json(apiOk(null))),

  // ── Groups ─────────────────────────────────────
  http.post(`${API}/groups/search`, () => json(paginated(mockGroups))),
  http.post(`${API}/groups/get_detail`, () => json(apiOk({ group: mockGroups[0], my_state: { role: "member", is_joined: true } }))),
  http.post(`${API}/groups/list_my`, () => json(paginated(mockGroups.slice(0, 2)))),
  http.post(`${API}/groups/members/list`, () => json(paginated(mockUsers.slice(0, 4).map(u => ({ user: u, role: "member", joined_at: "2026-01-01T00:00:00Z" }))))),
  http.post(`${API}/groups/announcements/list`, () => json(apiOk({ items: [
    { announcement_id: "ann1", title: "本周六弹唱活动", content: "下午3点户部山，带上你的乐器", created_at: "2026-07-25T10:00:00Z" },
  ]}))),

  // ── Chat ───────────────────────────────────────
  http.post(`${API}/private_chats/list`, () => json(paginated(mockChats))),
  http.post(`${API}/private_messages/get_history`, () => json(paginated([]))),
  http.post(`${API}/private_messages/mark_read`, () => json(apiOk(null))),
  http.post(`${API}/group_messages/get_history`, () => json(paginated([]))),
  http.post(`${API}/group_messages/mark_read`, () => json(apiOk(null))),

  // ── Feed ───────────────────────────────────────
  http.post(`${API}/feed/list_moments`, () => json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/feed/list_groups`, () => json(paginated(mockGroups))),
  http.post(`${API}/feed/list_users`, () => json(paginated(mockUsers.slice(0, 5)))),
  http.post(`${API}/feed/list_trending`, () => json(apiOk(mockTrending))),
  http.post(`${API}/feed/dismiss`, () => json(apiOk(null))),

  // ── Search ─────────────────────────────────────
  http.post(`${API}/search/all`, () => json(apiOk({ users: mockUsers, moments: mockMoments.slice(0, 8) }))),
  http.post(`${API}/search/suggest`, () => json(apiOk({ suggestions: mockTrending.hot_tags }))),

  // ── System ─────────────────────────────────────
  http.post(`${API}/system/health`, () => json(apiOk({ status: "healthy", timestamp: new Date().toISOString() }))),
  http.post(`${API}/system/get_client_config`, () => json(apiOk({
    features: { enable_public_registration: true, enable_guest_login: true, enable_moments: true, enable_nearby_people: true },
    limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200 },
    contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
    defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
  }))),
  http.post(`${API}/system/bootstrap`, () => json(apiOk({
    bootstrap_version: "v1",
    meta: { version: "v1", status: "active" },
    client_config: {
      features: { enable_public_registration: true, enable_guest_login: true, enable_moments: true, enable_nearby_people: true },
      limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9 },
      contact: { support_email: "" },
      defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" } },
    },
    profile: mockAccount,
    unread_summary: { notification_unread_count: 3, private_chat_unread_count: 2, group_chat_unread_count: 1, total_badge: 6 },
  }))),

  // ── Notify ─────────────────────────────────────
  http.post(`${API}/notify/list`, () => json(paginated(mockNotifications))),
  http.post(`${API}/notify/get_unread_count`, () => json(apiOk({ total: 3, by_type: { social: 1, interaction: 1, group: 1, report: 0, system: 0 } }))),
  http.post(`${API}/notify/mark_read`, () => json(apiOk(null))),

  // ── Map ────────────────────────────────────────
  http.post(`${API}/map/canvas`, () => json(apiOk({ heatmap: [], pins: [] }))),

  // ── Reports ────────────────────────────────────
  http.post(`${API}/reports/create`, () => json(apiOk({ report_id: "report_1" }))),
  http.post(`${API}/reports/list_my`, () => json(paginated([]))),

  // ── Share ──────────────────────────────────────
  http.post(`${API}/share/create`, () => json(apiOk({ share_code: "abc123" }))),
  http.post(`${API}/share/get`, () => json(apiOk({ share_code: "abc123", target_type: "moment", target_id: "moment_1" }))),
  http.post(`${API}/share/list_my`, () => json(paginated([]))),
  http.post(`${API}/share/revoke`, () => json(apiOk(null))),

  // ── Media ──────────────────────────────────────
  http.post(`${API}/media/presign_upload`, () => json(apiOk({ upload_id: "upload_1", upload_url: "/mock-upload", headers: {} }))),
  http.post(`${API}/media/complete_upload`, () => json(apiOk({ file_id: "file_1", url: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400" }))),
];
