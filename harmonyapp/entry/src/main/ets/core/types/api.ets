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

// ==================== 通用类型定义 ====================

export type UserType = 'temp' | 'normal' | 'admin' | 'system';
export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file';
export type MessageType = SendMessageType | 'system';
export type ReactionType = 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';

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

/**
 * 群组设置
 */
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

/**
 * 用户关系信息
 */
export interface UserRelationship {
  is_mutual_follow: boolean;
  is_following: boolean;
  is_followed_by: boolean;
  is_blocked: boolean;
  is_blocking: boolean;
}

// ==================== 认证API类型 ====================

/**
 * 通用认证数据类型
 */
export interface AuthData {
  user_id: number;
  nickname: string;
  access_token: string;
  // Web 场景通常不在 body 返回（使用 HttpOnly Cookie 下发）
  refresh_token?: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

// Web / App 分离（与 @api_spec.mdc 一致）
export type WebAuthData = Omit<AuthData, 'refresh_token'>;
export type AppAuthData = AuthData & { refresh_token: string };

/**
 * 用户注册请求
 */
export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  client_type?: 'web' | 'app';
}

/**
 * 用户登录请求
 */
export interface LoginRequest {
  login_id: string;
  password: string;
  client_type?: 'web' | 'app';
}

/**
 * 创建临时用户请求
 */
export interface CreateTempRequest {
  nickname: string;
  client_type?: 'web' | 'app';
}

/**
 * 刷新Token请求
 */
export interface RefreshTokenRequest {
  // app：在 body 传递；web：可不传（从 HttpOnly cookie 读取）
  refresh_token?: string;
  client_type?: 'web' | 'app';
}

/**
 * 检查认证状态数据
 */
export interface CheckStatusData {
  user_id: number;
  nickname: string;
  is_valid: boolean;
  expires_at: string;
}

/**
 * 请求密码重置请求
 */
export interface RequestPasswordResetRequest {
  login_id: string;
}

export interface RequestPasswordResetData {
  reset_id: string;
}

/**
 * 验证重置码请求
 */
export interface VerifyResetCodeRequest {
  reset_id: string;
  code: string;
}

export interface VerifyResetCodeData {
  reset_token: string;
}

/**
 * 重置密码请求
 */
export interface ResetPasswordRequest {
  reset_token: string;
  password: string;
}

/**
 * 修改密码请求
 */
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

/**
 * 获取活跃会话列表请求
 */
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

/**
 * 撤销会话请求
 */
export interface RevokeSessionRequest {
  id: string;
}

// ==================== 用户API类型 ====================

/**
 * 用户搜索结果项
 */
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

/**
 * 按关键字搜索用户请求
 */
export interface SearchUsersByKeywordRequest extends SearchBaseParams {
  kw: string;
}

/**
 * 按标签搜索用户请求
 */
export interface SearchUsersByTagRequest extends SearchBaseParams {
  tag: string;
}

/**
 * 按位置搜索用户请求
 */
export interface SearchUsersByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 获取用户资料请求
 */
export interface UserOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
}

/**
 * 获取用户关系请求
 */
export interface GetRelationshipRequest {
  id: string;
}

/**
 * 获取用户统计请求
 */
export interface GetUserStatsRequest {
  id: string;
}

export interface UserStatsData {
  moments_count: number;
  friends_count: number;
  followers_count: number;
  following_count: number;
  groups_count: number;
}

/**
 * 当前用户资料数据
 */
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

/**
 * 更新用户资料请求
 */
export interface UpdateProfileRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
}

/**
 * 更新用户位置请求
 */
export interface UpdateLocationRequest {
  latitude: number;
  longitude: number;
  location_name?: string;
}

/**
 * 账户升级请求
 */
export interface UpgradeAccountRequest {
  login_id: string;
  password: string;
}

/**
 * 用户设置数据
 */
export interface UserSettingsData {
  privacy_level: 'public' | 'friends' | 'private';
  location_visible: boolean;
  online_visible: boolean;
  last_online_visible: boolean;
  friends_visible: 'public' | 'friends' | 'private';
  following_visible: 'public' | 'friends' | 'private';
  followers_visible: 'public' | 'friends' | 'private';
  moments_visible: 'public' | 'friends' | 'private';
  stats_visible: 'public' | 'friends' | 'private';
  allow_follow: boolean;
  allow_private_chat: 'everyone' | 'friends' | 'nobody';
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
}

/**
 * 更新用户设置请求
 */
export interface UpdateSettingsRequest {
  privacy_level?: 'public' | 'friends' | 'private';
  location_visible?: boolean;
  online_visible?: boolean;
  last_online_visible?: boolean;
  friends_visible?: 'public' | 'friends' | 'private';
  following_visible?: 'public' | 'friends' | 'private';
  followers_visible?: 'public' | 'friends' | 'private';
  moments_visible?: 'public' | 'friends' | 'private';
  stats_visible?: 'public' | 'friends' | 'private';
  allow_follow?: boolean;
  allow_private_chat?: 'everyone' | 'friends' | 'nobody';
  searchable?: boolean;
  phone_searchable?: boolean;
  nickname_searchable?: boolean;
  location_searchable?: boolean;
}

/**
 * 删除账户请求
 */
export interface DeleteAccountRequest {
  confirmation: string;
}

/**
 * 关注用户请求
 */
export interface FollowUserRequest {
  id: string;
}

/**
 * 取消关注用户请求
 */
export interface UnfollowUserRequest {
  id: string;
}

/**
 * 拉黑用户请求
 */
export interface BlockUserRequest {
  id: string;
  reason?: string;
}

/**
 * 解除拉黑用户请求
 */
export interface UnblockUserRequest {
  id: string;
}

/**
 * 好友列表项
 */
export interface FriendListItem {
  user: UserInfo;
  is_online?: boolean;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

/**
 * 关注列表项
 */
export interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
}

/**
 * 粉丝列表项
 */
export interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
}

/**
 * 拉黑列表项
 */
export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

/**
 * 设置好友备注请求
 */
export interface SetFriendRemarkRequest {
  id: string;
  remark?: string;
}

/**
 * 获取用户好友列表请求
 */
export interface GetUserFriendsListRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 获取用户关注列表请求
 */
export interface GetUserFollowingListRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 获取用户粉丝列表请求
 */
export interface GetUserFollowersListRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 获取用户加入的公开群组请求
 */
export interface GetUserGroupsListRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 获取用户点赞的瞬间请求
 */
export interface GetUserLikedMomentsListRequest {
  id: string;
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string;
}

// ==================== 生活瞬间API类型 ====================

/**
 * 生活瞬间搜索结果项
 */
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

/**
 * 按关键字搜索瞬间请求
 */
export interface SearchMomentsByKeywordRequest extends SearchBaseParams {
  kw: string;
}

/**
 * 按标签搜索瞬间请求
 */
export interface SearchMomentsByTagRequest extends SearchBaseParams {
  tag: string;
}

/**
 * 按位置搜索瞬间请求
 */
export interface SearchMomentsByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 获取瞬间详情请求
 */
export interface MomentOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
}

/**
 * 获取用户瞬间列表请求
 */
export interface GetUserMomentsRequest extends SearchBaseParams {
  id: number;
}

/**
 * 创建瞬间请求
 */
export interface CreateMomentRequest {
  description: string;
  location: LocationInfo;
  tags?: string[];
  image_file_ids?: string[];
}

export interface CreateMomentData {
  id: string;
}

/**
 * 编辑瞬间请求
 */
export interface EditMomentRequest {
  id: string;
  description?: string;
  image_file_ids?: string[];
  tags?: string[];
  location?: LocationInfo;
}

/**
 * 删除瞬间请求
 */
export interface DeleteMomentRequest {
  id: string;
}

/**
 * 点赞瞬间请求
 */
export interface LikeMomentRequest {
  id: string;
}

/**
 * 取消点赞瞬间请求
 */
export interface UnlikeMomentRequest {
  id: string;
}

/**
 * 收藏瞬间请求
 */
export interface FavoriteMomentRequest {
  id: string;
}

/**
 * 取消收藏瞬间请求
 */
export interface UnfavoriteMomentRequest {
  id: string;
}

/**
 * 检查瞬间是否已点赞请求
 */
export interface CheckMomentLikedRequest {
  id: string;
}

export interface CheckLikedData {
  is_liked: boolean;
}

/**
 * 检查瞬间是否已收藏请求
 */
export interface CheckMomentFavoritedRequest {
  id: string;
}

export interface CheckFavoritedData {
  is_favorited: boolean;
}

/**
 * 获取瞬间点赞列表请求
 */
export interface GetMomentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface MomentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ==================== 评论API类型 ====================

/**
 * 添加评论请求
 */
export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
}

export interface CreateCommentData {
  id: string;
}

/**
 * 删除评论请求
 */
export interface DeleteCommentRequest {
  id: string;
}

/**
 * 获取评论列表请求
 */
export interface GetCommentsRequest {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 评论列表项
 */
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

/**
 * 获取子评论请求
 */
export interface GetRepliesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

/**
 * 点赞评论请求
 */
export interface LikeCommentRequest {
  id: string;
}

/**
 * 取消点赞评论请求
 */
export interface UnlikeCommentRequest {
  id: string;
}

/**
 * 获取评论点赞列表请求
 */
export interface GetCommentLikesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface CommentLikeItem {
  user: UserInfo;
  liked_at: string;
}

// ==================== 私聊管理API类型 ====================

/**
 * 发起私聊请求
 */
export interface CreatePrivateChatRequest {
  target_user_id: number;
  initial_message?: string;
}

export interface CreatePrivateChatData {
  id: string;
}

/**
 * 获取私聊列表请求
 */
export interface GetPrivateChatsRequest {
  limit?: number;
  cursor?: string;
  include_archived?: boolean;
}

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

/**
 * 获取私聊详情请求
 */
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

/**
 * 更新私聊设置请求
 */
export interface UpdatePrivateChatSettingsRequest {
  id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
  is_archived?: boolean;
}

/**
 * 删除私聊请求
 */
export interface DeletePrivateChatRequest {
  id: string;
}

// ==================== 群组管理API类型 ====================

/**
 * 群组搜索结果项
 */
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

/**
 * 按关键字搜索群组请求
 */
export interface SearchGroupsByKeywordRequest extends SearchBaseParams {
  kw: string;
}

/**
 * 按标签搜索群组请求
 */
export interface SearchGroupsByTagRequest extends SearchBaseParams {
  tag: string;
}

/**
 * 按位置搜索群组请求
 */
export interface SearchGroupsByLocationRequest extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 获取群组详情请求
 */
export interface GroupOpenTheDoorRequest {
  id: string;
  latitude: number;
  longitude: number;
}

/**
 * 获取群组设置请求
 */
export interface GetGroupSettingsRequest {
  id: string;
}

export interface GroupSettingsData {
  id: string;
  settings: GroupSettings;
  has_password: boolean;
  created_at: string;
}

/**
 * 创建群组请求
 */
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

/**
 * 更新群组请求
 */
export interface UpdateGroupChatRequest {
  id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  settings?: GroupSettings;
}

/**
 * 解散群组请求
 */
export interface DeleteGroupChatRequest {
  id: string;
}

/**
 * 获取成员列表请求
 */
export interface GetMembersRequest {
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

/**
 * 邀请成员请求
 */
export interface InviteMemberRequest {
  id: string;
  user_id: number;
  message?: string;
}

export interface InviteMemberData {
  id: string;
  expires_at: string;
}

/**
 * 加入群组请求
 */
export interface JoinGroupRequest {
  id: string;
  password?: string;
}

/**
 * 申请加入群组请求
 */
export interface ApplyToGroupRequest {
  id: string;
  message?: string;
}

export interface ApplyToGroupData {
  id: string;
}

/**
 * 设置成员角色请求
 */
export interface SetMemberRoleRequest {
  id: string;
  user_id: number;
  role: 'admin' | 'member';
}

/**
 * 禁言成员请求
 */
export interface MuteMemberRequest {
  id: string;
  user_id: number;
  duration_minutes: number;
}

/**
 * 取消禁言请求
 */
export interface UnmuteMemberRequest {
  id: string;
  user_id: number;
}

/**
 * 封禁成员请求
 */
export interface BanMemberRequest {
  id: string;
  user_id: number;
  reason?: string;
}

/**
 * 解除封禁请求
 */
export interface UnbanMemberRequest {
  id: string;
  user_id: number;
}

/**
 * 获取封禁列表请求
 */
export interface GetBannedMembersRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface BannedMemberItem {
  user: UserInfo;
  reason?: string;
  banned_at: string;
}

/**
 * 移除成员请求
 */
export interface RemoveMemberRequest {
  id: string;
  user_id: number;
}

/**
 * 获取群组申请列表请求
 */
export interface GetGroupApplicationsRequest {
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

/**
 * 审批群组申请请求
 */
export interface ApproveGroupApplicationRequest {
  application_id: string;
}

/**
 * 拒绝群组申请请求
 */
export interface RejectGroupApplicationRequest {
  application_id: string;
  reason?: string;
}

/**
 * 转让群主请求
 */
export interface TransferOwnershipRequest {
  id: string;
  new_owner_id: number;
}

/**
 * 获取我加入的群组请求
 */
export interface GetJoinedGroupChatsRequest {
  limit?: number;
  cursor?: string;
}

export interface JoinedGroupItem extends GroupInfo {
  my_role: 'owner' | 'admin' | 'member';
  unread_count: number;
  last_activity_at: string;
}

/**
 * 取消群组申请请求
 */
export interface CancelGroupApplicationRequest {
  application_id: string;
}

/**
 * 获取群组邀请列表请求
 */
export interface GetGroupInvitesRequest {
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

/**
 * 接受群组邀请请求
 */
export interface AcceptGroupInviteRequest {
  invite_id: string;
}

/**
 * 拒绝群组邀请请求
 */
export interface RejectGroupInviteRequest {
  invite_id: string;
}

/**
 * 退出群组请求
 */
export interface LeaveGroupRequest {
  id: string;
}

// ==================== 私聊消息API类型 ====================

/**
 * 发送私聊消息请求
 */
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

/**
 * 获取私聊消息历史请求
 */
export interface GetPrivateMessagesRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

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

/**
 * 编辑私聊消息请求
 */
export interface UpdatePrivateMessageRequest {
  id: string;
  content: string;
}

/**
 * 撤回私聊消息请求
 */
export interface RecallPrivateMessageRequest {
  id: string;
}

/**
 * 删除私聊消息请求
 */
export interface DeletePrivateMessageRequest {
  id: string;
}

/**
 * 标记私聊消息已读请求
 */
export interface MarkPrivateMessagesReadRequest {
  id: string;
  up_to?: string;
}

/**
 * 添加私聊消息反应请求
 */
export interface AddPrivateMessageReactionRequest {
  id: string;
  type: ReactionType;
}

/**
 * 移除私聊消息反应请求
 */
export interface RemovePrivateMessageReactionRequest {
  id: string;
  type: ReactionType;
}

/**
 * 获取私聊消息反应列表请求
 */
export interface GetPrivateMessageReactionsRequest {
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

// ==================== 群聊消息API类型 ====================

// 群聊消息类型包含系统消息，使用 ReceiveMessageType

/**
 * 发送群聊消息请求
 */
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

/**
 * 获取群聊消息历史请求
 */
export interface GetGroupMessagesRequest {
  id: string;
  limit?: number;
  cursor?: string;
  include_system?: boolean;
}

export interface GroupMessageItem {
  id: string;
  type: ReceiveMessageType;
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

/**
 * 编辑群聊消息请求
 */
export interface UpdateGroupMessageRequest {
  id: string;
  content: string;
}

/**
 * 撤回群聊消息请求
 */
export interface RecallGroupMessageRequest {
  id: string;
}

/**
 * 删除群聊消息请求
 */
export interface DeleteGroupMessageRequest {
  id: string;
  for_everyone?: boolean;
}

/**
 * 标记群聊消息已读请求
 */
export interface MarkGroupMessagesReadRequest {
  id: string;
  up_to?: string;
}

/**
 * 获取消息已读状态请求
 */
export interface GetMessageReadStatusRequest {
  id: string;
  limit?: number;
  cursor?: string;
}

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

/**
 * 添加群聊消息反应请求
 */
export interface AddGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

/**
 * 移除群聊消息反应请求
 */
export interface RemoveGroupMessageReactionRequest {
  id: string;
  type: ReactionType;
}

/**
 * 获取群聊消息反应列表请求
 */
export interface GetGroupMessageReactionsRequest {
  id: string;
  type?: ReactionType;
  limit?: number;
  cursor?: string;
}

// ==================== 举报审核API类型 ====================

/**
 * 创建举报请求
 */
export interface CreateReportRequest {
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group';
  target_id: string;
  reason: string;
  description?: string;
}

export interface CreateReportData {
  id: string;
}

/**
 * 获取我的举报记录请求
 */
export interface GetMyReportsRequest {
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

export type DevicePlatform = 'ios' | 'android' | 'harmony';

export interface RegisterDeviceTokenRequest {
  platform: DevicePlatform;
  device_token: string;
}

export interface UnregisterDeviceTokenRequest {
  platform: DevicePlatform;
  device_token: string;
}

export interface GetDeviceTokensRequest {
  limit?: number;
  cursor?: string;
}

export interface DeviceTokenItem {
  id: string;
  platform: DevicePlatform;
  device_token: string;
  created_at: string;
}

// ==================== 系统健康检查API类型 ====================

export interface HealthCheckData {
  status: 'healthy' | 'degraded' | 'unhealthy';
  version: string;
  timestamp: string;
}

// ==================== 文件上传API类型 ====================

export type UploadCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

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

export interface UploadChunkData {
  part_number: number;
  etag: string;
}

export interface UploadPartItem {
  part_number: number;
  etag: string;
}

export interface CompleteUploadRequest {
  upload_id: string;
  parts: UploadPartItem[];
}

export interface GetUploadStatusRequest {
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

// ==================== 信息流API类型 ====================

/**
 * 来源类型
 */
export type FeedMomentSource = 'following' | 'nearby' | 'all';

/**
 * 获取瞬间信息流请求
 */
export interface GetFeedMomentsRequest {
  source?: FeedMomentSource;
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedMomentItem extends MomentSearchItem {
  reason?: string;
}

/**
 * 获取群组推荐请求
 */
export interface GetFeedGroupsRequest {
  radius?: number;
  limit?: number;
  cursor?: string;
}

export interface FeedGroupItem extends GroupSearchItem {
  reason?: string;
  friends_in_group?: number;
}

/**
 * 获取用户推荐请求
 */
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

/**
 * 获取热门内容请求
 */
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

/**
 * 反馈信息流内容请求
 */
export interface FeedFeedbackRequest {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested' | 'report';
  reason?: string;
}

// ==================== 标签管理API类型 ====================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

export interface GetTrendingTagsRequest {
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
  items: string[];
}
