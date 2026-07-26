// Realistic mock data matching API response shapes (spec/api_spec_*.md).
// Used by MSW handlers during development/testing.
// Turn off via: VITE_API_MOCK=false

export function apiOk<T>(data: T) {
  return { code: 0, data };
}

export function paginated<T>(items: T[], nextCursor?: string) {
  return apiOk({ items, next_cursor: nextCursor ?? undefined });
}

// ── Auth ──────────────────────────────────────────
export const mockAuthData = {
  access_token: "mock-jwt",
  refresh_token: "mock-refresh",
  user_id: "user_lumos",
  user_type: "normal",
};

// ── Users ─────────────────────────────────────────
export const mockUsers = [
  { user_id: "user_lumos", nickname: "鹿鸣", avatar_url: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=200", tags: ["摄影", "旅行"], bio: "记录城市光影", gender: "female", location_name: "云龙湖" },
  { user_id: "user_sophia", nickname: "阿禾", avatar_url: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=200", tags: ["旅行", "美食"], bio: "在路上的生活", gender: "female", location_name: "彭城广场" },
  { user_id: "user_marco", nickname: "野原", avatar_url: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=200", tags: ["音乐", "咖啡"], bio: "音乐是灵魂的出口", gender: "male", location_name: "老街" },
  { user_id: "user_mika", nickname: "Mika", avatar_url: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=200", tags: ["美食", "探店"], bio: "吃遍徐州每一条街", gender: "female", location_name: "市中心" },
  { user_id: "user_chen", nickname: "青砚", avatar_url: "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=200", tags: ["健身", "户外"], bio: "每天进步一点点", gender: "male", location_name: "植物园" },
  { user_id: "user_lin", nickname: "林一", avatar_url: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=200", tags: ["音乐", "弹唱"], bio: "户部山的吉他手", gender: "male", location_name: "户部山" },
  { user_id: "user_xiaoman", nickname: "小满", avatar_url: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=200", tags: ["手工", "市集"], bio: "周末市集组织者", gender: "female", location_name: "云龙湖" },
];

export const mockAccount = {
  user_id: "user_lumos",
  nickname: "鹿鸣",
  avatar_url: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400",
  user_type: "normal",
  bio: "记录城市光影",
  tags: ["摄影", "旅行", "咖啡"],
  gender: "female",
  email: "lumos@example.com",
  location: { point: { latitude: 34.2604, longitude: 117.1855 }, location_name: "云龙湖" },
  presence_status: "online",
  last_online: "2026-07-26T15:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  stats: { moments_count: 72, friends_count: 42, followers_count: 315, following_count: 128, groups_count: 14 },
};

export const mockSettings = {
  privacy_level: "public",
  location_visible: true, presence_visible: true,
  social_lists_visible: true, groups_visible: true,
  reacted_moments_visible: true, stats_visible: true,
  allow_follow: true, allow_private_chat: "everyone",
  searchable: true, phone_searchable: false,
  nickname_searchable: true, location_searchable: true,
};

// ── Moments ───────────────────────────────────────
const momentImages = [
  "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=600",
  "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=600",
  "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600",
  "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=600",
  "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=600",
  "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=600",
];

export const mockMoments = Array.from({ length: 24 }, (_, i) => ({
  moment_id: `moment_${i + 1}`,
  user: mockUsers[i % mockUsers.length],
  description: ["清晨在绿意中找到了一片宁静", "城市里那些安静的角落", "周末午后最适合窝在这里看书", "湖畔的周末市集开始了", "户部山周末弹唱现场", "徐州的历史比想象的还要厚重"][i % 6],
  image_url: momentImages[i % momentImages.length],
  tag: ["自然", "城市", "生活", "活动", "音乐", "人文"][i % 6],
  location: { point: { latitude: 34.26 + i * 0.002, longitude: 117.18 + i * 0.003 }, location_name: ["植物园", "市中心", "老街手冲馆", "云龙湖", "彭城广场", "汉文化景区"][i % 6] },
  like_count: Math.floor(Math.random() * 300) + 20,
  comment_count: Math.floor(Math.random() * 30) + 1,
  created_at: new Date(Date.now() - i * 3600000).toISOString(),
}));

// ── Groups ────────────────────────────────────────
export const mockGroups = [
  { group_id: "group_1", name: "回龙窝胶片会", avatar_url: "https://images.unsplash.com/photo-1637354098670-9fb8b3d765fc?w=400", description: "胶片摄影爱好者交流", members_count: 143, location_name: "回龙窝", join_method: "open", is_searchable: true },
  { group_id: "group_2", name: "云龙湖晨跑团", avatar_url: "https://images.unsplash.com/photo-1758599669009-5a9002c09487?w=400", description: "每天清晨6点环湖跑", members_count: 89, location_name: "云龙湖", join_method: "open", is_searchable: true },
  { group_id: "group_3", name: "户部山弹唱角", avatar_url: "https://images.unsplash.com/photo-1776717494824-2036ea6cab02?w=400", description: "周末弹唱交流", members_count: 62, location_name: "户部山", join_method: "approval", is_searchable: true },
  { group_id: "group_4", name: "黄楼夜读角", avatar_url: "https://images.unsplash.com/photo-1774125384717-5c5a05ce8e9c?w=400", description: "读书分享小组", members_count: 34, location_name: "黄楼", join_method: "open", is_searchable: false },
];

// ── Notifications ─────────────────────────────────
export const mockNotifications = [
  { notification_id: "notif_1", type: "new_follower", is_read: false, created_at: "2026-07-26T14:00:00Z", payload: { user: mockUsers[1] } },
  { notification_id: "notif_2", type: "moment_liked", is_read: false, created_at: "2026-07-26T13:30:00Z", payload: { user: mockUsers[2], moment: mockMoments[0] } },
  { notification_id: "notif_3", type: "moment_commented", is_read: true, created_at: "2026-07-26T12:00:00Z", payload: { user: mockUsers[3], moment: mockMoments[1] } },
  { notification_id: "notif_4", type: "group_invite_received", is_read: false, created_at: "2026-07-26T10:00:00Z", payload: { group: mockGroups[0], inviter: mockUsers[4] } },
  { notification_id: "notif_5", type: "system_announcement", is_read: true, created_at: "2026-07-25T08:00:00Z", payload: { title: "系统升级通知", content: "HotSpot v2.0 即将发布" } },
  { notification_id: "notif_6", type: "reaction_added", is_read: false, created_at: "2026-07-26T09:00:00Z", payload: { user: mockUsers[5], emoji: "🔥", moment_id: mockMoments[2].moment_id } },
];

// ── Chats ─────────────────────────────────────────
export const mockChats = [
  { target_user_id: "user_sophia", last_message: { content: "周末一起去植物园吗？", type: "text", sent_at: "2026-07-26T14:30:00Z" }, is_pinned: true, unread_count: 2 },
  { target_user_id: "user_marco", last_message: { content: "今晚的弹唱你会来吗", type: "text", sent_at: "2026-07-26T13:00:00Z" }, unread_count: 0 },
  { target_user_id: "user_mika", last_message: { content: "推荐了一家新开的咖啡馆", type: "text", sent_at: "2026-07-25T18:00:00Z" }, unread_count: 1 },
];

// ── Trending ──────────────────────────────────────
export const mockTrending = {
  hot_tags: ["植物园", "云龙湖", "周末市集", "咖啡馆", "摄影", "户外运动", "街头摄影", "徐州美食"],
  hot_locations: [
    { location_name: "云龙湖风景区", latitude: 34.2444, longitude: 117.1989, heat: 42 },
    { location_name: "彭城广场", latitude: 34.2568, longitude: 117.1952, heat: 38 },
  ],
};
