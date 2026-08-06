// API 响应载荷类型——镜像 `{ code, message, data }` 中的 `data` 字段。
// 字段名沿用 spec 的 snake_case。分页响应统一包裹在 PaginatedData<T>。
// 唯一权威来源：spec/api_spec_*.md（每个 interface 标注来源章节）。
//
// PaginatedData<T> 与 ApiResponse<T> 定义在 client.ts；此处 PaginatedData 重声明
// 以结构等价的方式供仅 import types.ts 的调用方使用（避免与 client 循环依赖）。

// ─────────────────────────── 通用 / 共享原语 ───────────────────────────
// spec: api_spec_types.md — 领域原语与领域实体

export interface GeoPoint {
	// spec: types.md → 位置与地理信息
	latitude: number; // [-90, 90]
	longitude: number; // [-180, 180]
}

export interface Location {
	// spec: types.md → Location（请求侧：原始坐标 + 名称）
	point: GeoPoint;
	location_name: string;
}

export interface RelativeLocation {
	// spec: types.md → RelativeLocation（响应侧：坐标 + 名称 + 可选距离/方位）
	point: GeoPoint;
	location_name: string;
	distance?: number; // 距参考点的米数
	bearing_degrees?: number;
}

export interface MediaAsset {
	// spec: types.md → MediaAsset
	url: string;
	thumbnail_url?: string;
	width?: number;
	height?: number;
	mime_type?: string;
}

export interface MessageMedia extends MediaAsset {
	// spec: types.md → MessageMedia
	file_name?: string;
	file_size?: number;
	duration?: number;
}

export interface Reaction {
	// spec: types.md → Reaction
	emoji: string;
	count: number;
	reacted_by_me: boolean;
}

export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
export type UserType = 'temp' | 'normal';
export type UserPresenceStatus = 'online' | 'offline';
export type PrivacyLevel = 'public' | 'private';
export type GroupRole = 'owner' | 'admin' | 'member';
export type GroupJoinMethod = 'direct' | 'password' | 'invite_only';
export type ChatPerm = 'everyone' | 'friends' | 'nobody';
export type ReactTarget = 'message' | 'moment' | 'comment';
export type ReportReason =
	| 'spam'
	| 'harassment'
	| 'violence'
	| 'hate_speech'
	| 'sexual_content'
	| 'misinformation'
	| 'impersonation'
	| 'self_harm'
	| 'other';
export type ReportStatus = 'pending' | 'reviewing' | 'escalated' | 'resolved' | 'dismissed';
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
export type BlockReason = 'privacy' | 'stranger_quota_exceeded' | 'account_suspended';
export type AnnounceLevel = 'info' | 'warning' | 'error';
export type NotifyCategory = 'social' | 'interaction' | 'group' | 'report' | 'nearby' | 'system';
export type NotifyType =
	| 'new_follower'
	| 'mutual_follow'
	| 'moment_commented'
	| 'comment_reply'
	| 'mentioned'
	| 'reaction_added'
	| 'group_invite_received'
	| 'group_invite_revoked'
	| 'group_invite_accepted'
	| 'group_invite_rejected'
	| 'group_application_received'
	| 'group_application_cancelled'
	| 'group_application_accepted'
	| 'group_application_rejected'
	| 'report_resolved'
	| 'report_dismissed'
	| 'system_announcement'
	| 'nearby_new_moment';
export type Gender = 'male' | 'female' | 'other';

// ─────────────────────────── 领域实体 ───────────────────────────

export interface UserInfo {
	// spec: types.md → UserInfo
	user_id: string;
	nickname: string;
	avatar_url?: string;
}

export interface UserStats {
	// spec: types.md → UserStats
	moments_count: number;
	friends_count: number;
	followers_count: number;
	following_count: number;
	groups_count: number;
}

export interface Relationship {
	// spec: types.md → Relationship（@auth-only）
	is_following: boolean;
	is_followed_by: boolean;
	is_blocked: boolean;
}

export interface SocialContext {
	// spec: types.md → UserItem.social_context
	mutual_friends_count: number;
	mutual_friends_preview?: UserInfo[];
	common_groups_count?: number;
	common_tags?: string[];
}

export interface UserItem {
	// spec: types.md → UserItem（list/search/detail/feed）
	user: UserInfo;
	bio?: string;
	tags?: string[];
	presence_status?: UserPresenceStatus;
	last_online_at?: string;
	location?: RelativeLocation;
	stats?: UserStats;
	relation?: Relationship;
	social_context?: SocialContext;
}

export interface GroupInfo {
	// spec: types.md → GroupInfo
	group_id: string;
	name: string;
	avatar_url?: string;
	members_count: number;
}

export interface MyGroupState {
	// spec: types.md → MyGroupState（@auth-only）
	role: GroupRole;
	is_joined: boolean;
	unread_count: number;
	is_muted: boolean;
	muted_until?: string;
	is_pinned: boolean;
	joined_at?: string;
}

export interface GroupItem {
	// spec: types.md → GroupItem（list/search/detail/feed）
	group: GroupInfo;
	description?: string;
	tags?: string[];
	join_method: GroupJoinMethod;
	requires_approval: boolean;
	has_password?: boolean;
	location: RelativeLocation;
	last_activity_at?: string;
	online_members_count?: number;
	next_activity?: string;
	last_message?: {
		message_id: string;
		type: MessageType;
		content?: string;
		sender: UserInfo;
		sent_at: string;
	};
	created_at?: string;
	owner_id?: string;
	my_state?: MyGroupState;
	friends_in_group?: number;
	is_searchable?: boolean;
	max_members?: number;
}

export interface GroupAnnouncementItem {
	// spec: types.md → GroupAnnouncementItem
	announcement_id: string;
	group_id: string;
	title: string;
	content: string;
	created_by: UserInfo;
	created_at: string;
	is_pinned: boolean;
}

export interface GroupMemberItem {
	// spec: api_spec_group.md → members/list
	user: UserInfo;
	role: GroupRole;
	joined_at: string;
	muted_until?: string;
	presence_status?: UserPresenceStatus;
	last_online_at?: string;
}

export interface MomentItem {
	// spec: types.md → MomentItem（list/search/detail/feed）
	moment_id: string;
	description: string;
	images?: MediaAsset[];
	tags?: string[];
	mentions?: string[];
	created_at: string;
	likes_count: number;
	comments_count: number;
	shares_count: number;
	reactions?: Reaction[];
	user: UserInfo;
	location: RelativeLocation;
	privacy_level?: PrivacyLevel;
	my_state?: { is_favorited: boolean };
	recommendation?: 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';
}

export interface CommentItem {
	// spec: api_spec_moment.md → comments/create & comments/list
	comment_id: string;
	user: UserInfo;
	content: string;
	parent_id?: string;
	mentions?: string[];
	created_at: string;
	replies_count: number;
	reactions?: Reaction[];
	is_deleted: boolean;
	top_replies?: CommentItem[];
}

export interface MessageReplyTo {
	// spec: types.md → MessageReplyTo
	message_id: string;
	type: MessageType;
	sender: UserInfo;
	content?: string;
	share_code?: string;
	sent_at: string;
}

export interface PrivateMessageItem {
	// spec: api_spec_chat.md → PrivateMessageItem
	message_id: string;
	type: MessageType;
	content?: string;
	media?: MessageMedia;
	share_code?: string;
	sender: UserInfo;
	reply_to?: MessageReplyTo;
	sent_at: string;
	edited_at?: string;
	recalled_at?: string;
	is_read: boolean;
	reactions?: Reaction[];
}

export interface GroupMessageItem {
	// spec: api_spec_chat.md → GroupMessageItem
	message_id: string;
	type: MessageType;
	content?: string;
	media?: MessageMedia;
	share_code?: string;
	sender: UserInfo;
	reply_to?: MessageReplyTo;
	mentions?: string[];
	recalled_by?: UserInfo;
	sent_at: string;
	edited_at?: string;
	recalled_at?: string;
	read_count?: number;
	reactions?: Reaction[];
}

export interface NotifyPayload {
	// spec: api_spec_notify.md → NotifyPayload（按 type 判别的联合体）。
	// 此处建模为宽松 record——消费方按 type 分支读取字段。
	type: NotifyType;
	actor?: UserInfo;
	source?: string;
	source_moment_id?: string;
	moment_id?: string;
	moment_summary?: { moment_id: string; description_preview: string; cover_url?: string };
	comment_id?: string;
	comment_preview?: string;
	reply_id?: string;
	reply_content_preview?: string;
	replied_comment_preview?: string;
	target_type?: 'moment' | 'comment' | 'message';
	target_id?: string;
	emoji?: string;
	group_id?: string;
	group_name?: string;
	group_avatar_url?: string;
	invite_id?: string;
	application_id?: string;
	message?: string;
	reason?: string;
	revoked_by?: UserInfo;
	rejected_by?: UserInfo;
	user?: UserInfo;
	applicant?: UserInfo;
	report_id?: string;
	resolved_at?: string;
	dismissed_at?: string;
	note?: string;
	announcement_id?: string;
	severity?: AnnounceLevel;
	distance?: number;
}

export interface NotifyItem {
	// spec: api_spec_notify.md → NotifyItem
	notification_id: string;
	type: NotifyType;
	is_read: boolean;
	created_at: string;
	payload: NotifyPayload;
}

export interface UserSettings {
	// spec: types.md → UserSettings
	privacy_level: PrivacyLevel;
	location_visible: boolean;
	presence_visible: boolean;
	social_lists_visible: boolean;
	groups_visible: boolean;
	reacted_moments_visible: boolean;
	stats_visible: boolean;
	allow_follow: boolean;
	allow_private_chat: ChatPerm;
	searchable: boolean;
	phone_searchable: boolean;
	nickname_searchable: boolean;
	location_searchable: boolean;
	nearby_moment_notify: boolean;
}

export interface FeatureSwitches {
	// spec: types.md → FeatureSwitches
	enable_public_registration: boolean;
	enable_guest_login: boolean;
	enable_moments: boolean;
	enable_nearby_people: boolean;
}

export interface AppLimits {
	// spec: types.md → AppLimits
	max_group_members: number;
	max_moment_tags: number;
	max_moment_images: number;
	moment_description_max_length: number;
	comment_max_length: number;
	search_radius_max: number;
	message_recall_window_seconds: number;
	stranger_message_daily_quota: number;
	reaction_emojis: string[];
	group_invite_ttl_hours: number;
	group_application_ttl_hours: number;
}

export interface ContactInfo {
	// spec: types.md → ContactInfo
	support_email: string;
	privacy_policy_url: string;
	terms_of_service_url: string;
	official_website_url: string;
}

// ─────────────────────────── 端点响应 data 形状 ───────────────────────────

// 分页信封在此重声明，结构等价于 client.ts 的 PaginatedData<T>，
// 供仅 import types.ts 的调用方使用，避免与 client 循环依赖。
export interface PaginatedData<T> {
	items: T[];
	next_cursor?: string;
}

// --- Auth（spec: api_spec_auth.md）---
// web client_type：data 为空对象——token 走 HttpOnly cookie
export type AuthTokenPair = Record<string, never>;
export interface AuthStatus {
	// spec: auth.md → check_status
	is_valid: boolean;
	expires_at?: string;
}

// --- Users（spec: api_spec_user.md）---
export interface AccountData {
	// spec: user.md → get_account
	user_id: string;
	nickname: string;
	avatar_url?: string;
	user_type: UserType;
	bio?: string;
	tags?: string[];
	gender?: Gender;
	email?: string;
	phone?: string;
	location: Location;
	presence_status: UserPresenceStatus;
	last_online_at: string;
	created_at: string;
	stats: UserStats;
}

export interface UserProfileBundle {
	// spec: user.md → get_profile
	user: UserItem;
	recent_moments: PaginatedData<MomentItem>;
	/** 喜欢/收藏流（mock 演示数据，spec 未定义独立字段，扩展字段） */
	liked_moments?: PaginatedData<MomentItem>;
	favorited_moments?: PaginatedData<MomentItem>;
}

export interface FriendItem {
	// spec: user.md → list_friends
	user: UserInfo;
	presence_status?: UserPresenceStatus;
	last_online_at?: string;
	remark?: string;
	followed_at: string;
	followed_back_at: string;
}

export interface FollowingItem {
	// spec: user.md → list_following
	user: UserInfo;
	followed_at: string;
	is_mutual: boolean;
	remark?: string;
	presence_status?: UserPresenceStatus;
	last_online_at?: string;
}

export interface FollowerItem {
	// spec: user.md → list_followers
	user: UserInfo;
	followed_at: string;
	is_mutual: boolean;
	presence_status?: UserPresenceStatus;
	last_online_at?: string;
}

// --- Feed（spec: api_spec_feed.md）---
export interface TrendingBundle {
	// spec: feed.md → trending
	hot_moments: MomentItem[];
}

// --- Notifications（spec: api_spec_notify.md）---
export interface UnreadCountResult {
	// spec: notify.md → get_unread_count
	total: number;
	by_type: Record<NotifyCategory, number>;
}

// --- System（spec: api_spec_system.md）---
export interface ClientDefaults {
	default_location: Location;
	default_user_avatar_url: string;
	default_group_avatar_url: string;
}
export interface ClientConfig {
	// spec: system.md → get_client_config
	features: FeatureSwitches;
	limits: AppLimits;
	contact: ContactInfo;
	defaults: ClientDefaults;
}
export interface ApiVersionMeta {
	// spec: system.md → get_meta
	version: string;
	status: 'active' | 'deprecated' | 'sunset';
	deprecated_at?: string;
	sunset_at?: string;
	message?: string;
	recommended_version?: string;
}
export interface UnreadSummary {
	// spec: system.md → bootstrap
	notification_total: number;
	notification_by_type: Record<NotifyCategory, number>;
	private_chat_unread_count: number;
	group_chat_unread_count: number;
	total_badge: number;
}
export interface BootstrapData {
	// spec: system.md → bootstrap
	bootstrap_version: string;
	meta: ApiVersionMeta;
	client_config: ClientConfig;
	profile?: AccountData;
	unread_summary?: UnreadSummary;
}

// --- Chat（spec: api_spec_chat.md）---
export interface PrivateChatItem {
	// spec: chat.md → private_chats/list
	other_user: UserInfo;
	unread_count: number;
	is_muted: boolean;
	is_pinned: boolean;
	last_activity_at: string;
	last_message?: {
		message_id: string;
		content?: string;
		type: MessageType;
		share_code?: string;
		sender: UserInfo;
		sent_at: string;
	};
	presence_status?: UserPresenceStatus;
	created_at?: string;
	can_message: boolean;
	message_blocked_reason?: BlockReason;
}
export interface MessageSent {
	// spec: chat.md → MessageSent（private_messages/send、group_messages/send）
	message_id: string;
	sent_at: string;
}

// --- Groups（spec: api_spec_group.md）---
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected' | 'revoked';

// --- Reports（spec: api_spec_report.md）---
export interface ReportCreated {
	// spec: report.md → reports/create
	report_id: string;
}
