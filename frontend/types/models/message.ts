/**
 * 消息业务类型定义
 * 基于后端API响应的消息信息结构
 */

// 群组消息（来自后端API）
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

// 私聊消息（来自后端API）
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

// 会话项（来自后端API）
export interface ConversationItem {
  conversation_id: string
  peer_id: number
  peer_name: string
  last_message_preview: string
  last_message_time: string
  unread_count: number
}

// 私聊消息
export interface DirectMessage {
  message_id: string
  conversation_id: string
  sender_id: number
  sender_name: string
  sender_avatar_url?: string
  recipient_id: number
  recipient_name: string
  recipient_avatar_url?: string
  content: string
  message_type: string
  sent_at: string
  read_at?: string
  latitude?: number
  longitude?: number
}

// 会话
export interface Conversation {
  conversation_id: string
  type: 'direct' | 'group' | 'system'
  peer_id?: number
  group_id?: string
  name: string
  display_name?: string
  avatar_url?: string
  last_message_content?: string
  last_message_sender_name?: string
  last_message_time?: string
  unread_count: number
  is_online?: boolean
  is_pinned?: boolean
  last_message_type?: string
  last_message_id?: string
}
