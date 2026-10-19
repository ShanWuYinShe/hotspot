/**
 * Canned mock response data. All shapes match spec/api_spec_*.md.
 * Stateless pure functions — scenario/loggedIn state is set by MockInterceptor.
 *
 * ArkTS 约束：对象字面量必须对应显式声明的类/接口，故所有 mock 数据
 * 以具名常量 + 显式类型标注形式定义（对齐 core/types/api.ets 与 spec）。
 */

import { ApiEndpoints } from '../config/ApiEndpoints';
import {
  ApiResponse, PaginatedData,
  UserInfo, UserItem, UserStats, AccountInfo,
  MomentItem, CommentItem, GroupItem, GroupInfo, GroupViewerState, ReactionItem, Reaction,
  PrivateChatItem, PrivateMessageItem, MessageSentResult,
  AuthData, SessionStatus,
  TagItem,
  SearchSuggestResponse,
  ClientConfig, ApiVersionMeta,
  UnreadSummaryResult,
  MapCanvasResponse, NotifyItem,
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

function paginated<T extends Object>(items: T[], nextCursor: string | null = null): ApiResponse<Object> {
  // spec types.md：next_cursor 恒在，null = 没有更多数据
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
const MK_RELATIVE_LOCATION: RelativeLocation = { point: MK_POINT, location_name: '云龙湖', distance: 120, bearing_degrees: 0 };

const MK_RELATION: Relationship = { is_following: true, is_followed_by: false, is_blocked: false };

const MOCK_ACCOUNT: AccountInfo = {
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
  location: MK_RELATIVE_LOCATION,
  stats: MK_STATS,
  relation: MK_RELATION,
  social_context: null
};

const MK_REACTION: Reaction = { emoji: '👍', count: 0, reacted_by_me: false };

const MK_MOMENT: MomentItem = {
  moment_id: 'm1',
  description: '示例瞬间',
  images: [],
  tags: [],
  mentions: [],
  created_at: '',
  comments_count: 0,
  shares_count: 0,
  reactions: [MK_REACTION],
  user: MK_USER_1,
  is_deleted: false,
  location: MK_RELATIVE_LOCATION,
  is_favorited: false,
  recommendation: null
};

const MK_GROUP_INFO: GroupInfo = { group_id: 'g1', name: '附近群组', avatar_url: '', members_count: 42 };
const MK_GROUP_STATE: GroupViewerState = {
  role: 'member', unread_count: 0, is_muted: false, muted_until: null, is_pinned: false, joined_at: ''
};

const MK_GROUP: GroupItem = {
  group: MK_GROUP_INFO,
  description: '示例群组',
  tags: [],
  join_method: 'direct',
  requires_approval: false,
  location: MK_RELATIVE_LOCATION,
  last_activity_at: null,
  online_members_count: null,
  next_activity_at: null,
  created_at: null,
  owner_id: null,
  viewer_state: MK_GROUP_STATE,
  friends_in_group: null,
  last_message: null
};

const MOCK_AUTH: AuthData = {
  client_type: 'app',
  user_id: 'user_lumos',
  access_token: 'mock_access_token',
  refresh_token: 'mock_refresh_token',
  access_token_expires_at: '2099-01-01T00:00:00.000Z',
  refresh_token_expires_at: '2099-01-01T00:00:00.000Z'
};

const MOCK_AUTH_STATUS: SessionStatus = {
  user_id: 'user_lumos',
  expires_at: '2099-01-01T00:00:00.000Z'
};

// spec auth.md：未登录/无效 Token 时 auth/session 返回 10100 错误信封，
// 不存在「code=0 的空会话」形态——匿名态判定依据错误码而非空字段
const MOCK_PROFILE_BUNDLE: UserProfileBundle = {
  user: MK_USER_ITEM,
  recent_moments: { items: [] as MomentItem[], next_cursor: null }
};

const MOCK_COMMENT: CommentItem = {
  comment_id: 'c1',
  user: MK_USER_1,
  content: '示例评论',
  parent_id: null,
  mentions: [],
  created_at: '',
  replies_count: 0,
  is_deleted: false,
  reactions: [],
  top_replies: []
};

const MOCK_SENT: MessageSentResult = {
  message_id: 'm1',
  sent_at: '2099-01-01T00:00:00.000Z'
};

const MOCK_REACTIONS: ReactionItem[] = [
  { user: MK_USER_2, emoji: '❤️', reacted_at: '' }
];

const MOCK_UNREAD: UnreadSummaryResult = {
  total: 0,
  by_type: { social: 0, interaction: 0, group: 0, report: 0, nearby: 0, system: 0 }
};

const MOCK_CLIENT_CONFIG: ClientConfig = {
  features: {
    enable_public_registration: true,
    enable_guest_register: true,
    enable_moments: true,
    enable_nearby_people: true
  },
  limits: {
    max_group_members: 200,
    max_moment_tags: 5,
    max_moment_images: 9,
    max_chat_files: 9,
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
  status: 'active',
  deprecated_at: null,
  sunset_at: null,
  message: null,
  recommended_version: null
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

  // Auth（路径一律引用 ApiEndpoints，禁止字面量——唯一路径定义处）
  if (url === ApiEndpoints.AUTH.LOGIN) {
    switch (scenario) {
      case 'invalid-creds': return apiError(10100, '账号或密码错误');
      case 'banned': return apiError(10201, '账号已被封禁');
      case 'param-error': return apiError(10000, '缺少必填参数');
      case 'rate-limit': return apiError(10500, '请求过于频繁');
      default: return authOk();
    }
  }
  if (url === ApiEndpoints.AUTH.REGISTER || url === ApiEndpoints.AUTH.GUEST) {
    if (scenario === 'param-error') return apiError(10000, '');
    if (scenario === 'rate-limit') return apiError(10500, '');
    return authOk();
  }
  // 注册验证码（防枚举：无论 login_id 是否存在均成功，spec public.md）
  if (url === ApiEndpoints.AUTH.SEND_REGISTER_CODE) {
    if (scenario === 'param-error') return apiError(10000, '');
    if (scenario === 'rate-limit') return apiError(10500, '');
    return apiOkEmpty();
  }
  if (url === ApiEndpoints.AUTH.REFRESH_TOKEN) return apiOk(MOCK_AUTH);
  if (url === ApiEndpoints.AUTH.GET_STATUS)
    // spec auth.md：session 必须有效 Token；未登录/无效 Token 返回 10100（客户端据此判定匿名态）
    return loggedIn ? apiOk(MOCK_AUTH_STATUS) : apiError(10100, 'Access Token 无效或过期');
  if (url === ApiEndpoints.AUTH.LOGOUT) { setLoggedIn(false); return apiOkEmpty(); }

  const guard = () => loggedIn ? null : apiError(10100, 'Access Token 无效或过期');

  // Users
  if (url === ApiEndpoints.USER.GET_ACCOUNT) return guard() ?? apiOk(MOCK_ACCOUNT);
  if (url === ApiEndpoints.USER.GET_DETAIL) {
    // spec user.md → open_the_door：UserProfileBundle（user 嵌套 UserItem）
    const profile: UserProfileBundle = { user: MK_USER_ITEM, recent_moments: { items: [MK_MOMENT], next_cursor: null } };
    return apiOk(profile);
  }
  if (url === ApiEndpoints.USER.SEARCH) return paginated([MK_USER_ITEM]);
  const userListEndpoints: string[] = [ApiEndpoints.USER.LIST_FRIENDS, ApiEndpoints.USER.LIST_FOLLOWING, ApiEndpoints.USER.LIST_FOLLOWERS, ApiEndpoints.USER.LIST_GROUPS, ApiEndpoints.USER.LIST_MOMENTS, ApiEndpoints.USER.LIST_REACTED_MOMENTS];
  if (userListEndpoints.indexOf(url) !== -1) return guard() ?? paginated([] as UserItem[]);

  // Moments
  if (url === ApiEndpoints.MOMENT.SEARCH) return paginated([MK_MOMENT]);
  if (url === ApiEndpoints.MOMENT.GET_DETAIL) return apiOk(MK_MOMENT);
  if (url === ApiEndpoints.MOMENT.LIST_CREATED) return guard() ?? paginated([] as MomentItem[]);
  // 附近瞬间流（原 feed/nearby_moments 迁移；匿名可用，返回示例瞬间）
  if (url === ApiEndpoints.MOMENT.FETCH_NEARBY) return paginated([MK_MOMENT]);
  // 关注流（原 feed/following 迁移；依赖登录态）
  if (url === ApiEndpoints.MOMENT.FETCH_FOLLOWING) return guard() ?? paginated([] as MomentItem[]);

  // Groups
  if (url === ApiEndpoints.GROUP.SEARCH) return paginated([MK_GROUP]);
  if (url === ApiEndpoints.GROUP.GET_DETAIL) return apiOk(MK_GROUP);

  // Chat
  if (url === ApiEndpoints.CHAT.PRIVATE_CHATS.LIST) return guard() ?? paginated([] as PrivateChatItem[]);
  if (url === ApiEndpoints.CHAT.PRIVATE_MESSAGES.LIST) return guard() ?? paginated([] as PrivateMessageItem[]);
  if (url === ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND) return guard() ?? apiOk(MOCK_SENT);

  // Search / Notify / System / Map
  if (url === ApiEndpoints.SEARCH.SUGGEST) return apiOk(MOCK_SUGGEST);
  if (url === ApiEndpoints.NOTIFY.LIST) return guard() ?? paginated([] as NotifyItem[]);
  if (url === ApiEndpoints.NOTIFY.GET_UNREAD_COUNT) return guard() ?? apiOk(MOCK_UNREAD);
  if (url === ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG) return apiOk(MOCK_CLIENT_CONFIG);
  if (url === ApiEndpoints.SYSTEM.BOOTSTRAP) return apiOk(MOCK_BOOTSTRAP);
  if (url === ApiEndpoints.MAP.CANVAS) return apiOk(MOCK_MAP_CANVAS);
  if (url === ApiEndpoints.REACTIONS.SET) return guard() ?? apiOkEmpty();
  if (url === ApiEndpoints.REACTIONS.LIST) return paginated(MOCK_REACTIONS);
  if (url === ApiEndpoints.COMMENT.LIST) return paginated([] as CommentItem[]);
  if (url === ApiEndpoints.COMMENT.CREATE) return guard() ?? apiOk(MOCK_COMMENT);

  // Generic group write ops：命名空间兜底（非端点声明，组内写操作统一空成功）
  if (url.startsWith('/api/v1/groups/')) return guard() ?? apiOkEmpty();

  return null;
}
