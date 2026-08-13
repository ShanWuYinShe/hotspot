/**
 * WebSocket 实时通信类型定义
 * 严格遵循 api_spec_websocket.md 定义
 */

import { UserInfo, GroupInfo, ClientType, MessageType, MessageMedia, MessageReplyTo, GroupAnnouncementItem, MomentSummary, UserSettings, FollowSource, GroupJoinMethod, Location } from './api';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF38;
const TAG: string = 'WebSocketTypes';

// ============================================================================
// 订阅管理类型
// ============================================================================

export interface SubscriptionItem {
  type: 'group' | 'user'
  ids: string[]
}

export interface FailedSubscriptionId {
  type: 'group' | 'user'
  subscription_id: string
  reason: 'invalid' | 'denied' | 'limit_exceeded' | 'error'
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
  PRIVATE_TYPING = 'SendPrivateTyping',
  GROUP_TYPING = 'SendGroupTyping'
}

/**
 * 服务端推送事件类型
 */
export enum ServerEventType {
  // 连接
  CONNECTION_ESTABLISHED = 'ConnectionEstablished',
  PONG = 'Pong',

  // 会话
  SESSION_REVOKED = 'SessionRevoked',

  // 用户状态
  USER_ONLINE = 'UserOnline',
  USER_OFFLINE = 'UserOffline',

  // 社交
  NEW_FOLLOWER = 'NewFollower',
  MUTUAL_FOLLOW = 'MutualFollow',

  // 瞬间互动（含表情反应）
  REACTION_ADDED = 'ReactionAdded',
  REACTION_REMOVED = 'ReactionRemoved',
  MOMENT_COMMENTED = 'MomentCommented',
  COMMENT_REPLY = 'CommentReply',
  MESSAGE_MENTIONED = 'MessageMentioned',

  // 私聊
  PRIVATE_MESSAGE_NEW = 'PrivateMessageNew',
  PRIVATE_MESSAGE_EDITED = 'PrivateMessageEdited',
  PRIVATE_MESSAGE_RECALLED = 'PrivateMessageRecalled',
  PRIVATE_MESSAGE_READ = 'PrivateMessageRead',
  PRIVATE_TYPING = 'PrivateTyping',

  // 群聊
  GROUP_MESSAGE_NEW = 'GroupMessageNew',
  GROUP_MESSAGE_EDITED = 'GroupMessageEdited',
  GROUP_MESSAGE_RECALLED = 'GroupMessageRecalled',
  GROUP_MESSAGE_READ_COUNT_UPDATED = 'GroupMessageReadCountUpdated',
  GROUP_MEMBER_ROLE_CHANGED = 'GroupMemberRoleChanged',
  GROUP_TYPING = 'GroupTyping',

  // 群组
  GROUP_MEMBER_JOINED = 'GroupMemberJoined',
  GROUP_MEMBER_LEFT = 'GroupMemberLeft',
  GROUP_MEMBER_REMOVED = 'GroupMemberRemoved',
  GROUP_MEMBER_MUTED = 'GroupMemberMuted',
  GROUP_MEMBER_UNMUTED = 'GroupMemberUnmuted',
  GROUP_INVITE_RECEIVED = 'GroupInviteReceived',
  GROUP_INVITE_REVOKED = 'GroupInviteRevoked',
  GROUP_INVITE_EXPIRED = 'GroupInviteExpired',
  GROUP_INVITE_REJECTED = 'GroupInviteRejected',
  GROUP_INVITE_RESPONSE = 'GroupInviteResponse',
  GROUP_APPLICATION_RECEIVED = 'GroupApplicationReceived',
  GROUP_APPLICATION_CANCELLED = 'GroupApplicationCancelled',
  GROUP_APPLICATION_EXPIRED = 'GroupApplicationExpired',
  GROUP_APPLICATION_REJECTED = 'GroupApplicationRejected',
  GROUP_APPLICATION_RESPONSE = 'GroupApplicationResponse',
  GROUP_ANNOUNCEMENT_CREATED = 'GroupAnnouncementCreated',
  GROUP_UPDATED = 'GroupUpdated',
  GROUP_DISMISSED = 'GroupDismissed',

  // 用户
  USER_PROFILE_UPDATED = 'UserProfileUpdated',

  // 系统
  SYSTEM_ANNOUNCEMENT = 'SystemAnnouncement',
  ACCOUNT_BANNED = 'AccountBanned',
  ERROR = 'Error',
  SUBSCRIBE_ACK = 'SubscribeAck',
  SUBSCRIPTIONS_SNAPSHOT = 'SubscriptionsSnapshot',

  // 举报
  REPORT_RESOLVED = 'ReportResolved',
  REPORT_DISMISSED = 'ReportDismissed',

  // 附近发现
  NEARBY_NEW_MOMENT = 'NearbyNewMoment'
}

// ============================================================================
// 客户端发送事件定义
// ============================================================================

export interface AuthPayload {
  client_type: ClientType
  access_token?: string
  device_id?: string     // 设备标识（spec：App 端必填，支撑单设备单会话模型）
  resume_token?: string   // 重连时透传上次连接的 resume_token（spec L155），尝试增量恢复
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
  client_request_id: string
  operation: SubscriptionOperation
  items: SubscriptionItem[]
}

export interface SetSubscriptionsEvent {
  type: 'SetSubscriptions'
  payload: SetSubscriptionsPayload
}

export interface GetSubscriptionsPayload {
  client_request_id: string
}

export interface GetSubscriptionsEvent {
  type: 'GetSubscriptions'
  payload: GetSubscriptionsPayload
}

// 客户端发送版 payload（与服务端推送版 PrivateTypingPayload 结构不同，故拆分命名，避免同名冲突）
export interface PrivateTypingSendPayload {
  target_user_id: string
  is_typing: boolean
}

export interface PrivateTypingSendEvent {
  type: 'SendPrivateTyping'
  payload: PrivateTypingSendPayload
}

// 客户端发送版 payload（与服务端推送版 GroupTypingPayload 结构不同，故拆分命名，避免同名冲突）
export interface GroupTypingSendPayload {
  group_id: string
  is_typing: boolean
}

export interface GroupTypingSendEvent {
  type: 'SendGroupTyping'
  payload: GroupTypingSendPayload
}

// ============================================================================
// 服务端推送事件定义
// ============================================================================

export interface ServerWebSocketEvent {
  type: ServerEventType
  payload: Object
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

export interface ConnectionEstablishedPayload {
  connection_id: string
  user_id: string
  server_time: string
  resume_token: string
  // server_seq 为 string（非 number）：ArkTS/JS 的 number 是 IEEE754 double，无法
  // 无损表示后端 int64（安全整数上限 2^53），spec L330 契约统一字符串跨端透传/去重，
  // 客户端不做数值运算。后端 int64 经 JSON `,string` 序列化为字符串下发。
  server_seq: string
  config: ConnectionConfig
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

// ----- 会话事件 -----

// 会话被驱逐/撤销（单设备单会话模型下，同设备新登录或 admin 撤销触发）
// 客户端收到后必须：清本地用户态 + 断开连接 + 不自动重连（提示"账号在其他设备登录"或"会话已失效"）
export interface SessionRevokedPayload {
  reason: 'device_evicted' | 'revoked_by_user' | 'password_changed'
  revoked_at: string   // UTC ISO 8601
}

export interface SessionRevokedEvent {
  type: 'SessionRevoked'
  payload: SessionRevokedPayload
}
// 注：服务端推送此事件后以 close code 4001 断开连接（见 spec「Close Code 规范」）

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
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  actor: UserInfo
  followed_at: string
  source?: FollowSource
  source_moment_id?: string
}

export interface NewFollowerEvent {
  type: 'NewFollower'
  payload: NewFollowerPayload
}

export interface MutualFollowPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  actor: UserInfo
  mutual_at: string
}

export interface MutualFollowEvent {
  type: 'MutualFollow'
  payload: MutualFollowPayload
}

// ----- 生活瞬间互动事件 -----

// 表情反应（统一 message/moment/comment 三类目标）
export interface ReactionAddedPayload {
  notify_id?: string   // 仅当 actor ≠ 资源作者时携带（自我反应不通知）；自反应时省略
  target_type: 'message' | 'moment' | 'comment'
  target_id: string
  emoji: string
  actor: UserInfo
  reacted_at: string
}

export interface ReactionAddedEvent {
  type: 'ReactionAdded'
  payload: ReactionAddedPayload
}

// 反应移除不产生通知（非持久化通知），无 notify_id
export interface ReactionRemovedPayload {
  target_type: 'message' | 'moment' | 'comment'
  target_id: string
  emoji: string
  actor: UserInfo
  removed_at: string
}

export interface ReactionRemovedEvent {
  type: 'ReactionRemoved'
  payload: ReactionRemovedPayload
}

export interface MomentCommentedPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  moment_id: string
  moment_summary: MomentSummary
  comment_id: string
  actor: UserInfo
  content: string
  commented_at: string
}

export interface MomentCommentedEvent {
  type: 'MomentCommented'
  payload: MomentCommentedPayload
}

export interface CommentReplyPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  moment_id: string
  moment_summary: MomentSummary
  comment_id: string
  replied_comment_preview: string
  reply_id: string
  actor: UserInfo
  content: string
  replied_at: string
}

export interface CommentReplyEvent {
  type: 'CommentReply'
  payload: CommentReplyPayload
}

export interface MessageMentionedPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  target_type: 'moment' | 'comment'
  moment_id: string
  comment_id?: string
  moment_summary: MomentSummary
  comment_preview?: string
  actor: UserInfo
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
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
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
  banned_until?: string
}

export interface AccountBannedEvent {
  type: 'AccountBanned'
  payload: AccountBannedPayload
}

// ----- 聊天消息事件 -----

export interface PrivateMessageNewPayload {
  message_id: string
  target_user_id: string
  sender: UserInfo
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
  recalled_by: UserInfo
  recalled_at: string
}

export interface PrivateMessageRecalledEvent {
  type: 'PrivateMessageRecalled'
  payload: PrivateMessageRecalledPayload
}

export interface PrivateMessageReadPayload {
  up_to_message_id: string
  read_at: string
}

export interface PrivateMessageReadEvent {
  type: 'PrivateMessageRead'
  payload: PrivateMessageReadPayload
}

// 服务端推送版 payload（与客户端发送版 PrivateTypingSendPayload 结构不同）
export interface PrivateTypingPayload {
  target_user_id: string   // 输入状态所属私聊会话的对端（订阅者据此定位会话）
  actor: UserInfo          // 正在打字的用户（统一用 actor: UserInfo 表达"动作发起者"）
  is_typing: boolean
}

export interface PrivateTypingEvent {
  type: 'PrivateTyping'
  payload: PrivateTypingPayload
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
  recalled_by: UserInfo
  recalled_at: string
}

export interface GroupMessageRecalledEvent {
  type: 'GroupMessageRecalled'
  payload: GroupMessageRecalledPayload
}

// 群消息已读数变更（服务端节流聚合推送，仅用于 UI「已有 N 人已读」刷新）
export interface GroupMessageReadCountUpdatedPayload {
  group_id: string
  message_id: string
  read_count: number
  updated_at: string
}

export interface GroupMessageReadCountUpdatedEvent {
  type: 'GroupMessageReadCountUpdated'
  payload: GroupMessageReadCountUpdatedPayload
}

// 服务端推送版 payload（与客户端发送版 GroupTypingSendPayload 结构不同）
export interface GroupTypingPayload {
  group_id: string
  actor: UserInfo   // 正在打字的成员（统一 actor: UserInfo）
  is_typing: boolean
}

export interface GroupTypingEvent {
  type: 'GroupTyping'
  payload: GroupTypingPayload
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

// 成员角色变更（升/降级 admin）
export interface GroupMemberRoleChangedPayload {
  group_id: string
  user: UserInfo
  old_role: 'owner' | 'admin' | 'member'
  new_role: 'owner' | 'admin' | 'member'
  changed_by: UserInfo
  changed_at: string
}

export interface GroupMemberRoleChangedEvent {
  type: 'GroupMemberRoleChanged'
  payload: GroupMemberRoleChangedPayload
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
  removed_by: UserInfo
  reason?: 'kicked' | 'banned'
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
  muted_by: UserInfo
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
  unmuted_by: UserInfo
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

export interface GroupUpdatedFields {
  name?: string
  avatar_url?: string
  description?: string
  tags?: string[]
  join_method?: GroupJoinMethod
  is_searchable?: boolean
  max_members?: number
  has_password?: boolean
  location?: Location
}

export interface GroupUpdatedPayload {
  group_id: string
  updated_fields: GroupUpdatedFields
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
  reason?: 'owner_dismissed' | 'policy_violation'
}

export interface GroupDismissedEvent {
  type: 'GroupDismissed'
  payload: GroupDismissedPayload
}

// ----- 附近发现事件 -----

export interface NearbyNewMomentPayload {
  notify_id: string
  moment_id: string
  moment_summary: MomentSummary
  actor: UserInfo
  distance: number
  created_at: string
}

export interface NearbyNewMomentEvent {
  type: 'NearbyNewMoment'
  payload: NearbyNewMomentPayload
}

export interface GroupInviteReceivedPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
  invite_id: string
  group: GroupInfo
  inviter: UserInfo
  message?: string
  created_at: string
}

export interface GroupInviteReceivedEvent {
  type: 'GroupInviteReceived'
  payload: GroupInviteReceivedPayload
}

export interface GroupInviteResponsePayload {
  // 仅邀请被接受时推送（拒绝不推送）。原 status:'accepted' 单值字段已移除——事件名即语义
  invite_id: string
  group_id: string
  user: UserInfo       // 接受邀请的用户
  responded_at: string
}

export interface GroupInviteResponseEvent {
  type: 'GroupInviteResponse'
  payload: GroupInviteResponsePayload
}

// 邀请被受邀人拒绝 → 推送给邀请发起方。纯实时事件，不落库通知（spec 无 notify_id）
export interface GroupInviteRejectedPayload {
  invite_id: string
  group_id: string
  rejected_by: UserInfo     // 拒绝邀请的受邀人（邀请发起方可见其身份，群邀请为双向契约）
  rejected_at: string
  reason?: string
}

export interface GroupInviteRejectedEvent {
  type: 'GroupInviteRejected'
  payload: GroupInviteRejectedPayload
}

// 邀请被发起方（管理员）主动撤销 → 推送给受邀人（纯实时，无 notify_id）
export interface GroupInviteRevokedPayload {
  invite_id: string
  group_id: string
  revoked_by: UserInfo   // 撤销邀请的管理员
  revoked_at: string
  reason?: string
}

export interface GroupInviteRevokedEvent {
  type: 'GroupInviteRevoked'
  payload: GroupInviteRevokedPayload
}

// 邀请过期（超 expires_at 自动失效）→ 推送给受邀人（纯实时，无 notify_id）
export interface GroupInviteExpiredPayload {
  invite_id: string
  group_id: string
  expired_at: string
}

export interface GroupInviteExpiredEvent {
  type: 'GroupInviteExpired'
  payload: GroupInviteExpiredPayload
}

export interface GroupApplicationReceivedPayload {
  notify_id: string   // 与 notify/list 的 NotifyItem.notify_id 同主键
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

// 申请被发起方（申请人）主动取消 → 推送给群管理员（纯实时，无 notify_id）
export interface GroupApplicationCancelledPayload {
  application_id: string
  group_id: string
  applicant: UserInfo
  cancelled_at: string
  reason?: string
}

export interface GroupApplicationCancelledEvent {
  type: 'GroupApplicationCancelled'
  payload: GroupApplicationCancelledPayload
}

// 申请过期（超 expires_at 自动失效）→ 推送给群管理员（纯实时，无 notify_id）
export interface GroupApplicationExpiredPayload {
  application_id: string
  group_id: string
  applicant: UserInfo
  expired_at: string
}

export interface GroupApplicationExpiredEvent {
  type: 'GroupApplicationExpired'
  payload: GroupApplicationExpiredPayload
}

// 申请被群管理员拒绝 → 推送给申请人
export interface GroupApplicationRejectedPayload {
  notify_id: string
  application_id: string
  group: GroupInfo
  rejected_by: UserInfo   // 处理管理员（申请人可见管理员身份）
  rejected_at: string
  reason?: string
}

export interface GroupApplicationRejectedEvent {
  type: 'GroupApplicationRejected'
  payload: GroupApplicationRejectedPayload
}

export interface GroupApplicationResponsePayload {
  // 仅申请被接受时推送。原 status:'accepted' 单值字段已移除——事件名即语义
  notify_id: string
  application_id: string
  group: GroupInfo
  responded_at: string
}

export interface GroupApplicationResponseEvent {
  type: 'GroupApplicationResponse'
  payload: GroupApplicationResponsePayload
}

// ----- 举报事件 -----

// 举报处理结果（推送给举报人）
export interface ReportResolvedPayload {
  notify_id: string
  report_id: string
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group'
  target_id: string
  resolved_at: string
  note?: string               // 管理员处理说明（脱敏后）
}

export interface ReportResolvedEvent {
  type: 'ReportResolved'
  payload: ReportResolvedPayload
}

export interface ReportDismissedPayload {
  notify_id: string
  report_id: string
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group'
  target_id: string
  dismissed_at: string
  note?: string               // 管理员处理说明（脱敏后）
}

export interface ReportDismissedEvent {
  type: 'ReportDismissed'
  payload: ReportDismissedPayload
}

// ----- 系统/订阅事件 -----

export interface SubscribeAckPayload {
  client_request_id: string
  operation: SubscriptionOperation
  accepted: SubscriptionItem[]
  code: number
  message?: string
  failed_ids?: FailedSubscriptionId[]
}

export interface SubscribeAckEvent {
  type: 'SubscribeAck'
  payload: SubscribeAckPayload
}

export interface SubscriptionsSnapshotPayload {
  client_request_id: string
  user_ids: string[]
  group_ids: string[]
}

export interface SubscriptionsSnapshotEvent {
  type: 'SubscriptionsSnapshot'
  payload: SubscriptionsSnapshotPayload
}

export interface ErrorContext {
  operation?: string
  resource_type?: string
  resource_id?: string
}

export interface ErrorPayload {
  code: number
  message: string
  severity: 'fatal' | 'recoverable' | 'warning'
  retryable: boolean
  retry_after?: number
  context?: ErrorContext
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
  | PrivateTypingSendEvent
  | GroupTypingSendEvent

export type ServerEvent =
  | ConnectionEstablishedEvent
  | PongEvent
  | SessionRevokedEvent
  | UserOnlineEvent
  | UserOfflineEvent
  | NewFollowerEvent
  | MutualFollowEvent
  | ReactionAddedEvent
  | ReactionRemovedEvent
  | MomentCommentedEvent
  | CommentReplyEvent
  | MessageMentionedEvent
  | UserProfileUpdatedEvent
  | SystemAnnouncementEvent
  | AccountBannedEvent
  | PrivateMessageNewEvent
  | PrivateMessageEditedEvent
  | PrivateMessageRecalledEvent
  | PrivateMessageReadEvent
  | PrivateTypingEvent
  | GroupMessageNewEvent
  | GroupMessageEditedEvent
  | GroupMessageRecalledEvent
  | GroupMessageReadCountUpdatedEvent
  | GroupTypingEvent
  | GroupMemberJoinedEvent
  | GroupMemberRoleChangedEvent
  | GroupMemberLeftEvent
  | GroupMemberRemovedEvent
  | GroupMemberMutedEvent
  | GroupMemberUnmutedEvent
  | GroupInviteReceivedEvent
  | GroupInviteRevokedEvent
  | GroupInviteExpiredEvent
  | GroupInviteRejectedEvent
  | GroupInviteResponseEvent
  | GroupApplicationReceivedEvent
  | GroupApplicationCancelledEvent
  | GroupApplicationExpiredEvent
  | GroupApplicationRejectedEvent
  | GroupApplicationResponseEvent
  | GroupAnnouncementCreatedEvent
  | GroupUpdatedEvent
  | GroupDismissedEvent
  | ReportResolvedEvent
  | ReportDismissedEvent
  | NearbyNewMomentEvent
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

export function createAuthEvent(clientType: ClientType, accessToken?: string, resumeToken?: string, deviceId?: string): AuthEvent {
  return {
    type: 'Auth',
    payload: {
      client_type: clientType,
      access_token: accessToken,
      device_id: deviceId,
      resume_token: resumeToken
    }
  }
}

export function createSetSubscriptionsEvent(
  operation: SubscriptionOperation,
  items: SubscriptionItem[],
  clientMsgId?: string
): SetSubscriptionsEvent {
  return {
    type: 'SetSubscriptions',
    payload: {
      client_request_id: clientMsgId ?? generateClientMsgId(),
      operation: operation,
      items: items
    }
  }
}

export function createGetSubscriptionsEvent(clientMsgId?: string): GetSubscriptionsEvent {
  return {
    type: 'GetSubscriptions',
    payload: {
      client_request_id: clientMsgId ?? generateClientMsgId()
    }
  }
}

function generateClientMsgId(): string {
  return `${Date.now()}-${Math.floor(Math.random() * 1e9)}`
}

export function createPrivateTypingEvent(
  target_user_id: string,
  is_typing: boolean
): PrivateTypingSendEvent {
  return {
    type: 'SendPrivateTyping',
    payload: { target_user_id, is_typing }
  }
}

export function createGroupTypingEvent(
  group_id: string,
  is_typing: boolean
): GroupTypingSendEvent {
  return {
    type: 'SendGroupTyping',
    payload: { group_id, is_typing }
  }
}

export function parseServerEvent(data: string): ServerEvent | null {
  try {
    return (JSON.parse(data) as Object) as ServerEvent
  } catch (e) {
    hilog.error(DOMAIN_NUMBER, TAG, `Failed to parse WebSocket message: ${e}`)
    return null
  }
}

export function serializeClientEvent(event: ClientEvent): string {
  return JSON.stringify(event)
}
