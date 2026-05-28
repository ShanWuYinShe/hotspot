/**
 * WebSocket 类型定义
 * 严格遵循 @spec/api_spec_websocket.md
 */

import type {
  UserInfo,
  GroupInfo,
  GroupAnnouncementItem,
  SystemAnnouncementItem,
  Location,
  MessageMedia,
  MessageReplyTo,
  MessageType,
  MomentSummary,
  ClientType,
  FollowSource,
  GroupJoinMethod,
  GroupRole,
  UserSettings,
} from "@/lib/types/api";

// ============ 基础类型 ============

export type SubscriptionOperation = "subscribe" | "unsubscribe" | "set";
export type SubscriptionFailureReason =
  | "invalid"
  | "denied"
  | "limit_exceeded"
  | "error";
export type ErrorSeverity = "fatal" | "recoverable" | "warning";

// ============ WebSocket 事件类型枚举 ============

export type ServerEventType =
  // 连接管理
  | "ConnectionEstablished"
  | "Pong"
  // 用户状态
  | "UserOnline"
  | "UserOffline"
  // 关注关系
  | "NewFollower"
  | "MutualFollow"
  // 生活瞬间互动
  | "MomentLiked"
  | "MomentCommented"
  | "CommentLiked"
  | "CommentReply"
  | "MessageMentioned"
  // 私聊消息
  | "PrivateMessageNew"
  | "PrivateMessageEdited"
  | "PrivateMessageRecalled"
  | "PrivateMessageRead"
  | "PrivateUserTyping"
  // 群聊消息
  | "GroupMessageNew"
  | "GroupMessageEdited"
  | "GroupMessageRecalled"
  | "GroupUserTyping"
  // 群组管理
  | "GroupMemberJoined"
  | "GroupMemberLeft"
  | "GroupMemberRemoved"
  | "GroupMemberMuted"
  | "GroupMemberUnmuted"
  | "GroupInviteReceived"
  | "GroupInviteResponse"
  | "GroupApplicationReceived"
  | "GroupApplicationResponse"
  | "GroupAnnouncementCreated"
  | "GroupUpdated"
  | "GroupDismissed"
  // 用户
  | "UserProfileUpdated"
  // 系统
  | "SystemAnnouncement"
  | "AccountBanned"
  | "Error"
  | "SubscribeAck"
  | "SubscriptionsSnapshot"
  // 多端同步（⏳ 延期实现）
  | "SelfReadSync"
  | "SelfChatSettingsSync"
  | "SelfSettingsUpdated"
  | "SelfRelationSync";

export type ClientEventType =
  | "Auth"
  | "Ping"
  | "SetSubscriptions"
  | "GetSubscriptions"
  | "PrivateTyping"
  | "GroupTyping";

// ============ 通用事件结构 ============

export interface WebSocketEvent<TType extends string, TPayload = unknown> {
  type: TType;
  payload: TPayload;
}

export type ServerWebSocketEvent = WebSocketEvent<ServerEventType, unknown>;

// ============ 辅助类型 ============

export type { MessageReplyTo } from "@/lib/types/api";

export interface FailedSubscriptionId {
  subscription_id: string;
  reason: SubscriptionFailureReason;
}

// ============ 服务端推送事件 ============

export interface ConnectionEstablishedPayload {
  connection_id: string;
  user_id: string;
  server_time: string;
  config: {
    ping_interval: number;
    ping_timeout: number;
    max_subscriptions: {
      user: number;
      group: number;
    };
  };
}

export interface PongPayload {}

export interface UserOnlinePayload {
  user_id: string;
  online_at: string;
}

export interface UserOfflinePayload {
  user_id: string;
  offline_at: string;
}

export interface NewFollowerPayload {
  actor: UserInfo;
  followed_at: string;
  source?: FollowSource;
  source_moment_id?: string;
}

export interface MutualFollowPayload {
  actor: UserInfo;
  mutual_at: string;
}

export interface MomentLikedPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  actor: UserInfo;
  liked_at: string;
}

export interface MomentCommentedPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  comment_id: string;
  actor: UserInfo;
  content: string;
  commented_at: string;
}

export interface CommentLikedPayload {
  comment_id: string;
  moment_id: string;
  moment_summary: MomentSummary;
  comment_preview: string;
  actor: UserInfo;
  liked_at: string;
}

export interface CommentReplyPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  comment_id: string;
  replied_comment_preview: string;
  reply_id: string;
  actor: UserInfo;
  content: string;
  replied_at: string;
}

export interface MessageMentionedPayload {
  target_type: "moment" | "comment";
  moment_id: string;
  comment_id?: string;
  moment_summary: MomentSummary;
  comment_preview?: string;
  actor: UserInfo;
  mentioned_at: string;
}

export type SystemAnnouncementPayload = SystemAnnouncementItem;

export interface UserProfileUpdatedPayload {
  user: UserInfo;
  updated_at: string;
}

export interface PrivateMessageNewPayload {
  message_id: string;
  target_user_id: string;
  sender: UserInfo;
  type: MessageType;
  content?: string;
  share_code?: string;
  reply_to?: MessageReplyTo;
  media?: MessageMedia;
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
  target_user_id: string;
  up_to_message_id: string;
  reader_id: string;
  read_at: string;
}

export interface PrivateUserTypingPayload {
  target_user_id: string;
  user_id: string;
  is_typing: boolean;
}

export interface GroupMessageNewPayload {
  message_id: string;
  group_id: string;
  sender: UserInfo;
  type: MessageType;
  content?: string;
  share_code?: string;
  reply_to?: MessageReplyTo;
  mentions?: string[];
  media?: MessageMedia;
  sent_at: string;
}

export interface GroupMessageEditedPayload {
  message_id: string;
  group_id: string;
  content: string;
  edited_at: string;
}

export interface GroupMessageRecalledPayload {
  message_id: string;
  group_id: string;
  recalled_by: UserInfo;
  recalled_at: string;
}

export interface GroupUserTypingPayload {
  group_id: string;
  user_id: string;
  is_typing: boolean;
}

export interface GroupMemberJoinedPayload {
  group_id: string;
  user: UserInfo;
  role: GroupRole;
  member_count: number;
  joined_at: string;
}

export interface GroupMemberLeftPayload {
  group_id: string;
  user: UserInfo;
  left_at: string;
  member_count: number;
}

export interface GroupMemberRemovedPayload {
  group_id: string;
  user: UserInfo;
  removed_by: UserInfo;
  reason?: "kicked" | "banned";
  removed_at: string;
  member_count: number;
}

export interface GroupMemberMutedPayload {
  group_id: string;
  user: UserInfo;
  muted_by: UserInfo;
  muted_until: string;
  duration_minutes: number;
  muted_at: string;
}

export interface GroupMemberUnmutedPayload {
  group_id: string;
  user: UserInfo;
  unmuted_by: UserInfo;
  unmuted_at: string;
}

export interface GroupInviteReceivedPayload {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  created_at: string;
}

export interface GroupInviteResponsePayload {
  invite_id: string;
  group_id: string;
  user: UserInfo;
  status: "accepted";
  responded_at: string;
}

export interface GroupApplicationReceivedPayload {
  application_id: string;
  group_id: string;
  applicant: UserInfo;
  message?: string;
  created_at: string;
}

export interface GroupApplicationResponsePayload {
  application_id: string;
  group: GroupInfo;
  status: "accepted";
  responded_at: string;
}

export interface GroupUpdatedPayload {
  group_id: string;
  updated_fields: {
    name?: string;
    avatar_url?: string;
    description?: string;
    tags?: string[];
    join_method?: GroupJoinMethod;
    is_searchable?: boolean;
    max_members?: number;
    has_password?: boolean;
    location?: Location;
  };
  updated_by: string;
  updated_at: string;
}

export interface GroupDismissedPayload {
  group_id: string;
  dismissed_at: string;
  reason?: "owner_dismissed" | "policy_violation";
}

export interface ErrorPayload {
  code: number;
  message: string;
  severity: ErrorSeverity;
  retryable: boolean;
  retry_after?: number;
  context?: {
    operation?: string;
    resource_type?: string;
    resource_id?: string;
  };
}

export interface SubscribeAckPayload {
  client_request_id: string;
  operation: SubscriptionOperation;
  type: "user" | "group";
  ids: string[];
  code: number;
  message?: string;
  failed_ids?: FailedSubscriptionId[];
}

export interface SubscriptionsSnapshotPayload {
  client_request_id: string;
  user_ids: string[];
  group_ids: string[];
}

export interface AccountBannedPayload {
  reason: string;
  duration_days: number;
  banned_at: string;
}

export interface SelfReadSyncPayload {
  chat_type: "private" | "group";
  target_id: string;
  up_to_message_id?: string;
}

export interface SelfChatSettingsSyncPayload {
  chat_type: "private" | "group";
  target_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface SelfSettingsUpdatedPayload {
  settings: UserSettings;
}

export interface SelfRelationSyncPayload {
  action: "follow" | "unfollow" | "block" | "unblock";
  target_user_id: string;
}

// ============ 客户端发送事件 ============

export interface AuthPayload {
  client_type: ClientType;
  access_token?: string;
}

export interface PingPayload {}

export interface SetSubscriptionsPayload {
  client_request_id: string;
  operation: SubscriptionOperation;
  type: "user" | "group";
  ids: string[];
}

export interface GetSubscriptionsPayload {
  client_request_id: string;
}

export interface PrivateTypingPayload {
  target_user_id: string;
  is_typing: boolean;
}

export interface GroupTypingPayload {
  group_id: string;
  is_typing: boolean;
}

// ============ 类型联合 ============

export type ServerEvent =
  | WebSocketEvent<"ConnectionEstablished", ConnectionEstablishedPayload>
  | WebSocketEvent<"Pong", PongPayload>
  | WebSocketEvent<"UserOnline", UserOnlinePayload>
  | WebSocketEvent<"UserOffline", UserOfflinePayload>
  | WebSocketEvent<"NewFollower", NewFollowerPayload>
  | WebSocketEvent<"MutualFollow", MutualFollowPayload>
  | WebSocketEvent<"MomentLiked", MomentLikedPayload>
  | WebSocketEvent<"MomentCommented", MomentCommentedPayload>
  | WebSocketEvent<"CommentLiked", CommentLikedPayload>
  | WebSocketEvent<"CommentReply", CommentReplyPayload>
  | WebSocketEvent<"MessageMentioned", MessageMentionedPayload>
  | WebSocketEvent<"SystemAnnouncement", SystemAnnouncementPayload>
  | WebSocketEvent<"UserProfileUpdated", UserProfileUpdatedPayload>
  | WebSocketEvent<"PrivateMessageNew", PrivateMessageNewPayload>
  | WebSocketEvent<"PrivateMessageEdited", PrivateMessageEditedPayload>
  | WebSocketEvent<"PrivateMessageRecalled", PrivateMessageRecalledPayload>
  | WebSocketEvent<"PrivateMessageRead", PrivateMessageReadPayload>
  | WebSocketEvent<"PrivateUserTyping", PrivateUserTypingPayload>
  | WebSocketEvent<"GroupMessageNew", GroupMessageNewPayload>
  | WebSocketEvent<"GroupMessageEdited", GroupMessageEditedPayload>
  | WebSocketEvent<"GroupMessageRecalled", GroupMessageRecalledPayload>
  | WebSocketEvent<"GroupUserTyping", GroupUserTypingPayload>
  | WebSocketEvent<"GroupMemberJoined", GroupMemberJoinedPayload>
  | WebSocketEvent<"GroupMemberLeft", GroupMemberLeftPayload>
  | WebSocketEvent<"GroupMemberRemoved", GroupMemberRemovedPayload>
  | WebSocketEvent<"GroupMemberMuted", GroupMemberMutedPayload>
  | WebSocketEvent<"GroupMemberUnmuted", GroupMemberUnmutedPayload>
  | WebSocketEvent<"GroupInviteReceived", GroupInviteReceivedPayload>
  | WebSocketEvent<"GroupInviteResponse", GroupInviteResponsePayload>
  | WebSocketEvent<"GroupApplicationReceived", GroupApplicationReceivedPayload>
  | WebSocketEvent<"GroupApplicationResponse", GroupApplicationResponsePayload>
  | WebSocketEvent<"GroupAnnouncementCreated", GroupAnnouncementItem>
  | WebSocketEvent<"GroupUpdated", GroupUpdatedPayload>
  | WebSocketEvent<"GroupDismissed", GroupDismissedPayload>
  | WebSocketEvent<"AccountBanned", AccountBannedPayload>
  | WebSocketEvent<"Error", ErrorPayload>
  | WebSocketEvent<"SubscribeAck", SubscribeAckPayload>
  | WebSocketEvent<"SubscriptionsSnapshot", SubscriptionsSnapshotPayload>
  | WebSocketEvent<"SelfReadSync", SelfReadSyncPayload>
  | WebSocketEvent<"SelfChatSettingsSync", SelfChatSettingsSyncPayload>
  | WebSocketEvent<"SelfSettingsUpdated", SelfSettingsUpdatedPayload>
  | WebSocketEvent<"SelfRelationSync", SelfRelationSyncPayload>;

export type ClientEvent =
  | WebSocketEvent<"Auth", AuthPayload>
  | WebSocketEvent<"Ping", PingPayload>
  | WebSocketEvent<"SetSubscriptions", SetSubscriptionsPayload>
  | WebSocketEvent<"GetSubscriptions", GetSubscriptionsPayload>
  | WebSocketEvent<"PrivateTyping", PrivateTypingPayload>
  | WebSocketEvent<"GroupTyping", GroupTypingPayload>;

// ============ WebSocket 事件类型映射 ============

export type WebSocketEventMap = {
  ConnectionEstablished: ConnectionEstablishedPayload;
  Pong: PongPayload;
  UserOnline: UserOnlinePayload;
  UserOffline: UserOfflinePayload;
  NewFollower: NewFollowerPayload;
  MutualFollow: MutualFollowPayload;
  MomentLiked: MomentLikedPayload;
  MomentCommented: MomentCommentedPayload;
  CommentLiked: CommentLikedPayload;
  CommentReply: CommentReplyPayload;
  MessageMentioned: MessageMentionedPayload;
  SystemAnnouncement: SystemAnnouncementPayload;
  UserProfileUpdated: UserProfileUpdatedPayload;
  PrivateMessageNew: PrivateMessageNewPayload;
  PrivateMessageEdited: PrivateMessageEditedPayload;
  PrivateMessageRecalled: PrivateMessageRecalledPayload;
  PrivateMessageRead: PrivateMessageReadPayload;
  PrivateUserTyping: PrivateUserTypingPayload;
  GroupMessageNew: GroupMessageNewPayload;
  GroupMessageEdited: GroupMessageEditedPayload;
  GroupMessageRecalled: GroupMessageRecalledPayload;
  GroupUserTyping: GroupUserTypingPayload;
  GroupMemberJoined: GroupMemberJoinedPayload;
  GroupMemberLeft: GroupMemberLeftPayload;
  GroupMemberRemoved: GroupMemberRemovedPayload;
  GroupMemberMuted: GroupMemberMutedPayload;
  GroupMemberUnmuted: GroupMemberUnmutedPayload;
  GroupInviteReceived: GroupInviteReceivedPayload;
  GroupInviteResponse: GroupInviteResponsePayload;
  GroupApplicationReceived: GroupApplicationReceivedPayload;
  GroupApplicationResponse: GroupApplicationResponsePayload;
  GroupAnnouncementCreated: GroupAnnouncementItem;
  GroupUpdated: GroupUpdatedPayload;
  GroupDismissed: GroupDismissedPayload;
  AccountBanned: AccountBannedPayload;
  Error: ErrorPayload;
  SubscribeAck: SubscribeAckPayload;
  SubscriptionsSnapshot: SubscriptionsSnapshotPayload;
  SelfReadSync: SelfReadSyncPayload;
  SelfChatSettingsSync: SelfChatSettingsSyncPayload;
  SelfSettingsUpdated: SelfSettingsUpdatedPayload;
  SelfRelationSync: SelfRelationSyncPayload;
};
