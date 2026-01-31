/**
 * WebSocket 类型定义
 * 严格遵循 @spec/api_spec_websocket.mdc
 */

import type { UserInfo, Location, MessageMedia, MessageLocation } from '$lib/types/api';

// ============ 基础类型 ============

export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'location' | 'file';
export type ReactionType = 'like' | 'love' | 'laugh' | 'surprise' | 'sad' | 'angry';
export type GroupRole = 'owner' | 'admin' | 'member';
export type AnnouncementSeverity = 'info' | 'warning' | 'critical';

// ============ WebSocket 事件类型枚举 ============

export type ServerEventType =
	// 连接管理
	| 'ConnectionEstablished'
	| 'Pong'
	// 用户状态
	| 'UserOnline'
	| 'UserOffline'
	| 'UserLocationUpdate'
	// 关注关系
	| 'NewFollower'
	| 'MutualFollow'
	| 'FollowRequestReceived'
	| 'FollowRequestHandled'
	// 生活瞬间互动
	| 'MomentLiked'
	| 'MomentCommented'
	| 'CommentReply'
	| 'MessageMentioned'
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
	| 'GroupInviteReceived'
	| 'GroupInviteResponse'
	| 'GroupApplicationReceived'
	| 'GroupApplicationResponse'
	| 'GroupAnnouncementCreated'
	// 系统
	| 'Error'
	| 'SubscribeAck';

export type ClientEventType =
	| 'Auth'
	| 'Ping'
	| 'Subscribe'
	| 'Unsubscribe'
	| 'SetSubscriptions'
	| 'PrivateTyping'
	| 'GroupTyping';

// ============ 通用事件结构 ============

export interface WebSocketEvent<TType extends string, TPayload = unknown> {
	type: TType;
	payload: TPayload;
}

// ============ 辅助类型 ============

export interface MomentSummary {
	moment_id: string;
	cover_url?: string;
	text_preview: string;
}

export interface MessageReplyTo {
	message_id: string;
	user_id: number;
	nickname: string;
	content: string;
}

export type SubscriptionSourceId = string | number;

export interface FailedSubscriptionId {
	id: string | number;
	reason: 'not_found' | 'permission_denied' | 'blocked';
}

export type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';

// ============ 服务端推送事件 ============

// 连接管理
export interface ConnectionEstablishedPayload {
	connection_id: string;
	user_id: number;
	server_time: string;
}

export interface PongPayload {
	timestamp: string;
}

// 在线状态
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

// 关注关系
export interface NewFollowerPayload {
	user: UserInfo;
	followed_at: string;
	is_mutual: boolean;
}

export interface MutualFollowPayload {
	user: UserInfo;
	mutual_at: string;
}

export interface FollowRequestReceivedPayload {
	request_id: string;
	requester: UserInfo;
	message?: string;
	requested_at: string;
}

export interface FollowRequestHandledPayload {
	request_id: string;
	target_user: UserInfo;
	action: 'accept';
	handled_at: string;
}

// 生活瞬间互动
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

export interface MessageMentionedPayload {
	target_type: 'moment' | 'comment';
	moment_id: string;
	comment_id?: string;
	moment_summary?: MomentSummary;
	comment_preview?: string;
	user: UserInfo;
	mentioned_at: string;
}

export interface SystemAnnouncementPayload {
	announcement_id: string;
	title: string;
	content: string;
	severity: AnnouncementSeverity;
	action_url?: string;
	created_at: string;
}

// 私聊消息
export interface PrivateMessageNewPayload {
	message_id: string;
	target_user_id: number;
	sender: UserInfo;
	type: MessageType;
	content?: string;
	media?: MessageMedia;
	location?: MessageLocation;
	reply_to?: MessageReplyTo;
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
	up_to: string;
	reader_id: number;
	read_at: string;
}

export interface PrivateUserTypingPayload {
	target_user_id: number;
	user_id: number;
	is_typing: boolean;
}

// 群聊消息
export interface GroupMessageNewPayload {
	message_id: string;
	group_chat_id: string;
	sender: UserInfo;
	type: MessageType;
	content?: string;
	media?: MessageMedia;
	location?: MessageLocation;
	reply_to?: MessageReplyTo;
	mentions?: number[];
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
	up_to: string;
	reader_id: number;
	read_at: string;
}

export interface GroupUserTypingPayload {
	group_chat_id: string;
	user_id: number;
	is_typing: boolean;
}

// 群组管理
export interface GroupMemberJoinedPayload {
	group_chat_id: string;
	user: UserInfo;
	role: GroupRole;
	member_count: number;
	joined_at: string;
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
	status: 'accepted';
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
	status: 'accepted';
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

// 系统
export interface ErrorPayload {
	code: number;
	message: string;
}

export interface SubscribeAckPayload {
	client_msg_id: string;
	operation: SubscriptionOperation;
	type: string;
	ids: SubscriptionSourceId[];
	code: number;
	message?: string;
	failed_ids?: FailedSubscriptionId[];
}

// ============ 客户端发送事件 ============

export interface AuthPayload {
	access_token: string;
	device_fingerprint?: string;
}

export interface PingPayload {
	timestamp: string;
}

export interface SubscribePayload {
	client_msg_id: string;
	type: string;
	ids: SubscriptionSourceId[];
}

export interface UnsubscribePayload {
	client_msg_id: string;
	type: string;
	ids: SubscriptionSourceId[];
}

export interface SetSubscriptionsPayload {
	client_msg_id: string;
	type: string;
	ids: SubscriptionSourceId[];
}

export interface PrivateTypingPayload {
	target_user_id: number;
	is_typing: boolean;
}

export interface GroupTypingPayload {
	group_chat_id: string;
	is_typing: boolean;
}

// ============ 类型联合 ============

export type ServerEvent =
	| WebSocketEvent<'ConnectionEstablished', ConnectionEstablishedPayload>
	| WebSocketEvent<'Pong', PongPayload>
	| WebSocketEvent<'UserOnline', UserOnlinePayload>
	| WebSocketEvent<'UserOffline', UserOfflinePayload>
	| WebSocketEvent<'UserLocationUpdate', UserLocationUpdatePayload>
	| WebSocketEvent<'NewFollower', NewFollowerPayload>
	| WebSocketEvent<'MutualFollow', MutualFollowPayload>
	| WebSocketEvent<'FollowRequestReceived', FollowRequestReceivedPayload>
	| WebSocketEvent<'FollowRequestHandled', FollowRequestHandledPayload>
	| WebSocketEvent<'MomentLiked', MomentLikedPayload>
	| WebSocketEvent<'MomentCommented', MomentCommentedPayload>
	| WebSocketEvent<'CommentReply', CommentReplyPayload>
	| WebSocketEvent<'MessageMentioned', MessageMentionedPayload>
	| WebSocketEvent<'SystemAnnouncement', SystemAnnouncementPayload>
	| WebSocketEvent<'PrivateMessageNew', PrivateMessageNewPayload>
	| WebSocketEvent<'PrivateMessageEdited', PrivateMessageEditedPayload>
	| WebSocketEvent<'PrivateMessageRecalled', PrivateMessageRecalledPayload>
	| WebSocketEvent<'PrivateMessageDeleted', PrivateMessageDeletedPayload>
	| WebSocketEvent<'PrivateMessageReaction', PrivateMessageReactionPayload>
	| WebSocketEvent<'PrivateMessageRead', PrivateMessageReadPayload>
	| WebSocketEvent<'PrivateUserTyping', PrivateUserTypingPayload>
	| WebSocketEvent<'GroupMessageNew', GroupMessageNewPayload>
	| WebSocketEvent<'GroupMessageEdited', GroupMessageEditedPayload>
	| WebSocketEvent<'GroupMessageRecalled', GroupMessageRecalledPayload>
	| WebSocketEvent<'GroupMessageDeleted', GroupMessageDeletedPayload>
	| WebSocketEvent<'GroupMessageReaction', GroupMessageReactionPayload>
	| WebSocketEvent<'GroupMessageRead', GroupMessageReadPayload>
	| WebSocketEvent<'GroupUserTyping', GroupUserTypingPayload>
	| WebSocketEvent<'GroupMemberJoined', GroupMemberJoinedPayload>
	| WebSocketEvent<'GroupInviteReceived', GroupInviteReceivedPayload>
	| WebSocketEvent<'GroupInviteResponse', GroupInviteResponsePayload>
	| WebSocketEvent<'GroupApplicationReceived', GroupApplicationReceivedPayload>
	| WebSocketEvent<'GroupApplicationResponse', GroupApplicationResponsePayload>
	| WebSocketEvent<'GroupAnnouncementCreated', GroupAnnouncementCreatedPayload>
	| WebSocketEvent<'Error', ErrorPayload>
	| WebSocketEvent<'SubscribeAck', SubscribeAckPayload>;

export type ClientEvent =
	| WebSocketEvent<'Auth', AuthPayload>
	| WebSocketEvent<'Ping', PingPayload>
	| WebSocketEvent<'Subscribe', SubscribePayload>
	| WebSocketEvent<'Unsubscribe', UnsubscribePayload>
	| WebSocketEvent<'SetSubscriptions', SetSubscriptionsPayload>
	| WebSocketEvent<'PrivateTyping', PrivateTypingPayload>
	| WebSocketEvent<'GroupTyping', GroupTypingPayload>;
