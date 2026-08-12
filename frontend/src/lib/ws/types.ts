// WebSocket 契约事件类型（spec/api_spec_websocket.md，只读权威）。
// 统一 envelope：所有服务端推送与客户端发送均使用 { type, payload } 结构。

/** 统一事件 Envelope。 */
export interface WsEvent<T extends string = string, P = unknown> {
  type: T;
  payload: P;
}

// ---------------------------------------------------------------------------
// 事件类型枚举
// ---------------------------------------------------------------------------

/** 客户端可发送的事件类型。 */
export type ClientEventType = 'Auth' | 'Ping' | 'SetSubscriptions' | 'SendPrivateTyping' | 'SendGroupTyping';

/** 服务端可推送的全部事件类型（与 spec ServerEventType 枚举对齐）。 */
export type ServerEventType =
  // 连接管理
  | 'ConnectionEstablished'
  | 'Pong'
  // 会话
  | 'SessionRevoked'
  // 用户状态
  | 'UserOnline'
  | 'UserOffline'
  // 关注关系
  | 'NewFollower'
  | 'MutualFollow'
  // 互动（含表情反应）
  | 'ReactionAdded'
  | 'ReactionRemoved'
  | 'MomentCommented'
  | 'CommentReply'
  | 'MessageMentioned'
  // 私聊
  | 'PrivateMessageNew'
  | 'PrivateMessageEdited'
  | 'PrivateMessageRecalled'
  | 'PrivateMessageRead'
  | 'PrivateTyping'
  // 群聊
  | 'GroupMessageNew'
  | 'GroupMessageEdited'
  | 'GroupMessageRecalled'
  | 'GroupMessageReadCountUpdated'
  | 'GroupTyping'
  // 群组管理
  | 'GroupMemberJoined'
  | 'GroupMemberLeft'
  | 'GroupMemberRemoved'
  | 'GroupMemberRoleChanged'
  | 'GroupMemberMuted'
  | 'GroupMemberUnmuted'
  | 'GroupInviteReceived'
  | 'GroupInviteRevoked'
  | 'GroupInviteExpired'
  | 'GroupInviteRejected'
  | 'GroupInviteResponse'
  | 'GroupApplicationReceived'
  | 'GroupApplicationCancelled'
  | 'GroupApplicationExpired'
  | 'GroupApplicationRejected'
  | 'GroupApplicationResponse'
  | 'GroupAnnouncementCreated'
  | 'GroupUpdated'
  | 'GroupDismissed'
  // 举报
  | 'ReportResolved'
  | 'ReportDismissed'
  // 附近动态
  | 'NearbyNewMoment'
  // 用户
  | 'UserProfileUpdated'
  // 系统
  | 'SystemAnnouncement'
  | 'AccountBanned'
  // 协议控制
  | 'Error'
  | 'SubscribeAck';

// ---------------------------------------------------------------------------
// 通用类型
// ---------------------------------------------------------------------------

/** 用户基础信息（对齐 api_spec_types.md 的 UserInfo）。 */
export interface UserInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
}

/** 客户端类型（Web 端固定 'web'，App 端 'app'）。 */
export type ClientType = 'web' | 'app';

export type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';

export interface SubscriptionItem {
  type: 'user' | 'group';
  ids: string[];
}

// ---------------------------------------------------------------------------
// 连接管理事件 payload
// ---------------------------------------------------------------------------

export interface ConnectionEstablishedConfig {
  ping_interval: number; // 心跳间隔（秒）
  ping_timeout: number; // Pong 超时（秒）
  max_subscriptions: {
    user: number;
    group: number;
  };
}

export interface ConnectionEstablishedPayload {
  connection_id: string;
  user_id: string;
  server_time: string; // UTC ISO 8601
  resume_token: string; // 重连时透传给 Auth 尝试增量恢复
  server_seq: number;
  config: ConnectionEstablishedConfig;
}

export type ErrorSeverity = 'fatal' | 'recoverable' | 'warning';

export interface ErrorPayload {
  code: number; // 应用层错误码（20000-20999 / 22000-22099）
  message: string;
  severity: ErrorSeverity;
  retryable: boolean;
  retry_after?: number; // 秒，仅 retryable=true 时有效
  context?: {
    operation?: string;
    resource_type?: string;
    resource_id?: string;
  };
}

export type SubscriptionFailureReason = 'invalid' | 'denied' | 'limit_exceeded' | 'error';

export interface FailedSubscriptionId {
  type: 'user' | 'group';
  subscription_id: string;
  reason: SubscriptionFailureReason;
}

export interface SubscribeAckPayload {
  client_request_id: string;
  operation: SubscriptionOperation;
  accepted: { type: 'user' | 'group'; ids: string[] }[];
  code: number; // 0 = 成功
  message?: string;
  failed_ids?: FailedSubscriptionId[];
}

export interface SessionRevokedPayload {
  reason: 'device_evicted' | 'revoked_by_user' | 'password_changed';
  revoked_at: string;
}

export interface AccountBannedPayload {
  notification_id: string;
  reason: string;
  duration_days: number; // 0 = 永久
  banned_at: string;
  banned_until?: string;
}

// ---------------------------------------------------------------------------
// 私聊消息事件 payload
// ---------------------------------------------------------------------------

export interface PrivateMessageNewPayload {
  message_id: string;
  target_user_id: string; // 接收者 ID
  sender: UserInfo;
  type: string; // MessageType
  content?: string;
  share_code?: string;
  reply_to?: unknown;
  media?: unknown;
  sent_at: string;
}

export interface PrivateMessageEditedPayload {
  message_id: string;
  target_user_id: string;
  content: string;
  edited_at: string;
}

export interface PrivateMessageRecalledPayload {
  message_id: string;
  target_user_id: string;
  recalled_by: UserInfo;
  recalled_at: string;
}

export interface PrivateMessageReadPayload {
  up_to_message_id: string;
  read_at: string;
}

/** 服务端推送版输入状态（与客户端发送版 PrivateTypingSendPayload 拆分命名）。 */
export interface PrivateTypingPayload {
  target_user_id: string;
  actor: UserInfo;
  is_typing: boolean;
}

// ---------------------------------------------------------------------------
// 通知类事件 payload（核心几个按 spec 定义；其余用宽松对象）
// ---------------------------------------------------------------------------

export interface NewFollowerPayload {
  notification_id: string;
  actor: UserInfo;
  followed_at: string;
  source?: string; // FollowSource: 'moment' | 'search' | 'recommendation' | 'profile' | 'group'
  source_moment_id?: string;
}

export interface MomentCommentedPayload {
  notification_id: string;
  moment_id: string;
  comment_id: string;
  actor: UserInfo;
  content: string;
  commented_at: string;
  [key: string]: unknown; // moment_summary 等剩余字段宽松透传
}

export interface GroupInviteReceivedPayload {
  notification_id: string;
  invite_id: string;
  group: unknown; // GroupInfo
  inviter: UserInfo;
  message?: string;
  created_at: string;
}

export interface GroupInviteRejectedPayload {
  notification_id: string;
  invite_id: string;
  group_id: string;
  rejected_by: UserInfo;
  rejected_at: string;
  reason?: string;
}

export interface ReportResolvedPayload {
  notification_id: string;
  report_id: string;
  target_type: string; // ReportTargetType
  target_id: string;
  resolved_at: string;
  note?: string;
}

export interface SystemAnnouncementPayload {
  notification_id: string;
  announcement_id: string;
  title: string;
  content: string;
  severity: string; // AnnounceLevel
  action_url?: string;
  created_at: string;
}

// ---------------------------------------------------------------------------
// 客户端发送事件 payload
// ---------------------------------------------------------------------------

/** Web 端不传 access_token/device_id（由 HttpOnly Cookie 自动鉴权）。 */
export interface AuthPayload {
  client_type: ClientType;
  resume_token?: string; // 重连时携带上次连接的 resume_token
}

export type PingPayload = Record<string, never>;

export interface SetSubscriptionsPayload {
  client_request_id: string;
  operation: SubscriptionOperation;
  items: SubscriptionItem[];
}

export interface PrivateTypingSendPayload {
  target_user_id: string;
  is_typing: boolean;
}

export interface GroupTypingSendPayload {
  group_id: string;
  is_typing: boolean;
}

// ---------------------------------------------------------------------------
// type → payload 映射（驱动 emitter 的按类型收窄）
// ---------------------------------------------------------------------------

export interface ServerEventPayloadMap {
  ConnectionEstablished: ConnectionEstablishedPayload;
  Pong: PingPayload;
  Error: ErrorPayload;
  SubscribeAck: SubscribeAckPayload;
  SessionRevoked: SessionRevokedPayload;
  AccountBanned: AccountBannedPayload;
  PrivateMessageNew: PrivateMessageNewPayload;
  PrivateMessageEdited: PrivateMessageEditedPayload;
  PrivateMessageRecalled: PrivateMessageRecalledPayload;
  PrivateMessageRead: PrivateMessageReadPayload;
  PrivateTyping: PrivateTypingPayload;
  NewFollower: NewFollowerPayload;
  MomentCommented: MomentCommentedPayload;
  GroupInviteReceived: GroupInviteReceivedPayload;
  GroupInviteRejected: GroupInviteRejectedPayload;
  ReportResolved: ReportResolvedPayload;
  SystemAnnouncement: SystemAnnouncementPayload;
  // 其余事件按需宽松（避免过度工程）
  UserOnline: Record<string, unknown>;
  UserOffline: Record<string, unknown>;
  MutualFollow: Record<string, unknown>;
  ReactionAdded: Record<string, unknown>;
  ReactionRemoved: Record<string, unknown>;
  CommentReply: Record<string, unknown>;
  MessageMentioned: Record<string, unknown>;
  GroupMessageNew: Record<string, unknown>;
  GroupMessageEdited: Record<string, unknown>;
  GroupMessageRecalled: Record<string, unknown>;
  GroupMessageReadCountUpdated: Record<string, unknown>;
  GroupTyping: Record<string, unknown>;
  GroupMemberJoined: Record<string, unknown>;
  GroupMemberLeft: Record<string, unknown>;
  GroupMemberRemoved: Record<string, unknown>;
  GroupMemberRoleChanged: Record<string, unknown>;
  GroupMemberMuted: Record<string, unknown>;
  GroupMemberUnmuted: Record<string, unknown>;
  GroupInviteRevoked: Record<string, unknown>;
  GroupInviteExpired: Record<string, unknown>;
  GroupInviteResponse: Record<string, unknown>;
  GroupApplicationReceived: Record<string, unknown>;
  GroupApplicationCancelled: Record<string, unknown>;
  GroupApplicationExpired: Record<string, unknown>;
  GroupApplicationRejected: Record<string, unknown>;
  GroupApplicationResponse: Record<string, unknown>;
  GroupAnnouncementCreated: Record<string, unknown>;
  GroupUpdated: Record<string, unknown>;
  GroupDismissed: Record<string, unknown>;
  ReportDismissed: Record<string, unknown>;
  NearbyNewMoment: Record<string, unknown>;
  UserProfileUpdated: Record<string, unknown>;
}

export interface ClientEventPayloadMap {
  Auth: AuthPayload;
  Ping: PingPayload;
  SetSubscriptions: SetSubscriptionsPayload;
  SendPrivateTyping: PrivateTypingSendPayload;
  SendGroupTyping: GroupTypingSendPayload;
}

/** 服务端推送事件（type 强约束为 ServerEventType 的判别联合）。 */
export type ServerWsEvent = {
  [K in ServerEventType]: WsEvent<K, ServerEventPayloadMap[K]>;
}[ServerEventType];

/** 客户端发送事件（type 强约束为 ClientEventType 的判别联合）。 */
export type ClientWsEvent = {
  [K in ClientEventType]: WsEvent<K, ClientEventPayloadMap[K]>;
}[ClientEventType];
