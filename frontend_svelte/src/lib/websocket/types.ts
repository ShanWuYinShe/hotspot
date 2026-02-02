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
	| 'BatchEvents'
	// 用户状态
	| 'UserOnline'
	| 'UserOffline'
	| 'UserPresenceUpdate'
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
	| 'GroupMemberLeft'
	| 'GroupMemberRemoved'
	| 'GroupMemberRoleChanged'
	| 'GroupMemberMuted'
	| 'GroupMemberUnmuted'
	| 'GroupInviteReceived'
	| 'GroupInviteResponse'
	| 'GroupApplicationReceived'
	| 'GroupApplicationResponse'
	| 'GroupAnnouncementCreated'
	| 'GroupAnnouncementUpdated'
	| 'GroupUpdated'
	| 'GroupDeleted'
	// 系统
	| 'Error'
	| 'SubscribeAck'
	| 'SubscriptionsSnapshot';

export type ClientEventType =
	| 'Auth'
	| 'Ping'
	| 'Subscribe'
	| 'Unsubscribe'
	| 'SetSubscriptions'
	| 'GetSubscriptions'
	| 'PrivateTyping'
	| 'GroupTyping';

// ============ 通用事件结构 ============

export interface WebSocketEvent<TType extends string, TPayload = unknown> {
	type: TType;
	payload: TPayload;
}

// 服务端推送事件：包含 event_id 用于去重（幂等性保障）
export interface ServerWebSocketEvent extends WebSocketEvent<ServerEventType, unknown> {
	event_id: string; // 事件唯一ID（UUID），用于客户端去重
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
	server_time: string; // 服务端当前时间（UTC ISO 8601），客户端用于判断断开时长
	
	// 服务端配置（动态配置，客户端不得硬编码）
	config: {
		ping_interval: number; // 心跳间隔（秒）
		ping_timeout: number; // Pong 超时（秒）
		max_subscriptions: {
			user: number; // 用户订阅上限
			group: number; // 群组订阅上限
		};
	};
	
	// 未读计数（快速显示角标）
	unread_counts: {
		messages: number; // 未读私聊消息数
		group_messages: number; // 未读群聊消息数
		notifications: number; // 未读通知数
	};
}

export interface PongPayload {}

// 批量事件推送（性能优化）
export interface BatchEventsPayload {
	events: ServerWebSocketEvent[]; // 多个事件（已包含 event_id）
	batch_id: string; // 批次ID（UUID）
}

// 用户状态类型
export type UserPresenceStatus =
	| 'online' // 正在活跃（有 WebSocket 连接且 5 分钟内有活动）
	| 'idle' // 空闲/离开（有连接但 5 分钟没活动）
	| 'dnd' // 请勿打扰（用户主动设置）
	| 'invisible' // 隐身（对他人显示为 offline）
	| 'offline'; // 离线（无连接）

// 在线状态
export interface UserOnlinePayload {
	user_id: number;
	nickname: string;
	avatar_url?: string;
	status: UserPresenceStatus; // 状态类型
	status_text?: string; // 自定义状态文本（如"开会中"、"勿扰"）
	online_at: string;
}

export interface UserOfflinePayload {
	user_id: number;
	status: UserPresenceStatus; // 固定为 'offline'
	last_seen?: string; // 最后在线时间（可选）
	offline_at: string;
}

// 用户状态变更（online/idle/dnd/invisible 之间切换）
export interface UserPresenceUpdatePayload {
	user_id: number;
	status: UserPresenceStatus; // 新状态
	status_text?: string; // 自定义状态文本
	updated_at: string;
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

// 成员离开（自愿离开）
export interface GroupMemberLeftPayload {
	group_chat_id: string;
	user: UserInfo;
	left_at: string;
	member_count: number; // 剩余成员总数
}

// 成员被移除（被踢出/被封禁）
export interface GroupMemberRemovedPayload {
	group_chat_id: string;
	user: UserInfo;
	removed_by: number; // user_id（操作者）
	reason?: 'kicked' | 'banned'; // 移除原因
	removed_at: string;
	member_count: number;
}

// 成员角色变更
export interface GroupMemberRoleChangedPayload {
	group_chat_id: string;
	user: UserInfo;
	old_role: GroupRole; // 引用 @api_spec_common_types.mdc
	new_role: GroupRole;
	changed_by: number; // user_id（操作者）
	changed_at: string;
}

// 成员被禁言
export interface GroupMemberMutedPayload {
	group_chat_id: string;
	user: UserInfo;
	muted_by: number; // user_id（操作者）
	muted_until?: string; // 禁言结束时间（null 表示永久禁言）
	reason?: string;
	muted_at: string;
}

// 成员解除禁言
export interface GroupMemberUnmutedPayload {
	group_chat_id: string;
	user: UserInfo;
	unmuted_by: number; // user_id（操作者）
	unmuted_at: string;
}

// 群公告更新
export interface GroupAnnouncementUpdatedPayload {
	group_chat_id: string;
	announcement_id: string;
	content: string; // 更新后的内容
	updated_by: number; // user_id（操作者）
	updated_at: string;
}

// 群组信息更新
export interface GroupUpdatedPayload {
	group_chat_id: string;
	updated_fields: { // 只包含变更的字段
		name?: string;
		avatar_url?: string;
		description?: string;
	};
	updated_by: number; // user_id（操作者）
	updated_at: string;
}

// 群组解散
export interface GroupDeletedPayload {
	group_chat_id: string;
	deleted_by: number; // user_id（操作者）
	deleted_at: string;
	reason?: string;
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

// 订阅状态快照（响应 GetSubscriptions）
export interface SubscriptionsSnapshotPayload {
	client_msg_id: string; // 回传客户端消息ID
	user_ids: number[]; // 当前订阅的用户ID列表
	group_ids: string[]; // 当前订阅的群组ID列表
	total_count: number; // 总订阅数（user_ids.length + group_ids.length）
}

// ============ 客户端发送事件 ============

export interface AuthPayload {
	access_token: string;
	device_fingerprint?: string;
}

export interface PingPayload {}

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

export interface GetSubscriptionsPayload {
	client_msg_id: string; // 客户端消息ID，用于与响应对齐
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
	| WebSocketEvent<'BatchEvents', BatchEventsPayload>
	| WebSocketEvent<'UserOnline', UserOnlinePayload>
	| WebSocketEvent<'UserOffline', UserOfflinePayload>
	| WebSocketEvent<'UserPresenceUpdate', UserPresenceUpdatePayload>
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
	| WebSocketEvent<'GroupMemberLeft', GroupMemberLeftPayload>
	| WebSocketEvent<'GroupMemberRemoved', GroupMemberRemovedPayload>
	| WebSocketEvent<'GroupMemberRoleChanged', GroupMemberRoleChangedPayload>
	| WebSocketEvent<'GroupMemberMuted', GroupMemberMutedPayload>
	| WebSocketEvent<'GroupMemberUnmuted', GroupMemberUnmutedPayload>
	| WebSocketEvent<'GroupInviteReceived', GroupInviteReceivedPayload>
	| WebSocketEvent<'GroupInviteResponse', GroupInviteResponsePayload>
	| WebSocketEvent<'GroupApplicationReceived', GroupApplicationReceivedPayload>
	| WebSocketEvent<'GroupApplicationResponse', GroupApplicationResponsePayload>
	| WebSocketEvent<'GroupAnnouncementCreated', GroupAnnouncementCreatedPayload>
	| WebSocketEvent<'GroupAnnouncementUpdated', GroupAnnouncementUpdatedPayload>
	| WebSocketEvent<'GroupUpdated', GroupUpdatedPayload>
	| WebSocketEvent<'GroupDeleted', GroupDeletedPayload>
	| WebSocketEvent<'Error', ErrorPayload>
	| WebSocketEvent<'SubscribeAck', SubscribeAckPayload>
	| WebSocketEvent<'SubscriptionsSnapshot', SubscriptionsSnapshotPayload>;

export type ClientEvent =
	| WebSocketEvent<'Auth', AuthPayload>
	| WebSocketEvent<'Ping', PingPayload>
	| WebSocketEvent<'Subscribe', SubscribePayload>
	| WebSocketEvent<'Unsubscribe', UnsubscribePayload>
	| WebSocketEvent<'SetSubscriptions', SetSubscriptionsPayload>
	| WebSocketEvent<'GetSubscriptions', GetSubscriptionsPayload>
	| WebSocketEvent<'PrivateTyping', PrivateTypingPayload>
	| WebSocketEvent<'GroupTyping', GroupTypingPayload>;

// ============ WebSocket 事件类型映射 ============

/**
 * WebSocket 服务端事件类型映射
 * 用于 websocketStore.on() 方法的类型推断
 */
export type WebSocketEventMap = {
	// 连接管理
	ConnectionEstablished: ConnectionEstablishedPayload;
	Pong: PongPayload;
	BatchEvents: BatchEventsPayload;
	// 在线状态
	UserOnline: UserOnlinePayload;
	UserOffline: UserOfflinePayload;
	UserPresenceUpdate: UserPresenceUpdatePayload;
	UserLocationUpdate: UserLocationUpdatePayload;
	// 关注关系
	NewFollower: NewFollowerPayload;
	MutualFollow: MutualFollowPayload;
	FollowRequestReceived: FollowRequestReceivedPayload;
	FollowRequestHandled: FollowRequestHandledPayload;
	// 生活瞬间互动
	MomentLiked: MomentLikedPayload;
	MomentCommented: MomentCommentedPayload;
	CommentReply: CommentReplyPayload;
	MessageMentioned: MessageMentionedPayload;
	SystemAnnouncement: SystemAnnouncementPayload;
	// 私聊消息
	PrivateMessageNew: PrivateMessageNewPayload;
	PrivateMessageEdited: PrivateMessageEditedPayload;
	PrivateMessageRecalled: PrivateMessageRecalledPayload;
	PrivateMessageDeleted: PrivateMessageDeletedPayload;
	PrivateMessageReaction: PrivateMessageReactionPayload;
	PrivateMessageRead: PrivateMessageReadPayload;
	PrivateUserTyping: PrivateUserTypingPayload;
	// 群聊消息
	GroupMessageNew: GroupMessageNewPayload;
	GroupMessageEdited: GroupMessageEditedPayload;
	GroupMessageRecalled: GroupMessageRecalledPayload;
	GroupMessageDeleted: GroupMessageDeletedPayload;
	GroupMessageReaction: GroupMessageReactionPayload;
	GroupMessageRead: GroupMessageReadPayload;
	GroupUserTyping: GroupUserTypingPayload;
	// 群组管理
	GroupMemberJoined: GroupMemberJoinedPayload;
	GroupMemberLeft: GroupMemberLeftPayload;
	GroupMemberRemoved: GroupMemberRemovedPayload;
	GroupMemberRoleChanged: GroupMemberRoleChangedPayload;
	GroupMemberMuted: GroupMemberMutedPayload;
	GroupMemberUnmuted: GroupMemberUnmutedPayload;
	GroupInviteReceived: GroupInviteReceivedPayload;
	GroupInviteResponse: GroupInviteResponsePayload;
	GroupApplicationReceived: GroupApplicationReceivedPayload;
	GroupApplicationResponse: GroupApplicationResponsePayload;
	GroupAnnouncementCreated: GroupAnnouncementCreatedPayload;
	GroupAnnouncementUpdated: GroupAnnouncementUpdatedPayload;
	GroupUpdated: GroupUpdatedPayload;
	GroupDeleted: GroupDeletedPayload;
	// 系统
	Error: ErrorPayload;
	SubscribeAck: SubscribeAckPayload;
	SubscriptionsSnapshot: SubscriptionsSnapshotPayload;
};
