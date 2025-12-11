/**
 * WebSocket 实时通信类型定义
 */

// ============================================================================
// 通用类型（复用 api.ts 中的定义）
// ============================================================================

import type { UserInfo, GroupSettings, MessageType, ReactionType } from './api'

// ============================================================================
// 基础 WebSocket 事件格式
// ============================================================================

export interface WebSocketEvent<T = unknown> {
  type: string
  payload: T
}

// ============================================================================
// 客户端发送事件类型
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
// 服务端推送事件类型
// ============================================================================

export type ServerEventType =
  | 'ConnectionEstablished'
  | 'Heartbeat'
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  | 'NewFollower'           // 有人关注我
  | 'MutualFollow'          // 互相关注（成为好友）
  | 'Unfollowed'            // 取消关注
  | 'MomentLiked'
  | 'MomentCommented'
  | 'CommentReply'          // 评论被回复
  | 'SystemAnnouncement'    // 系统公告
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
  | 'GroupMemberUnmuted'    // 取消禁言
  | 'GroupDismissed'
  | 'GroupInviteReceived'
  | 'GroupInviteResponse'
  | 'GroupApplicationReceived'
  | 'GroupApplicationResponse'
  | 'SubscribeAck'
  | 'Error'

// ============================================================================
// 订阅管理类型
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

// 在线状态事件
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

// 关注关系事件（好友 = 互相关注）
export interface NewFollowerEvent {
  type: 'NewFollower'
  payload: {
    user: UserInfo
    followed_at: string
    is_mutual: boolean  // 是否互相关注（成为好友）
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
    was_mutual: boolean  // 之前是否互相关注
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
    comment_id: string        // 原评论ID（被回复的评论）
    reply_id: string          // 回复评论ID
    user: UserInfo            // 回复者
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
    content?: string       // 文本内容（仅 text/system 类型有值）
    reply_to?: string
    mentions?: number[]
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
    id: string                // 消息ID
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    user: UserInfo            // 操作者
    reaction_type: ReactionType
    has_reaction: boolean     // 操作后状态：true=添加了反应，false=移除了反应
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

export interface GroupProfileUpdatedEvent {
  type: 'GroupProfileUpdated'
  payload: {
    group_chat_id: string
    updated_fields: {
      name?: string
      description?: string
      avatar_url?: string
      settings?: Partial<GroupSettings>
    }
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
