/**
 * 客户端扩展类型定义
 * 包含前端特有的类型，不直接对应 API 契约
 */

import type { UserInfo, MomentItem, GroupItem, MessageType, PrivateMessageItem, GroupMessageItem } from './api';

// ============================================================================
// 认证相关
// ============================================================================

export interface StoredAccount {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  access_token: string;
  access_token_expires_at: string;
  refresh_token?: string; // App only
  refresh_token_expires_at: string;
  last_active_at: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  isLoading: boolean;
  currentAccount: StoredAccount | null;
  accounts: StoredAccount[];
}

// ============================================================================
// 位置相关
// ============================================================================

export interface LocationState {
  latitude: number | null;
  longitude: number | null;
  location_name: string | null;
  isLoading: boolean;
  error: string | null;
  lastUpdated: string | null;
}

// ============================================================================
// 设置相关
// ============================================================================

export type ThemeMode = 'light' | 'dark' | 'system';
export type Locale = 'en' | 'zh';

export interface SettingsState {
  theme: ThemeMode;
  locale: Locale;
}

// ============================================================================
// 反馈相关
// ============================================================================

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastMessage {
  id: string;
  type: ToastType;
  title: string;
  description?: string;
  duration?: number;
}

export interface FeedbackState {
  toasts: ToastMessage[];
}

// ============================================================================
// 聊天相关
// ============================================================================

export type MessageStatus = 'sending' | 'sent' | 'delivered' | 'read' | 'failed';

export interface LocalMessage {
  local_id: string;
  message_id?: string;
  type: MessageType;
  content?: string;
  file?: File;
  status: MessageStatus;
  sent_at: string;
  error?: string;
}

export interface ChatState {
  // 私聊消息（按 target_user_id 索引）
  privateMessages: Record<number, PrivateMessageItem[]>;
  // 群聊消息（按 group_chat_id 索引）
  groupMessages: Record<string, GroupMessageItem[]>;
  // 本地发送中的消息
  pendingMessages: LocalMessage[];
  // 正在输入状态
  typingUsers: Record<string, number[]>; // chatId -> user_ids
}

// ============================================================================
// 卡片渲染数据
// ============================================================================

export interface UserCardData extends UserInfo {
  bio?: string;
  tags?: string[];
  distance?: number;
  is_online?: boolean;
  is_following?: boolean;
  is_mutual?: boolean;
}

export interface MomentCardData extends MomentItem {
  distance_text?: string;
}

export interface GroupCardData extends GroupItem {
  distance_text?: string;
  is_joined?: boolean;
}

// ============================================================================
// 搜索相关
// ============================================================================

export type SearchTab = 'all' | 'users' | 'groups' | 'moments';

export interface SearchFilters {
  tab: SearchTab;
  keyword: string;
  radius?: number;
  tags?: string[];
}

export interface SearchResults {
  users: UserCardData[];
  groups: GroupCardData[];
  moments: MomentCardData[];
  isLoading: boolean;
  hasMore: boolean;
}

// ============================================================================
// 表单相关
// ============================================================================

export interface FormFieldError {
  field: string;
  message: string;
}

export interface FormState<T> {
  data: T;
  errors: FormFieldError[];
  isSubmitting: boolean;
  isValid: boolean;
}

// ============================================================================
// 分页相关
// ============================================================================

export interface PaginationState {
  cursor?: string;
  hasMore: boolean;
  isLoading: boolean;
  isLoadingMore: boolean;
}

// ============================================================================
// 模态框相关
// ============================================================================

export type ModalType =
  | 'confirm'
  | 'report'
  | 'create-moment'
  | 'create-group'
  | 'edit-profile'
  | 'image-preview'
  | 'location-picker';

export interface ModalState {
  type: ModalType | null;
  props?: Record<string, unknown>;
  isOpen: boolean;
}

// ============================================================================
// 路由相关
// ============================================================================

export interface RouteParams {
  id?: string;
  userId?: string;
  groupId?: string;
  momentId?: string;
}

// ============================================================================
// WebSocket 相关
// ============================================================================

export interface WebSocketConnectionState {
  status: 'disconnected' | 'connecting' | 'connected' | 'error';
  connectionId?: string;
  lastPingAt?: string;
  lastPongAt?: string;
  reconnectAttempts: number;
  error?: string;
}
