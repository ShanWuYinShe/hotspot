// Realistic mock data matching API response shapes (spec/api_spec_*.md).
// Used by MSW handlers during development/testing.
// Turn off via: VITE_API_MOCK=false
//
// Shapes are spec-faithful: UserInfo is nested under `user:`/`other_user:` on
// MomentItem / PrivateChatItem / suggest items, and reactions carry the
// `Reaction`/`ReactionItem` shapes so pages parsing per-spec render correctly.

export function apiOk<T>(data: T) {
  return { code: 0, data };
}

/** Business error envelope (HTTP stays 200; code != 0 per spec). */
export function apiError(code: number, message: string) {
  return { code, message };
}

// spec PaginatedData：next_cursor 恒在（string | null），null = 没有更多数据
export function paginated<T>(items: T[], nextCursor?: string) {
  return apiOk({ items, next_cursor: nextCursor ?? null });
}

// ── Auth scenario switch ──────────────────────────────────────
// Drives the auth/* handlers so tests can exercise success / invalid-creds /
// banned / param-error / rate-limit paths without a backend. Persisted in
// localStorage so a scenario survives reloads.
export type MockAuthScenario = "ok" | "invalid-creds" | "banned" | "param-error" | "rate-limit";
const AUTH_SCENARIO_KEY = "hotspot:mock-auth-scenario";

export function getMockAuthScenario(): MockAuthScenario {
  try {
    const v = localStorage.getItem(AUTH_SCENARIO_KEY);
    if (v === "ok" || v === "invalid-creds" || v === "banned" || v === "param-error" || v === "rate-limit") return v;
  } catch {}
  return "ok";
}

export function setMockAuthScenario(s: MockAuthScenario) {
  try { localStorage.setItem(AUTH_SCENARIO_KEY, s); } catch {}
}

// Whether the mock session is currently "logged in". login/logout flip this;
// get_status reads it so the boot flow reflects the mock state.
const AUTH_LOGGED_IN_KEY = "hotspot:mock-auth-logged-in";
export function isMockLoggedIn(): boolean {
  try { return localStorage.getItem(AUTH_LOGGED_IN_KEY) === "true"; } catch { return false; }
}
export function setMockLoggedIn(v: boolean) {
  try { localStorage.setItem(AUTH_LOGGED_IN_KEY, String(v)); } catch {}
}

// ── Users (spec UserInfo: { user_id, nickname, avatar_url? }) ────
// 头像一人一张（URL 均已验证 200），避免列表页大量重复头像的假感
const avatarA = "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=200";
const avatarB = "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=200";
const avatarC = "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200";
const avatarD = "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=200";
const avatarE = "https://images.unsplash.com/photo-1500648767791-00dcc994a43e?w=200";
const avatarF = "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200";
const avatarG = "https://images.unsplash.com/photo-1506794778202-cad84cf45f1d?w=200";

export interface MockUser {
  user_id: string;
  nickname: string;
  avatar_url: string;
}

export const mockUsers: MockUser[] = [
  { user_id: "user_lumos",   nickname: "鹿鸣", avatar_url: avatarA },
  { user_id: "user_sophia",  nickname: "阿禾", avatar_url: avatarB },
  { user_id: "user_marco",   nickname: "野原", avatar_url: avatarC },
  { user_id: "user_mika",    nickname: "Mika", avatar_url: avatarD },
  { user_id: "user_chen",    nickname: "青砚", avatar_url: avatarE },
  { user_id: "user_lin",     nickname: "林一", avatar_url: avatarF },
  { user_id: "user_xiaoman", nickname: "小满", avatar_url: avatarG },
];

// ── Relationship lists (spec RelatedUserItem: list_friends/list_following/list_followers 共用) ──
// user 为 UserItem（user 嵌套 UserInfo + presence 隐私受控字段）；
// followed_back_at 仅 list_friends；remark 仅查询自己时返回。
const PRESENCES = ["online", "offline", "online", "online", "offline", "online"] as const;
const isoDaysAgo = (days: number) => new Date(Date.now() - days * 86400000).toISOString();

const wrapUser = (u: (typeof mockUsers)[number], i: number) => ({
  user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
  presence_status: PRESENCES[i],
  last_online_at: isoDaysAgo(i + 1),
});

// spec RelatedUserItem 无 is_mutual 冗余字段——互关由 user.relation 计算
// （is_following && is_followed_by），mock 以 relation 承载该信息。
const relation = (is_following: boolean, is_followed_by: boolean) => ({
  is_following,
  is_followed_by,
  is_blocked: false,
});

export const mockFriends = mockUsers.slice(1, 7).map((u, i) => ({
  user: { ...wrapUser(u, i), relation: relation(true, true) },
  remark: ['胶片搭子', undefined, '晨跑伴', undefined, undefined, '书友'][i],
  followed_at: isoDaysAgo(30 + i),
  followed_back_at: isoDaysAgo(20 + i),
}));

// feed/nearby_users 废除后附近用户由 users/search（radius + sort_by='distance'）承担，
// mock 见 handlers.ts 的 users/search（mockUsers 包装为 UserItem）。

export const mockFollowing = mockUsers.slice(1, 6).map((u, i) => ({
  user: { ...wrapUser(u, i + 1), relation: relation(true, i % 2 === 0) },
  followed_at: isoDaysAgo(15 + i),
  remark: i === 0 ? '摄影同好' : undefined,
}));

export const mockFollowers = mockUsers.slice(2, 7).map((u, i) => ({
  user: { ...wrapUser(u, i + 2), relation: relation(i % 2 === 1, true) },
  followed_at: isoDaysAgo(i + 2),
}));

// spec UserItem：隐私受控字段与查看者状态恒在（null = 隐藏/不适用；搜索/列表场景
// stats/social_context 为 null）。供 users/search 等把扁平 UserInfo 包装成 UserItem。
// presence/location/tags 按用户索引稳定派生，让「附近的人」卡片能展示
// 距离/在线角标/兴趣 tag（此前恒 offline + location:null，卡片信息缺失）。
const USER_PRESENCES = ["online", "offline", "online", "offline", "online", "offline", "online"] as const;
export const USER_TAGS = ["摄影", "户外", "音乐", "美食", "阅读", "旅行", "健身"] as const;

export function mockUserItem(u: MockUser) {
  const i = Math.max(0, mockUsers.indexOf(u));
  return {
    user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
    bio: null,
    tags: [USER_TAGS[i % USER_TAGS.length]],
    presence_status: USER_PRESENCES[i % USER_PRESENCES.length],
    last_online_at: null,
    location: relativeLocation(
      { latitude: 34.25 + i * 0.002, longitude: 117.18 + i * 0.002 },
      "徐州",
      300 + i * 150,
      (i * 47) % 360
    ),
    stats: null,
    relation: null,
    social_context: null,
  };
}

// spec RelativeLocation：distance（米）/ bearing_degrees 由服务端基于参考点计算，
// 恒为 number | null（禁止 "0.4km" 之类格式化字符串）。
export function relativeLocation(
  point: { latitude: number; longitude: number },
  location_name: string,
  distance: number | null,
  bearing_degrees: number | null,
) {
  return { point, location_name, distance, bearing_degrees };
}

// ── Account (spec AccountData) ─────────────────────────────────
export const mockAccount = {
  user_id: "user_lumos",
  nickname: "鹿鸣",
  avatar_url: avatarA.replace("w=200", "w=400"),
  user_type: "normal",
  bio: "记录城市光影",
  tags: ["摄影", "旅行", "咖啡"],
  gender: "female",
  email: "lumos@example.com",
  phone: "",
  location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "云龙湖" },
  presence_status: "online",
  last_online_at: "2026-07-26T15:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  stats: { moments_count: 4, friends_count: 6, followers_count: 5, following_count: 5, groups_count: 2 },
};

// ── Settings (spec UserSettings) ───────────────────────────────
export const mockSettings = {
  privacy_level: "public",
  location_visible: true, presence_visible: true,
  social_lists_visible: true, groups_visible: true,
  reacted_moments_visible: true, stats_visible: true,
  allow_follow: true, allow_private_chat: "everyone",
  searchable: true, phone_searchable: false,
  nickname_searchable: true, location_searchable: true,
};

// ── Moments (spec MomentItem) ──────────────────────────────────
// user is a nested UserInfo; reactions carry the Reaction aggregate shape;
// privacy_level 不在 MomentItem——拥有者视角（OwnerMomentItem）由 handlers.ts
// 的 fetch_created/create/update 响应显式补充。
const momentPic = "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600";
// 图片池：随索引轮换，避免信息流/详情/档案页全部同图（mock 观感）。
const momentPics = [
  "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600",
  "https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=600",
  "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=600",
  "https://images.unsplash.com/photo-1477959858617-67f85cf4f1df?w=600",
  "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=600",
  "https://images.unsplash.com/photo-1500673922987-e212871fec22?w=600",
];

export const mockMoments = Array.from({ length: 24 }, (_, i) => {
  const u = mockUsers[i % mockUsers.length];
  return {
    moment_id: `moment_${i + 1}`,
    user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
    // 12 条描述轮换：默认一页 12 条内不出现重复文案。
    description: [
      "清晨在绿意中找到了一片宁静", "城市里那些安静的角落", "周末午后最适合窝在这里看书",
      "湖畔的周末市集开始了", "户部山周末弹唱现场", "徐州的历史比想象的还要厚重",
      "晚风把一整天的疲惫都吹散了", "巷口的面馆还是记忆里的味道", "骑行二十公里只为这场日落",
      "书店角落里淘到一本绝版画册", "雨后的云龙湖像加了滤镜", "老街的灯笼亮起来了",
    ][i % 12],
    images: [{ url: momentPics[i % momentPics.length] }],
    tags: [["自然", "城市", "生活", "活动", "音乐", "人文"][i % 6]],
    mentions: [] as string[],
    created_at: isoDaysAgo(45 + i),
    comments_count: 2 + (i % 4),
    shares_count: 0,
    reactions: [{ emoji: "👍", count: 30 + ((i * 37) % 160), reacted_by_me: i % 3 === 0 }],
    location: { point: { latitude: 34.26 + i * 0.002, longitude: 117.18 + i * 0.003 }, location_name: ["植物园", "市中心", "老街手冲馆", "云龙湖", "彭城广场", "汉文化景区"][i % 6], distance: 300 + i * 200, bearing_degrees: (i * 47) % 360 },
    is_favorited: i % 4 === 0,
    is_deleted: false,
    // spec：recommendation 为 RecommendationType | null（Feed 场景之外为 null）
    recommendation: (["nearby", null, null] as const)[i % 3],
  };
});

// ── Groups (spec GroupItem: nested { group: GroupInfo }) ───────
// location 为 RelativeLocation：distance（米）/bearing_degrees 数值恒在。
export const mockGroups = [
  { group: { group_id: "group_1", name: "回龙窝胶片会", avatar_url: "https://images.unsplash.com/photo-1452587925148-ce544e77e70d?w=400", members_count: 143 }, description: "胶片摄影爱好者交流", join_method: "direct", requires_approval: false, online_members_count: 12, location: { point: { latitude: 34.25, longitude: 117.19 }, location_name: "回龙窝", distance: 400, bearing_degrees: 210 }, tags: ["摄影"], viewer_state: { role: "member", unread_count: 0, is_muted: false, is_pinned: false, joined_at: "2026-07-01T00:00:00Z" } },
  { group: { group_id: "group_2", name: "云龙湖晨跑团", avatar_url: "https://images.unsplash.com/photo-1476480862126-209bfaa8edc8?w=400", members_count: 89 }, description: "每天清晨6点环湖跑", join_method: "password", requires_approval: false, online_members_count: 8, location: { point: { latitude: 34.24, longitude: 117.20 }, location_name: "云龙湖", distance: 800, bearing_degrees: 160 }, tags: ["户外"], viewer_state: null },
  { group: { group_id: "group_3", name: "户部山弹唱角", avatar_url: "https://images.unsplash.com/photo-1511671782779-c97d3d27a1d4?w=400", members_count: 62 }, description: "周末弹唱交流", join_method: "invite_only", requires_approval: true, online_members_count: 5, location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: "户部山", distance: 1100, bearing_degrees: 45 }, tags: ["音乐"], viewer_state: null },
  { group: { group_id: "group_4", name: "黄楼夜读角", avatar_url: "https://images.unsplash.com/photo-1481627834876-b7833e8f5570?w=400", members_count: 34 }, description: "读书分享小组", join_method: "direct", requires_approval: true, online_members_count: 3, next_activity_at: "周六晚 7 点共读《小王子》", location: { point: { latitude: 34.27, longitude: 117.18 }, location_name: "黄楼", distance: 1400, bearing_degrees: 330 }, tags: ["阅读"], viewer_state: null },
];

// ── Notify (spec NotifyItem) ────────────────────────────
export const mockNotifies = [
  { notify_id: "notif_1", type: "new_follower", is_read: false, created_at: "2026-07-26T14:00:00Z", payload: { type: "new_follower", actor: mockUsers[1] } },
  { notify_id: "notif_2", type: "reaction_added", is_read: false, created_at: "2026-07-26T13:30:00Z", payload: { type: "reaction_added", actor: mockUsers[2], target_type: "moment", target_id: mockMoments[0].moment_id, emoji: "❤️" } },
  { notify_id: "notif_3", type: "moment_commented", is_read: true, created_at: "2026-07-26T12:00:00Z", payload: { type: "moment_commented", actor: mockUsers[3], moment_id: mockMoments[1].moment_id, moment_summary: { moment_id: mockMoments[1].moment_id, description_preview: mockMoments[1].description.slice(0, 100) }, comment_id: "c1", comment_preview: "好美！" } },
  { notify_id: "notif_4", type: "group_invite_received", is_read: false, created_at: "2026-07-26T10:00:00Z", payload: { type: "group_invite_received", actor: mockUsers[4], group_id: mockGroups[0].group.group_id, group_name: mockGroups[0].group.name, group_avatar_url: mockGroups[0].group.avatar_url, invite_id: "inv_1" } },
  { notify_id: "notif_5", type: "system_announcement", is_read: true, created_at: "2026-07-25T08:00:00Z", payload: { type: "system_announcement", announcement_id: "ann_1", title: "服务升级通知", severity: "info" } },
  { notify_id: "notif_6", type: "reaction_added", is_read: false, created_at: "2026-07-26T09:00:00Z", payload: { type: "reaction_added", actor: mockUsers[5], target_type: "moment", target_id: mockMoments[2].moment_id, emoji: "🔥" } },
  { notify_id: "notif_7", type: "nearby_new_moment", is_read: false, created_at: "2026-07-26T08:30:00Z", payload: { type: "nearby_new_moment", actor: mockUsers[6], moment_id: mockMoments[3].moment_id, moment_summary: { moment_id: mockMoments[3].moment_id, description_preview: mockMoments[3].description.slice(0, 100), cover_url: mockMoments[3].images?.[0]?.url }, distance: 850 } },
  { notify_id: "notif_8", type: "group_application_rejected", is_read: true, created_at: "2026-07-25T20:00:00Z", payload: { type: "group_application_rejected", group_id: mockGroups[1].group.group_id, group_name: mockGroups[1].group.name, application_id: "appl_1", rejected_by: mockUsers[3], reason: "名额已满" } },
];

// ── Chats (spec PrivateChatItem: nested other_user: UserInfo) ──
export const mockChats = [
  { other_user: mockUsers[1], presence_status: "online", unread_count: 2, is_muted: false, is_pinned: true, last_activity_at: "2026-07-26T14:30:00Z", can_message: true, last_message: { message_id: "m1", content: "周末一起去植物园吗？", type: "text", sender: mockUsers[1], sent_at: "2026-07-26T14:30:00Z" } },
  { other_user: mockUsers[2], presence_status: "offline", unread_count: 0, is_muted: false, is_pinned: false, last_activity_at: "2026-07-26T13:00:00Z", can_message: true, last_message: { message_id: "m2", content: "今晚的弹唱你会来吗", type: "text", sender: mockUsers[2], sent_at: "2026-07-26T13:00:00Z" } },
  { other_user: mockUsers[3], presence_status: "online", unread_count: 1, is_muted: false, is_pinned: false, last_activity_at: "2026-07-25T18:00:00Z", can_message: true, last_message: { message_id: "m3", content: "推荐了一家新开的咖啡馆", type: "text", sender: mockUsers[3], sent_at: "2026-07-25T18:00:00Z" } },
];

