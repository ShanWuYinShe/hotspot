/**
 * 统一API类型定义 - 严格按照 API 规范
 *
 * 重要说明：
 * - 所有API响应都使用统一的 ApiResponse<T> 格式
 * - 业务成功判定：code === 0
 * - 对于分页查询，T 为 PaginatedData<XxxItem>
 */

// ============================================================================
// 基础响应类型
// ============================================================================

export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  request_id: string;
  data?: T;
}

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

export type UserType = 'temp' | 'normal' | 'admin' | 'system';

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface UserRelation {
  is_mutual_follow: boolean;
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
  is_blocking: boolean;
  has_pending_follow_request: boolean;
}

export interface Location {
  latitude: number;
  longitude: number;
  location_name: string;
}

export interface RelativeLocation extends Location {
  distance: number;
  bearing_degrees: number;
}

export interface GeoPoint {
  latitude: number;
  longitude: number;
}

export interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}

export type PrivacyLevel = 'public' | 'friends' | 'private';
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';
export type FieldVisibility = 'visible' | 'hidden';
export type HiddenReason = 'privacy' | 'blocked' | 'not_available';

export interface FieldVisibilityMeta {
  visibility: FieldVisibility;
  reason?: HiddenReason;
}

export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'system' | 'location';
export type SendMessageType = Exclude<MessageType, 'system'>;

export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

// ============================================================================
// 搜索请求基础参数
// ============================================================================

export interface SearchBaseParams {
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

// ============================================================================
// 认证 API 类型
// ============================================================================

export interface AuthTokenPairWeb {
  user_id: number;
  nickname: string;
  access_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

export interface AuthTokenPairApp {
  user_id: number;
  nickname: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

export type AuthData = AuthTokenPairWeb | AuthTokenPairApp;

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  client_type: 'web' | 'app';
  verify_token?: string;
}

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type: 'web' | 'app';
}

export interface GuestRequest {
  nickname: string;
  client_type: 'web' | 'app';
}

export interface RefreshTokenRequest {
  client_type: 'web' | 'app';
  refresh_token?: string;
}

export interface CheckStatusData {
  user_id: number;
  nickname: string;
  is_valid: boolean;
  expires_at: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface SessionItem {
  session_id: string;
  device_type?: string;
  device_name?: string;
  ip_address?: string;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

// ============================================================================
// 用户 API 类型
// ============================================================================

export interface UserItem extends UserInfo {
  bio?: string;
  tags?: string[];
  is_online: boolean | null;
  is_online_meta: FieldVisibilityMeta;
  last_online: string | null;
  last_online_meta: FieldVisibilityMeta;
  location?: RelativeLocation | null;
  location_meta: FieldVisibilityMeta;
  relation?: UserRelation;
}

export interface UserDetailData extends UserItem {
  created_at?: string;
  stats?: UserStatsData;
}

export interface UserStatsData {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface MyProfileData {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  gender?: 'male' | 'female' | 'other';
  email?: string;
  phone?: string;
  location: Location;
  is_online: boolean;
  last_online: string;
  created_at: string;
  stats: UserStatsData;
}

export interface UpdateProfileRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
}

export interface UpdateLocationRequest {
  location: Location;
}

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

export type UpdateSettingsRequest = Partial<UserSettingsData>;

export interface FriendListItem {
  user: UserInfo;
  is_online: boolean;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
  is_online: boolean;
}

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  is_online: boolean;
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

// ============================================================================
// 生活瞬间 API 类型
// ============================================================================

export interface MomentImage {
  url: string;
  width?: number;
  height?: number;
  content_type?: string;
}

export interface MomentItem {
  moment_id: string;
  user: UserInfo;
  description: string;
  images?: MomentImage[];
  tags: string[];
  created_at: string;
  likes_count: number;
  comments_count: number;
  location?: RelativeLocation;
  privacy_level: PrivacyLevel;
  mentions?: number[];
  is_liked?: boolean;
  is_favorited?: boolean;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  tags?: string[];
  image_file_ids?: string[];
  privacy_level?: PrivacyLevel;
  mentions?: number[];
}

export interface UpdateMomentRequest {
  moment_id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: Location;
  privacy_level?: PrivacyLevel;
  mentions?: number[];
}

export interface MomentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ============================================================================
// 评论 API 类型
// ============================================================================

export interface CommentListItem {
  user_id: number;
  comment_id: string;
  content: string;
  parent_id?: string;
  mentions?: number[];
  created_at: string;
  likes_count: number;
  replies_count: number;
}

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: number[];
}

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ============================================================================
// 私聊管理 API 类型
// ============================================================================

export interface MessageReplyTo {
  message_id: string;
  user_id: number;
  nickname: string;
  content: string;
}

export interface PrivateChatLastMessage {
  message_id: string;
  content?: string;
  type: MessageType;
  sender_id: number;
  sent_at: string;
}

export interface PrivateChatListItem {
  other_user: UserInfo;
  unread_count: number;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
  is_muted: boolean;
  is_pinned: boolean;
  is_online: boolean;
}

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
}

export interface PrivateChatDetailData {
  other_user: UserInfo;
  message_limit?: MessageLimit;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
  created_at: string;
  can_message: boolean;
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

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  media?: MessageMedia;
  location?: Location;
  location_address?: string;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_at?: string;
}

export interface SendPrivateMessageRequest {
  target_user_id: number;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  location?: Location;
  location_address?: string;
}

export interface MessageReactionItem {
  user: UserInfo;
  type: ReactionType;
  created_at: string;
}

// ============================================================================
// 群组管理 API 类型
// ============================================================================

export interface GroupInfo {
  group_chat_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface GroupMyState {
  role: 'owner' | 'admin' | 'member';
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
}

export interface GroupSettings {
  join_method?: 'open' | 'approval' | 'invite_only';
  is_searchable?: boolean;
  max_members?: number;
  password?: string;
  longitude?: number;
  latitude?: number;
  location_name?: string;
}

export interface GroupItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location?: RelativeLocation;
  my_state?: GroupMyState;
}

export interface GroupDetailData extends GroupItem {
  created_at: string;
  owner_id: number;
  settings?: GroupSettings;
  my_role?: 'owner' | 'admin' | 'member';
  is_joined?: boolean;
  unread_count?: number;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface JoinedGroupItem extends GroupItem {
  my_state: GroupMyState;
}

export interface CreateGroupChatRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  settings?: GroupSettings;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: 'owner' | 'admin' | 'member';
  joined_at: string;
  muted_until?: string;
  is_online: boolean;
}

export interface GroupApplicationItem {
  application_id: string;
  applicant: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  handled_at?: string;
  handled_by?: UserInfo;
}

export interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  expires_at?: string;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
  banned_by?: UserInfo;
}

export interface GroupAnnouncementItem {
  announcement_id: string;
  group_chat_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  created_at: string;
  is_pinned: boolean;
}

// ============================================================================
// 群聊消息 API 类型
// ============================================================================

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  mentions?: number[];
  media?: MessageMedia;
  location?: Location;
  location_address?: string;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_by?: number;
  recalled_at?: string;
  read_count?: number;
}

export interface SendGroupMessageRequest {
  group_chat_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  mentions?: number[];
  location?: Location;
  location_address?: string;
}

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

// ============================================================================
// 举报 API 类型
// ============================================================================

export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
export type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';

export interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;
  reason: string;
  description?: string;
}

export interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string;
  reason: string;
  description?: string;
  status: ReportStatus;
  created_at: string;
  handled_at?: string;
  target_snapshot?: {
    content?: string;
    media_urls?: string[];
    owner_id?: string;
  };
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

export interface NotificationItem {
  notification_id: string;
  type: string;
  title: string;
  content: string;
  payload?: Record<string, unknown>;
  is_read: boolean;
  created_at: string;
}

// ============================================================================
// 设备推送 API 类型
// ============================================================================

export type DevicePlatform = 'ios' | 'android' | 'harmony' | 'web';

export interface DeviceTokenItem {
  device_token_id: string;
  platform: DevicePlatform;
  device_token: string;
  created_at: string;
}

// ============================================================================
// 地图 API 类型
// ============================================================================

export interface MapViewport {
  bounds: Bounds;
  zoom: number;
}

export interface MapRenderRequest {
  viewport: MapViewport;
  filter?: {
    data_type?: 'user' | 'group' | 'moment' | 'all';
    only_friends?: boolean;
    time_range?: TimeRangeParams;
  };
}

export interface MapClusterData {
  count: number;
  label?: string;
  preview_urls?: string[];
}

export interface MapEntityData {
  title: string;
  subtitle?: string;
  cover_url?: string;
  payload_id: string;
  category?: string;
}

export interface MapRenderItem {
  id: string;
  type: 'cluster' | 'user' | 'group' | 'moment';
  location: GeoPoint;
  cluster?: MapClusterData;
  entity?: MapEntityData;
}

export interface MapRenderData {
  items: MapRenderItem[];
  toast_message?: string;
}

// ============================================================================
// 信息流 API 类型
// ============================================================================

export type FeedSource = 'following' | 'nearby' | 'all';

export interface FeedGroupItem extends GroupItem {
  reason?: string;
  friends_in_group?: number;
  my_role?: 'owner' | 'admin' | 'member';
  is_joined?: boolean;
}

export interface FeedUserItem extends UserItem {
  reason?: string;
  mutual_friends_count?: number;
  common_tags?: string[];
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: 'rising' | 'stable' | 'falling';
  hot_moment_preview?: MomentItem;
}

export interface FeedTrendingData {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentItem[];
}

// ============================================================================
// 标签 API 类型
// ============================================================================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

// ============================================================================
// 文件上传 API 类型
// ============================================================================

export type UploadCategory =
  | 'avatar'
  | 'group_avatar'
  | 'moment_image'
  | 'chat_image'
  | 'chat_video'
  | 'chat_audio'
  | 'chat_file';

export interface UploadMediaData {
  file_id: string;
  thumbnail_url?: string;
  file_size: number;
  mime_type: string;
  duration?: number;
  width?: number;
  height?: number;
}

export interface PresignUploadRequest {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: string;
  md5?: string;
}

export interface PresignUploadData {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

// ============================================================================
// 系统健康检查 API 类型
// ============================================================================

export interface HealthCheckData {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  timestamp: string;
}

// ============================================================================
// 系统配置 API 类型
// ============================================================================

export interface UploadConfig {
  image_max_size: number;
  video_max_size: number;
  audio_max_size: number;
  file_max_size: number;
  allowed_image_types: string[];
  allowed_video_types: string[];
}

export interface ContentDefaults {
  default_location_name: string;
  default_group_avatar: string;
  default_user_avatar: string;
  default_group_settings: GroupSettings;
}

export interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_login: boolean;
  enable_moments: boolean;
  enable_nearby_people: boolean;
}

export interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website: string;
}

export interface AppLimits {
  max_group_members: number;
  max_moment_tags: number;
  max_moment_images: number;
  moment_description_max_length: number;
  comment_max_length: number;
  search_radius_max: number;
}

export interface GlobalSettings {
  upload: UploadConfig;
  defaults: ContentDefaults;
  features: FeatureSwitches;
  contact: ContactInfo;
  limits: AppLimits;
  version: string;
}

// ============================================================================
// 管理员 RBAC API 类型
// ============================================================================

export interface AdminRoleBasic {
  role_id: string;
  name: string;
}

export interface AdminMeData {
  permissions: string[];
  roles: AdminRoleBasic[];
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
  created_at: string;
}

export interface AdminUserItem {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  email?: string;
  user_type: UserType;
  is_banned: boolean;
  banned_until?: string;
  banned_reason?: string;
  created_at: string;
  last_login_at?: string;
}

export interface AdminGroupItem {
  group_chat_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
  owner_id: number;
  is_active: boolean;
  created_at: string;
}

// ============================================================================
// 公告 API 类型
// ============================================================================

export interface AnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  is_active: boolean;
  starts_at?: string;
  ends_at?: string;
  created_at: string;
  created_by?: UserInfo;
}

// ============================================================================
// 审计日志 API 类型
// ============================================================================

export interface AuditLogItem {
  log_id: string;
  admin_user_id: number;
  admin_nickname: string;
  action: string;
  target_type?: string;
  target_id?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  created_at: string;
}
