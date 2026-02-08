// ==================== 基础类型 ====================

export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  // 服务端回显 `X-Request-Id`；每次请求必返，用于链路追踪
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

export interface SearchBaseParams {
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

// ==================== 通用类型定义 (Domain Primitives) ====================

export type UserType = 'temp' | 'normal' | 'admin' | 'system';
export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'location';
export type MessageType = SendMessageType | 'system';
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';
export type PrivacyLevel = 'public' | 'friends' | 'private';
export type GroupRole = 'owner' | 'admin' | 'member';
export type UserAccountStatus = 'active' | 'banned' | 'suspended';
export type GroupJoinMethod = 'open' | 'approval' | 'invite_only';

// 基础坐标点
export interface GeoPoint {
  latitude: number;
  longitude: number;
}

// 命名位置
export interface Location extends GeoPoint {
  location_name: string;
}

// 相对位置 (用于搜索结果/附近推荐)
export interface RelativeLocation extends Location {
  distance: number;
  bearing_degrees: number;
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

// 字段可见性元数据
export type FieldVisibility = 'visible' | 'hidden';
export type HiddenReason = 'privacy' | 'blocked' | 'not_available';

export interface FieldVisibilityMeta {
  visibility: FieldVisibility;
  reason?: HiddenReason;
}

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

// ==================== 领域实体定义 (Domain Entities) ====================

// 1. 用户 (User)

export interface UserInfo {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
}

export interface UserRelation {
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
  is_blocking: boolean;
  has_pending_follow_request: boolean;
}

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

export interface UserStats {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
}

export interface UserDetailData extends UserItem {
  created_at?: string;
  stats?: UserStatsData;
}

// My Profile Type (from spec)
export interface MyCard {
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

// 2. 群组 (Group)

export interface GroupInfo {
  group_id: string;
  name: string;
  avatar_url?: string;
  members_count: number;
}

export interface GroupMyState {
  role: GroupRole;
  is_joined: boolean;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  joined_at?: string;
}

export interface GroupItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location: RelativeLocation;
  created_at?: string;
  owner_id?: string;
  my_state?: GroupMyState;
}

export type JoinedGroupItem = GroupItem;

export interface GroupSettings {
  join_method: GroupJoinMethod;
  is_searchable: boolean;
  max_members: number;
  location: RelativeLocation;
}

export type GroupDetailData = GroupItem;

// 3. 生活瞬间 (Moment)

export interface MomentItem {
  moment_id: string;
  user: UserInfo;
  
  description: string;
  images?: MomentImage[];
  tags: string[];
  
  location?: RelativeLocation;
  
  created_at: string;
  
  likes_count: number;
  comments_count: number;
  
  is_liked: boolean;
  is_favorited: boolean;
}

// 4. 消息 (Message)

export interface MessageReplyTo {
  message_id: string;
  user_id: number;
  nickname: string;
  content: string;
}

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  media?: MessageMedia;
  sender_id: number;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  sent_at: string;
  edited_at?: string;
  recalled_at?: string;
}

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  media?: MessageMedia;
  sender: UserInfo;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  mentions?: number[];
  sent_at: string;
  edited_at?: string;
  recalled_by?: number;
  recalled_at?: string;
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

// ==================== 认证API类型 ====================

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

export type AuthData = AuthTokenPairApp; // HarmonyOS use App type

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  client_type: 'web' | 'app';
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
  is_valid: boolean;
  expires_at: string;
}

// Verification Code Types
export interface VerificationCodeSent {
  code_id: string;
  expires_at: string;
  retry_after?: number;
}

export interface VerificationResult {
  verified: boolean;
  verify_token: string;
  expires_at: string;
}

// Password Reset Types
export interface ResetRequested {
  reset_id: string;
}

export interface ResetVerified {
  reset_token: string;
}

// Session Management Types
export interface SessionItem {
  display_id: string;         // 展示用短标识
  revoke_token: string;         // 撤销用一次性令牌
  device_type?: string;
  device_name?: string;
  ip_address?: string;
  ip_location?: string;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
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
  session_id: string;
  device_type?: string;
  device_name?: string;
  ip_address?: string;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

export interface RevokeSessionRequest {
  session_id: string;
}

// ==================== 用户API类型 ====================

export interface SearchBaseParams {
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
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
  user_id: number;
  reference_point: GeoPoint;
}

export interface GetRelationshipRequest {
  user_id: number;
}

export interface UserStatsRequest {
  user_id: number;
}

export interface UserStatsData {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

export interface UserFriendsRequest {
  user_id: number;
  limit?: number;
  cursor?: string;
}

export interface UserFollowingRequest {
  user_id: number;
  limit?: number;
  cursor?: string;
}

export interface UserFollowersRequest {
  user_id: number;
  limit?: number;
  cursor?: string;
}

export interface UserGroupsRequest {
  user_id: number;
  limit?: number;
  cursor?: string;
}

export interface UserLikedMomentsRequest extends SearchBaseParams {
  user_id: number;
}

// 当前用户 API
export interface MyProfileRequest {}

export interface MyProfileData {
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
  groups_visible: PrivacyLevel;
  liked_moments_visible: PrivacyLevel;
  stats_visible: PrivacyLevel;
  allow_follow: boolean;
  allow_private_chat: 'everyone' | 'friends' | 'nobody';
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}

// User Relationship and Friend Types
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
  user_id: number;
}

export interface UnfollowRequest {
  user_id: number;
}

export interface BlockRequest {
  user_id: number;
  reason?: string;
}

export interface UnblockRequest {
  user_id: number;
}

export interface MyFriendsRequest {
  limit?: number;
  cursor?: string;
}

export interface FriendListItem {
  user: UserInfo;
  is_online: boolean;
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
  is_online: boolean;
}

export interface MyFollowersRequest {
  limit?: number;
  cursor?: string;
}

export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  is_online: boolean;
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
  user_id: number;
  remark?: string;
}

// ==================== 生活瞬间API类型 ====================

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
  moment_id: string;
  reference_point: GeoPoint;
}

export interface UserMomentsRequest extends SearchBaseParams {
  user_id: number;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  tags?: string[];
  image_file_ids?: string[];
}

export interface UpdateMomentRequest {
  moment_id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: Location;
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

export interface MyMomentsRequest extends SearchBaseParams {}

export interface MyFavoritesRequest extends SearchBaseParams {}

export interface CheckLikedRequest {
  moment_id: string;
}

export interface CheckLikedData {
  is_liked: boolean;
}

export interface CheckFavoritedRequest {
  moment_id: string;
}

export interface CheckFavoritedData {
  is_favorited: boolean;
}

export interface MomentLikesRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

export interface MomentLikeItem {
  user: UserInfo;
  liked_at: string;
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
  limit?: number;
  cursor?: string;
}

export interface CommentListItem {
  comment_id: string;
  user: UserInfo;
  content: string;
  parent_id?: string;
  created_at: string;
  likes_count: number;
  replies_count: number;
}

export interface CommentRepliesRequest {
  comment_id: string;
  limit?: number;
  cursor?: string;
}

export interface LikeCommentRequest {
  comment_id: string;
}

export interface UnlikeCommentRequest {
  comment_id: string;
}

export interface CommentLikesRequest {
  comment_id: string;
  limit?: number;
  cursor?: string;
}

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// Moment Status Types
export interface LikedStatus {
  is_liked: boolean;
}

export interface FavoritedStatus {
  is_favorited: boolean;
}

// Moment Request Types
export interface MomentOpenTheDoorRequest {
  moment_id: string;
  reference_point: GeoPoint;
}

export interface UserMomentsRequest extends SearchBaseParams {
  user_id: number;
}

export interface EditMomentRequest {
  moment_id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: Location;
}

export interface GetMomentInteractionStateRequest {
  moment_id: string;
}

// ==================== 私聊管理API类型 ====================

export interface MyPrivateChatsRequest {
  limit?: number;
  cursor?: string;
}

export interface PrivateChatOpenTheDoorRequest {
  target_user_id: number;
}

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
}

export interface PrivateChatLastMessage {
  message_id: string;
  content?: string;
  type: MessageType;
  sender_id: number;
  sent_at: string;
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

export interface UpdatePrivateChatSettingsRequest {
  target_user_id: number;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeletePrivateChatRequest {
  target_user_id: number;
}

// ==================== 群组管理API类型 ====================

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
  group_id: string;
}

export interface GroupSettingsRequest {
  group_id: string;
}

export interface GroupSettingsData {
  group_id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
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

export interface UpdateGroupMySettingsRequest {
  group_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeleteGroupRequest {
  group_id: string;
}

// ==================== 群组成员管理API类型 ====================

export interface GroupMembersRequest {
  group_id: string;
  role?: 'owner' | 'admin' | 'member';
  limit?: number;
  cursor?: string;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: 'owner' | 'admin' | 'member';
  joined_at: string;
  muted_until?: string;
  is_online: boolean;
}

export interface InviteMemberRequest {
  group_id: string;
  user_id: number;
  message?: string;
}

export interface InviteMemberData {
  invite_id: string;
  expires_at: string;
}

export interface JoinGroupRequest {
  group_id: string;
  password?: string;
}

export interface ApplyToGroupRequest {
  group_id: string;
  message?: string;
}

export interface ApplyToGroupData {
  application_id: string;
}

export interface SetMemberRoleRequest {
  group_id: string;
  user_id: number;
  role: 'admin' | 'member';
}

export interface MuteMemberRequest {
  group_id: string;
  user_id: number;
  duration_minutes: number;
}

export interface UnmuteMemberRequest {
  group_id: string;
  user_id: number;
}

export interface BanMemberRequest {
  group_id: string;
  user_id: number;
  reason?: string;
}

export interface UnbanMemberRequest {
  group_id: string;
  user_id: number;
}

export interface BannedMembersRequest {
  group_id: string;
  limit?: number;
  cursor?: string;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
  banned_by?: UserInfo;
}

export interface RemoveMemberRequest {
  group_id: string;
  user_id: number;
}

export interface GroupApplicationsRequest {
  group_id: string;
  status?: 'pending' | 'accepted' | 'rejected' | 'all';
  limit?: number;
  cursor?: string;
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
  new_owner_id: number;
}

export interface MyGroupsRequest {
  limit?: number;
  cursor?: string;
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

// ==================== 私聊消息API类型 ====================

export interface SendPrivateMessageRequest {
  target_user_id: number;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
}

export interface PrivateMessagesHistoryRequest {
  target_user_id: number;
  limit?: number;
  cursor?: string;
}

export interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallPrivateMessageRequest {
  message_id: string;
}

export interface DeletePrivateMessageRequest {
  message_id: string;
}

export interface MarkPrivateMessagesReadRequest {
  target_user_id: number;
  up_to_message_id?: string;
}

export interface AddPrivateMessageReactionRequest {
  message_id: string;
  type: ReactionType;
}

export interface RemovePrivateMessageReactionRequest {
  message_id: string;
  type: ReactionType;
}

export interface PrivateMessageReactionsRequest {
  message_id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
}

// Private Chat Management Types
export interface Conversation {
  other_user: UserInfo;
  message_limit?: MessageLimit;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  created_at: string;
  can_message: boolean;
}

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
}

export interface MessageReactionItem {
  user: UserInfo;
  type: ReactionType;
  created_at: string;
}

export interface PrivateChatLastMessage {
  message_id: string;
  content?: string;
  type: MessageType;
  sender_id: number;
  sent_at: string;
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

// ==================== 群聊消息API类型 ====================

export interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  mentions?: number[];
}

export interface GroupMessagesHistoryRequest {
  group_id: string;
  limit?: number;
  cursor?: string;
  include_system?: boolean;
}

export interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}

export interface RecallGroupMessageRequest {
  message_id: string;
}

export interface DeleteGroupMessageRequest {
  message_id: string;
  for_everyone?: boolean;
}

export interface MarkGroupMessagesReadRequest {
  group_id: string;
  up_to_message_id?: string;
}

export interface MessageReadStatusRequest {
  message_id: string;
  limit?: number;
  cursor?: string;
}

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

export interface AddGroupMessageReactionRequest {
  message_id: string;
  type: ReactionType;
}

export interface RemoveGroupMessageReactionRequest {
  message_id: string;
  type: ReactionType;
}

export interface GroupMessageReactionsRequest {
  message_id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
}

// ==================== 消息通用 API ====================

export interface MarkAllMessagesReadRequest {}

// ==================== 地图渲染 API ====================

export interface MapRenderItem {
  id: string;
  type: 'cluster' | 'user' | 'group' | 'moment';
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

export interface MapRenderRequest {
  viewport: MapViewport;
  filter?: {
    data_type?: 'user' | 'group' | 'moment' | 'all';
    only_friends?: boolean;
    time_range?: {
      start_time?: string;
      end_time?: string;
    };
  };
}

export interface MapRenderData {
  items: MapRenderItem[];
  toast_message?: string;
}

// ==================== 举报审核API类型 ====================

export interface CreateReportRequest {
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
}

export interface CreateReportData {
  report_id: string;
}

export interface ReportCreated {
  report_id: string;
}

export interface MyReportsRequest extends PaginationParams {}

export interface ReportItem {
  report_id: string;
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
  status: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
  created_at: string;
  handled_at?: string;
}

// ==================== 通知中心 API类型 ====================

export interface GetNotificationsRequest {
  limit?: number;
  cursor?: string;
  type?: 'like' | 'comment' | 'follow' | 'system' | 'mention';
}

export interface NotificationItem {
  notification_id: string;
  type: 'like' | 'comment' | 'follow' | 'system' | 'mention';
  title: string;
  message: string;
  created_at: string;
  read: boolean;
}

// ==================== 私聊API类型 ====================

export interface MyPrivateChatsRequest extends PaginationParams {}

export interface PrivateChatListItem {
  private_chat_id: string;
  user: UserInfo;
  last_message?: PrivateMessageItem;
  last_activity_at: string;
  unread_count: number;
}

export interface PrivateChatOpenTheDoorRequest {}

export interface PrivateChatLastMessage {
  private_chat_id: string;
  last_message: PrivateMessageItem;
}

export interface UpdatePrivateChatSettingsRequest {
  quiet?: boolean;
  enable_notifications?: boolean;
}

export interface DeletePrivateChatRequest {
  private_chat_id: string;
}

export interface CreateGroupRequest {
  user_ids: number[];
  name?: string;
  description?: string;
  avatar_file_id?: string;
}

export interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
}

export interface DeleteGroupRequest {
  group_id: string;
}

export interface MyGroupsRequest extends PaginationParams {}

export interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: 'pending' | 'accepted' | 'rejected';
  created_at: string;
  expires_at?: string;
}

export interface GroupInviteItem extends PaginationParams {}

export interface PrivateMessagesHistoryRequest {
  private_chat_id: string;
  limit?: number;
  cursor?: string;
}

export interface MarkPrivateMessagesReadRequest {
  private_chat_id: string;
  message_ids?: string[];
}

export interface PrivateMessageReactionsRequest {
  private_chat_id: string;
  message_id: string;
  limit?: number;
  cursor?: string;
}

export interface GroupMessagesHistoryRequest {
  group_id: string;
  limit?: number;
  cursor?: string;
}

export interface MarkGroupMessagesReadRequest {
  group_id: string;
  message_ids?: string[];
}

export interface MyReportsRequest {
  limit?: number;
  cursor?: string;
}

export interface ReportItem {
  report_id: string;
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
  status: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
  created_at: string;
  handled_at?: string;
}

// ==================== 通知中心 API ====================

export interface GetNotificationsRequest {
  limit?: number;
  cursor?: string;
  type?: 'like' | 'comment' | 'follow' | 'system' | 'mention';
}

export interface NotificationItem {
  notification_id: string;
  type: 'moment_like' | 'moment_comment' | 'comment_reply' | 'new_follower' | 'mutual_follow' | 'system_announcement' | 'message_mention';
  is_read: boolean;
  created_at: string;
  sender?: UserInfo;
  title?: string;
  content?: string;
  target_type?: 'moment' | 'comment' | 'user' | 'group' | 'web';
  target_id?: string;
  reference_id?: string;
  target_text_preview?: string;
  target_media_url?: string;
}

export interface GetUnreadNotificationCountRequest {}

export interface UnreadNotificationCountData {
  total: number;
  breakdown?: {
    like: number;
    comment: number;
    follow: number;
    system: number;
    mention: number;
  };
}

export interface MarkNotificationsReadRequest {
  notification_ids?: string[];
  type?: 'like' | 'comment' | 'follow' | 'system' | 'mention';
}

// ==================== 通知订阅设置API类型 ====================

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

// ==================== 推送设备管理API类型 ====================

export interface RegisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
}

export interface UnregisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
}

export interface MyDeviceTokensRequest {
  limit?: number;
  cursor?: string;
}

export interface DeviceTokenItem {
  device_token_id: string;
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
  created_at: string;
}

// ==================== 系统健康检查API类型 ====================

export interface SystemHealthRequest {}

export interface HealthCheckData {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  timestamp: string;
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

export interface PresignUploadData {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface CompleteUploadRequest {
  upload_id: string;
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

// ==================== 信息流API类型 ====================

export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

export interface RecommendationContext {
  type: RecommendationType;
  friends_preview?: UserInfo[];
}

export interface GetFeedMomentsRequest {
  source?: 'following' | 'nearby' | 'all';
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface GetFeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface GetFeedUsersRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface GetFeedTrendingRequest {
  limit?: number;
}

export interface FeedGroupItem extends GroupItem {
  friends_in_group?: number;
  my_role?: GroupRole;
  is_joined?: boolean;
}

export interface FeedUserItem extends UserItem {
  mutual_friends_count?: number;
  common_tags?: string[];
}

export interface TrendingBundle {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentItem[];
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: 'rising' | 'stable' | 'falling';
  hot_moment_preview?: MomentItem;
}

export interface FeedFeedbackRequest {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested';
  reason?: string;
}

// ==================== 标签管理API类型 ====================

export interface TagItem {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: 'daily' | 'weekly' | 'monthly';
}

export interface TrendingTagsData {
  items: TagItem[];
}

export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

export interface TagSuggestions {
  items: TagSuggestItem[];
}

export interface GetFeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupItem extends GroupItem {
  friends_in_group?: number;
  my_role?: GroupRole;
  is_joined?: boolean;
}

export interface GetFeedUsersRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedUserItem extends UserItem {
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
  hot_moment_preview?: MomentItem;
}

export interface FeedTrendingData {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentItem[];
}

export interface FeedFeedbackRequest {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested';
  reason?: string;
}

// ==================== 标签管理API类型 ====================

export interface TagItem {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: 'daily' | 'weekly' | 'monthly';
}

export interface TrendingTagsData {
  items: TagItem[];
}

export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

export interface TagSuggestions {
  items: TagSuggestItem[];
}

// ==================== 地图渲染API类型 ====================

export interface MapRenderRequest {
  viewport: MapViewport;
  filter?: {
    data_type?: 'user' | 'group' | 'moment' | 'all';
    time_range?: TimeRangeParams;
    only_friends?: boolean;
  };
}

export interface MapCanvas {
  items: MapRenderItem[];
  toast_message?: string;
}

// ==================== 通知中心 API类型 ====================

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

export interface GetNotificationsRequest extends PaginationParams {
  type?: 'like' | 'comment' | 'follow' | 'system' | 'mention';
}

export interface MarkNotificationsReadRequest {
  notification_ids?: string[];
  type?: 'like' | 'comment' | 'follow' | 'system' | 'mention';
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

export interface MySubscriptionsRequest {}

export interface UpdateSubscriptionsRequest extends Partial<Subscriptions> {}

// Device Token Types
export interface DeviceTokenItem {
  device_token_id: string;
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
  created_at: string;
}

export interface RegisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
}

export interface UnregisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony' | 'web';
  device_token: string;
}

export interface MyDeviceTokensRequest extends PaginationParams {}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: 'daily' | 'weekly' | 'monthly';
}

export interface TrendingTagsData {
  items: TagItem[];
}

export interface SuggestTagsRequest {
  prefix: string;
  limit?: number;
}

export interface TagSuggestItem {
  name: string;
  usage_count: number;
}

export interface TagSuggestData {
  items: TagSuggestItem[];
}

// ==================== 系统配置API类型 ====================

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

export interface FetchGlobalSettingsRequest {
  context: {
    os: 'ios' | 'android' | 'web' | 'harmony';
    app_version: string;
    device_id?: string;
    language?: string;
  };
  current_version?: string;
}

export interface FetchSettingsData {
  action: 'update' | 'no_change' | 'force_update';
  config?: GlobalSettings;
  latest_version: string;
}

// ==================== 管理员系统公告 API ====================

export interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  action_url?: string;
  created_at: string;
  created_by: UserInfo;
}

// Group Chat Additional Types
export interface GroupSettingsResponse {
  group_id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
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

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

export interface PublicAnnouncementListRequest {
  limit?: number;
  cursor?: string;
}

// ==================== Admin Types ====================

export interface AdminUserItem extends UserInfo {
  status: UserAccountStatus;
  created_at: string;
  last_active_at?: string;
  login_id: string;
  is_online: boolean;
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
  stats: UserStatsData;
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

export interface AdminGroupItem extends GroupInfo {
  owner: UserInfo;
  created_at: string;
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
  user_id: number;
  grant_all_permissions?: boolean;
  role_ids: string[];
}

export interface AdminRbacRevokeAdminRequest {
  user_id: number;
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

export type AuditDetails =
  | { type: 'user.ban'; reason?: string; duration_days?: number }
  | { type: 'user.unban'; reason?: string }
  | { type: 'group.dismiss'; reason?: string }
  | { type: 'announcement.create'; title: string }
  | { type: 'announcement.delete'; title: string }
  | { type: 'role.create'; role_name: string; permissions: string[] }
  | { type: 'role.update'; role_name: string; added_permissions?: string[]; removed_permissions?: string[] }
  | { type: 'role.delete'; role_name: string }
  | { type: 'permission.grant'; permission: string }
  | { type: 'permission.revoke'; permission: string }
  | { type: 'config.update'; config_key: string };

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

export interface AuditLogDetail extends AuditLogItem {
  before_state?: StateSnapshot;
  after_state?: StateSnapshot;
}

export type StateSnapshot =
  | { type: 'user'; nickname?: string; status?: string; roles?: string[] }
  | { type: 'group'; name?: string; members_count?: number }
  | { type: 'role'; name?: string; permissions?: string[] }
  | { type: 'config'; value?: string };

export interface AuditLogListRequest extends PaginationParams {
  action?: AuditAction;
  operator_id?: number;
  target_type?: AuditTargetType;
  target_id?: string;
  start_time?: string;
  end_time?: string;
}

export interface AuditLogDetailRequest {
  log_id: string;
}

export interface AuditLogExportRequest {
  action?: AuditAction;
  operator_id?: number;
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

export interface CreateAnnouncementRequest {
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  action_url?: string;
  send_push?: boolean;
}

export interface AnnouncementCreated {
  announcement_id: string;
  created_at: string;
}

export interface AnnouncementListRequest extends PaginationParams {}

// Admin Report Types
export interface ReportsRequest extends PaginationParams {
  status?: 'pending' | 'reviewing' | 'resolved' | 'dismissed';
  target_type?: 'user' | 'moment' | 'comment' | 'message' | 'group';
}

export interface HandleReportRequest {
  report_id: string;
  action: 'resolve' | 'dismiss' | 'escalate';
  note?: string;
}

// Admin User Types
export interface AdminUserSearchRequest extends PaginationParams {
  user_id?: number;
  nickname?: string;
  login_id?: string;
  status?: UserAccountStatus;
  user_type?: UserType;
}

export interface AdminUserDetailRequest {
  user_id: number;
}

export interface AdminBanUserRequest {
  user_id: number;
  reason?: string;
  duration_days?: number;
}

export interface AdminUnbanUserRequest {
  user_id: number;
  reason?: string;
}

export interface AdminGroupSearchRequest extends PaginationParams {
  group_id?: string;
  name?: string;
  owner_id?: number;
}

export interface AdminGroupDetailRequest {
  group_id: string;
}

export interface AdminDismissGroupRequest {
  group_id: string;
  reason?: string;
}

// Admin Audit Types
export interface AuditLogListRequest extends PaginationParams {
  action?: AuditAction;
  operator_id?: number;
  target_type?: AuditTargetType;
  target_id?: string;
  start_time?: string;
  end_time?: string;
}

export interface AuditLogDetailRequest {
  log_id: string;
}

// ==================== 搜索API类型 ====================

export type SearchSuggestItem =
  | { type: 'tag'; payload: TagSuggestItem }
  | { type: 'user'; payload: UserSuggestItem }
  | { type: 'group'; payload: GroupSuggestItem }
  | { type: 'moment'; payload: MomentSuggestItem };

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

export interface SearchSuggestions {
  items: SearchSuggestItem[];
}

export interface SearchSuggestRequest {
  prefix: string;
  limit?: number;
  location?: GeoPoint;
}

// ==================== 客户端扩展类型别名 ====================
// 这些类型别名用于客户端代码，实际 API 返回的是 MomentItem、GroupItem、UserItem
// 为了保持代码一致性，使用这些别名

export type MomentSearchItem = MomentItem;
export type GroupSearchItem = GroupItem;
export type UserSearchItem = UserItem;
