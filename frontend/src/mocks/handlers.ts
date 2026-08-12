import { http, HttpResponse, type JsonBodyType } from "msw";
import {
  apiOk, apiError, paginated,
  mockUsers, mockAccount, mockSettings,
  mockMoments, mockGroups, mockNotifications, mockChats, mockTrending,
  mockFriends, mockFollowing, mockFollowers, mockNearbyUsers,
  getMockAuthScenario, setMockLoggedIn, isMockLoggedIn,
} from "@/mocks/data";

const API = "/api/v1";

function json(data: JsonBodyType) {
  return HttpResponse.json(data);
}

/* ── Map hotspots mock 数据（对齐后端 map/canvas 响应结构）────────────── */
const parseKm = (d: string | undefined) => {
  if (!d) return undefined;
  const n = parseFloat(d);
  return Number.isNaN(n) ? undefined : Math.round(n * 1000);
};
const mockUserHotspots = mockUsers.map((u, i) => ({
  object_type: "user" as const,
  object_id: u.user_id,
  title: u.nickname,
  subtitle: "附近用户",
  cover_url: u.avatar_url,
  location: { latitude: 34.2604 + (i - 3) * 0.006, longitude: 117.1855 + (i % 2 === 0 ? 0.008 : -0.006) },
  distance_meters: 300 + i * 150,
}));
const mockGroupHotspots = mockGroups.map((g) => ({
  object_type: "group" as const,
  object_id: g.group.group_id,
  title: g.group.name,
  subtitle: g.description,
  cover_url: g.group.avatar_url,
  location: { latitude: g.location.point.latitude, longitude: g.location.point.longitude },
  distance_meters: parseKm(g.location.distance),
}));
const mockMomentHotspots = mockMoments.slice(0, 6).map((m) => ({
  object_type: "moment" as const,
  object_id: m.moment_id,
  title: m.description.slice(0, 30),
  subtitle: m.location.location_name,
  cover_url: m.images?.[0]?.url,
  location: { latitude: m.location.point.latitude, longitude: m.location.point.longitude },
  distance_meters: parseKm(m.location.distance),
  created_at: m.created_at,
}));

/**
 * Guard for auth-required endpoints: when the mock session is logged out,
 * return 10100 (spec's "not authenticated" signal) so the client exercises its
 * real auth-failure handling instead of silently succeeding. Returns null when
 * the session is valid, so handlers can early-return the guard's result.
 *
 * Optional-auth endpoints (moments/search, groups/search, search/*, map,
 * system/*, users/get_profile, feed/trending) are NOT guarded — anonymous
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

  // ── Users ──────────────────────────────────────
  // get_account / get_settings are auth-required → guarded. get_profile is
  // optional → not guarded (anonymous sees public profile, my_state omitted).
  http.post(`${API}/users/get_account`, () => requireMockAuth() ?? json(apiOk(mockAccount))),
  http.post(`${API}/users/update_account`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/get_settings`, () => requireMockAuth() ?? json(apiOk(mockSettings))),
  // update_settings 合并进内存态，get_settings 读合并后的结果（开关切换后可回读）。
  http.post(`${API}/users/update_settings`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { settings?: Partial<typeof mockSettings> };
    Object.assign(mockSettings, body.settings ?? {});
    return json(apiOk(null));
  }),
  http.post(`${API}/users/get_profile`, async ({ request }) => {
    // 按 user_id 路由（缺失/未知 id 回退第一位用户），stats/relation 用同一组
    // 派生数据，recent_moments 返回对应用户的瞬间（避免访问任意用户都看到同一个人）。
    const body = (await request.json().catch(() => ({}))) as { user_id?: string };
    const idx = Math.max(0, mockUsers.findIndex((u) => u.user_id === body?.user_id));
    const u = mockUsers[idx];
    const userMoments = mockMoments.filter((m) => m.user.user_id === u.user_id);
    return json(apiOk({
      // spec: UserProfileBundle — user 为 UserItem（嵌套 user: UserInfo + stats/relation 内嵌）
      user: {
        user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
        bio: `这是 ${u.nickname} 的简介`,
        tags: ['摄影', '户外'],
        presence_status: 'online',
        location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '徐州', distance: '0.3km' },
        stats: { moments_count: userMoments.length || 24, friends_count: 18, followers_count: 89, following_count: 42, groups_count: 5 },
        relation: { is_following: idx !== 0, is_followed_by: false, is_blocked: false }
      },
      recent_moments: { items: userMoments.length ? userMoments : mockMoments.slice(idx * 2, idx * 2 + 4) },
      // 喜欢/收藏流（mock 演示数据：与最近动态区分，避免三 tab 完全同图）
      liked_moments: { items: mockMoments.slice((idx + 1) % 6, (idx + 1) % 6 + 3) },
      favorited_moments: { items: mockMoments.slice((idx + 3) % 6, (idx + 3) % 6 + 3) },
    }));
  }),
  // spec: users/search 返回 UserItem（user 嵌套 UserInfo）；mockUsers 是扁平结构，需包装
  http.post(`${API}/users/search`, () =>
    json(paginated(mockUsers.map((u) => ({ user: u, tags: [], presence_status: "offline" }))))
  ),
  // list_* default to the current user → auth-required when no user_id given.
  http.post(`${API}/users/list_friends`, () => requireMockAuth() ?? json(paginated(mockFriends))),
  http.post(`${API}/users/list_following`, () => requireMockAuth() ?? json(paginated(mockFollowing))),
  http.post(`${API}/users/list_followers`, () => requireMockAuth() ?? json(paginated(mockFollowers))),
  http.post(`${API}/users/list_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/users/list_reacted_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(6, 18)))),
  http.post(`${API}/users/set_follow`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/set_block`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Moments ────────────────────────────────────
  // search/open_the_door 是 auth-optional（匿名可见公开 moments）。
  http.post(`${API}/moments/search`, () => json(paginated(mockMoments.slice(0, 12)))),
  // open_the_door 按 moment_id 路由（缺失/未知 id 回退第一条，避免 404 打断浏览）
  http.post(`${API}/moments/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const id = body?.moment_id;
    const found = mockMoments.find((m) => m.moment_id === id);
    return json(apiOk(found ?? mockMoments[0]));
  }),
  http.post(`${API}/moments/list_my`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 9)))),
  http.post(`${API}/moments/list_favorites`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 6)))),
  http.post(`${API}/moments/set_favorite`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/moments/create`, () => requireMockAuth() ?? json(apiOk(mockMoments[0]))),
  http.post(`${API}/moments/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string; description?: string; tags?: string[]; privacy_level?: string };
    const base = mockMoments.find((m) => m.moment_id === body?.moment_id) ?? mockMoments[0];
    return requireMockAuth() ?? json(apiOk({
      ...base,
      description: body.description ?? base.description,
      tags: body.tags ?? base.tags,
      privacy_level: body.privacy_level ?? base.privacy_level,
    }));
  }),
  http.post(`${API}/moments/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  // users/set_remark（需登录；remark 传空串=清除备注）
  http.post(`${API}/users/set_remark`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Comments ───────────────────────────────────
  // list 是 auth-optional；create 需要登录。返回 spec CommentItem
  // （user: UserInfo 嵌套；reactions 为聚合结构）。
  // list 按 moment_id 派生评论（评论人/文案随瞬间变化，避免所有瞬间显示同一批评论）。
  http.post(`${API}/comments/list`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const seed = mockMoments.findIndex((m) => m.moment_id === body?.moment_id);
    const idx = seed < 0 ? 0 : seed;
    return json(paginated([
      { comment_id: `c${idx}-1`, user: mockUsers[(idx + 1) % mockUsers.length], content: "拍得真好看！", created_at: "2026-07-26T14:00:00Z", replies_count: 2, is_deleted: false, mentions: [], reactions: [{ emoji: "👍", count: 3, reacted_by_me: false }] },
      { comment_id: `c${idx}-2`, user: mockUsers[(idx + 2) % mockUsers.length], content: mockMoments[idx].tags[0] ? `这个${mockMoments[idx].tags[0]}主题好棒` : "什么相机拍的？", created_at: "2026-07-26T13:00:00Z", replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
    ]));
  }),
  http.post(`${API}/comments/create`, async ({ request }) => {
    // 回显请求内容，让乐观追加的评论在确认后保持文本（而非变空白）。
    const body = (await request.json().catch(() => ({}))) as { content?: string };
    return requireMockAuth() ?? json(apiOk({
      comment_id: `c${Date.now()}`, user: mockUsers[0], content: body?.content ?? "",
      created_at: new Date().toISOString(),
      replies_count: 0, is_deleted: false, mentions: [], reactions: [],
    }));
  }),

  // ── Reactions ──────────────────────────────────
  // set 需要登录。返回 spec ReactionItem。
  http.post(`${API}/reactions/set`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Media（spec: api_spec_media.md）──────────────
  // presign/complete/batch_* 需登录。PUT 直传到 mock CDN 由末尾 http.put 拦截返回 200。
  http.post(`${API}/media/presign_upload`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { file_name?: string };
    const uploadId = `up-${Date.now()}`;
    return requireMockAuth() ?? json(apiOk({
      upload_id: uploadId,
      upload_url: `https://mock-cdn.local/upload/${uploadId}`,
      headers: { 'Content-Type': 'application/octet-stream' },
      expires_at: new Date(Date.now() + 600000).toISOString(),
    }));
  }),
  http.post(`${API}/media/complete_upload`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { upload_id?: string };
    return requireMockAuth() ?? json(apiOk({
      file_id: `fid-${body?.upload_id ?? Date.now()}`,
      file_size: 1024,
      mime_type: 'image/jpeg',
    }));
  }),
  http.post(`${API}/media/batch_presign_upload`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { files?: { file_name?: string }[] };
    const files = body?.files ?? [];
    const ts = Date.now();
    return requireMockAuth() ?? json(apiOk({
      uploads: files.map((f, i) => {
        const uploadId = `up-${ts}-${i}`;
        return {
          upload_id: uploadId,
          upload_url: `https://mock-cdn.local/upload/${uploadId}`,
          headers: { 'Content-Type': 'application/octet-stream' },
          expires_at: new Date(Date.now() + 600000).toISOString(),
        };
      }),
    }));
  }),
  http.post(`${API}/media/batch_complete_upload`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { upload_ids?: string[] };
    const ids = body?.upload_ids ?? [];
    return requireMockAuth() ?? json(apiOk({
      uploads: ids.map((id) => ({ file_id: `fid-${id}`, file_size: 1024, mime_type: 'image/jpeg' })),
    }));
  }),
  // 对象存储 PUT 直传拦截（mock CDN），让发图/头像上传在 mock 环境完整跑通
  http.put('https://mock-cdn.local/upload/*', () => new HttpResponse(null, { status: 200 })),

  // ── Groups ─────────────────────────────────────
  // open_the_door/members/announcements 是 auth-optional；list_my 需要登录。
  // open_the_door 按 group_id 路由（缺失/未知 id 回退第一条）。
  http.post(`${API}/groups/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string };
    const found = mockGroups.find((g) => g.group.group_id === body?.group_id);
    return json(apiOk({ ...(found ?? mockGroups[0]), my_state: { role: "member" as const, is_joined: true, unread_count: 0, is_muted: false, is_pinned: false } }));
  }),
  http.post(`${API}/groups/list_my`, () => requireMockAuth() ?? json(paginated(mockGroups.slice(0, 2)))),
  http.post(`${API}/groups/members/list`, () => json(paginated(mockUsers.slice(0, 4).map(u => ({ user: u, role: "member", joined_at: "2026-01-01T00:00:00Z" }))))),
  // 成员管理写操作（owner/admin）
  http.post(`${API}/groups/members/set_role`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/members/set_ban`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/members/set_mute`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/members/invite`, () => requireMockAuth() ?? json(apiOk({ invite_id: `inv-${Date.now()}`, expires_at: new Date(Date.now() + 86400000).toISOString() }))),
  http.post(`${API}/groups/members/remove`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/announcements/list`, () => json(apiOk({ items: [
    { announcement_id: "ann1", group_id: "group_1", title: "本周六弹唱活动", content: "下午3点户部山，带上你的乐器", created_by: mockUsers[0], created_at: "2026-07-25T10:00:00Z", is_pinned: true },
  ]}))),
  // 公告写操作（owner/admin；create 自带幂等键）
  http.post(`${API}/groups/announcements/create`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; title?: string; content?: string; is_pinned?: boolean };
    return requireMockAuth() ?? json(apiOk({ announcement_id: `ann-${Date.now()}`, group_id: body?.group_id ?? 'group_1', title: body?.title ?? '', content: body?.content ?? '', created_by: mockUsers[0], created_at: new Date().toISOString(), is_pinned: Boolean(body?.is_pinned) }));
  }),
  http.post(`${API}/groups/announcements/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; announcement_id?: string; title?: string; content?: string; is_pinned?: boolean };
    return requireMockAuth() ?? json(apiOk({ announcement_id: body?.announcement_id ?? 'ann1', group_id: body?.group_id ?? 'group_1', title: body?.title ?? '', content: body?.content ?? '', created_by: mockUsers[0], created_at: new Date().toISOString(), is_pinned: Boolean(body?.is_pinned) }));
  }),
  http.post(`${API}/groups/announcements/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  // create 是写操作 → 需要登录。
  http.post(`${API}/groups/create`, () => requireMockAuth() ?? json(apiOk(mockGroups[0]))),
  // join/leave 是写操作 → 需要登录。join 按 join_method 路由：
  // direct 直接加入（返回 joined）；需审核/邀请类返回 pending_approval（带 application_id）。
  // 为让页面可重复演练，join 总是"可成功"：joined 状态以请求为准（服务端幂等，见 spec 自然去重）。
  http.post(`${API}/groups/join`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; password?: string };
    const group = mockGroups.find((g) => g.group.group_id === body?.group_id) ?? mockGroups[0];
    const needsApproval = group.requires_approval || group.join_method === "invite_only";
    return json(apiOk(
      needsApproval
        ? { status: "pending_approval", application: { application_id: `app_${body?.group_id ?? "group_1"}` } }
        : { status: "joined" }
    ));
  }),
  http.post(`${API}/groups/leave`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Chat ───────────────────────────────────────
  // All chat endpoints are auth-required (spec: private_chats/*, *_messages/*).
  http.post(`${API}/private_chats/list`, () => requireMockAuth() ?? json(paginated(mockChats))),
  // get_history 按 target_user_id 路由：命中 mockChats 时返回 last_message 种子
  // + 一条历史消息（刷新后会话内容仍在）；新会话（未命中）返回空列表，走引导空态。
  http.post(`${API}/private_messages/get_history`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { target_user_id?: string };
    const chat = mockChats.find((c) => c.other_user.user_id === body?.target_user_id);
    if (!chat) return requireMockAuth() ?? json(paginated([]));
    return requireMockAuth() ?? json(paginated([
      chat.last_message,
      { message_id: `${chat.last_message.message_id}-seed`, type: "text", content: "嗨，最近怎么样？", sender: chat.other_user, sent_at: "2026-07-25T10:00:00Z" },
    ]));
  }),
  // send: echo back a spec PrivateMessageItem so ChatView's optimistic append
  // is confirmed (message persists instead of rolling back). 回显请求 content，
  // 让消息文本在回执后保持一致。
  http.post(`${API}/private_messages/send`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { content?: string };
    return requireMockAuth() ?? json(apiOk({
      message_id: `m${Date.now()}`, type: "text", content: body?.content ?? "", sender: mockUsers[0],
      sent_at: new Date().toISOString(), is_read: false,
    }));
  }),
  http.post(`${API}/private_messages/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/group_messages/get_history`, async ({ request }) => {
    // 按 group_id 路由：命中 mockGroups 时返回种子消息（刷新后群聊内容仍在）；
    // 未命中返回空列表。种子消息 sender 取群成员（mockUsers 前几位）。
    const body = (await request.json().catch(() => ({}))) as { group_id?: string };
    const group = mockGroups.find((g) => g.group.group_id === body?.group_id);
    if (!group) return requireMockAuth() ?? json(paginated([]));
    const sender = mockUsers[mockGroups.indexOf(group) % mockUsers.length];
    return requireMockAuth() ?? json(paginated([
      { message_id: `gm-${group.group.group_id}-1`, type: "text", content: `欢迎来到「${group.group.name}」！`, sender, sent_at: "2026-07-26T09:00:00Z" },
      { message_id: `gm-${group.group.group_id}-2`, type: "text", content: "本周活动安排见公告，记得准时参加～", sender: mockUsers[(mockGroups.indexOf(group) + 1) % mockUsers.length], sent_at: "2026-07-26T09:30:00Z" },
    ]));
  }),
  http.post(`${API}/group_messages/send`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { content?: string };
    return requireMockAuth() ?? json(apiOk({
      message_id: `gm${Date.now()}`, type: "text", content: body?.content ?? "", sender: mockUsers[0],
      sent_at: new Date().toISOString(), is_read: false,
    }));
  }),
  http.post(`${API}/group_messages/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),
  // 消息撤回/编辑（spec chat：recall 受窗口/群管理员豁免，update 仅 type=text）
  http.post(`${API}/private_messages/recall`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/private_messages/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { message_id?: string; content?: string };
    return requireMockAuth() ?? json(apiOk({ message_id: body?.message_id ?? 'm1', content: body?.content ?? '', edited_at: new Date().toISOString() }));
  }),
  http.post(`${API}/group_messages/recall`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/group_messages/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { message_id?: string; content?: string };
    return requireMockAuth() ?? json(apiOk({ message_id: body?.message_id ?? 'm1', content: body?.content ?? '', edited_at: new Date().toISOString() }));
  }),

  // ── Feed ───────────────────────────────────────
  // nearby 需要登录；nearby_users/nearby_groups/trending 可选（匿名可见公开数据）。
  http.post(`${API}/feed/nearby`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${API}/feed/nearby_groups`, () => json(paginated(mockGroups))),
  http.post(`${API}/feed/nearby_users`, () => json(paginated(mockNearbyUsers))),
  http.post(`${API}/feed/trending`, () => json(apiOk(mockTrending))),

  // ── Search ─────────────────────────────────────
  http.post(`${API}/search/suggest`, () => json(apiOk({
    items: [
      { type: "tag" as const, payload: { name: "植物园", usage_count: 42 } },
      { type: "tag" as const, payload: { name: "云龙湖", usage_count: 38 } },
      { type: "tag" as const, payload: { name: "咖啡馆", usage_count: 25 } },
    ],
  }))),

  // ── System ─────────────────────────────────────
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
    unread_summary: { notification_total: 3, notification_by_type: { social: 1, interaction: 1, group: 1, report: 0, system: 0 }, private_chat_unread_count: 2, group_chat_unread_count: 1, total_badge: 6 },
  }))),

  // ── Notify ─────────────────────────────────────
  // All notify endpoints are auth-required.
  http.post(`${API}/notify/list`, () => requireMockAuth() ?? json(paginated(mockNotifications))),
  http.post(`${API}/notify/get_unread_count`, () => requireMockAuth() ?? json(apiOk({ total: 3, by_type: { social: 1, interaction: 1, group: 1, report: 0, system: 0 } }))),
  http.post(`${API}/notify/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Map ────────────────────────────────────────
  // map/canvas is auth-optional (anonymous sees public hotspots).
  // 响应结构对齐当前后端实现（backend-go/internal/modules/map）：
  // `{ hotspots?: { items } }` 无 mode 字段；location 为 GeoPoint + distance_meters。
  // data_type 单值筛选：'user' | 'group' | 'moment'；缺省/'all' 返回全部混合。
  http.post(`${API}/map/canvas`, async ({ request }) => {
    let dataType: string | undefined;
    try {
      dataType = ((await request.json()) as any)?.filter?.data_type;
    } catch {
      /* 请求体缺失时按全部返回 */
    }
    const pool =
      dataType === "user"
        ? mockUserHotspots
        : dataType === "group"
          ? mockGroupHotspots
          : dataType === "moment"
            ? mockMomentHotspots
            : [...mockUserHotspots, ...mockGroupHotspots, ...mockMomentHotspots];
    return json(apiOk({ hotspots: { items: pool } }));
  }),

  // ── Reports ────────────────────────────────────
  http.post(`${API}/reports/create`, () => requireMockAuth() ?? json(apiOk({ report_id: "report_1" }))),
];
