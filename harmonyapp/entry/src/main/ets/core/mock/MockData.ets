/**
 * Canned mock response data. All shapes match spec/api_spec_*.md.
 * Stateless pure functions — scenario/loggedIn state is set by MockInterceptor.
 *
 * ArkTS 约束：对象字面量必须对应显式声明的类/接口，故所有 mock 数据
 * 以具名常量 + 显式类型标注形式定义（对齐 core/types/api.ets 与 spec）。
 */

import {
  ApiResponse, PaginatedData,
  UserInfo, UserItem, UserStats, MyProfile,
  MomentItem, MomentMyState, CommentItem, GroupItem, GroupInfo, MyGroupState, ReactionItem,
  PrivateChatItem, PrivateMessageItem, MessageSent,
  AuthData, AuthStatus,
  TrendingBundle, TagItem,
  SearchSuggestResponse,
  ClientConfig, ApiVersionMeta,
  UnreadCountResult,
  MapCanvasResponse, NotificationItem,
  GeoPoint, Location, RelativeLocation, Relationship, UserProfileBundle
} from '../types/api';

// ── 基础 helper ────────────────────────────────────────────────

function apiOk<T extends Object>(data: T): ApiResponse<Object> {
  return { code: 0, data: data };
}

function apiOkEmpty(): ApiResponse<Object> {
  return { code: 0 };
}

function apiError(code: number, message: string): ApiResponse<Object> {
  return { code: code, message: message };
}

function paginated<T extends Object>(items: T[], nextCursor?: string): ApiResponse<Object> {
  const data: PaginatedData<T> = { items: items, next_cursor: nextCursor };
  return apiOk(data);
}

// ── Mock 实体（显式类型，对齐 spec 字段） ───────────────────────

const MK_USER_1: UserInfo = { user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: '' };
const MK_USER_2: UserInfo = { user_id: 'user_sophia', nickname: '阿禾', avatar_url: '' };
const MK_USER_3: UserInfo = { user_id: 'user_marco', nickname: '野原', avatar_url: '' };

const MK_STATS: UserStats = {
  moments_count: 0, friends_count: 0, followers_count: 0, following_count: 0, groups_count: 0
};

const MK_POINT: GeoPoint = { latitude: 34.26, longitude: 117.18 };
const MK_LOCATION: Location = { point: MK_POINT, location_name: '云龙湖' };
// 响应类位置（含 distance/bearing_degrees，spec: RelativeLocation）
const MK_RELATIVE_LOCATION: RelativeLocation = { point: MK_POINT, location_name: '云龙湖', distance: 120 };

const MK_RELATION: Relationship = { is_following: true, is_followed_by: false, is_blocked: false };

const MOCK_ACCOUNT: MyProfile = {
  user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: '',
  user_type: 'normal', bio: '', tags: [], email: '', phone: '',
  location: MK_LOCATION,
  presence_status: 'online', last_online_at: '', created_at: '',
  stats: MK_STATS
};

const MK_USER_ITEM: UserItem = {
  user: MK_USER_1,
  bio: '示例用户',
  tags: [],
  presence_status: 'online',
  last_online_at: '',
  stats: MK_STATS,
  relation: MK_RELATION
};

const MK_MOMENT_STATE: MomentMyState = { is_favorited: false };

const MK_MOMENT: MomentItem = {
  moment_id: 'm1',
  description: '示例动态',
  tags: [],
  created_at: '',
  likes_count: 0,
  comments_count: 0,
  shares_count: 0,
  user: MK_USER_1,
  location: MK_RELATIVE_LOCATION,
  my_state: MK_MOMENT_STATE
};

const MK_GROUP_INFO: GroupInfo = { group_id: 'g1', name: '附近群组', avatar_url: '', members_count: 42 };
const MK_GROUP_STATE: MyGroupState = { role: 'member', is_joined: false, unread_count: 0, is_muted: false, is_pinned: false };

const MK_GROUP: GroupItem = {
  group: MK_GROUP_INFO,
  description: '示例群组',
  location: MK_RELATIVE_LOCATION,
  my_state: MK_GROUP_STATE
};

const MOCK_AUTH: AuthData = {
  user_id: 'user_lumos',
  access_token: 'mock_access_token',
  refresh_token: 'mock_refresh_token',
  access_token_expires_at: '2099-01-01T00:00:00.000Z',
  refresh_token_expires_at: '2099-01-01T00:00:00.000Z'
};

const MOCK_AUTH_STATUS: AuthStatus = {
  is_valid: true,
  expires_at: '2099-01-01T00:00:00.000Z'
};

const MOCK_AUTH_STATUS_INVALID: AuthStatus = { is_valid: false };

const MOCK_PROFILE_BUNDLE: UserProfileBundle = {
  user: MK_USER_ITEM,
  recent_moments: { items: [] as MomentItem[], next_cursor: undefined }
};

const MOCK_COMMENT: CommentItem = {
  comment_id: 'c1',
  user: MK_USER_1,
  content: '示例评论',
  mentions: [],
  created_at: '',
  replies_count: 0,
  is_deleted: false,
  reactions: []
};

const MOCK_SENT: MessageSent = {
  message_id: 'm1',
  sent_at: '2099-01-01T00:00:00.000Z'
};

const MOCK_REACTIONS: ReactionItem[] = [
  { user: MK_USER_2, emoji: '❤️', reacted_at: '' }
];

const MOCK_UNREAD: UnreadCountResult = {
  total: 0,
  by_type: { social: 0, interaction: 0, group: 0, report: 0, nearby: 0, system: 0 }
};

const MOCK_CLIENT_CONFIG: ClientConfig = {
  features: {
    enable_public_registration: true,
    enable_guest_login: true,
    enable_moments: true,
    enable_nearby_people: true
  },
  limits: {
    max_group_members: 200,
    max_moment_tags: 5,
    max_moment_images: 9,
    moment_description_max_length: 500,
    comment_max_length: 200,
    search_radius_max: 50000,
    message_recall_window_seconds: 300,
    stranger_message_daily_quota: 10,
    reaction_emojis: ['👍', '❤️', '🔥'],
    group_invite_ttl_hours: 72,
    group_application_ttl_hours: 72
  },
  contact: {
    support_email: 'support@hotspot.example',
    privacy_policy_url: 'https://example.com/privacy',
    terms_of_service_url: 'https://example.com/terms',
    official_website_url: 'https://example.com'
  },
  defaults: {
    default_location: MK_LOCATION,
    default_user_avatar_url: '',
    default_group_avatar_url: ''
  }
};

const MOCK_META: ApiVersionMeta = {
  version: 'v1',
  status: 'active'
};

// spec BootstrapData 的未登录投影（@auth-only 的 profile/unread_summary 登录态才返回）
interface MockBootstrapData {
  bootstrap_version: string;
  meta: ApiVersionMeta;
  client_config: ClientConfig;
}

const MOCK_BOOTSTRAP: MockBootstrapData = {
  bootstrap_version: 'mock-1',
  meta: MOCK_META,
  client_config: MOCK_CLIENT_CONFIG
};

const MOCK_TRENDING: TrendingBundle = { hot_moments: [] as MomentItem[] };
const MOCK_SUGGEST: SearchSuggestResponse = { items: [] };
const MOCK_MAP_CANVAS: MapCanvasResponse = { mode: 'empty' };
const MOCK_TAGS: TagItem[] = [{ name: '云龙湖', usage_count: 12 }];

// ── In-memory state (set by MockInterceptor) ───────────────────
let loggedIn = false;
let scenario = 'ok';

export function setLoggedIn(v: boolean) { loggedIn = v; }
export function isLoggedIn(): boolean { return loggedIn; }
export function setScenarioStatic(s: string) { scenario = s; }
export function getScenario(): string { return scenario; }

// ── Scenario-driven canned responses ──────────────────────────
export function getCannedResponse(url: string, loggedIn: boolean, scenario: string): ApiResponse<Object> | null {
  const authOk = (): ApiResponse<Object> => {
    setLoggedIn(true);
    return apiOk(MOCK_AUTH);
  };

  // Auth
  if (url === '/api/v1/auth/login') {
    switch (scenario) {
      case 'invalid-creds': return apiError(10100, '账号或密码错误');
      case 'banned': return apiError(10201, '账号已被封禁');
      case 'param-error': return apiError(10000, '缺少必填参数');
      case 'rate-limit': return apiError(10500, '请求过于频繁');
      default: return authOk();
    }
  }
  if (url === '/api/v1/auth/register' || url === '/api/v1/auth/guest') {
    if (scenario === 'param-error') return apiError(10000, '');
    if (scenario === 'rate-limit') return apiError(10500, '');
    return authOk();
  }
  if (url === '/api/v1/auth/refresh_token') return apiOk(MOCK_AUTH);
  if (url === '/api/v1/auth/check_status')
    return loggedIn ? apiOk(MOCK_AUTH_STATUS) : apiOk(MOCK_AUTH_STATUS_INVALID);
  if (url === '/api/v1/auth/logout') { setLoggedIn(false); return apiOkEmpty(); }

  const guard = () => loggedIn ? null : apiError(10100, 'Access Token 无效或过期');

  // Users
  if (url === '/api/v1/users/get_account') return guard() ?? apiOk(MOCK_ACCOUNT);
  if (url === '/api/v1/users/get_profile') return apiOk(MOCK_PROFILE_BUNDLE);
  if (url === '/api/v1/users/open_the_door') return apiOk(MK_USER_ITEM);
  if (url === '/api/v1/users/search') return paginated([MK_USER_ITEM]);
  if (url.startsWith('/api/v1/users/list_')) return guard() ?? paginated([] as UserItem[]);

  // Moments
  if (url === '/api/v1/moments/search') return paginated([MK_MOMENT]);
  if (url === '/api/v1/moments/open_the_door') return apiOk(MK_MOMENT);
  if (url === '/api/v1/moments/list_my') return guard() ?? paginated([] as MomentItem[]);

  // Groups
  if (url === '/api/v1/groups/search') return paginated([MK_GROUP]);
  if (url === '/api/v1/groups/open_the_door') return apiOk(MK_GROUP);

  // Chat
  if (url === '/api/v1/private_chats/list') return guard() ?? paginated([] as PrivateChatItem[]);
  if (url === '/api/v1/private_messages/get_history') return guard() ?? paginated([] as PrivateMessageItem[]);
  if (url === '/api/v1/private_messages/send') return guard() ?? apiOk(MOCK_SENT);

  // Feed / Search / Notify / System / Map
  if (url === '/api/v1/feed/trending') return apiOk(MOCK_TRENDING);
  if (url === '/api/v1/search/suggest') return apiOk(MOCK_SUGGEST);
  if (url === '/api/v1/notify/list') return guard() ?? paginated([] as NotificationItem[]);
  if (url === '/api/v1/notify/get_unread_count') return guard() ?? apiOk(MOCK_UNREAD);
  if (url === '/api/v1/system/get_client_config') return apiOk(MOCK_CLIENT_CONFIG);
  if (url === '/api/v1/system/bootstrap') return apiOk(MOCK_BOOTSTRAP);
  if (url === '/api/v1/map/canvas') return apiOk(MOCK_MAP_CANVAS);
  if (url === '/api/v1/reactions/set') return guard() ?? apiOkEmpty();
  if (url === '/api/v1/reactions/list') return paginated(MOCK_REACTIONS);
  if (url === '/api/v1/comments/list') return paginated([] as CommentItem[]);
  if (url === '/api/v1/comments/create') return guard() ?? apiOk(MOCK_COMMENT);

  // Generic group write ops
  if (url.startsWith('/api/v1/groups/')) return guard() ?? apiOkEmpty();

  return null;
}
