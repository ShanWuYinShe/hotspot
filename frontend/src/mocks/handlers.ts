import { http, HttpResponse, type JsonBodyType } from "msw";
import {
  apiOk, apiError, paginated,
  mockUsers, mockAccount, mockSettings,
  mockMoments, mockGroups, mockNotifications, mockChats, mockTrending, mockReactionItems,
  getMockAuthScenario, setMockLoggedIn, isMockLoggedIn,
} from "./data";

const API = "/api/v1";

function json(data: JsonBodyType) {
  return HttpResponse.json(data);
}

/**
 * Guard for auth-required endpoints: when the mock session is logged out,
 * return 10100 (spec's "not authenticated" signal) so the client exercises its
 * real auth-failure handling instead of silently succeeding. Returns null when
 * the session is valid, so handlers can early-return the guard's result.
 *
 * Optional-auth endpoints (moments/search, groups/search, search/*, map,
 * system/*, users/get_profile, feed/list_trending) are NOT guarded — anonymous
 * users see public data per spec.
 */
function requireMockAuth(): ReturnType<typeof json> | null {
  return isMockLoggedIn() ? null : json(apiError(10100, "Access Token 无效或过期"));
}

/**
 * Build the auth success response. The web client authenticates via HttpOnly
 * cookies (spec), so data is {} and tokens ride Set-Cookie. MSW sets them so
 * the cookie-based client behaves like the real backend.
 */
function authOkResponse() {
  setMockLoggedIn(true);
  // Web: AuthTokenPairWeb = {} ; tokens in HttpOnly cookies (same names as spec).
  return HttpResponse.json(apiOk({}), {
    headers: {
      "Set-Cookie": [
        "access_token=mock-jwt; Path=/; HttpOnly; SameSite=Strict",
        "refresh_token=mock-refresh; Path=/api/v1/auth/refresh_token; HttpOnly; SameSite=Strict",
      ].join(", "),
    },
  });
}

/**
 * Scenario → business error code for the LOGIN endpoint (the only auth flow
 * where 10201 "account banned" is meaningful per spec).
 */
function loginScenarioError() {
  switch (getMockAuthScenario()) {
    case "invalid-creds": return json(apiError(10100, "账号或密码错误"));
    case "banned":        return json(apiError(10201, "账号已被封禁: 违规操作"));
    case "param-error":   return json(apiError(10000, "缺少必填参数: login_id"));
    case "rate-limit":    return HttpResponse.json(apiError(10500, "请求过于频繁"), { headers: { "Retry-After": "5" } });
    case "ok":
    default:              return null;
  }
}

/**
 * Scenario → error for register/guest/reset_password. 10201 (banned) does NOT
 * apply to new/guest/reset accounts per spec, so it's excluded.
 */
function authFormScenarioError() {
  switch (getMockAuthScenario()) {
    case "invalid-creds": return json(apiError(10100, "操作失败"));
    case "param-error":   return json(apiError(10000, "缺少必填参数"));
    case "rate-limit":    return HttpResponse.json(apiError(10500, "请求过于频繁"), { headers: { "Retry-After": "5" } });
    case "ok":
    case "banned":
    default:              return null;
  }
}

export const handlers = [
  // ── Auth ───────────────────────────────────────────
  http.post(`${API}/auth/login`, () => loginScenarioError() ?? authOkResponse()),
  http.post(`${API}/auth/register`, () => authFormScenarioError() ?? authOkResponse()),
  http.post(`${API}/auth/guest`, () => authFormScenarioError() ?? authOkResponse()),
  http.post(`${API}/auth/check_status`, () => {
    // Reflect the mock login state so AuthContext's boot flow works correctly.
    if (isMockLoggedIn()) {
      return json(apiOk({ is_valid: true, expires_at: new Date(Date.now() + 3600000).toISOString() }));
    }
    return json(apiOk({ is_valid: false }));
  }),
  http.post(`${API}/auth/logout`, () => {
    setMockLoggedIn(false);
    // Spec: clear both auth cookies with Max-Age=0 (device_id is NOT cleared).
    return HttpResponse.json(apiOk(null), {
      headers: {
        "Set-Cookie": [
          "access_token=; Path=/; Max-Age=0",
          "refresh_token=; Path=/api/v1/auth/refresh_token; Max-Age=0",
        ].join(", "),
      },
    });
  }),
  http.post(`${API}/auth/refresh_token`, () => authOkResponse()),
  http.post(`${API}/auth/request_password_reset`, () => json(apiOk(null))),
  http.post(`${API}/auth/reset_password`, () => authFormScenarioError() ?? authOkResponse()),
  http.post(`${API}/auth/change_password`, () => json(apiOk(null))),
  http.post(`${API}/auth/list_sessions`, () => json(apiOk({ items: [] }))),

  // ── Users ──────────────────────────────────────
  // get_account / get_settings are auth-required → guarded. get_profile is
  // optional → not guarded (anonymous sees public profile, my_state omitted).
  http.post(`${API}/users/get_account`, () => requireMockAuth() ?? json(apiOk(mockAccount))),
  http.post(`${API}/users/update_account`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/get_settings`, () => requireMockAuth() ?? json(apiOk(mockSettings))),
  http.post(`${API}/users/update_settings`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/update_location`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/get_profile`, () => json(apiOk({
    user: mockUsers[1],
    stats: { moments_count: 24, friends_count: 18, followers_count: 89, following_count: 42, groups_count: 5 },
    relation: { is_following: true, is_followed_by: false, is_blocked: false },
    recent_moments: { items: mockMoments.slice(0, 6) },
  }))),
  http.post(`${API}/users/get_detail`, () => json(apiOk({ user: mockUsers[1] }))),
  http.post(`${API}/users/search`, () => json(paginated(mockUsers))),
  http.post(`${API}/users/batch_get`, () => json(apiOk({ users: Object.fromEntries(mockUsers.map(u => [u.user_id, u])) }))),
  // list_* default to the current user → auth-required when no user_id given.
  http.post(`${API}/users/list_friends`, () => requireMockAuth() ?? json(paginated(mockUsers.slice(0, 6)))),
  http.post(`${API}/users/list_following`, () => requireMockAuth() ?? json(paginated(mockUsers.slice(1, 6)))),
  http.post(`${API}/users/list_followers`, () => requireMockAuth() ?? json(paginated(mockUsers.slice(2, 7)))),
  http.post(`${API}/users/list_groups`, () => requireMockAuth() ?? json(paginated(mockGroups))),
  http.post(`${API}/users/list_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/users/list_reacted_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(6, 18)))),
  http.post(`${API}/users/follow`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/block`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/list_blocked`, () => requireMockAuth() ?? json(paginated([]))),

  // ── Moments ────────────────────────────────────
  // search/get_detail are auth-optional (anonymous sees public moments).
  http.post(`${API}/moments/search`, () => json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/moments/get_detail`, () => json(apiOk(mockMoments[0]))),
  http.post(`${API}/moments/list_my`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 9)))),
  http.post(`${API}/moments/list_favorites`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 6)))),
  http.post(`${API}/moments/favorite`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/moments/create`, () => requireMockAuth() ?? json(apiOk(mockMoments[0]))),

  // ── Comments ───────────────────────────────────
  // list/list_replies/get_detail are auth-optional; create requires auth.
  // Returns spec CommentItem (user: UserInfo nested; reactions aggregate).
  http.post(`${API}/comments/list`, () => json(paginated([
    { comment_id: "c1", user: mockUsers[1], content: "拍得真好看！", created_at: "2026-07-26T14:00:00Z", replies_count: 2, is_deleted: false, mentions: [], reactions: [{ emoji: "👍", count: 3, reacted_by_me: false }] },
    { comment_id: "c2", user: mockUsers[2], content: "什么相机拍的？", created_at: "2026-07-26T13:00:00Z", replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
  ]))),
  http.post(`${API}/comments/list_replies`, () => json(paginated([]))),
  http.post(`${API}/comments/get_detail`, () => json(apiOk({ comment_id: "c1", user: mockUsers[1], content: "拍得真好看！", created_at: "2026-07-26T14:00:00Z", replies_count: 2, is_deleted: false, mentions: [], reactions: [{ emoji: "👍", count: 3, reacted_by_me: false }] }))),
  http.post(`${API}/comments/create`, () => requireMockAuth() ?? json(apiOk({
    comment_id: `c${Date.now()}`, user: mockUsers[0], content: "", created_at: new Date().toISOString(),
    replies_count: 0, is_deleted: false, mentions: [], reactions: [],
  }))),
  http.post(`${API}/comments/delete`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Reactions ──────────────────────────────────
  // list is auth-optional; set requires auth. Returns spec ReactionItem.
  http.post(`${API}/reactions/list`, () => json(paginated(mockReactionItems))),
  http.post(`${API}/reactions/set`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Groups ─────────────────────────────────────
  // search/get_detail/members/announcements are auth-optional; list_my requires auth.
  http.post(`${API}/groups/search`, () => json(paginated(mockGroups))),
  http.post(`${API}/groups/get_detail`, () => json(apiOk({ ...mockGroups[0], my_state: { role: "member" as const, is_joined: true, unread_count: 0, is_muted: false, is_pinned: false } }))),
  http.post(`${API}/groups/list_my`, () => requireMockAuth() ?? json(paginated(mockGroups.slice(0, 2)))),
  http.post(`${API}/groups/members/list`, () => json(paginated(mockUsers.slice(0, 4).map(u => ({ user: u, role: "member", joined_at: "2026-01-01T00:00:00Z" }))))),
  http.post(`${API}/groups/announcements/list`, () => json(apiOk({ items: [
    { announcement_id: "ann1", group_id: "group_1", title: "本周六弹唱活动", content: "下午3点户部山，带上你的乐器", created_by: mockUsers[0], created_at: "2026-07-25T10:00:00Z", is_pinned: true },
  ]}))),
  // join/leave/create are write ops → auth-required.
  http.post(`${API}/groups/join`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/leave`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/create`, () => requireMockAuth() ?? json(apiOk(mockGroups[0]))),

  // ── Chat ───────────────────────────────────────
  // All chat endpoints are auth-required (spec: private_chats/*, *_messages/*).
  http.post(`${API}/private_chats/list`, () => requireMockAuth() ?? json(paginated(mockChats))),
  http.post(`${API}/private_messages/get_history`, () => requireMockAuth() ?? json(paginated([]))),
  // send: echo back a spec PrivateMessageItem so ChatView's optimistic append
  // is confirmed (message persists instead of rolling back).
  http.post(`${API}/private_messages/send`, () => requireMockAuth() ?? json(apiOk({
    message_id: `m${Date.now()}`, type: "text", content: "", sender: mockUsers[0],
    sent_at: new Date().toISOString(), is_read: false,
  }))),
  http.post(`${API}/private_messages/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/group_messages/get_history`, () => requireMockAuth() ?? json(paginated([]))),
  http.post(`${API}/group_messages/send`, () => requireMockAuth() ?? json(apiOk({
    message_id: `gm${Date.now()}`, type: "text", content: "", sender: mockUsers[0],
    sent_at: new Date().toISOString(), is_read: false,
  }))),
  http.post(`${API}/group_messages/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Feed ───────────────────────────────────────
  // list_moments/groups/users require auth; list_trending is optional (Home uses it anonymously).
  http.post(`${API}/feed/list_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/feed/list_groups`, () => requireMockAuth() ?? json(paginated(mockGroups))),
  http.post(`${API}/feed/list_users`, () => requireMockAuth() ?? json(paginated(mockUsers.slice(0, 5)))),
  http.post(`${API}/feed/list_trending`, () => json(apiOk(mockTrending))),
  http.post(`${API}/feed/dismiss`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Search ─────────────────────────────────────
  http.post(`${API}/search/all`, () => json(apiOk({
    users: paginated(mockUsers).data,
    groups: paginated(mockGroups).data,
    moments: paginated(mockMoments.slice(0, 8)).data,
  }))),
  http.post(`${API}/search/suggest`, () => json(apiOk({
    items: [
      { type: "tag" as const, payload: { name: "植物园", usage_count: 42 } },
      { type: "tag" as const, payload: { name: "云龙湖", usage_count: 38 } },
      { type: "tag" as const, payload: { name: "咖啡馆", usage_count: 25 } },
    ],
  }))),

  // ── System ─────────────────────────────────────
  http.post(`${API}/system/health`, () => json(apiOk({ status: "healthy", timestamp: new Date().toISOString() }))),
  http.post(`${API}/system/get_client_config`, () => json(apiOk({
    features: { enable_public_registration: true, enable_guest_login: true, enable_moments: true, enable_nearby_people: true },
    limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
    contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
    defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
  }))),
  http.post(`${API}/system/bootstrap`, () => json(apiOk({
    bootstrap_version: "v1",
    meta: { version: "v1", status: "active" },
    client_config: {
      features: { enable_public_registration: true, enable_guest_login: true, enable_moments: true, enable_nearby_people: true },
      limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
      contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
      defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
    },
    profile: mockAccount,
    unread_summary: { notification_unread_count: 3, private_chat_unread_count: 2, group_chat_unread_count: 1, total_badge: 6 },
  }))),

  // ── Notify ─────────────────────────────────────
  // All notify endpoints are auth-required.
  http.post(`${API}/notify/list`, () => requireMockAuth() ?? json(paginated(mockNotifications))),
  http.post(`${API}/notify/get_unread_count`, () => requireMockAuth() ?? json(apiOk({ total: 3, by_type: { social: 1, interaction: 1, group: 1, report: 0, system: 0 } }))),
  http.post(`${API}/notify/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Map ────────────────────────────────────────
  // map/canvas is auth-optional (anonymous sees public hotspots).
  http.post(`${API}/map/canvas`, () => json(apiOk({
    mode: "hotspots" as const,
    hotspots: {
      items: mockMoments.slice(0, 6).map(m => ({
        object_type: "moment" as const,
        object_id: m.moment_id,
        title: m.description.slice(0, 30),
        subtitle: m.location.location_name,
        cover_url: m.images?.[0]?.url,
        location: m.location,
        created_at: m.created_at,
      })),
    },
  }))),

  // ── Reports ────────────────────────────────────
  http.post(`${API}/reports/create`, () => requireMockAuth() ?? json(apiOk({ report_id: "report_1" }))),
  http.post(`${API}/reports/list_my`, () => requireMockAuth() ?? json(paginated([]))),

  // ── Share ──────────────────────────────────────
  // share/get is public (optional auth); create/list_my/revoke require auth.
  http.post(`${API}/share/create`, () => requireMockAuth() ?? json(apiOk({ share_code: "abc123" }))),
  http.post(`${API}/share/get`, () => json(apiOk({
    share: { share_id: "share_1", share_code: "abc123", share_type: "moment" as const, creator: mockAccount, created_at: "2026-07-26T12:00:00Z" },
    payload: { shared_moment_id: mockMoments[0].moment_id },
    preview: { title: mockMoments[0].description.slice(0, 50), cover_url: mockMoments[0].images?.[0]?.url },
  }))),
  http.post(`${API}/share/list_my`, () => requireMockAuth() ?? json(paginated([]))),
  http.post(`${API}/share/revoke`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Media ──────────────────────────────────────
  http.post(`${API}/media/presign_upload`, () => requireMockAuth() ?? json(apiOk({ upload_id: "upload_1", upload_url: "/mock-upload", headers: {}, expires_at: new Date(Date.now() + 3600000).toISOString() }))),
  http.post(`${API}/media/complete_upload`, () => requireMockAuth() ?? json(apiOk({ file_id: "file_1", thumbnail_url: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=200", file_size: 102400, mime_type: "image/jpeg", width: 800, height: 600 }))),
];
