/**
 * WebSocket 实时通信类型定义
 * 严格遵循 api_spec.mdc WebSocket 实时事件定义
 */

import { UserInfo, ReactionType, GroupSettings, GroupJoinMethod, RelativeLocation } from './api';

// ============================================================================
// 通用类型
// ============================================================================

/**
 * 消息类型（接收时使用，包含系统消息）
 */
export enum ReceiveMessageType {
  TEXT = 'text',
  IMAGE = 'image',
  VIDEO = 'video',
  AUDIO = 'audio',
  FILE = 'file',
  SYSTEM = 'system'
}

/**
 * 群组更新字段
 */
export interface GroupUpdatedFields {
  name?: string
  description?: string
  avatar_url?: string
  settings?: GroupSettingsUpdateFields
}

export interface GroupSettingsUpdateFields {
  join_method?: GroupJoinMethod
  is_searchable?: boolean
  max_members?: number
  location?: RelativeLocation
}

// ============================================================================
// 订阅管理类型
// ============================================================================

/**
 * 订阅项定义
 * 注意: ids 类型根据 type 不同而不同
 *   - group: string[] (群组ID)
 *   - user: string[] (用户ID)
 */
export interface SubscriptionItem {
  type: 'group' | 'user'
  ids: string[]
}

/**
 * 失败的订阅项
 */
export interface FailedSubscriptionId {
  subscription_id: string
  reason: 'invalid' | 'denied' | 'error'
}

// ============================================================================
// WebSocket 事件类型枚举
// ============================================================================

/**
 * 客户端发送事件类型
 */
export enum ClientEventType {
  AUTH = 'Auth',
  PING = 'Ping',
  SUBSCRIBE = 'Subscribe',
  UNSUBSCRIBE = 'Unsubscribe',
  SET_SUBSCRIPTIONS = 'SetSubscriptions',
  GET_SUBSCRIPTIONS = 'GetSubscriptions',
  PRIVATE_TYPING = 'PrivateTyping',
  GROUP_TYPING = 'GroupTyping'
}

/**
 * 服务端推送事件类型
 */
export enum ServerEventType {
  CONNECTION_ESTABLISHED = 'ConnectionEstablished',
  PONG = 'Pong',
  BATCH_EVENTS = 'BatchEvents',
  USER_ONLINE = 'UserOnline',
  USER_OFFLINE = 'UserOffline',
  USER_PRESENCE_UPDATE = 'UserPresenceUpdate',
  USER_LOCATION_UPDATE = 'UserLocationUpdate',
  NEW_FOLLOWER = 'NewFollower',
  MUTUAL_FOLLOW = 'MutualFollow',
  FOLLOW_REQUEST_RECEIVED = 'FollowRequestReceived',
  FOLLOW_REQUEST_HANDLED = 'FollowRequestHandled',
  MOMENT_LIKED = 'MomentLiked',
  MOMENT_COMMENTED = 'MomentCommented',
  COMMENT_REPLY = 'CommentReply',
  MESSAGE_MENTIONED = 'MessageMentioned',
  SYSTEM_ANNOUNCEMENT = 'SystemAnnouncement',
  PRIVATE_MESSAGE_NEW = 'PrivateMessageNew',
  PRIVATE_MESSAGE_EDITED = 'PrivateMessageEdited',
  PRIVATE_MESSAGE_RECALLED = 'PrivateMessageRecalled',
  PRIVATE_MESSAGE_DELETED = 'PrivateMessageDeleted',
  PRIVATE_MESSAGE_REACTION = 'PrivateMessageReaction',
  PRIVATE_MESSAGE_READ = 'PrivateMessageRead',
  PRIVATE_USER_TYPING = 'PrivateUserTyping',
  GROUP_MESSAGE_NEW = 'GroupMessageNew',
  GROUP_MESSAGE_EDITED = 'GroupMessageEdited',
  GROUP_MESSAGE_RECALLED = 'GroupMessageRecalled',
  GROUP_MESSAGE_DELETED = 'GroupMessageDeleted',
  GROUP_MESSAGE_REACTION = 'GroupMessageReaction',
  GROUP_MESSAGE_READ = 'GroupMessageRead',
  GROUP_USER_TYPING = 'GroupUserTyping',
  GROUP_MEMBER_JOINED = 'GroupMemberJoined',
  GROUP_MEMBER_LEFT = 'GroupMemberLeft',
  GROUP_MEMBER_REMOVED = 'GroupMemberRemoved',
  GROUP_MEMBER_ROLE_CHANGED = 'GroupMemberRoleChanged',
  GROUP_MEMBER_MUTED = 'GroupMemberMuted',
  GROUP_MEMBER_UNMUTED = 'GroupMemberUnmuted',
  GROUP_INVITE_RECEIVED = 'GroupInviteReceived',
  GROUP_INVITE_RESPONSE = 'GroupInviteResponse',
  GROUP_APPLICATION_RECEIVED = 'GroupApplicationReceived',
  GROUP_APPLICATION_RESPONSE = 'GroupApplicationResponse',
  GROUP_ANNOUNCEMENT_CREATED = 'GroupAnnouncementCreated',
  GROUP_ANNOUNCEMENT_UPDATED = 'GroupAnnouncementUpdated',
  GROUP_UPDATED = 'GroupUpdated',
  GROUP_DELETED = 'GroupDeleted',
  SUBSCRIBE_ACK = 'SubscribeAck',
  SUBSCRIPTIONS_SNAPSHOT = 'SubscriptionsSnapshot',
  ERROR = 'Error'
}

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
}

export interface PingEvent {
  type: 'Ping'
  payload: PingPayload
}

/**
 * 订阅事件
 */
export interface SubscribePayload {
  client_msg_id: string
  type: 'group' | 'user'
  ids: string[]
}

export interface SubscribeEvent {
  type: 'Subscribe'
  payload: SubscribePayload
}

export interface UnsubscribePayload {
  client_msg_id: string
  type: 'group' | 'user'
  ids: string[]
}

export interface UnsubscribeEvent {
  type: 'Unsubscribe'
  payload: UnsubscribePayload
}

export interface SetSubscriptionsPayload {
  client_msg_id: string
  type: 'group' | 'user'
  ids: string[]
}

export interface GetSubscriptionsPayload {
  client_msg_id: string
}

export interface GetSubscriptionsEvent {
  type: 'GetSubscriptions'
  payload: GetSubscriptionsPayload
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
  group_id: string
  is_typing: boolean
}

export interface GroupTypingEvent {
  type: 'GroupTyping'
  payload: GroupTypingPayload
}

// ============================================================================
// 服务端推送事件定义
// ============================================================================

/**
 * 服务端推送事件基础接口（包含 event_id 用于去重）
 */
export interface ServerWebSocketEvent {
  type: ServerEventType
  payload: Object
  event_id: string  // 事件唯一ID（UUID），用于客户端去重（幂等性保障）
}

// ----- 连接管理事件 -----

/**
 * ConnectionEstablished 订阅上限配置
 */
export interface MaxSubscriptions {
  user: number   // 用户订阅上限
  group: number  // 群组订阅上限
}

/**
 * ConnectionEstablished 配置信息
 */
export interface ConnectionConfig {
  ping_interval: number  // 心跳间隔（秒）
  ping_timeout: number   // Pong 超时（秒）
  max_subscriptions: MaxSubscriptions
}

/**
 * ConnectionEstablished 未读计数
 */
export interface UnreadCounts {
  messages: number        // 未读私聊消息数
  group_messages: number  // 未读群聊消息数
  notifications: number   // 未读通知数
}

export interface ConnectionEstablishedPayload {
  connection_id: string
  user_id: number
  server_time: string  // 服务端当前时间（UTC ISO 8601），客户端用于判断断开时长
  config: ConnectionConfig  // 服务端配置（动态配置，客户端不得硬编码）
  unread_counts: UnreadCounts  // 未读计数（快速显示角标）
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

/**
 * 批量事件推送（性能优化）
 */
export interface BatchEventsPayload {
  batch_id: string  // 批次ID（UUID）
  events: ServerWebSocketEvent[]  // 多个事件（已包含 event_id）
}

export interface BatchEventsEvent {
  type: 'BatchEvents'
  payload: BatchEventsPayload
}

// ----- 在线状态事件 -----

/**
 * 用户状态类型
 */
export enum UserPresenceStatus {
  ONLINE = 'online',      // 正在活跃（有 WebSocket 连接且 5 分钟内有活动）
  IDLE = 'idle',        // 空闲/离开（有连接但 5 分钟没活动）
  DND = 'dnd',         // 请勿打扰（用户主动设置）
  INVISIBLE = 'invisible',   // 隐身（对他人显示为 offline）
  OFFLINE = 'offline'     // 离线（无连接）
}

export interface UserOnlinePayload {
  user_id: number
  nickname: string
  avatar_url?: string
  status: UserPresenceStatus  // 状态类型
  status_text?: string  // 自定义状态文本（如"开会中"、"勿扰"）
  online_at: string
}

export interface UserOnlineEvent {
  type: 'UserOnline'
  payload: UserOnlinePayload
}

export interface UserOfflinePayload {
  user_id: number
  status: UserPresenceStatus  // 固定为 'offline'
  last_seen?: string  // 最后在线时间（可选）
  offline_at: string
}

export interface UserOfflineEvent {
  type: 'UserOffline'
  payload: UserOfflinePayload
}

/**
 * 用户状态变更（online/idle/dnd/invisible 之间切换）
 */
export interface UserPresenceUpdatePayload {
  user_id: number
  status: UserPresenceStatus  // 新状态
  status_text?: string  // 自定义状态文本
  updated_at: string
}

export interface UserPresenceUpdateEvent {
  type: 'UserPresenceUpdate'
  payload: UserPresenceUpdatePayload
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

/**
 * 收到关注请求（当 allow_follow 设置为需要审批时）
 */
export interface FollowRequestReceivedPayload {
  request_id: string
  requester: UserInfo
  message?: string
  requested_at: string
}

export interface FollowRequestReceivedEvent {
  type: 'FollowRequestReceived'
  payload: FollowRequestReceivedPayload
}

/**
 * 关注请求被处理（发送给请求方）
 */
export interface FollowRequestHandledPayload {
  request_id: string
  target_user: UserInfo  // 被请求关注的用户
  action: 'accept'  // 仅接受时推送，拒绝不通过 WebSocket 推送
  handled_at: string
}

export interface FollowRequestHandledEvent {
  type: 'FollowRequestHandled'
  payload: FollowRequestHandledPayload
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
  comment_id: string  // 原评论ID（被回复的评论）
  reply_to_content: string  // 原评论内容摘要
  reply_id: string  // 回复评论ID
  user: UserInfo  // 回复者
  content: string
  replied_at: string
}

export interface CommentReplyEvent {
  type: 'CommentReply'
  payload: CommentReplyPayload
}

/**
 * 瞬间摘要信息
 */
export interface MomentSummary {
  moment_id: string
  cover_url?: string  // 封面图 (第一张图片)
  text_preview: string  // 文本内容摘要 (前50字)
}

/**
 * 在内容中被提及
 */
export interface MessageMentionedPayload {
  target_type: 'moment' | 'comment'
  moment_id: string
  comment_id?: string
  moment_summary?: MomentSummary
  comment_preview?: string
  user: UserInfo  // 提及我的人
  mentioned_at: string
}

export interface MessageMentionedEvent {
  type: 'MessageMentioned'
  payload: MessageMentionedPayload
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
  group_id: string
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
  recalled_by: number
  recalled_at: string
}

export interface GroupMessageRecalledEvent {
  type: 'GroupMessageRecalled'
  payload: GroupMessageRecalledPayload
}

export interface GroupMessageDeletedPayload {
  message_id: string
  group_id: string
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
  group_id: string
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
  group_id: string
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
  group_id: string
  user_id: number
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
  role: 'owner' | 'admin' | 'member'  // 引用 @api_spec_common_types.mdc
  member_count: number  // 当前成员总数
  joined_at: string
}

export interface GroupMemberJoinedEvent {
  type: 'GroupMemberJoined'
  payload: GroupMemberJoinedPayload
}

/**
 * 成员离开（自愿离开）
 */
export interface GroupMemberLeftPayload {
  group_id: string
  user: UserInfo
  left_at: string
  member_count: number  // 剩余成员总数
}

export interface GroupMemberLeftEvent {
  type: 'GroupMemberLeft'
  payload: GroupMemberLeftPayload
}

/**
 * 成员被移除（被踢出/被封禁）
 */
export interface GroupMemberRemovedPayload {
  group_id: string
  user: UserInfo
  removed_by: number  // user_id（操作者）
  reason?: 'kicked' | 'banned'  // 移除原因
  removed_at: string
  member_count: number
}

export interface GroupMemberRemovedEvent {
  type: 'GroupMemberRemoved'
  payload: GroupMemberRemovedPayload
}

/**
 * 成员角色变更
 */
export interface GroupMemberRoleChangedPayload {
  group_id: string
  user: UserInfo
  old_role: 'owner' | 'admin' | 'member'  // 引用 @api_spec_common_types.mdc
  new_role: 'owner' | 'admin' | 'member'
  changed_by: number  // user_id（操作者）
  changed_at: string
}

export interface GroupMemberRoleChangedEvent {
  type: 'GroupMemberRoleChanged'
  payload: GroupMemberRoleChangedPayload
}

/**
 * 成员被禁言
 */
export interface GroupMemberMutedPayload {
  group_id: string
  user: UserInfo
  muted_by: number  // user_id（操作者）
  muted_until?: string  // 禁言结束时间（null 表示永久禁言）
  reason?: string
  muted_at: string
}

export interface GroupMemberMutedEvent {
  type: 'GroupMemberMuted'
  payload: GroupMemberMutedPayload
}

/**
 * 成员解除禁言
 */
export interface GroupMemberUnmutedPayload {
  group_id: string
  user: UserInfo
  unmuted_by: number  // user_id（操作者）
  unmuted_at: string
}

export interface GroupMemberUnmutedEvent {
  type: 'GroupMemberUnmuted'
  payload: GroupMemberUnmutedPayload
}

/**
 * 群公告创建
 */
export interface GroupAnnouncementCreatedPayload {
  group_id: string
  announcement_id: string
  content: string
  created_by: number  // user_id（操作者）
  created_at: string
}

export interface GroupAnnouncementCreatedEvent {
  type: 'GroupAnnouncementCreated'
  payload: GroupAnnouncementCreatedPayload
}

/**
 * 群公告更新
 */
export interface GroupAnnouncementUpdatedPayload {
  group_id: string
  announcement_id: string
  content: string  // 更新后的内容
  updated_by: number  // user_id（操作者）
  updated_at: string
}

export interface GroupAnnouncementUpdatedEvent {
  type: 'GroupAnnouncementUpdated'
  payload: GroupAnnouncementUpdatedPayload
}

/**
 * 群组更新字段
 */
export interface GroupUpdatedFields {
  name?: string
  avatar_url?: string
  description?: string
}

/**
 * 群组信息更新
 */
export interface GroupUpdatedPayload {
  group_id: string
  updated_fields: GroupUpdatedFields  // 只包含变更的字段
  updated_by: number  // user_id（操作者）
  updated_at: string
}

export interface GroupUpdatedEvent {
  type: 'GroupUpdated'
  payload: GroupUpdatedPayload
}

/**
 * 群组解散
 */
export interface GroupDeletedPayload {
  group_id: string
  deleted_by: number  // user_id（操作者）
  deleted_at: string
  reason?: string
}

export interface GroupDeletedEvent {
  type: 'GroupDeleted'
  payload: GroupDeletedPayload
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
  status: 'accepted'  // 仅接受时推送，拒绝不通过 WebSocket 推送
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
  status: 'accepted'  // 仅接受时推送，拒绝不通过 WebSocket 推送
  responded_at: string
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: GroupApplicationResponsePayload
}

// ----- 系统事件 -----

export interface SubscribeAckPayload {
  client_msg_id: string
  operation: 'subscribe' | 'unsubscribe' | 'set'
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

/**
 * 订阅状态快照（响应 GetSubscriptions）
 */
export interface SubscriptionsSnapshotPayload {
  client_msg_id: string  // 回传客户端消息ID
  user_ids: string[]  // 当前订阅的用户ID列表
  group_ids: string[]  // 当前订阅的群组ID列表
  total_count: number  // 总订阅数（user_ids.length + group_ids.length）
}

export interface SubscriptionsSnapshotEvent {
  type: 'SubscriptionsSnapshot'
  payload: SubscriptionsSnapshotPayload
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
  | GetSubscriptionsEvent
  | PrivateTypingEvent
  | GroupTypingEvent

export type ServerEvent =
  | ConnectionEstablishedEvent
  | PongEvent
  | BatchEventsEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | UserPresenceUpdateEvent
  | UserLocationUpdateEvent
  | NewFollowerEvent
  | MutualFollowEvent
  | FollowRequestReceivedEvent
  | FollowRequestHandledEvent
  | MomentLikedEvent
  | MomentCommentedEvent
  | CommentReplyEvent
  | MessageMentionedEvent
  | SystemAnnouncementEvent
  | PrivateMessageNewEvent
  | PrivateMessageEditedEvent
  | PrivateMessageRecalledEvent
  | PrivateMessageDeletedEvent
  | PrivateMessageReactionEvent
  | PrivateMessageReadEvent
  | PrivateUserTypingEvent
  | GroupMessageNewEvent
  | GroupMessageEditedEvent
  | GroupMessageRecalledEvent
  | GroupMessageDeletedEvent
  | GroupMessageReactionEvent
  | GroupMessageReadEvent
  | GroupUserTypingEvent
  | GroupMemberJoinedEvent
  | GroupMemberLeftEvent
  | GroupMemberRemovedEvent
  | GroupMemberRoleChangedEvent
  | GroupMemberMutedEvent
  | GroupMemberUnmutedEvent
  | GroupInviteReceivedEvent
  | GroupInviteResponseEvent
  | GroupApplicationReceivedEvent
  | GroupApplicationResponseEvent
  | GroupAnnouncementCreatedEvent
  | GroupAnnouncementUpdatedEvent
  | GroupUpdatedEvent
  | GroupDeletedEvent
  | SubscribeAckEvent
  | SubscriptionsSnapshotEvent
  | ErrorEvent

// ============================================================================
// 辅助类型
// ============================================================================

/**
 * WebSocket 连接状态
 */
export enum WebSocketState {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  ERROR = 'error'
}

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
    payload: {}
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
export function createSubscribeEvent(item: SubscriptionItem, clientMsgId?: string): SubscribeEvent {
  return {
    type: 'Subscribe',
    payload: {
      client_msg_id: clientMsgId ?? generateClientMsgId(),
      type: item.type,
      ids: item.ids
    }
  }
}

/**
 * 创建取消订阅事件
 */
export function createUnsubscribeEvent(item: SubscriptionItem, clientMsgId?: string): UnsubscribeEvent {
  return {
    type: 'Unsubscribe',
    payload: {
      client_msg_id: clientMsgId ?? generateClientMsgId(),
      type: item.type,
      ids: item.ids
    }
  }
}

/**
 * 创建替换式订阅事件
 */
export function createSetSubscriptionsEvent(item: SubscriptionItem, clientMsgId?: string): SetSubscriptionsEvent {
  return {
    type: 'SetSubscriptions',
    payload: {
      client_msg_id: clientMsgId ?? generateClientMsgId(),
      type: item.type,
      ids: item.ids
    }
  }
}

/**
 * 创建订阅状态查询事件
 */
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
  group_id: string,
  is_typing: boolean
): GroupTypingEvent {
  return {
    type: 'GroupTyping',
    payload: { group_id, is_typing }
  }
}

/**
 * 已处理的事件ID集合（用于去重，维护最近1000个）
 */
const processedEventIds: Set<string> = new Set()
const MAX_PROCESSED_EVENT_IDS = 1000

/**
 * 检查事件是否已处理（基于 event_id 去重）
 */
function isEventProcessed(eventId: string): boolean {
  return processedEventIds.has(eventId)
}

/**
 * 标记事件已处理
 */
function markEventProcessed(eventId: string): void {
  // 如果超过最大数量，删除最旧的（FIFO）
  if (processedEventIds.size >= MAX_PROCESSED_EVENT_IDS) {
    const firstId = processedEventIds.values().next().value
    processedEventIds.delete(firstId)
  }
  processedEventIds.add(eventId)
}

/**
 * 解析服务端事件（支持 ServerWebSocketEvent 格式，包含 event_id）
 * 返回 ServerEvent 或 BatchEventsEvent
 */
export function parseServerEvent(data: string): ServerEvent | BatchEventsEvent | null {
  try {
    const raw = JSON.parse(data) as Object
    
    // 检查是否是批量事件
    if (raw.type === 'BatchEvents' && raw.payload && raw.payload.events) {
      // 批量事件本身不需要 event_id，但内部事件需要逐个处理
      // 对批量事件中的每个事件进行去重检查
      const batchEvent = raw as BatchEventsEvent
      batchEvent.payload.events = batchEvent.payload.events.filter((subEvent) => {
        if (subEvent.event_id) {
          if (isEventProcessed(subEvent.event_id)) {
            console.warn(`批量事件中重复事件已忽略: ${subEvent.event_id}`)
            return false
          }
          markEventProcessed(subEvent.event_id)
        }
        return true
      })
      return batchEvent
    }
    
    // 检查是否有 event_id（ServerWebSocketEvent 格式）
    if (raw.event_id) {
      const eventId = raw.event_id as string
      
      // 去重检查
      if (isEventProcessed(eventId)) {
        console.warn(`重复事件已忽略: ${eventId}`)
        return null
      }
      
      // 标记为已处理
      markEventProcessed(eventId)
    }
    
    // 转换为 ServerEvent（移除 event_id，因为类型定义中不包含）
    const rawRecord = raw as Record<string, Object>;
    const event: Record<string, Object> = {};
    const rawKeys = Object.keys(rawRecord);
    for (let i = 0; i < rawKeys.length; i++) {
      const key = rawKeys[i];
      if (key !== 'event_id') {
        event[key] = rawRecord[key];
      }
    }
    return event as unknown as ServerEvent
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
