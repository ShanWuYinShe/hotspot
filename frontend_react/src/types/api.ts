// @api_spec_common_types.mdc
export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  data?: T;
  request_id: string;
}

export interface PaginationParams {
  limit?: number;
  cursor?: string;
}

export interface PaginationMeta {
  has_more: boolean;
  next_cursor?: string;
}

export interface PaginatedData<T> {
  items: T[];
  pagination: PaginationMeta;
}

export interface GeoPoint {
  latitude: number;
  longitude: number;
}

export interface Location extends GeoPoint {
  location_name: string;
}

export interface RelativeLocation extends Location {
  distance: number;
  bearing_degrees: number;
}

export type UserType = 'temp' | 'normal' | 'admin' | 'system';
export type ClientType = 'web' | 'app';
export type Gender = 'male' | 'female' | 'apache_helicopter';
export type PrivacyLevel = 'public' | 'friends' | 'private';
export type FeedSource = 'following' | 'nearby' | 'all';
export type FeedContentType = 'moment' | 'group' | 'user';
export type FeedFeedbackAction = 'hide' | 'not_interested';
export type VerificationCodeType = 'register' | 'reset_password' | 'change_phone';
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';
export type ReportHandleAction = 'resolve' | 'dismiss' | 'escalate';
export type MemberLeaveReason = 'left' | 'kicked' | 'banned';
export type GroupJoinMethod = 'open' | 'approval' | 'invite_only';
export type GroupRole = 'owner' | 'admin' | 'member';
export type SetableGroupRole = 'admin' | 'member';
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'location' | 'system';
export type SendMessageType = Exclude<MessageType, 'system'>;
export type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';
export type AnnouncementSeverity = 'info' | 'warning' | 'error';
export type NotificationFilterType = 'like' | 'comment' | 'follow' | 'system' | 'mention';
export type NotificationType =
  | 'moment_like'
  | 'moment_comment'
  | 'comment_reply'
  | 'new_follower'
  | 'mutual_follow'
  | 'follow_request'
  | 'follow_request_handled'
  | 'group_invite'
  | 'group_invite_response'
  | 'group_application'
  | 'group_application_response'
  | 'group_announcement'
  | 'system_announcement'
  | 'message_mention';
export type NotificationTargetType = 'moment' | 'comment' | 'user' | 'group' | 'web';
export type DevicePlatform = 'ios' | 'android' | 'harmony' | 'web';
export type MediaCategory =
  | 'avatar'
  | 'group_avatar'
  | 'moment_image'
  | 'chat_image'
  | 'chat_video'
  | 'chat_audio'
  | 'chat_file';
export type TagTrend = 'rising' | 'stable' | 'falling';
export type TagTimeframe = 'daily' | 'weekly' | 'monthly';
export type MapDataType = 'user' | 'group' | 'moment' | 'all';
export type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';
export type MapItemType = 'cluster' | 'user' | 'group' | 'moment';
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected';
export type ResponseStatus = 'accepted' | 'rejected';
export type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';
export type UserAccountStatus = 'normal' | 'banned';
export type PrivateChatPermission = 'everyone' | 'friends' | 'nobody';
export type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface MyCard {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  gender?: Gender;
  email?: string;
  phone?: string;
  location: Location;
  is_online: boolean;
  last_online: string;
  created_at: string;
  stats: UserStats;
}

export interface UserSettings {
  privacy_level: PrivacyLevel;
  location_visible: boolean;
  online_visible: boolean;
  last_online_visible: boolean;
  friends_visible: PrivacyLevel;
  following_visible: PrivacyLevel;
  followers_visible: PrivacyLevel;
  groups_visible: PrivacyLevel;
  moments_visible: PrivacyLevel;
  liked_moments_visible: PrivacyLevel;
  stats_visible: PrivacyLevel;
  allow_follow: boolean;
  allow_private_chat: PrivateChatPermission;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}

export interface FriendItem {
  user: UserInfo;
  is_online?: boolean;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
  is_online?: boolean;
}

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  is_online?: boolean;
}

export interface FollowRequestItem {
  request_id: string;
  requester: UserInfo;
  message?: string;
  requested_at: string;
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

export interface Relationship {
  is_mutual_follow: boolean;
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
  has_pending_follow_request: boolean;
}

export interface UserCard extends UserInfo {
  bio?: string;
  tags?: string[];
  is_online?: boolean;
  last_online?: string;
  location?: Location;
  stats: {
    moments_count: number;
    friends_count: number;
    followers_count: number;
    following_count: number;
  };
  relation?: Relationship;
}

export interface AuthTokenPairWeb {
  user_id: number;
  nickname: string;
  access_token: string;
  access_token_expires_at: string;
}

export interface AuthTokenPairApp {
  user_id: number;
  nickname: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

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

export interface SearchBaseParams extends PaginationParams {
  reference_point?: GeoPoint;
  radius?: number;
}

export interface UserSearchItem extends UserInfo {
  bio?: string;
  tags?: string[];
  is_online?: boolean;
  last_online?: string;
  location?: RelativeLocation;
}

export interface GroupInfo {
  group_chat_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface GroupSearchItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location: RelativeLocation;
}

export interface GroupCard extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location: Location;
  created_at: string;
  owner_id: number;
  my_state?: MyGroupState;
}

export interface GroupSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
  max_members: number;
  location: Location;
}

export interface GroupSettingsResponse {
  group_chat_id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
}

export interface MyGroupState {
  role: GroupRole;
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
}

export interface MyGroupItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location: Location;
  last_activity_at: string;
  role: GroupRole;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at: string;
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

export interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: AnnouncementSeverity;
  action_url?: string;
  created_at: string;
  created_by: UserInfo;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until?: string;
  is_online?: boolean;
}

export interface Invitation {
  invite_id: string;
  expires_at: string;
}

export interface Application {
  application_id: string;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
  banned_by?: UserInfo;
}

export interface GroupApplicationItem {
  application_id: string;
  applicant: UserInfo;
  message?: string;
  status: ApplicationStatus;
  created_at: string;
  handled_at?: string;
  handled_by?: UserInfo;
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

export interface MomentImage {
  url: string;
  width?: number;
  height?: number;
  content_type?: string;
}

export interface MomentBase {
  moment_id: string;
  description: string;
  images?: MomentImage[];
  tags: string[];
  mentions?: number[];
  created_at: string;
  likes_count: number;
  comments_count: number;
}

export interface MyMomentState {
  is_liked: boolean;
  is_favorited: boolean;
}

export interface FeedMomentItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
  is_liked: boolean;
  is_favorited: boolean;
}

export interface MomentCard extends MomentBase {
  user: UserInfo;
  location: Location;
  my_state?: MyMomentState;
}

export interface UserMomentListItem extends MomentBase {
  location: Location;
}

export interface MomentSearchItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
}

export interface MomentUpsertBody {
  description?: string;
  location?: Location;
  tags?: string[];
  image_file_ids?: string[];
  privacy_level?: PrivacyLevel;
  mentions?: number[];
}

export interface CreateMomentRequest extends MomentUpsertBody {
  description: string;
  location: Location;
}

export interface EditMomentRequest extends MomentUpsertBody {
  moment_id: string;
}

export interface MomentLikeItem {
  user: UserInfo;
  liked_at: string;
}

export interface LikedStatus {
  is_liked: boolean;
}

export interface FavoritedStatus {
  is_favorited: boolean;
}

export interface CommentListItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id?: string;
  mentions?: number[];
  created_at: string;
  likes_count: number;
  replies_count: number;
}

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: number[];
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

export interface MessageLocation {
  location: Location;
  address?: string;
}

export interface MessageReplyTo {
  message_id: string;
  user_id: number;
  nickname: string;
  content: string;
}

export interface MessageBase {
  message_id: string;
  type: MessageType;
  content?: string;
  media?: MessageMedia;
  location?: MessageLocation;
  sender: UserInfo;
  reply_to?: MessageReplyTo;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_at?: string;
}

export type PrivateMessageItem = MessageBase;

export interface GroupMessageItem extends MessageBase {
  mentions?: number[];
  recalled_by?: number;
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

export interface PrivateChatListItem {
  other_user: UserInfo;
  unread_count: number;
  last_message?: {
    message_id: string;
    content?: string;
    type: MessageType;
    sender_id: number;
    sent_at: string;
  };
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

export interface Conversation {
  other_user: UserInfo;
  message_limit?: MessageLimit;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  created_at: string;
  can_message: boolean;
  recent_messages: {
    items: PrivateMessageItem[];
    has_more: boolean;
    next_cursor?: string;
  };
}

export interface NotificationItem {
  notification_id: string;
  type: NotificationType;
  is_read: boolean;
  created_at: string;
  sender?: UserInfo;
  title?: string;
  content?: string;
  target_type?: NotificationTargetType;
  target_id?: string | number;
  reference_id?: string;
  target_text_preview?: string;
  target_media_url?: string;
}

export interface UnreadCount {
  total: number;
  breakdown?: {
    like: number;
    comment: number;
    follow: number;
    system: number;
    mention: number;
  };
}

export interface Subscriptions {
  new_follower_enabled: boolean;
  mutual_follow_enabled: boolean;
  follow_request_enabled: boolean;
  follow_request_handled_enabled: boolean;
  group_invite_enabled: boolean;
  group_invite_response_enabled: boolean;
  group_application_enabled: boolean;
  group_application_response_enabled: boolean;
  group_announcement_enabled: boolean;
  message_mention_enabled: boolean;
  moment_like_enabled: boolean;
  moment_comment_enabled: boolean;
  comment_reply_enabled: boolean;
  system_announcement_enabled: boolean;
}

export interface DeviceTokenItem {
  device_token_id: string;
  platform: DevicePlatform;
  device_token: string;
  created_at: string;
}

export interface GetFeedMomentsRequest extends PaginationParams {
  source?: FeedSource;
  reference_point?: GeoPoint;
  radius?: number;
}

export interface FeedGroupItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location: RelativeLocation;
  friends_in_group?: number;
  my_state?: MyGroupState;
}

export interface FeedUserItem extends UserInfo {
  bio?: string;
  tags?: string[];
  is_online?: boolean;
  last_online?: string;
  location?: RelativeLocation;
  mutual_friends_count?: number;
  common_tags?: string[];
  relation?: Relationship;
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: TagTrend;
  hot_moment_preview?: FeedMomentItem;
}

export interface TrendingBundle {
  trending_tags: TrendingTagItem[];
  hot_moments: FeedMomentItem[];
}

export interface TagItem {
  name: string;
  usage_count: number;
  trend?: TagTrend;
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: TagTimeframe;
}

export interface TrendingTags {
  items: TagItem[];
}

export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestions {
  items: TagSuggestItem[];
}

export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

export interface MapViewport {
  northeast: GeoPoint;
  southwest: GeoPoint;
  zoom: number;
}

export interface MapRenderItem {
  id: string;
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

export interface MapRenderRequest {
  viewport: MapViewport;
  filter?: {
    data_type?: MapDataType;
    time_range?: TimeRangeParams;
    only_friends?: boolean;
  };
}

export interface HealthStatus {
  status: SystemHealthStatus;
  version: string;
  timestamp: string;
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
}

export interface AdminSettings extends GlobalSettings {
  updated_at: string;
  updated_by_user_id: number;
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

export interface ReportCreated {
  report_id: string;
}

export interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string | number;
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

export interface AnnouncementCreated {
  announcement_id: string;
  created_at: string;
}

export interface AdminUserItem extends UserInfo {
  status: UserAccountStatus;
  created_at: string;
  last_active_at?: string;
  login_id: string;
  is_online: boolean;
}

export interface AdminBanRecord {
  banned_at: string;
  banned_by: UserInfo;
  reason?: string;
  duration_days?: number;
  unbanned_at?: string;
  unbanned_by?: UserInfo;
}

export interface AdminUserDetail extends UserInfo {
  status: UserAccountStatus;
  created_at: string;
  last_active_at?: string;
  login_id: string;
  email?: string;
  phone?: string;
  bio?: string;
  tags?: string[];
  location?: Location;
  is_online: boolean;
  stats: UserStats;
  ban_history?: AdminBanRecord[];
}

export interface AdminGroupItem extends GroupInfo {
  owner: UserInfo;
  created_at: string;
}

export interface AdminDismissalRecord {
  dismissed_at: string;
  dismissed_by: UserInfo;
  reason?: string;
}

export interface AdminGroupDetail extends GroupInfo {
  description?: string;
  tags?: string[];
  max_members: number;
  owner: UserInfo;
  created_at: string;
  location?: Location;
  settings: {
    join_method: GroupJoinMethod;
    is_searchable: boolean;
    has_password: boolean;
  };
  admins?: UserInfo[];
  stats: {
    messages_count: number;
    active_members_today: number;
  };
  dismissal_history?: AdminDismissalRecord[];
}

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
  target_name?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface AuditLogDetail extends AuditLogItem {
  before_state?: Record<string, unknown>;
  after_state?: Record<string, unknown>;
}

export interface AuditLogExportResult {
  export_id: string;
  status: 'pending' | 'processing' | 'ready' | 'failed';
  download_url?: string;
  expires_at?: string;
}
/*
// @api_spec_common_types.mdc
export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  data?: T;
  request_id: string;
}

export interface PaginationParams {
  limit?: number; // Default: 20, Max: 100
  cursor?: string; // Opaque string
}

export interface PaginationMeta {
  has_more: boolean;
  next_cursor?: string;
}

export interface PaginatedData<T> {
  items: T[];
  pagination: PaginationMeta;
}

export interface GeoPoint {
  latitude: number;
  longitude: number;
}

export interface Location extends GeoPoint {
  location_name: string;
}

export interface RelativeLocation extends Location {
  distance: number;
  bearing_degrees: number;
}

export type VerificationCodeType = 'register' | 'reset_password' | 'change_phone';
export type ClientType = 'web' | 'app';
export type UserType = 'temp' | 'normal' | 'admin' | 'system';

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface AuthTokenPairWeb {
  user_id: number;
  nickname: string;
  access_token: string;
  access_token_expires_at: string;
}

export interface AuthTokenPairApp {
  user_id: number;
  nickname: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface MyCard {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  // gender?: Gender; // Gender not defined yet
  email?: string;
  phone?: string;
  location: Location;
  is_online: boolean;
  last_online: string;
  created_at: string;
  stats: UserStats;
}

// Common Enum Types
export type PrivacyLevel = 'public' | 'friends' | 'private';
export type FeedSource = 'following' | 'nearby' | 'all';
export type FeedContentType = 'moment' | 'group' | 'user';
export type FeedFeedbackAction = 'hide' | 'not_interested';

// Moment Types
export interface MomentImage {
  url: string;
  width?: number;
  height?: number;
  content_type?: string;
}

export interface MomentBase {
  moment_id: string;
  description: string;
  images?: MomentImage[];
  tags: string[];
  mentions?: number[];
  created_at: string;
  likes_count: number;
  comments_count: number;
}

export interface MyMomentState {
  is_liked: boolean;
  is_favorited: boolean;
}

export interface FeedMomentItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
  is_liked: boolean;
  is_favorited: boolean;
}

export interface MomentCard extends MomentBase {
  user: UserInfo;
  location: Location;
  my_state?: MyMomentState;
}

export interface UserMomentListItem extends MomentBase {
  location: Location;
}

export interface MomentSearchItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
}

export interface UserSearchItem extends UserInfo {
  bio?: string;
  tags?: string[];
  
  // 隐私受控字段（隐私限制时不返回，不暴露原因）
  is_online?: boolean;
  last_online?: string;
  
  // 位置信息（RelativeLocation 包含距离）
  location?: RelativeLocation;
}

export interface MomentUpsertBody {
  description?: string;
  location?: Location;
  tags?: string[];
  image_file_ids?: string[];
  privacy_level?: PrivacyLevel;
  mentions?: number[];
}

export interface CreateMomentRequest extends MomentUpsertBody {
  description: string;
  location: Location;
}

export interface EditMomentRequest extends MomentUpsertBody {
  moment_id: string;
}

// Feed Types
export interface GetFeedMomentsRequest extends PaginationParams {
  source?: FeedSource;
  reference_point?: GeoPoint;
  radius?: number;
}

// Comment Types
export interface CommentListItem {
  comment_id: string;
  user: UserInfo;
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
*/
