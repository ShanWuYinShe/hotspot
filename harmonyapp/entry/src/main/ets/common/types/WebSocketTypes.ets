/**
 * WebSocket 实时通信类型定义
 * 严格遵循 api_spec.mdc WebSocket 实时事件定义
 */

// ============================================================================
// 通用类型
// ============================================================================

/**
 * 用户基本信息
 */
export interface UserInfo {
  user_id: number
  nickname: string
  avatar_url?: string
  user_type: 'temp' | 'normal' | 'admin' | 'system'
}

/**
 * 反应类型
 */
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow'

/**
 * 消息类型（接收时使用，包含系统消息）
 */
export type ReceiveMessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'

/**
 * 群组设置（部分）
 */
export interface GroupSettings {
  join_method?: 'open' | 'approval' | 'invite_only'
  is_searchable?: boolean
  max_members?: number
  password_hash?: string
  longitude?: number
  latitude?: number
  location_name?: string
}

/**
 * 群组更新字段
 */
export interface GroupUpdatedFields {
  name?: string
  description?: string
  avatar_url?: string
  settings?: Partial<GroupSettings>
}

// ============================================================================
// 订阅管理类型
// ============================================================================

/**
 * 订阅项定义
 * 注意: source_ids 类型根据 source_type 不同而不同
 *   - group_chat: string[] (群聊ID)
 *   - user: number[] (用户ID)
 *   - notification: string[] (通知类型)
 */
export interface SubscriptionItem {
  source_type: 'group_chat' | 'user' | 'notification'
  source_ids: string[] | number[]
}

/**
 * 失败的订阅项
 */
export interface FailedSubscriptionItem {
  source_type: 'group_chat' | 'user' | 'notification'
  source_ids: string[] | number[]
  reason: string
}

// ============================================================================
// WebSocket 事件类型枚举
// ============================================================================

/**
 * 客户端发送事件类型
 */
export type ClientEventType =
  | 'Ping'
  | 'Pong'
  | 'Subscribe'
  | 'Unsubscribe'
  | 'SetSubscriptions'
  | 'MarkAsRead'
  | 'Typing'

/**
 * 服务端推送事件类型
 */
export type ServerEventType =
  | 'ConnectionEstablished'
  | 'Heartbeat'
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  | 'NewFollower'
  | 'MutualFollow'
  | 'Unfollowed'
  | 'MomentLiked'
  | 'MomentCommented'
  | 'CommentReply'
  | 'SystemAnnouncement'
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
  | 'GroupMemberUnmuted'
  | 'GroupDismissed'
  | 'GroupInviteReceived'
  | 'GroupInviteResponse'
  | 'GroupApplicationReceived'
  | 'GroupApplicationResponse'
  | 'SubscribeAck'
  | 'Error'

// ============================================================================
// 客户端发送事件定义
// ============================================================================

/**
 * 心跳事件
 */
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

/**
 * 订阅事件
 */
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

/**
 * 标记已读事件
 */
export interface MarkAsReadEvent {
  type: 'MarkAsRead'
  payload: {
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    up_to: string
  }
}

/**
 * 正在输入事件
 */
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

// ----- 连接管理事件 -----

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

// ----- 在线状态事件 -----

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

// ----- 关注关系事件 -----

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

// ----- 生活瞬间互动事件 -----

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

// ----- 聊天消息事件 -----

export interface ChatMessageNewEvent {
  type: 'ChatMessageNew'
  payload: {
    id: string
    chat_type: 'private_chat' | 'group_chat'
    chat_id: string
    sender: UserInfo
    message_type: ReceiveMessageType
    content?: string
    reply_to?: string
    mentions?: number[]
    // 媒体消息字段
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

// ----- 群组事件 -----

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

// ----- 系统事件 -----

export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: {
    operation: 'subscribe' | 'unsubscribe' | 'set'
    items: SubscriptionItem[]
    code: number
    message?: string
    failed_items?: FailedSubscriptionItem[]
  }
}

export interface ErrorEvent {
  type: 'Error'
  payload: {
    code: number
    message: string
    details?: Record<string, Object>
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

/**
 * WebSocket 连接状态
 */
export type WebSocketState = 'connecting' | 'connected' | 'disconnected' | 'error'

/**
 * 重连策略配置
 */
export interface ReconnectConfig {
  baseDelay: number      // 基础延迟（毫秒），默认 1000
  maxDelay: number       // 最大延迟（毫秒），默认 30000
  maxAttempts: number    // 最大重连次数，默认 10
  jitterFactor: number   // 随机抖动因子，默认 0.2
}

// ============================================================================
// 辅助函数
// ============================================================================

/**
 * 创建 Ping 事件
 */
export function createPingEvent(): PingEvent {
  return {
    type: 'Ping',
    payload: {
      timestamp: new Date().toISOString()
    }
  }
}

/**
 * 创建订阅事件
 */
export function createSubscribeEvent(items: SubscriptionItem[]): SubscribeEvent {
  return {
    type: 'Subscribe',
    payload: { items }
  }
}

/**
 * 创建取消订阅事件
 */
export function createUnsubscribeEvent(items: SubscriptionItem[]): UnsubscribeEvent {
  return {
    type: 'Unsubscribe',
    payload: { items }
  }
}

/**
 * 创建替换式订阅事件
 */
export function createSetSubscriptionsEvent(items: SubscriptionItem[]): SetSubscriptionsEvent {
  return {
    type: 'SetSubscriptions',
    payload: { items }
  }
}

/**
 * 创建标记已读事件
 */
export function createMarkAsReadEvent(
  chat_type: 'private_chat' | 'group_chat',
  chat_id: string,
  up_to: string
): MarkAsReadEvent {
  return {
    type: 'MarkAsRead',
    payload: { chat_type, chat_id, up_to }
  }
}

/**
 * 创建正在输入事件
 */
export function createTypingEvent(
  chat_type: 'private_chat' | 'group_chat',
  chat_id: string,
  is_typing: boolean
): TypingEvent {
  return {
    type: 'Typing',
    payload: { chat_type, chat_id, is_typing }
  }
}

/**
 * 解析服务端事件
 */
export function parseServerEvent(data: string): ServerEvent | null {
  try {
    const event = JSON.parse(data) as ServerEvent
    return event
  } catch (e) {
    console.error('Failed to parse WebSocket message:', e)
    return null
  }
}

/**
 * 序列化客户端事件
 */
export function serializeClientEvent(event: ClientEvent): string {
  return JSON.stringify(event)
}
