/**
 * WebSocket 类型定义
 * 严格遵循 @spec/api_spec_websocket.mdc
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
	UserSettings
} from '$lib/types/api';

// ============ 基础类型 ============
// AnnouncementSeverity：定义于 api.ts（spec/api_spec_types.mdc），通过 SystemAnnouncementItem 间接引用
export type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';
export type SubscriptionFailureReason = 'invalid' | 'denied' | 'limit_exceeded' | 'error';
export type ErrorSeverity = 'fatal' | 'recoverable' | 'warning';

// ============ WebSocket 事件类型枚举 ============

export type ServerEventType =
	// 连接管理
	| 'ConnectionEstablished'
	| 'Pong'
	// 用户状态
	| 'UserOnline'
	| 'UserOffline'
	// 关注关系
	| 'NewFollower'
	| 'MutualFollow'
	// 生活瞬间互动
	| 'MomentLiked'
	| 'MomentCommented'
	| 'CommentLiked'
	| 'CommentReply'
	| 'MessageMentioned'
	// 私聊消息
	| 'PrivateMessageNew'
	| 'PrivateMessageEdited'
	| 'PrivateMessageRecalled'
	| 'PrivateMessageRead'
	| 'PrivateUserTyping'
	// 群聊消息
	| 'GroupMessageNew'
	| 'GroupMessageEdited'
	| 'GroupMessageRecalled'
	| 'GroupUserTyping'
	// 群组管理
	| 'GroupMemberJoined'
	| 'GroupMemberLeft'
	| 'GroupMemberRemoved'
	| 'GroupMemberMuted'
	| 'GroupMemberUnmuted'
	| 'GroupInviteReceived'
	| 'GroupInviteResponse'
	| 'GroupApplicationReceived'
	| 'GroupApplicationResponse'
	| 'GroupAnnouncementCreated'
	| 'GroupUpdated'
	| 'GroupDismissed'
	// 用户
	| 'UserProfileUpdated'
	// 系统
	| 'SystemAnnouncement'
	| 'AccountBanned'
	| 'Error'
	| 'SubscribeAck'
	| 'SubscriptionsSnapshot'
	// 多端同步（⏳ 延期实现）
	| 'SelfReadSync'
	| 'SelfChatSettingsSync'
	| 'SelfSettingsUpdated'
	| 'SelfRelationSync';

export type ClientEventType =
	| 'Auth'
	| 'Ping'
	| 'SetSubscriptions'
	| 'GetSubscriptions'
	| 'PrivateTyping'
	| 'GroupTyping';

// ============ 通用事件结构 ============

export interface WebSocketEvent<TType extends string, TPayload = unknown> {
	type: TType;
	payload: TPayload;
}

// 服务端推送事件
export type ServerWebSocketEvent = WebSocketEvent<ServerEventType, unknown>;

// ============ 辅助类型 ============

// MomentSummary 定义于 api.ts (对应 api_spec_types.mdc)

// MessageReplyTo 从 $lib/types/api 导入（含 share_code?: string，与 spec 一致）
export type { MessageReplyTo } from '$lib/types/api';

export interface FailedSubscriptionId {
	subscription_id: string;
	reason: SubscriptionFailureReason;
}

// ============ 服务端推送事件 ============

// 连接管理
export interface ConnectionEstablishedPayload {
	connection_id: string;
	user_id: string;
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
}

export interface PongPayload {}

// 在线状态
export interface UserOnlinePayload {
	user_id: string;
	online_at: string;
}

export interface UserOfflinePayload {
	user_id: string;
	offline_at: string; // 离线时间（即最后在线时间）
}

// 关注关系
export interface NewFollowerPayload {
	user: UserInfo;
	followed_at: string;
	source?: FollowSource;
	source_moment_id?: string;
}

export interface MutualFollowPayload {
	user: UserInfo;
	mutual_at: string;
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

export interface CommentLikedPayload {
	comment_id: string;
	moment_id: string;
	moment_summary?: MomentSummary;
	comment_preview: string;
	user: UserInfo;
	liked_at: string;
}

export interface CommentReplyPayload {
	moment_id: string;
	moment_summary?: MomentSummary;
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

/** spec: type SystemAnnouncementPayload = SystemAnnouncementItem */
export type SystemAnnouncementPayload = SystemAnnouncementItem;

// 用户资料变更
export interface UserProfileUpdatedPayload {
	user: UserInfo;
	updated_at: string;
}

// 私聊消息
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
	up_to: string;
	reader_id: string;
	read_at: string;
}

export interface PrivateUserTypingPayload {
	target_user_id: string;
	user_id: string;
	is_typing: boolean;
}

// 群聊消息
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

// 群组管理
export interface GroupMemberJoinedPayload {
	group_id: string;
	user: UserInfo;
	role: GroupRole;
	member_count: number;
	joined_at: string;
}

// 成员离开（自愿离开）
export interface GroupMemberLeftPayload {
	group_id: string;
	user: UserInfo;
	left_at: string;
	member_count: number; // 剩余成员总数
}

// 成员被移除（被踢出/被封禁）
export interface GroupMemberRemovedPayload {
	group_id: string;
	user: UserInfo;
	removed_by: string; // user_id（操作者）
	reason?: 'kicked' | 'banned'; // 移除原因
	removed_at: string;
	member_count: number;
}

// 群组成员禁言
export interface GroupMemberMutedPayload {
	group_id: string;
	user: UserInfo;
	muted_by: string; // user_id（操作者）
	muted_until: string; // 禁言截止时间
	duration_minutes: number;
	muted_at: string;
}

// 群组成员解除禁言
export interface GroupMemberUnmutedPayload {
	group_id: string;
	user: UserInfo;
	unmuted_by: string; // user_id（操作者）
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
	status: 'accepted';
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
	status: 'accepted';
	responded_at: string;
}

// 群组信息更新
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
	updated_by: string; // user_id（操作者）
	updated_at: string;
}

// 群组解散（群主主动解散或因违规被解散）
export interface GroupDismissedPayload {
	group_id: string;
	dismissed_at: string;
	reason?: 'owner_dismissed' | 'policy_violation';
}

// 系统
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
	client_msg_id: string;
	operation: SubscriptionOperation;
	type: 'user' | 'group';
	ids: string[];
	code: number;
	message?: string;
	failed_ids?: FailedSubscriptionId[];
}

// 订阅状态快照（响应 GetSubscriptions）
export interface SubscriptionsSnapshotPayload {
	client_msg_id: string;
	user_ids: string[];
	group_ids: string[];
}

// 账号被封禁
export interface AccountBannedPayload {
	reason: string;
	duration_days: number;
	banned_at: string;
}

// === 多端同步事件（⏳ 延期实现）===

/** 已读状态同步：当某设备标记消息已读时，推送给该用户的其他设备 */
export interface SelfReadSyncPayload {
	chat_type: 'private' | 'group';
	target_id: string;
	up_to_message_id?: string;
}

/** 会话设置同步：静音/置顶变更 */
export interface SelfChatSettingsSyncPayload {
	chat_type: 'private' | 'group';
	target_id: string;
	is_muted?: boolean;
	is_pinned?: boolean;
}

/** 用户设置同步：隐私等全局设置变更 */
export interface SelfSettingsUpdatedPayload {
	settings: UserSettings;
}

/** 关系变更同步 */
export interface SelfRelationSyncPayload {
	action: 'follow' | 'unfollow' | 'block' | 'unblock';
	target_user_id: string;
}

// ============ 客户端发送事件 ============

export interface AuthPayload {
	client_type: ClientType;
	access_token?: string; // App 端必填，Web 端不传
}

export interface PingPayload {}

export interface SetSubscriptionsPayload {
	client_msg_id: string;
	operation: SubscriptionOperation;
	type: 'user' | 'group';
	ids: string[];
}

export interface GetSubscriptionsPayload {
	client_msg_id: string;
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
	| WebSocketEvent<'ConnectionEstablished', ConnectionEstablishedPayload>
	| WebSocketEvent<'Pong', PongPayload>
	| WebSocketEvent<'UserOnline', UserOnlinePayload>
	| WebSocketEvent<'UserOffline', UserOfflinePayload>
	| WebSocketEvent<'NewFollower', NewFollowerPayload>
	| WebSocketEvent<'MutualFollow', MutualFollowPayload>
	| WebSocketEvent<'MomentLiked', MomentLikedPayload>
	| WebSocketEvent<'MomentCommented', MomentCommentedPayload>
	| WebSocketEvent<'CommentLiked', CommentLikedPayload>
	| WebSocketEvent<'CommentReply', CommentReplyPayload>
	| WebSocketEvent<'MessageMentioned', MessageMentionedPayload>
	| WebSocketEvent<'SystemAnnouncement', SystemAnnouncementPayload>
	| WebSocketEvent<'UserProfileUpdated', UserProfileUpdatedPayload>
	| WebSocketEvent<'PrivateMessageNew', PrivateMessageNewPayload>
	| WebSocketEvent<'PrivateMessageEdited', PrivateMessageEditedPayload>
	| WebSocketEvent<'PrivateMessageRecalled', PrivateMessageRecalledPayload>
	| WebSocketEvent<'PrivateMessageRead', PrivateMessageReadPayload>
	| WebSocketEvent<'PrivateUserTyping', PrivateUserTypingPayload>
	| WebSocketEvent<'GroupMessageNew', GroupMessageNewPayload>
	| WebSocketEvent<'GroupMessageEdited', GroupMessageEditedPayload>
	| WebSocketEvent<'GroupMessageRecalled', GroupMessageRecalledPayload>
	| WebSocketEvent<'GroupUserTyping', GroupUserTypingPayload>
	| WebSocketEvent<'GroupMemberJoined', GroupMemberJoinedPayload>
	| WebSocketEvent<'GroupMemberLeft', GroupMemberLeftPayload>
	| WebSocketEvent<'GroupMemberRemoved', GroupMemberRemovedPayload>
	| WebSocketEvent<'GroupMemberMuted', GroupMemberMutedPayload>
	| WebSocketEvent<'GroupMemberUnmuted', GroupMemberUnmutedPayload>
	| WebSocketEvent<'GroupInviteReceived', GroupInviteReceivedPayload>
	| WebSocketEvent<'GroupInviteResponse', GroupInviteResponsePayload>
	| WebSocketEvent<'GroupApplicationReceived', GroupApplicationReceivedPayload>
	| WebSocketEvent<'GroupApplicationResponse', GroupApplicationResponsePayload>
	| WebSocketEvent<'GroupAnnouncementCreated', GroupAnnouncementItem>
	| WebSocketEvent<'GroupUpdated', GroupUpdatedPayload>
	| WebSocketEvent<'GroupDismissed', GroupDismissedPayload>
	| WebSocketEvent<'AccountBanned', AccountBannedPayload>
	| WebSocketEvent<'Error', ErrorPayload>
	| WebSocketEvent<'SubscribeAck', SubscribeAckPayload>
	| WebSocketEvent<'SubscriptionsSnapshot', SubscriptionsSnapshotPayload>
	// 多端同步（⏳ 延期实现）
	| WebSocketEvent<'SelfReadSync', SelfReadSyncPayload>
	| WebSocketEvent<'SelfChatSettingsSync', SelfChatSettingsSyncPayload>
	| WebSocketEvent<'SelfSettingsUpdated', SelfSettingsUpdatedPayload>
	| WebSocketEvent<'SelfRelationSync', SelfRelationSyncPayload>;

export type ClientEvent =
	| WebSocketEvent<'Auth', AuthPayload>
	| WebSocketEvent<'Ping', PingPayload>
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
	// 在线状态
	UserOnline: UserOnlinePayload;
	UserOffline: UserOfflinePayload;
	// 关注关系
	NewFollower: NewFollowerPayload;
	MutualFollow: MutualFollowPayload;
	// 生活瞬间互动
	MomentLiked: MomentLikedPayload;
	MomentCommented: MomentCommentedPayload;
	CommentLiked: CommentLikedPayload;
	CommentReply: CommentReplyPayload;
	MessageMentioned: MessageMentionedPayload;
	SystemAnnouncement: SystemAnnouncementPayload;
	UserProfileUpdated: UserProfileUpdatedPayload;
	// 私聊消息
	PrivateMessageNew: PrivateMessageNewPayload;
	PrivateMessageEdited: PrivateMessageEditedPayload;
	PrivateMessageRecalled: PrivateMessageRecalledPayload;
	PrivateMessageRead: PrivateMessageReadPayload;
	PrivateUserTyping: PrivateUserTypingPayload;
	// 群聊消息
	GroupMessageNew: GroupMessageNewPayload;
	GroupMessageEdited: GroupMessageEditedPayload;
	GroupMessageRecalled: GroupMessageRecalledPayload;
	GroupUserTyping: GroupUserTypingPayload;
	// 群组管理
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
	// 系统
	AccountBanned: AccountBannedPayload;
	Error: ErrorPayload;
	SubscribeAck: SubscribeAckPayload;
	SubscriptionsSnapshot: SubscriptionsSnapshotPayload;
	// 多端同步（⏳ 延期实现）
	SelfReadSync: SelfReadSyncPayload;
	SelfChatSettingsSync: SelfChatSettingsSyncPayload;
	SelfSettingsUpdated: SelfSettingsUpdatedPayload;
	SelfRelationSync: SelfRelationSyncPayload;
};
