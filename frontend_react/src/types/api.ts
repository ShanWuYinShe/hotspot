// HotSpot API 类型定义
// 基于 @api_spec_common_types.mdc

// ============== 基础类型 ==============

export type UserType = 'temp' | 'normal' | 'admin' | 'system';
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'location' | 'system';
export type SendMessageType = Exclude<MessageType, 'system'>;
export type PrivacyLevel = 'public' | 'friends' | 'private';
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';
export type GroupRole = 'owner' | 'admin' | 'member';
export type SetableGroupRole = 'admin' | 'member';
export type Gender = 'male' | 'female' | 'apache_helicopter';
export type ClientType = 'web' | 'app';
export type UserAccountStatus = 'normal' | 'banned';
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected';
export type ResponseStatus = 'accepted' | 'rejected';
export type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';
export type GroupJoinMethod = 'open' | 'approval' | 'invite_only';
export type FeedSource = 'following' | 'nearby' | 'all';
export type PrivateChatPermission = 'everyone' | 'friends' | 'nobody';
export type AnnouncementSeverity = 'info' | 'warning' | 'error';
export type TagTrend = 'rising' | 'stable' | 'falling';
export type TagTimeframe = 'daily' | 'weekly' | 'monthly';
export type NotificationFilterType = 'like' | 'comment' | 'follow' | 'system' | 'mention';
export type NotificationType = 
  | 'moment_like' | 'moment_comment' | 'comment_reply'
  | 'new_follower' | 'mutual_follow' | 'follow_request' | 'follow_request_handled'
  | 'group_invite' | 'group_invite_response'
  | 'group_application' | 'group_application_response'
  | 'group_announcement'
  | 'system_announcement' | 'message_mention';
export type NotificationTargetType = 'moment' | 'comment' | 'user' | 'group' | 'web';
export type FeedContentType = 'moment' | 'group' | 'user';
export type FeedFeedbackAction = 'hide' | 'not_interested';
export type DevicePlatform = 'ios' | 'android' | 'harmony' | 'web';
export type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';
export type MapDataType = 'user' | 'group' | 'moment' | 'all';
export type MapItemType = 'cluster' | 'user' | 'group' | 'moment';

// ============== 位置类型 ==============

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

// ============== 分页类型 ==============

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

// ============== 媒体类型 ==============

export interface MomentImage {
  url: string;
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

export interface MessageLocation {
  location: Location;
  address?: string;
}

// ============== 用户类型 ==============

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface UserSearchItem extends UserInfo {
  bio?: string;
  tags?: string[];
  is_online?: boolean;
  last_online?: string;
  location?: RelativeLocation;
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
  social_context?: {
    mutual_friends_count: number;
    mutual_friends_preview?: UserInfo[];
    common_groups_count?: number;
    common_tags?: string[];
  };
}

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
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
  user_type?: UserType;
}

// ============== 群组类型 ==============

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

export interface MyGroupState {
  role: GroupRole;
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
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

// ============== 瞬间类型 ==============

export interface MyMomentState {
  is_liked: boolean;
  is_favorited: boolean;
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

export interface MomentSearchItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
}

export interface MomentCard extends MomentBase {
  user: UserInfo;
  location: Location;
  my_state?: MyMomentState;
}

export interface UserMomentListItem extends MomentBase {
  location: Location;
}

export interface FeedMomentItem extends MomentBase {
  user: UserInfo;
  location: RelativeLocation;
  is_liked: boolean;
  is_favorited: boolean;
  recommendation?: RecommendationContext;
}

export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

export interface RecommendationContext {
  type: RecommendationType;
  friends_preview?: UserInfo[];
}

// ============== 评论类型 ==============

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

// ============== 消息类型 ==============

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

// ============== 通知类型 ==============

export interface NotificationItem {
  notification_id: string;
  type: NotificationType;
  is_read: boolean;
  created_at: string;
  sender?: UserInfo;
  title?: string;
  content?: string;
  target_type?: NotificationTargetType;
  target_id?: string;
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

// ============== 地图类型 ==============

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

// ============== API 响应类型 ==============

export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  data?: T;
  request_id: string;
}

// ============== 认证类型 ==============

export interface AuthData {
  user_id: number;
  nickname: string;
  access_token: string;
  refresh_token?: string;
  access_token_expires_at: string;
  refresh_token_expires_at?: string;
}

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type: ClientType;
}

export interface RegisterRequest {
  login_id: string;
  password: string;
  nickname: string;
  client_type: ClientType;
  verify_token?: string;
}

export interface GuestRequest {
  nickname: string;
  client_type: ClientType;
}

export interface RefreshTokenRequest {
  client_type: ClientType;
  refresh_token?: string;  // App 端必填，Web 端留空
}
