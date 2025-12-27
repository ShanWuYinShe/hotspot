/**
 * 统一API类型定义 - 严格按照 .cursor/rules/api_spec.mdc 规范
 * 
 * 重要说明：
 * - 所有API响应都使用统一的 ApiResponse<T> 格式
 * - 严格遵循 XxxRequest/XxxResponse 命名规范
 * - ApiResponse<T> 中的 T 就是 XxxResponse 类型
 * - 对于分页查询，T 为 PaginatedData<XxxResponse>
 */

// ============================================================================
// 基础响应类型
// ============================================================================

export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  // 服务端回显 `X-Request-Id`；每次请求必返，用于链路追踪
  request_id: string;
  data?: T;
}

// ============================================================================
// 系统 API 类型
// ============================================================================

export interface PaginationMeta {
  has_more: boolean;
  next_cursor?: string;
}

export interface PaginatedData<T> {
  items: T[];
  pagination: PaginationMeta;
}

// ============================================================================
// 核心通用类型
// ============================================================================

/**
 * 用户类型枚举
 */
export type UserType = 'temp' | 'normal' | 'admin' | 'system';

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface LocationInfo {
  latitude: number;
  longitude: number;
  name: string;
}

// 隐私级别枚举（用于各种隐私可见性设置）
export type PrivacyLevel = 'public' | 'friends' | 'private';

// 消息反应类型枚举
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';

export interface GeoData {
  latitude: number;
  longitude: number;
  location_name: string;
  distance: number;
  degrees: number;
}

// ============================================================================
// 地图聚合 / 视口查询（瓦片/网格）
// ============================================================================

export interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}

export interface MapClusterRequest {
  bounds: Bounds
  zoom: number
  /**
   * 可选：同一 zoom 下，网格越大 cluster 越稀疏（服务端可用该值调节聚合精度）
   */
  grid_size?: number
}

export interface MapClusterItem {
  /**
   * 服务端返回的网格 key（例如 tile_x:tile_y:zoom 或 geohash 前缀）
   * 前端只透传/展示，不做解析依赖。
   */
  key: string
  latitude: number
  longitude: number
  count: number
}

export type MapClustersData =
  | { type: 'users'; items: MapClusterItem[] }
  | { type: 'groups'; items: MapClusterItem[] }
  | { type: 'moments'; items: MapClusterItem[] }

export interface MapSearchByBoundsRequest {
  bounds: Bounds
  latitude: number
  longitude: number
  limit?: number
}

export type MapItemsData =
  | { type: 'users'; items: UserSearchItem[] }
  | { type: 'groups'; items: GroupSearchItem[] }
  | { type: 'moments'; items: MomentSearchItem[] }

export interface SearchBaseParams {
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

export interface GroupInfo {
  id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

// 时间范围筛选参数
export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

// 群组设置
export interface GroupSettings {
  join_method?: 'open' | 'approval' | 'invite_only';
  is_searchable?: boolean;
  max_members?: number;
  // 输入字段：仅用于创建/更新/加入时提交；不回显；服务端会慢哈希存储
  password?: string;
  longitude?: number;
  latitude?: number;
  location_name?: string;
}

// ============================================================================
// 用户ID编码规范工具函数
// ============================================================================

export { 
  parseUserType, 
  isTempUser, 
  isNormalUser, 
  isSystemUser,
  getUserTypeDisplayName,
  getUserTypeTagStyle,
  SYSTEM_USER_IDS,
  SYSTEM_USER_ID_MAX
} from '~/utils/userTypes'

// ============================================================================
// 认证API类型
// ============================================================================

export interface AuthData {
  user_id: number;
  nickname: string;
  access_token: string;
  // Web 场景通常不在 body 返回（使用 HttpOnly cookie 下发）
  refresh_token?: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

// Web / App 分离（与 @api_spec.mdc 一致）
export type WebAuthData = Omit<AuthData, 'refresh_token'>;
export type AppAuthData = AuthData & { refresh_token: string };

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  client_type?: 'web' | 'app';
}

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type?: 'web' | 'app';
}

export interface LogoutRequest {}

export interface GuestRequest {
  nickname: string;
  client_type?: 'web' | 'app';
}

export interface RefreshTokenRequest {
  // app：在 body 传递；web：可不传（从 HttpOnly cookie 读取）
  refresh_token?: string;
  client_type?: 'web' | 'app';
}

export interface CheckStatusRequest {}

export interface CheckStatusData {
  user_id: number;
  nickname: string;
  is_valid: boolean;
  expires_at: string;
}

export interface RequestPasswordResetRequest {
  login_id: string;
}

export interface RequestPasswordResetData {
  reset_id: string;
}

export interface VerifyResetCodeRequest {
  reset_id: string;
  code: string;
}

export interface VerifyResetCodeData {
  reset_token: string;
}

export interface ResetPasswordRequest {
  reset_token: string;
  password: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface SessionsRequest {
  limit?: number;
  cursor?: string;
}

export interface SessionItem {
  id: string;
  device_type?: string;
  device_name?: string;
  ip_address?: string;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

export interface RevokeSessionRequest {
  id: string;
}

// ============================================================================
// 用户API类型
// ============================================================================

export interface UserRelationship {
  is_mutual_follow: boolean;
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
  is_blocking: boolean;
}

export interface UserSearchItem {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  is_online?: boolean;      // 在线状态（受 online_visible 隐私设置控制）
  last_online?: string;     // 最后上线时间（受 last_online_visible 隐私设置控制，仅离线时返回）
  geo?: GeoData | null;     // 位置信息（受 location_visible 隐私设置控制，为 false 时返回 null）
}

export interface SearchUsersByKeywordRequest extends SearchBaseParams {
  kw: string;
}

export interface SearchUsersByTagRequest extends SearchBaseParams {
  tag: string;
}

export interface SearchUsersByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

export interface UserOpenTheDoorRequest {
  id: number;
  latitude: number;
  longitude: number;
}

export interface UserRelationshipRequest {
  id: number;
}

export interface UserStatsRequest {
  id: number;
}

export interface UserStatsData {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface UserFriendsRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface UserFollowingRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface UserFollowersRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface UserGroupsRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface UserLikedMomentsRequest {
  id: number;
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

// 当前用户 API
export interface MyProfileRequest {}

export interface MyProfileData {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  email?: string;
  phone?: string;
  location_name: string;
  is_online: boolean;
  last_online?: string;
  created_at: string;
}

export interface UpdateProfileRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
}

export interface UpdateLocationRequest {
  latitude: number;
  longitude: number;
  location_name?: string;
}

export interface UpgradeAccountRequest {
  login_id: string;
  password: string;
}

export interface MyStatsRequest {}

export interface MySettingsRequest {}

export interface UserSettingsData {
  privacy_level: PrivacyLevel;
  location_visible: boolean;
  online_visible: boolean;
  last_online_visible: boolean;
  friends_visible: PrivacyLevel;
  following_visible: PrivacyLevel;
  followers_visible: PrivacyLevel;
  moments_visible: PrivacyLevel;
  stats_visible: PrivacyLevel;
  allow_follow: boolean;
  allow_private_chat: 'everyone' | 'friends' | 'nobody';
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}

export interface UpdateSettingsRequest {
  privacy_level?: PrivacyLevel;
  location_visible?: boolean;
  online_visible?: boolean;
  last_online_visible?: boolean;
  friends_visible?: PrivacyLevel;
  following_visible?: PrivacyLevel;
  followers_visible?: PrivacyLevel;
  moments_visible?: PrivacyLevel;
  stats_visible?: PrivacyLevel;
  allow_follow?: boolean;
  allow_private_chat?: 'everyone' | 'friends' | 'nobody';
  searchable?: boolean;
  phone_searchable?: boolean;
  nickname_searchable?: boolean;
  location_searchable?: boolean;
}

export interface DeleteAccountRequest {
  confirmation: string;
}

export interface FollowRequest {
  id: number;
}

export interface UnfollowRequest {
  id: number;
}

export interface BlockRequest {
  id: number;
  reason?: string;
}

export interface UnblockRequest {
  id: number;
}

export interface MyFriendsRequest {
  limit?: number;
  cursor?: string;
}

export interface FriendListItem {
  user: UserInfo;
  is_online?: boolean;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

export interface MyFollowingRequest {
  limit?: number;
  cursor?: string;
}

export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
}

export interface MyFollowersRequest {
  limit?: number;
  cursor?: string;
}

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
}

export interface MyBlockedRequest {
  limit?: number;
  cursor?: string;
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

export interface SetRemarkRequest {
  id: number;
  remark?: string;
}

// ============================================================================
// 生活瞬间 API 类型
// ============================================================================

export interface MomentSearchItem {
  id: string;
  user: UserInfo;
  description: string;
  images?: string[];
  tags: string[];
  created_at: string;
  likes_count: number;
  comments_count: number;
  geo: GeoData;
}

export interface SearchMomentsByKeywordRequest extends SearchBaseParams {
  kw: string;
}

export interface SearchMomentsByTagRequest extends SearchBaseParams {
  tag: string;
}

export interface SearchMomentsByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

export interface MomentOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
}

export interface MomentCommentsRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

export interface UserMomentsRequest {
  id: number;
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

export interface CreateMomentRequest {
  description: string;
  location: LocationInfo;
  tags?: string[];
  image_file_ids?: string[];
}

export interface CreateMomentData {
  id: string;
}

export interface UpdateMomentRequest {
  id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: LocationInfo;
}

export interface DeleteMomentRequest {
  id: string;
}

export interface LikeMomentRequest {
  id: string;
}

export interface UnlikeMomentRequest {
  id: string;
}

export interface FavoriteMomentRequest {
  id: string;
}

export interface UnfavoriteMomentRequest {
  id: string;
}

export interface MyMomentsRequest {
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

export interface MyFavoritesRequest {
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

export interface CheckLikedRequest {
  id: string;
}

export interface CheckLikedData {
  is_liked: boolean;
}

export interface CheckFavoritedRequest {
  id: string;
}

export interface CheckFavoritedData {
  is_favorited: boolean;
}

export interface MomentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface MomentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ============================================================================
// 评论 API 类型
// ============================================================================

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
}

export interface CreateCommentData {
  id: string;
}

export interface DeleteCommentRequest {
  id: string;
}

export interface CommentsRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

export interface CommentListItem {
  id: string;
  user: UserInfo;
  content: string;
  parent_id?: string;
  reply_to_user?: UserInfo;
  created_at: string;
  likes_count: number;
  replies_count: number;
}

export interface CommentRepliesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface LikeCommentRequest {
  id: string;
}

export interface UnlikeCommentRequest {
  id: string;
}

export interface CommentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ============================================================================
// 私聊管理 API 类型
// ============================================================================

export interface CreatePrivateChatRequest {
  target_user_id: number;
  initial_message?: string;
}

export interface CreatePrivateChatData {
  id: string;
}

export interface MyPrivateChatsRequest {
  limit?: number;
  cursor?: string;
  include_archived?: boolean;
}

export interface PrivateChatListItem {
  id: string;
  other_user: UserInfo;
  unread_count: number;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
}

export interface PrivateChatLastMessage {
  id: string;
  content?: string;
  type: MessageType;
  sender_id: number;
  sent_at: string;
}

export interface PrivateChatOpenTheDoorRequest {
  id: string;
}

export interface PrivateChatDetailData {
  id: string;
  other_user: UserInfo;
  message_limit?: MessageLimit;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  is_archived: boolean;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
  created_at: string;
}

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
}

export interface UpdatePrivateChatSettingsRequest {
  id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
  is_archived?: boolean;
}

export interface DeletePrivateChatRequest {
  id: string;
}

// ============================================================================
// 群组管理 API 类型
// ============================================================================

export interface GroupSearchItem {
  id: string;
  name: string;
  description?: string;
  avatar_url?: string;
  tags?: string[];
  members_count: number;
  has_password: boolean;
  geo: GeoData;
}

export interface SearchGroupsByKeywordRequest extends SearchBaseParams {
  kw: string;
}

export interface SearchGroupsByTagRequest extends SearchBaseParams {
  tag: string;
}

export interface SearchGroupsByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

export interface GroupOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
}

export interface GroupSettingsRequest {
  id: string;
}

export interface GroupSettingsData {
  id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
}

export interface CreateGroupChatRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  settings?: GroupSettings;
}

export interface CreateGroupChatData {
  id: string;
}

export interface UpdateGroupChatRequest {
  id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  settings?: GroupSettings;
}

export interface DeleteGroupChatRequest {
  id: string;
}

// ============================================================================
// 群组成员管理 API 类型
// ============================================================================

export interface GroupMembersRequest {
  id: string;
  role?: 'owner' | 'admin' | 'member';
  limit?: number;
  cursor?: string;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: 'owner' | 'admin' | 'member';
  joined_at: string;
  muted_until?: string;
}

export interface InviteMemberRequest {
  id: string;
  user_id: number;
  message?: string;
}

export interface InviteMemberData {
  id: string;
  expires_at: string;
}

export interface JoinGroupRequest {
  id: string;
  password?: string;
}

export interface ApplyToGroupRequest {
  id: string;
  message?: string;
}

export interface ApplyToGroupData {
  id: string;
}

export interface SetMemberRoleRequest {
  id: string;
  user_id: number;
  role: 'admin' | 'member';
}

export interface MuteMemberRequest {
  id: string;
  user_id: number;
  duration_minutes: number;
}

export interface UnmuteMemberRequest {
  id: string;
  user_id: number;
}

export interface BanMemberRequest {
  id: string;
  user_id: number;
  reason?: string;
}

export interface UnbanMemberRequest {
  id: string;
  user_id: number;
}

export interface BannedMembersRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
}

export interface RemoveMemberRequest {
  id: string;
  user_id: number;
}

export interface GroupApplicationsRequest {
  id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  limit?: number;
  cursor?: string;
}

export interface GroupApplicationItem {
  id: string;
  applicant: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  handled_at?: string;
}

export interface ApproveApplicationRequest {
  application_id: string;
}

export interface RejectApplicationRequest {
  application_id: string;
  reason?: string;
}

export interface TransferOwnershipRequest {
  id: string;
  new_owner_id: number;
}

export interface MyGroupChatsRequest {
  limit?: number;
  cursor?: string;
}

export interface JoinedGroupItem extends GroupInfo {
  my_role: 'owner' | 'admin' | 'member';
  unread_count: number;
  last_activity_at: string;
}

export interface CancelApplicationRequest {
  application_id: string;
}

export interface MyInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  limit?: number;
  cursor?: string;
}

export interface GroupInviteItem {
  id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  expires_at?: string;
}

export interface AcceptInviteRequest {
  invite_id: string;
}

export interface RejectInviteRequest {
  invite_id: string;
}

export interface LeaveGroupRequest {
  id: string;
}

// ============================================================================
// 私聊消息 API 类型
// ============================================================================

// 消息类型枚举（统一使用，包含 system）
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'system';

// 发送消息类型枚举（客户端发送时使用，不含 system）
export type SendMessageType = Exclude<MessageType, 'system'>;

export interface SendPrivateMessageRequest {
  id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to?: string;
}

export interface SendPrivateMessageData {
  id: string;
  sent_at: string;
}

export interface PrivateMessagesHistoryRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface PrivateMessageItem {
  id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to?: string;
  media_url?: string;
  thumbnail_url?: string;
  file_name?: string;
  file_size?: number;
  duration?: number;
  width?: number;
  height?: number;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_at?: string;
}

export interface UpdatePrivateMessageRequest {
  id: string;
  content: string;
}

export interface RecallPrivateMessageRequest {
  id: string;
}

export interface DeletePrivateMessageRequest {
  id: string;
}

export interface MarkPrivateMessagesReadRequest {
  id: string;
  up_to?: string;
}

export interface AddPrivateMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface RemovePrivateMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface PrivateMessageReactionsRequest {
  id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 群聊消息 API 类型
// ============================================================================

export interface SendGroupMessageRequest {
  id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to?: string;
  mentions?: number[];
}

export interface SendGroupMessageData {
  id: string;
  sent_at: string;
}

export interface GroupMessagesHistoryRequest {
  id: string;
  limit?: number;
  cursor?: string;
  include_system?: boolean;
}

export interface GroupMessageItem {
  id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to?: string;
  mentions?: number[];
  media_url?: string;
  thumbnail_url?: string;
  file_name?: string;
  file_size?: number;
  duration?: number;
  width?: number;
  height?: number;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_by?: number;
  recalled_at?: string;
  read_count?: number;
}

export interface UpdateGroupMessageRequest {
  id: string;
  content: string;
}

export interface RecallGroupMessageRequest {
  id: string;
}

export interface DeleteGroupMessageRequest {
  id: string;
  for_everyone?: boolean;
}

export interface MarkGroupMessagesReadRequest {
  id: string;
  up_to?: string;
}

export interface MessageReadStatusRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

export interface AddGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface RemoveGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface GroupMessageReactionsRequest {
  id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
}

export interface MessageReactionItem {
  user: UserInfo;
  type: ReactionType;
  created_at: string;
}

// ============================================================================
// 消息通用 API 类型
// ============================================================================

export interface MarkAllMessagesReadRequest {}

// ============================================================================
// 举报审核 API 类型
// ============================================================================

export interface CreateReportRequest {
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
}

export interface CreateReportData {
  id: string;
}

export interface MyReportsRequest {
  limit?: number;
  cursor?: string;
}

export interface ReportItem {
  id: string;
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
  status: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
  created_at: string;
  handled_at?: string;
}

export interface ReportsRequest {
  status?: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
  target_type?: 'user' | 'moment' | 'comment' | 'message' | 'group';
  limit?: number;
  cursor?: string;
}

export interface HandleReportRequest {
  id: string;
  action: 'resolve' | 'dismiss' | 'escalate';
  note?: string;
}

// ============================================================================
// 通知订阅设置 API 类型
// ============================================================================

export interface SubscriptionsData {
  new_follower_enabled: boolean;
  mutual_follow_enabled: boolean;
  group_invite_enabled: boolean;
  group_invite_response_enabled: boolean;
  group_application_enabled: boolean;
  group_application_response_enabled: boolean;
  message_mention_enabled: boolean;
  moment_like_enabled: boolean;
  moment_comment_enabled: boolean;
  comment_reply_enabled: boolean;
  system_announcement_enabled: boolean;
}

export interface MySubscriptionsRequest {}

export interface UpdateSubscriptionsRequest {
  new_follower_enabled?: boolean;
  mutual_follow_enabled?: boolean;
  group_invite_enabled?: boolean;
  group_invite_response_enabled?: boolean;
  group_application_enabled?: boolean;
  group_application_response_enabled?: boolean;
  message_mention_enabled?: boolean;
  moment_like_enabled?: boolean;
  moment_comment_enabled?: boolean;
  comment_reply_enabled?: boolean;
  system_announcement_enabled?: boolean;
}

// ============================================================================
// 推送设备管理 API 类型
// ============================================================================

export interface RegisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony';
  device_token: string;
}

export interface UnregisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony';
  device_token: string;
}

export interface MyDeviceTokensRequest {
  limit?: number;
  cursor?: string;
}

export interface DeviceTokenItem {
  id: string;
  platform: 'ios' | 'android' | 'harmony';
  device_token: string;
  created_at: string;
}

// ============================================================================
// 系统健康检查 API 类型
// ============================================================================

export interface HealthCheckRequest {}

export interface HealthCheckData {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  timestamp: string;
}

// ============================================================================
// 文件与媒体上传 API 类型
// ============================================================================

export interface UploadMediaRequest {
  file: File;
  category: 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';
}

export interface UploadMediaData {
  file_id: string;
  thumbnail_url?: string;
  file_size: number;
  mime_type: string;
  duration?: number;
  width?: number;
  height?: number;
}

export interface InitiateUploadRequest {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: 'chat_video' | 'chat_file';
  chunk_size?: number;
}

export interface InitiateUploadData {
  upload_id: string;
  chunk_count: number;
  upload_urls: string[];
}

export interface UploadChunkRequest {
  upload_id: string;
  part_number: number;
}

export interface UploadChunkData {
  part_number: number;
  etag: string;
}

export interface CompleteUploadRequest {
  upload_id: string;
  parts: {
    part_number: number;
    etag: string;
  }[];
}

export interface UploadStatusRequest {
  upload_id: string;
}

export interface UploadStatusData {
  upload_id: string;
  status: 'in_progress' | 'completed' | 'failed';
  uploaded_parts: number[];
  total_parts: number;
  created_at: string;
  expires_at: string;
}

export interface AbortUploadRequest {
  upload_id: string;
}

// ============================================================================
// 信息流 API 类型
// ============================================================================

export interface FeedMomentsRequest {
  source?: 'following' | 'nearby' | 'all';
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedMomentItem extends MomentSearchItem {
  reason?: string;
}

export interface FeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupItem extends GroupSearchItem {
  reason?: string;
  friends_in_group?: number;
}

export interface FeedUsersRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedUserItem extends UserSearchItem {
  reason?: string;
  mutual_friends_count?: number;
  common_tags?: string[];
}

export interface FeedTrendingRequest {
  limit?: number;
}

export interface FeedTrendingData {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentSearchItem[];
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: 'rising' | 'stable' | 'falling';
  hot_moment_preview?: MomentSearchItem;
}

export interface FeedFeedbackRequest {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested' | 'report';
  reason?: string;
}

// ============================================================================
// 标签管理 API 类型
// ============================================================================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: 'daily' | 'weekly' | 'monthly';
}

export interface TrendingTagsData {
  items: TagData[];
}

export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestData {
  items: TagSuggestItem[];
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

// ============================================================================
// 系统配置管理 API 类型
// ============================================================================

export interface CreateConfigRequest {
  key: string;
  value: string;
  type: 'string' | 'number' | 'boolean';
  category: string;
  description: string;
}

export interface CreateConfigData {
  key: string;
}

export interface ConfigDetailRequest {
  key: string;
}

export interface ConfigItem {
  key: string;
  value: string;
  type: 'string' | 'number' | 'boolean';
  category: string;
  description: string;
  updated_at: string;
}

export interface ConfigListRequest {
  category?: string;
  limit?: number;
  cursor?: string;
}

export interface UpdateConfigRequest {
  key: string;
  value: string;
}

export interface DeleteConfigRequest {
  key: string;
}

export interface ConfigHistoryRequest {
  key: string;
  limit?: number;
  cursor?: string;
}

export interface ConfigHistoryItem {
  id: string;
  key: string;
  old_value: string;
  new_value: string;
  changed_by: number;
  changed_at: string;
}

export interface ConfigChangesRequest {
  start_time?: string;
  end_time?: string;
  changed_by?: number;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 管理员 RBAC API 类型
// ============================================================================

export interface AdminRoleBasic {
  id: string;
  name: string;
}

export interface AdminMeData {
  is_super: boolean;
  // '*' 表示拥有全部权限（超级管理员）
  permissions: string[];
  roles: AdminRoleBasic[];
}

export interface AdminPermissionItem {
  key: string;
  description: string;
}

export interface AdminPermissionsData {
  items: AdminPermissionItem[];
}

export interface AdminRoleItem {
  id: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface AdminRolesData {
  items: AdminRoleItem[];
}

export interface AdminAdminItem {
  user: UserInfo;
  is_super: boolean;
  roles: AdminRoleBasic[];
}

export interface AdminAdminsData {
  items: AdminAdminItem[];
}

export interface AdminRbacRoleCreateRequest {
  name: string;
  description?: string;
}

export interface AdminRbacRoleUpdateRequest {
  id: string;
  name?: string;
  description?: string;
  clear_description?: boolean;
}

export interface AdminRbacRoleDeleteRequest {
  id: string;
}

export interface AdminRbacRoleSetPermissionsRequest {
  id: string;
  permission_keys: string[];
}

export interface AdminRbacGrantAdminRequest {
  user_id: number;
  is_super?: boolean;
  role_ids: string[];
}

export interface AdminRbacRevokeAdminRequest {
  user_id: number;
}
