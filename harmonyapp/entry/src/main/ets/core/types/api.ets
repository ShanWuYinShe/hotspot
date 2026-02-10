// ==================== 基础类型 ====================

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

export interface PaginationParams {
  limit?: number;
  cursor?: string;
}

// ==================== 通用类型定义 (Domain Primitives) ====================

// 消息类型枚举
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share';

// 用户类型枚举
export type UserType = 'temp' | 'normal' | 'admin' | 'system';

// 隐私级别
export type PrivacyLevel = 'public' | 'followers' | 'friends' | 'private';

// 角色
export type GroupRole = 'owner' | 'admin' | 'member';

// 客户端类型
export type ClientType = 'web' | 'app';

// 用户在线状态
export type UserPresenceStatus = 'online' | 'offline';

// 举报状态
export type ReportStatus = 'pending' | 'reviewing' | 'resolved' | 'dismissed';

// 举报目标类型
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';

// 举报处理操作
export type ReportHandleAction = 'resolve' | 'dismiss' | 'escalate';

// 群组加入方式
export type GroupJoinMethod = 'open' | 'approval' | 'invite_only';

// 公告严重程度
export type AnnouncementSeverity = 'info' | 'warning' | 'error';

// 关注来源
export type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// 推荐理由类型
export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

export interface RecommendationContext {
  type: RecommendationType;
}

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
  distance?: number;
  bearing_degrees?: number;
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
  zoom: number;
}

// ==================== 媒体资源 ====================

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
  is_blocked_by: boolean;
}

export interface SocialContext {
  mutual_friends_count: number;
  mutual_friends_preview?: UserInfo[];
  common_groups_count?: number;
  common_tags?: string[];
}

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

// 2. 群组 (Group)

export interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface MyGroupState {
  role: GroupRole;
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
}

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
  settings_location?: Location;
}

export interface GroupSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
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
  severity: AnnouncementSeverity;
  action_url?: string;
  created_at: string;
}

// 3. 生活瞬间 (Moment)

export interface MomentMyState {
  is_liked: boolean;
  is_favorited: boolean;
}

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
  my_state?: MomentMyState;

  // Feed 场景返回
  recommendation?: RecommendationContext;
}

// 4. 消息 (Message)

export interface MessageReplyTo {
  message_id: string;
  user_id: string;
  nickname: string;
  content: string;
  share_code?: string;
}

export interface MessageSent {
  message_id: string;
  sent_at: string;
}

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  media?: MessageMedia;
  share_code?: string;
  sender_id: string;
  reply_to?: MessageReplyTo;
  sent_at: string;
  edited_at?: string;
  recalled_at?: string;
  is_read?: boolean;
}

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  media?: MessageMedia;
  share_code?: string;
  sender: UserInfo;
  reply_to?: MessageReplyTo;
  mentions?: string[];
  recalled_by?: string;
  sent_at: string;
  edited_at?: string;
  recalled_at?: string;
  read_count?: number;
}

// 5. 评论 (Comment)

export interface CommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id?: string;
  created_at: string;
  likes_count: number;
  replies_count: number;
  is_liked?: boolean;
}

export interface LikeItem {
  user: UserInfo;
  liked_at: string;
}

// 6. 私聊 (Private Chat)

export interface PrivateChatLastMessage {
  message_id: string;
  content?: string;
  type: MessageType;
  share_code?: string;
  sender_id: string;
  sent_at: string;
}

export interface PrivateChatItem {
  other_user: UserInfo;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  last_message?: PrivateChatLastMessage;
  presence_status?: UserPresenceStatus;
}

export interface PrivateChatOpenResult {
  chat: PrivateChatItem;
  recent_messages: PaginatedData<PrivateMessageItem>;
}

// 7. 群组详情 (Group Open)

export interface GroupOpenResult {
  group: GroupItem;
  recent_messages: PaginatedData<GroupMessageItem>;
}

// ==================== 认证API类型 ====================

export interface AuthTokenPairApp {
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
  nickname: string;
  password: string;
  client_type: ClientType;
}

export interface LoginRequest {
  login_id: string;
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
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ==================== 用户API类型 ====================

export type Gender = 'male' | 'female' | 'apache_helicopter';
export type PrivateChatPermission = 'everyone' | 'friends' | 'nobody';

export interface SearchUsersRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest';
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface UserOpenTheDoorRequest {
  user_id: string;
  reference_point: GeoPoint;
}

export interface UserFriendsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface UserFollowingRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface UserFollowersRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface UserGroupsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface UserMomentsRequest {
  user_id: string;
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface UserLikedMomentsRequest {
  user_id: string;
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

// My Profile

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

export interface UpdateProfileRequest {
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
  friends_visible: PrivacyLevel;
  following_visible: PrivacyLevel;
  followers_visible: PrivacyLevel;
  groups_visible: PrivacyLevel;
  liked_moments_visible: PrivacyLevel;
  stats_visible: PrivacyLevel;
  allow_follow: boolean;
  allow_private_chat: PrivateChatPermission;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}

export interface UpdateSettingsRequest {
  privacy_level?: PrivacyLevel;
  location_visible?: boolean;
  presence_visible?: boolean;
  friends_visible?: PrivacyLevel;
  following_visible?: PrivacyLevel;
  followers_visible?: PrivacyLevel;
  groups_visible?: PrivacyLevel;
  liked_moments_visible?: PrivacyLevel;
  stats_visible?: PrivacyLevel;
  allow_follow?: boolean;
  allow_private_chat?: PrivateChatPermission;
  searchable?: boolean;
  phone_searchable?: boolean;
  nickname_searchable?: boolean;
  location_searchable?: boolean;
}

export interface DeleteAccountRequest {
  confirmation: string;
}

export interface FollowRequest {
  user_id: string;
  source?: FollowSource;
  source_moment_id?: string;
}

export interface UnfollowRequest {
  user_id: string;
}

export interface BlockRequest {
  user_id: string;
  reason?: string;
}

export interface UnblockRequest {
  user_id: string;
}

export interface MyFriendsRequest {
  pagination?: PaginationParams;
}

export interface MyFollowingRequest {
  pagination?: PaginationParams;
}

export interface MyFollowersRequest {
  pagination?: PaginationParams;
}

export interface MyBlockedRequest {
  pagination?: PaginationParams;
}

export interface SetRemarkRequest {
  user_id: string;
  remark: string;
}

export interface FriendItem {
  user: UserInfo;
  presence_status?: UserPresenceStatus;
  last_online?: string;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
  presence_status?: UserPresenceStatus;
  last_online?: string;
}

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  presence_status?: UserPresenceStatus;
  last_online?: string;
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

// ==================== 生活瞬间API类型 ====================

export interface SearchMomentsRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface MomentOpenTheDoorRequest {
  moment_id: string;
  reference_point: GeoPoint;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  tags?: string[];
  image_file_ids?: string[];
  privacy_level?: PrivacyLevel;
}

export interface EditMomentRequest {
  moment_id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
}

export interface DeleteMomentRequest {
  moment_id: string;
}

export interface LikeMomentRequest {
  moment_id: string;
}

export interface UnlikeMomentRequest {
  moment_id: string;
}

export interface FavoriteMomentRequest {
  moment_id: string;
}

export interface UnfavoriteMomentRequest {
  moment_id: string;
}

export interface MyMomentsRequest {
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface MyFavoritesRequest {
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface MomentLikesRequest {
  moment_id: string;
  pagination?: PaginationParams;
}

// ==================== 评论API类型 ====================

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
}

export interface DeleteCommentRequest {
  comment_id: string;
}

export interface CommentsRequest {
  moment_id: string;
  pagination?: PaginationParams;
}

export interface CommentRepliesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}

export interface LikeCommentRequest {
  comment_id: string;
}

export interface UnlikeCommentRequest {
  comment_id: string;
}

export interface CommentLikesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}

// ==================== 私聊管理API类型 ====================

export interface MyPrivateChatsRequest {
  pagination?: PaginationParams;
}

export interface PrivateChatOpenTheDoorRequest {
  target_user_id: string;
}

export interface UpdatePrivateChatSettingsRequest {
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

export interface GetPrivateMessagesHistoryRequest {
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

export interface MarkPrivateMessagesReadRequest {
  target_user_id?: string;
  up_to_message_id?: string;
}

// ==================== 群组管理API类型 ====================

export interface SearchGroupsRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface GroupOpenTheDoorRequest {
  group_id: string;
  reference_point: GeoPoint;
}

export interface CreateGroupRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  join_method?: GroupJoinMethod;
  is_searchable?: boolean;
  max_members?: number;
  location?: Location;
}

export interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  join_method?: GroupJoinMethod;
  is_searchable?: boolean;
  max_members?: number;
  location?: Location;
}

export interface DismissGroupRequest {
  group_id: string;
}

export interface UpdateGroupMySettingsRequest {
  group_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

// ==================== 群组成员管理API类型 ====================

export interface GroupMembersRequest {
  group_id: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}

export interface SearchGroupMembersRequest {
  group_id: string;
  keyword: string;
  pagination?: PaginationParams;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until?: string;
  presence_status?: UserPresenceStatus;
  last_online?: string;
}

export interface InviteMemberRequest {
  group_id: string;
  user_id: string;
  message?: string;
}

export interface Invitation {
  invite_id: string;
  expires_at: string;
}

export interface JoinGroupRequest {
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

export interface MuteMemberRequest {
  group_id: string;
  user_id: string;
  duration_minutes: number;
}

export interface UnmuteMemberRequest {
  group_id: string;
  user_id: string;
}

export interface BanMemberRequest {
  group_id: string;
  user_id: string;
  reason?: string;
}

export interface UnbanMemberRequest {
  group_id: string;
  user_id: string;
}

export interface BannedMembersRequest {
  group_id: string;
  pagination?: PaginationParams;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
  banned_by?: UserInfo;
}

export interface RemoveMemberRequest {
  group_id: string;
  user_id: string;
}

export interface GroupApplicationsRequest {
  group_id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  pagination?: PaginationParams;
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

export interface ApproveGroupApplicationRequest {
  application_id: string;
}

export interface RejectGroupApplicationRequest {
  application_id: string;
  reason?: string;
}

export interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}

export interface MyGroupsRequest {
  pagination?: PaginationParams;
}

export interface CancelApplicationRequest {
  application_id: string;
}

export interface MyInvitesRequest {
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  pagination?: PaginationParams;
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

export interface AcceptInviteRequest {
  invite_id: string;
}

export interface RejectInviteRequest {
  invite_id: string;
}

export interface LeaveGroupRequest {
  group_id: string;
}

// ==================== 群公告API类型 ====================

export interface CreateGroupAnnouncementRequest {
  group_id: string;
  title: string;
  content: string;
  is_pinned?: boolean;
}

export interface GroupAnnouncementsRequest {
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

export interface GetGroupMessagesHistoryRequest {
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

export interface MarkGroupMessagesReadRequest {
  group_id: string;
  up_to_message_id?: string;
}

// ==================== 地图渲染 API ====================

export interface MapRenderCluster {
  count: number;
  label?: string;
  preview_urls?: string[];
}

export interface MapRenderEntity {
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
  cluster?: MapRenderCluster;
  entity?: MapRenderEntity;
}

export interface MapRenderTimeRange {
  start_time?: string;
  end_time?: string;
}

export interface MapRenderFilter {
  data_type?: 'user' | 'group' | 'moment' | 'all';
  only_friends?: boolean;
  time_range?: MapRenderTimeRange;
}

export interface MapRenderRequest {
  viewport: MapViewport;
  filter?: MapRenderFilter;
}

export interface MapRenderData {
  items: MapRenderItem[];
  toast_message?: string;
}

// ==================== 举报API类型 ====================

export interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;
  reason: string;
  description?: string;
}

export interface ReportCreated {
  report_id: string;
}

export interface MyReportsRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
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
}

// ==================== 信息流API类型 ====================

export type FeedSource = 'following' | 'nearby' | 'all';

export interface GetFeedMomentsRequest {
  source: FeedSource;
  radius?: number;
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface GetFeedGroupsRequest {
  radius?: number;
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface GetFeedUsersRequest {
  radius?: number;
  reference_point: GeoPoint;
  pagination?: PaginationParams;
}

export interface GetFeedTrendingRequest {
  limit?: number;
}

export interface TrendingBundle {
  trending_tags: TagItem[];
  hot_moments: MomentItem[];
}

// ==================== 标签管理API类型 ====================

export interface TagItem {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export type TagTimeframe = 'daily' | 'weekly' | 'monthly';

export interface TrendingTagsRequest {
  timeframe?: TagTimeframe;
  limit?: number;
}

// ==================== 搜索API类型 ====================

export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

export interface SearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint;
  types?: SearchSuggestType[];
  radius?: number;
  limit?: number;
}

export interface UserSuggestItem extends UserInfo {
  bio?: string;
  tags?: string[];
  distance?: number;
}

export interface GroupSuggestItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password?: boolean;
  distance?: number;
}

export interface MomentSuggestItem {
  moment_id: string;
  description: string;
  images?: MomentImage[];
  tags?: string[];
  user: UserInfo;
  distance?: number;
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

export interface SearchSuggestItemTag {
  type: 'tag';
  payload: TagSuggestItem;
}

export interface SearchSuggestItemUser {
  type: 'user';
  payload: UserSuggestItem;
}

export interface SearchSuggestItemGroup {
  type: 'group';
  payload: GroupSuggestItem;
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
  enable_guest_login: boolean;
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
}

export interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website: string;
}

export interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
}

export interface HealthStatus {
  status: 'healthy' | 'degraded';
  version: string;
  timestamp: string;
}

export interface AnnouncementListRequest {
  pagination?: PaginationParams;
}

// ==================== 文件与媒体上传API类型 ====================

export type UploadCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

export interface PresignUploadRequest {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: UploadCategory;
  md5?: string;
}

export interface PresignedUpload {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface BatchPresignUploadRequest {
  files: PresignUploadRequest[];
}

export interface BatchPresignedUpload {
  uploads: PresignedUpload[];
}

export interface CompleteUploadRequest {
  upload_id: string;
}

export interface UploadedMedia {
  file_id: string;
  url: string;
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

export interface NavigateUserAction {
  type: 'navigate_user';
  shared_user_id: string;
}

export interface NavigateGroupAction {
  type: 'navigate_group';
  shared_group_id: string;
}

export interface NavigateMomentAction {
  type: 'navigate_moment';
  shared_moment_id: string;
}

export interface UpdateLocationAction {
  type: 'update_location';
  shared_location: Location;
}

export type ShareApplyAction =
  NavigateUserAction | NavigateGroupAction | NavigateMomentAction | UpdateLocationAction;

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

export interface ShareApplyResult {
  share_type: ShareType;
  action: ShareApplyAction;
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

export interface ApplyShareRequest {
  share_code: string;
}

export interface ShareListRequest {
  share_type?: ShareType;
  pagination?: PaginationParams;
}

export interface RevokeShareRequest {
  share_id: string;
}

// ==================== Admin Types ====================

export interface AdminUserItem {
  user: UserInfo;
  status: UserAccountStatus;
  created_at: string;
  last_active_at?: string;
  login_id: string;
  is_online: boolean;
}

export interface AdminUserDetail {
  user: UserInfo;
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
  description?: string;
  tags?: string[];
  max_members: number;
  owner: UserInfo;
  created_at: string;
  location?: Location;
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

export interface AdminRbacGrantAdminRequest {
  user_id: string;
  grant_all_permissions?: boolean;
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
  target_name?: string;
  details?: AuditDetails;
  ip_location?: string;
  ip_address?: string;
  device_summary?: string;
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
  severity: AnnouncementSeverity;
  action_url?: string;
  send_push?: boolean;
}

export interface AnnouncementCreated {
  announcement_id: string;
  created_at: string;
}

// Admin Report Types
export interface AdminReportSearchRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}

export interface HandleReportRequest {
  report_id: string;
  action: ReportHandleAction;
  note?: string;
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
