/**
 * 后端API类型定义
 * 严格按照后端文档 backend_doc.mdc 定义所有API的请求和响应类型
 * 遵循命名规范：XxxRequest 和 XxxResponse
 */

// ==================== 基础类型 ====================

// 统一响应格式
export interface ApiResponse<T> {
  code: number;        // 业务状态码：0-成功，非0-错误
  msg: string;         // 响应消息
  data: T | null;      // 成功时为具体业务数据；失败时为 null
}

// 分页响应格式
export interface PaginatedResponse<T> {
  items: T[];               // 数据项数组
  has_more: boolean;        // 是否有更多数据
  next_cursor?: string;     // 下一页游标（64位整数转字符串）
}

// 基础成功响应
export interface BasicSuccess {
  success: boolean;         // 操作是否成功
}

// 带消息的成功响应
export interface SuccessWithMessage {
  success: boolean;         // 操作是否成功
  message: string;          // 响应消息
}

// 未读计数模型
export interface UnreadCount {
  group_id?: string;        // 群组ID（UUID转字符串）
  conversation_id?: string; // 会话ID（UUID转字符串）
  unread_count?: number;    // 具体未读数（32位整数）
  total_unread_count?: number; // 汇总未读数（32位整数）
}

// ==================== 认证API类型 ====================

// 1. POST /api/v1/public/auth/get_nonce - 获取认证随机数
export interface GetNonceRequest {
  // 无请求参数
}

export interface GetNonceResponse {
  nonce_id: string;         // nonce标识符
  nonce: string;            // 随机数
  timestamp: number;        // 时间戳
  expires_in: number;       // 有效期（秒）
}

// 通用认证响应类型
export interface AuthResponse {
  user_id: string;          // 64位整数转字符串（高8位编码用户类型）
  nickname: string;         // 用户昵称
  access_token: string;     // 访问令牌
  refresh_token: string;    // 刷新令牌
  expires_at: string;       // 过期时间（ISO 8601）
}

// 2. POST /api/v1/public/auth/register - 用户注册
export interface RegisterRequest {
  login_id: string;         // 登录ID（邮箱/手机号）
  nickname: string;         // 用户昵称
  password_hash: string;    // SHA256(password + nonce + timestamp)
  nonce_id: string;         // nonce标识符
  timestamp: number;        // 时间戳
  latitude?: number;        // 纬度（可选）
  longitude?: number;       // 经度（可选）
}

// 用户注册响应 - 专门的API响应别名
export type RegisterResponse = AuthResponse;

// 3. POST /api/v1/public/auth/login - 用户登录
export interface LoginRequest {
  login_id: string;         // 登录ID（邮箱/手机号）
  password_hash: string;    // SHA256(password + nonce + timestamp)
  nonce_id: string;         // nonce标识符
  timestamp: number;        // 时间戳
}

// 用户登录响应 - 专门的API响应别名
export type LoginResponse = AuthResponse;

// 4. POST /api/v1/public/auth/create_temp - 创建临时用户
export interface CreateTempRequest {
  nickname: string;         // 用户昵称
  latitude?: number;        // 纬度（可选）
  longitude?: number;       // 经度（可选）
}

// 创建临时用户响应 - 专门的API响应别名
export type CreateTempResponse = AuthResponse;

// 5. POST /api/v1/public/auth/refresh_token - 刷新Token
export interface RefreshTokenRequest {
  refresh_token: string;    // 刷新令牌
  latitude?: number;        // 纬度（可选）
  longitude?: number;       // 经度（可选）
}

// 刷新Token响应 - 专门的API响应别名
export type RefreshTokenResponse = AuthResponse;

// 6. POST /api/v1/public/auth/check_status - 检查认证状态
export interface CheckStatusRequest {
  // 无请求参数，通过JWT token验证
}

export interface CheckStatusResponse {
  user_id: string;          // 64位整数转字符串（高8位编码用户类型）
  nickname: string;         // 用户昵称
  is_valid: boolean;        // token是否有效
  expires_at: string;       // 过期时间（ISO 8601）
}

// 7. POST /api/v1/protected/auth/set_password - 设置密码
export interface SetPasswordRequest {
  password_hash: string;    // SHA256(password + nonce + timestamp)
  nonce_id: string;         // nonce标识符
  timestamp: number;        // 时间戳
}

export interface SetPasswordResponse {
  success: boolean;         // 是否设置成功
  message: string;          // 响应消息
  can_login: boolean;       // 是否可以登录
}

// ==================== 用户API类型 ====================

// 通用用户信息响应类型
export interface UserResponse {
  user_id: string;          // 64位整数转字符串
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 头像URL
  bio?: string;             // 个人简介
  created_at?: string;      // 创建时间（ISO 8601）
  last_active_at?: string;  // 最后活跃时间（ISO 8601）
  latitude?: number;        // 纬度
  longitude?: number;       // 经度
  location_name?: string;   // 位置名称
  distance?: number;        // 距离（米）
}

// 1. POST /api/v1/public/user/search_by_name - 按昵称搜索用户
export interface SearchUserByNameRequest {
  name: string;             // 搜索关键词
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 按昵称搜索用户响应 - 专门的API响应别名
export type SearchUserByNameResponse = UserResponse;

// 2. POST /api/v1/public/user/search_by_id - 获取用户资料
export interface SearchUserByIdRequest {
  user_id: string;          // 用户ID
}

// 按ID获取用户资料响应 - 专门的API响应别名
export type SearchUserByIdResponse = UserResponse;

// 3. POST /api/v1/public/user/search_by_location - 搜索附近用户
export interface SearchUserByLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  radius: number;           // 搜索半径（米）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 搜索附近用户响应 - 扩展UserResponse并要求distance字段必填
export interface SearchUserByLocationResponse extends UserResponse {
  distance: number;         // 距离（米，必填）
}

// 4. POST /api/v1/protected/user/update_location - 更新用户位置
export interface UpdateLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name?: string;   // 位置名称
}

// 更新位置响应 - 专门的API响应别名
export type UpdateLocationResponse = SuccessWithMessage;

// 5. POST /api/v1/protected/user/update_profile - 更新用户资料
export interface UpdateProfileRequest {
  nickname?: string;        // 用户昵称
  avatar_url?: string;      // 头像URL
  bio?: string;             // 个人简介
}

// 更新用户资料响应 - 专门的API响应别名
export type UpdateProfileResponse = SuccessWithMessage;

// 6. POST /api/v1/protected/user/upgrade_account - 账户升级
export interface UpgradeAccountRequest {
  login_id: string;         // 新的登录ID（邮箱/手机号）
  password_hash: string;    // SHA256(password + nonce + timestamp)
  nonce_id: string;         // nonce标识符
  timestamp: number;        // 时间戳
}

// 账户升级响应 - 专门的API响应别名
export type UpgradeAccountResponse = AuthResponse;

// 7. POST /api/v1/protected/user/get_settings - 获取用户设置
export interface GetSettingsRequest {
  // 无请求参数，通过JWT token验证
}

export interface GetSettingsResponse {
  privacy_level: string;    // 隐私级别：'public'|'friends'|'private'
  location_sharing: boolean; // 位置分享开关
  notification_enabled: boolean; // 通知开关
  mention_notifications: boolean; // 提及通知开关
  group_invite_notifications: boolean; // 群组邀请通知开关
  moment_notifications: boolean; // 生活瞬间通知开关
}

// 8. POST /api/v1/protected/user/update_settings - 更新用户设置
export interface UpdateSettingsRequest {
  privacy_level?: string;   // 隐私级别：'public'|'friends'|'private'
  location_sharing?: boolean; // 位置分享开关
  notification_enabled?: boolean; // 通知开关
  mention_notifications?: boolean; // 提及通知开关
  group_invite_notifications?: boolean; // 群组邀请通知开关
  moment_notifications?: boolean; // 生活瞬间通知开关
}

// 更新用户设置响应 - 专门的API响应别名
export type UpdateSettingsResponse = SuccessWithMessage;

// 9. POST /api/v1/protected/user/delete_account - 删除账户
export interface DeleteAccountRequest {
  confirmation: string;     // 确认字符串，必须为 "DELETE_MY_ACCOUNT"
}

// 删除账户响应 - 专门的API响应别名
export type DeleteAccountResponse = SuccessWithMessage;

// ==================== 用户关系管理API类型 ====================

// 1. POST /api/v1/protected/user/add_friend - 发送好友申请
export interface AddFriendRequest {
  user_id: string;          // 目标用户ID
  message?: string;         // 申请消息（可选）
}

export interface AddFriendResponse {
  success: boolean;         // 是否发送成功
  message: string;          // 响应消息
  request_id: string;       // 好友申请ID
}

// 2. POST /api/v1/protected/user/accept_friend - 接受好友申请
export interface AcceptFriendRequest {
  request_id: string;       // 好友申请ID
}

export interface AcceptFriendResponse {
  success: boolean;         // 是否接受成功
  message: string;          // 响应消息
  friend_id: string;        // 新好友的用户ID
}

// 3. POST /api/v1/protected/user/reject_friend - 拒绝好友申请
export interface RejectFriendRequest {
  request_id: string;       // 好友申请ID
}

export interface RejectFriendResponse {
  success: boolean;         // 是否拒绝成功
  message: string;          // 响应消息
}

// 4. POST /api/v1/protected/user/follow - 关注用户
export interface FollowUserRequest {
  user_id: string;          // 目标用户ID
}

export interface FollowUserResponse {
  success: boolean;         // 是否关注成功
  message: string;          // 响应消息
  following_count: number;  // 当前关注数量
}

// 5. POST /api/v1/protected/user/unfollow - 取消关注
export interface UnfollowUserRequest {
  user_id: string;          // 目标用户ID
}

export interface UnfollowUserResponse {
  success: boolean;         // 是否取消关注成功
  message: string;          // 响应消息
  following_count: number;  // 当前关注数量
}

// 6. POST /api/v1/protected/user/block - 拉黑用户
export interface BlockUserRequest {
  user_id: string;          // 目标用户ID
  reason?: string;          // 拉黑原因（可选）
}

export interface BlockUserResponse {
  success: boolean;         // 是否拉黑成功
  message: string;          // 响应消息
  blocked_at: string;       // 拉黑时间（ISO 8601）
}

// 7. POST /api/v1/protected/user/get_friends - 获取好友列表
export interface GetFriendsListRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetFriendsListResponse {
  user_id: string;          // 好友用户ID
  nickname: string;         // 好友昵称
  avatar_url?: string;      // 好友头像URL
  bio?: string;             // 好友简介
  is_online: boolean;       // 是否在线
  last_seen?: string;       // 最后在线时间（ISO 8601）
  became_friends_at: string; // 成为好友时间（ISO 8601）
}

// 8. POST /api/v1/protected/user/get_blocked - 获取拉黑列表
export interface GetBlockedListRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetBlockedListResponse {
  user_id: string;          // 被拉黑用户ID
  nickname: string;         // 被拉黑用户昵称
  avatar_url?: string;      // 被拉黑用户头像URL
  blocked_at: string;       // 拉黑时间（ISO 8601）
  reason?: string;          // 拉黑原因
}

// ==================== 群组管理API类型 ====================

// 通用群组信息响应类型
export interface GroupResponse {
  group_id: string;         // UUID转字符串
  name: string;             // 群组名称
  description?: string;     // 群组描述
  owner_id: string;         // 群主ID（64位整数转字符串）
  creator_name: string;     // 创建者昵称
  created_at: string;       // 创建时间（ISO 8601）
  last_active_at: string;   // 最后活跃时间（ISO 8601）
  latitude: number;         // 纬度
  longitude: number;        // 经度
  member_count: number;     // 成员数量（32位整数）
  distance?: number;        // 距离（米）
  location_name: string;    // 位置名称
  is_password_required: boolean; // 是否需要密码
  is_member: boolean;       // 当前用户是否为成员
  user_role?: string;       // 当前用户在群组中的角色（owner/admin/member，仅当is_member为true时有值）
  avatar_url?: string;      // 群组头像URL
}

// 1. POST /api/v1/public/group/search_by_location - 按位置搜索群组
export interface SearchGroupByLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  radius: number;           // 搜索半径（米）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 按位置搜索群组响应 - 扩展GroupResponse并要求distance字段必填
export interface SearchGroupByLocationResponse extends GroupResponse {
  distance: number;         // 距离（米，必填）
}

// 2. POST /api/v1/public/group/search_by_name - 按名称搜索群组
export interface SearchGroupByNameRequest {
  name: string;             // 搜索关键词
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 按名称搜索群组响应 - 专门的API响应别名
export type SearchGroupByNameResponse = GroupResponse;

// 3. POST /api/v1/public/group/search_by_id - 按ID获取群组详情
export interface SearchGroupByIdRequest {
  group_id: string;         // 群组ID
}

// 按ID获取群组详情响应 - 专门的API响应别名
export type SearchGroupByIdResponse = GroupResponse;

// 4. POST /api/v1/public/group/members - 获取群组成员
export interface GetMembersRequest {
  group_id: string;         // 群组ID
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetMembersResponse {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 用户头像URL
  role: string;             // 用户角色：'owner'|'admin'|'member'
  joined_at: string;        // 加入时间（ISO 8601）
}

// 5. POST /api/v1/protected/group/create - 创建群组
export interface CreateGroupRequest {
  name: string;             // 群组名称
  description?: string;     // 群组描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  password?: string;        // 群组密码（可选）
}

export interface CreateGroupResponse {
  group_id: string;         // 新创建的群组ID
  name: string;             // 群组名称
  description?: string;     // 群组描述
  created_at: string;       // 创建时间（ISO 8601）
  message: string;          // 响应消息
}

// 6. POST /api/v1/protected/group/join - 加入群组
export interface JoinGroupRequest {
  group_id: string;         // 群组ID
  password?: string;        // 群组密码（如需要）
}

export interface JoinGroupResponse {
  success: boolean;         // 是否加入成功
  message: string;          // 响应消息
  group_id: string;         // 群组ID
  user_role: string;        // 用户在群组中的角色
}

// 7. POST /api/v1/protected/group/leave - 离开群组
export interface LeaveGroupRequest {
  group_id: string;         // 群组ID
}

export interface LeaveGroupResponse {
  success: boolean;         // 是否离开成功
  message: string;          // 响应消息
  group_id: string;         // 群组ID
}

// 8. POST /api/v1/protected/group/kick_member - 踢出群组成员
export interface KickMemberRequest {
  group_id: string;         // 群组ID
  user_id: string;          // 被踢出的用户ID
  reason?: string;          // 踢出原因（可选）
}

export interface KickMemberResponse {
  success: boolean;         // 是否踢出成功
  message: string;          // 响应消息
  kicked_user_id: string;   // 被踢出的用户ID
}

// 9. POST /api/v1/protected/group/transfer_ownership - 转移群组所有权
export interface TransferOwnershipRequest {
  group_id: string;         // 群组ID
  new_owner_id: string;     // 新群主的用户ID
}

export interface TransferOwnershipResponse {
  success: boolean;         // 是否转移成功
  message: string;          // 响应消息
  new_owner_id: string;     // 新群主的用户ID
  transferred_at: string;   // 转移时间（ISO 8601）
}

// 10. POST /api/v1/protected/group/get_my_groups - 获取用户参与的群组列表
export interface GetMyGroupsRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  role_filter?: string;     // 角色过滤：'owner'|'admin'|'member'（可选）
}

// 获取我的群组响应 - 专门的API响应别名
export type GetMyGroupsResponse = GroupResponse;

// ==================== 消息通讯API类型（统一架构）====================

// 1. POST /api/v1/protected/chat/send - 发送消息（统一接口）
export interface SendMessageRequest {
  group_id: string;         // 群组ID（统一架构：私聊和群聊都是群组）
  content: string;          // 消息内容
  message_type?: string;    // 消息类型，默认'text'
  reply_to_message_id?: string; // 回复的消息ID
  mentioned_users?: string[]; // 提及的用户ID数组
  latitude?: number;        // 纬度（位置消息）
  longitude?: number;       // 经度（位置消息）
}

export interface SendMessageResponse {
  message_id: string;       // 新发送的消息ID
  group_id: string;         // 群组ID
  sent_at: string;          // 发送时间（ISO 8601）
  message: string;          // 响应消息
}

// 2. POST /api/v1/protected/chat/history - 获取消息历史（统一接口）
export interface GetHistoryRequest {
  group_id: string;         // 群组ID（统一架构）
  limit?: number;           // 返回数量限制，默认50
  cursor?: string;          // 分页游标
  include_system_messages?: boolean; // 是否包含系统消息，默认true
}

export interface GetHistoryResponse {
  message_id: string;       // UUID转字符串
  group_id: string;         // 群组ID（统一架构）
  sender_id?: string;       // 发送者ID（系统消息可为null）
  sender_name?: string;     // 发送者昵称（系统消息可为null）
  content: string;          // 消息内容
  message_type: string;     // 消息类型：'text'|'image'|'video'|'audio'|'file'|'location'|'system'
  system_event_type?: string; // 系统事件类型（仅系统消息）
  reply_to_message_id?: string; // 回复的消息ID
  is_edited: boolean;       // 是否已编辑
  reaction_count: number;   // 反应数量
  mentioned_users: string[]; // 提及的用户ID数组
  affected_users: string[]; // 受影响的用户ID数组（系统消息用）
  sent_at: string;          // 发送时间（ISO 8601）
  edited_at?: string;       // 编辑时间（ISO 8601）
  latitude?: number;        // 纬度（位置消息）
  longitude?: number;       // 经度（位置消息）
}

// 3. POST /api/v1/protected/chat/delete - 删除消息
export interface DeleteMessageRequest {
  message_id: string;       // 消息ID
}

export interface DeleteMessageResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
  message_id: string;       // 被删除的消息ID
}

// 4. POST /api/v1/protected/chat/mark_read - 标记消息已读
export interface MarkMessagesReadRequest {
  group_id: string;         // 群组ID（统一架构）
  message_ids?: string[];   // 消息ID数组（可选，不提供则标记所有未读消息）
}

export interface MarkMessagesReadResponse {
  success: boolean;         // 是否标记成功
  message: string;          // 响应消息
  marked_count: number;     // 标记的消息数量
}

// 5. POST /api/v1/protected/chat/unread_count - 获取未读消息数
export interface GetUnreadCountRequest {
  group_id?: string;        // 群组ID（可选，不提供则获取所有群组的未读数）
}

export interface GetUnreadCountResponse {
  group_id: string;         // 群组ID
  unread_count: number;     // 未读消息数
  last_message_at?: string; // 最后一条消息时间（ISO 8601）
}

// 6. POST /api/v1/protected/chat/conversations - 获取会话列表
export interface GetConversationsRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetConversationsResponse {
  group_id: string;         // 群组ID（统一架构）
  group_name: string;       // 群组名称
  group_type: string;       // 群组类型：'private'|'group'
  last_message_content?: string; // 最后一条消息内容
  last_message_at?: string; // 最后一条消息时间（ISO 8601）
  unread_count: number;     // 未读消息数
  is_muted: boolean;        // 是否静音
  avatar_url?: string;      // 群组头像URL（群聊）或对方头像URL（私聊）
}

// ==================== 生活瞬间API类型 ====================

// 通用生活瞬间响应类型
export interface MomentResponse {
  moment_id: string;        // UUID转字符串
  user_id: string;          // 用户ID（64位整数转字符串）
  nickname: string;         // 用户昵称
  user_avatar?: string;     // 用户头像URL
  description: string;      // 瞬间描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  created_at: string;       // 创建时间（ISO 8601）
  likes_count: number;      // 点赞数（32位整数）
  comments_count: number;   // 评论数（32位整数）
  liked_by_me: boolean;     // 当前用户是否已点赞
  distance?: number;        // 距离（米）
  tags: string[];           // 标签数组
  images?: string[];        // 图片URL数组
}

// 1. POST /api/v1/protected/moments/create - 创建生活瞬间
export interface CreateMomentRequest {
  description: string;      // 瞬间描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  tags?: string[];          // 标签数组
  images?: string[];        // 图片URL数组
}

export interface CreateMomentResponse {
  moment_id: string;        // 新创建的瞬间ID
  description: string;      // 瞬间描述
  created_at: string;       // 创建时间（ISO 8601）
  message: string;          // 响应消息
}

// 2. POST /api/v1/public/moments/search_by_location - 按位置搜索生活瞬间
export interface SearchMomentsByLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  radius: number;           // 搜索半径（米）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 按位置搜索生活瞬间响应 - 扩展MomentResponse并要求distance字段必填
export interface SearchMomentsByLocationResponse extends MomentResponse {
  distance: number;         // 距离（米，必填）
}

// 3. POST /api/v1/protected/moments/like - 点赞生活瞬间
export interface LikeMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface LikeMomentResponse {
  success: boolean;         // 是否点赞成功
  message: string;          // 响应消息
  likes_count: number;      // 更新后的点赞数
}

// 4. POST /api/v1/protected/moments/unlike - 取消点赞
export interface UnlikeMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface UnlikeMomentResponse {
  success: boolean;         // 是否取消点赞成功
  message: string;          // 响应消息
  likes_count: number;      // 更新后的点赞数
}

// 5. POST /api/v1/protected/moments/delete - 删除生活瞬间
export interface DeleteMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface DeleteMomentResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
  moment_id: string;        // 被删除的瞬间ID
}

// 6. POST /api/v1/public/moments/search_by_tag - 按标签搜索生活瞬间
export interface SearchMomentsByTagRequest {
  tag: string;              // 标签名称
  latitude?: number;        // 纬度（可选，用于距离排序）
  longitude?: number;       // 经度（可选，用于距离排序）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 按标签搜索生活瞬间响应 - 专门的API响应别名
export type SearchMomentsByTagResponse = MomentResponse;

// 7. POST /api/v1/public/moments/search_by_id - 按ID获取生活瞬间详情
export interface SearchMomentByIdRequest {
  moment_id: string;        // 瞬间ID
}

// 按ID获取生活瞬间详情响应 - 专门的API响应别名
export type SearchMomentByIdResponse = MomentResponse;

// 8. POST /api/v1/protected/moments/get_history - 获取用户历史瞬间
export interface GetMomentsHistoryRequest {
  user_id?: string;         // 用户ID（可选，不提供则获取当前用户的历史）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

// 获取历史瞬间响应 - 专门的API响应别名
export type GetMomentsHistoryResponse = MomentResponse;

// ==================== 评论互动API类型 ====================

// 1. POST /api/v1/protected/comments/add - 添加评论
export interface AddCommentRequest {
  moment_id: string;        // 瞬间ID
  content: string;          // 评论内容
  parent_comment_id?: string; // 父评论ID（回复评论时使用）
}

export interface AddCommentResponse {
  comment_id: string;       // 新评论ID
  moment_id: string;        // 瞬间ID
  content: string;          // 评论内容
  created_at: string;       // 创建时间（ISO 8601）
  message: string;          // 响应消息
}

// 2. POST /api/v1/protected/comments/reply - 回复评论
export interface ReplyCommentRequest {
  comment_id: string;       // 父评论ID
  content: string;          // 回复内容
}

export interface ReplyCommentResponse {
  comment_id: string;       // 新创建的回复ID
  parent_comment_id: string; // 父评论ID
  content: string;          // 回复内容
  created_at: string;       // 创建时间（ISO 8601）
  message: string;          // 响应消息
}

// 3. POST /api/v1/protected/comments/delete - 删除评论
export interface DeleteCommentRequest {
  comment_id: string;       // 评论ID
}

export interface DeleteCommentResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
  comment_id: string;       // 被删除的评论ID
}

// 4. POST /api/v1/public/comments/list - 获取评论列表
export interface GetCommentsListRequest {
  moment_id: string;        // 瞬间ID
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetCommentsListResponse {
  comment_id: string;       // 评论ID
  moment_id: string;        // 瞬间ID
  user_id: string;          // 评论者用户ID
  nickname: string;         // 评论者昵称
  avatar_url?: string;      // 评论者头像URL
  content: string;          // 评论内容
  parent_comment_id?: string; // 父评论ID（回复评论）
  reply_count: number;      // 回复数量
  created_at: string;       // 创建时间（ISO 8601）
}

// ==================== 通知推送API类型 ====================

// 1. POST /api/v1/protected/notifications/list - 获取通知列表
export interface GetNotificationsRequest {
  type?: string;            // 通知类型过滤：'friend_request'|'group_invite'|'moment_like'|'comment'|'mention'
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  unread_only?: boolean;    // 是否只获取未读通知，默认false
}

export interface GetNotificationsResponse {
  notification_id: string;  // 通知ID
  type: string;             // 通知类型
  title: string;            // 通知标题
  content: string;          // 通知内容
  is_read: boolean;         // 是否已读
  created_at: string;       // 创建时间（ISO 8601）
  related_user_id?: string; // 相关用户ID
  related_user_name?: string; // 相关用户昵称
  related_object_id?: string; // 相关对象ID（群组、瞬间等）
  action_data?: Record<string, Object>; // 操作数据（JSON格式）
}

// 2. POST /api/v1/protected/notifications/mark_read - 标记通知已读
export interface MarkNotificationsReadRequest {
  notification_ids?: string[]; // 通知ID数组（可选，不提供则标记所有未读通知）
}

export interface MarkNotificationsReadResponse {
  success: boolean;         // 是否标记成功
  message: string;          // 响应消息
  marked_count: number;     // 标记的通知数量
}

// ==================== 内容安全API类型 ====================

// 1. POST /api/v1/protected/security/report - 举报内容
export interface ReportContentRequest {
  content_type: string;     // 内容类型：'user'|'group'|'moment'|'message'|'comment'
  content_id: string;       // 内容ID
  reason: string;           // 举报原因
  description?: string;     // 详细描述（可选）
}

export interface ReportContentResponse {
  success: boolean;         // 是否举报成功
  message: string;          // 响应消息
  report_id: string;        // 举报记录ID
  submitted_at: string;     // 提交时间（ISO 8601）
}

// ==================== 系统状态API类型 ====================

// 数据库状态信息
interface DatabaseStatus {
  status: string;         // 连接状态：'connected'|'disconnected'
  response_time: number;  // 响应时间（毫秒）
}

// Redis状态信息
interface RedisStatus {
  status: string;         // 连接状态：'connected'|'disconnected'
  response_time: number;  // 响应时间（毫秒）
}

// 系统指标信息
interface SystemMetrics {
  cpu_usage: number;      // CPU使用率（0-100）
  memory_usage: number;   // 内存使用率（0-100）
  active_connections: number; // 活跃连接数
  total_requests: number; // 总请求数
  error_rate: number;     // 错误率（0-100）
}

// 1. POST /api/v1/public/status/health - 系统健康检查
export interface HealthCheckRequest {
  // 空请求体
}

export interface HealthCheckResponse {
  status: string;           // 系统状态：'healthy'|'degraded'|'unhealthy'
  timestamp: string;        // 检查时间（ISO 8601）
  version: string;          // 系统版本
  uptime: number;           // 运行时间（秒）
  database: DatabaseStatus; // 数据库状态
  redis: RedisStatus;       // Redis状态
  metrics: SystemMetrics;   // 系统指标
}

