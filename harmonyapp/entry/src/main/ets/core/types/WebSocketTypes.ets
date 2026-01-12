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
  source_ids: string[]
}

/**
 * 失败的订阅项
 */
export interface FailedSubscriptionItem {
  source_type: 'group_chat' | 'user' | 'notification'
  source_ids: string[]
  reason: string
}

// ============================================================================
// WebSocket 事件类型枚举
// ============================================================================

/**
 * 客户端发送事件类型
 */
export type ClientEventType =
  | 'Auth'
  | 'Ping'
  | 'Subscribe'
  | 'Unsubscribe'
  | 'SetSubscriptions'
  | 'PrivateTyping'
  | 'GroupTyping'

/**
 * 服务端推送事件类型
 */
export type ServerEventType =
  | 'ConnectionEstablished'
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
  | 'PrivateMessageNew'
  | 'PrivateMessageEdited'
  | 'PrivateMessageRecalled'
  | 'PrivateMessageDeleted'
  | 'PrivateMessageReaction'
  | 'PrivateMessageRead'
  | 'GroupMessageNew'
  | 'GroupMessageEdited'
  | 'GroupMessageRecalled'
  | 'GroupMessageDeleted'
  | 'GroupMessageReaction'
  | 'GroupMessageRead'
  | 'PrivateUserTyping'
  | 'GroupUserTyping'
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
 * 鉴权事件（握手后首帧）
 */
export interface AuthPayload {
  access_token: string
  device_fingerprint?: string
}

export interface AuthEvent {
  type: 'Auth'
  payload: AuthPayload
}

/**
 * 心跳事件
 */
export interface PingPayload {
  timestamp: string
}

export interface PingEvent {
  type: 'Ping'
  payload: PingPayload
}

/**
 * 订阅事件
 */
export interface SubscribePayload {
  items: SubscriptionItem[]
}

export interface SubscribeEvent {
  type: 'Subscribe'
  payload: SubscribePayload
}

export interface UnsubscribePayload {
  items: SubscriptionItem[]
}

export interface UnsubscribeEvent {
  type: 'Unsubscribe'
  payload: UnsubscribePayload
}

export interface SetSubscriptionsPayload {
  items: SubscriptionItem[]
}

export interface SetSubscriptionsEvent {
  type: 'SetSubscriptions'
  payload: SetSubscriptionsPayload
}

/**
 * 正在输入事件
 */
export interface PrivateTypingPayload {
  target_user_id: number
  is_typing: boolean
}

export interface PrivateTypingEvent {
  type: 'PrivateTyping'
  payload: PrivateTypingPayload
}

export interface GroupTypingPayload {
  group_chat_id: string
  is_typing: boolean
}

export interface GroupTypingEvent {
  type: 'GroupTyping'
  payload: GroupTypingPayload
}

// ============================================================================
// 服务端推送事件定义
// ============================================================================

// ----- 连接管理事件 -----

export interface ConnectionEstablishedPayload {
  connection_id: string
  user_id: number
  server_time: string
}

export interface ConnectionEstablishedEvent {
  type: 'ConnectionEstablished'
  payload: ConnectionEstablishedPayload
}

export interface PongPayload {
  timestamp: string
}

export interface PongEvent {
  type: 'Pong'
  payload: PongPayload
}

// ----- 在线状态事件 -----

export interface UserOnlinePayload {
  user_id: number
  nickname: string
  avatar_url?: string
  online_at: string
}

export interface UserOnlineEvent {
  type: 'UserOnline'
  payload: UserOnlinePayload
}

export interface UserOfflinePayload {
  user_id: number
  offline_at: string
}

export interface UserOfflineEvent {
  type: 'UserOffline'
  payload: UserOfflinePayload
}

export interface UserLocationUpdatePayload {
  user_id: number
  latitude: number
  longitude: number
  location_name: string
  updated_at: string
}

export interface UserLocationUpdateEvent {
  type: 'UserLocationUpdate'
  payload: UserLocationUpdatePayload
}

// ----- 关注关系事件 -----

export interface NewFollowerPayload {
  user: UserInfo
  followed_at: string
  is_mutual: boolean
}

export interface NewFollowerEvent {
  type: 'NewFollower'
  payload: NewFollowerPayload
}

export interface MutualFollowPayload {
  user: UserInfo
  mutual_at: string
}

export interface MutualFollowEvent {
  type: 'MutualFollow'
  payload: MutualFollowPayload
}

export interface UnfollowedPayload {
  user_id: number
  unfollowed_at: string
  was_mutual: boolean
}

export interface UnfollowedEvent {
  type: 'Unfollowed'
  payload: UnfollowedPayload
}

// ----- 生活瞬间互动事件 -----

export interface MomentLikedPayload {
  moment_id: string
  user: UserInfo
  liked_at: string
}

export interface MomentLikedEvent {
  type: 'MomentLiked'
  payload: MomentLikedPayload
}

export interface MomentCommentedPayload {
  moment_id: string
  comment_id: string
  user: UserInfo
  content: string
  commented_at: string
}

export interface MomentCommentedEvent {
  type: 'MomentCommented'
  payload: MomentCommentedPayload
}

export interface CommentReplyPayload {
  moment_id: string
  comment_id: string
  reply_id: string
  user: UserInfo
  content: string
  replied_at: string
}

export interface CommentReplyEvent {
  type: 'CommentReply'
  payload: CommentReplyPayload
}

export interface SystemAnnouncementPayload {
  id: string
  title: string
  content: string
  severity: 'info' | 'warning' | 'error'
  action_url?: string
  created_at: string
}

export interface SystemAnnouncementEvent {
  type: 'SystemAnnouncement'
  payload: SystemAnnouncementPayload
}

// ----- 聊天消息事件 -----

/**
 * 消息媒体信息
 */
export interface MessageMedia {
  url: string
  thumbnail_url?: string
  file_name?: string
  file_size?: number
  duration?: number
  width?: number
  height?: number
  mime_type?: string
}

// 私聊消息事件
export interface PrivateMessageNewPayload {
  message_id: string
  target_user_id: number
  sender: UserInfo
  type: ReceiveMessageType
  content?: string
  reply_to_message_id?: string
  media?: MessageMedia
  sent_at: string
}

export interface PrivateMessageNewEvent {
  type: 'PrivateMessageNew'
  payload: PrivateMessageNewPayload
}

export interface PrivateMessageEditedPayload {
  message_id: string
  target_user_id: number
  content: string
  edited_at: string
}

export interface PrivateMessageEditedEvent {
  type: 'PrivateMessageEdited'
  payload: PrivateMessageEditedPayload
}

export interface PrivateMessageRecalledPayload {
  message_id: string
  target_user_id: number
  recalled_by: number
  recalled_at: string
}

export interface PrivateMessageRecalledEvent {
  type: 'PrivateMessageRecalled'
  payload: PrivateMessageRecalledPayload
}

export interface PrivateMessageDeletedPayload {
  message_id: string
  target_user_id: number
  deleted_by: number
  deleted_at: string
}

export interface PrivateMessageDeletedEvent {
  type: 'PrivateMessageDeleted'
  payload: PrivateMessageDeletedPayload
}

export interface PrivateMessageReactionPayload {
  message_id: string
  target_user_id: number
  user: UserInfo
  reaction_type: ReactionType
  has_reaction: boolean
  created_at: string
}

export interface PrivateMessageReactionEvent {
  type: 'PrivateMessageReaction'
  payload: PrivateMessageReactionPayload
}

export interface PrivateMessageReadPayload {
  target_user_id: number
  up_to_message_id: string
  reader_id: number
  read_at: string
}

export interface PrivateMessageReadEvent {
  type: 'PrivateMessageRead'
  payload: PrivateMessageReadPayload
}

// 群聊消息事件
export interface GroupMessageNewPayload {
  message_id: string
  group_chat_id: string
  sender: UserInfo
  type: ReceiveMessageType
  content?: string
  reply_to_message_id?: string
  mentions?: number[]
  media?: MessageMedia
  sent_at: string
}

export interface GroupMessageNewEvent {
  type: 'GroupMessageNew'
  payload: GroupMessageNewPayload
}

export interface GroupMessageEditedPayload {
  message_id: string
  group_chat_id: string
  content: string
  edited_at: string
}

export interface GroupMessageEditedEvent {
  type: 'GroupMessageEdited'
  payload: GroupMessageEditedPayload
}

export interface GroupMessageRecalledPayload {
  message_id: string
  group_chat_id: string
  recalled_by: number
  recalled_at: string
}

export interface GroupMessageRecalledEvent {
  type: 'GroupMessageRecalled'
  payload: GroupMessageRecalledPayload
}

export interface GroupMessageDeletedPayload {
  message_id: string
  group_chat_id: string
  deleted_by: number
  for_everyone: boolean
  deleted_at: string
}

export interface GroupMessageDeletedEvent {
  type: 'GroupMessageDeleted'
  payload: GroupMessageDeletedPayload
}

export interface GroupMessageReactionPayload {
  message_id: string
  group_chat_id: string
  user: UserInfo
  reaction_type: ReactionType
  has_reaction: boolean
  created_at: string
}

export interface GroupMessageReactionEvent {
  type: 'GroupMessageReaction'
  payload: GroupMessageReactionPayload
}

export interface GroupMessageReadPayload {
  group_chat_id: string
  up_to_message_id: string
  reader_id: number
  read_at: string
}

export interface GroupMessageReadEvent {
  type: 'GroupMessageRead'
  payload: GroupMessageReadPayload
}

export interface PrivateUserTypingPayload {
  target_user_id: number
  user_id: number
  is_typing: boolean
}

export interface PrivateUserTypingEvent {
  type: 'PrivateUserTyping'
  payload: PrivateUserTypingPayload
}

export interface GroupUserTypingPayload {
  group_chat_id: string
  user_id: number
  is_typing: boolean
}

export interface GroupUserTypingEvent {
  type: 'GroupUserTyping'
  payload: GroupUserTypingPayload
}

// ----- 群组事件 -----

export interface GroupMemberJoinedPayload {
  group_chat_id: string
  user: UserInfo
  joined_at: string
}

export interface GroupMemberJoinedEvent {
  type: 'GroupMemberJoined'
  payload: GroupMemberJoinedPayload
}

export interface GroupMemberLeftPayload {
  group_chat_id: string
  user_id: number
  left_at: string
}

export interface GroupMemberLeftEvent {
  type: 'GroupMemberLeft'
  payload: GroupMemberLeftPayload
}

export interface GroupProfileUpdatedPayload {
  group_chat_id: string
  updated_fields: GroupUpdatedFields
  updated_at: string
}

export interface GroupProfileUpdatedEvent {
  type: 'GroupProfileUpdated'
  payload: GroupProfileUpdatedPayload
}

export interface GroupMemberRoleChangedPayload {
  group_chat_id: string
  user_id: number
  old_role: 'owner' | 'admin' | 'member'
  new_role: 'owner' | 'admin' | 'member'
  changed_at: string
}

export interface GroupMemberRoleChangedEvent {
  type: 'GroupMemberRoleChanged'
  payload: GroupMemberRoleChangedPayload
}

export interface GroupMemberMutedPayload {
  group_chat_id: string
  user_id: number
  muted_until: string
  muted_at: string
}

export interface GroupMemberMutedEvent {
  type: 'GroupMemberMuted'
  payload: GroupMemberMutedPayload
}

export interface GroupMemberUnmutedPayload {
  group_chat_id: string
  user_id: number
  unmuted_at: string
}

export interface GroupMemberUnmutedEvent {
  type: 'GroupMemberUnmuted'
  payload: GroupMemberUnmutedPayload
}

export interface GroupDismissedPayload {
  group_chat_id: string
  dismissed_by: number
  dismissed_at: string
}

export interface GroupDismissedEvent {
  type: 'GroupDismissed'
  payload: GroupDismissedPayload
}

export interface GroupInviteReceivedPayload {
  id: string
  group_chat_id: string
  group_name: string
  inviter: UserInfo
  message?: string
  created_at: string
}

export interface GroupInviteReceivedEvent {
  type: 'GroupInviteReceived'
  payload: GroupInviteReceivedPayload
}

export interface GroupInviteResponsePayload {
  id: string
  group_chat_id: string
  user: UserInfo
  status: 'accepted' | 'rejected'
  responded_at: string
}

export interface GroupInviteResponseEvent {
  type: 'GroupInviteResponse'
  payload: GroupInviteResponsePayload
}

export interface GroupApplicationReceivedPayload {
  id: string
  group_chat_id: string
  applicant: UserInfo
  message?: string
  created_at: string
}

export interface GroupApplicationReceivedEvent {
  type: 'GroupApplicationReceived'
  payload: GroupApplicationReceivedPayload
}

export interface GroupApplicationResponsePayload {
  id: string
  group_chat_id: string
  group_name: string
  status: 'accepted' | 'rejected'
  responded_at: string
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: GroupApplicationResponsePayload
}

// ----- 系统事件 -----

export interface SubscribeAckPayload {
  operation: 'subscribe' | 'unsubscribe' | 'set'
  items: SubscriptionItem[]
  code: number
  message?: string
  failed_items?: FailedSubscriptionItem[]
}

export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: SubscribeAckPayload
}

export interface ErrorPayload {
  code: number
  message: string
}

export interface ErrorEvent {
  type: 'Error'
  payload: ErrorPayload
}

// ============================================================================
// 联合类型
// ============================================================================

export type ClientEvent =
  | AuthEvent
  | PingEvent
  | SubscribeEvent
  | UnsubscribeEvent
  | SetSubscriptionsEvent
  | PrivateTypingEvent
  | GroupTypingEvent

export type ServerEvent =
  | ConnectionEstablishedEvent
  | PongEvent
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
 * 创建 Auth 事件
 */
export function createAuthEvent(accessToken: string, deviceFingerprint?: string): AuthEvent {
  return {
    type: 'Auth',
    payload: {
      access_token: accessToken,
      device_fingerprint: deviceFingerprint
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
 * 创建私聊正在输入事件
 */
export function createPrivateTypingEvent(
  target_user_id: number,
  is_typing: boolean
): PrivateTypingEvent {
  return {
    type: 'PrivateTyping',
    payload: { target_user_id, is_typing }
  }
}

/**
 * 创建群聊正在输入事件
 */
export function createGroupTypingEvent(
  group_chat_id: string,
  is_typing: boolean
): GroupTypingEvent {
  return {
    type: 'GroupTyping',
    payload: { group_chat_id, is_typing }
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
