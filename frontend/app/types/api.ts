/**
 * 统一API类型定义 - 严格按照 backend_doc.mdc 规范
 * 
 * 重要说明：
 * - 所有API响应都使用统一的 ApiResponse<T> 格式
 * - 严格遵循 XxxRequest/XxxResponse 命名规范
 * - ApiResponse<T> 中的 T 就是 XxxResponse 类型
 * - 对于分页查询，T 为 PaginatedResponse<XxxResponse>
 */

// ============================================================================
// 基础响应类型
// ============================================================================

export interface ApiResponse<T = any> {
  code: number;        // 业务状态码：0-成功，非0-错误
  message?: string;    // 响应消息（失败时必须返回，成功时可选）
  data?: T;            // 成功时为具体业务数据
}

export interface PaginatedResponse<T> {
  items: T[];          // 数据项数组
  pagination: {
    has_more: boolean;   // 是否有更多数据
    next_cursor?: string; // 下一页游标（UUID字符串）
  };
}

// ============================================================================
// 用户ID编码规范工具函数
// ============================================================================

export { 
  getUserType, 
  isTempUser, 
  isRegularUser, 
  isSystemUser,
  getUserTypeDisplayName,
  getUserTypeTagStyle,
  UserType,
  EXAMPLE_USER_IDS
} from '~/utils/userTypes'

// ============================================================================
// 地理位置类型
// ============================================================================

export interface Coordinates {
  latitude: number;
  longitude: number;
  accuracy?: number;
  timestamp?: number;
}

// ============================================================================
// 密码安全工具函数
// ============================================================================

/**
 * 密码哈希工具已移至 ~/utils/password.ts
 * 使用方法：
 * import { hashPassword } from '~/utils/password'
 * const hash = await hashPassword('myPassword')
 * 
 * 根据 api_spec.mdc 2.10：只需要 SHA256(password)
 */

// ============================================================================
// 认证API类型
// ============================================================================

// 通用认证响应类型
export interface AuthResponse {
  user_id: string;          // 64位整数转字符串（高8位编码用户类型）
  nickname: string;         // 用户昵称
  access_token: string;     // 访问令牌
  refresh_token: string;    // 刷新令牌
  expires_at: string;       // 过期时间（ISO 8601）
}

// 1. POST /api/v1/public/auth/register - 用户注册
export interface RegisterRequest {
  login_id: string;         // 登录ID（邮箱/手机号）
  nickname: string;         // 用户昵称
  password_hash: string;    // SHA256(password) - 根据 api_spec.mdc 2.10
}

export interface RegisterResponse extends AuthResponse {}

// 2. POST /api/v1/public/auth/login - 用户登录
export interface LoginRequest {
  login_id: string;         // 登录ID（邮箱/手机号）
  password_hash: string;    // SHA256(password) - 根据 api_spec.mdc 2.10
}

export interface LoginResponse extends AuthResponse {}

// 3. POST /api/v1/public/auth/create_temp_user - 创建临时用户
export interface CreateTempRequest {
  nickname: string;         // 用户昵称
}

export interface CreateTempResponse extends AuthResponse {}

// 4. POST /api/v1/protected/auth/refresh_token - 刷新Token
export interface RefreshTokenRequest {
  refresh_token: string;    // 刷新令牌
}

export interface RefreshTokenResponse extends AuthResponse {}

// 5. POST /api/v1/protected/auth/check_status - 检查认证状态
export interface CheckStatusRequest {}

export interface CheckStatusResponse {
  user_id: string;
  nickname: string;
  is_valid: boolean;
  expires_at: string;
}

// 6. POST /api/v1/protected/auth/set_password - 设置密码
export interface SetPasswordRequest {
  password_hash: string;    // SHA256(password) - 根据 api_spec.mdc 2.10
}

export interface SetPasswordResponse {
  can_login: boolean;
}

// 7. POST /api/v1/public/auth/request_password_reset - 请求密码重置
export interface RequestPasswordResetRequest {
  login_id: string;         // 登录ID（邮箱/手机号）
}

export interface RequestPasswordResetResponse {
  reset_id: string;
}

// 8. POST /api/v1/public/auth/verify_reset_code - 验证重置码
export interface VerifyResetCodeRequest {
  reset_id: string;         // 重置请求ID
  code: string;             // 验证码
}

export interface VerifyResetCodeResponse {
  reset_token: string;
}

// 9. POST /api/v1/public/auth/reset_password - 重置密码
export interface ResetPasswordRequest {
  reset_token: string;      // 验证后获得的临时token
  password_hash: string;    // SHA256(password) - 根据 api_spec.mdc 2.10
}

// 10. POST /api/v1/protected/auth/get_sessions - 获取活跃会话列表
export interface ListSessionsRequest {
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

export interface ListSessionsResponse extends PaginatedData<SessionItem> {}

// 11. POST /api/v1/protected/auth/revoke_session - 撤销会话
export interface RevokeSessionRequest {
  id: string;               // 要撤销的会话ID
}

// ============================================================================
// 用户API类型
// ============================================================================

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

export interface SearchUserByNameResponse extends PaginatedResponse<UserResponse> {}

// 2. POST /api/v1/public/user/search_by_id - 获取用户资料
export interface SearchUserByIdRequest {
  user_id: string;          // 用户ID
}

export interface SearchUserByIdResponse extends UserResponse {}

// 3. POST /api/v1/public/user/search_by_location - 搜索附近用户
export interface SearchUserByLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  radius: number;           // 搜索半径（米）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface SearchUserByLocationResponseItem extends UserResponse {
  distance: number;         // 距离（米，必填）
}

export interface SearchUserByLocationResponse extends PaginatedResponse<SearchUserByLocationResponseItem> {}

// 4. POST /api/v1/protected/user/update_location - 更新用户位置
export interface UpdateLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name?: string;   // 位置名称
}

export interface UpdateLocationResponse {
  success: boolean;         // 是否更新成功
  message: string;          // 响应消息
}

// 5. POST /api/v1/protected/user/update_profile - 更新用户资料
export interface UpdateProfileRequest {
  nickname?: string;        // 用户昵称
  avatar_url?: string;      // 头像URL
  bio?: string;             // 个人简介
}

export interface UpdateProfileResponse {
  success: boolean;         // 是否更新成功
  message: string;          // 响应消息
}

// 6. POST /api/v1/protected/user/upgrade_account - 账户升级
export interface UpgradeAccountRequest {
  login_id: string;         // 新的登录ID（邮箱/手机号）
  password_hash: string;    // SHA256(password) - 根据 api_spec.mdc 2.10
}

export interface UpgradeAccountResponse {}

// 7. POST /api/v1/protected/user/get_settings - 获取用户设置
export interface GetSettingsRequest {}

// ✅ 与 api_spec.mdc 第3.2.9节保持一致
export interface UserSettingsData {
  privacy_level: 'public' | 'friends' | 'private';
  is_location_sharing: boolean;
  can_be_followed: boolean;
  allow_private_chat: boolean;  // 是否允许别人发起私聊
}

export type GetSettingsResponse = ApiResponse<UserSettingsData>

// 8. POST /api/v1/protected/user/update_settings - 更新用户设置
export interface UpdateSettingsRequest {
  privacy_level?: 'public' | 'friends' | 'private';
  is_location_sharing?: boolean;
  can_be_followed?: boolean;
  allow_private_chat?: boolean;
}

export type UpdateSettingsResponse = ApiResponse

// 9. POST /api/v1/protected/user/delete_account - 删除账户
export interface DeleteAccountRequest {
  confirmation: string;     // 确认字符串，必须为 "DELETE_MY_ACCOUNT"
}

export interface DeleteAccountResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
}

// 10. POST /api/v1/protected/user/add_friend - 发送好友申请
export interface AddFriendRequest {
  to_user_id: string;       // 目标用户ID
  message?: string;         // 申请消息（可选）
}

export interface AddFriendResponse {
  success: boolean;         // 是否发送成功
  message: string;          // 响应消息
  relationship_id: string;  // 关系记录ID
}

// 11. POST /api/v1/protected/user/accept_friend - 接受好友申请
export interface AcceptFriendRequest {
  relationship_id: string;  // 关系记录ID
}

export interface AcceptFriendResponse {
  success: boolean;         // 是否接受成功
  message: string;          // 响应消息
}

// 12. POST /api/v1/protected/user/reject_friend - 拒绝好友申请
export interface RejectFriendRequest {
  relationship_id: string;  // 关系记录ID
}

export interface RejectFriendResponse {
  success: boolean;         // 是否拒绝成功
  message: string;          // 响应消息
}

// 13. POST /api/v1/protected/user/follow_user - 关注用户
export interface FollowRequest {
  to_user_id: string;       // 目标用户ID
}

export interface FollowResponse {
  success: boolean;         // 是否关注成功
  message: string;          // 响应消息
  relationship_id: string;  // 关系记录ID
}

// 14. POST /api/v1/protected/user/unfollow_user - 取消关注
export interface UnfollowRequest {
  to_user_id: string;       // 目标用户ID
}

export interface UnfollowResponse {
  success: boolean;         // 是否取消成功
  message: string;          // 响应消息
}

// 15. POST /api/v1/protected/user/block_user - 拉黑用户
export interface BlockRequest {
  to_user_id: string;       // 目标用户ID
  reason?: string;          // 拉黑原因（可选）
}

export interface BlockResponse {
  success: boolean;         // 是否拉黑成功
  message: string;          // 响应消息
  relationship_id: string;  // 关系记录ID
}

// 16. POST /api/v1/protected/user/friends_list - 获取好友列表
export interface GetFriendsListRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  status?: string;          // 状态过滤：'pending'|'accepted'|'all'，默认'accepted'
}

export interface GetFriendsListResponseItem {
  user_id: string;          // 好友用户ID
  nickname: string;         // 好友昵称
  avatar_url?: string;      // 好友头像URL
  bio?: string;             // 好友个人简介
  relationship_type: 'friend' | 'pending_sent' | 'pending_received'; // 关系类型
  relationship_id: string;  // 关系记录ID
  created_at: string;       // 建立关系时间（ISO 8601）
  last_active_at?: string;  // 好友最后活跃时间（ISO 8601）
  is_online?: boolean;      // 是否在线（可选）
}

export interface GetFriendsListResponse extends PaginatedResponse<GetFriendsListResponseItem> {}

// 17. POST /api/v1/protected/user/blocked_list - 获取拉黑列表
export interface GetBlockedListRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetBlockedListResponseItem {
  user_id: string;          // 被拉黑用户ID
  nickname: string;         // 被拉黑用户昵称
  avatar_url?: string;      // 被拉黑用户头像URL
  relationship_id: string;  // 关系记录ID
  blocked_at: string;       // 拉黑时间（ISO 8601）
  reason?: string;          // 拉黑原因
}

export interface GetBlockedListResponse extends PaginatedResponse<GetBlockedListResponseItem> {}

// ============================================================================
// 群组管理API类型
// ============================================================================

// 通用群组信息响应类型（适用于公开群组和私聊会话）
// 私聊会话的特殊值：group_type='private_chat', name=null, latitude=null, longitude=null, location_name=null, is_password_required=false, avatar_url=null
// member_count为实际成员数量（初始为2，可随添加成员而增长）
export interface GroupResponse {
  group_id: string;         // UUID转字符串
  group_type: 'public_chat' | 'private_chat'; // 群组类型
  name?: string;            // 群组名称（私聊为null，由前端动态生成）
  description?: string;     // 群组描述（私聊为null）
  owner_id: string;         // 群主ID（64位整数转字符串）
  creator_name: string;     // 创建者昵称
  created_at: string;       // 创建时间（ISO 8601）
  last_active_at: string;   // 最后活跃时间（ISO 8601）
  latitude?: number;        // 纬度（私聊为null）
  longitude?: number;       // 经度（私聊为null）
  member_count: number;     // 成员数量（32位整数）
  distance?: number;        // 距离（米，仅位置搜索时返回）
  location_name?: string;   // 位置名称（私聊为null）
  is_password_required: boolean; // 是否需要密码（私聊固定为false）
  is_member: boolean;       // 当前用户是否为成员
  user_role?: 'owner' | 'admin' | 'member'; // 当前用户在群组中的角色
  avatar_url?: string;      // 群组头像URL（私聊为null，由前端动态生成）
}

// 1. POST /api/v1/public/group/search_by_location - 按位置搜索群组
export interface SearchGroupByLocationRequest {
  latitude: number;         // 纬度
  longitude: number;        // 经度
  radius: number;           // 搜索半径（米）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface SearchGroupByLocationResponseItem extends GroupResponse {
  distance: number;         // 距离（米，必填）
}

export interface SearchGroupByLocationResponse extends PaginatedResponse<SearchGroupByLocationResponseItem> {}

// 2. POST /api/v1/public/group/search_by_name - 按名称搜索群组
export interface SearchGroupByNameRequest {
  name: string;             // 搜索关键词
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface SearchGroupByNameResponse extends PaginatedResponse<GroupResponse> {}

// 3. POST /api/v1/public/group/search_by_id - 按ID获取群组详情
export interface SearchGroupByIdRequest {
  group_id: string;         // 群组ID
}

export interface SearchGroupByIdResponse extends GroupResponse {}

// 4. POST /api/v1/public/group/members - 获取群组成员
export interface GetMembersRequest {
  group_id: string;         // 群组ID
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GroupMemberResponseData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 用户头像URL
  role: 'owner' | 'admin' | 'member'; // 用户角色
  joined_at: string;        // 加入时间（ISO 8601）
}

export interface GetMembersResponse extends PaginatedResponse<GroupMemberResponseData> {}


// 5. POST /api/v1/protected/group/create_public - 创建公开群组
export interface CreatePublicGroupRequest {
  name: string;             // 群组名称
  description?: string;     // 群组描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  password?: string;        // 群组密码（可选）
}

export interface CreatePublicGroupResponse extends GroupResponse {
  message: string;          // 响应消息
}

// 6. POST /api/v1/protected/group/create_private - 创建私聊会话
export interface CreatePrivateGroupRequest {
  participant_id: string;   // 私聊对象的用户ID
  initial_message?: string; // 初始消息（可选）
}

export interface CreatePrivateGroupResponse extends GroupResponse {
  message: string;          // 响应消息
  initial_message_id?: string; // 初始消息ID（如果发送了初始消息）
}

// 7. POST /api/v1/protected/group/convert_to_public - 将私聊转换为公开群组
export interface ConvertToPublicGroupRequest {
  group_id: string;         // 私聊群组ID
  name: string;             // 群组名称
  description?: string;     // 群组描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  password?: string;        // 群组密码（可选）
}

export interface ConvertToPublicGroupResponse extends GroupResponse {
  message: string;          // 响应消息
}

// 8. POST /api/v1/protected/group/join - 加入群组
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

// 9. POST /api/v1/protected/group/leave - 离开群组
export interface LeaveGroupRequest {
  group_id: string;         // 群组ID
}

export interface LeaveGroupResponse {
  success: boolean;         // 是否离开成功
  message: string;          // 响应消息
}

// 10. POST /api/v1/protected/group/kick - 踢出群组成员
export interface KickGroupRequest {
  group_id: string;         // 群组ID
  user_id: string;          // 被踢出的用户ID
  reason?: string;          // 踢出原因
}

export interface KickGroupResponse {
  success: boolean;         // 是否踢出成功
  message: string;          // 响应消息
}

// 11. POST /api/v1/protected/group/transfer_ownership - 转移群组所有权
export interface TransferOwnershipRequest {
  group_id: string;         // 群组ID
  new_owner_id: string;     // 新群主用户ID
}

export interface TransferOwnershipResponse {
  success: boolean;         // 是否转移成功
  message: string;          // 响应消息
  new_owner_id: string;     // 新群主用户ID
}

// 12. POST /api/v1/protected/group/my_groups - 获取用户参与的群组列表
export interface MyGroupsRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  group_type_filter?: string; // 群组类型过滤：'public_chat'|'private_chat'|'all'，默认'all'
  include_stats?: boolean;  // 是否包含统计信息（未读数等），默认true
}

export interface MyGroupsResponse {
  group_id: string;         // UUID转字符串
  name?: string;            // 群组名称（私聊群组可为null）
  description?: string;     // 群组描述
  owner_id: string;         // 群主ID（64位整数转字符串）
  creator_name: string;     // 创建者昵称
  created_at: string;       // 创建时间（ISO 8601）
  last_active_at: string;   // 最后活跃时间（ISO 8601）
  latitude?: number;        // 纬度（私聊群组可为null）
  longitude?: number;       // 经度（私聊群组可为null）
  member_count: number;     // 成员数量（32位整数）
  distance?: number;        // 距离（米）
  location_name?: string;   // 位置名称（私聊群组可为null）
  is_password_required: boolean; // 是否需要密码
  avatar_url?: string;      // 群组头像URL
  group_type: 'public_chat' | 'private_chat'; // 群组类型
  
  // 当前用户在此群组中的状态（来自group_members）
  user_role: 'owner' | 'admin' | 'member'; // 用户角色
  user_status: 'active' | 'muted' | 'banned' | 'left'; // 用户状态
  unread_count: number;     // 未读消息数
  notifications_enabled: boolean; // 通知开关
  last_read_at?: string;    // 最后已读时间（ISO 8601）
  joined_at: string;        // 加入时间（ISO 8601）
}

// ============================================================================
// 统一消息通讯API类型
// ============================================================================

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
  message_type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'location' | 'system'; // 消息类型
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

// 3. POST /api/v1/protected/chat/delete - 删除消息（统一接口）
export interface DeleteMessageRequest {
  message_id: string;       // 消息ID
  delete_for_everyone?: boolean; // 是否为所有人删除，默认false
}

export interface DeleteMessageResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
}

// 4. POST /api/v1/protected/chat/mark-read - 标记消息已读（统一接口）
export interface MarkReadRequest {
  group_id: string;         // 群组ID
  message_id?: string;      // 最后已读消息ID（可选，不提供则标记所有消息为已读）
}

export interface MarkReadResponse {
  success: boolean;         // 是否标记成功
  message: string;          // 响应消息
}

// 5. POST /api/v1/protected/chat/unread_count - 获取未读消息数（统一接口）
export interface UnreadCountRequest {
  group_id?: string;        // 群组ID（可选，不提供则获取所有群组的未读数）
}

export interface UnreadCountResponse {
  total_unread: number;     // 总未读消息数
  groups?: Array<{          // 各群组未读数（仅当未指定group_id时返回）
    group_id: string;       // 群组ID
    unread_count: number;   // 未读消息数
  }>;
}

// 6. POST /api/v1/protected/chat/conversations - 获取会话列表（统一接口）
export interface ConversationsRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  include_archived?: boolean; // 是否包含已归档会话，默认false
}

export interface ConversationsResponse {
  group_id: string;         // 群组ID
  group_name: string;       // 群组名称
  group_type: 'private_chat' | 'public_chat'; // 群组类型
  group_avatar?: string;    // 群组头像URL
  last_message?: {          // 最后一条消息
    message_id: string;     // 消息ID
    sender_id: string;      // 发送者ID
    sender_name: string;    // 发送者昵称
    content: string;        // 消息内容
    message_type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'location' | 'system'; // 消息类型
    sent_at: string;        // 发送时间（ISO 8601）
  };
  unread_count: number;     // 未读消息数
  last_activity_at: string; // 最后活跃时间（ISO 8601）
  is_muted: boolean;        // 是否静音
  is_archived: boolean;     // 是否已归档
  member_count: number;     // 成员数量
}

// ============================================================================
// 生活瞬间API类型
// ============================================================================

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

export interface SearchMomentsByLocationResponseItem extends MomentResponse {
  distance: number;         // 距离（米，必填）
}

export interface SearchMomentsByLocationResponse extends PaginatedResponse<SearchMomentsByLocationResponseItem> {}

// 3. POST /api/v1/public/moments/search_by_tags - 按标签搜索生活瞬间
export interface SearchMomentsByTagsRequest {
  tags: string[];           // 标签数组
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
  latitude?: number;        // 纬度（可选）
  longitude?: number;       // 经度（可选）
}

export interface SearchMomentsByTagsResponse extends PaginatedResponse<MomentResponse> {}

// 4. POST /api/v1/public/moments/search_by_id - 按ID搜索生活瞬间
export interface SearchMomentByIdRequest {
  moment_id: string;        // 瞬间ID
}

export interface SearchMomentByIdResponse extends MomentResponse {}

// 5. POST /api/v1/public/moments/history - 获取生活瞬间历史
export interface GetMomentsHistoryRequest {
  user_id?: string;         // 用户ID（可选，不提供则获取所有用户）
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetMomentsHistoryResponse extends PaginatedResponse<MomentResponse> {}

// 6. POST /api/v1/protected/moments/delete - 删除生活瞬间
export interface DeleteMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface DeleteMomentResponse {
  success: boolean;         // 是否删除成功
  message: string;          // 响应消息
}

// 7. POST /api/v1/protected/moments/like - 点赞生活瞬间
export interface LikeMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface LikeMomentResponse {
  success: boolean;         // 是否点赞成功
  message: string;          // 响应消息
  likes_count: number;      // 更新后的点赞数
}

// 8. POST /api/v1/protected/moments/unlike - 取消点赞
export interface UnlikeMomentRequest {
  moment_id: string;        // 瞬间ID
}

export interface UnlikeMomentResponse {
  success: boolean;         // 是否取消点赞成功
  message: string;          // 响应消息
  likes_count: number;      // 更新后的点赞数
}

// ============================================================================
// 评论互动API类型
// ============================================================================

// 通用评论响应类型
export interface CommentResponse {
  comment_id: string;       // 评论ID
  moment_id: string;        // 瞬间ID
  user_id: string;          // 评论者ID
  nickname: string;         // 评论者昵称
  user_avatar?: string;     // 评论者头像URL
  content: string;          // 评论内容
  parent_comment_id?: string; // 父评论ID（回复时）
  created_at: string;       // 创建时间（ISO 8601）
  replies_count: number;    // 回复数量
}

// 1. POST /api/v1/protected/comments/add - 添加评论
export interface AddCommentRequest {
  moment_id: string;        // 瞬间ID
  content: string;          // 评论内容
}

export interface AddCommentResponse {
  comment_id: string;       // 新创建的评论ID
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
}

// 4. POST /api/v1/public/comments/list - 获取评论列表
export interface GetCommentsRequest {
  moment_id: string;        // 瞬间ID
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetCommentsResponse extends PaginatedResponse<CommentResponse> {}

// ============================================================================
// 内容安全API类型
// ============================================================================

// 通用举报响应类型
export interface ReportResponse {
  report_id: string;        // 举报记录ID
  reporter_id: string;      // 举报者ID
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group'; // 举报目标类型
  target_id: string;        // 举报目标ID
  reason: 'spam' | 'harassment' | 'inappropriate_content' | 'fake_account' | 'violence' | 'other'; // 举报原因
  description?: string;     // 详细描述
  status: 'pending' | 'reviewing' | 'resolved' | 'dismissed'; // 处理状态
  created_at: string;       // 创建时间（ISO 8601）
  handled_at?: string;      // 处理时间（ISO 8601）
  handler_note?: string;    // 处理备注
}

// 1. POST /api/v1/protected/reports/create - 提交举报
export interface CreateReportRequest {
  target_type: 'user' | 'moment' | 'comment' | 'message' | 'group'; // 举报目标类型
  target_id: string;        // 举报目标ID
  reason: 'spam' | 'harassment' | 'inappropriate_content' | 'fake_account' | 'violence' | 'other'; // 举报原因
  description?: string;     // 详细描述
}

export interface CreateReportResponse {
  report_id: string;        // 举报记录ID
  message: string;          // 响应消息
  created_at: string;       // 创建时间（ISO 8601）
}

// 2. POST /api/v1/protected/reports/my_reports - 我的举报记录
export interface GetMyReportsRequest {
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetMyReportsResponse extends PaginatedResponse<ReportResponse> {}

// 3. POST /api/v1/protected/reports/list - 举报列表（管理员）
export interface GetReportsRequest {
  status?: string;          // 状态过滤：'pending'|'reviewing'|'resolved'|'dismissed'
  target_type?: string;     // 目标类型过滤
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetReportsResponse extends PaginatedResponse<ReportResponse> {}

// 4. POST /api/v1/protected/reports/handle - 处理举报（管理员）
export interface HandleReportRequest {
  report_id: string;        // 举报记录ID
  action: 'resolve' | 'dismiss' | 'escalate'; // 处理动作
  note?: string;            // 处理备注
}

export interface HandleReportResponse {
  success: boolean;         // 是否处理成功
  message: string;          // 响应消息
  handled_at: string;       // 处理时间（ISO 8601）
}

// 5. POST /api/v1/protected/users/block - 屏蔽用户
export interface BlockUserRequest {
  user_id: string;          // 要屏蔽的用户ID
}

export interface BlockUserResponse {
  success: boolean;         // 是否屏蔽成功
  message: string;          // 响应消息
}

// ============================================================================
// 通知推送API类型
// ============================================================================

// 通用通知响应类型
export interface NotificationResponse {
  notification_id: string;  // 通知ID
  user_id: string;          // 接收者ID
  type: 'friend_request' | 'friend_accepted' | 'group_invite' | 'group_invite_response' | 'group_application' | 'group_application_response' | 'message_mention' | 'moment_like' | 'moment_comment' | 'comment_reply' | 'system_announcement' | 'system_maintenance' | 'account_warning' | 'security_alert'; // 通知类型
  title: string;            // 通知标题
  content: string;          // 通知内容
  data?: object;            // 扩展数据（JSON）
  is_read: boolean;         // 是否已读
  created_at: string;       // 创建时间（ISO 8601）
  read_at?: string;         // 已读时间（ISO 8601）
}

// 1. POST /api/v1/protected/notifications/send - 发送通知
export interface SendNotificationRequest {
  user_ids: string[];       // 接收者ID数组
  type: 'friend_request' | 'friend_accepted' | 'group_invite' | 'group_invite_response' | 'group_application' | 'group_application_response' | 'message_mention' | 'moment_like' | 'moment_comment' | 'comment_reply' | 'system_announcement' | 'system_maintenance' | 'account_warning' | 'security_alert'; // 通知类型
  title: string;            // 通知标题
  content: string;          // 通知内容
  data?: object;            // 扩展数据（JSON）
}

export interface SendNotificationResponse {
  success: boolean;         // 是否发送成功
  message: string;          // 响应消息
  sent_count: number;       // 成功发送数量
}

// 2. POST /api/v1/protected/notifications/list - 获取通知列表
export interface GetNotificationsRequest {
  type?: string;            // 通知类型过滤
  is_read?: boolean;        // 已读状态过滤
  limit?: number;           // 返回数量限制，默认20
  cursor?: string;          // 分页游标
}

export interface GetNotificationsResponse extends PaginatedResponse<NotificationResponse> {}

// 3. POST /api/v1/protected/notifications/mark_read - 标记通知已读
export interface MarkNotificationReadRequest {
  notification_ids: string[]; // 通知ID数组
}

export interface MarkNotificationReadResponse {
  success: boolean;         // 是否标记成功
  message: string;          // 响应消息
  marked_count: number;     // 标记数量
}

// 4. POST /api/v1/protected/notifications/unread_count - 获取未读通知数
export interface GetUnreadNotificationCountRequest {
  type?: string;            // 通知类型过滤
}

export interface GetUnreadNotificationCountResponse {
  total_unread: number;     // 总未读数量
  by_type?: Array<{         // 按类型分组的未读数（仅当未指定type时返回）
    type: 'friend_request' | 'friend_accepted' | 'group_invite' | 'group_invite_response' | 'group_application' | 'group_application_response' | 'message_mention' | 'moment_like' | 'moment_comment' | 'comment_reply' | 'system_announcement' | 'system_maintenance' | 'account_warning' | 'security_alert'; // 通知类型
    unread_count: number;   // 未读数量
  }>;
}

// 5. POST /api/v1/protected/notifications/settings - 获取通知设置
export interface GetNotificationSettingsRequest {
  // 空请求体
}

export interface GetNotificationSettingsResponse {
  push_enabled: boolean;    // 推送通知开关
  email_enabled: boolean;   // 邮件通知开关
  friend_request: boolean;  // 好友请求通知
  group_invite: boolean;    // 群组邀请通知
  message: boolean;         // 消息通知
  moment_like: boolean;     // 瞬间点赞通知
  comment: boolean;         // 评论通知
  system: boolean;          // 系统通知
}

// 6. POST /api/v1/protected/notifications/update_settings - 更新通知设置
export interface UpdateNotificationSettingsRequest {
  push_enabled?: boolean;   // 推送通知开关
  email_enabled?: boolean;  // 邮件通知开关
  friend_request?: boolean; // 好友请求通知
  group_invite?: boolean;   // 群组邀请通知
  message?: boolean;        // 消息通知
  moment_like?: boolean;    // 瞬间点赞通知
  comment?: boolean;        // 评论通知
  system?: boolean;         // 系统通知
}

export interface UpdateNotificationSettingsResponse {
  success: boolean;         // 是否更新成功
  message: string;          // 响应消息
}

// ============================================================================
// 系统状态API类型
// ============================================================================

// 1. POST /api/v1/public/status/health - 系统健康检查
export interface HealthCheckRequest {
  // 空请求体
}

export interface HealthCheckResponse {
  status: 'healthy' | 'degraded' | 'unhealthy'; // 系统状态
  timestamp: string;        // 检查时间（ISO 8601）
  version: string;          // 系统版本
  uptime: number;           // 运行时间（秒）
  database: {               // 数据库状态
    status: 'connected' | 'disconnected'; // 连接状态
    response_time: number;  // 响应时间（毫秒）
  };
  redis: {                  // Redis状态
    status: 'connected' | 'disconnected'; // 连接状态
    response_time: number;  // 响应时间（毫秒）
  };
  metrics: {                // 系统指标
    cpu_usage: number;      // CPU使用率（0-100）
    memory_usage: number;   // 内存使用率（0-100）
    active_connections: number; // 活跃连接数
    total_requests: number; // 总请求数
    error_rate: number;     // 错误率（0-100）
  };
}

// ===== 兼容性类型别名 =====
// 为了保持向后兼容，提供一些常用的类型别名

// 基础类型定义
export type MessageType = 'text' | 'image' | 'file' | 'location' | 'system';

// 用户设置类型 - 与 backend_doc.mdc 中的 GetSettingsResponse 保持一致
export interface UserSettings {
  // 通知设置 - 与 GetNotificationSettingsResponse 保持一致
  notifications: {
    push_enabled: boolean;    // 推送通知开关
    email_enabled: boolean;   // 邮件通知开关
    sms_enabled: boolean;     // 短信通知开关（扩展字段）
    message_notifications: boolean; // 消息通知（对应 message）
    group_notifications: boolean;   // 群组通知（对应 group_invite）
    friend_requests: boolean; // 好友请求通知（对应 friend_request）
    direct_message_toast: boolean;  // 私信弹窗提醒（扩展字段）
    group_message_toast: boolean;   // 群消息弹窗提醒（扩展字段）
    system_notification_toast: boolean; // 系统通知弹窗提醒（对应 system）
    moment_notifications: boolean;   // 生活瞬间通知（对应 moment_like）
    comment_notifications: boolean;  // 评论通知（对应 comment）
  };
  // 隐私设置 - 与 GetSettingsResponse 保持一致
  privacy: {
    privacy_level: 'public' | 'friends' | 'private'; // 隐私级别
    location_sharing: boolean; // 位置分享开关
    online_status: boolean;    // 在线状态显示（扩展字段）
    show_in_search: boolean;   // 搜索可见性（扩展字段）
  };
  // 外观设置 - 前端管理
  appearance: {
    language: string;
    animations: boolean;
    compact_mode: boolean;
    high_contrast: boolean;
  };
  // 偏好设置 - 前端管理
  preferences: {
    language: string;
    distance_unit: string;
    time_format: string;
  };
  // 账户设置 - 扩展字段
  account: {
    auto_save: boolean;
    backup_enabled: boolean;
    nickname?: string;    // 用户昵称（扩展字段）
    bio?: string;         // 个人简介（扩展字段）
    email?: string;       // 邮箱地址（扩展字段）
    phone?: string;       // 手机号码（扩展字段）
    avatar_url?: string;  // 头像URL（扩展字段）
  };
}

// 上传相关类型
export interface UploadResponse {
  file_id: string;
  file_url: string;
  file_name: string;
  file_size: number;
  file_type: string;
  upload_time: string;
}

export interface BatchUploadResponse {
  files: UploadResponse[];
  total_count: number;
  success_count: number;
  failed_count: number;
}

export interface UploadConfig {
  max_file_size: number;
  allowed_types: string[];
  max_files_per_batch: number;
}

// 基础成功响应
export interface BasicSuccess {
  success: boolean;
  message?: string;
}

// 未读消息数量
export interface UnreadCount {
  total_unread: number;
  conversation_unread: number;
  group_unread: number;
}


// 缺失的群组管理类型
export interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  is_public?: boolean;
  password?: string;
}

export interface UpdateGroupResponse {
  group_id: string;
  name: string;
  description: string;
  is_public: boolean;
  updated_at: string;
}

export interface DeleteGroupRequest {
  group_id: string;
}

export interface DeleteGroupResponse {
  success: boolean;
  message: string;
}

export interface InviteUserRequest {
  group_id: string;
  user_id: string;
}

export interface InviteUserResponse {
  success: boolean;
  message: string;
}

export interface KickMemberResponse {
  success: boolean;
  message: string;
}

export interface GroupMembersResponse {
  items: GroupMemberResponseData[];
  has_more: boolean;
  next_cursor?: string;
}

export interface GetUserGroupsResponse {
  items: GroupResponse[];
  has_more: boolean;
  next_cursor?: string;
}

// 缺失的动态相关类型
export interface UserMomentsHistoryRequest {
  user_id: string;
  limit?: number;
  cursor?: string;
}

export interface UserMomentsHistoryResponse {
  items: MomentResponse[];
  has_more: boolean;
  next_cursor?: string;
}

// ============================================================================
// 私聊管理 API 类型（api_spec.mdc 3.5节）
// ============================================================================

export interface PrivateChatLastMessage {
  id: string;
  content: string;
  type: 'text' | 'image' | 'video' | 'audio' | 'file';
  sender_id: string;
  sent_at: string;
}

export interface PrivateChatListItem {
  private_chat_id: string;
  other_user: UserInfo;
  unread_count: number;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
}

export interface PrivateChatDetailData {
  private_chat_id: string;
  other_user: UserInfo;
  other_user_status: {
    is_online: boolean;
    last_seen?: string;
  };
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  is_archived: boolean;
  last_message?: PrivateChatLastMessage;
  last_activity_at: string;
  created_at: string;
}

// ============================================================================
// 私聊消息 API 类型（api_spec.mdc 3.8节）
// ============================================================================

export interface PrivateMessageItem {
  id: string;
  type: 'text' | 'image' | 'video' | 'audio' | 'file';
  content: string;
  sender: UserInfo;
  reply_to?: string;
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_at?: string;
}

// ============================================================================
// 群聊消息 API 类型（api_spec.mdc 3.9节）
// ============================================================================

export interface GroupMessageItem {
  id: string;
  type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system';
  content: string;
  sender: UserInfo;
  reply_to?: string;
  mentions?: string[];
  sent_at: string;
  edited_at?: string;
  is_recalled: boolean;
  recalled_by?: string;
  recalled_at?: string;
  read_count?: number;
}

export interface MessageReadItem {
  user: UserInfo;
  read_at: string;
}

export interface MessageReactionItem {
  user: UserInfo;
  type: 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow';
  created_at: string;
}

// ============================================================================
// 通知中心 API 类型（api_spec.mdc 3.13节）
// ============================================================================

export interface NotificationData {
  [key: string]: any;  // 动态通知数据
}

export interface UnreadCategoryStats {
  social: number;
  system: number;
  content: number;
}

export interface InboxItemData {
  id: string;
  category: 'social' | 'system' | 'content';
  type: 'friend_request' | 'friend_accepted' | 'group_invite' | 'group_invite_response' | 'group_application' | 'group_application_response' | 'message_mention' | 'moment_like' | 'moment_comment' | 'comment_reply' | 'system_announcement' | 'system_maintenance' | 'account_warning' | 'security_alert';
  title?: string;
  content: string;
  data?: NotificationData;
  action_url?: string;
  severity?: 'info' | 'warning' | 'error';
  created_at: string;
  is_read: boolean;
}

// ============================================================================
// 信息流 API 类型（api_spec.mdc 3.17节）
// ============================================================================

export interface FeedSystemInfo {
  title: string;
  content: string;
  severity?: 'info' | 'warning' | 'error';
  action_url?: string;
  created_at: string;
}

export interface FeedItem {
  item_id: string;
  type: 'moment' | 'group_card' | 'system';
  moment?: MomentInfo;
  group?: GroupInfo;
  system?: FeedSystemInfo;
  reason: string;
  score?: number;
}

// ============================================================================
// 评论与点赞 API 类型（api_spec.mdc 3.4节）
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

// ============================================================================
// 标签管理 API 类型（api_spec.mdc 3.19节）
// ============================================================================

export interface TagData {
  name: string;
  usage_count: number;
  trend?: 'rising' | 'stable' | 'falling';
}

// ============================================================================
// 批量操作 API 类型（api_spec.mdc 3.18节）
// ============================================================================

export interface BatchItemResult {
  index: number;
  code: number;
  data?: any;
  message?: string;
}

export interface BatchResultData {
  total: number;
  succeeded: number;
  failed: number;
  results: BatchItemResult[];
}

// ============================================================================
// 排行榜 API 类型（api_spec.mdc 3.15节）
// ============================================================================

export interface GetCreatorLeaderboardRequest {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  latitude?: number;
  longitude?: number;
  radius?: number;
  limit?: number;
}

export interface CreatorRankItem {
  rank: number;
  user: UserInfo;
  score: number;
}

export interface GetCreatorLeaderboardData {
  items: CreatorRankItem[];
}

export interface GetMomentLeaderboardRequest {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  latitude?: number;
  longitude?: number;
  radius?: number;
  limit?: number;
}

export interface MomentRankItem {
  rank: number;
  moment_id: string;
  score: number;
}

export interface GetMomentLeaderboardData {
  items: MomentRankItem[];
}

export interface GetMyCreatorRankRequest {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  latitude?: number;
  longitude?: number;
  radius?: number;
}

export interface MyCreatorRankData {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  my_rank: number | null;
  my_score: number;
  total_participants: number;
  top_3: CreatorRankItem[];
}

export interface GetMyMomentRankRequest {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  latitude?: number;
  longitude?: number;
  radius?: number;
}

export interface MyMomentRankData {
  timeframe: 'daily' | 'weekly' | 'monthly' | 'all_time';
  scope?: 'nearby' | 'friends';
  my_rank: number | null;
  my_score: number;
  total_participants: number;
  top_3: MomentRankItem[];
}