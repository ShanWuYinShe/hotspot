/**
 * WebSocket 实时通信类型定义
 * 严格遵循 api_spec_websocket.mdc 定义
 */

import { UserInfo, ClientType, MessageType, MessageMedia, MessageReplyTo, GroupAnnouncementItem, SystemAnnouncementItem, Location } from './api';

// ============================================================================
// 订阅管理类型
// ============================================================================

export interface SubscriptionItem {
  type: 'group' | 'user'
  ids: string[]
}

export interface FailedSubscriptionId {
  subscription_id: string
  reason: 'invalid' | 'denied' | 'error'
}

export type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';

// ============================================================================
// WebSocket 事件类型枚举
// ============================================================================

/**
 * 客户端发送事件类型
 */
export enum ClientEventType {
  AUTH = 'Auth',
  PING = 'Ping',
  SET_SUBSCRIPTIONS = 'SetSubscriptions',
  GET_SUBSCRIPTIONS = 'GetSubscriptions',
  PRIVATE_TYPING = 'PrivateTyping',
  GROUP_TYPING = 'GroupTyping',
  GROUP_ENTER = 'GroupEnter',
  GROUP_LEAVE = 'GroupLeave'
}

/**
 * 服务端推送事件类型
 */
export enum ServerEventType {
  // 连接
  CONNECTION_ESTABLISHED = 'ConnectionEstablished',
  PONG = 'Pong',

  // 用户状态
  USER_ONLINE = 'UserOnline',
  USER_OFFLINE = 'UserOffline',

  // 社交
  NEW_FOLLOWER = 'NewFollower',
  MUTUAL_FOLLOW = 'MutualFollow',

  // 瞬间互动
  MOMENT_LIKED = 'MomentLiked',
  MOMENT_COMMENTED = 'MomentCommented',
  COMMENT_LIKED = 'CommentLiked',
  COMMENT_REPLY = 'CommentReply',
  MESSAGE_MENTIONED = 'MessageMentioned',

  // 私聊
  PRIVATE_MESSAGE_NEW = 'PrivateMessageNew',
  PRIVATE_MESSAGE_EDITED = 'PrivateMessageEdited',
  PRIVATE_MESSAGE_RECALLED = 'PrivateMessageRecalled',
  PRIVATE_MESSAGE_READ = 'PrivateMessageRead',
  PRIVATE_USER_TYPING = 'PrivateUserTyping',

  // 群聊
  GROUP_MESSAGE_NEW = 'GroupMessageNew',
  GROUP_MESSAGE_EDITED = 'GroupMessageEdited',
  GROUP_MESSAGE_RECALLED = 'GroupMessageRecalled',
  GROUP_USER_TYPING = 'GroupUserTyping',

  // 群组
  GROUP_MEMBER_JOINED = 'GroupMemberJoined',
  GROUP_MEMBER_LEFT = 'GroupMemberLeft',
  GROUP_MEMBER_REMOVED = 'GroupMemberRemoved',
  GROUP_MEMBER_MUTED = 'GroupMemberMuted',
  GROUP_MEMBER_UNMUTED = 'GroupMemberUnmuted',
  GROUP_INVITE_RECEIVED = 'GroupInviteReceived',
  GROUP_INVITE_RESPONSE = 'GroupInviteResponse',
  GROUP_APPLICATION_RECEIVED = 'GroupApplicationReceived',
  GROUP_APPLICATION_RESPONSE = 'GroupApplicationResponse',
  GROUP_ANNOUNCEMENT_CREATED = 'GroupAnnouncementCreated',
  GROUP_UPDATED = 'GroupUpdated',
  GROUP_DISMISSED = 'GroupDismissed',
  GROUP_USER_ENTERED = 'GroupUserEntered',
  GROUP_USER_EXITED = 'GroupUserExited',

  // 用户
  USER_PROFILE_UPDATED = 'UserProfileUpdated',

  // 系统
  SYSTEM_ANNOUNCEMENT = 'SystemAnnouncement',
  ACCOUNT_BANNED = 'AccountBanned',
  ERROR = 'Error',
  SUBSCRIBE_ACK = 'SubscribeAck',
  SUBSCRIPTIONS_SNAPSHOT = 'SubscriptionsSnapshot'
}

// ============================================================================
// 客户端发送事件定义
// ============================================================================

export interface AuthPayload {
  client_type: ClientType
  access_token?: string
}

export interface AuthEvent {
  type: 'Auth'
  payload: AuthPayload
}

export interface PingPayload {
}

export interface PingEvent {
  type: 'Ping'
  payload: PingPayload
}

export interface SetSubscriptionsPayload {
  client_msg_id: string
  operation: SubscriptionOperation
  type: 'group' | 'user'
  ids: string[]
}

export interface SetSubscriptionsEvent {
  type: 'SetSubscriptions'
  payload: SetSubscriptionsPayload
}

export interface GetSubscriptionsPayload {
  client_msg_id: string
}

export interface GetSubscriptionsEvent {
  type: 'GetSubscriptions'
  payload: GetSubscriptionsPayload
}

export interface PrivateTypingPayload {
  target_user_id: string
  is_typing: boolean
}

export interface PrivateTypingEvent {
  type: 'PrivateTyping'
  payload: PrivateTypingPayload
}

export interface GroupTypingPayload {
  group_id: string
  is_typing: boolean
}

export interface GroupTypingEvent {
  type: 'GroupTyping'
  payload: GroupTypingPayload
}

export interface GroupEnterPayload {
  group_id: string
}

export interface GroupEnterEvent {
  type: 'GroupEnter'
  payload: GroupEnterPayload
}

export interface GroupLeavePayload {
  group_id: string
}

export interface GroupLeaveEvent {
  type: 'GroupLeave'
  payload: GroupLeavePayload
}

// ============================================================================
// 服务端推送事件定义
// ============================================================================

export interface ServerWebSocketEvent {
  type: ServerEventType
  payload: Object
  event_id: string
}

// ----- 连接管理事件 -----

export interface MaxSubscriptions {
  user: number
  group: number
}

export interface ConnectionConfig {
  ping_interval: number
  ping_timeout: number
  max_subscriptions: MaxSubscriptions
}

export interface UnreadSummary {
  private_chats: number
  groups: number
  total_messages: number
}

export interface ConnectionEstablishedPayload {
  connection_id: string
  user_id: string
  server_time: string
  config: ConnectionConfig
  unread_summary: UnreadSummary
}

export interface ConnectionEstablishedEvent {
  type: 'ConnectionEstablished'
  payload: ConnectionEstablishedPayload
}

export interface PongPayload {
}

export interface PongEvent {
  type: 'Pong'
  payload: PongPayload
}

// ----- 在线状态事件 -----

export interface UserOnlinePayload {
  user_id: string
  online_at: string
}

export interface UserOnlineEvent {
  type: 'UserOnline'
  payload: UserOnlinePayload
}

export interface UserOfflinePayload {
  user_id: string
  offline_at: string
}

export interface UserOfflineEvent {
  type: 'UserOffline'
  payload: UserOfflinePayload
}

// ----- 关注关系事件 -----

export interface NewFollowerPayload {
  user: UserInfo
  followed_at: string
  is_mutual: boolean
  source?: string
  source_moment_id?: string
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

// ----- 生活瞬间互动事件 -----

export interface MomentSummary {
  moment_id: string
  cover_url?: string
  text_preview: string
}

export interface MomentLikedPayload {
  moment_id: string
  moment_summary: MomentSummary
  user: UserInfo
  liked_at: string
}

export interface MomentLikedEvent {
  type: 'MomentLiked'
  payload: MomentLikedPayload
}

export interface MomentCommentedPayload {
  moment_id: string
  moment_summary: MomentSummary
  comment_id: string
  user: UserInfo
  content: string
  commented_at: string
}

export interface MomentCommentedEvent {
  type: 'MomentCommented'
  payload: MomentCommentedPayload
}

export interface CommentLikedPayload {
  comment_id: string
  moment_id: string
  moment_summary?: MomentSummary
  comment_preview?: string
  user: UserInfo
  liked_at: string
}

export interface CommentLikedEvent {
  type: 'CommentLiked'
  payload: CommentLikedPayload
}

export interface CommentReplyPayload {
  moment_id: string
  comment_id: string
  reply_to_content: string
  reply_id: string
  user: UserInfo
  content: string
  replied_at: string
}

export interface CommentReplyEvent {
  type: 'CommentReply'
  payload: CommentReplyPayload
}

export interface MessageMentionedPayload {
  target_type: 'moment' | 'comment'
  moment_id: string
  comment_id?: string
  moment_summary?: MomentSummary
  comment_preview?: string
  user: UserInfo
  mentioned_at: string
}

export interface MessageMentionedEvent {
  type: 'MessageMentioned'
  payload: MessageMentionedPayload
}

// ----- 用户事件 -----

export interface UserProfileUpdatedPayload {
  user: UserInfo
  updated_at: string
}

export interface UserProfileUpdatedEvent {
  type: 'UserProfileUpdated'
  payload: UserProfileUpdatedPayload
}

// ----- 系统事件 -----

export interface SystemAnnouncementPayload {
  announcement_id: string
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

export interface AccountBannedPayload {
  reason: string
  duration_days: number
  banned_at: string
}

export interface AccountBannedEvent {
  type: 'AccountBanned'
  payload: AccountBannedPayload
}

// ----- 聊天消息事件 -----

export interface PrivateMessageNewPayload {
  message_id: string
  target_user_id: string
  sender_id: string
  type: MessageType
  content?: string
  share_code?: string
  reply_to?: MessageReplyTo
  media?: MessageMedia
  sent_at: string
}

export interface PrivateMessageNewEvent {
  type: 'PrivateMessageNew'
  payload: PrivateMessageNewPayload
}

export interface PrivateMessageEditedPayload {
  message_id: string
  target_user_id: string
  content: string
  edited_at: string
}

export interface PrivateMessageEditedEvent {
  type: 'PrivateMessageEdited'
  payload: PrivateMessageEditedPayload
}

export interface PrivateMessageRecalledPayload {
  message_id: string
  target_user_id: string
  recalled_by: string
  recalled_at: string
}

export interface PrivateMessageRecalledEvent {
  type: 'PrivateMessageRecalled'
  payload: PrivateMessageRecalledPayload
}

export interface PrivateMessageReadPayload {
  target_user_id: string
  up_to: string
  reader_id: string
  read_at: string
}

export interface PrivateMessageReadEvent {
  type: 'PrivateMessageRead'
  payload: PrivateMessageReadPayload
}

export interface PrivateUserTypingPayload {
  target_user_id: string
  user_id: string
  is_typing: boolean
}

export interface PrivateUserTypingEvent {
  type: 'PrivateUserTyping'
  payload: PrivateUserTypingPayload
}

// 群聊消息事件

export interface GroupMessageNewPayload {
  message_id: string
  group_id: string
  sender: UserInfo
  type: MessageType
  content?: string
  share_code?: string
  reply_to?: MessageReplyTo
  mentions?: string[]
  media?: MessageMedia
  sent_at: string
}

export interface GroupMessageNewEvent {
  type: 'GroupMessageNew'
  payload: GroupMessageNewPayload
}

export interface GroupMessageEditedPayload {
  message_id: string
  group_id: string
  content: string
  edited_at: string
}

export interface GroupMessageEditedEvent {
  type: 'GroupMessageEdited'
  payload: GroupMessageEditedPayload
}

export interface GroupMessageRecalledPayload {
  message_id: string
  group_id: string
  recalled_by: string
  recalled_at: string
}

export interface GroupMessageRecalledEvent {
  type: 'GroupMessageRecalled'
  payload: GroupMessageRecalledPayload
}

export interface GroupUserTypingPayload {
  group_id: string
  user_id: string
  is_typing: boolean
}

export interface GroupUserTypingEvent {
  type: 'GroupUserTyping'
  payload: GroupUserTypingPayload
}

// ----- 群组事件 -----

export interface GroupMemberJoinedPayload {
  group_id: string
  user: UserInfo
  role: 'owner' | 'admin' | 'member'
  member_count: number
  joined_at: string
}

export interface GroupMemberJoinedEvent {
  type: 'GroupMemberJoined'
  payload: GroupMemberJoinedPayload
}

export interface GroupMemberLeftPayload {
  group_id: string
  user: UserInfo
  left_at: string
  member_count: number
}

export interface GroupMemberLeftEvent {
  type: 'GroupMemberLeft'
  payload: GroupMemberLeftPayload
}

export interface GroupMemberRemovedPayload {
  group_id: string
  user: UserInfo
  removed_by: string
  reason?: string
  removed_at: string
  member_count: number
}

export interface GroupMemberRemovedEvent {
  type: 'GroupMemberRemoved'
  payload: GroupMemberRemovedPayload
}

export interface GroupMemberMutedPayload {
  group_id: string
  user: UserInfo
  muted_by: string
  muted_until: string
  duration_minutes: number
  muted_at: string
}

export interface GroupMemberMutedEvent {
  type: 'GroupMemberMuted'
  payload: GroupMemberMutedPayload
}

export interface GroupMemberUnmutedPayload {
  group_id: string
  user: UserInfo
  unmuted_by: string
  unmuted_at: string
}

export interface GroupMemberUnmutedEvent {
  type: 'GroupMemberUnmuted'
  payload: GroupMemberUnmutedPayload
}

export interface GroupAnnouncementCreatedPayload extends GroupAnnouncementItem {}

export interface GroupAnnouncementCreatedEvent {
  type: 'GroupAnnouncementCreated'
  payload: GroupAnnouncementCreatedPayload
}

export interface GroupUpdatedPayload {
  group_id: string
  updated_fields: Record<string, Object>
  updated_by: string
  updated_at: string
}

export interface GroupUpdatedEvent {
  type: 'GroupUpdated'
  payload: GroupUpdatedPayload
}

export interface GroupDismissedPayload {
  group_id: string
  dismissed_at: string
  reason?: string
}

export interface GroupDismissedEvent {
  type: 'GroupDismissed'
  payload: GroupDismissedPayload
}

export interface GroupInviteReceivedPayload {
  invite_id: string
  group_id: string
  group_name: string
  group_avatar_url?: string
  inviter: UserInfo
  message?: string
  created_at: string
}

export interface GroupInviteReceivedEvent {
  type: 'GroupInviteReceived'
  payload: GroupInviteReceivedPayload
}

export interface GroupInviteResponsePayload {
  invite_id: string
  group_id: string
  user: UserInfo
  status: 'accepted'
  responded_at: string
}

export interface GroupInviteResponseEvent {
  type: 'GroupInviteResponse'
  payload: GroupInviteResponsePayload
}

export interface GroupApplicationReceivedPayload {
  application_id: string
  group_id: string
  applicant: UserInfo
  message?: string
  created_at: string
}

export interface GroupApplicationReceivedEvent {
  type: 'GroupApplicationReceived'
  payload: GroupApplicationReceivedPayload
}

export interface GroupApplicationResponsePayload {
  application_id: string
  group_id: string
  group_name: string
  status: 'accepted'
  responded_at: string
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: GroupApplicationResponsePayload
}

export interface GroupUserEnteredPayload {
  group_id: string
  user: UserInfo
  entered_at: string
}

export interface GroupUserEnteredEvent {
  type: 'GroupUserEntered'
  payload: GroupUserEnteredPayload
}

export interface GroupUserExitedPayload {
  group_id: string
  user: UserInfo
  exited_at: string
}

export interface GroupUserExitedEvent {
  type: 'GroupUserExited'
  payload: GroupUserExitedPayload
}

// ----- 系统/订阅事件 -----

export interface SubscribeAckPayload {
  client_msg_id: string
  operation: SubscriptionOperation
  type: 'group' | 'user'
  ids: string[]
  code: number
  message?: string
  failed_ids?: FailedSubscriptionId[]
}

export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: SubscribeAckPayload
}

export interface SubscriptionsSnapshotPayload {
  client_msg_id: string
  user_ids: string[]
  group_ids: string[]
  total_count: number
}

export interface SubscriptionsSnapshotEvent {
  type: 'SubscriptionsSnapshot'
  payload: SubscriptionsSnapshotPayload
}

export interface ErrorPayload {
  code: number
  message: string
  severity: 'fatal' | 'recoverable' | 'warning'
  retryable: boolean
  retry_after?: number
  context?: Record<string, Object>
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
  | SetSubscriptionsEvent
  | GetSubscriptionsEvent
  | PrivateTypingEvent
  | GroupTypingEvent
  | GroupEnterEvent
  | GroupLeaveEvent

export type ServerEvent =
  | ConnectionEstablishedEvent
  | PongEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | NewFollowerEvent
  | MutualFollowEvent
  | MomentLikedEvent
  | MomentCommentedEvent
  | CommentLikedEvent
  | CommentReplyEvent
  | MessageMentionedEvent
  | UserProfileUpdatedEvent
  | SystemAnnouncementEvent
  | AccountBannedEvent
  | PrivateMessageNewEvent
  | PrivateMessageEditedEvent
  | PrivateMessageRecalledEvent
  | PrivateMessageReadEvent
  | PrivateUserTypingEvent
  | GroupMessageNewEvent
  | GroupMessageEditedEvent
  | GroupMessageRecalledEvent
  | GroupUserTypingEvent
  | GroupMemberJoinedEvent
  | GroupMemberLeftEvent
  | GroupMemberRemovedEvent
  | GroupMemberMutedEvent
  | GroupMemberUnmutedEvent
  | GroupInviteReceivedEvent
  | GroupInviteResponseEvent
  | GroupApplicationReceivedEvent
  | GroupApplicationResponseEvent
  | GroupAnnouncementCreatedEvent
  | GroupUpdatedEvent
  | GroupDismissedEvent
  | GroupUserEnteredEvent
  | GroupUserExitedEvent
  | SubscribeAckEvent
  | SubscriptionsSnapshotEvent
  | ErrorEvent

// ============================================================================
// 辅助类型
// ============================================================================

export enum WebSocketState {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  ERROR = 'error'
}

export interface ReconnectConfig {
  baseDelay: number
  maxDelay: number
  maxAttempts: number
  jitterFactor: number
}

// ============================================================================
// 辅助函数
// ============================================================================

export function createPingEvent(): PingEvent {
  return {
    type: 'Ping',
    payload: {}
  }
}

export function createAuthEvent(clientType: ClientType, accessToken?: string): AuthEvent {
  return {
    type: 'Auth',
    payload: {
      client_type: clientType,
      access_token: accessToken
    }
  }
}

export function createSetSubscriptionsEvent(
  operation: SubscriptionOperation,
  item: SubscriptionItem,
  clientMsgId?: string
): SetSubscriptionsEvent {
  return {
    type: 'SetSubscriptions',
    payload: {
      client_msg_id: clientMsgId ?? generateClientMsgId(),
      operation: operation,
      type: item.type,
      ids: item.ids
    }
  }
}

export function createGetSubscriptionsEvent(clientMsgId?: string): GetSubscriptionsEvent {
  return {
    type: 'GetSubscriptions',
    payload: {
      client_msg_id: clientMsgId ?? generateClientMsgId()
    }
  }
}

function generateClientMsgId(): string {
  return `${Date.now()}-${Math.floor(Math.random() * 1e9)}`
}

export function createPrivateTypingEvent(
  target_user_id: string,
  is_typing: boolean
): PrivateTypingEvent {
  return {
    type: 'PrivateTyping',
    payload: { target_user_id, is_typing }
  }
}

export function createGroupTypingEvent(
  group_id: string,
  is_typing: boolean
): GroupTypingEvent {
  return {
    type: 'GroupTyping',
    payload: { group_id, is_typing }
  }
}

export function createGroupEnterEvent(group_id: string): GroupEnterEvent {
  return {
    type: 'GroupEnter',
    payload: { group_id }
  }
}

export function createGroupLeaveEvent(group_id: string): GroupLeaveEvent {
  return {
    type: 'GroupLeave',
    payload: { group_id }
  }
}

/**
 * 已处理的事件ID集合（用于去重，维护最近1000个）
 */
const processedEventIds: Set<string> = new Set()
const MAX_PROCESSED_EVENT_IDS = 1000

function isEventProcessed(eventId: string): boolean {
  return processedEventIds.has(eventId)
}

function markEventProcessed(eventId: string): void {
  if (processedEventIds.size >= MAX_PROCESSED_EVENT_IDS) {
    const firstId = processedEventIds.values().next().value as Object
    processedEventIds.delete(firstId as string)
  }
  processedEventIds.add(eventId)
}

/**
 * 解析服务端事件（支持 event_id 去重）
 */
export function parseServerEvent(data: string): ServerEvent | null {
  try {
    const raw = JSON.parse(data) as Record<string, Object>

    // 检查是否有 event_id（去重）
    if (raw['event_id']) {
      const eventId = raw['event_id'] as string

      if (isEventProcessed(eventId)) {
        console.warn(`重复事件已忽略: ${eventId}`)
        return null
      }

      markEventProcessed(eventId)
    }

    // 转换为 ServerEvent（移除 event_id）
    const event: Record<string, Object> = {};
    const rawKeys = Object.keys(raw);
    for (let i = 0; i < rawKeys.length; i++) {
      const key = rawKeys[i];
      if (key !== 'event_id') {
        event[key] = raw[key];
      }
    }
    return (event as Object) as ServerEvent
  } catch (e) {
    console.error('Failed to parse WebSocket message:', e)
    return null
  }
}

export function serializeClientEvent(event: ClientEvent): string {
  return JSON.stringify(event)
}
