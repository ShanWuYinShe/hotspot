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
export type ReportStatus = 'pending' | 'resolved' | 'dismissed';
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
export type BlockReason = 'blocked' | 'privacy' | 'stranger_quota_exceeded';
export type AnnounceLevel = 'info' | 'warning' | 'error';
export type NotifyCategory = 'social' | 'interaction' | 'group' | 'report' | 'nearby' | 'system';
export type NotifyType =
	| 'new_follower'
	| 'mutual_follow'
	| 'moment_commented'
	| 'comment_reply'
	| 'mentioned'
	| 'group_mentioned'
	| 'reaction_added'
	| 'group_invite_received'
	| 'group_application_received'
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
	last_message?: LastMessagePreview; // 群列表场景返回（仅 groups/list_my）；LastMessagePreview 定义见消息节
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

export interface JoinGroupResult {
	// spec: api_spec_group.md → POST /groups/join
	status: 'joined' | 'pending_approval';
	application?: { application_id: string }; // 仅 pending_approval 时返回
}

// --- 群组管理（spec: api_spec_group.md）---
// set_role 可设置的角色（owner 不可设置）
export type SettableGroupRole = 'admin' | 'member';

// 入群申请项（管理员 applications/list、申请人 list_applications 复用）
export interface GroupApplicationItem {
	application_id: string;
	group: GroupInfo;
	applicant: UserInfo;
	message?: string;
	status: ApplicationStatus;
	created_at: string;
	expires_at?: string;
	handled_at?: string;
	handled_by?: UserInfo;
}

// 收到的邀请项（invites/received/list）
export interface GroupInviteItem {
	invite_id: string;
	group: GroupInfo;
	inviter: UserInfo;
	message?: string;
	status: ApplicationStatus;
	created_at: string;
	expires_at?: string;
	responded_at?: string;
}

// 已发出的邀请项（invites/sent/list，群主/管理员视角）
export interface GroupSentInviteItem {
	invite_id: string;
	invitee: UserInfo;
	inviter: UserInfo;
	status: ApplicationStatus;
	message?: string;
	created_at: string;
	expires_at?: string;
	responded_at?: string;
}

// 封禁成员项（members/list_banned）
export interface BannedMemberItem {
	user: UserInfo;
	banned_at: string;
	reason?: string;
	banned_by?: UserInfo;
}

// members/invite 返回（spec: group.md → Invitation）
// invite_id/expires_at 可选：被邀请者已是成员（no-op）时省略；TTL=0（永不过期）时 expires_at 省略
export interface Invitation {
	invite_id?: string;
	expires_at?: string;
}

// invites/received/accept 返回
export interface InviteAcceptedResult {
	group_id: string;
}

// --- 群组管理 Request 类型（逐端点对齐 spec）---
export interface SearchGroupsRequest {
	keyword?: string;
	tag?: string;
	radius?: number;
	sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
	reference_point?: GeoPoint;
	pagination?: PaginationParams;
}

export interface UpdateGroupRequest {
	group_id: string;
	name?: string;
	description?: string;
	avatar_file_id?: string;
	tags?: string[];
	password?: string;
	clear_password?: boolean;
	join_method?: GroupJoinMethod;
	requires_approval?: boolean;
	is_searchable?: boolean;
	members_searchable?: boolean; // 是否允许非成员检索群成员列表（groups/members/search）
	max_members?: number;
	location?: Location;
}

// spec: group.md → groups/create
export interface CreateGroupRequest {
	name: string;
	description?: string;
	avatar_file_id?: string;
	tags?: string[];
	password?: string;
	join_method?: GroupJoinMethod; // 默认 'direct'
	requires_approval?: boolean;   // 默认 false
	is_searchable?: boolean;       // 默认 true
	members_searchable?: boolean;  // 默认 true
	max_members?: number;          // 默认 200
	location: Location;
}

export interface DismissGroupRequest {
	group_id: string;
	reason?: string;
}

export interface TransferOwnershipRequest {
	group_id: string;
	new_owner_id: string;
}

export interface SearchGroupMembersRequest {
	group_id: string;
	keyword: string;
	role?: GroupRole;
	pagination?: PaginationParams;
}

export interface InviteMemberRequest {
	group_id: string;
	user_id: string;
	message?: string;
}

export interface GroupSentInvitesRequest {
	group_id: string;
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface RevokeSentInviteRequest {
	group_id: string;
	invite_id: string;
	reason?: string;
}

export interface SetMemberRoleRequest {
	group_id: string;
	user_id: string;
	role: SettableGroupRole;
}

export interface SetMuteRequest {
	group_id: string;
	user_id: string;
	active: boolean;
	duration_minutes?: number;
}

export interface SetMemberBanRequest {
	group_id: string;
	user_id: string;
	active: boolean;
	reason?: string;
}

export interface BannedMembersRequest {
	group_id: string;
	pagination?: PaginationParams;
}

export interface RemoveMemberRequest {
	group_id: string;
	user_id: string;
}

export interface GroupApplicationsRequest {
	group_id: string;
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface ApproveGroupApplicationRequest {
	group_id: string;
	application_id: string;
}

export interface RejectGroupApplicationRequest {
	group_id: string;
	application_id: string;
	reason?: string;
}

export interface CreateGroupAnnouncementRequest {
	group_id: string;
	title: string;
	content: string;
	is_pinned?: boolean;
}

export interface UpdateGroupAnnouncementRequest {
	group_id: string;
	announcement_id: string;
	title?: string;
	content?: string;
	is_pinned?: boolean;
}

export interface DeleteGroupAnnouncementRequest {
	group_id: string;
	announcement_id: string;
}

export interface MyApplicationsRequest {
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface CancelApplicationRequest {
	group_id: string;
	application_id: string;
	reason?: string;
}

export interface MyInvitesRequest {
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface AcceptInviteRequest {
	invite_id: string;
}

export interface RejectInviteRequest {
	invite_id: string;
	reason?: string;
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
	is_deleted: boolean; // 软删除标记；true 时 description 为空，客户端展示"内容已删除"占位
}

// 收藏列表项（spec: moment.md → FavoritedMomentItem，list_favorites 返回）
export interface FavoritedMomentItem extends MomentItem {
	favorited_at: string; // 收藏时间（UTC ISO 8601）
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

// 会话最近消息预览（GroupItem.last_message 与 PrivateChatItem.last_message 共用，spec: types.md → LastMessagePreview）
export interface LastMessagePreview {
	message_id: string;
	type: MessageType;
	content?: string;    // 文本预览（仅 text 类型有意义）
	share_code?: string; // 最后一条消息为 share 类型时包含，客户端据此渲染分享卡片预览
	sender: UserInfo;
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
	message_id?: string;
	content_preview?: string;
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
	title?: string;        // system_announcement：公告标题（直接渲染通知行）
	severity?: AnnounceLevel;
	distance?: number;
}

export interface NotifyItem {
	// spec: api_spec_notify.md → NotifyItem
	notify_id: string;
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
	max_chat_files: number; // 单条聊天消息可附带的最大文件数（媒体批量上传的 chat 类上限）
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

// 分页请求参数（spec: api_spec_types.md → 分页与元数据）
export interface PaginationParams {
	limit?: number;  // default: 20, max: 100
	cursor?: string; // 不透明字符串，客户端只透传
}

// 分页信封在此重声明，结构等价于 client.ts 的 PaginatedData<T>，
// 供仅 import types.ts 的调用方使用，避免与 client 循环依赖。
export interface PaginatedData<T> {
	items: T[];
	next_cursor?: string;
}

// --- Auth（spec: api_spec_auth.md）---
// client_type 为判别字段：web 仅含判别字段（token 走 HttpOnly cookie）；app 含 token 字段
export interface AuthTokenPairApp {
	client_type: 'app';
	user_id: string;
	access_token: string;
	refresh_token: string;
	access_token_expires_at: string;
	refresh_token_expires_at: string;
}
export interface AuthTokenPairWeb {
	client_type: 'web';
}
export type AuthTokenPair = AuthTokenPairApp | AuthTokenPairWeb;
export interface AuthStatus {
	// spec: auth.md → check_status
	is_valid: boolean;
	user_id?: string;   // is_valid=true 时返回，客户端可直接获得身份
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
	// spec: user.md → open_the_door（include_recent_moments=true 时返回 recent_moments）
	user: UserItem;
	recent_moments?: PaginatedData<MomentItem>;
}

// 用户关系列表项（list_friends / list_following / list_followers 共用，spec: user.md → RelatedUserItem）
// 字段可见性按端点语义返回：followed_back_at 仅 list_friends；remark 仅查询自己时返回
export interface RelatedUserItem {
	user: UserItem;
	followed_at: string;
	followed_back_at?: string; // 对方回关时间（成为好友的时间）；仅 list_friends 返回
	is_mutual: boolean;
	remark?: string; // @owner-only：仅查询自己时返回
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
	// spec: system.md → bootstrap（缓存命中时 meta/client_config 省略，客户端用缓存）
	bootstrap_version: string;
	meta?: ApiVersionMeta;
	client_config?: ClientConfig;
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
	last_message?: LastMessagePreview; // 列表场景返回
	presence_status?: UserPresenceStatus;
	created_at?: string;
	can_message: boolean;
	message_blocked_reason?: BlockReason;
}
export interface PrivateChatOpenResult {
	// spec: chat.md → private_chats/open_the_door（纯查询，无创建副作用）
	chat: PrivateChatItem; // 会话不存在时返回空态对象（unread_count=0、is_muted/is_pinned=false）
	recent_messages: PaginatedData<PrivateMessageItem>; // 始终返回，会话不存在时 items 为空数组
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
	is_duplicate: boolean; // true 表示幂等回放（已有举报单），report_id 为已有举报的 ID
}

// --- Media（spec: api_spec_media.md）---
export type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

export interface PresignedUpload {
	// spec: media.md → presign_upload 响应
	upload_id: string;
	upload_url: string;
	headers: Record<string, string>;
	expires_at: string;
}

export interface UploadedMedia {
	// spec: media.md → complete_upload 响应
	file_id: string;
	thumbnail_url?: string;
	file_size: number;
	mime_type: string;
	duration?: number;
	width?: number;
	height?: number;
}

// spec: media.md → 批量上传（瞬间多图等场景，最多 9 个，原子操作）
export interface PresignUploadRequest {
	file_name: string;
	file_size: number;
	mime_type: string;
	category: MediaCategory;
	md5?: string; // 文件内容 MD5（可选，服务端用于校验完整性）
}

export interface BatchPresignUploadRequest {
	files: PresignUploadRequest[];
}

export interface BatchPresignedUpload {
	uploads: PresignedUpload[];     // 与请求 files 一一对应
}

export interface BatchCompleteUploadRequest {
	upload_ids: string[];
}

export interface BatchCompleteUpload {
	uploads: UploadedMedia[];       // 与请求 upload_ids 一一对应、顺序一致
}
