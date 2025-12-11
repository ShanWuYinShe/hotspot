/**
 * 统一API类型定义 - 严格按照 backend_doc.mdc 规范
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

export interface ApiResponse<T = any> {
  code: number;
  message?: string;
  data?: T;
}

export interface PaginatedData<T> {
  items: T[];
  pagination: {
    has_more: boolean;
    next_cursor?: string;
  };
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

export interface GeoData {
  latitude: number;
  longitude: number;
  location_name: string;
  distance: number;
  degrees: number;
}

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
// 密码安全工具函数
// ============================================================================

/**
 * 密码哈希工具已移至 ~/utils/password.ts
 * 使用方法：
 * import { hashPassword } from '~/utils/password'
 * const hash = await hashPassword('myPassword')
 * 
 * 只需要 SHA256(password)
 */

// ============================================================================
// 认证API类型
// ============================================================================

export interface AuthData {
  user_id: number;
  nickname: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password_hash: string;
}

export interface LoginRequest {
  login_id: string;
  password_hash: string;
}

export interface CreateTempRequest {
  nickname: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
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

// 重置密码
export interface ResetPasswordRequest {
  reset_token: string;
  password_hash: string;
}

export interface ChangePasswordRequest {
  old_password_hash: string;
  new_password_hash: string;
}

export interface GetSessionsRequest {
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
  geo: GeoData;
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

export interface GetRelationshipRequest {
  id: number;
}

export interface UpdateLocationRequest {
  latitude: number;
  longitude: number;
  location_name?: string;
}

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

export interface UpgradeAccountRequest {
  login_id: string;
  password_hash: string;
}

export interface GetSettingsRequest {}

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

export type GetSettingsResponse = ApiResponse<UserSettingsData>

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

export type UpdateSettingsResponse = ApiResponse

export interface DeleteAccountRequest {
  confirmation: string;
}

export interface FollowUserRequest {
  id: number;
}

export interface UnfollowUserRequest {
  id: number;
}

export interface BlockUserRequest {
  id: number;
  reason?: string;
}

export interface UnblockUserRequest {
  id: number;
}

export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

export interface GetFriendsRequest {
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

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
}

export interface GetFollowingRequest {
  limit?: number;
  cursor?: string;
}

export interface GetFollowersRequest {
  limit?: number;
  cursor?: string;
}

export interface SetFriendRemarkRequest {
  id: number;
  remark?: string;
}

export interface OpenTheDoorRequest {
  id: number;
}

export interface GetUserStatsRequest {
  id: number;
}

export interface GetMyStatsRequest {}

export interface GetUserFriendsListRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface GetUserFollowingListRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface GetUserFollowersListRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface GetUserGroupsListRequest {
  id: number;
  limit?: number;
  cursor?: string;
}

export interface GetUserLikedMomentsListRequest {
  id: number;
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

export interface UserStatsData {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface GetBlockedUsersRequest {
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 群组管理API类型
// ============================================================================

export interface GroupSettingsData {
  id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
}

export interface JoinedGroupItem extends GroupInfo {
  my_role: 'owner' | 'admin' | 'member';
  unread_count: number;
  last_activity_at: string;
}

export interface GroupSettings {
  join_method?: 'open' | 'approval' | 'invite_only';
  is_searchable?: boolean;
  max_members?: number;
  password_hash?: string;
  longitude?: number;
  latitude?: number;
  location_name?: string;
}

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

export interface GetGroupMembersRequest {
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

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
}

export interface GroupApplicationItem {
  id: string;
  applicant: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  handled_at?: string;
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

// ============================================================================
// 群组管理 API 类型
// ============================================================================

export interface CreateGroupChatRequest {
  name: string;
  description?: string;
  avatar_url?: string;
  settings?: GroupSettings;
}

export interface CreateGroupChatData {
  id: string;
}

export interface GetJoinedGroupChatsRequest {
  limit?: number;
  cursor?: string;
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
  password_hash?: string;
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

export interface GetBannedMembersRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface RemoveMemberRequest {
  id: string;
  user_id: number;
}

export interface LeaveGroupRequest {
  id: string;
}

export interface GetGroupApplicationsRequest {
  id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  limit?: number;
  cursor?: string;
}

export interface ApproveGroupApplicationRequest {
  application_id: string;
}

export interface RejectGroupApplicationRequest {
  application_id: string;
  reason?: string;
}

export interface CancelGroupApplicationRequest {
  application_id: string;
}

export interface TransferOwnershipRequest {
  id: string;
  new_owner_id: number;
}

export interface GetGroupInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  limit?: number;
  cursor?: string;
}

export interface AcceptGroupInviteRequest {
  invite_id: string;
}

export interface RejectGroupInviteRequest {
  invite_id: string;
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

export interface GetPrivateChatsRequest {
  limit?: number;
  cursor?: string;
  include_archived?: boolean;
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
// 消息 API 类型
// ============================================================================

export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file';
export type MessageType = SendMessageType | 'system';

export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';

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

export interface GetPrivateMessagesRequest {
  id: string;
  limit?: number;
  cursor?: string;
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

export interface GetPrivateMessageReactionsRequest {
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

export interface GetGroupMessagesRequest {
  id: string;
  limit?: number;
  cursor?: string;
  include_system?: boolean;
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

export interface GetMessageReadStatusRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface AddGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface RemoveGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

export interface GetGroupMessageReactionsRequest {
  id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
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

export interface MomentOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
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

export interface GetUserMomentsRequest extends SearchBaseParams {
  id: number;
}

export interface GetMyMomentsRequest extends SearchBaseParams {}

export interface GetFavoritesRequest extends SearchBaseParams {}

export interface CreateMomentRequest {
  description: string;
  location: LocationInfo;
  tags?: string[];
  image_file_ids?: string[];
}

export interface CreateMomentData {
  id: string;
}

export interface EditMomentRequest {
  id: string;
  description?: string;
  images?: string[];
  tags?: string[];
  location?: LocationInfo;
}

export interface DeleteMomentRequest {
  id: string;
}

export interface GetMomentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
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

export interface CheckMomentLikedRequest {
  id: string;
}

export interface CheckLikedData {
  is_liked: boolean;
}

export interface CheckMomentFavoritedRequest {
  id: string;
}

export interface CheckFavoritedData {
  is_favorited: boolean;
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

export interface GetCommentsRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 举报 API 类型
// ============================================================================

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

export interface CreateReportRequest {
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
}

export interface CreateReportData {
  id: string;
}

export interface GetMyReportsRequest {
  limit?: number;
  cursor?: string;
}

export interface GetReportsRequest {
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
// 系统 API 类型
// ============================================================================

export interface HealthCheckRequest {}

// ============================================================================
// 上传 API 类型
// ============================================================================

export interface UploadResponse {
  file_id: string;
  file_url: string;
  file_name: string;
  file_size: number;
  file_type: string;
  upload_time: string;
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

export interface InitiateUploadData {
  upload_id: string;
  chunk_count: number;
  upload_urls: string[];
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

export interface UploadStatusData {
  upload_id: string;
  status: 'in_progress' | 'completed' | 'failed';
  uploaded_parts: number[];
  total_parts: number;
  created_at: string;
  expires_at: string;
}

export interface UserMomentsHistoryRequest {
  user_id: string;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 私聊详情类型
// ============================================================================

export interface PrivateChatLastMessage {
  id: string;
  content?: string;
  type: SendMessageType;
  sender_id: number;
  sent_at: string;
}

export interface PrivateChatListItem {
  id: string;
  other_user: UserInfo;
  unread_count: number;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
}

export interface PrivateChatOpenTheDoorRequest {
  id: string;
}

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
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

// ============================================================================
// 消息详情类型
// ============================================================================

export interface PrivateMessageItem {
  id: string;
  type: SendMessageType;
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

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

export interface MessageReactionItem {
  user: UserInfo;
  type: ReactionType;
  created_at: string;
}

// ============================================================================
// 通知订阅 API 类型
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

export interface GetSubscriptionsRequest {}

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
// 推送设备管理 API 类型（3.13）
// ============================================================================

// 设备平台类型
export type DevicePlatform = 'ios' | 'android' | 'harmony';

// 注册设备推送Token
export interface RegisterDeviceTokenRequest {
  platform: DevicePlatform;
  device_token: string;
}

// 注销设备推送Token
export interface UnregisterDeviceTokenRequest {
  platform: DevicePlatform;
  device_token: string;
}

// 获取设备推送Token列表
export interface GetDeviceTokensRequest {
  limit?: number;
  cursor?: string;
}

// 设备推送Token
export interface DeviceTokenItem {
  id: string;
  platform: DevicePlatform;
  device_token: string;
  created_at: string;
}

// ============================================================================
// 系统健康检查 API 类型（3.14）
// ============================================================================

// 系统健康检查响应（简化版）
export interface HealthCheckData {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  timestamp: string;
}

// ============================================================================
// 信息流 API 类型
// ============================================================================

export type FeedMomentSource = 'following' | 'nearby' | 'all';

export interface GetFeedMomentsRequest {
  source?: FeedMomentSource;
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedMomentItem extends MomentSearchItem {
  reason?: string;
}

export interface GetFeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupItem extends GroupSearchItem {
  reason?: string;
  friends_in_group?: number;
}

export interface GetFeedUsersRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedUserItem extends UserSearchItem {
  reason?: string;
  mutual_friends_count?: number;
  common_tags?: string[];
}

export interface GetFeedTrendingRequest {
  limit?: number;
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: 'rising' | 'stable' | 'falling';
  hot_moment_preview?: MomentSearchItem;
}

export interface FeedTrendingData {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentSearchItem[];
}

export interface FeedFeedbackRequest {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested' | 'report';
  reason?: string;
}

// ============================================================================
// 评论点赞 API 类型
// ============================================================================

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

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

export interface GetRepliesRequest {
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

export interface GetCommentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 标签管理 API 类型
// ============================================================================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

// 获取热门标签
export interface GetTrendingTagsRequest {
  limit?: number;
  timeframe?: 'daily' | 'weekly' | 'monthly';
}

export interface TrendingTagsData {
  items: TagData[];
}

// 标签搜索建议
export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestData {
  items: string[];
}
