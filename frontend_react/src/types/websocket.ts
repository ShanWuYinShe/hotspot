/**
 * WebSocket 实时通信类型定义
 * 严格对应后端 API 规范
 */

import type {
  UserInfo,
  GroupSettings,
  MessageType,
  ReactionType,
  MessageMedia,
  JoinedGroupItem,
  MessageReplyTo,
  Location,
} from './api';

// ============================================================================
// 基础类型
// ============================================================================

export interface WebSocketEvent<T = unknown> {
  type: string;
  payload: T;
}

export type WebSocketState = 'connecting' | 'connected' | 'disconnected' | 'error';

// ============================================================================
// 订阅相关
// ============================================================================

export type SubscriptionItem =
  | {
      type: 'group';
      ids: string[];
    }
  | {
      type: 'user';
      ids: number[];
    };

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
  | 'GroupTyping';

export interface AuthEvent {
  type: 'Auth';
  payload: {
    access_token: string;
    device_fingerprint?: string;
  };
}

export interface PingEvent {
  type: 'Ping';
  payload: {
    timestamp: string;
  };
}

export interface SubscribeEvent {
  type: 'Subscribe';
  payload: {
    items: SubscriptionItem[];
  };
}

export interface UnsubscribeEvent {
  type: 'Unsubscribe';
  payload: {
    items: SubscriptionItem[];
  };
}

export interface SetSubscriptionsEvent {
  type: 'SetSubscriptions';
  payload: {
    items: SubscriptionItem[];
  };
}

export interface PrivateTypingEvent {
  type: 'PrivateTyping';
  payload: {
    target_user_id: number;
    is_typing: boolean;
  };
}

export interface GroupTypingEvent {
  type: 'GroupTyping';
  payload: {
    group_chat_id: string;
    is_typing: boolean;
  };
}

export type ClientEvent =
  | AuthEvent
  | PingEvent
  | SubscribeEvent
  | UnsubscribeEvent
  | SetSubscriptionsEvent
  | PrivateTypingEvent
  | GroupTypingEvent;

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
  | 'FollowRequestReceived'
  | 'FollowRequestHandled'
  // 生活瞬间
  | 'MomentLiked'
  | 'MomentCommented'
  | 'MomentMentioned'
  | 'CommentMentioned'
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
  // 群组管理
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
  | 'GroupAnnouncementCreated'
  | 'GroupAnnouncementUpdated'
  | 'GroupAnnouncementDeleted'
  // 多端同步
  | 'PrivateChatDeleted'
  | 'PrivateChatSettingsUpdated'
  | 'NotificationReadStateUpdated'
  // 系统
  | 'Error'
  | 'SubscribeAck';

// 连接管理事件
export interface ConnectionEstablishedPayload {
  connection_id: string;
  user_id: number;
  server_time: string;
}

export interface PongPayload {
  timestamp: string;
}

// 用户状态事件
export interface UserOnlinePayload {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  online_at: string;
}

export interface UserOfflinePayload {
  user_id: number;
  offline_at: string;
}

export interface UserLocationUpdatePayload {
  user_id: number;
  location: Location;
  updated_at: string;
}

export interface UserProfileUpdatedPayload {
  user_id: number;
  nickname?: string;
  avatar_url?: string;
  bio?: string;
  updated_at: string;
}

// 关注关系事件
export interface NewFollowerPayload {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
}

export interface MutualFollowPayload {
  user: UserInfo;
  mutual_at: string;
}

export interface UnfollowedPayload {
  user_id: number;
  unfollowed_at: string;
  was_mutual: boolean;
}

// 生活瞬间互动事件
export interface MomentSummary {
  moment_id: string;
  cover_url?: string;
  text_preview: string;
}

export interface MomentLikedPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  user: UserInfo;
  liked_at: string;
}

export interface MomentCommentedPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  comment_id: string;
  user: UserInfo;
  content: string;
  commented_at: string;
}

export interface CommentReplyPayload {
  moment_id: string;
  comment_id: string;
  reply_to_content: string;
  reply_id: string;
  user: UserInfo;
  content: string;
  replied_at: string;
}

export interface MomentMentionedPayload {
  moment_id: string;
  moment_summary: MomentSummary;
  user: UserInfo;
  mentioned_at: string;
}

export interface CommentMentionedPayload {
  moment_id: string;
  comment_id: string;
  user: UserInfo;
  content: string;
  mentioned_at: string;
}

export interface SystemAnnouncementPayload {
  announcement_id: string;
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  action_url?: string;
  created_at: string;
}

// 私聊消息事件
export interface PrivateMessageNewPayload {
  message_id: string;
  target_user_id: number;
  sender: UserInfo;
  type: MessageType;
  content?: string;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  media?: MessageMedia;
  location?: Location;
  location_address?: string;
  sent_at: string;
}

export interface PrivateMessageEditedPayload {
  message_id: string;
  target_user_id: number;
  content: string;
  edited_at: string;
}

export interface PrivateMessageRecalledPayload {
  message_id: string;
  target_user_id: number;
  recalled_by: number;
  recalled_at: string;
}

export interface PrivateMessageDeletedPayload {
  message_id: string;
  target_user_id: number;
  deleted_by: number;
  deleted_at: string;
}

export interface PrivateMessageReactionPayload {
  message_id: string;
  target_user_id: number;
  user: UserInfo;
  reaction_type: ReactionType;
  has_reaction: boolean;
  created_at: string;
}

export interface PrivateMessageReadPayload {
  target_user_id: number;
  up_to_message_id: string;
  reader_id: number;
  read_at: string;
}

export interface PrivateUserTypingPayload {
  target_user_id: number;
  user_id: number;
  is_typing: boolean;
}

// 群聊消息事件
export interface GroupMessageNewPayload {
  message_id: string;
  group_chat_id: string;
  sender: UserInfo;
  type: MessageType;
  content?: string;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  mentions?: number[];
  media?: MessageMedia;
  location?: Location;
  location_address?: string;
  sent_at: string;
}

export interface GroupMessageEditedPayload {
  message_id: string;
  group_chat_id: string;
  content: string;
  edited_at: string;
}

export interface GroupMessageRecalledPayload {
  message_id: string;
  group_chat_id: string;
  recalled_by: number;
  recalled_at: string;
}

export interface GroupMessageDeletedPayload {
  message_id: string;
  group_chat_id: string;
  deleted_by: number;
  for_everyone: boolean;
  deleted_at: string;
}

export interface GroupMessageReactionPayload {
  message_id: string;
  group_chat_id: string;
  user: UserInfo;
  reaction_type: ReactionType;
  has_reaction: boolean;
  created_at: string;
}

export interface GroupMessageReadPayload {
  group_chat_id: string;
  up_to_message_id: string;
  reader_id: number;
  read_at: string;
}

export interface GroupUserTypingPayload {
  group_chat_id: string;
  user_id: number;
  is_typing: boolean;
}

// 群组事件
export interface GroupMemberJoinedPayload {
  group_chat_id: string;
  user: UserInfo;
  role: 'owner' | 'admin' | 'member';
  member_count: number;
  joined_at: string;
}

export interface GroupMemberLeftPayload {
  group_chat_id: string;
  user_id: number;
  member_count: number;
  reason: 'left' | 'kicked' | 'banned';
  left_at: string;
}

export interface GroupUpdatedFields {
  name?: string;
  description?: string;
  avatar_url?: string;
  settings?: Partial<GroupSettings>;
}

export interface GroupProfileUpdatedPayload {
  group_chat_id: string;
  updated_fields: GroupUpdatedFields;
  updated_at: string;
}

export interface GroupMemberRoleChangedPayload {
  group_chat_id: string;
  user_id: number;
  old_role: 'owner' | 'admin' | 'member';
  new_role: 'owner' | 'admin' | 'member';
  changed_at: string;
}

export interface GroupMemberMutedPayload {
  group_chat_id: string;
  user_id: number;
  muted_until: string;
  muted_at: string;
}

export interface GroupMemberUnmutedPayload {
  group_chat_id: string;
  user_id: number;
  unmuted_at: string;
}

export interface GroupDismissedPayload {
  group_chat_id: string;
  dismissed_by: number;
  dismissed_at: string;
}

export interface GroupInviteReceivedPayload {
  invite_id: string;
  group_chat_id: string;
  group_name: string;
  group_avatar_url?: string;
  inviter: UserInfo;
  message?: string;
  created_at: string;
}

export interface GroupInviteResponsePayload {
  invite_id: string;
  group_chat_id: string;
  user: UserInfo;
  status: 'accepted' | 'rejected';
  responded_at: string;
}

export interface GroupApplicationReceivedPayload {
  application_id: string;
  group_chat_id: string;
  applicant: UserInfo;
  message?: string;
  created_at: string;
}

export interface GroupApplicationResponsePayload {
  application_id: string;
  group_chat_id: string;
  group_name: string;
  status: 'accepted' | 'rejected';
  responded_at: string;
}

export interface GroupAnnouncementCreatedPayload {
  announcement_id: string;
  group_chat_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  is_pinned: boolean;
  created_at: string;
}

export interface GroupAnnouncementUpdatedPayload {
  announcement_id: string;
  group_chat_id: string;
  title?: string;
  content?: string;
  is_pinned?: boolean;
  updated_at: string;
}

export interface GroupAnnouncementDeletedPayload {
  announcement_id: string;
  group_chat_id: string;
  deleted_at: string;
}

// 多端同步事件
export interface SelfJoinedGroupPayload {
  group: JoinedGroupItem;
  joined_at: string;
}

export interface SelfLeftGroupPayload {
  group_chat_id: string;
  left_at: string;
}

export interface PrivateChatDeletedPayload {
  target_user_id: number;
  deleted_at: string;
}

export interface PrivateChatSettingsUpdatedPayload {
  target_user_id: number;
  is_muted?: boolean;
  is_pinned?: boolean;
  updated_at: string;
}

export interface NotificationReadStateUpdatedPayload {
  total_unread_count: number;
  read_at: string;
}

// 订阅确认事件
export type SubscriptionFailureReason =
  | 'invalid_item'
  | 'not_found'
  | 'permission_denied'
  | 'too_many_subscriptions'
  | 'rate_limited'
  | 'internal_error';

export interface SubscribeAckPayload {
  operation: 'subscribe' | 'unsubscribe' | 'set';
  items: SubscriptionItem[];
  code: number;
  message?: string;
  failed_items?: (SubscriptionItem & { reason: SubscriptionFailureReason })[];
}

// 错误事件
export interface ErrorPayload {
  code: number;
  message: string;
}

// 服务端事件类型映射
export interface ServerEventPayloadMap {
  ConnectionEstablished: ConnectionEstablishedPayload;
  Pong: PongPayload;
  UserOnline: UserOnlinePayload;
  UserOffline: UserOfflinePayload;
  UserLocationUpdate: UserLocationUpdatePayload;
  UserProfileUpdated: UserProfileUpdatedPayload;
  NewFollower: NewFollowerPayload;
  MutualFollow: MutualFollowPayload;
  Unfollowed: UnfollowedPayload;
  MomentLiked: MomentLikedPayload;
  MomentCommented: MomentCommentedPayload;
  MomentMentioned: MomentMentionedPayload;
  CommentMentioned: CommentMentionedPayload;
  CommentReply: CommentReplyPayload;
  SystemAnnouncement: SystemAnnouncementPayload;
  PrivateMessageNew: PrivateMessageNewPayload;
  PrivateMessageEdited: PrivateMessageEditedPayload;
  PrivateMessageRecalled: PrivateMessageRecalledPayload;
  PrivateMessageDeleted: PrivateMessageDeletedPayload;
  PrivateMessageReaction: PrivateMessageReactionPayload;
  PrivateMessageRead: PrivateMessageReadPayload;
  PrivateUserTyping: PrivateUserTypingPayload;
  GroupMessageNew: GroupMessageNewPayload;
  GroupMessageEdited: GroupMessageEditedPayload;
  GroupMessageRecalled: GroupMessageRecalledPayload;
  GroupMessageDeleted: GroupMessageDeletedPayload;
  GroupMessageReaction: GroupMessageReactionPayload;
  GroupMessageRead: GroupMessageReadPayload;
  GroupUserTyping: GroupUserTypingPayload;
  GroupMemberJoined: GroupMemberJoinedPayload;
  GroupMemberLeft: GroupMemberLeftPayload;
  GroupProfileUpdated: GroupProfileUpdatedPayload;
  GroupMemberRoleChanged: GroupMemberRoleChangedPayload;
  GroupMemberMuted: GroupMemberMutedPayload;
  GroupMemberUnmuted: GroupMemberUnmutedPayload;
  GroupDismissed: GroupDismissedPayload;
  GroupInviteReceived: GroupInviteReceivedPayload;
  GroupInviteResponse: GroupInviteResponsePayload;
  GroupApplicationReceived: GroupApplicationReceivedPayload;
  GroupApplicationResponse: GroupApplicationResponsePayload;
  GroupAnnouncementCreated: GroupAnnouncementCreatedPayload;
  GroupAnnouncementUpdated: GroupAnnouncementUpdatedPayload;
  GroupAnnouncementDeleted: GroupAnnouncementDeletedPayload;
  PrivateChatDeleted: PrivateChatDeletedPayload;
  PrivateChatSettingsUpdated: PrivateChatSettingsUpdatedPayload;
  NotificationReadStateUpdated: NotificationReadStateUpdatedPayload;
  SubscribeAck: SubscribeAckPayload;
  Error: ErrorPayload;
}

// 类型安全的服务端事件
export type ServerEvent = {
  [K in keyof ServerEventPayloadMap]: {
    type: K;
    payload: ServerEventPayloadMap[K];
  };
}[keyof ServerEventPayloadMap];

// 按事件 type 提取 payload 的辅助类型
export type ServerPayload<K extends keyof ServerEventPayloadMap> = ServerEventPayloadMap[K];
