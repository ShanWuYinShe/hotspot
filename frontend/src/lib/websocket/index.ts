/**
 * WebSocket 模块入口
 * 严格遵循 @spec/api_spec_websocket.md
 */

export { useWebSocketStore } from './store';
export type { ConnectionState } from './store';

export { SubscriptionManager } from './subscription-manager';
export type { SubscriptionType } from './subscription-manager';

export { WebSocketEventHandlers, setPresenceStore } from './handlers';
export type { EventHandler } from './handlers';

export type {
  ServerEventType,
  ClientEventType,
  ServerWebSocketEvent,
  ClientEvent,
  WebSocketEvent,
  WebSocketEventMap,
  ConnectionEstablishedPayload,
  PongPayload,
  UserOnlinePayload,
  UserOfflinePayload,
  NewFollowerPayload,
  MutualFollowPayload,
  MomentLikedPayload,
  MomentCommentedPayload,
  CommentLikedPayload,
  CommentReplyPayload,
  MessageMentionedPayload,
  SystemAnnouncementPayload,
  UserProfileUpdatedPayload,
  PrivateMessageNewPayload,
  PrivateMessageEditedPayload,
  PrivateMessageRecalledPayload,
  PrivateMessageReadPayload,
  PrivateUserTypingPayload,
  GroupMessageNewPayload,
  GroupMessageEditedPayload,
  GroupMessageRecalledPayload,
  GroupUserTypingPayload,
  GroupMemberJoinedPayload,
  GroupMemberLeftPayload,
  GroupMemberRemovedPayload,
  GroupMemberMutedPayload,
  GroupMemberUnmutedPayload,
  GroupInviteReceivedPayload,
  GroupInviteResponsePayload,
  GroupApplicationReceivedPayload,
  GroupApplicationResponsePayload,
  GroupUpdatedPayload,
  GroupDismissedPayload,
  ErrorPayload,
  SubscribeAckPayload,
  SubscriptionsSnapshotPayload,
  AccountBannedPayload,
  SelfReadSyncPayload,
  SelfChatSettingsSyncPayload,
  SelfSettingsUpdatedPayload,
  SelfRelationSyncPayload,
  AuthPayload,
  PingPayload,
  SetSubscriptionsPayload,
  GetSubscriptionsPayload,
  PrivateTypingPayload,
  GroupTypingPayload,
  SubscriptionOperation,
  SubscriptionFailureReason,
  ErrorSeverity,
  FailedSubscriptionId,
} from './types';
