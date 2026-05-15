import { api, type PaginatedData } from './client';

// ── Types ────────────────────────────────────────────────────────
export interface User {
  id: number;
  nickname: string;
  avatar_url: string;
  bio: string;
  presence_status: 'online' | 'offline';
  distance?: number;
  tags?: string[];
}

export interface Group {
  id: number;
  name: string;
  banner_url: string;
  member_count: number;
  distance?: number;
  privacy: 'public' | 'private';
}

export interface MomentItem {
  id: number;
  title: string;
  content: string;
  images: string[];
  like_count: number;
  comment_count: number;
  distance?: number;
  created_at: string;
  user: User;
  tags: string[];
}

export interface TrendingTag {
  name: string;
  usage_count: number;
}

// ── Auth ─────────────────────────────────────────────────────────
// Web 端 token 通过 HttpOnly Set-Cookie 传递，响应体为空对象
export const authService = {
  login: (login_id: string, password: string) =>
    api.post<Record<string, never>>('/auth/login', { login_id, password, client_type: 'web' }),

  register: (login_id: string, nickname: string, password: string) =>
    api.post<Record<string, never>>('/auth/register', { login_id, nickname, password, client_type: 'web' }),

  guest: (nickname: string) =>
    api.post<Record<string, never>>('/auth/guest', { nickname, client_type: 'web' }),

  logout: () => api.post('/auth/logout'),

  checkStatus: () => api.post<{ is_valid: boolean }>('/auth/check_status'),
};

// ── Users ────────────────────────────────────────────────────────
export const usersService = {
  me: () => api.post<User>('/users/me/profile/get'),

  updateProfile: (data: { nickname?: string; bio?: string; avatar_url?: string }) =>
    api.post<User>('/users/me/profile/update', data),

  search: (query: { keyword?: string; cursor?: string }) =>
    api.post<PaginatedData<User>>('/users/search', query),

  listFriends: (userId?: number) =>
    api.post<PaginatedData<User>>('/users/list_friends', userId ? { user_id: userId } : {}),

  listFollowing: (userId?: number) =>
    api.post<PaginatedData<User>>('/users/list_following', userId ? { user_id: userId } : {}),

  listFollowers: (userId?: number) =>
    api.post<PaginatedData<User>>('/users/list_followers', userId ? { user_id: userId } : {}),

  listGroups: (userId?: number) =>
    api.post<PaginatedData<Group>>('/users/list_groups', userId ? { user_id: userId } : {}),

  listMoments: (userId?: number) =>
    api.post<PaginatedData<MomentItem>>('/users/list_moments', userId ? { user_id: userId } : {}),

  follow: (userId: number) =>
    api.post('/users/me/follow', { user_id: userId }),

  unfollow: (userId: number) =>
    api.post('/users/me/unfollow', { user_id: userId }),

  updateLocation: (lat: number, lng: number) =>
    api.post('/users/me/location/update', { latitude: lat, longitude: lng }),
};

// ── Feed ─────────────────────────────────────────────────────────
export const feedService = {
  listUsers: (cursor?: string) =>
    api.post<PaginatedData<User>>('/feed/list_users', cursor ? { cursor } : {}),

  listGroups: (cursor?: string) =>
    api.post<PaginatedData<Group>>('/feed/list_groups', cursor ? { cursor } : {}),

  listMoments: (cursor?: string) =>
    api.post<PaginatedData<MomentItem>>('/feed/list_moments', cursor ? { cursor } : {}),

  listTrending: () =>
    api.post<TrendingTag[]>('/feed/list_trending'),
};

// ── Groups ───────────────────────────────────────────────────────
export const groupsService = {
  search: (query: { keyword?: string; cursor?: string }) =>
    api.post<PaginatedData<Group>>('/groups/search', query),

  getMembers: (groupId: number) =>
    api.post<PaginatedData<User>>('/groups/members/list', { group_id: groupId }),

  create: (data: { name: string; description?: string; privacy?: string }) =>
    api.post<Group>('/groups/create', data),

  join: (groupId: number) =>
    api.post('/groups/me/join', { group_id: groupId }),

  leave: (groupId: number) =>
    api.post('/groups/me/leave', { group_id: groupId }),

  listMyGroups: () =>
    api.post<Group[]>('/groups/me/list'),
};

// ── Moments ──────────────────────────────────────────────────────
export const momentsService = {
  create: (data: { title: string; content: string; images?: string[] }) =>
    api.post<MomentItem>('/moments/create', data),

  getById: (id: number) =>
    api.post<MomentItem>('/moments/get', { moment_id: id }),
};

// ── Notifications ────────────────────────────────────────────────
export const notificationsService = {
  list: () => api.post<Notification[]>('/notifications/list'),
  unreadCount: () => api.post<{ count: number }>('/notifications/unread_count'),
};

export interface Notification {
  id: number;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

// ── System ───────────────────────────────────────────────────────
export const systemService = {
  health: () => api.post<{ status: string }>('/system/health'),
};
