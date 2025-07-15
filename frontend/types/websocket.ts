/**
 * WebSocket消息类型定义
 * 与后端 ws/message_types.rs 严格对应
 */

// ==================== 客户端到服务器的消息 ====================

export interface ClientMessage {
  type: 'Connect' | 'Ping' | 'JoinGroup' | 'LeaveGroup' | 'JoinConversation' | 'LeaveConversation'
  data?: {
    token?: string
    group_id?: string
    conversation_id?: string
  }
}

export interface ConnectMessage extends ClientMessage {
  type: 'Connect'
  data: {
    token: string
  }
}

export interface PingMessage extends ClientMessage {
  type: 'Ping'
}

export interface JoinGroupMessage extends ClientMessage {
  type: 'JoinGroup'
  data: {
    group_id: string
  }
}

export interface LeaveGroupMessage extends ClientMessage {
  type: 'LeaveGroup'
  data: {
    group_id: string
  }
}

export interface JoinConversationMessage extends ClientMessage {
  type: 'JoinConversation'
  data: {
    conversation_id: string
  }
}

export interface LeaveConversationMessage extends ClientMessage {
  type: 'LeaveConversation'
  data: {
    conversation_id: string
  }
}

// ==================== 服务器到客户端的消息 ====================

export interface ServerMessage {
  type: 'Pong' | 'Connected' | 'Error' | 'Success' | 'NewMessage' | 'UserStartedTyping' | 'UserStoppedTyping' | 'MessageRead' | 'GroupEvent' | 'SystemNotification' | 'MessageDeleted'
  data?: any
}

export interface PongMessage extends ServerMessage {
  type: 'Pong'
}

export interface ConnectedMessage extends ServerMessage {
  type: 'Connected'
  data: {
    user_id: number
    session_id: string
  }
}

export interface ErrorMessage extends ServerMessage {
  type: 'Error'
  data: {
    code: string
    message: string
  }
}

export interface SuccessMessage extends ServerMessage {
  type: 'Success'
  data: {
    message: string
  }
}

export interface NewMessageMessage extends ServerMessage {
  type: 'NewMessage'
  data: MessageNotification
}

export interface UserStartedTypingMessage extends ServerMessage {
  type: 'UserStartedTyping'
  data: {
    room_id: string
    user_id: number
    nickname: string
  }
}

export interface UserStoppedTypingMessage extends ServerMessage {
  type: 'UserStoppedTyping'
  data: {
    room_id: string
    user_id: number
  }
}

export interface MessageReadMessage extends ServerMessage {
  type: 'MessageRead'
  data: {
    room_id: string
    user_id: number
    message_id?: string
  }
}

export interface GroupEventMessage extends ServerMessage {
  type: 'GroupEvent'
  data: GroupEventData
}

export interface SystemNotificationMessage extends ServerMessage {
  type: 'SystemNotification'
  data: SystemNotificationData
}

export interface MessageDeletedMessage extends ServerMessage {
  type: 'MessageDeleted'
  data: {
    message_id: string
    room_id: string
    room_type: RoomType
  }
}

// ==================== 数据结构定义 ====================

export interface MessageNotification {
  message_id: string
  room_id: string
  room_type: RoomType
  sender_id: number
  sender_nickname: string
  content: string
  message_type: MessageType
  created_at: string
  latitude?: number
  longitude?: number
}

export interface GroupEventData {
  group_id: string
  event_type: GroupEventType
  actor_id: number
  actor_nickname: string
  target_id?: number
  target_nickname?: string
  created_at: string
}

export interface SystemNotificationData {
  notification_id: string
  notification_type: SystemNotificationType
  title: string
  content: string
  data?: Record<string, any>
  created_at: string
}

// ==================== 枚举类型 ====================

export enum RoomType {
  Group = 'group',
  Conversation = 'conversation'
}

export enum MessageType {
  Text = 'text',
  Image = 'image',
  Voice = 'voice',
  Video = 'video',
  Location = 'location',
  File = 'file'
}

export enum GroupEventType {
  MemberJoined = 'member_joined',
  MemberLeft = 'member_left',
  MemberKicked = 'member_kicked',
  OwnerTransferred = 'owner_transferred',
  GroupUpdated = 'group_updated',
  GroupDeleted = 'group_deleted'
}

export enum SystemNotificationType {
  FriendRequest = 'friend_request',
  GroupInvitation = 'group_invitation',
  System = 'system',
  Announcement = 'announcement',
  NewCheckIn = 'new_check_in',
  GroupDigest = 'group_digest'
}

// ==================== 房间ID类型 ====================

export type RoomId = 
  | { type: 'group'; id: string }
  | { type: 'conversation'; id: string }
  | { type: 'user'; id: number }

// ==================== 联合类型 ====================

export type WebSocketMessage = 
  | PongMessage
  | ConnectedMessage
  | ErrorMessage
  | SuccessMessage
  | NewMessageMessage
  | UserStartedTypingMessage
  | UserStoppedTypingMessage
  | MessageReadMessage
  | GroupEventMessage
  | SystemNotificationMessage
  | MessageDeletedMessage

export type WebSocketClientMessage =
  | ConnectMessage
  | PingMessage
  | JoinGroupMessage
  | LeaveGroupMessage
  | JoinConversationMessage
  | LeaveConversationMessage

// ==================== 配置和状态 ====================

export enum WebSocketConnectionState {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  RECONNECTING = 'reconnecting',
  FAILED = 'failed'
}

export interface WebSocketConfig {
  url: string
  reconnectInterval: number
  maxReconnectAttempts: number
  heartbeatInterval: number
  enableCompression: boolean
  enableLogging: boolean
}

export interface WebSocketStats {
  connectionTime: number
  reconnectCount: number
  messagesReceived: number
  messagesSent: number
  lastHeartbeat: string
  averageLatency: number
}
export type MomentNotification = SystemNotificationData
export type GroupEventNotification = GroupEventData
export type FriendRequestNotification = SystemNotificationData
export type ReadReceiptNotification = SystemNotificationData

export type UserTypingNotification = SystemNotificationData
