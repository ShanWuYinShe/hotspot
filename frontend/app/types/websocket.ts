/**
 * WebSocket 实时通信类型定义
 * ✅ 严格按照 @api_spec.mdc 第4章规范定义
 */

// ============================================================================
// 通用类型（复用 api.ts 中的定义）
// ============================================================================

import type { UserInfo } from './api'

// ============================================================================
// 基础 WebSocket 事件格式（与 api_spec.mdc 第4.3节一致）
// ============================================================================

export interface WebSocketEvent<T = any> {
  type: string
  payload: T
}

// ============================================================================
// 客户端发送事件类型（api_spec.mdc 第4.4节）
// ============================================================================

export type ClientEventType =
  | 'Ping'
  | 'Pong'
  | 'Subscribe'
  | 'Unsubscribe'
  | 'SetSubscriptions'
  | 'MarkAsRead'
  | 'Typing'

// ============================================================================
// 服务端推送事件类型（api_spec.mdc 第4.3节）
// ============================================================================

export type ServerEventType =
  | 'ConnectionEstablished'
  | 'Heartbeat'
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  | 'FriendRequestReceived'
  | 'FriendRequestAccepted'
  | 'FriendRemoved'
  | 'MomentLiked'
  | 'MomentCommented'
  | 'ChatMessageNew'
  | 'ChatMessageEdited'
  | 'ChatMessageRecalled'
  | 'ChatMessageDeleted'
  | 'ChatMessageReaction'
  | 'ChatMessageRead'
  | 'UserTyping'
  | 'GroupMemberJoined'
  | 'GroupMemberLeft'
  | 'GroupProfileUpdated'
  | 'GroupMemberRoleChanged'
  | 'GroupMemberMuted'
  | 'GroupDismissed'
  | 'GroupInviteReceived'
  | 'GroupInviteResponse'
  | 'GroupApplicationReceived'
  | 'GroupApplicationResponse'
  | 'InboxItem'
  | 'SubscribeAck'
  | 'Error'

// ============================================================================
// 订阅管理类型（api_spec.mdc 第4.4节）
// ============================================================================

export interface SubscriptionItem {
  source_type: 'group_chat' | 'user' | 'notification'
  source_ids: string[]
}

// ============================================================================
// 客户端发送事件定义
// ============================================================================

// 心跳事件
export interface PingEvent {
  type: 'Ping'
  payload: {
    timestamp: string
  }
}

export interface PongEvent {
  type: 'Pong'
  payload: {
    timestamp: string
  }
}

// 订阅事件
export interface SubscribeEvent {
  type: 'Subscribe'
  payload: {
    items: SubscriptionItem[]
  }
}

export interface UnsubscribeEvent {
  type: 'Unsubscribe'
  payload: {
    items: SubscriptionItem[]
  }
}

export interface SetSubscriptionsEvent {
  type: 'SetSubscriptions'
  payload: {
    items: SubscriptionItem[]
  }
}

// 实时状态同步事件
export interface MarkAsReadEvent {
  type: 'MarkAsRead'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    up_to: string
  }
}

export interface TypingEvent {
  type: 'Typing'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    is_typing: boolean
  }
}

// ============================================================================
// 服务端推送事件定义
// ============================================================================

// 连接管理事件
export interface ConnectionEstablishedEvent {
  type: 'ConnectionEstablished'
  payload: {
    connection_id: string
    user_id: string
    server_time: string
  }
}

export interface HeartbeatEvent {
  type: 'Heartbeat'
  payload: {
    timestamp: string
  }
}

// 在线状态事件
export interface UserOnlineEvent {
  type: 'UserOnline'
  payload: {
    user_id: string
    nickname: string
    avatar_url?: string
    online_at: string
  }
}

export interface UserOfflineEvent {
  type: 'UserOffline'
  payload: {
    user_id: string
    offline_at: string
  }
}

export interface UserLocationUpdateEvent {
  type: 'UserLocationUpdate'
  payload: {
    user_id: string
    latitude: number
    longitude: number
    location_name: string
    updated_at: string
  }
}

// 好友关系事件
export interface FriendRequestReceivedEvent {
  type: 'FriendRequestReceived'
  payload: {
    id: string
    from_user: UserInfo
    message?: string
    created_at: string
  }
}

export interface FriendRequestAcceptedEvent {
  type: 'FriendRequestAccepted'
  payload: {
    id: string
    user: UserInfo
    accepted_at: string
  }
}

export interface FriendRemovedEvent {
  type: 'FriendRemoved'
  payload: {
    user_id: string
    removed_at: string
  }
}

// 生活瞬间互动事件
export interface MomentLikedEvent {
  type: 'MomentLiked'
  payload: {
    moment_id: string
    user: UserInfo
    liked_at: string
  }
}

export interface MomentCommentedEvent {
  type: 'MomentCommented'
  payload: {
    moment_id: string
    comment_id: string
    user: UserInfo
    content: string
    commented_at: string
  }
}

// 聊天消息事件
export interface ChatMessageNewEvent {
  type: 'ChatMessageNew'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    sender: UserInfo
    type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
    content: string
    reply_to?: string
    mentions?: string[]
    sent_at: string
  }
}

export interface ChatMessageEditedEvent {
  type: 'ChatMessageEdited'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    content: string
    edited_at: string
  }
}

export interface ChatMessageRecalledEvent {
  type: 'ChatMessageRecalled'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    recalled_by: string
    recalled_at: string
  }
}

export interface ChatMessageDeletedEvent {
  type: 'ChatMessageDeleted'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    deleted_by: string
    for_everyone: boolean
    deleted_at: string
  }
}

export interface ChatMessageReactionEvent {
  type: 'ChatMessageReaction'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    user: UserInfo
    reaction_type: 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow'
    action: 'add' | 'remove'
    created_at: string
  }
}

export interface ChatMessageReadEvent {
  type: 'ChatMessageRead'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    up_to: string
    reader_id: string
    read_at: string
  }
}

export interface UserTypingEvent {
  type: 'UserTyping'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    user_id: string
    nickname: string
    is_typing: boolean
  }
}

// 群组事件
export interface GroupMemberJoinedEvent {
  type: 'GroupMemberJoined'
  payload: {
    group_chat_id: string
    user: UserInfo
    joined_at: string
  }
}

export interface GroupMemberLeftEvent {
  type: 'GroupMemberLeft'
  payload: {
    group_chat_id: string
    user_id: string
    left_at: string
  }
}

export interface GroupProfileUpdatedEvent {
  type: 'GroupProfileUpdated'
  payload: {
    group_chat_id: string
    updated_fields: {
      name?: string
      description?: string
      avatar_url?: string
      settings?: any
    }
    updated_at: string
  }
}

export interface GroupMemberRoleChangedEvent {
  type: 'GroupMemberRoleChanged'
  payload: {
    group_chat_id: string
    user_id: string
    old_role: 'owner' | 'admin' | 'member'
    new_role: 'owner' | 'admin' | 'member'
    changed_at: string
  }
}

export interface GroupMemberMutedEvent {
  type: 'GroupMemberMuted'
  payload: {
    group_chat_id: string
    user_id: string
    muted_until: string
    muted_at: string
  }
}

export interface GroupDismissedEvent {
  type: 'GroupDismissed'
  payload: {
    group_chat_id: string
    dismissed_by: string
    dismissed_at: string
  }
}

export interface GroupInviteReceivedEvent {
  type: 'GroupInviteReceived'
  payload: {
    id: string
    group_chat_id: string
    group_name: string
    inviter: UserInfo
    message?: string
    created_at: string
  }
}

export interface GroupInviteResponseEvent {
  type: 'GroupInviteResponse'
  payload: {
    id: string
    group_chat_id: string
    user: UserInfo
    status: 'accepted' | 'rejected'
    responded_at: string
  }
}

export interface GroupApplicationReceivedEvent {
  type: 'GroupApplicationReceived'
  payload: {
    id: string
    group_chat_id: string
    applicant: UserInfo
    message?: string
    created_at: string
  }
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: {
    id: string
    group_chat_id: string
    group_name: string
    status: 'accepted' | 'rejected'
    responded_at: string
  }
}

// 收件箱事件
export interface InboxItemEvent {
  type: 'InboxItem'
  payload: {
    id: string
    category: 'social' | 'system' | 'content'
    type: string
    title?: string
    content: string
    data?: Record<string, any>
    action_url?: string
    severity?: 'info' | 'warning' | 'error'
    created_at: string
  }
}

// 订阅确认事件
export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: {
    operation: 'subscribe' | 'unsubscribe' | 'set'
    items: SubscriptionItem[]
    code: number
    message?: string
    failed_items?: {
      source_type: 'group_chat' | 'user' | 'notification'
      source_ids: string[]
      reason: string
    }[]
  }
}

// 错误事件
export interface ErrorEvent {
  type: 'Error'
  payload: {
    code: number
    message: string
    details?: Record<string, any>
  }
}

// ============================================================================
// 联合类型
// ============================================================================

export type ClientEvent =
  | PingEvent
  | PongEvent
  | SubscribeEvent
  | UnsubscribeEvent
  | SetSubscriptionsEvent
  | MarkAsReadEvent
  | TypingEvent

export type ServerEvent =
  | ConnectionEstablishedEvent
  | HeartbeatEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | UserLocationUpdateEvent
  | FriendRequestReceivedEvent
  | FriendRequestAcceptedEvent
  | FriendRemovedEvent
  | MomentLikedEvent
  | MomentCommentedEvent
  | ChatMessageNewEvent
  | ChatMessageEditedEvent
  | ChatMessageRecalledEvent
  | ChatMessageDeletedEvent
  | ChatMessageReactionEvent
  | ChatMessageReadEvent
  | UserTypingEvent
  | GroupMemberJoinedEvent
  | GroupMemberLeftEvent
  | GroupProfileUpdatedEvent
  | GroupMemberRoleChangedEvent
  | GroupMemberMutedEvent
  | GroupDismissedEvent
  | GroupInviteReceivedEvent
  | GroupInviteResponseEvent
  | GroupApplicationReceivedEvent
  | GroupApplicationResponseEvent
  | InboxItemEvent
  | SubscribeAckEvent
  | ErrorEvent

// ============================================================================
// 辅助类型
// ============================================================================

// WebSocket 连接状态
export type WebSocketState = 'connecting' | 'connected' | 'disconnected' | 'error'

// 重连策略配置
export interface ReconnectConfig {
  baseDelay: number      // 基础延迟（毫秒），默认 1000
  maxDelay: number       // 最大延迟（毫秒），默认 30000
  maxAttempts: number    // 最大重连次数，默认 10
  jitterFactor: number   // 随机抖动因子，默认 0.2
}
