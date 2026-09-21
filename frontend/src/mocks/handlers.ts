import { http, HttpResponse, type JsonBodyType } from "msw";
import {
  apiOk, apiError, paginated,
  mockUsers, mockAccount, mockSettings,
  mockMoments, mockGroups, mockNotifies, mockChats,
  mockFriends, mockFollowing, mockFollowers,
  mockUserItem, relativeLocation,
  getMockAuthScenario, setMockLoggedIn, isMockLoggedIn,
  USER_TAGS, USER_BIOS,
} from "@/mocks/data";

const API = "/api/v1";
const PUBLIC_API = "/public/v1";
const ADMIN_API = "/admin/api/v1";

function json(data: JsonBodyType) {
  return HttpResponse.json(data);
}

// mock 瞬间 created_at 随真实时间动态漂移（isoDaysAgo），评论固定写死日期会
// 出现"评论早于发布"的时间倒置。评论时间一律派生自所属瞬间的 created_at 之后
// 数小时，保证时序恒正。
function hoursAfter(iso: string | undefined, hours: number): string {
  const base = iso ? new Date(iso).getTime() : Date.now();
  return new Date(base + hours * 3600000).toISOString();
}

/**
 * spec 分页：mock 数据量小、恒单页返回，这里仅支持 pagination.limit 截断
 * （cursor 翻页不模拟）。页面按契约传 limit 时必须生效，否则 rail/列表会
 * 渲染超出请求数量的条目。
 */
async function withPaginationLimit<T>(
  request: Request,
  items: T[],
): Promise<T[]> {
  const body = (await request
    .json()
    .catch(() => ({}))) as { pagination?: { limit?: number } } | null;
  const limit = body?.pagination?.limit;
  return typeof limit === "number" && limit > 0 ? items.slice(0, limit) : items;
}

/**
 * groups/search 的 mock 过滤/排序：keyword 模糊匹配名称/简介/标签，tag 精确匹配，
 * radius 显式传入时按距离过滤（与前端约定一致：显式半径才过滤），
 * sort_by=distance 距离升序、popular 成员数降序。
 */
function searchMockGroups(raw: unknown): typeof mockGroups {
  const body = (raw ?? {}) as {
    keyword?: string;
    tag?: string;
    radius?: number;
    sort_by?: string;
    pagination?: { limit?: number };
  };
  let items = [...mockGroups];
  const keyword = body?.keyword?.trim().toLowerCase();
  if (keyword) {
    items = items.filter((g) =>
      `${g.group.name}${g.description}${g.tags.join("")}`
        .toLowerCase()
        .includes(keyword),
    );
  }
  if (body?.tag) {
    items = items.filter((g) => g.tags.includes(body.tag!));
  }
  if (typeof body?.radius === "number" && body.radius > 0) {
    items = items.filter((g) => (g.location.distance ?? 0) <= body.radius!);
  }
  if (body?.sort_by === "distance") {
    items.sort((a, b) => (a.location.distance ?? 0) - (b.location.distance ?? 0));
  } else if (body?.sort_by === "popular") {
    items.sort((a, b) => b.group.members_count - a.group.members_count);
  }
  const limit = body?.pagination?.limit;
  return typeof limit === "number" && limit > 0 ? items.slice(0, limit) : items;
}

/**
 * bootstrap 响应：契约对齐真实后端（spec api_spec_base.md「Web 端 Cookie 契约」）——
 * 请求未携带 device_id Cookie 时随响应 Set-Cookie 下发（非 HttpOnly、SameSite=Lax、
 * Path=/，登出不清除），保证后续 auth/guest、auth/register 可从 Cookie 读取设备标识。
 */
function bootstrapJson(data: JsonBodyType, request: Request) {
  const headers: Record<string, string> = {};
  if (!request.headers.get("cookie")?.includes("device_id=")) {
    headers["Set-Cookie"] = `device_id=mock-${crypto.randomUUID()}; Path=/; Max-Age=15552000; SameSite=Lax`;
  }
  return HttpResponse.json(data, { headers });
}

/* ── Map hotspots mock 数据（spec api_spec_public.md → map/canvas）────────── */
// MapHotspotItem.location 为 RelativeLocation（point 嵌套 + 米数 distance /
// bearing_degrees，相对视口几何中心）；响应为带 mode 的 tagged union，mock 数据量
// 小，恒返回 hotspots 明细模式。
const mockUserHotspots = mockUsers.map((u, i) => ({
  object_type: "user" as const,
  object_id: u.user_id,
  title: u.nickname,
  subtitle: USER_TAGS[i],
  cover_url: u.avatar_url,
  location: relativeLocation(
    { latitude: 34.2604 + (i - 3) * 0.006, longitude: 117.1855 + (i % 2 === 0 ? 0.008 : -0.006) },
    "徐州",
    300 + i * 150,
    (i * 53) % 360,
  ),
  created_at: null,
}));
const mockGroupHotspots = mockGroups.map((g) => ({
  object_type: "group" as const,
  object_id: g.group.group_id,
  title: g.group.name,
  subtitle: g.description ?? null,
  cover_url: g.group.avatar_url,
  location: g.location,
  created_at: null,
}));
const mockMomentHotspots = mockMoments.slice(0, 6).map((m) => ({
  object_type: "moment" as const,
  object_id: m.moment_id,
  title: m.description.slice(0, 30),
  subtitle: m.location.location_name,
  cover_url: m.images?.[0]?.url ?? null,
  location: m.location,
  created_at: m.created_at,
}));

/**
 * Guard for auth-required endpoints: when the mock session is logged out,
 * return 10100 (spec's "not authenticated" signal) so the client exercises its
 * real auth-failure handling instead of silently succeeding. Returns null when
 * the session is valid, so handlers can early-return the guard's result.
 *
 * Optional-auth endpoints (moments/search, moments/fetch_nearby, groups/search,
 * search/*, map, system/*, users/open_the_door) are NOT guarded — anonymous
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
  // Web: AuthTokenPairWeb = { client_type: 'web' }（判别字段）；tokens in HttpOnly cookies (same names as spec).
  return HttpResponse.json(apiOk({ client_type: 'web' }), {
    headers: {
      "Set-Cookie": [
        "access_token=mock-jwt; Path=/; HttpOnly; SameSite=Strict",
        "refresh_token=mock-refresh; Path=/public/v1/auth/refresh_token; HttpOnly; SameSite=Strict",
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
  http.post(`${PUBLIC_API}/auth/signin`, () => loginScenarioError() ?? authOkResponse()),
  http.post(`${PUBLIC_API}/auth/register`, () => authFormScenarioError() ?? authOkResponse()),
  http.post(`${PUBLIC_API}/auth/guest`, () => authFormScenarioError() ?? authOkResponse()),
  http.post(`${API}/auth/session`, () => {
    // spec auth.md → session：有效 Token 返回 SessionStatus {user_id, expires_at}；
    // 未登录/无效 Token 返回 10100 错误信封（客户端据此直接判定匿名态）。
    if (isMockLoggedIn()) {
      return json(apiOk({ user_id: mockAccount.user_id, expires_at: new Date(Date.now() + 3600000).toISOString() }));
    }
    return json(apiError(10100, "Access Token 无效或过期"));
  }),
  http.post(`${API}/auth/signout`, () => {
    setMockLoggedIn(false);
    // Spec: clear both auth cookies with Max-Age=0 (device_id is NOT cleared).
    return HttpResponse.json(apiOk(null), {
      headers: {
        "Set-Cookie": [
          "access_token=; Path=/; Max-Age=0",
          "refresh_token=; Path=/public/v1/auth/refresh_token; Max-Age=0",
        ].join(", "),
      },
    });
  }),
  // 刷新与 mock 登录态一致：未登录（无有效 refresh cookie）按 spec 返回 10100，
  // 避免 client 的自动刷新把登出状态“复活”成登录态。
  http.post(`${PUBLIC_API}/auth/refresh_token`, () =>
    isMockLoggedIn() ? authOkResponse() : json(apiError(10100, "Refresh Token 无效或过期"))),
  http.post(`${PUBLIC_API}/auth/forgot_password`, () => json(apiOk(null))),
  http.post(`${PUBLIC_API}/auth/reset_password`, () => authFormScenarioError() ?? authOkResponse()),
  // 发送注册验证码（防枚举：无论 login_id 是否已注册均成功）
  http.post(`${PUBLIC_API}/auth/send_register_code`, () => authFormScenarioError() ?? json(apiOk(null))),

  // ── Auth 会话管理（spec: api_spec_auth.md，/api/v1/auth/*）─────────────
  // change_password：mock 无密码真值校验，恒成功（真实后端 10401=旧密码错误）。
  http.post(`${API}/auth/change_password`, () => requireMockAuth() ?? json(apiOk(null))),
  // fetch_sessions：返回当前会话 + 两个其他设备会话（当前置顶，其余按 last_active_at 倒序）。
  http.post(`${API}/auth/fetch_sessions`, () => requireMockAuth() ?? json(apiOk({
    items: [
      { session_id: "sess_cur", device_id: "mock-device-1", device_type: "Web", device_name: "Chrome on macOS", ip_address: "192.168.xxx.xxx", ip_location: "江苏徐州", created_at: "2026-09-01T08:00:00Z", last_active_at: new Date().toISOString(), is_current: true },
      { session_id: "sess_ipad", device_id: "mock-device-2", device_type: "iOS", device_name: "iPad Air", ip_address: "223.xxx.xxx.7", ip_location: "江苏南京", created_at: "2026-09-10T09:30:00Z", last_active_at: new Date(Date.now() - 3600000).toISOString(), is_current: false },
      { session_id: "sess_phone", device_id: "mock-device-3", device_type: "Android", device_name: "HarmonyOS Phone", ip_address: "117.xxx.xxx.42", ip_location: "江苏徐州", created_at: "2026-09-12T21:10:00Z", last_active_at: new Date(Date.now() - 86400000).toISOString(), is_current: false },
    ],
  }))),
  http.post(`${API}/auth/revoke_session`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/auth/revoke_other_sessions`, () => requireMockAuth() ?? json(apiOk({ revoked_count: 2 }))),

  // ── Users ──────────────────────────────────────
  // get_account / get_settings are auth-required → guarded. open_the_door is
  // optional → not guarded (anonymous sees public profile, my_state omitted).
  http.post(`${API}/users/account`, () => requireMockAuth() ?? json(apiOk(mockAccount))),
  http.post(`${API}/users/update_account`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/settings`, () => requireMockAuth() ?? json(apiOk(mockSettings))),
  // update_settings 合并进内存态，get_settings 读合并后的结果（开关切换后可回读）。
  http.post(`${API}/users/update_settings`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { settings?: Partial<typeof mockSettings> };
    Object.assign(mockSettings, body.settings ?? {});
    return json(apiOk(null));
  }),
  http.post(`${API}/users/open_the_door`, async ({ request }) => {
    // 按 user_id 路由（缺失/未知 id 回退第一位用户），stats/relation 用同一组
    // 派生数据，recent_moments 返回对应用户的瞬间（避免访问任意用户都看到同一个人）；
    // include_recent_moments=false 时省略 recent_moments（spec: user.md → open_the_door）。
    const body = (await request.json().catch(() => ({}))) as { user_id?: string; include_recent_moments?: boolean };
    const idx = Math.max(0, mockUsers.findIndex((u) => u.user_id === body?.user_id));
    const u = mockUsers[idx];
    const userMoments = mockMoments.filter((m) => m.user.user_id === u.user_id);
    const profile: Record<string, unknown> = {
      // spec: UserProfileBundle — user 为 UserItem（嵌套 user: UserInfo + stats/relation 内嵌）
      user: {
        user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
        bio: ['胶片摄影爱好者，周末常在回龙窝扫街。', '晨跑打卡第 300 天，云龙湖一圈配速 5 分 30 秒。', '黑胶收藏者，偶尔在户部山弹唱角客串。', '甜点师，探店是副业。', '独立书店店员，卖书也读书。', '旅行摄影师，地图上全是针脚。', '健身教练，撸铁也撸猫。'][idx] ?? null,
        tags: ['摄影', '户外'],
        presence_status: 'online',
        location: relativeLocation({ latitude: 34.26, longitude: 117.18 }, '徐州', 300, 45),
        stats: { moments_count: userMoments.length, friends_count: 6, followers_count: 5 + idx * 3, following_count: 5, groups_count: 2 },
        relation: { is_following: idx !== 0, is_followed_by: false, is_blocked: false },
        social_context: null,
      }
    };
    if (body.include_recent_moments) {
      profile.recent_moments = { items: userMoments.length ? userMoments : mockMoments.slice(idx * 2, idx * 2 + 4), next_cursor: null };
    }
    return json(apiOk(profile));
  }),
  // spec: users/search 返回 UserItem（隐私受控字段/查看者状态恒在 nullable）；
  // mockUsers 是扁平 UserInfo，统一经 mockUserItem 包装
  http.post(`${API}/users/search`, () => json(paginated(mockUsers.map(mockUserItem)))),
  // list_* default to the current user → auth-required when no user_id given.
  http.post(`${API}/users/fetch_friends`, () => requireMockAuth() ?? json(paginated(mockFriends))),
  http.post(`${API}/users/fetch_following`, () => requireMockAuth() ?? json(paginated(mockFollowing))),
  http.post(`${API}/users/fetch_followers`, () => requireMockAuth() ?? json(paginated(mockFollowers))),
http.post(`${API}/users/fetch_moments`, async ({ request }) => {
	// spec: user_id 指定目标用户；缺失时回退当前用户（mockUsers[0]）
	const body = (await request.json().catch(() => ({}))) as { user_id?: string };
	const target = mockUsers.find((u) => u.user_id === body?.user_id) ?? mockUsers[0];
	const own = mockMoments.filter((m) => m.user.user_id === target.user_id);
	return requireMockAuth() ?? json(paginated(own.length ? own : mockMoments.slice(0, 4)));
}),
  http.post(`${API}/users/fetch_reacted_moments`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(6, 18)))),
  http.post(`${API}/users/set_follow`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/users/set_block`, () => requireMockAuth() ?? json(apiOk(null))),
  // 我的拉黑列表（spec: users/fetch_blocked，blocked_at 倒序；解除拉黑走 set_block active=false）
  http.post(`${API}/users/fetch_blocked`, () => requireMockAuth() ?? json(paginated([
    { user: mockUsers[5], blocked_at: "2026-09-05T10:00:00Z", reason: "私信骚扰" },
    { user: mockUsers[6], blocked_at: "2026-08-21T09:00:00Z", reason: null },
  ]))),
  // 访客升级（spec: users/upgrade；返回升级后完整账户数据，user_type 变为 normal）
  http.post(`${API}/users/upgrade`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { login_id?: string };
    return json(apiOk({ ...mockAccount, user_type: "normal", email: body?.login_id ? `${body.login_id}@example.com` : mockAccount.email }));
  }),

  // ── Moments ────────────────────────────────────
  // search/open_the_door 是 auth-optional（匿名可见公开 moments）。
  http.post(`${API}/moments/search`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, mockMoments.slice(0, 12))))),
  // open_the_door 按 moment_id 路由（缺失/未知 id 回退第一条，避免 404 打断浏览）
  http.post(`${API}/moments/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const id = body?.moment_id;
    const found = mockMoments.find((m) => m.moment_id === id);
    return json(apiOk(found ?? mockMoments[0]));
  }),
  // 拥有者视角端点返回 OwnerMomentItem（MomentItem + privacy_level，spec moment.md）
http.post(`${API}/moments/fetch_created`, () => {
	const own = mockMoments.filter((m) => m.user.user_id === mockUsers[0].user_id);
	return requireMockAuth() ?? json(paginated(own.map((m) => ({ ...m, privacy_level: "public" as const }))));
}),
  http.post(`${API}/moments/fetch_following`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 8)))),
  http.post(`${API}/moments/fetch_nearby`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, mockMoments.slice(0, 12))))),
  // fetch_favorites 依赖查看者收藏关系（无匿名版）→ 需登录
  http.post(`${API}/moments/fetch_favorites`, () => requireMockAuth() ?? json(paginated(mockMoments.slice(0, 6).map((m) => ({ ...m, favorited_at: '2026-08-01T08:00:00.000Z' }))))),
  // groups/search：附近群组/首页群组轮播共用（此前漏配 handler，请求旁路真实后端）
  http.post(`${API}/groups/search`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, searchMockGroups(await request.json().catch(() => ({}))).map((g) => ({ ...g, viewer_state: isMockLoggedIn() ? g.viewer_state : null })))))),
  http.post(`${PUBLIC_API}/groups/search`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, searchMockGroups(await request.json().catch(() => ({}))).map((g) => ({ ...g, viewer_state: isMockLoggedIn() ? g.viewer_state : null })))))),
  // moments/create：回显已创建瞬间（mock 阶段不持久化，本次会话 feed 可见性由前端状态承接）
  http.post(`${API}/moments/create`, async ({ request }) =>
    requireMockAuth() ?? json(apiOk((async () => {
      const body = (await request.json().catch(() => ({}))) as { content?: string; tags?: string[] };
      const u = mockUsers[0];
      return {
        moment_id: `moment_${Date.now()}`, user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
        description: body?.content ?? '', images: [], tags: body?.tags ?? [], mentions: [],
        created_at: new Date().toISOString(), comments_count: 0, shares_count: 0,
        reactions: [{ emoji: "👍", count: 0, reacted_by_me: false }], location: null,
        is_favorited: false, is_deleted: false, recommendation: null,
      };
    })()))),
  http.post(`${API}/moments/set_favorite`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string; active?: boolean };
    return requireMockAuth() ?? json(apiOk({ moment_id: body?.moment_id ?? '', is_favorited: body?.active ?? true, favorited_at: new Date().toISOString() }));
  }),
  http.post(`${API}/moments/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/reactions/fetch`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { target_id?: string };
    return requireMockAuth() ?? json(apiOk({ items: [
      { reaction_id: `r-${body?.target_id ?? 0}-1`, target_type: "moment", target_id: body?.target_id ?? "", emoji: "👍", count: 3, reacted_by_me: false, preview_users: [mockUsers[1], mockUsers[2]] },
      { reaction_id: `r-${body?.target_id ?? 0}-2`, target_type: "moment", target_id: body?.target_id ?? "", emoji: "❤️", count: 1, reacted_by_me: false, preview_users: [mockUsers[3]] },
    ], next_cursor: null }));
  }),
  http.post(`${API}/comments/fetch_replies`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { comment_id?: string };
    const cid = body?.comment_id ?? "c";
    return requireMockAuth() ?? json(paginated([
      { comment_id: `${cid}-r1`, parent_id: cid, user: mockUsers[2], content: "同感，下次一起去！", created_at: hoursAfter(undefined, -2), replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
      { comment_id: `${cid}-r2`, parent_id: cid, user: mockUsers[4], content: "蹲一个定位 📍", created_at: hoursAfter(undefined, -1), replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
    ]));
  }),
  http.post(`${API}/comments/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/comments/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { comment_id?: string };
    return requireMockAuth() ?? json(apiOk({ comment_id: body?.comment_id ?? "c0", replies: { items: [], next_cursor: null } }));
  }),
  http.post(`${API}/private_chats/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { target_user_id?: string };
    const chat = mockChats.find((c) => c.other_user.user_id === body?.target_user_id);
    const other = chat?.other_user ?? mockUsers[1];
    return requireMockAuth() ?? json(apiOk({
      other_user: other, presence_status: "online", can_message: true,
      recent_messages: { items: [], next_cursor: null },
    }));
  }),
  http.post(`${API}/private_chats/update_preferences`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/private_chats/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/update_preferences`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/members/search`, () => requireMockAuth() ?? json(paginated([]))),
  http.post(`${API}/users/update_location`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${PUBLIC_API}/comments/fetch_replies`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { comment_id?: string };
    const cid = body?.comment_id ?? "c";
    return json(paginated([
      { comment_id: `${cid}-r1`, parent_id: cid, user: mockUsers[2], content: "同感，下次一起去！", created_at: hoursAfter(undefined, -1), replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
    ]));
  }),
  http.post(`${PUBLIC_API}/comments/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { comment_id?: string };
    return json(apiOk({ comment_id: body?.comment_id ?? "c0", replies: { items: [], next_cursor: null } }));
  }),
  http.post(`${PUBLIC_API}/reactions/fetch`, () => json(apiOk({ items: [], next_cursor: null }))),
  http.post(`${PUBLIC_API}/users/batch_fetch`, () => json(paginated(mockUsers.slice(0, 3).map(mockUserItem)))),
  http.post(`${API}/users/batch_fetch`, () => requireMockAuth() ?? json(paginated(mockUsers.slice(0, 3).map(mockUserItem)))),
  http.post(`${API}/moments/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string; description?: string; tags?: string[]; privacy_level?: string };
    const base = mockMoments.find((m) => m.moment_id === body?.moment_id) ?? mockMoments[0];
    return requireMockAuth() ?? json(apiOk({
      ...base,
      description: body.description ?? base.description,
      tags: body.tags ?? base.tags,
      // OwnerMomentItem.privacy_level：MomentItem 无此字段，未指定时按默认 public 回显
      privacy_level: body.privacy_level ?? "public",
    }));
  }),
  http.post(`${API}/moments/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  // users/set_remark（需登录；remark 传空串=清除备注）
  http.post(`${API}/users/set_remark`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Comments ───────────────────────────────────
  // list 是 auth-optional；create 需要登录。返回 spec CommentItem
  // （user: UserInfo 嵌套；reactions 为聚合结构）。
  // list 按 moment_id 派生评论（评论人/文案随瞬间变化，避免所有瞬间显示同一批评论）。
  http.post(`${API}/comments/fetch`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const seed = mockMoments.findIndex((m) => m.moment_id === body?.moment_id);
    const idx = seed < 0 ? 0 : seed;
    const postedAt = mockMoments[idx]?.created_at;
    return json(paginated([
      { comment_id: `c${idx}-1`, user: mockUsers[(idx + 1) % mockUsers.length], content: "拍得真好看！", created_at: hoursAfter(postedAt, 3), replies_count: 2, is_deleted: false, mentions: [], reactions: [{ emoji: "👍", count: 3, reacted_by_me: false }] },
      { comment_id: `c${idx}-2`, user: mockUsers[(idx + 2) % mockUsers.length], content: mockMoments[idx].tags[0] ? `这个${mockMoments[idx].tags[0]}主题好棒` : "什么相机拍的？", created_at: hoursAfter(postedAt, 5), replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
...Array.from({ length: idx % 4 }, (_, k) => ({
	comment_id: `c${idx}-x${k}`, user: mockUsers[(idx + 3 + k) % mockUsers.length],
	content: ["这个构图太舒服了", "下次带我一个", "已收藏，周末就去", "光线绝了"][k % 4],
	created_at: hoursAfter(postedAt, 6 + k), replies_count: 0, is_deleted: false, mentions: [], reactions: [],
})),
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
  // presign/complete 为数组语义（1..N）。PUT 直传到 mock CDN 由末尾 http.put 拦截返回 200。
  http.post(`${API}/media/presign_upload`, async ({ request }) => {
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
  http.post(`${API}/media/complete_upload`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { upload_ids?: string[] };
    const ids = body?.upload_ids ?? [];
    return requireMockAuth() ?? json(apiOk({
      uploads: ids.map((id) => ({ file_id: `fid-${id}`, file_size: 1024, mime_type: 'image/jpeg' })),
    }));
  }),
  // 对象存储 PUT 直传拦截（mock CDN），让发图/头像上传在 mock 环境完整跑通
  http.put('https://mock-cdn.local/upload/*', () => new HttpResponse(null, { status: 200 })),

  // ── Groups ─────────────────────────────────────
  // open_the_door/members/announcements 是 auth-optional；fetch_joined 需要登录。
  // open_the_door 按 group_id 路由（缺失/未知 id 回退第一条）。
  http.post(`${API}/groups/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string };
    const found = mockGroups.find((g) => g.group.group_id === body?.group_id);
    return json(apiOk({ ...(found ?? mockGroups[0]), viewer_state: { role: "member" as const, unread_count: 0, is_muted: false, is_pinned: false, joined_at: "2026-07-01T00:00:00Z" } }));
  }),
  http.post(`${API}/groups/fetch_joined`, () => requireMockAuth() ?? json(paginated(mockGroups.slice(0, 2)))),
  http.post(`${API}/groups/members/fetch`, () => json(paginated(mockUsers.slice(0, 4).map((u, i) => ({ user: u, role: "member", joined_at: "2026-01-01T00:00:00Z", presence_status: i % 2 === 0 ? "online" : "offline", last_online_at: null }))))),
  // 成员管理写操作（owner/admin）
  http.post(`${API}/groups/admin/members/set_role`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/admin/members/set_ban`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/admin/members/set_mute`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/admin/invites/create`, () => requireMockAuth() ?? json(apiOk({ invite_id: `inv-${Date.now()}`, expires_at: new Date(Date.now() + 86400000).toISOString() }))),
  http.post(`${API}/groups/admin/members/remove`, () => requireMockAuth() ?? json(apiOk(null))),
  // 封禁名单（读，owner/admin 可见；最近封禁在前）
  http.post(`${API}/groups/admin/members/fetch_banned`, () => json(paginated([
    { user: mockUsers[5], banned_at: "2026-07-20T08:00:00Z", reason: "恶意刷屏" },
    { user: mockUsers[6], banned_at: "2026-07-18T08:00:00Z", reason: "发布违规内容" },
  ]))),
  // 已发邀请：sent/list 默认仅返回 pending（契约过期语义：过期自动置 revoked）
  http.post(`${API}/groups/admin/invites/fetch_sent`, () => json(paginated(mockUsers.slice(1, 4).map((u, i) => ({
    invite_id: `inv-sent-${i + 1}`,
    invitee: u,
    inviter: mockUsers[0],
    status: "pending",
    created_at: new Date(Date.now() - (i + 1) * 86400000).toISOString(),
    expires_at: new Date(Date.now() + (7 - i) * 86400000).toISOString(),
  }))))),
  // 撤销已发邀请（写操作 → 需登录）
  http.post(`${API}/groups/admin/invites/revoke`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${PUBLIC_API}/groups/announcements/fetch`, () => json(apiOk({ items: [
    { announcement_id: "ann1", group_id: "group_1", title: "本周六弹唱活动", content: "下午3点户部山，带上你的乐器", created_by: mockUsers[0], created_at: "2026-07-25T10:00:00Z", is_pinned: true },
  ]}))),
  // 公告写操作（owner/admin；create 自带幂等键）
  http.post(`${API}/groups/admin/announcements/create`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; title?: string; content?: string; is_pinned?: boolean };
    return requireMockAuth() ?? json(apiOk({ announcement_id: `ann-${Date.now()}`, group_id: body?.group_id ?? 'group_1', title: body?.title ?? '', content: body?.content ?? '', created_by: mockUsers[0], created_at: new Date().toISOString(), is_pinned: Boolean(body?.is_pinned) }));
  }),
  http.post(`${API}/groups/admin/announcements/update`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; announcement_id?: string; title?: string; content?: string; is_pinned?: boolean };
    return requireMockAuth() ?? json(apiOk({ announcement_id: body?.announcement_id ?? 'ann1', group_id: body?.group_id ?? 'group_1', title: body?.title ?? '', content: body?.content ?? '', created_by: mockUsers[0], created_at: new Date().toISOString(), is_pinned: Boolean(body?.is_pinned) }));
  }),
  http.post(`${API}/groups/admin/announcements/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  // create 是写操作 → 需要登录。
  http.post(`${API}/groups/create`, () => requireMockAuth() ?? json(apiOk(mockGroups[0]))),
  // join/leave 是写操作 → 需要登录。join 按 join_method 路由：
  // direct 直接加入（返回 joined）；需审核/邀请类返回 pending_approval。
  // EnterGroupResult 为扁平结构：application_id 恒在（joined 时为 null，见 spec group.md）。
  // 为让页面可重复演练，join 总是"可成功"：joined 状态以请求为准（服务端幂等，见 spec 自然去重）。
  http.post(`${API}/groups/enter`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { group_id?: string; password?: string };
    const group = mockGroups.find((g) => g.group.group_id === body?.group_id) ?? mockGroups[0];
    const needsApproval = group.requires_approval || group.join_method === "invite_only";
    return json(apiOk(
      needsApproval
        ? { status: "pending_approval", application_id: `app_${body?.group_id ?? "group_1"}` }
        : { status: "joined", application_id: null }
    ));
  }),
  http.post(`${API}/groups/exit`, () => requireMockAuth() ?? json(apiOk(null))),
  // 群设置更新（写操作 → 需登录；内容类更新返回更新后完整 GroupItem，见 spec 设计规范）
  http.post(`${API}/groups/admin/update`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as {
      group_id?: string; name?: string; description?: string; tags?: string[];
      join_method?: string; requires_approval?: boolean; avatar_file_id?: string;
    };
    const base = mockGroups.find((g) => g.group.group_id === body?.group_id) ?? mockGroups[0];
    return json(apiOk({
      ...base,
      group: {
        ...base.group,
        name: body.name ?? base.group.name,
        avatar_url: body.avatar_file_id ? `https://mock-cdn.local/files/${body.avatar_file_id}` : base.group.avatar_url,
      },
      description: body.description ?? base.description,
      tags: body.tags ?? base.tags,
      join_method: (body.join_method ?? base.join_method) as "direct" | "password" | "invite_only",
      requires_approval: body.requires_approval ?? base.requires_approval,
    }));
  }),
  // 申请审批（管理员视角；list 默认仅返回 pending，与契约工作台语义一致）
  http.post(`${API}/groups/admin/applications/fetch`, () => json(paginated([
    { application_id: "app_1", group: mockGroups[0].group, applicant: mockUsers[1], message: "想加入一起拍照", status: "pending", created_at: new Date(Date.now() - 2 * 3600000).toISOString(), expires_at: new Date(Date.now() + 46 * 3600000).toISOString() },
    { application_id: "app_2", group: mockGroups[0].group, applicant: mockUsers[2], status: "pending", created_at: new Date(Date.now() - 5 * 3600000).toISOString(), expires_at: new Date(Date.now() + 43 * 3600000).toISOString() },
  ]))),
  http.post(`${API}/groups/admin/applications/approve`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/admin/applications/reject`, () => requireMockAuth() ?? json(apiOk(null))),
  // 我收到的邀请（received/list 默认仅 pending；inv_1 与通知 notif_4 呼应，演示通知与待办双入口）
  http.post(`${API}/groups/invites/fetch_received`, () => json(paginated([
    { invite_id: "inv_1", group: mockGroups[0].group, inviter: mockUsers[4], message: "周末一起拍照呀", status: "pending", created_at: new Date(Date.now() - 10 * 3600000).toISOString(), expires_at: new Date(Date.now() + 38 * 3600000).toISOString() },
    { invite_id: "inv_3", group: mockGroups[3].group, inviter: mockUsers[2], status: "pending", created_at: new Date(Date.now() - 26 * 3600000).toISOString(), expires_at: new Date(Date.now() + 22 * 3600000).toISOString() },
  ]))),
  http.post(`${API}/groups/invites/accept`, () => requireMockAuth() ?? json(apiOk({ group_id: "group_1" }))),
  http.post(`${API}/groups/invites/reject`, () => requireMockAuth() ?? json(apiOk(null))),
  // 转让/解散（仅群主；dismiss 成功后由前端 onDismissed 处理跳转，WS GroupDismissed 真实后端推送）
  http.post(`${API}/groups/admin/transfer_ownership`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${API}/groups/admin/dismiss`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Chat ───────────────────────────────────────
  // All chat endpoints are auth-required (spec: private_chats/*, *_messages/*).
  http.post(`${API}/private_chats/fetch`, () => requireMockAuth() ?? json(paginated(mockChats))),
  // get_history 按 target_user_id 路由：命中 mockChats 时返回 last_message 种子
  // + 一条历史消息（刷新后会话内容仍在）；新会话（未命中）返回空列表，走引导空态。
  http.post(`${API}/private_messages/fetch`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { target_user_id?: string };
    const chat = mockChats.find((c) => c.other_user.user_id === body?.target_user_id);
    if (!chat) return requireMockAuth() ?? json(paginated([]));
    // 双向对话种子：包含自己（mockUsers[0]）的消息，让左右气泡样式都能被演练。
    // spec：历史按 sent_at 倒序（新→旧）返回，客户端 reverse 后旧→新渲染
    return requireMockAuth() ?? json(paginated([
      chat.last_message,
      { message_id: `${chat.last_message.message_id}-seed3`, type: "text", content: "下次组队去拍云龙湖日出？", sender: mockUsers[0], sent_at: "2026-07-25T10:06:00Z" },
      { message_id: `${chat.last_message.message_id}-seed2`, type: "text", content: "还不错，刚从老街扫街回来", sender: mockUsers[0], sent_at: "2026-07-25T10:05:00Z" },
      { message_id: `${chat.last_message.message_id}-seed1`, type: "text", content: "嗨，最近怎么样？", sender: chat.other_user, sent_at: "2026-07-25T10:00:00Z" },
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
  http.post(`${API}/group_messages/fetch`, async ({ request }) => {
    // 按 group_id 路由：命中 mockGroups 时返回种子消息（刷新后群聊内容仍在）；
    // 未命中返回空列表。种子消息 sender 取群成员（mockUsers 前几位）。
    const body = (await request.json().catch(() => ({}))) as { group_id?: string };
    const group = mockGroups.find((g) => g.group.group_id === body?.group_id);
    if (!group) return requireMockAuth() ?? json(paginated([]));
    const sender = mockUsers[mockGroups.indexOf(group) % mockUsers.length];
    // 三条消息三个发送者（含自己），展示群聊昵称/头像与左右气泡。
    // spec：历史按 sent_at 倒序（新→旧）返回
    return requireMockAuth() ?? json(paginated([
      { message_id: `gm-${group.group.group_id}-3`, type: "text", content: "周末主题拍摄有人组队吗？", sender: mockUsers[0], sent_at: "2026-07-26T09:45:00Z" },
      { message_id: `gm-${group.group.group_id}-2`, type: "text", content: "本周活动安排见公告，记得准时参加～", sender: mockUsers[(mockGroups.indexOf(group) + 1) % mockUsers.length], sent_at: "2026-07-26T09:30:00Z" },
      { message_id: `gm-${group.group.group_id}-1`, type: "text", content: `欢迎来到「${group.group.name}」！`, sender, sent_at: "2026-07-26T09:00:00Z" },
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

  // ── Search ─────────────────────────────────────
  http.post(`${API}/search/suggest`, () => json(apiOk({
    items: [
      { type: "tag" as const, payload: { name: "植物园", usage_count: 42 } },
      { type: "tag" as const, payload: { name: "云龙湖", usage_count: 38 } },
      { type: "tag" as const, payload: { name: "咖啡馆", usage_count: 25 } },
    ],
  }))),

  // ── System ─────────────────────────────────────
  http.post(`${PUBLIC_API}/system/client_config`, () => json(apiOk({
    features: { enable_public_registration: true, enable_guest_register: true, enable_moments: true, enable_nearby_people: true },
    limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, max_chat_files: 9, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
    contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
    defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
  }))),
  http.post(`${API}/system/bootstrap`, ({ request }) => bootstrapJson(apiOk({
    bootstrap_version: "v1",
    meta: { version: "v1", status: "active" },
    client_config: {
      features: { enable_public_registration: true, enable_guest_register: true, enable_moments: true, enable_nearby_people: true },
      limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, max_chat_files: 9, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
      contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
      defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
    },
    profile: mockAccount,
    unread_summary: { notify_total: 3, notify_by_type: { social: 1, interaction: 1, group: 1, report: 0, nearby: 0, system: 0 }, private_chat_unread_count: 2, group_chat_unread_count: 1, total_badge: 6 },
  }), request)),

  // ── Notify ─────────────────────────────────────
  // All notify endpoints are auth-required.
  http.post(`${API}/notify/fetch`, () => requireMockAuth() ?? json(paginated(mockNotifies))),
  http.post(`${API}/notify/unread_summary`, () => requireMockAuth() ?? json(apiOk({ total: 3, by_type: { social: 1, interaction: 1, group: 1, report: 0, nearby: 0, system: 0 } }))),
  http.post(`${API}/notify/mark_read`, () => requireMockAuth() ?? json(apiOk(null))),

  // ── Map ────────────────────────────────────────
  // map/canvas 仅 public 版（spec api_spec_public.md）：响应为 tagged union
  // `{ mode: 'heatmap' | 'hotspots' | 'empty', ... }`，hotspots 模式携带
  // PaginatedData（next_cursor 恒在）。mock 数据量小，恒返回 hotspots 明细模式。
  // data_type 单值筛选：'user' | 'group' | 'moment'；缺省返回全部混合。
  http.post(`${PUBLIC_API}/map/canvas`, async ({ request }) => {
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
    return json(apiOk({ mode: "hotspots", hotspots: { items: pool, next_cursor: null } }));
  }),

  // ── Reports ────────────────────────────────────
  // spec ReportCreated：is_duplicate 恒在（true = 幂等回放已有举报单）
  http.post(`${API}/reports/create`, () => requireMockAuth() ?? json(apiOk({ report_id: "report_1", is_duplicate: false }))),

  // ── Share（spec: api_spec_share.md share/create）──────────────
  // 幂等端点（Idempotency-Key）；share_code 为 12 位 [A-HJ-NP-Z2-9] 大写字符集。
  http.post(`${API}/share/create`, async ({ request }) => {
    const auth = requireMockAuth();
    if (auth) return auth;
    const body = (await request.json().catch(() => ({}))) as { share_type?: string };
    const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
    const code = Array.from({ length: 12 }, () => alphabet[Math.floor(Math.random() * alphabet.length)]).join("");
    return json(apiOk({
      share_id: `share_${Date.now()}`,
      share_code: code,
      expires_at: body?.share_type === "location" ? null : new Date(Date.now() + 7 * 86400000).toISOString(),
    }));
  }),

  // 我提交的入群申请（spec: groups/applications/fetch_submitted；cancel 撤回 pending 申请）
  http.post(`${API}/groups/applications/fetch_submitted`, () => requireMockAuth() ?? json(paginated([
    { application_id: "app_sub_1", group: mockGroups[2].group, applicant: mockUsers[0], message: "想加入一起拍摄 vlog", status: "pending", created_at: new Date(Date.now() - 6 * 3600000).toISOString(), expires_at: new Date(Date.now() + 66 * 3600000).toISOString() },
    { application_id: "app_sub_2", group: mockGroups[3].group, applicant: mockUsers[0], status: "rejected", created_at: new Date(Date.now() - 3 * 86400000).toISOString(), handled_at: new Date(Date.now() - 2 * 86400000).toISOString(), handled_by: mockUsers[2] },
  ]))),
  http.post(`${API}/groups/applications/cancel`, () => requireMockAuth() ?? json(apiOk(null))),
  // ── Public 双版本端点（/public/v1）────────────────
  // 匿名浏览时 client 会切到 public 前缀；这里为常用只读端点提供 mock。
  // spec: public users/search 返回 PublicUserItem（同 UserItem 去 relation/social_context，
  // 隐私受控字段恒在 nullable）；mockUsers 是扁平 UserInfo，统一经 mockUserItem 包装
  http.post(`${PUBLIC_API}/users/search`, () => json(paginated(mockUsers.map(mockUserItem)))),
  http.post(`${PUBLIC_API}/users/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { user_id?: string; include_recent_moments?: boolean };
    const idx = Math.max(0, mockUsers.findIndex((u) => u.user_id === body?.user_id));
    const u = mockUsers[idx];
    const userMoments = mockMoments.filter((m) => m.user.user_id === u.user_id);
    const profile: Record<string, unknown> = {
      user: {
        user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
        bio: USER_BIOS[idx % USER_BIOS.length],
        tags: ['摄影', '户外'],
        presence_status: 'online',
        location: relativeLocation({ latitude: 34.26, longitude: 117.18 }, '徐州', 300, 45),
        stats: { moments_count: userMoments.length || 24, friends_count: 18, followers_count: 89, following_count: 42, groups_count: 5 },
        relation: null,
        social_context: null,
      }
    };
    if (body.include_recent_moments) {
      profile.recent_moments = { items: userMoments.length ? userMoments : mockMoments.slice(idx * 2, idx * 2 + 4), next_cursor: null };
    }
    return json(apiOk(profile));
  }),
  http.post(`${PUBLIC_API}/users/fetch_friends`, () => json(paginated(mockFriends))),
  http.post(`${PUBLIC_API}/users/fetch_following`, () => json(paginated(mockFollowing))),
  http.post(`${PUBLIC_API}/users/fetch_followers`, () => json(paginated(mockFollowers))),
  http.post(`${PUBLIC_API}/users/fetch_moments`, () => json(paginated(mockMoments.slice(0, 12)))),
  http.post(`${PUBLIC_API}/users/fetch_reacted_moments`, () => json(paginated(mockMoments.slice(6, 18)))),
  http.post(`${PUBLIC_API}/moments/search`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, mockMoments.slice(0, 12))))),
  http.post(`${PUBLIC_API}/moments/fetch_nearby`, async ({ request }) =>
    json(paginated(await withPaginationLimit(request, mockMoments.slice(0, 12))))),
  http.post(`${PUBLIC_API}/moments/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const found = mockMoments.find((m) => m.moment_id === body?.moment_id);
    return json(apiOk(found ?? mockMoments[0]));
  }),
  http.post(`${PUBLIC_API}/comments/fetch`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { moment_id?: string };
    const seed = mockMoments.findIndex((m) => m.moment_id === body?.moment_id);
    const idx = seed < 0 ? 0 : seed;
    const postedAt = mockMoments[idx]?.created_at;
    return json(paginated([
      { comment_id: `c${idx}-1`, user: mockUsers[(idx + 1) % mockUsers.length], content: "拍得真好看！", created_at: hoursAfter(postedAt, 3), replies_count: 2, is_deleted: false, mentions: [], reactions: [{ emoji: "👍", count: 3, reacted_by_me: false }] },
      { comment_id: `c${idx}-2`, user: mockUsers[(idx + 2) % mockUsers.length], content: mockMoments[idx].tags[0] ? `这个${mockMoments[idx].tags[0]}主题好棒` : "什么相机拍的？", created_at: hoursAfter(postedAt, 5), replies_count: 0, is_deleted: false, mentions: [], reactions: [] },
    ]));
  }),
  http.post(`${PUBLIC_API}/groups/open_the_door`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { group_id?: string };
    const found = mockGroups.find((g) => g.group.group_id === body?.group_id);
    return json(apiOk({ ...(found ?? mockGroups[0]), viewer_state: null }));
  }),
  http.post(`${PUBLIC_API}/groups/members/fetch`, () => json(paginated(mockUsers.slice(0, 4).map((u, i) => ({ user: u, role: "member", joined_at: "2026-01-01T00:00:00Z", presence_status: i % 2 === 0 ? "online" : "offline", last_online_at: null }))))),
  http.post(`${PUBLIC_API}/search/suggest`, () => {
    // spec api_spec_public.md → PublicSearchSuggestItem：payload 为匿名对偶类型——
    // user → PublicUserItem（user 嵌套）；group → PublicGroupItem（group 嵌套，
    // 无 viewer_state/friends_in_group 查看者字段）；tag → TagItem。
    const { viewer_state: _viewerState, ...publicGroup } = mockGroups[0];
    return json(apiOk({
      items: [
        { type: "tag", payload: { name: "植物园", usage_count: 42 } },
        { type: "user", payload: mockUserItem(mockUsers[0]) },
        { type: "group", payload: publicGroup },
      ],
    }));
  }),
  http.post(`${PUBLIC_API}/system/bootstrap`, ({ request }) => bootstrapJson(apiOk({
    bootstrap_version: "mock-v1",
    meta: { version: "v1", status: "active", deprecated_at: null, sunset_at: null, message: null, recommended_version: null },
    // ClientConfig 四个子对象恒在（spec api_spec_system.md）：limits/contact/defaults 必填
    client_config: {
      features: { enable_public_registration: true, enable_guest_register: true, enable_moments: true, enable_nearby_people: true },
      limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, max_chat_files: 9, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
      contact: { support_email: "support@hotspot.app", privacy_policy_url: "", terms_of_service_url: "", official_website_url: "" },
      defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_user_avatar_url: "", default_group_avatar_url: "" },
    },
    unread_summary: { notify_total: 0, notify_by_type: { social: 0, interaction: 0, group: 0, report: 0, nearby: 0, system: 0 }, private_chat_unread_count: 0, group_chat_unread_count: 0, total_badge: 0 },
  }), request)),

  // ── 平台管理（spec: api_spec_admin.md，/admin/api/v1；仅平台管理员）──
  // mock 当前登录用户为 superadmin（permissions: ['*']），便于开发期全模块联调。

  http.post(`${ADMIN_API}/rbac/me`, () => requireMockAuth() ?? json(apiOk({
    permissions: ["*"],
    roles: [{ role_id: "role_super", name: "超级管理员" }],
  }))),
  http.post(`${ADMIN_API}/rbac/permissions/fetch`, () => requireMockAuth() ?? json(apiOk({
    items: [
      { key: "admin.rbac.manage", description: "角色与平台管理员管理" },
      { key: "admin.audit.read", description: "查看审计日志" },
      { key: "admin.audit.read_ip", description: "查看审计日志中的 IP（PII）" },
      { key: "admin.app_config.read", description: "查看应用配置" },
      { key: "admin.app_config.write", description: "更新应用配置" },
      { key: "admin.users.read", description: "查看用户" },
      { key: "admin.users.read_ip", description: "查看用户注册/登录 IP（PII）" },
      { key: "admin.users.write", description: "封禁/解封用户" },
      { key: "admin.groups.read", description: "查看群组" },
      { key: "admin.groups.write", description: "解散群组" },
      { key: "admin.reports.read", description: "查看举报" },
      { key: "admin.reports.write", description: "裁决举报" },
      { key: "admin.tags.read", description: "查看标签" },
      { key: "admin.tags.write", description: "屏蔽/删除标签" },
      { key: "admin.announcements.read", description: "查看系统公告" },
      { key: "admin.announcements.write", description: "发布/管理系统公告" },
    ],
  }))),
  http.post(`${ADMIN_API}/rbac/roles/fetch`, () => requireMockAuth() ?? json(apiOk({
    items: [
      { role_id: "role_super", name: "超级管理员", description: "拥有全部平台权限", permissions: ["*"] },
      { role_id: "role_moderator", name: "内容审核", description: "举报裁决与内容处置", permissions: ["admin.reports.read", "admin.reports.write", "admin.tags.read", "admin.tags.write", "admin.groups.read", "admin.groups.write", "admin.audit.read"] },
      { role_id: "role_ops", name: "用户运营", description: "用户查询与封禁", permissions: ["admin.users.read", "admin.users.write", "admin.announcements.read", "admin.announcements.write"] },
    ],
  }))),
  http.post(`${ADMIN_API}/rbac/roles/create`, () => requireMockAuth() ?? json(apiOk({ role_id: `role_${Date.now()}`, name: "新角色", description: null, permissions: [] }))),
  http.post(`${ADMIN_API}/rbac/roles/update`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${ADMIN_API}/rbac/roles/delete`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${ADMIN_API}/rbac/roles/set_permissions`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${ADMIN_API}/rbac/admins/fetch`, () => requireMockAuth() ?? json(paginated([
    { user: mockUsers[0], granted_at: "2026-01-10T08:00:00Z", permissions: ["*"], roles: [{ role_id: "role_super", name: "超级管理员" }] },
    { user: mockUsers[1], granted_at: "2026-03-02T09:30:00Z", permissions: ["admin.reports.read", "admin.reports.write", "admin.tags.read", "admin.tags.write"], roles: [{ role_id: "role_moderator", name: "内容审核" }] },
  ]))),
  http.post(`${ADMIN_API}/rbac/admins/award`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${ADMIN_API}/rbac/admins/revoke`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/audit/fetch`, () => requireMockAuth() ?? json(paginated([
    { log_id: "log_8", action: "report.judge", operator: mockUsers[0], target_type: "report", target_id: "rep_2", target_name: null, details: { type: "report.judge", report_id: "rep_2", action: "resolve", note: "确认垃圾营销" }, ip_location: "江苏徐州", ip_address: "117.136.8.1", device_summary: "Chrome 152 / macOS", created_at: "2026-09-01T12:00:00Z" },
    { log_id: "log_7", action: "user.ban", operator: mockUsers[0], target_type: "user", target_id: "user_lin", target_name: "林一", details: { type: "user.ban", reason: "多次发布违规内容", duration_days: 7 }, ip_location: "江苏徐州", ip_address: "117.136.8.1", device_summary: "Chrome 152 / macOS", created_at: "2026-08-30T10:00:00Z" },
    { log_id: "log_6", action: "announcement.create", operator: mockUsers[0], target_type: "announcement", target_id: "ann_1", target_name: "国庆假期活动预告", details: { type: "announcement.create", title: "国庆假期活动预告" }, ip_location: "江苏徐州", ip_address: null, device_summary: "Chrome 152 / macOS", created_at: "2026-08-28T09:00:00Z" },
    { log_id: "log_5", action: "group.dismiss", operator: mockUsers[1], target_type: "group", target_id: "group_4", target_name: "黄楼夜读角", details: { type: "group.dismiss", reason: "长期无管理" }, ip_location: "江苏南京", ip_address: "58.241.0.2", device_summary: "Safari 19 / iPadOS", created_at: "2026-08-20T16:30:00Z" },
    { log_id: "log_4", action: "tag.set_ban", operator: mockUsers[1], target_type: "tag", target_id: "刷单", target_name: "刷单", details: { type: "tag.set_ban", name: "刷单", active: true }, ip_location: "江苏南京", ip_address: "58.241.0.2", device_summary: "Safari 19 / iPadOS", created_at: "2026-08-18T11:00:00Z" },
    { log_id: "log_3", action: "role.create", operator: mockUsers[0], target_type: "role", target_id: "role_ops", target_name: "用户运营", details: { type: "role.create", role_name: "用户运营", permissions: ["admin.users.read", "admin.users.write"] }, ip_location: "江苏徐州", ip_address: "117.136.8.1", device_summary: "Chrome 152 / macOS", created_at: "2026-08-10T08:00:00Z" },
  ]))),

  http.post(`${ADMIN_API}/app_config`, () => requireMockAuth() ?? json(apiOk({
    features: { enable_public_registration: true, enable_guest_register: true, enable_moments: true, enable_nearby_people: true },
    limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, max_chat_files: 9, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ["👍", "❤️", "🔥", "😂", "😮", "😢", "🙏"], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
    contact: { support_email: "support@hotspot.app", privacy_policy_url: "https://hotspot.app/privacy", terms_of_service_url: "https://hotspot.app/terms", official_website_url: "https://hotspot.app" },
    upload: { image_max_size: 10485760, video_max_size: 104857600, audio_max_size: 20971520, file_max_size: 52428800, allowed_image_types: ["image/jpeg", "image/png", "image/webp"], allowed_video_types: ["video/mp4", "video/webm"] },
    defaults: { default_location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" }, default_group_avatar_url: "", default_user_avatar_url: "", default_group_settings: { join_method: "direct", requires_approval: false, has_password: false, is_searchable: true, members_searchable: true, max_members: 200, location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "徐州" } } },
    rate_limits: { global_per_hour: 1000, user_per_hour: 300, auth_register_per_10minutes: 5, auth_signin_per_10minutes: 10, auth_guest_per_10minutes: 10, auth_refresh_per_10minutes: 30, auth_reset_per_10minutes: 3, search_per_minute: 20, write_per_minute: 30, moments_create_per_minute: 5, comments_create_per_minute: 10, reports_create_per_hour: 10, upload_presign_per_hour: 60, ws_subscribe_per_second: 10, admin_read_per_minute: 60, admin_write_per_minute: 20 },
    response_defaults: { search_suggest_count: 5, chat_recent_messages_count: 20, group_announcements_count: 10, system_announcements_count: 10, comment_top_replies_count: 3, user_profile_moments_count: 6 },
    updated_at: "2026-08-28T09:05:00Z",
    updated_by_user_id: "user_lumos",
  }))),
  http.post(`${ADMIN_API}/app_config/update`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/users/search`, ({ request }) => {
    if (isMockLoggedIn() === false) return requireMockAuth()!;
    return json(paginated(mockUsers.map((u, i) => ({
      ...mockUserItem(u),
      status: i === 5 ? "banned" : "normal",
      banned_until: i === 5 ? "2026-09-06T00:00:00Z" : null,
      privacy_level: "public",
      created_at: "2026-01-15T08:00:00Z",
      last_active_at: "2026-09-09T02:00:00Z",
      login_id: `login_${u.user_id.split("_")[1]}`,
      register_ip: "117.136.8.10",
      register_location: "江苏徐州",
      last_login_ip: "117.136.8.11",
      last_login_location: "江苏徐州",
      email: `${u.user_id.split("_")[1]}@example.com`,
      phone: null,
    }))));
  }),
  http.post(`${ADMIN_API}/users/open_the_door`, () => requireMockAuth() ?? json(apiError(10300, "资源不存在"))),
  http.post(`${ADMIN_API}/users/set_ban`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/groups/search`, () => requireMockAuth() ?? json(paginated(mockGroups.map((g, i) => ({
    ...g,
    viewer_state: null,
    owner: mockUsers[i % mockUsers.length],
    created_at: "2026-02-01T08:00:00Z",
  }))))),
  http.post(`${ADMIN_API}/groups/open_the_door`, () => requireMockAuth() ?? json(apiError(10300, "资源不存在"))),
  http.post(`${ADMIN_API}/groups/dismiss`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/reports/fetch`, () => requireMockAuth() ?? json(paginated([
    { report_id: "rep_1", target_type: "moment", target_id: "moment_2", reason: "spam", description: "疑似营销刷屏", status: "pending", created_at: "2026-09-08T10:00:00Z", handled_at: null, message_snapshot: null, resolution: null, reporter: mockUsers[2], handled_by: null, handle_note: null },
    { report_id: "rep_2", target_type: "user", target_id: "user_lin", reason: "harassment", description: "私信骚扰", status: "resolved", created_at: "2026-09-01T09:00:00Z", handled_at: "2026-09-01T12:00:00Z", message_snapshot: null, resolution: { action: "resolve", resolved_at: "2026-09-01T12:00:00Z", note: "确认垃圾营销" }, reporter: mockUsers[3], handled_by: mockUsers[0], handle_note: "确认垃圾营销" },
    { report_id: "rep_3", target_type: "comment", target_id: "comment_7", reason: "hate_speech", description: null, status: "pending", created_at: "2026-09-07T20:00:00Z", handled_at: null, message_snapshot: null, resolution: null, reporter: mockUsers[4], handled_by: null, handle_note: null },
    { report_id: "rep_4", target_type: "group", target_id: "group_4", reason: "misinformation", description: "群内传播不实信息", status: "dismissed", created_at: "2026-08-25T09:00:00Z", handled_at: "2026-08-26T10:00:00Z", message_snapshot: null, resolution: { action: "dismiss", resolved_at: "2026-08-26T10:00:00Z", note: "证据不足" }, reporter: mockUsers[5], handled_by: mockUsers[1], handle_note: "证据不足" },
  ]))),
  http.post(`${ADMIN_API}/reports/judge`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/tags/search`, () => requireMockAuth() ?? json(paginated([
    { name: "摄影", usage_count: 128, tag_id: "tag_1", created_at: "2026-01-01T00:00:00Z" },
    { name: "旅行", usage_count: 96, tag_id: "tag_2", created_at: "2026-01-01T00:00:00Z" },
    { name: "音乐", usage_count: 74, tag_id: "tag_3", created_at: "2026-01-01T00:00:00Z" },
    { name: "美食", usage_count: 65, tag_id: "tag_4", created_at: "2026-01-01T00:00:00Z" },
    { name: "刷单", usage_count: 12, tag_id: "tag_5", created_at: "2026-06-01T00:00:00Z" },
    { name: "代开发票", usage_count: 3, tag_id: "tag_6", created_at: "2026-06-15T00:00:00Z" },
  ]))),
  http.post(`${ADMIN_API}/tags/set_ban`, () => requireMockAuth() ?? json(apiOk(null))),

  http.post(`${ADMIN_API}/announcements/fetch`, () => requireMockAuth() ?? json(paginated([
    { announcement_id: "ann_2", title: "社区规范更新", content: "《社区规范》 v2.1 已生效，请留意点赞与评论的新约定。", severity: "warning", action_url: "https://hotspot.app/terms", created_at: "2026-08-20T10:00:00Z", created_by: mockUsers[0], send_push: true, push_sent_at: "2026-08-20T10:05:00Z", view_count: 1520 },
    { announcement_id: "ann_1", title: "国庆假期活动预告", content: "国庆期间将开启「城市漫游」主题活动，发布带 #城市漫游 标签的瞬间即可参与。", severity: "info", action_url: null, created_at: "2026-08-28T09:00:00Z", created_by: mockUsers[0], send_push: false, push_sent_at: null, view_count: 860 },
  ]))),
  http.post(`${ADMIN_API}/announcements/create`, () => requireMockAuth() ?? json(apiOk({
    announcement_id: `ann_${Date.now()}`, title: "新公告", content: "", severity: "info", action_url: null,
    created_at: new Date().toISOString(), created_by: mockUsers[0], send_push: false, push_sent_at: null, view_count: 0,
  }))),
  http.post(`${ADMIN_API}/announcements/update`, () => requireMockAuth() ?? json(apiOk(null))),
  http.post(`${ADMIN_API}/announcements/delete`, () => requireMockAuth() ?? json(apiOk(null))),
];
