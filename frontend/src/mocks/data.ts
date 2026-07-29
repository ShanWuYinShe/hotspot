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

export function paginated<T>(items: T[], nextCursor?: string) {
  return apiOk({ items, next_cursor: nextCursor ?? undefined });
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
// check_status reads it so the boot flow reflects the mock state.
const AUTH_LOGGED_IN_KEY = "hotspot:mock-auth-logged-in";
export function isMockLoggedIn(): boolean {
  try { return localStorage.getItem(AUTH_LOGGED_IN_KEY) === "true"; } catch { return false; }
}
export function setMockLoggedIn(v: boolean) {
  try { localStorage.setItem(AUTH_LOGGED_IN_KEY, String(v)); } catch {}
}

// ── Users (spec UserInfo: { user_id, nickname, avatar_url? }) ────
const avatarA = "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=200";
const avatarB = "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=200";

export interface MockUser {
  user_id: string;
  nickname: string;
  avatar_url: string;
}

export const mockUsers: MockUser[] = [
  { user_id: "user_lumos",   nickname: "鹿鸣", avatar_url: avatarA },
  { user_id: "user_sophia",  nickname: "阿禾", avatar_url: avatarB },
  { user_id: "user_marco",   nickname: "野原", avatar_url: avatarA },
  { user_id: "user_mika",    nickname: "Mika", avatar_url: avatarB },
  { user_id: "user_chen",    nickname: "青砚", avatar_url: avatarA },
  { user_id: "user_lin",     nickname: "林一", avatar_url: avatarB },
  { user_id: "user_xiaoman", nickname: "小满", avatar_url: avatarA },
];

// ── Relationship lists (spec FriendItem / FollowingItem / FollowerItem) ──
// Each item nests UserInfo under `user:` and carries the spec's relationship
// fields. Previously the handlers returned the bare mockUsers array (flat
// {user_id, nickname, avatar_url}), which did not match the spec item shape and
// left pages reading user.avatar_url / presence_status with undefined values.
const PRESENCES = ["online", "offline", "online", "online", "offline", "online"] as const;
const isoDaysAgo = (days: number) => new Date(Date.now() - days * 86400000).toISOString();

export const mockFriends = mockUsers.slice(0, 6).map((u, i) => ({
  user: u,
  presence_status: PRESENCES[i],
  last_online_at: isoDaysAgo(i),
  remark: i % 3 === 0 ? `${u.nickname}备注` : undefined,
  followed_at: isoDaysAgo(30 + i),
  followed_back_at: isoDaysAgo(20 + i),
}));

export const mockFollowing = mockUsers.slice(1, 6).map((u, i) => ({
  user: u,
  followed_at: isoDaysAgo(15 + i),
  is_mutual: i % 2 === 0,
  presence_status: PRESENCES[i + 1],
  last_online_at: isoDaysAgo(i + 1),
}));

export const mockFollowers = mockUsers.slice(2, 7).map((u, i) => ({
  user: u,
  followed_at: isoDaysAgo(i + 2),
  is_following_back: i % 2 === 1,
  presence_status: PRESENCES[i + 2],
  last_online_at: isoDaysAgo(i + 1),
}));

const mockLocation = (location_name: string, distance?: string) => ({
  point: { latitude: 34.26, longitude: 117.18 },
  location_name,
  distance,
});

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
  stats: { moments_count: 72, friends_count: 42, followers_count: 315, following_count: 128, groups_count: 14 },
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
// likes_count coexists with reactions per spec.
const momentPic = "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600";

export const mockMoments = Array.from({ length: 24 }, (_, i) => {
  const u = mockUsers[i % mockUsers.length];
  return {
    moment_id: `moment_${i + 1}`,
    user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
    description: ["清晨在绿意中找到了一片宁静", "城市里那些安静的角落", "周末午后最适合窝在这里看书", "湖畔的周末市集开始了", "户部山周末弹唱现场", "徐州的历史比想象的还要厚重"][i % 6],
    images: [{ url: momentPic }],
    tags: [["自然", "城市", "生活", "活动", "音乐", "人文"][i % 6]],
    created_at: new Date(Date.now() - i * 3600000).toISOString(),
    likes_count: Math.floor(Math.random() * 300) + 20,
    comments_count: Math.floor(Math.random() * 30) + 1,
    shares_count: 0,
    reactions: [{ emoji: "👍", count: Math.floor(Math.random() * 200) + 10, reacted_by_me: i % 3 === 0 }],
    location: { point: { latitude: 34.26 + i * 0.002, longitude: 117.18 + i * 0.003 }, location_name: ["植物园", "市中心", "老街手冲馆", "云龙湖", "彭城广场", "汉文化景区"][i % 6], distance: `${(0.3 + i * 0.2).toFixed(1)}km` },
    my_state: { is_favorited: i % 4 === 0 },
  };
});

// ── Reaction items (spec ReactionItem: { user, emoji, reacted_at }) ──
// Used by reactions/list (the reactions detail view for a moment/comment).
export const mockReactionItems = mockUsers.slice(0, 5).map((u, i) => ({
  user: { user_id: u.user_id, nickname: u.nickname, avatar_url: u.avatar_url },
  emoji: ["👍", "❤️", "🔥"][i % 3],
  reacted_at: new Date(Date.now() - i * 600000).toISOString(),
}));

// ── Groups (spec GroupItem: nested { group: GroupInfo }) ───────
export const mockGroups = [
  { group: { group_id: "group_1", name: "回龙窝胶片会", avatar_url: avatarA.replace("w=200", "w=400"), members_count: 143 }, description: "胶片摄影爱好者交流", location: { point: { latitude: 34.25, longitude: 117.19 }, location_name: "回龙窝", distance: "0.4km" }, tags: ["摄影"], my_state: { role: "member", is_joined: true, unread_count: 0, is_muted: false, is_pinned: false } },
  { group: { group_id: "group_2", name: "云龙湖晨跑团", avatar_url: avatarB.replace("w=200", "w=400"), members_count: 89 }, description: "每天清晨6点环湖跑", location: { point: { latitude: 34.24, longitude: 117.20 }, location_name: "云龙湖", distance: "0.8km" }, tags: ["户外"], my_state: { role: "member", is_joined: false, unread_count: 0, is_muted: false, is_pinned: false } },
  { group: { group_id: "group_3", name: "户部山弹唱角", avatar_url: avatarA.replace("w=200", "w=400"), members_count: 62 }, description: "周末弹唱交流", location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: "户部山", distance: "1.1km" }, tags: ["音乐"], my_state: { role: "member", is_joined: false, unread_count: 0, is_muted: false, is_pinned: false } },
  { group: { group_id: "group_4", name: "黄楼夜读角", avatar_url: avatarB.replace("w=200", "w=400"), members_count: 34 }, description: "读书分享小组", location: { point: { latitude: 34.27, longitude: 117.18 }, location_name: "黄楼", distance: "1.4km" }, tags: ["阅读"], my_state: { role: "member", is_joined: false, unread_count: 0, is_muted: false, is_pinned: false } },
];

// ── Notifications (spec NotifyItem) ────────────────────────────
export const mockNotifications = [
  { notification_id: "notif_1", type: "new_follower", is_read: false, created_at: "2026-07-26T14:00:00Z", payload: { type: "new_follower", actor: mockUsers[1] } },
  { notification_id: "notif_2", type: "moment_liked", is_read: false, created_at: "2026-07-26T13:30:00Z", payload: { type: "moment_liked", actor: mockUsers[2], moment_id: mockMoments[0].moment_id, moment_summary: { moment_id: mockMoments[0].moment_id, description_preview: mockMoments[0].description.slice(0, 100), cover_url: mockMoments[0].images?.[0]?.url } } },
  { notification_id: "notif_3", type: "moment_commented", is_read: true, created_at: "2026-07-26T12:00:00Z", payload: { type: "moment_commented", actor: mockUsers[3], moment_id: mockMoments[1].moment_id, moment_summary: { moment_id: mockMoments[1].moment_id, description_preview: mockMoments[1].description.slice(0, 100) }, comment_id: "c1", comment_preview: "好美！" } },
  { notification_id: "notif_4", type: "group_invite_received", is_read: false, created_at: "2026-07-26T10:00:00Z", payload: { type: "group_invite_received", actor: mockUsers[4], group_id: mockGroups[0].group.group_id, group_name: mockGroups[0].group.name, group_avatar_url: mockGroups[0].group.avatar_url, invite_id: "inv_1" } },
  { notification_id: "notif_5", type: "system_announcement", is_read: true, created_at: "2026-07-25T08:00:00Z", payload: { type: "system_announcement", announcement_id: "ann_1", severity: "info" } },
  { notification_id: "notif_6", type: "reaction_added", is_read: false, created_at: "2026-07-26T09:00:00Z", payload: { type: "reaction_added", actor: mockUsers[5], target_type: "moment", target_id: mockMoments[2].moment_id, emoji: "🔥" } },
];

// ── Chats (spec PrivateChatItem: nested other_user: UserInfo) ──
export const mockChats = [
  { other_user: mockUsers[1], presence_status: "online", unread_count: 2, is_muted: false, is_pinned: true, last_activity_at: "2026-07-26T14:30:00Z", can_message: true, last_message: { message_id: "m1", content: "周末一起去植物园吗？", type: "text", sender: mockUsers[1], sent_at: "2026-07-26T14:30:00Z" } },
  { other_user: mockUsers[2], presence_status: "offline", unread_count: 0, is_muted: false, is_pinned: false, last_activity_at: "2026-07-26T13:00:00Z", can_message: true, last_message: { message_id: "m2", content: "今晚的弹唱你会来吗", type: "text", sender: mockUsers[2], sent_at: "2026-07-26T13:00:00Z" } },
  { other_user: mockUsers[3], presence_status: "online", unread_count: 1, is_muted: false, is_pinned: false, last_activity_at: "2026-07-25T18:00:00Z", can_message: true, last_message: { message_id: "m3", content: "推荐了一家新开的咖啡馆", type: "text", sender: mockUsers[3], sent_at: "2026-07-25T18:00:00Z" } },
];

// ── Trending (spec TrendingBundle: { hot_moments }) ────────────
export const mockTrending = {
  hot_moments: mockMoments.slice(0, 6),
};
