/**
 * WebSocket 实时通信类型定义
 * 严格对应后端 API 规范 (api_spec.mdc)
 */

import type { UserInfo, GroupSettings, MessageType, ReactionType, MessageMedia, JoinedGroupItem, MessageReplyTo } from './api'

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

// 订阅项定义（批量订阅同类型数据源）
export type SubscriptionItem =
  | {
      source_type: 'group_chat'
      source_ids: string[] // group_chat_id（UUID string）
    }
  | {
      source_type: 'user'
      source_ids: number[] // user_id（safe integer, <= Number.MAX_SAFE_INTEGER）
    }
  | {
      source_type: 'notification'
      source_ids: string[] // notification_type（string）
    }

// ============================================================================
// 客户端发送事件 (Client -> Server)
// ============================================================================

export type ClientEventType =
  | 'Auth'
  | 'Ping'
  | 'Subscribe'
  | 'Unsubscribe'
  | 'SetSubscriptions'
  | 'PrivateTyping'
  | 'GroupTyping'

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

export interface PrivateTypingEvent {
  type: 'PrivateTyping'
  payload: {
    target_user_id: number
    is_typing: boolean
  }
}

export interface GroupTypingEvent {
  type: 'GroupTyping'
  payload: {
    group_chat_id: string
    is_typing: boolean
  }
}

export type ClientEvent =
  | AuthEvent
  | PingEvent
  | SubscribeEvent
  | UnsubscribeEvent
  | SetSubscriptionsEvent
  | PrivateTypingEvent
  | GroupTypingEvent

// ============================================================================
// 服务端推送事件 (Server -> Client)
// ============================================================================

export type ServerEventType =
  // 连接管理
  | 'ConnectionEstablished'
  | 'Pong'
  // 用户状态
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  | 'UserProfileUpdated'
  // 关注关系
  | 'NewFollower'
  | 'MutualFollow'
  | 'Unfollowed'
  // 生活瞬间
  | 'MomentLiked'
  | 'MomentCommented'
  | 'MomentMentioned' // 新增：瞬间被@提及
  | 'CommentMentioned' // 新增：评论被@提及
  | 'CommentReply'
  | 'SystemAnnouncement'
  // 私聊消息
  | 'PrivateMessageNew'
  | 'PrivateMessageEdited'
  | 'PrivateMessageRecalled'
  | 'PrivateMessageDeleted'
  | 'PrivateMessageReaction'
  | 'PrivateMessageRead'
  | 'PrivateUserTyping'
  // 群聊消息
  | 'GroupMessageNew'
  | 'GroupMessageEdited'
  | 'GroupMessageRecalled'
  | 'GroupMessageDeleted'
  | 'GroupMessageReaction'
  | 'GroupMessageRead'
  | 'GroupUserTyping'
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
  | 'GroupAnnouncementCreated' // 新增：群公告创建
  | 'GroupAnnouncementUpdated' // 新增：群公告更新
  | 'GroupAnnouncementDeleted' // 新增：群公告删除
  // 多端同步 (Self-Sync)
  | 'SelfJoinedGroup'
  | 'SelfLeftGroup'
  | 'PrivateChatDeleted'
  | 'PrivateChatSettingsUpdated'
  | 'NotificationReadStateUpdated'
  | 'SelfProfileUpdated'
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

export interface PongEvent {
  type: 'Pong'
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
    location: {
      latitude: number
      longitude: number
      location_name: string
    }
    updated_at: string
  }
}

export interface UserProfileUpdatedEvent {
  type: 'UserProfileUpdated'
  payload: {
    user_id: number
    nickname?: string
    avatar_url?: string
    bio?: string
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
export interface MomentSummary {
  moment_id: string
  cover_url?: string // 封面图 (第一张图片)
  text_preview: string // 文本内容摘要 (前50字)
}

export interface MomentLikedEvent {
  type: 'MomentLiked'
  payload: {
    moment_id: string
    moment_summary: MomentSummary
    user: UserInfo
    liked_at: string
  }
}

export interface MomentCommentedEvent {
  type: 'MomentCommented'
  payload: {
    moment_id: string
    moment_summary: MomentSummary
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
    comment_id: string // 原评论ID（被回复的评论）
    reply_to_content: string // 原评论内容摘要
    reply_id: string // 回复评论ID
    user: UserInfo // 回复者
    content: string
    replied_at: string
  }
}

// @ 提及事件
export interface MomentMentionedEvent {
  type: 'MomentMentioned'
  payload: {
    moment_id: string
    moment_summary: MomentSummary
    user: UserInfo // 提及我的人
    mentioned_at: string
  }
}

export interface CommentMentionedEvent {
  type: 'CommentMentioned'
  payload: {
    moment_id: string
    comment_id: string
    user: UserInfo // 提及我的人
    content: string // 评论内容摘要
    mentioned_at: string
  }
}

export interface SystemAnnouncementEvent {
  type: 'SystemAnnouncement'
  payload: {
    announcement_id: string
    title: string
    content: string
    severity: 'info' | 'warning' | 'error'
    action_url?: string
    created_at: string
  }
}

// 私聊消息事件
export interface PrivateMessageNewEvent {
  type: 'PrivateMessageNew'
  payload: {
    message_id: string
    target_user_id: number // 接收者ID (对于发送者多端同步=接收方ID)
    sender: UserInfo
    type: MessageType
    content?: string // 文本内容
    reply_to_message_id?: string
    reply_to?: MessageReplyTo
    // 媒体信息
    media?: MessageMedia
    // 位置消息信息
    location?: {
      latitude: number
      longitude: number
      location_name: string
    }
    location_address?: string
    sent_at: string
  }
}

export interface PrivateMessageEditedEvent {
  type: 'PrivateMessageEdited'
  payload: {
    message_id: string
    target_user_id: number
    content: string
    edited_at: string
  }
}

export interface PrivateMessageRecalledEvent {
  type: 'PrivateMessageRecalled'
  payload: {
    message_id: string
    target_user_id: number
    recalled_by: number
    recalled_at: string
  }
}

export interface PrivateMessageDeletedEvent {
  type: 'PrivateMessageDeleted'
  payload: {
    message_id: string
    target_user_id: number
    deleted_by: number
    deleted_at: string
  }
}

export interface PrivateMessageReactionEvent {
  type: 'PrivateMessageReaction'
  payload: {
    message_id: string
    target_user_id: number
    user: UserInfo
    reaction_type: ReactionType
    has_reaction: boolean
    created_at: string
  }
}

export interface PrivateMessageReadEvent {
  type: 'PrivateMessageRead'
  payload: {
    target_user_id: number
    up_to_message_id: string
    reader_id: number
    read_at: string
  }
}

// 群聊消息事件
export interface GroupMessageNewEvent {
  type: 'GroupMessageNew'
  payload: {
    message_id: string
    group_chat_id: string
    sender: UserInfo
    type: MessageType
    content?: string
    reply_to_message_id?: string
    reply_to?: MessageReplyTo
    mentions?: number[] // user_id
    // 媒体信息
    media?: MessageMedia
    // 位置消息信息
    location?: {
      latitude: number
      longitude: number
      location_name: string
    }
    location_address?: string
    sent_at: string
  }
}

export interface GroupMessageEditedEvent {
  type: 'GroupMessageEdited'
  payload: {
    message_id: string
    group_chat_id: string
    content: string
    edited_at: string
  }
}

export interface GroupMessageRecalledEvent {
  type: 'GroupMessageRecalled'
  payload: {
    message_id: string
    group_chat_id: string
    recalled_by: number
    recalled_at: string
  }
}

export interface GroupMessageDeletedEvent {
  type: 'GroupMessageDeleted'
  payload: {
    message_id: string
    group_chat_id: string
    deleted_by: number
    for_everyone: boolean
    deleted_at: string
  }
}

export interface GroupMessageReactionEvent {
  type: 'GroupMessageReaction'
  payload: {
    message_id: string
    group_chat_id: string
    user: UserInfo
    reaction_type: ReactionType
    has_reaction: boolean
    created_at: string
  }
}

export interface GroupMessageReadEvent {
  type: 'GroupMessageRead'
  payload: {
    group_chat_id: string
    up_to_message_id: string
    reader_id: number
    read_at: string
  }
}

export interface PrivateUserTypingEvent {
  type: 'PrivateUserTyping'
  payload: {
    target_user_id: number
    user_id: number
    is_typing: boolean
  }
}

export interface GroupUserTypingEvent {
  type: 'GroupUserTyping'
  payload: {
    group_chat_id: string
    user_id: number
    is_typing: boolean
  }
}

// 群组事件
export interface GroupMemberJoinedEvent {
  type: 'GroupMemberJoined'
  payload: {
    group_chat_id: string
    user: UserInfo
    role: 'owner' | 'admin' | 'member'
    member_count: number // 当前成员总数
    joined_at: string
  }
}

export interface GroupMemberLeftEvent {
  type: 'GroupMemberLeft'
  payload: {
    group_chat_id: string
    user_id: number
    member_count: number // 当前成员总数
    reason: 'left' | 'kicked' | 'banned' // 离开原因：主动退出、被移出、被封禁
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
    invite_id: string
    group_chat_id: string
    group_name: string
    group_avatar_url?: string
    inviter: UserInfo
    message?: string
    created_at: string
  }
}

export interface GroupInviteResponseEvent {
  type: 'GroupInviteResponse'
  payload: {
    invite_id: string
    group_chat_id: string
    user: UserInfo
    status: 'accepted' | 'rejected'
    responded_at: string
  }
}

export interface GroupApplicationReceivedEvent {
  type: 'GroupApplicationReceived'
  payload: {
    application_id: string
    group_chat_id: string
    applicant: UserInfo
    message?: string
    created_at: string
  }
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: {
    application_id: string
    group_chat_id: string
    group_name: string
    status: 'accepted' | 'rejected'
    responded_at: string
  }
}

// 群公告事件
export interface GroupAnnouncementCreatedEvent {
  type: 'GroupAnnouncementCreated'
  payload: {
    announcement_id: string
    group_chat_id: string
    title: string
    content: string
    created_by: UserInfo
    is_pinned: boolean
    created_at: string
  }
}

export interface GroupAnnouncementUpdatedEvent {
  type: 'GroupAnnouncementUpdated'
  payload: {
    announcement_id: string
    group_chat_id: string
    title?: string
    content?: string
    is_pinned?: boolean
    updated_at: string
  }
}

export interface GroupAnnouncementDeletedEvent {
  type: 'GroupAnnouncementDeleted'
  payload: {
    announcement_id: string
    group_chat_id: string
    deleted_at: string
  }
}

// 多端同步事件 (Self-Sync)
export interface SelfJoinedGroupEvent {
  type: 'SelfJoinedGroup'
  payload: {
    group: JoinedGroupItem // 包含 my_role 和 location，确保 Zero-Lookup
    joined_at: string
  }
}

export interface SelfLeftGroupEvent {
  type: 'SelfLeftGroup'
  payload: {
    group_chat_id: string
    left_at: string
  }
}

export interface PrivateChatDeletedEvent {
  type: 'PrivateChatDeleted'
  payload: {
    target_user_id: number
    deleted_at: string
  }
}

export interface PrivateChatSettingsUpdatedEvent {
  type: 'PrivateChatSettingsUpdated'
  payload: {
    target_user_id: number
    is_muted?: boolean
    is_pinned?: boolean
    updated_at: string
  }
}

export interface NotificationReadStateUpdatedEvent {
  type: 'NotificationReadStateUpdated'
  payload: {
    total_unread_count: number
    read_at: string
  }
}

export interface SelfProfileUpdatedEvent {
  type: 'SelfProfileUpdated'
  payload: {
    nickname?: string
    avatar_url?: string
    bio?: string
    updated_at: string
  }
}

// 订阅确认事件
export type SubscriptionFailureReason =
  | 'invalid_item' // 订阅项格式不合法（source_type/source_ids 类型不匹配、空数组等）
  | 'not_found' // 订阅目标不存在（部分 id 不存在）
  | 'permission_denied' // 无权限订阅该目标
  | 'too_many_subscriptions' // 超出订阅数量限制
  | 'rate_limited' // 触发限流（服务端拒绝本次操作，可稍后重试）
  | 'internal_error' // 服务端内部错误（可重试）

export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: {
    operation: 'subscribe' | 'unsubscribe' | 'set'
    items: SubscriptionItem[]
    code: number // 0表示成功，非0表示失败
    message?: string // 错误消息（失败时）
    failed_items?: (SubscriptionItem & { reason: SubscriptionFailureReason })[]
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
  | PongEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | UserLocationUpdateEvent
  | UserProfileUpdatedEvent
  | NewFollowerEvent
  | MutualFollowEvent
  | UnfollowedEvent
  | MomentLikedEvent
  | MomentCommentedEvent
  | MomentMentionedEvent
  | CommentMentionedEvent
  | CommentReplyEvent
  | SystemAnnouncementEvent
  | PrivateMessageNewEvent
  | PrivateMessageEditedEvent
  | PrivateMessageRecalledEvent
  | PrivateMessageDeletedEvent
  | PrivateMessageReactionEvent
  | PrivateMessageReadEvent
  | GroupMessageNewEvent
  | GroupMessageEditedEvent
  | GroupMessageRecalledEvent
  | GroupMessageDeletedEvent
  | GroupMessageReactionEvent
  | GroupMessageReadEvent
  | PrivateUserTypingEvent
  | GroupUserTypingEvent
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
  | GroupAnnouncementCreatedEvent
  | GroupAnnouncementUpdatedEvent
  | GroupAnnouncementDeletedEvent
  | SelfJoinedGroupEvent
  | SelfLeftGroupEvent
  | PrivateChatDeletedEvent
  | PrivateChatSettingsUpdatedEvent
  | NotificationReadStateUpdatedEvent
  | SelfProfileUpdatedEvent
  | SubscribeAckEvent
  | ErrorEvent

// 按事件 type 提取 payload 的辅助类型（给 websocket.service / useWebSocket 等使用）
export type ServerPayload<K extends ServerEventType> = Extract<ServerEvent, { type: K }>['payload']
