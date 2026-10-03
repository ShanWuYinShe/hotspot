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
	// spec: types.md → RelativeLocation（响应侧：坐标 + 名称 + 距离/方位，恒定 nullable）
	point: GeoPoint;
	location_name: string;
	distance: number | null; // 距浏览锚点的距离（米）；null = 无参考点/不可见
	bearing_degrees: number | null; // 相对浏览锚点的方位角 [0, 360)；null = 无参考点/不可见
}

export interface MediaAsset {
	// spec: types.md → MediaAsset（元数据字段恒定 nullable）
	url: string;
	thumbnail_url: string | null; // null = 无缩略图
	width: number | null; // null = 未知
	height: number | null; // null = 未知
	mime_type: string | null; // MIME 类型字符串，如 "image/jpeg"；null = 未知
}

export interface MessageMedia extends MediaAsset {
	// spec: types.md → MessageMedia（恒定 nullable：字段恒在，null = 不适用）
	file_name: string | null;
	file_size: number | null;
	duration: number | null;
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
export type ReportAction = 'resolve' | 'dismiss';
export type ReportStatus = 'pending' | 'resolved' | 'dismissed';
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
// spec: api_spec_errors.md → ForbiddenReason（10200 的 data.forbidden_reason，12 项权威枚举）
export type ForbiddenReason =
	| 'not_admin'
	| 'insufficient_admin_permission'
	| 'admin_protected_target'
	| 'not_group_member'
	| 'not_group_admin'
	| 'not_group_owner'
	| 'not_resource_owner'
	| 'privacy_setting'
	| 'blocked_by_user'
	| 'stranger_quota_exceeded'
	| 'banned_in_group';
export type AnnounceSeverity = 'info' | 'warning' | 'error';
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
// spec: types.md → RecommendationType（Feed 推荐理由）
export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

// ─────────────────────────── 领域实体 ───────────────────────────

export interface UserInfo {
	// spec: types.md → UserInfo
	user_id: string;
	nickname: string;
	avatar_url: string | null; // null = 未设置头像（客户端使用默认头像）
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
	// spec: types.md → SocialContext（查看者相关，恒定 nullable）
	mutual_friends_count: number;
	mutual_friends_preview: UserInfo[]; // 前2-3个共同好友
	common_groups_count: number | null;
	common_tags: string[];
}

export interface UserItem {
	// spec: types.md → UserItem（list/search/detail/feed；恒定 nullable 通则：权限/隐私字段恒在，null=隐藏/不适用）
	user: UserInfo;
	bio?: string;
	tags: string[];
	presence_status: UserPresenceStatus | null;
	last_online_at: string | null;
	location: RelativeLocation | null;
	stats: UserStats | null;
	relation: Relationship | null;
	social_context: SocialContext | null;
}

export interface GroupInfo {
	// spec: types.md → GroupInfo（恒定 nullable）
	group_id: string;
	name: string;
	avatar_url: string | null;
	members_count: number;
}

export interface GroupViewerState {
	// spec: types.md → GroupViewerState（GroupItem.viewer_state 的取值类型；viewer_state 为 null 即非成员）
	role: GroupRole;
	unread_count: number;
	is_muted: boolean;
	muted_until: string | null;
	is_pinned: boolean;
	joined_at: string; // spec: 必填非空（成员必有入群时间）
}

export interface GroupItem {
	// spec: types.md → GroupItem（公开固定形状；群管理字段 is_searchable/max_members/has_password 见 GroupSettings）
	group: GroupInfo;
	description: string | null;
	tags: string[];
	join_method: GroupJoinMethod;
	requires_approval: boolean;
	location: RelativeLocation | null;
	last_activity_at: string | null;
	online_members_count: number | null;
	next_activity_at: string | null;
	last_message: MessageBrief | null; // 仅 groups/fetch_joined 返回
	created_at: string | null;
	owner_id: string | null;
	viewer_state: GroupViewerState | null;
	friends_in_group: number | null;
}

export interface GroupSettings {
	// spec: types.md → GroupSettings（仅群管理守卫子树 groups/admin/settings 返回）
	join_method: GroupJoinMethod;
	requires_approval: boolean;
	has_password: boolean;
	is_searchable: boolean;
	members_searchable: boolean;
	max_members: number;
	location: Location;
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
	// spec: api_spec_group.md → members/list（公开形状；管理字段见 AdminGroupMemberItem）
	user: UserInfo;
	role: GroupRole;
	joined_at: string;
	presence_status: UserPresenceStatus | null;
	last_online_at: string | null;
}

export interface AdminGroupMemberItem {
	// spec: api_spec_group.md → groups/admin/members/list（muted_until 真值，null=未禁言）
	user: UserInfo;
	role: GroupRole;
	joined_at: string;
	muted_until: string | null;
}

export interface JoinGroupResult {
	// spec: api_spec_group.md → POST /groups/enter
	status: 'joined' | 'pending_approval';
	application_id: string | null; // 仅 pending_approval 时非 null
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
export interface AdminBannedMemberItem {
	user: UserInfo;
	banned_at: string;
	reason: string | null;
	banned_by: UserInfo | null;
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
	description?: string | null;
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
	description?: string | null;
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

export interface CreateGroupInviteRequest {
	group_id: string;
	user_id: string;
	message?: string;
}

export interface ListSentInvitesRequest {
	group_id: string;
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface RevokeGroupInviteRequest {
	group_id: string;
	invite_id: string;
	reason?: string;
}

export interface SetMemberRoleRequest {
	group_id: string;
	user_id: string;
	role: SettableGroupRole;
}

export interface SetMemberMuteRequest {
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

export interface ListSubmittedApplicationsRequest {
	status?: ApplicationStatus | 'all';
	pagination?: PaginationParams;
}

export interface CancelApplicationRequest {
	group_id: string;
	application_id: string;
	reason?: string;
}

export interface ListReceivedInvitesRequest {
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
	// spec: types.md → MomentItem（公开固定形状；拥有者视角见 OwnerMomentItem）
	moment_id: string;
	description: string;
	images: MediaAsset[];
	tags: string[];
	mentions: string[];
	created_at: string;
	comments_count: number;
	shares_count: number;
	reactions: Reaction[];
	user: UserInfo;
	location: RelativeLocation | null;
	is_favorited: boolean; // 恒定布尔：匿名/未收藏为 false
	recommendation: RecommendationType | null; // Feed 场景返回（推荐理由）；非 Feed 场景为 null
	is_deleted: boolean; // 软删除标记；true 时 description 为空，客户端展示"内容已删除"占位
}

export interface OwnerMomentItem extends MomentItem {
	// spec: api_spec_moment.md → OwnerMomentItem（moments/fetch_created、create、update 返回）
	privacy_level: PrivacyLevel;
}

// 收藏列表项（spec: moment.md → FavoritedMomentItem，list_favorites 返回）
export interface FavoritedMomentItem extends MomentItem {
	favorited_at: string; // 收藏时间（UTC ISO 8601）
}

export interface CommentItem {
	// spec: api_spec_moment.md → comments/create & comments/fetch
	comment_id: string;
	user: UserInfo;
	content: string;
	parent_id: string | null; // 父评论 ID；null = 顶层评论
	mentions: string[];
	created_at: string;
	replies_count: number;
	reactions: Reaction[];
	is_deleted: boolean;
	top_replies: CommentItem[]; // 仅 comments/fetch 顶层列表返回前 N 条回复预览；其他端点为空数组
}

export interface MessageBrief {
	// spec: types.md → MessageBrief（reply_to 回复引用与 last_message 会话最近消息预览共用）
	message_id: string;
	type: MessageType;
	content: string | null;    // 文本预览（仅 text 类型有意义）
	share_code: string | null; // share 类型消息包含分享码
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
	reply_to?: MessageBrief | null;
	sent_at: string;
	edited_at?: string;
	recalled_at?: string;
	is_read: boolean;
	reactions: Reaction[];
}

export interface GroupMessageItem {
	// spec: api_spec_chat.md → GroupMessageItem
	message_id: string;
	type: MessageType;
	content?: string;
	media?: MessageMedia;
	share_code?: string;
	sender: UserInfo;
	reply_to?: MessageBrief | null;
	mentions: string[];
	recalled_by?: UserInfo;
	sent_at: string;
	edited_at?: string;
	recalled_at?: string;
	read_count?: number;
	reactions: Reaction[];
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
	severity?: AnnounceSeverity;
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
	enable_guest_register: boolean;
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
	next_cursor: string | null; // 恒定 nullable：null = 没有更多数据
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
export interface SessionStatus {
	// spec: auth.md → session
	user_id: string;
	expires_at: string;
}

// 会话管理（spec: auth.md → fetch_sessions / revoke_session / revoke_other_sessions）
export interface SessionItem {
	session_id: string;         // 会话标识，用于展示和撤销操作
	device_id: string;          // 设备标识（会话主键的一部分）
	device_type: string | null; // 设备类型展示（如 "iOS" / "Android" / "Web"）
	device_name: string | null; // 设备名称展示（如 "iPhone 15 Pro" / "Chrome on macOS"）
	ip_address: string | null;  // 脱敏显示
	ip_location: string | null; // IP 地理位置
	created_at: string;
	last_active_at: string;
	is_current: boolean;        // 当前请求所属会话
}
export interface SessionList {
	items: SessionItem[];
}
export interface ChangePasswordRequest {
	old_password: string;
	new_password: string;
}
export interface RevokeSessionRequest {
	session_id: string;
}
export interface RevokeOtherSessionsResult {
	revoked_count: number;
}

// --- Users（spec: api_spec_user.md）---
export interface AccountData {
	// spec: user.md → get_account（恒定 nullable：可选值显式 | null，非省略）
	user_id: string;
	nickname: string;
	avatar_url: string | null;
	user_type: UserType;
	bio: string | null;
	tags: string[];
	gender: Gender | null;
	email: string | null;
	phone: string | null;
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
// 互相关注由 user.relation 计算（is_following && is_followed_by），无冗余字段
export interface RelatedUserItem {
	user: UserItem;
	followed_at: string;
	followed_back_at: string | null; // 对方回关时间（成为好友的时间）；仅 list_friends 返回，其余 null
	remark: string | null; // 仅查询自己（user_id=当前用户）时返回；查他人 null
}

// 批量用户解析结果（users/batch_fetch 响应，spec: types.md → UsersBatchResult；
// api 版与 public 版共用，无查看者字段的纯 UserInfo 映射）
export interface UsersBatchResult {
	users: Record<string, UserInfo>; // user_id → UserInfo；不存在的 user_id 不出现在 map 中
}

// --- Notify（spec: api_spec_notify.md）---
export interface UnreadSummaryResult {
	// spec: notify.md → unread_summary
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
	deprecated_at: string | null; // 进入废弃状态时间；未废弃为 null
	sunset_at: string | null; // 计划下线时间；无下线计划为 null
	message: string | null; // 面向客户端的提示信息
	recommended_version: string | null; // 推荐升级的版本号
}
export interface UnreadSummary {
	// spec: system.md → bootstrap
	notify_total: number;
	notify_by_type: Record<NotifyCategory, number>;
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
	// spec: chat.md → private_chats/fetch
	other_user: UserInfo;
	unread_count: number;
	is_muted: boolean;
	is_pinned: boolean;
	last_activity_at: string;
	last_message?: MessageBrief | null; // 列表场景返回
	presence_status: UserPresenceStatus | null;
	created_at: string | null;
	can_message: boolean;
	forbidden_reason: ForbiddenReason | null; // 仅 can_message=false 时非 null
}
export interface PrivateChatOpenResult {
	// spec: chat.md → private_chats/open_the_door（纯查询，无创建副作用）
	chat: PrivateChatItem; // 会话不存在时返回空态对象（unread_count=0、is_muted/is_pinned=false）
	recent_messages: PaginatedData<PrivateMessageItem>; // 始终返回，会话不存在时 items 为空数组
}
export interface MessageSentResult {
	// spec: chat.md → MessageSentResult（private_messages/send、group_messages/send）
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

// spec: media.md → presign_upload/complete_upload 数组语义（1..N，无独立批量端点）
export interface PresignFileInput {
	file_name: string;
	file_size: number;
	mime_type: string;
	category: MediaCategory;
	md5?: string; // 文件内容 MD5（可选，服务端用于校验完整性）
}

export interface PresignUploadRequest {
	files: PresignFileInput[]; // 1..N；上限按类别配置
}

export interface PresignedUploads {
	uploads: PresignedUpload[]; // 与请求 files 一一对应且顺序一致
}

export interface CompleteUploadRequest {
	upload_ids: string[]; // 1..N
}

export interface CompletedUploads {
	uploads: UploadedMedia[]; // 与请求 upload_ids 一一对应、顺序一致
}

// ── Map Canvas ──────────────────────────────────────────────
// spec: api_spec_public.md → 地图渲染（map）（仅 public 版，无登录镜像）

// spec: api_spec_public.md → MapDataType（过滤器类别；'all' 或 null：用户/群组/动态都参与）
export type MapDataType = 'user' | 'group' | 'moment' | 'all';

// spec: api_spec_public.md → Bounds（约束 north >= south；east >= west，允许退化视口）
export interface Bounds {
	north: number; // [-90, 90]
	south: number; // [-90, 90]
	east: number; // [-180, 180]
	west: number; // [-180, 180]
}

// spec: api_spec_public.md → MapViewport（仅由经纬度边界决定，不需要客户端提供 zoom）
export interface MapViewport {
	bounds: Bounds;
}

// spec: api_spec_public.md → TimeRangeParams（UTC ISO 8601；同时提供时 start_time <= end_time）
export interface TimeRangeParams {
	start_time: string | null;
	end_time: string | null;
}

// spec: api_spec_public.md → MapCanvasFilter
export interface MapCanvasFilter {
	data_type: MapDataType | null;
	time_range: TimeRangeParams | null;
}

// spec: api_spec_public.md → POST /public/v1/map/canvas 请求
// 注：无 reference_point 字段。distance/bearing 的计算锚点 = viewport 几何中心。
export interface MapCanvasRequest {
	viewport: MapViewport;
	filter: MapCanvasFilter | null;
	// 仅在返回 hotspots 时使用；limit 覆盖全局默认为 default=50、max=200
	pagination: PaginationParams | null;
}

// spec: api_spec_public.md → MapHeatBreakdown（按类型拆分；data_type='all' 时返回完整三分类）
export interface MapHeatBreakdown {
	user: number | null; // 该网格内用户数量
	group: number | null; // 该网格内群组数量
	moment: number | null; // 该网格内动态数量
}

export interface MapHeatBucket {
	cell_token: string; // S2 单元格字符串 token（规避 JS 精度问题），可作 key 或调试
	level: number; // 该单元格所在的 S2 level
	location: GeoPoint; // 聚合单元的代表坐标（如 S2 单元格中心点）
	total_count: number; // 当前过滤条件下该网格内参与统计实体的总数
	breakdown: MapHeatBreakdown | null;
}

export interface MapHeatCanvas {
	items: MapHeatBucket[];
}

// spec: api_spec_public.md → HotspotObjectType
export type HotspotObjectType = 'user' | 'group' | 'moment';

export interface MapHotspotItem {
	object_type: HotspotObjectType; // 对象类型
	object_id: string; // 业务 ID（user_id / group_id / moment_id）
	title: string; // 昵称 / 群名 / 动态标题
	subtitle: string | null; // 简短描述 / 地点名等
	cover_url: string | null; // 头像 / 群头像 / 动态封面
	// 位置统一使用 RelativeLocation（含 distance/bearing），与 search 各端点对齐；
	// distance/bearing 相对于视口几何中心计算，由服务端计算
	location: RelativeLocation;
	created_at: string | null; // 对于动态等有时间语义的对象可返回创建时间
}

// 响应为 tagged union：模式由服务端按视口尺度与数据密度决定（大尺度→heatmap，小尺度→hotspots）；
// 视口内无任何已知对象时返回 { mode: 'empty' }
export type MapCanvasResponse =
	| { mode: 'heatmap'; heatmap: MapHeatCanvas }
	| { mode: 'hotspots'; hotspots: PaginatedData<MapHotspotItem> }
	| { mode: 'empty' };

// ── Search Suggest ──────────────────────────────────────────

// spec: api_spec_types.md → 搜索建议类型（支持按类型过滤）
export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

// spec: api_spec_types.md → TagItem（标签项，搜索建议等模块共用）
export interface TagItem {
	name: string;
	usage_count: number;
}

// spec: api_spec_types.md → MomentSummary（瞬间摘要：通知、WebSocket 推送等轻量展示场景）
export interface MomentSummary {
	moment_id: string;
	description_preview: string; // 截断的描述文本（最多 100 字符）
	cover_url: string | null; // 首张图片缩略图 URL；null = 无图片
}

// spec: api_spec_types.md → MomentSuggestItem（基于 MomentSummary 的轻量扩展，仅用于建议场景）
export interface MomentSuggestItem extends MomentSummary {
	author: UserInfo;
	location: RelativeLocation | null;
}

// spec: api_spec_discovery.md → POST /api/v1/search/suggest
// reference_point/radius 恒定 nullable：null 时服务端按「reference_point 解析规则」回退链解析
export interface SearchSuggestRequest {
	prefix: string;
	reference_point: GeoPoint | null;
	types: SearchSuggestType[]; // 空数组表示返回所有类型
	radius: number | null;
}

// 登录视角建议项判别联合：user/group 分支为 UserItem/GroupItem（含查看者状态）；
// 匿名版 PublicSearchSuggestResponse 见 api_spec_public.md
export type SearchSuggestItem =
	| { type: 'tag'; payload: TagItem }
	| { type: 'user'; payload: UserItem }
	| { type: 'group'; payload: GroupItem }
	| { type: 'moment'; payload: MomentSuggestItem };

export interface SearchSuggestResponse {
	items: SearchSuggestItem[];
}

// ── 请求参数类型（对齐后端 json tag）─────────────────────────

// Auth
export interface RegisterRequest {
	login_id: string;
	verification_code: string | null; // 手机号/邮箱型必填（所有权验证）；用户名型为 null
	nickname: string;
	password: string;
	client_type: string;
	device_id: string | null; // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
export interface LoginRequest {
	login_id: string;
	password: string;
	client_type: string;
	device_id: string | null; // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
export interface GuestRequest {
	nickname: string | null; // 可选；null 时服务端自动生成默认昵称
	client_type: string;
	device_id: string | null; // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
// spec: api_spec_public.md → auth/send_register_code（手机号/邮箱型 login_id 的所有权验证）
export interface SendRegisterCodeRequest {
	login_id: string;
	client_type: string;
}
export interface RequestPasswordResetRequest {
	login_id: string;
	client_type: string;
}
export interface ResetPasswordRequest {
	login_id: string;
	verification_code: string;
	new_password: string;
	client_type: string;
	device_id: string | null; // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}

// Users
// spec: api_spec_user.md → users/update_location（真实位置上报，非浏览锚点；location 必填）
export interface UpdateLocationRequest {
	location: Location;
}

export interface UpdateAccountRequest {
	nickname?: string;
	avatar_file_id?: string;
	bio?: string;
	tags?: string[];
	gender?: Gender;
}
export interface SetFollowRequest {
	user_id: string;
	active: boolean;
}
export interface SetBlockRequest {
	user_id: string;
	active: boolean;
}
export interface SearchUsersRequest {
	keyword?: string;
	tag?: string;
	radius?: number;
	sort_by?: string;
	reference_point?: GeoPoint;
	pagination?: PaginationParams;
}
export interface UpdateSettingsRequest {
	settings: Record<string, unknown>;
}

// spec: api_spec_user.md → users/upgrade（访客升级；仅用户名型 login_id，手机号/邮箱待验证码服务）
export interface UpgradeAccountRequest {
	login_id: string;
	password: string;
}

// spec: api_spec_user.md → users/fetch_blocked（我的拉黑列表；命名空间天生私有）
export interface FetchBlockedRequest {
	pagination: PaginationParams | null;
}
export interface BlockedUserItem {
	user: UserInfo;
	blocked_at: string;
	reason: string | null;
}

// Moments 流（spec: api_spec_moment.md → moments/fetch_following、fetch_nearby）
export interface FetchFollowingMomentsRequest {
	reference_point?: GeoPoint; // 用于距离/方位计算；不传时服务端按回退链解析
	pagination?: PaginationParams;
}
export interface FetchNearbyMomentsRequest {
	radius?: number; // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪
	reference_point?: GeoPoint;
	pagination?: PaginationParams;
}

// Moments
export interface CreateMomentRequest {
	description: string;
	location: Location; // spec: 必填（无定位时不允许发布，取默认位置兜底由调用方负责）
	privacy_level?: string;
	tags?: string[];
	image_file_ids?: string[];
	mentions?: string[];
}
export interface UpdateMomentRequest {
	moment_id: string;
	description?: string | null;
	location?: Location;
	privacy_level?: string;
	tags?: string[];
	image_file_ids?: string[];
	mentions?: string[];
}
export interface SearchMomentsRequest {
	keyword?: string;
	tag?: string;
	sort_by?: 'relevance' | 'newest' | 'popular' | null; // 至少提供 keyword/tag/sort_by='popular' 之一；'popular' 即热门模式
	pagination?: PaginationParams;
}
export interface FavoriteMomentRequest {
	moment_id: string;
	active: boolean;
}

// Messages
export type SendMessageType = Exclude<MessageType, 'system'>;
export interface SendPrivateMessageRequest {
	target_user_id: string;
	type: SendMessageType;
	content?: string | null;
	file_id?: string | null;
	share_code?: string | null;
	reply_to_message_id?: string | null;
}
export interface ListPrivateMessagesRequest {
	target_user_id: string;
	pagination?: PaginationParams;
}
export interface SendGroupMessageRequest {
	group_id: string;
	type: SendMessageType;
	content?: string | null;
	file_id?: string | null;
	share_code?: string | null;
	reply_to_message_id?: string | null;
	mentions?: string[];
}
export interface ListGroupMessagesRequest {
	group_id: string;
	pagination?: PaginationParams;
}
export interface MarkPrivateReadRequest {
	scope: 'session' | 'all';
	target_user_id?: string | null;
	up_to_message_id?: string | null;
}
export interface MarkGroupReadRequest {
	scope: 'group' | 'all';
	group_id?: string | null;
	up_to_message_id?: string | null;
}
export interface UpdatePrivateChatPreferencesRequest {
	target_user_id: string;
	is_muted?: boolean;
	is_pinned?: boolean;
}
export interface UpdateGroupPreferencesRequest {
	group_id: string;
	is_muted?: boolean;
	is_pinned?: boolean;
}
export interface GetGroupSettingsRequest {
	// spec: api_spec_group.md → groups/admin/settings
	group_id: string;
}
export interface ListAdminGroupMembersRequest {
	// spec: api_spec_group.md → groups/admin/members/list
	group_id: string;
	role?: GroupRole;
	pagination?: PaginationParams;
}
export interface RecallGroupMessageAdminRequest {
	// spec: api_spec_group.md → groups/admin/messages/recall（管理员撤回任意成员消息，无时间窗）
	group_id: string;
	message_id: string;
}

// Reactions / Comments / Reports / Notify
export interface SetReactionRequest {
	target_type: ReactTarget;
	target_id: string;
	emoji: string;
	active: boolean;
}
export interface CreateCommentRequest {
	moment_id: string;
	content: string;
	parent_id?: string;
	mentions?: string[];
}
export interface DeleteCommentRequest {
	comment_id: string;
}
export interface CreateReportRequest {
	target_type: ReportTargetType;
	target_id: string;
	reason: ReportReason;
	description?: string | null;
	// 消息举报快照由服务端受理时基于消息真值生成并冻结，客户端不提交快照内容
}
export interface MarkNotifyReadRequest {
	// spec: notify.md → mark_read（参数必显式：scope 必填；scope='ids' 时 notify_ids 1-50 个）
	scope: 'ids' | 'all';
	notify_ids?: string[];
}

// Media / System
export interface BootstrapRequest {
	client_type: 'web';
	device_id: string | null;
}

// --- 平台管理（spec: api_spec_admin.md，/admin/api/v1 节点）---
// 仅供平台管理员使用；平台管理员在 /api/v1 上与普通用户同权，平台权限不外溢。

export type UserAccountStatus = 'normal' | 'banned';

export interface AdminRoleBasic {
	role_id: string;
	name: string;
}

// rbac/me → 我的平台管理员权限；'*' 表示 superadmin（全部权限，PII 键仍需显式持有）
export interface AdminPermissions {
	permissions: string[];
	roles: AdminRoleBasic[];
}

export interface AdminPermissionItem {
	key: string;
	description: string;
}

export interface AdminPermissionDictionary {
	items: AdminPermissionItem[];
}

export interface AdminRoleItem {
	role_id: string;
	name: string;
	description: string | null;
	permissions: string[];
}

export interface AdminRoleList {
	items: AdminRoleItem[];
}

export interface AdminOperatorItem {
	user: UserInfo;
	granted_at: string; // 最近一次授权时间
	permissions: string[]; // '*' = superadmin
	roles: AdminRoleBasic[];
}

export type AuditAction =
	| 'user.ban'
	| 'user.unban'
	| 'group.dismiss'
	| 'group.member_mute'
	| 'group.member_ban'
	| 'group.member_remove'
	| 'announcement.create'
	| 'announcement.update'
	| 'announcement.delete'
	| 'role.create'
	| 'role.update'
	| 'role.delete'
	| 'role.set_permissions'
	| 'admin.award'
	| 'admin.revoke'
	| 'report.judge'
	| 'tag.set_ban'
	| 'config.update';

export type AuditTargetType = 'user' | 'group' | 'announcement' | 'role' | 'admin' | 'report' | 'tag' | 'config';

export type AuditDetails = Extract<
	{ type: 'user.ban'; reason: string | null; duration_days: number | null }
	| { type: 'user.unban'; reason: string | null }
	| { type: 'group.dismiss'; reason: string | null }
	| { type: 'group.member_mute'; reason: string | null; duration_minutes: number | null }
	| { type: 'group.member_ban'; reason: string | null }
	| { type: 'group.member_remove'; reason: string | null }
	| { type: 'announcement.create'; title: string }
	| { type: 'announcement.update'; title: string; changed_fields: string[] }
	| { type: 'announcement.delete'; title: string }
	| { type: 'role.create'; role_name: string; permissions: string[] }
	| { type: 'role.update'; role_name: string; added_permissions: string[]; removed_permissions: string[] }
	| { type: 'role.delete'; role_name: string }
	| { type: 'role.set_permissions'; role_name: string; permissions: string[] }
	| { type: 'admin.award'; target_user_id: string; role_ids: string[]; award_all_permissions: boolean | null }
	| { type: 'admin.revoke'; target_user_id: string }
	| { type: 'report.judge'; report_id: string; action: ReportAction; note: string | null }
	| { type: 'tag.set_ban'; name: string; active: boolean }
	| { type: 'config.update'; config_key: string; old_value: unknown; new_value: unknown },
	{ type: AuditAction }
>;

// 字段可见性：ip_location/device_summary 需 admin.audit.read；ip_address 需 admin.audit.read_ip（不足为 null）
export interface AuditLogItem {
	log_id: string;
	action: AuditAction;
	operator: UserInfo;
	target_type: AuditTargetType;
	target_id: string;
	target_name: string | null;
	details: AuditDetails | null;
	ip_location: string | null;
	ip_address: string | null;
	device_summary: string | null;
	created_at: string;
}

// 地理统计（POST /admin/api/v1/stats/geo，spec api_spec_admin.md「统计分析（stats）API」）
export interface StatsTimeRange {
	start_time: string | null;
	end_time: string | null;
}

export interface StatsEntityBreakdown {
	user: number;
	group: number;
	moment: number;
}

export interface AdminRegionRankItem {
	cell_token: string;
	level: number;
	center: { latitude: number; longitude: number };
	total_count: number;
	breakdown: StatsEntityBreakdown;
}

export interface AdminPlaceRankItem {
	location_name: string;
	total_count: number;
	breakdown: StatsEntityBreakdown;
}

export interface AdminGeoStatsTotals {
	user: number;
	group: number;
	moment: number;
	total: number;
}

export interface AdminGeoStatsTrendPoint {
	date: string;
	user: number;
	group: number;
	moment: number;
}

export interface AdminGeoStatsResponse {
	generated_at: string;
	totals: AdminGeoStatsTotals;
	regions: AdminRegionRankItem[];
	places: AdminPlaceRankItem[];
	trends: AdminGeoStatsTrendPoint[];
}

export interface AdminGeoDrilldownItem {
	object_id: string;
	title: string;
	location_name: string | null;
	latitude: number;
	longitude: number;
	created_at: string;
}

export interface AdminGeoDrilldownPage {
	items: AdminGeoDrilldownItem[];
	next_cursor: string | null;
}

// AppConfig：完整应用配置；features/limits/contact 构成 ClientConfig 投影
export interface UploadConfig {
	image_max_size: number;
	video_max_size: number;
	audio_max_size: number;
	file_max_size: number;
	allowed_image_types: string[];
	allowed_video_types: string[];
}

export interface ContentDefaults {
	default_location: Location;
	default_group_avatar_url: string;
	default_user_avatar_url: string;
	default_group_settings: GroupSettings;
}

export interface RateLimitConfig {
	global_per_hour: number;
	user_per_hour: number;
	auth_register_per_10minutes: number;
	auth_signin_per_10minutes: number;
	auth_guest_per_10minutes: number;
	auth_refresh_per_10minutes: number;
	auth_reset_per_10minutes: number;
	search_per_minute: number;
	write_per_minute: number;
	moments_create_per_minute: number;
	comments_create_per_minute: number;
	reports_create_per_hour: number;
	upload_presign_per_hour: number;
	ws_subscribe_per_second: number;
	admin_read_per_minute: number;
	admin_write_per_minute: number;
}

export interface ResponseDefaults {
	search_suggest_count: number;
	chat_recent_messages_count: number;
	group_announcements_count: number;
	system_announcements_count: number;
	comment_top_replies_count: number;
	user_profile_moments_count: number;
}

export interface AppConfig {
	features: FeatureSwitches;
	limits: AppLimits;
	contact: ContactInfo;
	upload: UploadConfig;
	defaults: ContentDefaults;
	rate_limits: RateLimitConfig;
	response_defaults: ResponseDefaults;
	updated_at: string | null;
	updated_by_user_id: string | null;
}

export interface UpdateAppConfigRequest {
	upload?: Partial<UploadConfig> | null;
	defaults?: {
		default_location?: Partial<Location> | null;
		default_group_avatar_url?: string | null;
		default_user_avatar_url?: string | null;
		default_group_settings?: Partial<GroupSettings> | null;
	} | null;
	rate_limits?: Partial<RateLimitConfig> | null;
	response_defaults?: Partial<ResponseDefaults> | null;
	features?: Partial<FeatureSwitches> | null;
	limits?: Partial<AppLimits> | null;
	contact?: Partial<ContactInfo> | null;
}

// AdminUserItem 继承 UserItem；查看者字段（relation/social_context）平台视角为 null
export interface AdminUserItem extends UserItem {
	status: UserAccountStatus;
	banned_until: string | null; // null = 未封禁/永久封禁
	privacy_level: PrivacyLevel;
	created_at: string;
	last_active_at: string | null;
	login_id: string;
	register_ip: string | null; // 需 admin.users.read_ip
	register_location: string | null;
	last_login_ip: string | null; // 需 admin.users.read_ip
	last_login_location: string | null;
	email: string | null;
	phone: string | null;
}

// AdminGroupItem 继承 GroupItem；viewer_state 平台视角为 null；owner 平台视角恒返回
export interface AdminGroupItem extends GroupItem {
	owner: UserInfo;
	created_at: string;
}

// ReportItem（spec: types.md；客户端此前未消费，随平台管理举报裁决引入）
export interface ReportSnapshot {
	chat_type: 'private' | 'group';
	target_chat_id: string;
	message_type: MessageType;
	snapshot_content: string | null;
	snapshot_media: MessageMedia | null;
	sent_at: string;
}

export interface ReportResolution {
	action: ReportAction;
	resolved_at: string;
	note: string | null;
}

export interface ReportItem {
	report_id: string;
	target_type: ReportTargetType;
	target_id: string;
	reason: ReportReason;
	description: string | null;
	status: ReportStatus;
	created_at: string;
	handled_at: string | null;
	message_snapshot: ReportSnapshot | null;
	resolution: ReportResolution | null;
}

export interface AdminReportItem extends ReportItem {
	reporter: UserInfo;
	handled_by: UserInfo | null;
	handle_note: string | null;
}

// AdminTagItem 扩展 TagItem；业务唯一键为 name（tag_id 仅展示/审计关联）
export interface AdminTagItem extends TagItem {
	tag_id: string;
	created_at: string;
}

export interface SystemAnnouncementItem {
	announcement_id: string;
	title: string;
	content: string;
	severity: AnnounceSeverity;
	action_url: string | null;
	created_at: string;
}

export interface AdminAnnouncementItem extends SystemAnnouncementItem {
	created_by: UserInfo;
	send_push: boolean;
	push_sent_at: string | null;
	view_count: number;
}

// ===== 分享码落地（spec api_spec_public.md share/open_the_door；仅 public 版，无登录镜像） =====

export type ShareType = 'user' | 'group' | 'moment' | 'location';

// spec: api_spec_types.md → SharePayload（与 share_type 判别匹配的四载荷联合）
export type SharePayload =
	| { shared_user_id: string }
	| { shared_group_id: string }
	| { shared_moment_id: string }
	| { shared_location: Location };

// spec: api_spec_share.md → share/create（幂等端点：必须携带 Idempotency-Key）
export interface CreateShareRequest {
	share_type: ShareType;
	payload: SharePayload;
	expires_in_seconds: number | null; // null = 默认由系统配置
}
export interface ShareCreated {
	share_id: string;
	share_code: string;
	expires_at: string | null;
}

export interface ShareBase {
	share_id: string;
	share_code: string;
	share_type: ShareType;
	creator: UserInfo;
	created_at: string;
	expires_at: string | null;
}

export interface ShareDetail {
	share: ShareBase;
	payload:
		| { shared_user_id: string }
		| { shared_group_id: string }
		| { shared_moment_id: string }
		| { shared_location: Location };
	// 目标资源不可访问（删除/隐私变更）时为 null；分享撤销/过期整体 10300
	preview: Record<string, unknown> | null;
}