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
export type UserType = "temp" | "normal" | "admin" | "system";

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

export interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}

// 隐私级别枚举（用于各种隐私可见性设置）
export type PrivacyLevel = "public" | "friends" | "private";

// 消息反应类型枚举
export type ReactionType = "like" | "love" | "laugh" | "angry" | "sad" | "wow";

// 字段可见性（隐私控制）
export type FieldVisibility = "visible" | "hidden";
export type HiddenReason = "privacy" | "blocked" | "not_available";

export interface FieldVisibilityMeta {
  visibility: FieldVisibility;
  reason?: HiddenReason; // visibility='hidden' 时必须提供
}

// ============================================================================
// 地图聚合 / 视口查询（Public）
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
  type: "cluster" | "user" | "group" | "moment";
  location: GeoPoint;
  cluster?: MapClusterData;
  entity?: MapEntityData;
}

export interface MapRenderData {
  items: MapRenderItem[];
  toast_message?: string;
}

// 请求参考点（仅用于"以某点为参考进行距离/方位/范围查询"的输入坐标，不包含名称）
export interface GeoPoint {
  latitude: number;
  longitude: number;
}

// 搜索请求基础参数（所有搜索 API 必须包含参考点）
export interface SearchBaseParams {
  reference_point: GeoPoint;
  limit?: number;         // 每页数量，默认 20，最大 100
  cursor?: string;        // 分页游标
}

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

// 时间范围筛选参数
export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

// 群组设置
export interface GroupSettings {
  join_method?: "open" | "approval" | "invite_only";
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
  getUserTypeBadge,
  SYSTEM_USER_IDS,
  SYSTEM_USER_ID_MAX,
} from "~/utils/userTypes";

// ============================================================================
// 认证API类型
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

// Web / App 分离（与 @api_spec.mdc 一致）
export type WebAuthData = AuthTokenPairWeb;
export type AppAuthData = AuthTokenPairApp;

export type AuthData = AuthTokenPairWeb | AuthTokenPairApp;

export interface SendVerificationCodeRequest {
  phone: string;
  type: 'register' | 'reset_password' | 'change_phone';
  captcha_token?: string;
}

export interface SendVerificationCodeData {
  code_id: string;
  expires_at: string;
  retry_after?: number;
}

export interface VerifyCodeRequest {
  code_id: string;
  code: string;
}

export interface VerifyCodeData {
  verified: boolean;
  verify_token: string;
  expires_at: string;
}

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  // 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
  client_type: "web" | "app";
  verify_token?: string; // 可选：如果通过手机验证码注册，提供此 token
}

export interface LoginRequest {
  login_id: string;
  password: string;
  // 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
  client_type: "web" | "app";
}

export interface LogoutRequest {}

export interface GuestRequest {
  nickname: string;
  // 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
  client_type: "web" | "app";
}

export interface RefreshTokenRequest {
  // 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
  client_type: "web" | "app";
  // refresh_token 投递规则（强制）：
  // - app：必须在 body 传 refresh_token
  // - web：必须不传（从 HttpOnly cookie 读取），若传入则视为参数错误
  refresh_token?: string;
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
  verify_token?: string; // 可选：如果通过验证码验证，提供此 token
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

// ============================================================================
// 用户API类型
// ============================================================================

export interface UserItem extends UserInfo {
  bio?: string;
  tags?: string[];
  // 隐私字段：必须显式返回 value 或 null，并提供 meta（不再通过"省略字段"表达隐藏）
  is_online: boolean | null;
  is_online_meta: FieldVisibilityMeta;
  last_online: string | null;
  last_online_meta: FieldVisibilityMeta;
  location?: RelativeLocation | null;
  location_meta: FieldVisibilityMeta;
  
  relation?: UserRelation;
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

export interface UserDetailData extends UserItem {
  created_at?: string;
  stats?: UserStatsData;
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
  user_id: number;
  nickname: string;
  avatar_url?: string;
  bio?: string;
  tags?: string[];
  gender?: 'male' | 'female' | 'other';
  email?: string;
  phone?: string;
  location: Location; // Must be present
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
  moments_visible: PrivacyLevel;
  stats_visible: PrivacyLevel;
  allow_follow: boolean;
  allow_private_chat: "everyone" | "friends" | "nobody";
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
  allow_private_chat?: "everyone" | "friends" | "nobody";
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
  privacy_level: PrivacyLevel; // 可见范围
  mentions?: number[]; // @ 提及的用户 ID 列表
  // 互动状态：仅登录后返回（未登录时字段不存在）
  is_liked?: boolean;
  is_favorited?: boolean;
}

export type CreateMomentData = MomentItem;

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

export interface MomentCommentsRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

export interface UserMomentsRequest extends SearchBaseParams {
  user_id: number;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  tags?: string[];
  image_file_ids?: string[];
  privacy_level?: PrivacyLevel; // 可见范围（不提供时使用用户全局设置）
  mentions?: number[]; // @ 提及的用户 ID 列表（最多 10 个）
}

export interface UpdateMomentRequest {
  moment_id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: Location;
  privacy_level?: PrivacyLevel; // 可见范围
  mentions?: number[]; // @ 提及的用户 ID 列表（最多 10 个）
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

// ============================================================================
// 评论 API 类型
// ============================================================================

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: number[]; // @ 提及的用户 ID 列表（最多 10 个）
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
  parent_id?: string; // 仅子评论存在；顶层评论必须省略
  mentions?: number[]; // @ 提及的用户 ID 列表
  created_at: string;
  likes_count: number;
  replies_count: number;
}

export type CreateCommentData = CommentListItem;

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

// ============================================================================
// 私聊管理 API 类型
// ============================================================================

export interface MyPrivateChatsRequest {
  limit?: number;
  cursor?: string;
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

export interface MessageReplyTo {
  message_id: string;
  user_id: number;
  nickname: string;
  content: string;
}

export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'system' | 'location';
export type SendMessageType = Exclude<MessageType, 'system'>;

export interface PrivateChatLastMessage {
  message_id: string;
  content?: string;
  type: MessageType;
  sender_id: number;
  sent_at: string;
}

export interface PrivateChatOpenTheDoorRequest {
  target_user_id: number;
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

export interface MessageLimit {
  remaining: number;
  total: number;
  is_unlocked: boolean;
}

export interface UpdatePrivateChatSettingsRequest {
  target_user_id: number;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeletePrivateChatRequest {
  target_user_id: number;
}

// ============================================================================
// 群组管理 API 类型
// ============================================================================

export interface GroupItem extends GroupInfo {
  description?: string;
  tags?: string[];
  has_password: boolean;
  location?: RelativeLocation;
  my_state?: GroupMyState;
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
  group_chat_id: string;
  reference_point: GeoPoint;
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

export interface GroupSettingsRequest {
  group_chat_id: string;
}

export interface GroupSettingsData {
  group_chat_id: string;
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

export type CreateGroupChatData = JoinedGroupItem;

export interface UpdateGroupChatRequest {
  group_chat_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  settings?: GroupSettings;
}

export interface UpdateGroupMySettingsRequest {
  group_chat_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}

export interface DeleteGroupChatRequest {
  group_chat_id: string;
}

// ============================================================================
// 群组成员管理 API 类型
// ============================================================================

export interface GroupMembersRequest {
  group_chat_id: string;
  role?: "owner" | "admin" | "member";
  limit?: number;
  cursor?: string;
}

export interface GroupMemberItem {
  user: UserInfo;
  role: "owner" | "admin" | "member";
  joined_at: string;
  muted_until?: string;
  is_online: boolean;
}

export interface InviteMemberRequest {
  group_chat_id: string;
  user_id: number;
  message?: string;
}

export interface InviteMemberData {
  invite_id: string;
  expires_at: string;
}

export interface JoinGroupRequest {
  group_chat_id: string;
  password?: string;
}

export interface ApplyToGroupRequest {
  group_chat_id: string;
  message?: string;
}

export interface ApplyToGroupData {
  application_id: string;
}

export interface SetMemberRoleRequest {
  group_chat_id: string;
  user_id: number;
  role: "admin" | "member";
}

export interface MuteMemberRequest {
  group_chat_id: string;
  user_id: number;
  duration_minutes: number;
}

export interface UnmuteMemberRequest {
  group_chat_id: string;
  user_id: number;
}

export interface BanMemberRequest {
  group_chat_id: string;
  user_id: number;
  reason?: string;
}

export interface UnbanMemberRequest {
  group_chat_id: string;
  user_id: number;
}

export interface BannedMembersRequest {
  group_chat_id: string;
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
  group_chat_id: string;
  user_id: number;
}

export interface GroupApplicationsRequest {
  group_chat_id: string;
  status?: "pending" | "accepted" | "rejected" | "all";
  limit?: number;
  cursor?: string;
}

export interface GroupApplicationItem {
  application_id: string;
  applicant: UserInfo;
  message?: string;
  status: "pending" | "accepted" | "rejected";
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
  group_chat_id: string;
  new_owner_id: number;
}

export interface MyGroupChatsRequest {
  limit?: number;
  cursor?: string;
}

export interface JoinedGroupItem extends GroupItem {
  my_state: GroupMyState;
}

export interface CancelApplicationRequest {
  application_id: string;
}

export interface MyInvitesRequest {
  status?: "pending" | "accepted" | "rejected" | "all";
  limit?: number;
  cursor?: string;
}

export interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: "pending" | "accepted" | "rejected";
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
  group_chat_id: string;
}

// ============================================================================
// 群公告 API 类型
// ============================================================================

export interface GroupAnnouncementItem {
  announcement_id: string;
  group_chat_id: string;
  title: string;
  content: string;
  created_by: UserInfo;
  created_at: string;
  is_pinned: boolean;
}

export interface CreateGroupAnnouncementRequest {
  group_chat_id: string;
  title: string;
  content: string;
  is_pinned?: boolean;
}

export interface ListGroupAnnouncementsRequest {
  group_chat_id: string;
  limit?: number;
  cursor?: string;
}

export interface UpdateGroupAnnouncementRequest {
  announcement_id: string;
  title?: string;
  content?: string;
  is_pinned?: boolean;
}

export interface DeleteGroupAnnouncementRequest {
  announcement_id: string;
}

// ============================================================================
// 私聊消息 API 类型
// ============================================================================

export interface SendPrivateMessageRequest {
  target_user_id: number;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  location?: Location; // 位置消息信息（当 type='location' 时必填）
  location_address?: string; // 位置消息详细地址（可选）
}

export interface PrivateMessagesHistoryRequest {
  target_user_id: number;
  limit?: number;
  cursor?: string;
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
  address?: string; // 详细地址（可选）
}

export interface PrivateMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  media?: MessageMedia;
  location?: Location; // 位置消息
  location_address?: string; // 位置消息详细地址
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_at?: string;
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

export interface MessageReactionItem {
  user: UserInfo;
  type: ReactionType;
  created_at: string;
}

// ============================================================================
// 群聊消息 API 类型
// ============================================================================

export interface SendGroupMessageRequest {
  group_chat_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
  mentions?: number[];
  location?: Location; // 位置消息信息（当 type='location' 时必填）
  location_address?: string; // 位置消息详细地址（可选）
}

export interface GroupMessagesHistoryRequest {
  group_chat_id: string;
  limit?: number;
  cursor?: string;
  include_system?: boolean;
}

export interface GroupMessageItem {
  message_id: string;
  type: MessageType;
  content?: string;
  sender: UserInfo;
  reply_to_message_id?: string;
  reply_to?: MessageReplyTo;
  mentions?: number[];
  media?: MessageMedia;
  location?: Location; // 位置消息
  location_address?: string; // 位置消息详细地址
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_by?: number;
  recalled_at?: string;
  read_count?: number;
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
  group_chat_id: string;
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

// ============================================================================
// 消息通用 API 类型
// ============================================================================

export interface MarkAllMessagesReadRequest {}

// ============================================================================
// 举报审核 API 类型
// ============================================================================

export interface CreateReportRequest {
  target_type: "user" | "moment" | "comment" | "message" | "group";
  target_id: string;
  reason: string;
  description?: string;
}

export interface CreateReportData {
  report_id: string;
}

export interface MyReportsRequest {
  limit?: number;
  cursor?: string;
}

export interface ReportItem {
  report_id: string;
  target_type: "user" | "moment" | "comment" | "message" | "group";
  target_id: string;
  reason: string;
  description?: string;
  status: "pending" | "reviewing" | "resolved" | "dismissed";
  created_at: string;
  handled_at?: string;
  target_snapshot?: {
    content?: string;
    media_urls?: string[];
    owner_id?: string;
  };
}

export interface ReportsRequest {
  status?: "pending" | "reviewing" | "resolved" | "dismissed";
  target_type?: "user" | "moment" | "comment" | "message" | "group";
  limit?: number;
  cursor?: string;
}

export interface HandleReportRequest {
  report_id: string;
  action: "resolve" | "dismiss" | "escalate";
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
  platform: "ios" | "android" | "harmony" | "web";
  device_token: string;
}

export interface UnregisterDeviceTokenRequest {
  platform: "ios" | "android" | "harmony" | "web";
  device_token: string;
}

export interface MyDeviceTokensRequest {
  limit?: number;
  cursor?: string;
}

export interface DeviceTokenItem {
  device_token_id: string;
  platform: "ios" | "android" | "harmony" | "web";
  device_token: string;
  created_at: string;
}

// ============================================================================
// 系统健康检查 API 类型
// ============================================================================

export interface HealthCheckRequest {}

export interface HealthCheckData {
  status: "healthy" | "degraded" | "unhealthy";
  version: string;
  timestamp: string;
}

// ============================================================================
// 文件与媒体上传 API 类型
// ============================================================================

export interface UploadMediaRequest {
  file: File;
  category:
    | "avatar"
    | "group_avatar"
    | "moment_image"
    | "chat_image"
    | "chat_video"
    | "chat_audio"
    | "chat_file";
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

export interface CompleteUploadRequest {
  upload_id: string;
}

// Chunked Upload
export interface InitChunkedUploadRequest {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: string;
  chunk_count: number;
}

export interface InitChunkedUploadData {
  upload_id: string;
  chunk_size: number;
  chunk_count: number;
  upload_urls: string[]; // 预生成的上传 URL 列表
  expires_at: string;
}

export interface UploadChunkRequest {
  upload_id: string;
  part_number: number;
}

export interface UploadChunkData {
  part_number: number;
  etag: string;
}

export interface FinishChunkedUploadRequest {
  upload_id: string;
}

export interface CheckChunkedUploadRequest {
  upload_id: string;
}

export interface CheckChunkedUploadData {
  upload_id: string;
  chunk_size: number;
  uploaded_chunks: number[];
}

// ============================================================================
// 信息流 API 类型
// ============================================================================

export interface FeedMomentsRequest {
  source?: "following" | "nearby" | "all";
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupItem extends GroupItem {
  reason?: string;
  friends_in_group?: number;
  my_role?: 'owner' | 'admin' | 'member';
  is_joined?: boolean;
}

export interface FeedUsersRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedUserItem extends UserItem {
  reason?: string;
  mutual_friends_count?: number;
  common_tags?: string[];
}

export interface FeedTrendingRequest {
  limit?: number;
}

export interface FeedTrendingData {
  trending_tags: TrendingTagItem[];
  hot_moments: MomentItem[];
}

export interface TrendingTagItem {
  name: string;
  usage_count: number;
  trend: "rising" | "stable" | "falling";
  hot_moment_preview?: MomentItem;
}

export interface FeedFeedbackRequest {
  content_type: "moment" | "group" | "user";
  content_id: string;
  action: "hide" | "not_interested" | "report";
  reason?: string;
}

// ============================================================================
// 标签管理 API 类型
// ============================================================================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: "rising" | "stable" | "falling";
}

export interface TrendingTagsRequest {
  limit?: number;
  timeframe?: "daily" | "weekly" | "monthly";
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

export interface ClientContext {
  os: "ios" | "android" | "web" | "harmony";
  app_version: string;
  device_id?: string;
  language?: string;
}

export interface FetchGlobalSettingsRequest {
  context: ClientContext;
  current_version?: string;
}

export interface FetchSettingsData {
  action: "update" | "no_change" | "force_update";
  config?: GlobalSettings;
  latest_version: string;
}

export interface AdminGetGlobalSettingsRequest {}

export interface AdminGlobalSettingsData {
  settings: GlobalSettings;
  updated_at: string;
  updated_by_user_id: number;
}

export interface UpdateUploadConfigRequest {
  image_max_size?: number;
  video_max_size?: number;
  audio_max_size?: number;
  file_max_size?: number;
  allowed_image_types?: string[];
  allowed_video_types?: string[];
}

export interface UpdateDefaultsRequest {
  default_location_name?: string;
  default_group_avatar?: string;
  default_user_avatar?: string;
  default_group_settings?: Partial<GroupSettings>;
}

export interface UpdateFeaturesRequest {
  enable_public_registration?: boolean;
  enable_guest_login?: boolean;
  enable_moments?: boolean;
  enable_nearby_people?: boolean;
}

export interface UpdateContactRequest {
  support_email?: string;
  privacy_policy_url?: string;
  terms_of_service_url?: string;
  official_website?: string;
}

export interface UpdateLimitsRequest {
  max_group_members?: number;
  max_moment_tags?: number;
  max_moment_images?: number;
  moment_description_max_length?: number;
  comment_max_length?: number;
  search_radius_max?: number;
}

// ============================================================================
// 管理员 RBAC API 类型
// ============================================================================

export interface AdminRoleBasic {
  role_id: string;
  name: string;
}

export interface AdminMeData {
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
  role_id: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface AdminRolesData {
  items: AdminRoleItem[];
}

export interface AdminAdminItem {
  user: UserInfo;
  // '*' 表示拥有全部权限（超级管理员）
  permissions: string[];
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
  role_id: string;
  name?: string;
  description?: string;
  clear_description?: boolean;
}

export interface AdminRbacRoleDeleteRequest {
  role_id: string;
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

// ============================================================================
// 关注请求 API 类型
// ============================================================================

export interface FollowRequestItem {
  request_id: string;
  user: UserInfo;
  message?: string;
  created_at: string;
}

export interface FollowRequestsRequest {
  limit?: number;
  cursor?: string;
}

export interface HandleFollowRequestRequest {
  request_id: string;
  action: 'accept' | 'reject';
}

// ============================================================================
// 管理员用户管理 API 类型
// ============================================================================

export interface AdminUserSearchRequest {
  kw?: string;
  status?: 'normal' | 'banned';
  user_type?: UserType;
  limit?: number;
  cursor?: string;
}

export interface AdminUserDetailRequest {
  user_id: number;
}

export interface AdminUserItem {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
  status: 'normal' | 'banned';
  email?: string;
  phone?: string;
  created_at: string;
  last_login_at?: string;
  moments_count: number;
  groups_count: number;
}

export interface AdminUserDetailData extends AdminUserItem {
  bio?: string;
  tags?: string[];
  banned_at?: string;
  banned_reason?: string;
  banned_by?: UserInfo;
}

export interface AdminBanUserRequest {
  user_id: number;
  reason?: string;
}

export interface AdminUnbanUserRequest {
  user_id: number;
}

// ============================================================================
// 管理员群组管理 API 类型
// ============================================================================

export interface AdminGroupSearchRequest {
  kw?: string;
  limit?: number;
  cursor?: string;
}

export interface AdminGroupDetailRequest {
  group_chat_id: string;
}

export interface AdminGroupItem {
  group_chat_id: string;
  name: string;
  avatar_url?: string;
  description?: string;
  owner_id: number;
  owner?: UserInfo;
  members_count: number;
  created_at: string;
}

export interface AdminGroupDetailData extends AdminGroupItem {
  tags?: string[];
  settings?: GroupSettings;
}

export interface AdminDismissGroupRequest {
  group_chat_id: string;
  reason?: string;
}

// ============================================================================
// 系统公告 API 类型
// ============================================================================

export interface SystemAnnouncementItem {
  announcement_id: string;
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  action_url?: string;
  created_at: string;
  created_by: UserInfo;
}

export interface PublicAnnouncementListRequest {
  limit?: number;
  cursor?: string;
}

export interface AdminCreateAnnouncementRequest {
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'error';
  action_url?: string;
}

export interface AdminAnnouncementListRequest {
  limit?: number;
  cursor?: string;
}

export interface AdminDeleteAnnouncementRequest {
  announcement_id: string;
}

// ============================================================================
// 审计日志 API 类型
// ============================================================================

export type AuditAction =
  | 'user_ban'
  | 'user_unban'
  | 'group_dismiss'
  | 'report_handle'
  | 'announcement_create'
  | 'announcement_delete'
  | 'admin_grant'
  | 'admin_revoke'
  | 'role_create'
  | 'role_update'
  | 'role_delete'
  | 'config_update';

export interface AuditLogItem {
  audit_id: string;
  action: AuditAction;
  operator: UserInfo;
  target_type?: 'user' | 'group' | 'announcement' | 'report' | 'role' | 'config';
  target_id?: string;
  details?: Record<string, unknown>;
  created_at: string;
  ip_address?: string;
}

export interface AdminAuditListRequest {
  action?: AuditAction;
  operator_id?: number;
  target_type?: 'user' | 'group' | 'announcement' | 'report' | 'role' | 'config';
  target_id?: string;
  start_time?: string;
  end_time?: string;
  limit?: number;
  cursor?: string;
}

export interface AdminAuditDetailRequest {
  audit_id: string;
}

export interface AdminAuditExportRequest {
  action?: AuditAction;
  operator_id?: number;
  target_type?: 'user' | 'group' | 'announcement' | 'report' | 'role' | 'config';
  start_time?: string;
  end_time?: string;
  format?: 'csv' | 'json';
}

export interface AdminAuditExportData {
  task_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  download_url?: string;
  expires_at?: string;
}

export interface AdminAuditExportStatusRequest {
  task_id: string;
}
