/**
 * WebSocket 实时通信类型定义
 * 严格对应后端 API 规范 (api_spec.mdc)
 */

import type { UserInfo, GroupSettings, MessageType, ReactionType } from './api'

// ============================================================================
// 基础类型
// ============================================================================

export interface WebSocketEvent<T = unknown> {
  type: string
  payload: T
}

export type WebSocketState = 'connecting' | 'connected' | 'disconnected' | 'error'

// ============================================================================
// 订阅相关
// ============================================================================

export type SubscriptionSourceType = 'group_chat' | 'user' | 'notification'

export interface SubscriptionItem {
  source_type: SubscriptionSourceType
  source_ids: string[]
}

// ============================================================================
// 客户端发送事件 (Client -> Server)
// ============================================================================

export type ClientEventType =
  | 'Auth'
  | 'Ping'
  | 'Pong'
  | 'Subscribe'
  | 'Unsubscribe'
  | 'SetSubscriptions'
  | 'MarkAsRead'
  | 'Typing'

export interface AuthEvent {
  type: 'Auth'
  payload: {
    access_token: string
    device_fingerprint?: string
  }
}

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

export type ClientEvent =
  | AuthEvent
  | PingEvent
  | PongEvent
  | SubscribeEvent
  | UnsubscribeEvent
  | SetSubscriptionsEvent
  | MarkAsReadEvent
  | TypingEvent

// ============================================================================
// 服务端推送事件 (Server -> Client)
// ============================================================================

export type ServerEventType =
  // 连接管理
  | 'ConnectionEstablished'
  | 'Heartbeat'
  // 用户状态
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  // 关注关系
  | 'NewFollower'
  | 'MutualFollow'
  | 'Unfollowed'
  // 生活瞬间
  | 'MomentLiked'
  | 'MomentCommented'
  | 'CommentReply'
  | 'SystemAnnouncement'
  // 聊天消息
  | 'ChatMessageNew'
  | 'ChatMessageEdited'
  | 'ChatMessageRecalled'
  | 'ChatMessageDeleted'
  | 'ChatMessageReaction'
  | 'ChatMessageRead'
  | 'UserTyping'
  // 群组
  | 'GroupMemberJoined'
  | 'GroupMemberLeft'
  | 'GroupProfileUpdated'
  | 'GroupMemberRoleChanged'
  | 'GroupMemberMuted'
  | 'GroupMemberUnmuted'
  | 'GroupDismissed'
  | 'GroupInviteReceived'
  | 'GroupInviteResponse'
  | 'GroupApplicationReceived'
  | 'GroupApplicationResponse'
  // 系统
  | 'Error'
  | 'SubscribeAck'

// 连接管理事件
export interface ConnectionEstablishedEvent {
  type: 'ConnectionEstablished'
  payload: {
    connection_id: string
    user_id: number
    server_time: string
  }
}

export interface HeartbeatEvent {
  type: 'Heartbeat'
  payload: {
    timestamp: string
  }
}

// 用户状态事件
export interface UserOnlineEvent {
  type: 'UserOnline'
  payload: {
    user_id: number
    nickname: string
    avatar_url?: string
    online_at: string
  }
}

export interface UserOfflineEvent {
  type: 'UserOffline'
  payload: {
    user_id: number
    offline_at: string
  }
}

export interface UserLocationUpdateEvent {
  type: 'UserLocationUpdate'
  payload: {
    user_id: number
    latitude: number
    longitude: number
    location_name: string
    updated_at: string
  }
}

// 关注关系事件
export interface NewFollowerEvent {
  type: 'NewFollower'
  payload: {
    user: UserInfo
    followed_at: string
    is_mutual: boolean
  }
}

export interface MutualFollowEvent {
  type: 'MutualFollow'
  payload: {
    user: UserInfo
    mutual_at: string
  }
}

export interface UnfollowedEvent {
  type: 'Unfollowed'
  payload: {
    user_id: number
    unfollowed_at: string
    was_mutual: boolean
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

export interface CommentReplyEvent {
  type: 'CommentReply'
  payload: {
    moment_id: string
    comment_id: string
    reply_id: string
    user: UserInfo
    content: string
    replied_at: string
  }
}

export interface SystemAnnouncementEvent {
  type: 'SystemAnnouncement'
  payload: {
    id: string
    title: string
    content: string
    severity: 'info' | 'warning' | 'error'
    action_url?: string
    created_at: string
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
    message_type: MessageType
    content?: string
    reply_to?: string
    mentions?: number[]
    // 媒体字段
    media_url?: string
    thumbnail_url?: string
    file_name?: string
    file_size?: number
    duration?: number
    width?: number
    height?: number
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
    recalled_by: number
    recalled_at: string
  }
}

export interface ChatMessageDeletedEvent {
  type: 'ChatMessageDeleted'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    deleted_by: number
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
    reaction_type: ReactionType
    has_reaction: boolean
    created_at: string
  }
}

export interface ChatMessageReadEvent {
  type: 'ChatMessageRead'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    up_to: string
    reader_id: number
    read_at: string
  }
}

export interface UserTypingEvent {
  type: 'UserTyping'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    user_id: number
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
    user_id: number
    left_at: string
  }
}

export interface GroupUpdatedFields {
  name?: string
  description?: string
  avatar_url?: string
  settings?: Partial<GroupSettings>
}

export interface GroupProfileUpdatedEvent {
  type: 'GroupProfileUpdated'
  payload: {
    group_chat_id: string
    updated_fields: GroupUpdatedFields
    updated_at: string
  }
}

export interface GroupMemberRoleChangedEvent {
  type: 'GroupMemberRoleChanged'
  payload: {
    group_chat_id: string
    user_id: number
    old_role: 'owner' | 'admin' | 'member'
    new_role: 'owner' | 'admin' | 'member'
    changed_at: string
  }
}

export interface GroupMemberMutedEvent {
  type: 'GroupMemberMuted'
  payload: {
    group_chat_id: string
    user_id: number
    muted_until: string
    muted_at: string
  }
}

export interface GroupMemberUnmutedEvent {
  type: 'GroupMemberUnmuted'
  payload: {
    group_chat_id: string
    user_id: number
    unmuted_at: string
  }
}

export interface GroupDismissedEvent {
  type: 'GroupDismissed'
  payload: {
    group_chat_id: string
    dismissed_by: number
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

// 订阅确认事件
export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: {
    operation: 'subscribe' | 'unsubscribe' | 'set'
    items: SubscriptionItem[]
    code: number
    message?: string
    failed_items?: {
      source_type: SubscriptionSourceType
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
  }
}

// 联合类型
export type ServerEvent =
  | ConnectionEstablishedEvent
  | HeartbeatEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | UserLocationUpdateEvent
  | NewFollowerEvent
  | MutualFollowEvent
  | UnfollowedEvent
  | MomentLikedEvent
  | MomentCommentedEvent
  | CommentReplyEvent
  | SystemAnnouncementEvent
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
  | GroupMemberUnmutedEvent
  | GroupDismissedEvent
  | GroupInviteReceivedEvent
  | GroupInviteResponseEvent
  | GroupApplicationReceivedEvent
  | GroupApplicationResponseEvent
  | SubscribeAckEvent
  | ErrorEvent

// 按事件 type 提取 payload 的辅助类型（给 websocket.service / useWebSocket 等使用）
export type ServerPayload<K extends ServerEventType> = Extract<ServerEvent, { type: K }>['payload']
