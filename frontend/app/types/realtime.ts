/**
 * WebSocket 实时消息类型定义
 * 仅包含前端实际用到的字段，保持与后端事件语义一致
 */

export type RoomType = 'group' | 'conversation'

// 服务器 -> 客户端 载荷
export interface WsConnectedData {
  user_id?: string
  session_id?: string
}

export interface WsNewMessagePayload {
  message_id?: string
  room_id?: string
  room_type?: RoomType
  sender_id: string
  sender_nickname: string
  content: string
  message_type?: string
  created_at?: string
}

export interface WsMessageStatusUpdatePayload {
  message_id: string
  status: 'sent' | 'delivered' | 'read'
  timestamp: string
}

export interface WsUserStatusUpdatePayload {
  user_id: string
  status: 'online' | 'offline'
  last_seen?: string
}

export interface WsErrorPayload {
  code: string | number
  message?: string
  msg?: string
  details?: any
}

export interface WsGroupEventPayload {
  group_id: string
  event_type: string
  actor_nickname?: string
  details?: any
}

export interface WsSystemNotificationPayload {
  title?: string
  content?: string
}

export interface WsMessageDeletedPayload {
  room_id?: string
  room_type?: RoomType
  message_id?: string
}

export interface WsMessageReadPayload {
  room_id?: string
  room_type?: RoomType
  message_id?: string
}

export interface WsUserTypingPayload {
  room_id?: string
  room_type?: RoomType
  user_id: string
}

// 服务器 -> 客户端 消息
export type ServerMessage =
  | { type: 'Connected'; data?: WsConnectedData }
  | { type: 'Pong'; data?: undefined }
  | { type: 'NewMessage'; data: WsNewMessagePayload }
  | { type: 'MessageStatusUpdate'; data: WsMessageStatusUpdatePayload }
  | { type: 'UserStatusUpdate'; data: WsUserStatusUpdatePayload }
  | { type: 'Error'; data?: WsErrorPayload }
  | { type: 'GroupEvent'; data?: WsGroupEventPayload }
  | { type: 'SystemNotification'; data?: WsSystemNotificationPayload }
  | { type: 'MessageDeleted'; data?: WsMessageDeletedPayload }
  | { type: 'MessageRead'; data?: WsMessageReadPayload }
  | { type: 'UserStartedTyping'; data?: WsUserTypingPayload }
  | { type: 'UserStoppedTyping'; data?: WsUserTypingPayload }

// 客户端 -> 服务器 消息
export type ClientMessage =
  | { type: 'Connect'; data: { token: string } }
  | { type: 'Ping' }
  | { type: 'JoinUserRoom' }
  | { type: 'LeaveUserRoom' }
  | { type: 'JoinGroup'; data: { group_id: string } }
  | { type: 'LeaveGroup'; data: { group_id: string } }
  | { type: 'JoinConversation'; data: { conversation_id: string } }
  | { type: 'LeaveConversation'; data: { conversation_id: string } }

// 前端统一领域模型（供 stores/UI 使用）
export interface RealtimeMessage {
  id: string
  conversation_id: string
  sender_id: string
  sender_name: string
  sender_avatar?: string
  content: string
  message_type: 'text' | 'image' | 'file' | 'location'
  created_at: string
  is_read: boolean
}

export interface MessageStatusUpdate {
  message_id: string
  status: 'sent' | 'delivered' | 'read'
  timestamp: string
}

export interface UserStatusUpdate {
  user_id: string
  status: 'online' | 'offline'
  last_seen?: string
}


