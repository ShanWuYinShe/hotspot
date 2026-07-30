/**
 * Canned mock response data. All shapes match spec/api_spec_*.md.
 * Stateless pure functions — scenario/loggedIn state is set by MockInterceptor.
 */

function apiOk<T>(data: T): Record<string, Object> {
  return { code: 0, data: data as Object } as Record<string, Object>;
}

function apiError(code: number, message: string): Record<string, Object> {
  return { code: code, message: message } as Record<string, Object>;
}

function paginated<T>(items: T[], nextCursor?: string): Record<string, Object> {
  return apiOk({ items: items, next_cursor: nextCursor } as Object);
}

const MK_USER = [
  { user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: '' },
  { user_id: 'user_sophia', nickname: '阿禾', avatar_url: '' },
  { user_id: 'user_marco', nickname: '野原', avatar_url: '' },
];

const MOCK_ACCOUNT = {
  user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: '',
  user_type: 'normal', bio: '', tags: [],
  location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '云龙湖' },
  presence_status: 'online', last_online_at: '', created_at: '',
  stats: { moments_count: 0, friends_count: 0, followers_count: 0, following_count: 0, groups_count: 0 },
};

// ── In-memory state (set by MockInterceptor) ───────────────────
let loggedIn = false;
let scenario = 'ok';

export function setLoggedIn(v: boolean) { loggedIn = v; }
export function isLoggedIn(): boolean { return loggedIn; }
export function setScenarioStatic(s: string) { scenario = s; }
export function getScenario(): string { return scenario; }

// ── Scenario-driven canned responses ──────────────────────────
export function getCannedResponse(url: string, loggedIn: boolean, scenario: string): Record<string, Object> | null {
  const authOk = (): Record<string, Object> => {
    setLoggedIn(true);
    return apiOk({} as Object);
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
  if (url === '/api/v1/auth/refresh_token') return apiOk({} as Object);
  if (url === '/api/v1/auth/check_status')
    return loggedIn ? apiOk({ is_valid: true, expires_at: new Date(Date.now() + 3600000).toISOString() }) : apiOk({ is_valid: false });
  if (url === '/api/v1/auth/logout') { setLoggedIn(false); return apiOk(null); }

  const guard = () => loggedIn ? null : apiError(10100, 'Access Token 无效或过期');

  // Users
  if (url === '/api/v1/users/get_account') return guard() ?? apiOk(MOCK_ACCOUNT);
  if (url === '/api/v1/users/get_profile') return apiOk({ user: MK_USER[1], stats: { moments_count: 24, followers_count: 89, following_count: 42 }, relation: { is_following: true, is_followed_by: false, is_blocked: false } });
  if (url === '/api/v1/users/get_detail') return apiOk({ user: MK_USER[1] });
  if (url === '/api/v1/users/search') return paginated(MK_USER);
  if (url.startsWith('/api/v1/users/list_')) return guard() ?? paginated(MK_USER);

  // Moments
  if (url === '/api/v1/moments/search') return paginated([{ moment_id: 'm1', user: MK_USER[0], description: '示例动态', tags: [], created_at: '', likes_count: 0, comments_count: 0, shares_count: 0, location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '' } }]);
  if (url === '/api/v1/moments/get_detail') return apiOk({ moment_id: 'm1', user: MK_USER[0], description: '示例动态', tags: [], created_at: '', likes_count: 0, comments_count: 0, shares_count: 0, location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '' }, my_state: { is_favorited: false } });
  if (url === '/api/v1/moments/list_my') return guard() ?? paginated([]);

  // Groups
  if (url === '/api/v1/groups/search') return paginated([{ group: { group_id: 'g1', name: '附近群组', avatar_url: '', members_count: 42 }, description: '', location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '' } }]);
  if (url === '/api/v1/groups/get_detail') return apiOk({ group: { group_id: 'g1', name: '附近群组', avatar_url: '', members_count: 42 }, description: '', location: { point: { latitude: 34.26, longitude: 117.18 }, location_name: '' }, my_state: { is_joined: false } });

  // Chat
  if (url === '/api/v1/private_chats/list') return guard() ?? paginated([]);
  if (url === '/api/v1/private_messages/get_history') return guard() ?? paginated([]);
  if (url === '/api/v1/private_messages/send') return guard() ?? apiOk({ message_id: 'm1', type: 'text', content: '', sender: MK_USER[0], sent_at: '', is_read: false });

  // Feed / Search / Notify / System / Map
  if (url === '/api/v1/feed/trending') return apiOk({ hot_moments: [] });
  if (url === '/api/v1/search/suggest') return apiOk({ items: [] });
  if (url === '/api/v1/notify/list') return guard() ?? paginated([]);
  if (url === '/api/v1/notify/get_unread_count') return guard() ?? apiOk({ total: 0 });
  if (url === '/api/v1/system/get_client_config') return apiOk({
    features: { enable_public_registration: true, enable_guest_login: true, enable_moments: true, enable_nearby_people: true },
    limits: { max_group_members: 200, max_moment_tags: 5, max_moment_images: 9, moment_description_max_length: 500, comment_max_length: 200, search_radius_max: 50000, message_recall_window_seconds: 300, stranger_message_daily_quota: 10, reaction_emojis: ['👍', '❤️', '🔥'], group_invite_ttl_hours: 72, group_application_ttl_hours: 72 },
  });
  if (url === '/api/v1/system/bootstrap') return apiOk({ meta: { version: 'v1', status: 'active' } });
  if (url === '/api/v1/map/canvas') return apiOk({ mode: 'empty' });
  if (url === '/api/v1/reactions/set') return guard() ?? apiOk(null);
  if (url === '/api/v1/reactions/list') return paginated([{ user: MK_USER[1], emoji: '❤️', reacted_at: '' }]);
  if (url === '/api/v1/comments/list') return paginated([]);
  if (url === '/api/v1/comments/create') return guard() ?? apiOk({ comment_id: 'c1', user: MK_USER[0], content: '', created_at: '', replies_count: 0, is_deleted: false, mentions: [], reactions: [] });

  // Generic group write ops
  if (url.startsWith('/api/v1/groups/')) return guard() ?? apiOk(null);

  return null;
}
