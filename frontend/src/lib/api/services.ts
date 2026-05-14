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

export interface Moment {
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

export interface Event {
  id: number;
  title: string;
  description: string;
  cover_url: string;
  start_time: string;
  end_time: string;
  location: string;
  participant_count: number;
  is_joined: boolean;
  tags: string[];
}

export interface ChatMessage {
  id: number;
  sender_id: number;
  content: string;
  created_at: string;
}

export interface Conversation {
  id: number;
  type: 'user' | 'group';
  target_id: number;
  target_name: string;
  target_avatar: string;
  last_message: string;
  last_message_at: string;
  unread_count: number;
}

export interface Notification {
  id: number;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

// ── Auth ─────────────────────────────────────────────────────────
export const authService = {
  login: (username: string, password: string) =>
    api.post<{ token: string; user: User }>('/auth/login', { username, password }),

  register: (username: string, password: string, nickname: string) =>
    api.post<{ token: string; user: User }>('/auth/register', { username, password, nickname }),

  logout: () => api.post('/auth/logout'),

  me: () => api.post<User>('/auth/me'),
};

// ── Users ────────────────────────────────────────────────────────
export const usersService = {
  nearby: (cursor?: string) =>
    api.post<PaginatedData<User>>('/users/nearby', { cursor }),

  getById: (id: number) =>
    api.post<User>(`/users/${id}`),

  updateProfile: (data: Partial<User>) =>
    api.post<User>('/users/me', data),

  follow: (userId: number) =>
    api.post(`/users/${userId}/follow`),

  unfollow: (userId: number) =>
    api.post(`/users/${userId}/unfollow`),
};

// ── Groups ───────────────────────────────────────────────────────
export const groupsService = {
  nearby: (cursor?: string) =>
    api.post<PaginatedData<Group>>('/groups/nearby', { cursor }),

  getById: (id: number) =>
    api.post<Group>(`/groups/${id}`),

  join: (groupId: number) =>
    api.post(`/groups/${groupId}/join`),

  leave: (groupId: number) =>
    api.post(`/groups/${groupId}/leave`),
};

// ── Moments ──────────────────────────────────────────────────────
export const momentsService = {
  nearby: (cursor?: string) =>
    api.post<PaginatedData<Moment>>('/moments/nearby', { cursor }),

  getById: (id: number) =>
    api.post<Moment>(`/moments/${id}`),

  create: (data: { title: string; content: string; images?: string[] }) =>
    api.post<Moment>('/moments', data),
};

// ── Chats ────────────────────────────────────────────────────────
export const chatsService = {
  list: () =>
    api.post<Conversation[]>('/chats'),

  messages: (conversationId: number, cursor?: string) =>
    api.post<PaginatedData<ChatMessage>>(`/chats/${conversationId}/messages`, { cursor }),

  send: (conversationId: number, content: string) =>
    api.post<ChatMessage>(`/chats/${conversationId}/send`, { content }),
};

// ── Notifications ────────────────────────────────────────────────
export const notificationsService = {
  list: (type?: string) =>
    api.post<Notification[]>('/notifications', { type }),

  markRead: (id: number) =>
    api.post(`/notifications/${id}/read`),
};

// ── Events ───────────────────────────────────────────────────────
export const eventsService = {
  nearby: () =>
    api.post<Event[]>('/events/nearby'),

  getById: (id: number) =>
    api.post<Event>(`/events/${id}`),

  join: (eventId: number) =>
    api.post(`/events/${eventId}/join`),

  leave: (eventId: number) =>
    api.post(`/events/${eventId}/leave`),
};

// ── System ───────────────────────────────────────────────────────
export const systemService = {
  health: () => api.post<{ status: string }>('/system/health'),
};
