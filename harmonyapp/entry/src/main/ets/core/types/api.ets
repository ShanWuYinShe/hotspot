// ==================== 基础类型 ====================

export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  data?: T;
}

// 通用分页容器（spec types.md：next_cursor 恒在，null = 没有更多数据）
export interface PaginatedData<T> {
  items: T[];
  next_cursor: string | null;
}

export interface PaginationParams {
  limit?: number;
  cursor?: string;
}

// ==================== 通用类型定义 (Domain Primitives) ====================

// 消息类型枚举
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share';

// 用户类型枚举
export type UserType = 'temp' | 'normal';

// 隐私级别
export type PrivacyLevel = 'public' | 'private';

// 角色
export type GroupRole = 'owner' | 'admin' | 'member';
export type SettableGroupRole = 'admin' | 'member';

// 客户端类型
export type ClientType = 'web' | 'app';

// 设备标识（spec: DeviceId = UUID v4），持久化、驱动单设备单会话踢人语义
export type DeviceId = string;

// 用户在线状态
export type UserPresenceStatus = 'online' | 'offline';

// 举报状态
export type ReportStatus = 'pending' | 'resolved' | 'dismissed';

// 举报目标类型
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';

// 举报处理操作
export type ReportAction = 'resolve' | 'dismiss';

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

// 群组准入方式（叠加 requires_approval 决定是否需审核）
export type GroupJoinMethod = 'direct' | 'password' | 'invite_only';

// 公告严重程度 (spec: AnnounceSeverity)
export type AnnounceSeverity = 'info' | 'warning' | 'error';

// spec Reaction: reactions 聚合 (MomentItem/CommentItem/MessageItem 共用)
export interface Reaction {
  emoji: string;
  count: number;
  reacted_by_me: boolean;
}

// spec ReactionItem: reactions/fetch 返回的反应者列表项
export interface ReactionItem {
  user: UserInfo;
  emoji: string;
  reacted_at: string;
}

// 关注来源
export type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// 推荐理由类型
export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

// 账号状态（系统管理员视角）
export type UserAccountStatus = 'normal' | 'banned';

// ==================== 位置与地理信息 ====================

// 基础坐标点
export interface GeoPoint {
  latitude: number;
  longitude: number;
}

// 命名位置 (用于 Request)
export interface Location {
  point: GeoPoint;
  location_name: string;
}

// 相对位置 (用于 Response)
export interface RelativeLocation {
  point: GeoPoint;
  location_name: string;
  distance: number | null;
  bearing_degrees: number | null;
}

// 视口边界
export interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}

export interface MapViewport {
  bounds: Bounds;
}

// ==================== 媒体资源 ====================

export interface MomentImage {
  url: string;
  thumbnail_url: string | null;
  width: number | null;
  height: number | null;
  mime_type: string | null;
}

export interface MessageMedia {
  url: string;
  thumbnail_url: string | null;
  file_name: string | null;
  file_size: number | null;
  duration: number | null;
  width: number | null;
  height: number | null;
  mime_type: string | null;
}

// ==================== 领域实体定义 (Domain Entities) ====================

// 1. 用户 (User)

export interface UserInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
}

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface Relationship {
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
}

export interface SocialContext {
  mutual_friends_count: number;
  mutual_friends_preview?: UserInfo[];
  common_groups_count?: number;
  common_tags?: string[];
}

export interface UserItem {
  user: UserInfo;
  bio: string | null;
  tags: string[];

  // 隐私受控字段：恒定 nullable，null = 隐藏/不适用
  presence_status: UserPresenceStatus | null;
  last_online_at: string | null;
  location: RelativeLocation | null;
  stats: UserStats | null;
  relation: Relationship | null;
  social_context: SocialContext | null;
}

// 2. 群组 (Group)

export interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface GroupViewerState {
  role: GroupRole;
  unread_count: number;
  is_muted: boolean;
  muted_until: string | null;
  is_pinned: boolean;
  joined_at: string;
}

// 消息摘要快照（spec types.md → MessageBrief，reply_to 与 last_message 共用）
export interface MessageBrief {
  message_id: string;
  type: MessageType;
  content: string | null;
  share_code: string | null; // 最后一条消息为 share 类型时包含
  sender: UserInfo;
  sent_at: string;
}

export interface GroupItem {
  group: GroupInfo;
  description: string | null;
  tags: string[];
  join_method: GroupJoinMethod;
  requires_approval: boolean;
  location: RelativeLocation | null;
  last_activity_at: string | null;
  online_members_count: number | null;
  next_activity_at: string | null;
  created_at: string | null;
  owner_id: string | null;
  viewer_state: GroupViewerState | null;
  friends_in_group: number | null;
  // 仅 groups/fetch_joined 返回最近消息预览；其他场景为 null
  last_message: MessageBrief | null;
}

export interface GroupSettings {
  join_method: GroupJoinMethod;   // 默认 'direct'
  requires_approval: boolean;     // 默认 false
  has_password: boolean;          // 群是否设有密码
  is_searchable: boolean;
  members_searchable: boolean;    // 默认 true（spec types.md → GroupSettings）
  max_members: number;
  location: Location;
}

export interface GroupAnnouncementItem {
  announcement_id: string;
  group_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  created_at: string;
  is_pinned: boolean;
}

export interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url: string | null;
  created_at: string;
}

// 3. 生活瞬间 (Moment)

export interface MomentSummary {
  moment_id: string;
  cover_url: string | null;     // 首张图片缩略图 URL；null = 无图片（spec types.md → MomentSummary）
  description_preview: string;
}

export interface MomentItem {
  moment_id: string;
  description: string;
  images: MomentImage[];
  tags: string[];
  mentions: string[];
  created_at: string;
  comments_count: number;
  shares_count: number;
  reactions: Reaction[];
  user: UserInfo;
  is_deleted: boolean;
  location: RelativeLocation | null;
  is_favorited: boolean;
  recommendation: RecommendationType | null;
}

// 4. 消息 (Message)

export interface MessageSentResult {
  message_id: string;
  sent_at: string;
}

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;
  sender: UserInfo;
  reply_to: MessageBrief | null;
  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;
  is_read: boolean;
  reactions: Reaction[];
}

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;
  sender: UserInfo;
  reply_to: MessageBrief | null;
  mentions: string[];
  recalled_by: UserInfo | null;
  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;
  read_count: number | null;
  reactions: Reaction[];
}

// 5. 评论 (Comment)

export interface CommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id: string | null;
  mentions: string[];
  created_at: string;
  replies_count: number;
  is_deleted: boolean;
  reactions: Reaction[];
  // 顶层评论列表场景(comments/fetch)返回前 N 条回复预览；其他场景为空数组
  top_replies: CommentItem[];
}

// spec user.md → open_the_door：recent_moments 仅 include_recent_moments=true 时返回
export interface UserProfileBundle {
  user: UserItem;
  recent_moments: PaginatedData<MomentItem> | null;
}

// ReactionItem 已在通用类型区域定义; LikeItem 为历史遗留,删除以消除混淆。

// 6. 私聊 (Private Chat)



export interface PrivateChatItem {
  other_user: UserInfo;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  last_message: MessageBrief | null;
  presence_status: UserPresenceStatus | null;
  created_at: string | null;
  can_message: boolean;
  forbidden_reason: ForbiddenReason | null;  // 仅 can_message=false 时非 null（spec chat.md → PrivateChatItem）
}

export interface PrivateChatOpenResult {
  chat: PrivateChatItem;
  recent_messages: PaginatedData<PrivateMessageItem>;
}

// ==================== 认证API类型 ====================

export interface AuthTokenPairApp {
  client_type: 'app'; // 判别字段（spec auth.md → AuthTokenPair）
  user_id: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

// HarmonyOS 使用 App 类型
export interface AuthData extends AuthTokenPairApp {}

export interface RegisterRequest {
  login_id: string;
  // 手机号/邮箱型 login_id 必填（所有权验证，经 auth/send_register_code 发送）；用户名型不传
  verification_code?: string | null;
  nickname: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId;
}

export interface SendRegisterCodeRequest {
  login_id: string;
  client_type: ClientType;
}

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId;
}

export interface GuestRequest {
  nickname: string | null; // 可选；null 时服务端自动生成默认昵称（spec auth.md → guest）
  client_type: ClientType;
  device_id: DeviceId;
}

export interface RefreshTokenRequest {
  client_type: ClientType;
  device_id: DeviceId;
  refresh_token?: string;
}

export interface SessionStatus {
  user_id: string;
  expires_at: string;
}

export interface SessionItem {
  session_id: string;
  device_id: DeviceId;
  device_type: string | null;
  device_name: string | null;
  ip_address: string | null;
  ip_location: string | null;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

export interface SessionsRequest {}

export interface RevokeSessionRequest {
  session_id: string;
}

export interface RevokeAllResult {
  revoked_count: number;
}

export interface RequestPasswordResetRequest {
  login_id: string;
  client_type: ClientType;
}

export interface ResetPasswordRequest {
  login_id: string;
  verification_code: string;
  new_password: string;
  client_type: ClientType;
  device_id: DeviceId; // App 必传（spec auth.md → reset_password）
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ==================== 用户API类型 ====================

export type Gender = 'male' | 'female' | 'other';
export type ChatPerm = 'everyone' | 'friends' | 'nobody';
// 10200 失败时的权限拒绝原因（spec api_spec_errors.md → ForbiddenReason 权威定义，12 项）
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
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected' | 'revoked';

export interface SearchUsersRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest';
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface OpenUserDoorRequest {
  user_id: string;
  include_recent_moments?: boolean; // true 时返回主页首屏聚合（spec user.md → open_the_door）
}

// UserFriendsRequest / UserFollowingRequest / UserFollowersRequest 已合并入
// FetchFriendsRequest / FetchFollowingRequest / FetchFollowersRequest（user_id 可选）

export interface FetchUserGroupsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface FetchUserMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface FetchUserReactedMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

// My Profile

export interface AccountInfo {
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

export interface UpdateAccountRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
  gender?: Gender;
}

export interface UpdateLocationRequest {
  location: Location;
}

export interface UpgradeAccountRequest {
  login_id: string;
  password: string;
}

export interface UserSettings {
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

export interface UpdateSettingsRequest {
  settings: Partial<UserSettings>;
}

export interface DeleteAccountRequest {
  confirmation: string;
}

export interface FollowRequest {
  user_id: string;
  active?: boolean;
  source?: FollowSource;
  source_moment_id?: string;
}

export interface BlockRequest {
  user_id: string;
  active?: boolean;
  reason?: string;
}

export interface FetchFriendsRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchFollowingRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchFollowersRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchBlockedRequest {
  pagination?: PaginationParams;
}

export interface SetRemarkRequest {
  user_id: string;
  remark: string;
}

// 用户关系列表项（list_friends / list_following / list_followers 共用，spec user.md → RelatedUserItem）
// user 为 UserItem（presence 隐私受控字段随 user 返回）；followed_back_at 仅 list_friends
export interface RelatedUserItem {
  user: UserItem;
  followed_at: string;
  followed_back_at: string | null; // 对方回关时间（成为好友的时间）；仅 list_friends 返回
  remark: string | null; // @owner-only：仅查询自己时返回
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason: string | null;
}

// ==================== 生活瞬间API类型 ====================

// 全球 keyword/tag 搜索（spec moment.md → moments/search；附近发现走 moments/fetch_nearby，feed 域已废除）
export interface SearchMomentsRequest {
  keyword?: string;
  tag?: string;
  sort_by?: 'relevance' | 'newest' | 'popular';
  pagination?: PaginationParams;
}

export interface OpenMomentDoorRequest {
  moment_id: string;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}

export interface EditMomentRequest {
  moment_id: string;
  description?: string;
  location?: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}

export interface DeleteMomentRequest {
  moment_id: string;
}

export interface FavoriteMomentRequest {
  moment_id: string;
  active?: boolean;
}

export interface FetchCreatedMomentsRequest {
  pagination?: PaginationParams;
}

// 关注动态：我关注的人发布的瞬间（spec moment.md → moments/fetch_following，原 feed/following 迁移）
export interface FetchFollowingMomentsRequest {
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

// 附近瞬间发现流（spec moment.md → moments/fetch_nearby，原 feed/nearby_moments 迁移）
export interface FetchNearbyMomentsRequest {
  radius?: number;
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface FetchFavoritesRequest {
  pagination?: PaginationParams;
}

// ==================== 评论API类型 ====================

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: string[];
}

export interface DeleteCommentRequest {
  comment_id: string;
}

export interface FetchCommentsRequest {
  moment_id: string;
  pagination?: PaginationParams;
}

export interface FetchCommentRepliesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}

// ==================== Reactions API类型 ====================

export type ReactTarget = 'moment' | 'comment' | 'message';

export interface SetReactionRequest {
  target_type: ReactTarget;
  target_id: string;
  emoji: string;
  active: boolean;
}

export interface FetchReactionsRequest {
  target_type: string;
  target_id: string;
  emoji?: string;
  pagination?: PaginationParams;
}

// ==================== 私聊管理API类型 ====================

export interface FetchPrivateChatsRequest {
  pagination?: PaginationParams;
}

export interface OpenPrivateChatDoorRequest {
  target_user_id: string;
}

export interface UpdatePrivateChatPreferencesRequest {
  target_user_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeletePrivateChatRequest {
  target_user_id: string;
}

// ==================== 私聊消息API类型 ====================

export interface SendPrivateMessageRequest {
  target_user_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
}

export interface FetchPrivateMessagesRequest {
  target_user_id: string;
  pagination?: PaginationParams;
}

export interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallPrivateMessageRequest {
  message_id: string;
}

export interface MarkPrivateReadRequest {
  scope: string;
  target_user_id?: string;
  up_to_message_id?: string;
}

// ==================== 群组管理API类型 ====================

export interface SearchGroupsRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface OpenGroupDoorRequest {
  group_id: string;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  join_method?: GroupJoinMethod;
  requires_approval?: boolean;
  is_searchable?: boolean;
  members_searchable?: boolean; // 默认 true
  max_members?: number;
  location: Location;
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
  next_activity_at?: string | null;
  is_searchable?: boolean;
  members_searchable?: boolean;
  max_members?: number;
  location?: Location;
}

export interface DismissGroupRequest {
  group_id: string;
}

export interface UpdateGroupPreferencesRequest {
  group_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

// ==================== 群组成员管理API类型 ====================

export interface FetchGroupMembersRequest {
  group_id: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}

export interface SearchGroupMembersRequest {
  group_id: string;
  keyword: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until?: string;
  presence_status?: UserPresenceStatus;
  last_online_at?: string;
}

export interface CreateGroupInviteRequest {
  group_id: string;
  user_id: string;
  message?: string;
}

// invite_id/expires_at 可选（spec group.md → Invitation：no-op 已是成员/TTL=0 时省略）
export interface Invitation {
  invite_id: string | null;
  expires_at: string | null;
}

export interface EnterGroupRequest {
  group_id: string;
  password?: string;
  message?: string;
}

export interface JoinGroupResult {
  status: 'joined' | 'pending_approval';
  application?: Application;
}

export interface Application {
  application_id: string;
}

export interface SetMemberRoleRequest {
  group_id: string;
  user_id: string;
  role: 'admin' | 'member';
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
  active?: boolean;
  reason?: string;
}

export interface FetchBannedMembersRequest {
  group_id: string;
  pagination?: PaginationParams;
}

export interface AdminBannedMemberItem {
  user: UserInfo;
  reason: string | null;
  banned_at: string;
  banned_by: UserInfo | null;
}

export interface RemoveMemberRequest {
  group_id: string;
  user_id: string;
}

export interface FetchGroupApplicationsRequest {
  group_id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface GroupApplicationItem {
  application_id: string;
  applicant: UserInfo;
  group: GroupInfo;
  expires_at: string | null;
  message: string | null;
  status: ApplicationStatus;
  created_at: string;
  handled_at: string | null;
  handled_by: UserInfo | null;
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

export interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}

export interface FetchJoinedGroupsRequest {
  pagination?: PaginationParams;
}

export interface FetchSubmittedApplicationsRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface CancelApplicationRequest {
  group_id: string;
  application_id: string;
}

export interface ListSentInvitesRequest {
  group_id: string;
  status?: ApplicationStatus | 'all';
  pagination?: PaginationParams;
}

export interface GroupSentInviteItem {
  invite_id: string;
  inviter: UserInfo;
  invitee: UserInfo;
  status: ApplicationStatus;
  message: string | null;
  created_at: string;
  expires_at: string | null;
  responded_at: string | null;
}

export interface RevokeSentInviteRequest {
  group_id: string;
  invite_id: string;
  reason?: string;
}

export interface FetchReceivedInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'revoked' | 'all';
  pagination?: PaginationParams;
}

export interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: ApplicationStatus;
  created_at: string;
  expires_at?: string;
}

export interface AcceptInviteRequest {
  invite_id: string;
}

/** acceptGroupInvite 响应：刚加入的群组 id（spec group:912-917） */
export interface InviteAcceptedResult {
  group_id: string;
}

export interface RejectInviteRequest {
  invite_id: string;
  reason?: string;
}

export interface ExitGroupRequest {
  group_id: string;
}

// ==================== 群公告API类型 ====================

export interface CreateGroupAnnouncementRequest {
  group_id: string;
  title: string;
  content: string;
  is_pinned?: boolean;
}

export interface FetchGroupAnnouncementsRequest {
  group_id: string;
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

// ==================== 群聊消息API类型 ====================

export interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
  mentions?: string[];
}

export interface FetchGroupMessagesRequest {
  group_id: string;
  pagination?: PaginationParams;
}

export interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallGroupMessageRequest {
  message_id: string;
}

export interface MarkGroupReadRequest {
  scope: string;
  group_id?: string;
  up_to_message_id?: string;
}

// ==================== 地图 API ====================

export type MapDataType = 'user' | 'group' | 'moment' | 'all';

export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

export interface MapHeatBreakdown {
  user: number | null;
  group: number | null;
  moment: number | null;
}

export interface MapHeatBucket {
  cell_token: string;
  level: number;
  location: GeoPoint;
  total_count: number;
  breakdown: MapHeatBreakdown;
}

export interface MapHeatCanvas {
  items: MapHeatBucket[];
}

export type HotspotObjectType = 'user' | 'group' | 'moment';

export interface MapHotspotItem {
  object_type: HotspotObjectType;
  object_id: string;
  title: string;
  subtitle: string | null;
  cover_url: string | null;
  location: RelativeLocation;
  created_at: string | null;
}

export interface MapHeatFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapHotspotFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapHeatRequest {
	viewport: MapViewport;
	filter?: MapHeatFilter;
}

export interface MapHotspotsRequest {
	viewport: MapViewport;
	filter?: MapHotspotFilter;
	pagination?: PaginationParams;
}

// spec public.md → map/canvas hotspots 分支 = PaginatedData<MapHotspotItem>（next_cursor 恒在）
export interface MapHotspotsCanvas {
  items: MapHotspotItem[];
  next_cursor: string | null;
}

export interface MapCanvasFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapCanvasRequest {
  viewport: MapViewport;
  filter?: MapCanvasFilter;
  pagination?: PaginationParams;
}

export type MapCanvasMode = 'heatmap' | 'hotspots' | 'empty';

export interface MapCanvasResponse {
  mode: MapCanvasMode;
  heatmap?: MapHeatCanvas;
  hotspots?: MapHotspotsCanvas;
}

// ==================== 举报API类型 ====================

export interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description?: string;
}

export interface ReportCreated {
  report_id: string;
  is_duplicate: boolean;
}

export interface FetchReportsRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}

// 会话类型（spec types.md → ReportSnapshot.chat_type）
export type ReportChatType = 'private' | 'group';

// 消息举报快照（spec types.md：由服务端在举报受理时基于消息真值生成并冻结，客户端不提交）
export interface ReportSnapshot {
  chat_type: ReportChatType;           // 会话类型
  target_chat_id: string;              // 会话目标 ID：private 会话 = 对方 user_id，group 会话 = group_id
  message_type: MessageType;
  snapshot_content: string | null;     // 文本内容（text 类型）；非文本类型为 null
  snapshot_media: MessageMedia | null; // 媒体引用（图片/视频等）；文本类型为 null
  sent_at: string;
}

// 处理结果摘要（脱敏，不含处理者身份；spec types.md → ReportResolution）
export interface ReportResolution {
  action: ReportAction;
  resolved_at: string;
  note: string | null;   // 可选的处理说明（脱敏后）
}

// 举报项（report + admin 两模块共用：spec types.md → ReportItem）
export interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description: string | null;
  status: ReportStatus;
  created_at: string;
  handled_at: string | null;
  // 消息举报快照（仅 target_type='message'）
  message_snapshot: ReportSnapshot | null;

  // 处理结果摘要（恒定 nullable：null = 尚未处理）
  resolution: ReportResolution | null;
}

// 标签项（搜索、发现等模块共用）
export interface TagItem {
  name: string;
  usage_count: number;
}

// ==================== 搜索API类型 ====================

export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

export interface SearchSuggestRequest {
  prefix: string;
  reference_point?: GeoPoint;
  types?: SearchSuggestType[];
  radius?: number;
}

// 瞬间建议项（spec types.md → MomentSuggestItem = MomentSummary + 作者与位置）
export interface MomentSuggestItem {
  moment_id: string;
  description_preview: string;
  cover_url: string | null;     // 首张图片缩略图 URL；null = 无图片
  author: UserInfo;
  location: RelativeLocation | null;
}

// spec discovery.md → SearchSuggestItem：tag 分支复用 TagItem，moment 分支为 MomentSuggestItem
export interface SearchSuggestItemTag {
  type: 'tag';
  payload: TagItem;
}

// 登录视角建议项（spec discovery.md → SearchSuggestItem）：
// user/group 分支为完整 UserItem/GroupItem（含查看者状态），不复用精简投影
export interface SearchSuggestItemUser {
  type: 'user';
  payload: UserItem;
}

export interface SearchSuggestItemGroup {
  type: 'group';
  payload: GroupItem;
}

export interface SearchSuggestItemMoment {
  type: 'moment';
  payload: MomentSuggestItem;
}

export type SearchSuggestItem =
  | SearchSuggestItemTag
  | SearchSuggestItemUser
  | SearchSuggestItemGroup
  | SearchSuggestItemMoment;

export interface SearchSuggestResponse {
  items: SearchSuggestItem[];
}

// ==================== 系统配置API类型 ====================

export interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_register: boolean;
  enable_moments: boolean;
  enable_nearby_people: boolean;
}

export interface AppLimits {
  max_group_members: number;
  max_moment_tags: number;
  max_moment_images: number;
  moment_description_max_length: number;
  comment_max_length: number;
  search_radius_max: number;
  max_chat_files: number; // 单条聊天消息可附带的最大文件数（spec types.md → AppLimits）
  message_recall_window_seconds: number;
  stranger_message_daily_quota: number;
  reaction_emojis: string[];
  group_invite_ttl_hours: number;
  group_application_ttl_hours: number;
}

export interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website_url: string;
}

export interface ClientDefaults {
  default_location: Location;
  default_user_avatar_url: string;
  default_group_avatar_url: string;
}

export interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
  defaults: ClientDefaults;
}

export type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';

export interface HealthStatus {
  status: SystemHealthStatus;
  timestamp: string;
}

export type ApiVersionStatus = 'active' | 'deprecated' | 'sunset';

export interface ApiVersionMeta {
  version: string;
  status: ApiVersionStatus;
  deprecated_at: string | null;
  sunset_at: string | null;
  message: string | null;
  recommended_version: string | null;
}

export interface AnnouncementListRequest {}

// ==================== 文件与媒体上传API类型 ====================

export type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

export interface PresignedUpload {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface PresignFileInput {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: MediaCategory;
  md5?: string;
}

export interface PresignUploadRequest {
  files: PresignFileInput[];
}

export interface PresignedUploads {
  uploads: PresignedUpload[];
}

export interface CompleteUploadRequest {
  upload_ids: string[];
}

export interface CompletedUploads {
  uploads: UploadedMedia[];
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

// ==================== 分享API类型 ====================

export type ShareType = 'user' | 'group' | 'moment' | 'location';

export interface UserSharePayload {
  shared_user_id: string;
}

export interface GroupSharePayload {
  shared_group_id: string;
}

export interface MomentSharePayload {
  shared_moment_id: string;
}

export interface LocationSharePayload {
  shared_location: Location;
}

export type SharePayload =
  | UserSharePayload
  | GroupSharePayload
  | MomentSharePayload
  | LocationSharePayload;

export interface ShareBase {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  creator: UserInfo;
  created_at: string;
  expires_at: string | null;
}

export interface ShareCreated {
  share_id: string;
  share_code: string;
  expires_at: string | null;
}

export interface SharePreview {
  title: string;
  subtitle: string | null;
  cover_url: string | null;
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
  expires_at: string | null;
  is_revoked: boolean;
  preview: SharePreview | null;
}

export interface CreateShareRequest {
  share_type: ShareType;
  payload: SharePayload;
  expires_in_seconds?: number;
}

export interface GetShareRequest {
  share_code: string;
}

export interface FetchCreatedSharesRequest {
  share_type?: ShareType;
  pagination?: PaginationParams;
}

export interface RevokeShareRequest {
  share_id?: string;
  share_code?: string;
}

// ==================== Admin Types ====================

export interface AdminUserItem {
  user: UserInfo;
  status: UserAccountStatus;
  created_at: string;
  last_active_at: string | null;
  login_id: string;
  presence_status: UserPresenceStatus | null;
}

export interface AdminUserDetail {
  user: UserInfo;
  status: UserAccountStatus;
  created_at: string;
  last_active_at: string | null;
  login_id: string;
  email: string | null;
  phone: string | null;
  bio: string | null;
  tags: string[];
  location: RelativeLocation | null;
  presence_status: UserPresenceStatus | null;
  stats: UserStats;
  ban_history?: AdminBanRecord[];
}

export interface AdminBanRecord {
  banned_at: string;
  banned_by: UserInfo;
  reason?: string;
  duration_days?: number;
  unbanned_at?: string;
  unbanned_by?: UserInfo;
}

export interface AdminGroupItem {
  group: GroupInfo;
  owner: UserInfo;
  created_at: string;
}

export interface AdminGroupDetailSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
  has_password: boolean;
}

export interface AdminGroupDetailStats {
  messages_count: number;
  active_members_today: number;
}

export interface AdminGroupDetail {
  group: GroupInfo;
  description: string | null;
  tags: string[];
  max_members: number;
  owner: UserInfo;
  created_at: string;
  location: RelativeLocation | null;
  settings: AdminGroupDetailSettings;
  admins?: UserInfo[];
  stats: AdminGroupDetailStats;
  dismissal_history?: AdminDismissalRecord[];
}

export interface AdminDismissalRecord {
  dismissed_at: string;
  dismissed_by: UserInfo;
  reason?: string;
}

// Admin RBAC Types
export interface AdminPermissions {
  permissions: string[];
  roles: AdminRoleBasic[];
}

export interface AdminRoleBasic {
  role_id: string;
  name: string;
}

export interface AdminPermissionItem {
  key: string;
  description: string;
}

export interface AdminRoleItem {
  role_id: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface AdminAdminItem {
  user: UserInfo;
  permissions: string[];
  roles: AdminRoleBasic[];
}

export interface AdminRbacRoleCreateRequest {
  name: string;
  description?: string;
}

export interface AdminRbacRoleUpdateRequest {
  role_id: string;
  name?: string;
  description?: string;
  clear_description?: boolean;
}

export interface AdminRbacRoleSetPermissionsRequest {
  role_id: string;
  permission_keys: string[];
}

export interface AdminRbacAwardAdminRequest {
  user_id: string;
  award_all_permissions?: boolean;
  role_ids: string[];
}

export interface AdminRbacRevokeAdminRequest {
  user_id: string;
}

// Admin Audit Types
export type AuditAction =
  | 'user.ban'
  | 'user.unban'
  | 'group.dismiss'
  | 'announcement.create'
  | 'announcement.delete'
  | 'role.create'
  | 'role.update'
  | 'role.delete'
  | 'permission.grant'
  | 'permission.revoke'
  | 'config.update';

export type AuditTargetType = 'user' | 'group' | 'announcement' | 'role' | 'config';

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

export interface AuditDetailsUserBan {
  type: 'user.ban';
  reason?: string;
  duration_days?: number;
}

export interface AuditDetailsUserUnban {
  type: 'user.unban';
  reason?: string;
}

export interface AuditDetailsGroupDismiss {
  type: 'group.dismiss';
  reason?: string;
}

export interface AuditDetailsAnnouncementCreate {
  type: 'announcement.create';
  title: string;
}

export interface AuditDetailsAnnouncementDelete {
  type: 'announcement.delete';
  title: string;
}

export interface AuditDetailsRoleCreate {
  type: 'role.create';
  role_name: string;
  permissions: string[];
}

export interface AuditDetailsRoleUpdate {
  type: 'role.update';
  role_name: string;
  added_permissions?: string[];
  removed_permissions?: string[];
}

export interface AuditDetailsRoleDelete {
  type: 'role.delete';
  role_name: string;
}

export interface AuditDetailsPermissionGrant {
  type: 'permission.grant';
  permission: string;
}

export interface AuditDetailsPermissionRevoke {
  type: 'permission.revoke';
  permission: string;
}

export interface AuditDetailsConfigUpdate {
  type: 'config.update';
  config_key: string;
}

export type AuditDetails =
  | AuditDetailsUserBan
  | AuditDetailsUserUnban
  | AuditDetailsGroupDismiss
  | AuditDetailsAnnouncementCreate
  | AuditDetailsAnnouncementDelete
  | AuditDetailsRoleCreate
  | AuditDetailsRoleUpdate
  | AuditDetailsRoleDelete
  | AuditDetailsPermissionGrant
  | AuditDetailsPermissionRevoke
  | AuditDetailsConfigUpdate;

export interface AuditLogDetail extends AuditLogItem {
  before_state?: StateSnapshot;
  after_state?: StateSnapshot;
}

export interface StateSnapshotUser {
  type: 'user';
  nickname?: string;
  status?: string;
  roles?: string[];
}

export interface StateSnapshotGroup {
  type: 'group';
  name?: string;
  members_count?: number;
}

export interface StateSnapshotRole {
  type: 'role';
  name?: string;
  permissions?: string[];
}

export interface StateSnapshotConfig {
  type: 'config';
  value?: string;
}

export type StateSnapshot =
  | StateSnapshotUser
  | StateSnapshotGroup
  | StateSnapshotRole
  | StateSnapshotConfig;

export interface AuditLogListRequest {
  action?: AuditAction;
  operator_id?: string;
  target_type?: AuditTargetType;
  target_id?: string;
  start_time?: string;
  end_time?: string;
  pagination?: PaginationParams;
}

export interface AuditLogDetailRequest {
  log_id: string;
}

export interface AuditLogExportRequest {
  action?: AuditAction;
  operator_id?: string;
  target_type?: AuditTargetType;
  start_time?: string;
  end_time?: string;
  format?: 'csv' | 'json';
}

export interface AuditLogExportResult {
  export_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  download_url?: string;
  expires_at?: string;
}

export interface AuditLogExportStatusRequest {
  export_id: string;
}

// Admin Announcement Types
export interface CreateAnnouncementRequest {
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url?: string;
  send_push?: boolean;
}

export interface AnnouncementCreated {
  announcement_id: string;
}

// Admin Report Types
export interface AdminReportSearchRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}

export interface JudgeAdminReportRequest {
  report_id: string;
  action: ReportAction;
  note?: string;
  ban_duration_days?: number;
}

// Admin User Types
export interface AdminUserSearchRequest {
  user_id?: string;
  nickname?: string;
  login_id?: string;
  status?: UserAccountStatus;
  user_type?: UserType;
  pagination?: PaginationParams;
}

export interface AdminUserDetailRequest {
  user_id: string;
}

export interface AdminBanUserRequest {
  user_id: string;
  reason?: string;
  duration_days?: number;
}

export interface AdminUnbanUserRequest {
  user_id: string;
  reason?: string;
}

export interface AdminGroupSearchRequest {
  group_id?: string;
  name?: string;
  owner_id?: string;
  pagination?: PaginationParams;
}

export interface AdminGroupDetailRequest {
  group_id: string;
}

export interface AdminDismissGroupRequest {
  group_id: string;
  reason?: string;
}

// ==================== Response 包装类型 (ArkTS 要求命名类型) ====================

export interface SystemAnnouncementListData {
  items: SystemAnnouncementItem[];
}

export interface SessionListData {
  items: SessionItem[];
}

export interface GroupAnnouncementListData {
  items: GroupAnnouncementItem[];
}

// ==================== 通知中心 API 类型 ====================

export type NotifyType =
  'new_follower' | 'mutual_follow' | 'moment_commented'
  | 'comment_reply' | 'mentioned' | 'group_mentioned'
  | 'reaction_added'
  | 'group_invite_received'
  | 'group_application_received' | 'group_application_accepted' | 'group_application_rejected'
  | 'report_resolved' | 'report_dismissed'
  | 'system_announcement'
  | 'nearby_new_moment';

// Spec 定义 NotifyPayload 为 discriminated union（按 type 区分），
// 但 ArkTS 当前不支持基于字面量类型的类型窄化（type narrowing），
// 无法在 if/switch 中自动推导 payload 的具体字段。
// 因此使用扁平接口替代，运行时由业务层按 type 字段手动判断。
// 若未来 ArkTS 支持 discriminated union 窄化，应迁移至联合类型以获得编译期安全。
export interface NotifyPayload {
  type: NotifyType;
  actor?: UserInfo;
  applicant?: UserInfo;
  user?: UserInfo;
  source?: FollowSource;
  source_moment_id?: string;
  moment_id?: string;
  moment_summary?: MomentSummary;
  comment_id?: string;
  comment_preview?: string;
  replied_comment_preview?: string;
  reply_id?: string;
  reply_content_preview?: string;
  target_type?: 'message' | 'moment' | 'comment';
  emoji?: string;
  group_id?: string;
  group_name?: string;
  group_avatar_url?: string;
  message_id?: string;
  content_preview?: string;
  invite_id?: string;
  message?: string;
  application_id?: string;
  announcement_id?: string;
  title?: string; // system_announcement：公告标题（spec notify.md）
  severity?: AnnounceSeverity;
  revoked_by?: UserInfo;
  rejected_by?: UserInfo;
  reason?: string;
  report_id?: string;
  target_id?: string;
  resolved_at?: string;
  dismissed_at?: string;
  note?: string;
  distance?: number;
}

export interface NotifyItem {
  notify_id: string;
  type: NotifyType;
  is_read: boolean;
  created_at: string;
  payload: NotifyPayload;
}

export interface UnreadCountByType {
  social: number;
  interaction: number;
  group: number;
  report: number;
  nearby: number;
  system: number;
}

export interface UnreadSummaryResult {
  total: number;
  by_type: UnreadCountByType;
}

export interface FetchNotifyRequest {
  type?: NotifyType;
  is_read?: boolean;
  pagination?: PaginationParams;
}

// spec api_spec_notify.md → notify/mark_read：scope 必显式（'ids' 需 notify_ids 1-50；'all' 全部已读）。
export interface NotifyMarkReadRequest {
  scope: 'ids' | 'all';
  notify_ids?: string[];
}
