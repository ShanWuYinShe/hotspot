/**
 * 消息相关API类型定义
 * 与后端 message API 严格对应
 */

import type { PaginatedResponse, Pagination } from '~/types/api'

// ==================== 群组消息 ====================

export interface SendGroupMessageRequest {
  group_id: string
  content: string
  message_type: string // 默认为 "text"，可以是 "image" 等
  latitude: number
  longitude: number
}

export interface SendGroupMessageResponse {
  group_id: string
  message_id: string
  message_type: string
  sent_at: string
}

export interface GroupMessageHistoryRequest extends Pagination {
  group_id: string
}

export interface GroupMessageHistoryItem {
  message_id: string
  group_id: string
  sender_id: string
  sender_name: string
  content: string
  message_type: string
  sent_at: string
  latitude: number
  longitude: number
}

export type GroupMessageHistoryResponse = PaginatedResponse<GroupMessageHistoryItem>

export interface DeleteGroupMessageRequest {
  message_id: string
}

// ==================== 私聊消息 ====================

export interface SendDirectMessageRequest {
  recipient_id: string
  content: string
  message_type: string // 默认为 "text"，可以是 "image" 等
  latitude: number
  longitude: number
}

export interface SendDirectMessageResponse {
  message_id: string
  conversation_id: string
  recipient_id: string
  message_type: string
  sent_at: string
}

export interface DirectMessageHistoryRequest extends Pagination {
  conversation_id?: string
  user_id?: number
}

export interface DirectMessageHistoryItem {
  message_id: string
  conversation_id: string
  sender_id: string
  sender_name: string
  recipient_id: string
  content: string
  message_type: string
  sent_at: string
  read_at?: string
  latitude: number
  longitude: number
}

export type DirectMessageHistoryResponse = PaginatedResponse<DirectMessageHistoryItem>

export interface DeleteDirectMessageRequest {
  message_id: string
}

// ==================== 标记已读 ====================

export interface MarkDirectMessageReadRequest {
  conversation_id: string
  message_id?: string
}

export interface MarkDirectMessageReadResponse {
  success: boolean
  marked_count?: number
}

// ==================== 会话列表 ====================

export interface ConversationsRequest extends Pagination {
  // 分页参数已在 Pagination 中定义
}

export interface ConversationItem {
  conversation_id: string
  peer_id: number
  peer_name: string
  last_message_preview: string
  last_message_time: string
  unread_count: number
}

export type ConversationsResponse = PaginatedResponse<ConversationItem>

// ==================== 通用响应 ====================

export interface DeleteMessageResponse {
  success: boolean
}

export interface UnreadCountResponse {
  unread_count: number
}

export interface SendTypingIndicatorRequest {
  target_id: string
  type: 'direct' | 'group'
  is_typing: boolean
}

// ==================== 消息通知相关 ====================

// 消息通知
export interface MessageNotification {
  message_id: string
  room_id: string
  room_type: 'group' | 'conversation'
  sender_id: number
  sender_nickname: string
  content: string
  message_type: string
  created_at: string
  latitude?: number
  longitude?: number
}

// 群组事件数据
export interface GroupEventData {
  group_id: string
  event_type: GroupEventType
  actor_id: number
  actor_nickname: string
  target_id?: number
  target_nickname?: string
  created_at: string
}

// 群组事件类型
export enum GroupEventType {
  MemberJoined = 'member_joined',
  MemberLeft = 'member_left',
  MemberKicked = 'member_kicked',
  OwnerTransferred = 'owner_transferred',
  GroupUpdated = 'group_updated',
  GroupDeleted = 'group_deleted'
}

// 系统通知数据
export interface SystemNotificationData {
  notification_id: string
  notification_type: SystemNotificationType
  title: string
  content: string
  data?: any
  created_at: string
}

// 系统通知类型
export enum SystemNotificationType {
  FriendRequest = 'friend_request',
  GroupInvitation = 'group_invitation',
  System = 'system',
  Announcement = 'announcement',
  NewCheckIn = 'new_check_in',
  GroupDigest = 'group_digest'
}
