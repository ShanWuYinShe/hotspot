// HotSpot API 类型定义
// 严格对齐 openapi/components/schemas/ 下的 OpenAPI 3.1 规范

// ============================================================================
// 枚举类型（common.yaml + 各模块）
// ============================================================================

export type UserType = 'temp' | 'normal';
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
export type SendMessageType = Exclude<MessageType, 'system'>;
export type PrivacyLevel = 'public' | 'private';
export type GroupRole = 'owner' | 'admin' | 'member';
export type SetableGroupRole = 'admin' | 'member';
export type Gender = 'male' | 'female' | 'apache_helicopter';
export type ClientType = 'web' | 'app';
export type UserPresenceStatus = 'online' | 'offline';
export type UserAccountStatus = 'normal' | 'banned';
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected';
export type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
export type ReportHandleAction = 'resolve' | 'dismiss' | 'escalate';
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
export type GroupJoinMethod = 'open' | 'approval' | 'invite_only';
export type FeedSource = 'following' | 'nearby' | 'all';
export type DismissTargetType = 'moment' | 'user' | 'group';
export type PrivateChatPermission = 'everyone' | 'friends' | 'nobody';
export type AnnouncementSeverity = 'info' | 'warning' | 'error';
export type TagTrend = 'rising' | 'stable' | 'falling';
export type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';
export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';
export type NotificationType =
	| 'new_follower'
	| 'mutual_follow'
	| 'moment_liked'
	| 'moment_commented'
	| 'comment_liked'
	| 'comment_reply'
	| 'mentioned'
	| 'group_invite_received'
	| 'group_application_accepted'
	| 'system_announcement';
// NotificationTargetType 已删除：spec 中不存在该类型，mentioned payload 的 target_type 为内联 'moment' | 'comment'
export type MediaCategory =
	| 'avatar'
	| 'group_avatar'
	| 'moment_image'
	| 'chat_image'
	| 'chat_video'
	| 'chat_audio'
	| 'chat_file';
export type MapDataType = 'user' | 'group' | 'moment' | 'all';
export type MapItemType = 'cluster' | 'user' | 'group' | 'moment';
export type ShareType = 'user' | 'group' | 'moment' | 'location';
// 设计说明：被拉黑场景统一返回 'privacy'，不暴露拉黑状态
export type CantMessageReason = 'privacy' | 'stranger_limit' | 'account_suspended';
export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';
export type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';

// ============================================================================
// 位置类型（common.yaml）
// ============================================================================

/** 基础坐标点（经纬度） */
export interface GeoPoint {
	latitude: number;
	longitude: number;
}

/** 位置信息（仅用于 Request 请求参数） */
export interface Location {
	point: GeoPoint;
	location_name: string;
}

/**
 * 相对位置（用于所有 Response 响应数据）
 * distance/bearing_degrees 基于 reference_point 计算，搜索/列表/Feed 场景有值，详情场景为 undefined
 */
export interface RelativeLocation {
	point: GeoPoint;
	location_name: string;
	distance?: number;
	bearing_degrees?: number;
}

export interface Bounds {
	north: number;
	south: number;
	east: number;
	west: number;
}

export interface MapViewport {
	bounds: Bounds;
	zoom: number;
}

// ============================================================================
// 分页与搜索（common.yaml）
// ============================================================================

export interface PaginationParams {
	limit?: number;
	cursor?: string;
}

export interface PaginatedData<T> {
	items: T[];
	next_cursor?: string;
}

export interface TimeRangeParams {
	start_time?: string;
	end_time?: string;
}

// ============================================================================
// 媒体资源（common.yaml + media.yaml）
// ============================================================================

export interface MomentImage {
	url: string;
	thumbnail_url?: string;
	width?: number;
	height?: number;
	content_type?: string;
}

export interface MessageMedia {
	url: string;
	thumbnail_url?: string;
	file_name?: string;
	file_size?: number;
	duration?: number;
	width?: number;
	height?: number;
	mime_type?: string;
}

export interface PresignedUpload {
	upload_id: string;
	upload_url: string;
	headers: Record<string, string>;
	expires_at: string;
}

export interface UploadedMedia {
	file_id: string;
	thumbnail_url?: string;
	file_size: number;
	mime_type: string;
	duration?: number;
	width?: number;
	height?: number;
}

export interface RecommendationContext {
	type: RecommendationType;
}

// ============================================================================
// 用户类型（common.yaml + users.yaml）
// ============================================================================

/** 用户基础信息（用于嵌套在其他对象中），不包含 user_type */
export interface UserInfo {
	user_id: string;
	nickname: string;
	avatar_url?: string;
}

/** 用户统计信息 */
export interface UserStats {
	moments_count: number;
	friends_count: number;
	followers_count: number;
	following_count: number;
	groups_count: number;
}

/** 用户关系信息（@auth-only） */
// 设计说明：不暴露 is_blocked_by，被拉黑的用户不应感知到自己被拉黑
export interface Relationship {
	is_following: boolean;
	is_followed_by: boolean;
	is_blocked: boolean;
}

/** 社交上下文（@auth-only） */
export interface SocialContext {
	mutual_friends_count: number;
	mutual_friends_preview?: UserInfo[];
	common_groups_count?: number;
	common_tags?: string[];
}

/**
 * 用户统一数据类型（用于列表、搜索、详情、Feed 等所有场景）
 * 替代原 UserSearchItem/UserCard/FeedUserItem
 */
export interface UserItem {
	user: UserInfo;
	bio?: string;
	tags?: string[];
	presence_status?: UserPresenceStatus;
	last_online?: string;
	location?: RelativeLocation;
	stats?: UserStats;
	// @auth-only
	relation?: Relationship;
	social_context?: SocialContext;
}

/** 用户设置 */
export interface UserSettings {
	privacy_level: PrivacyLevel;
	location_visible: boolean;
	presence_visible: boolean;
	social_lists_visible: boolean;
	groups_visible: boolean;
	liked_moments_visible: boolean;
	stats_visible: boolean;
	allow_follow: boolean;
	allow_private_chat: PrivateChatPermission;
	searchable: boolean;
	phone_searchable: boolean;
	nickname_searchable: boolean;
	location_searchable: boolean;
}

/**
 * 我的资料（/users/me/profile/get 返回）
 * user_type 仅在此处返回，UserInfo 不包含
 */
export interface MyProfile {
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
	last_online: string;
	created_at: string;
	stats: UserStats;
}

/** 好友列表项 */
export interface FriendItem {
	user: UserInfo;
	presence_status?: UserPresenceStatus;
	last_online?: string;
	remark?: string;
	followed_at: string;
	followed_back_at: string;
}

/** 关注列表项 */
export interface FollowingItem {
	user: UserInfo;
	followed_at: string;
	is_mutual: boolean;
	presence_status?: UserPresenceStatus;
	last_online?: string;
}

/** 粉丝列表项 */
export interface FollowerItem {
	user: UserInfo;
	followed_at: string;
	is_following_back: boolean;
	presence_status?: UserPresenceStatus;
	last_online?: string;
}

/** 拉黑列表项 */
export interface BlockedUserItem {
	user: UserInfo;
	blocked_at: string;
	reason?: string;
}

// ============================================================================
// 群组类型（common.yaml + groups.yaml）
// ============================================================================

/** 群组基础信息（用于嵌套在其他对象中） */
export interface GroupInfo {
	group_id: string;
	name: string;
	avatar_url?: string;
	members_count: number;
}

/** 群组设置 */
export interface GroupSettings {
	join_method: GroupJoinMethod;
	is_searchable: boolean;
	max_members: number;
	location: Location;
}

/** 我在群内的状态（@auth-only） */
export interface MyGroupState {
	role: GroupRole;
	is_joined: boolean;
	unread_count: number;
	is_muted: boolean;
	is_pinned: boolean;
	joined_at?: string;
}

/**
 * 群组统一数据类型（用于列表、搜索、详情、Feed 等所有场景）
 * 替代原 GroupSearchItem/GroupCard/FeedGroupItem/MyGroupItem
 */
export interface GroupItem {
	group: GroupInfo;
	description?: string;
	tags?: string[];
	has_password: boolean;
	location: RelativeLocation;
	last_activity_at?: string;
	created_at?: string;
	owner_id?: string;
	// @auth-only
	my_state?: MyGroupState;
	friends_in_group?: number;
	// @group-admin-only
	join_method?: GroupJoinMethod;
	is_searchable?: boolean;
	max_members?: number;
}

/** 群公告列表项 */
export interface GroupAnnouncementItem {
	announcement_id: string;
	group_id: string;
	title: string;
	content: string;
	created_by: UserInfo;
	created_at: string;
	is_pinned: boolean;
}

/** 群组成员列表项 */
export interface GroupMemberItem {
	user: UserInfo;
	role: GroupRole;
	joined_at: string;
	muted_until?: string;
	presence_status?: UserPresenceStatus;
	last_online?: string;
}

/** 封禁成员列表项 */
export interface BannedMemberItem {
	user: UserInfo;
	banned_at: string;
	reason?: string;
	banned_by?: UserInfo;
}

/** 群组申请列表项 */
export interface GroupApplicationItem {
	application_id: string;
	applicant: UserInfo;
	message?: string;
	status: ApplicationStatus;
	created_at: string;
	handled_at?: string;
	handled_by?: UserInfo;
}

// MyApplicationItem 已删除：spec 声明 groups/me/applications/list 返回 GroupApplicationItem，以 spec 为准

/** 群组邀请列表项 */
export interface GroupInviteItem {
	invite_id: string;
	group: GroupInfo;
	inviter: UserInfo;
	message?: string;
	status: ApplicationStatus;
	created_at: string;
	expires_at?: string;
}

/** 邀请响应 */
export interface Invitation {
	invite_id: string;
	expires_at: string;
}

/** 申请响应 */
export interface Application {
	application_id: string;
}

/** 加入群组响应 */
export interface JoinGroupResult {
	status: 'joined' | 'pending_approval';
	application?: Application;
}

// ============================================================================
// 瞬间类型（common.yaml + moments.yaml）
// ============================================================================

/** 瞬间摘要快照（用于通知、WebSocket 事件等场景的轻量级预览） */
export interface MomentSummary {
	moment_id: string;
	cover_url?: string;
	text_preview: string;
}

/**
 * 瞬间统一数据类型（用于列表、搜索、详情、Feed、用户瞬间历史等所有场景）
 * 替代原 MomentBase/MomentSearchItem/MomentCard/FeedMomentItem/UserMomentListItem
 */
export interface MomentItem {
	moment_id: string;
	description: string;
	images?: MomentImage[];
	tags: string[];
	mentions?: string[];
	created_at: string;
	likes_count: number;
	comments_count: number;
	shares_count: number;
	user: UserInfo;
	location: RelativeLocation;
	// @owner-only
	privacy_level?: PrivacyLevel;
	// @auth-only
	my_state?: {
		is_liked: boolean;
		is_favorited: boolean;
	};
	// Feed 场景返回
	recommendation?: RecommendationContext;
}

/** 点赞列表项（瞬间/评论共用） */
export interface LikeItem {
	user: UserInfo;
	liked_at: string;
}

// ============================================================================
// 评论类型（moments.yaml）
// ============================================================================

/** 评论项 */
export interface CommentItem {
	comment_id: string;
	user: UserInfo;
	content: string;
	parent_id?: string;
	mentions?: string[];
	created_at: string;
	likes_count: number;
	replies_count: number;
	// @auth-only
	is_liked?: boolean;
}

// ============================================================================
// 消息类型（chat.yaml）
// ============================================================================

/** 回复引用快照 */
export interface MessageReplyTo {
	message_id: string;
	user_id: string;
	nickname: string;
	content: string;
	share_code?: string;
}

/** 消息发送响应（私聊/群聊共用） */
export interface MessageSent {
	message_id: string;
	sent_at: string;
}

/** 私聊消息项 */
export interface PrivateMessageItem {
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
}

/** 群聊消息项（使用 sender: UserInfo） */
export interface GroupMessageItem {
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
}

/**
 * 私聊统一数据类型（用于列表、详情等所有场景）
 * 替代原 PrivateChatListItem/Conversation
 */
export interface PrivateChatItem {
	other_user: UserInfo;
	unread_count: number;
	is_muted: boolean;
	is_pinned: boolean;
	last_activity_at: string;
	can_message: boolean;
	cant_message_reason?: CantMessageReason;
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
}

/** 私聊详情响应 */
export interface PrivateChatOpenResult {
	chat: PrivateChatItem;
	recent_messages: PaginatedData<PrivateMessageItem>;
}

// ============================================================================
// 通知类型
// ============================================================================

export type NotificationPayload =
	| { type: 'new_follower'; actor: UserInfo; source?: FollowSource; source_moment_id?: string }
	| { type: 'mutual_follow'; actor: UserInfo }
	| { type: 'moment_liked'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary }
	| {
			type: 'moment_commented';
			actor: UserInfo;
			moment_id: string;
			moment_summary: MomentSummary;
			comment_id: string;
			comment_preview: string;
	  }
	| {
			type: 'comment_liked';
			actor: UserInfo;
			comment_id: string;
			comment_preview: string;
			moment_id: string;
			moment_summary?: MomentSummary;
	  }
	| {
			type: 'comment_reply';
			actor: UserInfo;
			comment_id: string;
			comment_preview: string;
			moment_id: string;
			moment_summary?: MomentSummary;
	  }
	| {
			type: 'mentioned';
			actor: UserInfo;
			target_type: 'moment' | 'comment';
			moment_id: string;
			moment_summary?: MomentSummary;
			comment_id?: string;
			comment_preview?: string;
	  }
	| {
			type: 'group_invite_received';
			actor: UserInfo;
			group_id: string;
			group_name: string;
			invite_id: string;
	  }
	| {
			type: 'group_application_accepted';
			group_id: string;
			group_name: string;
			application_id: string;
	  }
	| { type: 'system_announcement'; announcement_id: string; severity: AnnouncementSeverity };

export interface NotificationItem {
	notification_id: string;
	type: NotificationType;
	is_read: boolean;
	created_at: string;
	payload: NotificationPayload;
}

export interface UnreadCountByType {
	social: number;
	interaction: number;
	group: number;
	system: number;
}

export interface UnreadCountResult {
	total: number;
	by_type: UnreadCountByType;
}

export interface NotificationListRequest {
	type?: NotificationType;
	pagination?: PaginationParams;
}

export interface MarkAsReadRequest {
	notification_ids: string[];
}

// ============================================================================
// 地图类型（map.yaml）
// ============================================================================

export interface MapRenderItem {
	render_id: string;
	type: MapItemType;
	location: GeoPoint;
	cluster?: {
		count: number;
		label?: string;
		preview_urls?: string[];
	};
	entity?: {
		title: string;
		subtitle?: string;
		cover_url?: string;
		payload_id: string;
		category?: string;
	};
}

export interface MapCanvas {
	items: MapRenderItem[];
	toast_message?: string;
}

// ============================================================================
// 标签与搜索类型（search.yaml + feed.yaml）
// ============================================================================

export interface TagSuggestItem {
	name: string;
	usage_count: number;
}

/** TagItem = TagSuggestItem + optional trend */
export interface TagItem {
	name: string;
	usage_count: number;
	trend?: TagTrend;
}

/** 热门标签项（trend 必填，扩展自 TagItem） */
export interface TrendingTagItem {
	name: string;
	usage_count: number;
	trend: TagTrend;
	hot_moment_preview?: MomentItem;
}

export interface TrendingBundle {
	trending_tags: TrendingTagItem[];
	hot_moments: MomentItem[];
}

export interface UserSuggestItem {
	user_id: string;
	nickname: string;
	avatar_url?: string;
	bio?: string;
	location?: RelativeLocation;
}

export interface GroupSuggestItem {
	group_id: string;
	name: string;
	avatar_url?: string;
	description?: string;
	members_count: number;
	location?: RelativeLocation;
}

export interface MomentSuggestItem {
	moment_id: string;
	description_preview: string;
	author: UserInfo;
	location?: RelativeLocation;
}

export type SearchSuggestItem =
	| { type: 'tag'; payload: TagSuggestItem }
	| { type: 'user'; payload: UserSuggestItem }
	| { type: 'group'; payload: GroupSuggestItem }
	| { type: 'moment'; payload: MomentSuggestItem };

export interface SearchSuggestResponse {
	items: SearchSuggestItem[];
}

// ============================================================================
// API 响应类型
// ============================================================================

export interface ApiResponse<T = void> {
	code: number;
	message?: string;
	data?: T;
	request_id?: string;
}

// ============================================================================
// 认证类型（auth.yaml）
// ============================================================================

/**
 * 前端 Auth Store 使用的认证状态（仅 Web 端）
 *
 * spec: Web 端 auth 响应体为 AuthTokenPairWeb = Record<string, never>（空对象）
 * Token 通过 HttpOnly Cookie 传递，前端不可见
 * user_id/user_type/expires_at 通过 check_status + users/me/profile/get 获取
 */
export interface AuthData {
	user_id: string;
	user_type: UserType;
	access_token_expires_at: string;
}

export interface LoginRequest {
	login_id: string;
	password: string;
	client_type: ClientType;
}

export interface RegisterRequest {
	login_id: string;
	nickname: string;
	password: string;
	client_type: ClientType;
}

export interface GuestRequest {
	nickname: string;
	client_type: ClientType;
}

export interface RefreshTokenRequest {
	client_type: ClientType;
	refresh_token?: string;
}

export interface AuthStatus {
	is_valid: boolean;
	expires_at?: string;
}

export interface RevokeAllResult {
	revoked_count: number;
}

/** 验证码发送成功响应（用于 email/phone 变更请求） */
export interface VerificationSent {
	expires_at: string; // 验证码过期时间（UTC ISO 8601）
}

// ============================================================================
// 举报类型（report.yaml）
// ============================================================================

export interface ReportCreated {
	report_id: string;
}

export interface ReportItem {
	report_id: string;
	target_type: ReportTargetType;
	target_id: string;
	reason: ReportReason;
	description?: string;
	status: ReportStatus;
	created_at: string;
	handled_at?: string;
}

// ============================================================================
// 管理员类型（admin-core.yaml + admin-moderation.yaml）
// ============================================================================

// === RBAC 类型 ===

export interface AdminPermissionItem {
	key: string;
	description: string;
}

export interface AdminRoleBasic {
	role_id: string;
	name: string;
}

export interface AdminRoleItem {
	role_id: string;
	name: string;
	description?: string;
	permissions: string[];
}

export interface AdminPermissions {
	permissions: string[]; // '*' 表示拥有全部权限
	roles: AdminRoleBasic[];
}

export interface AdminOperatorItem {
	user: UserInfo;
	permissions: string[]; // '*' 表示拥有全部权限
	roles: AdminRoleBasic[];
}

// === 审计日志类型 ===

export type AuditAction =
	| 'user.ban'
	| 'user.unban'
	| 'group.dismiss'
	| 'group.member_mute'
	| 'group.member_ban'
	| 'group.member_remove'
	| 'announcement.create'
	| 'announcement.delete'
	| 'role.create'
	| 'role.update'
	| 'role.delete'
	| 'role.set_permissions'
	| 'admin.grant'
	| 'admin.revoke'
	| 'report.handle'
	| 'tag.delete'
	| 'permission.grant'
	| 'permission.revoke'
	| 'config.update';

export type AuditTargetType =
	| 'user'
	| 'group'
	| 'announcement'
	| 'role'
	| 'admin'
	| 'report'
	| 'tag'
	| 'config';

/** 审计详情可辨识联合（type 与 AuditAction 一一对应） */
export type AuditDetails =
	| { type: 'user.ban'; reason?: string; duration_days?: number }
	| { type: 'user.unban'; reason?: string }
	| { type: 'group.dismiss'; reason?: string }
	| { type: 'group.member_mute'; reason?: string; duration_minutes?: number }
	| { type: 'group.member_ban'; reason?: string }
	| { type: 'group.member_remove'; reason?: string }
	| { type: 'announcement.create'; title: string }
	| { type: 'announcement.delete'; title: string }
	| { type: 'role.create'; role_name: string; permissions: string[] }
	| { type: 'role.update'; role_name: string; added_permissions?: string[]; removed_permissions?: string[] }
	| { type: 'role.delete'; role_name: string }
	| { type: 'role.set_permissions'; role_name: string; permissions: string[] }
	| { type: 'admin.grant'; target_user_id: string; role_ids: string[]; grant_all_permissions?: boolean }
	| { type: 'admin.revoke'; target_user_id: string }
	| { type: 'report.handle'; report_id: string; action: string; note?: string }
	| { type: 'tag.delete'; name: string }
	| { type: 'permission.grant'; permission: string }
	| { type: 'permission.revoke'; permission: string }
	| { type: 'config.update'; config_key: string; old_value: unknown; new_value: unknown };

export interface AuditLogItem {
	log_id: string;
	action: AuditAction;
	operator: UserInfo;
	target_type: AuditTargetType;
	target_id: string;
	target_name?: string;
	details?: AuditDetails;
	ip_location?: string;
	ip_address?: string;
	device_summary?: string;
	created_at: string;
}

// === Admin 扩展类型（管理员视角的完整数据）===

export interface AdminUserItem extends UserItem {
	status: UserAccountStatus;
	created_at: string;
	last_active_at?: string;
	login_id: string;
	register_ip?: string;
	register_location?: string;
	last_login_ip?: string;
	last_login_location?: string;
	email?: string;
	phone?: string;
}

export interface AdminGroupItem extends GroupItem {
	owner: UserInfo;
	admins: UserInfo[];
	stats: {
		messages_count: number;
		active_members_today: number;
		reports_count?: number;
		last_reported_at?: string;
	};
}

export interface AdminReportItem extends ReportItem {
	reporter_user_id: string;
	reporter: UserInfo;
	handled_by?: UserInfo;
	handled_action?: ReportHandleAction;
	handled_note?: string;
	target_snapshot?: {
		content?: string;
		media_urls?: string[];
		owner_id?: string;
	};
}

export interface AdminAnnouncementItem extends SystemAnnouncementItem {
	created_by: UserInfo;
	send_push: boolean;
	push_sent_at?: string;
	view_count: number;
}

export interface AdminTagItem {
	name: string;
	usage_count: number;
	created_at: string;
	last_used_at?: string;
}

export type AdminTagSortBy =
	| 'usage_count_desc'
	| 'usage_count_asc'
	| 'created_at_desc'
	| 'last_used_at_desc';

export interface SystemAnnouncementItem {
	announcement_id: string;
	title: string;
	content: string;
	severity: AnnouncementSeverity;
	action_url?: string;
	created_at: string;
}

export interface AnnouncementCreated {
	announcement_id: string;
}

// ============================================================================
// 系统配置类型（system.yaml）
// ============================================================================

export interface ClientConfig {
	features: {
		enable_public_registration: boolean;
		enable_guest_login: boolean;
		enable_moments: boolean;
		enable_nearby_people: boolean;
	};
	limits: {
		max_group_members: number;
		max_moment_tags: number;
		max_moment_images: number;
		moment_description_max_length: number;
		comment_max_length: number;
		search_radius_max: number;
	};
	contact: {
		support_email: string;
		privacy_policy_url: string;
		terms_of_service_url: string;
		official_website: string;
	};
	defaults: {
		default_location: {
			point: { latitude: number; longitude: number };
			location_name: string;
		};
		default_user_avatar: string;
		default_group_avatar: string;
	};
}

export interface UploadConfig {
	image_max_size: number;
	video_max_size: number;
	audio_max_size: number;
	file_max_size: number;
	allowed_image_types: string[];
	allowed_video_types: string[];
}

export interface ContentDefaults {
	default_location: {
		point: { latitude: number; longitude: number };
		location_name: string;
	};
	default_group_avatar: string;
	default_user_avatar: string;
	default_group_settings: GroupSettings;
}

export interface RateLimitConfig {
	global_per_hour: number;
	user_per_hour: number;
	auth_register_per_10minutes: number;
	auth_login_per_10minutes: number;
	auth_guest_per_10minutes: number;
	auth_refresh_per_10minutes: number;
	auth_reset_per_10minutes: number;
	moments_create_per_minute: number;
	comments_create_per_minute: number;
	reports_create_per_hour: number;
}

export interface ResponseDefaults {
	search_suggest_count: number;
	trending_tags_count: number;
	hot_moments_count: number;
	chat_recent_messages_count: number;
	group_announcements_count: number;
	system_announcements_count: number;
}

export interface ServerConfig {
	upload: UploadConfig;
	defaults: ContentDefaults;
	rate_limits: RateLimitConfig;
	response_defaults: ResponseDefaults;
	updated_at?: string;
	updated_by_user_id?: string;
}

/**
 * 完整应用配置（admin/app_config/get 返回）
 * spec: AppConfig 包含 ClientConfig 的所有客户端可见字段 + 仅服务端使用字段 + 元数据
 */
export interface AppConfig {
	// 客户端可见配置（构成 ClientConfig）
	features: ClientConfig['features'];
	limits: ClientConfig['limits'];
	contact: ClientConfig['contact'];
	// 仅服务端使用
	upload: UploadConfig;
	defaults: ContentDefaults;
	rate_limits: RateLimitConfig;
	response_defaults: ResponseDefaults;
	// 元数据
	updated_at?: string;
	updated_by_user_id?: string;
}

export type ApiVersionStatus = 'active' | 'deprecated' | 'sunset';

export interface ApiVersionMeta {
	version: string;
	status: ApiVersionStatus;
	deprecated_at?: string;
	sunset_at?: string;
	message?: string;
	recommended_version?: string;
}

export interface HealthStatus {
	status: SystemHealthStatus;
	timestamp: string;
}

// ============================================================================
// 会话管理类型（auth.yaml）
// ============================================================================

export interface SessionItem {
	session_id: string;
	device_type?: string;
	device_name?: string;
	ip_address?: string;
	ip_location?: string;
	created_at: string;
	last_active_at: string;
	is_current: boolean;
}

// ============================================================================
// 分享系统（share.yaml）
// ============================================================================

export interface LocationSharePayload {
	shared_location: Location;
}

export interface MomentSharePayload {
	shared_moment_id: string;
}

export interface GroupSharePayload {
	shared_group_id: string;
}

export interface UserSharePayload {
	shared_user_id: string;
}

export type SharePayload =
	| LocationSharePayload
	| MomentSharePayload
	| GroupSharePayload
	| UserSharePayload;

export interface ShareBase {
	share_id: string;
	share_code: string;
	share_type: ShareType;
	creator: UserInfo;
	created_at: string;
	expires_at?: string;
}

export interface ShareCreated {
	share_id: string;
	share_code: string;
	expires_at?: string;
}

export interface SharePreview {
	title: string;
	subtitle?: string;
	cover_url?: string;
}

export interface ShareDetail {
	share: ShareBase;
	payload: SharePayload;
	preview?: SharePreview;
}

export interface ShareListItem {
	share_id: string;
	share_code: string;
	share_type: ShareType;
	created_at: string;
	expires_at?: string;
	is_revoked: boolean;
	preview?: SharePreview;
}

export interface CreateShareRequest {
	share_type: ShareType;
	payload: SharePayload;
	expires_in_seconds?: number;
}

export interface GetShareRequest {
	share_code: string;
}

