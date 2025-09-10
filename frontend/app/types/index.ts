/**
 * API 类型定义 - 与后端完全对齐
 * 基于后端 Rust API 模型生成
 */

// DateTime类型使用string表示（ISO 8601格式）
type DateTime = string;

// 基于 hotspot.md 中的枚举定义
export type MessageType = 'text' | 'image' | 'voice' | 'video' | 'location' | 'file';
export type RoomType = 'group' | 'conversation';

// 基础成功响应类型
export interface BasicSuccess {
  success: boolean;
}

// 密码状态响应类型
export interface PasswordStatusResult {
  has_password: boolean;
}

// ============ 基础响应类型 ============
export interface ApiResponse<T = any> {
  code: number; // 0表示成功，非0表示失败
  msg: string; // 状态消息 (后端使用 "msg" 字段)
  data: T | null; // 成功时为数据对象，失败时为null
}

// ============ 分页相关类型 ============
// 基于 hotspot.md 中的 PaginationParams 和 PaginatedResponse<T> 定义
export interface PaginationParams {
  cursor?: string; // 游标分页，字符串类型（后端64位整数转换）
  limit?: number; // 1-100，默认20
}

export interface PaginationMeta {
  has_more: boolean; // 是否有更多数据
  next_cursor?: string; // 下一页游标（后端64位整数转换为字符串）
}

export interface PaginatedResponse<T> {
  items: T[];
  has_more: boolean;
  next_cursor?: string;
}

// ============ 用户相关类型 ============
export interface AuthUser {
  user_id: string;
  nickname: string;
}

// 🔥 获取 nonce 请求和响应（密码安全机制）
export interface GetNonceRequest {
  login_id: string; // 邮箱/手机号，用于生成专用nonce
}

export interface NonceResponse {
  nonce_id: string;    // nonce唯一标识符（UUID）
  nonce: string;       // 32位随机字符串
  timestamp: number;   // 服务器时间戳（毫秒）
  expires_at: number;  // nonce过期时间戳（毫秒）
}

// 🔥 更新：使用安全的密码哈希机制
export interface RegisterRequest {
  login_id: string;        // 邮箱/手机号，必填，唯一，长度1-100
  nickname: string;        // 必填，长度1-32
  password_hash: string;   // SHA256(password + nonce + timestamp + login_id)
  nonce_id: string;        // 从 get_nonce 接口获取的nonce标识符
  timestamp: number;       // 从 get_nonce 接口获取的时间戳
}

export interface LoginRequest {
  login_id: string;        // 邮箱/手机号，必填
  password_hash: string;   // SHA256(password + nonce + timestamp + login_id)
  nonce_id: string;        // 从 get_nonce 接口获取的nonce标识符
  timestamp: number;       // 从 get_nonce 接口获取的时间戳
}

export interface RefreshTokenRequest {
  refresh_token: string; // 刷新令牌，必填
}

export interface CreateTempUserRequest {
  nickname?: string; // 可选，长度1-32，默认"临时用户"
}

// 基于 hotspot.md 中的 AuthResponse 定义
export interface AuthResponse {
  user_id: string; // 64位整数转换为字符串
  nickname: string;
  access_token: string;
  refresh_token: string;
  expires_at: string; // ISO 8601 datetime string
  is_temp?: boolean; // 是否为临时用户
}

export interface RefreshTokenResponse {
  access_token: string;
  refresh_token: string;
  expires_at: string;
}

// 前端 User 类型 - 不包含敏感信息如 login_id
export interface User {
  user_id: string; // 64位整数转换为字符串
  nickname: string;
  avatar_url?: string;
  email?: string; // 用于显示，不是登录凭据
  bio?: string;
  is_temp?: boolean;
  created_at?: string;
  last_active_at?: string;
  latitude?: number; // 地理坐标，64位浮点数转换为number
  longitude?: number; // 地理坐标，64位浮点数转换为number
  location_name?: string;
  distance?: number; // 距离，64位浮点数转换为number
}

// 兼容性类型别名
export type UserInfo = User;

export interface SearchUserByLocationRequest {
  latitude: number; // 纬度，-90~90
  longitude: number; // 经度，-180~180
  radius?: number; // 搜索半径，单位米，建议100~10000
  radius_meters?: number; // 兼容后端字段名
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface SearchUserByNameRequest {
  keyword: string; // 搜索关键词，长度1-50
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface SearchUserByIdRequest {
  user_id: string;
}

export interface UpdateLocationRequest {
  latitude: number; // 纬度，-90~90
  longitude: number; // 经度，-180~180
  location_name: string; // 位置名称，长度1-100
}

// 🔥 更新：设置密码请求（使用安全哈希）
export interface SetPasswordRequest {
  new_password_hash: string;    // SHA256(new_password + nonce + timestamp + user_id)
  old_password_hash?: string;   // SHA256(old_password + nonce + timestamp + user_id)，首次设置可为空
  nonce_id: string;            // 从 get_nonce 接口获取的nonce标识符
  timestamp: number;           // 从 get_nonce 接口获取的时间戳
}

export interface UpdateNicknameRequest {
  nickname: string; // 用户昵称，长度1-32
}

// 🔥 更新：账户升级请求（使用安全哈希）
export interface UpgradeAccountRequest {
  login_id: string;        // 邮箱/手机号，唯一，长度1-100
  password_hash: string;   // SHA256(password + nonce + timestamp + login_id)
  nonce_id: string;        // 从 get_nonce 接口获取的nonce标识符
  timestamp: number;       // 从 get_nonce 接口获取的时间戳
}

export interface UserSettingsRequest {
  notification_level?: string;
  message_notifications?: boolean;
  group_notifications?: boolean;
  system_notifications?: boolean;
  location_notifications?: boolean;
  quiet_hours_enabled?: boolean;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  notification_radius_km?: number;
  sound_enabled?: boolean;
  vibration_enabled?: boolean;
}

// 用户设置响应类型
export interface UserSettings {
  notifications: {
    push_enabled: boolean;
    email_enabled: boolean;
    sms_enabled: boolean;
    group_notifications: boolean;
    message_notifications: boolean;
    friend_requests: boolean;
    // 新增细粒度消息提示设置
    direct_message_toast: boolean; // 私聊消息Toast提示
    group_message_toast: boolean; // 群组消息Toast提示
    system_notification_toast: boolean; // 系统通知Toast提示
    error_notifications: boolean; // 错误提示开关
    success_notifications: boolean; // 成功提示开关
  };
  privacy: {
    profile_visibility: "public" | "friends" | "private";
    location_sharing: boolean;
    online_status: boolean;
    show_in_search: boolean;
  };
  preferences: {
    language: string;
    theme: "light" | "dark" | "auto";
    distance_unit: "km" | "miles";
    time_format: "12h" | "24h";
  };
  account: {
    nickname: string;
    bio?: string;
    email?: string;
    phone?: string;
    avatar_url?: string;
  };
}

export interface PasswordStatusResponse {
  is_password_set: boolean;
}

// 类型别名
export type SearchUserByLocationResponse = PaginatedResponse<UserInfo>;
export type SearchUserByNameResponse = PaginatedResponse<UserInfo> & { users?: UserInfo[] };
export type SearchUserByIdResponse = UserInfo;

// ============ 群组相关类型 ============
export interface CreateGroupRequest {
  name: string;
  description?: string;
  password?: string;
  latitude: number;
  longitude: number;
  location_name: string;
}

export interface CreateGroupResponse {
  group_id: string; // UUID as string
}

export interface JoinGroupRequest {
  group_id: string; // UUID as string
  password?: string;
}

export interface JoinGroupResponse {
  success: boolean;
  role: string;
}

export interface LeaveGroupRequest {
  group_id: string;
}

export interface LeaveGroupResponse {
  success: boolean;
}

export interface KickMemberRequest {
  group_id: string;
  user_id: string;
}

export interface KickMemberResponse {
  success: boolean;
}

export interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  avatar_url?: string;
}

export interface UpdateGroupResponse {
  success: boolean;
}

export interface DeleteGroupRequest {
  group_id: string;
}

export interface DeleteGroupResponse {
  success: boolean;
}

export interface InviteUserRequest {
  group_id: string;
  user_id: string;
}

export interface InviteUserResponse {
  success: boolean;
}

export interface GetGroupRequest {
  group_id: string;
}

// 基于 hotspot.md 中的 Group 定义
export interface Group {
  group_id: string; // UUID
  name: string;
  description?: string;
  owner_id: string; // 64位整数转换为字符串
  creator_name: string;
  created_at: string;
  last_active_at: string;
  latitude: number; // 地理坐标，64位浮点数转换为number
  longitude: number; // 地理坐标，64位浮点数转换为number
  member_count: number; // 32位整数，使用number类型
  distance?: number; // 距离，64位浮点数转换为number
  location_name: string;
  is_password_required: boolean;
  is_member: boolean;
  user_role: string;
  avatar_url?: string;
}

// 兼容性类型别名
export type GroupInfo = Group;

export interface GetGroupResponse {
  group: GroupInfo;
}

export interface GroupMembersRequest {
  group_id: string;
  cursor?: string; // 游标分页，字符串类型
  limit?: number;
}

export interface GroupMemberInfo {
  id: string;
  user_id: string;
  nickname: string;
  last_active_at: string; // DateTime<Utc>
  role: string;
  join_time: string; // DateTime<Utc>
  avatar_url?: string; // 前端展示友好（若后端无此字段则不使用）
}

export interface SearchGroupByLocationRequest {
  latitude: number;
  longitude: number;
  radius: number;
  cursor?: string; // 游标分页，字符串类型
  limit?: number;
}

export interface SearchGroupByNameRequest {
  keyword: string;
  cursor?: string; // 游标分页，字符串类型
  limit?: number;
}

export interface SearchGroupByIdRequest {
  group_id: string;
}

export interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}

export interface TransferOwnershipResponse {
  success: boolean;
}

export interface GetUserGroupsRequest {
  limit?: number;
  offset?: number;
}

export interface GetOwnedGroupsRequest {
  limit?: number;
  offset?: number;
}

// 类型别名
export type GroupMembersResponse = PaginatedResponse<GroupMemberInfo>;
export type SearchGroupByLocationResponse = PaginatedResponse<GroupInfo>;
export type SearchGroupByNameResponse = PaginatedResponse<GroupInfo>;
export type SearchGroupByIdResponse = GroupInfo;
export type GetUserGroupsResponse = PaginatedResponse<GroupInfo>;
export type GetOwnedGroupsResponse = PaginatedResponse<GroupInfo>;

// ============ 消息相关类型 ============
export interface SendGroupMessageRequest {
  group_id: string;     // 群组ID（UUID转字符串）
  content: string;      // 消息内容，1-2000字符
  message_type?: MessageType; // 消息类型，默认'text'
}

export interface SendGroupMessageResponse {
  group_id: string;
  message_id: string;
  message_type: string;
  sent_at: string; // DateTime<Utc>
}

export interface GroupMessageHistoryRequest {
  group_id: string;
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface SendDirectMessageRequest {
  recipient_id: string; // 接收者ID（64位整数转字符串）
  content: string;      // 消息内容，1-2000字符
  message_type?: MessageType; // 消息类型，默认'text'
}

export interface SendDirectMessageResponse {
  message_id: string;
  conversation_id: string;
  recipient_id: string;
  message_type: string;
  sent_at: string; // DateTime<Utc>
}

export interface DirectMessageHistoryRequest {
  conversation_id?: string;
  user_id?: string;
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface DeleteGroupMessageRequest {
  message_id: string;
}

export interface DeleteDirectMessageRequest {
  message_id: string;
}

export interface MarkReadRequest {
  conversation_id: string; // 会话ID，必需
  message_id?: string; // 消息ID，可选
}

export interface MarkReadResponse {
  success: boolean;
  marked_count?: number;
}

export interface GetGroupUnreadCountRequest {
  group_id: string;
}

export interface UnreadCountResponse {
  unread_count: number; // 32位整数，使用number类型
}

// 基于 hotspot.md 中的 Message 定义
export interface Message {
  message_id: string; // UUID
  sender_id: string; // 64位整数转换为字符串
  sender_name: string;
  content: string;
  message_type: 'text' | 'image' | 'location'; // MessageType 枚举
  sent_at: string;
  group_id?: string; // UUID，群组消息时存在
  conversation_id?: string; // UUID，私聊消息时存在
  latitude?: number; // 地理坐标，64位浮点数转换为number
  longitude?: number; // 地理坐标，64位浮点数转换为number
}

// 兼容性类型别名
export type DirectMessageHistoryItem = Message;
export type GroupMessageHistoryItem = Message;

export interface DeleteMessageResponse {
  success: boolean;
}

export interface ConversationsRequest {
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

// 基于 hotspot.md 中的 ConversationSummary 定义
export interface ConversationSummary {
  conversation_id: string; // UUID
  peer_user_id: string; // 64位整数转换为字符串
  peer_nickname: string;
  last_message_preview: string;
  last_message_at: string;
  unread_count: number; // 32位整数，使用number类型
}

// 兼容性类型别名
export type ConversationItem = ConversationSummary;

// 类型别名
export type GroupMessageHistoryResponse =
  PaginatedResponse<GroupMessageHistoryItem>;
export type DirectMessageHistoryResponse =
  PaginatedResponse<DirectMessageHistoryItem>;
export type ConversationsResponse = PaginatedResponse<ConversationItem>;

// ============ 动态相关类型 ============
export interface CreateMomentRequest {
  latitude: number;     // 纬度，-90~90
  longitude: number;    // 经度，-180~180
  description?: string; // 瞬间描述，≤500字符
  tags?: string[];      // 标签数组，单个标签1-20字符，总数≤10
  image_urls?: string[]; // 图片URL数组，总数≤9
}

export interface CreateMomentResponse {
  moment_id: string;
}

export interface DeleteMomentRequest {
  moment_id: string;
}

export interface DeleteMomentResponse {
  success: boolean;
}

export interface SearchMomentByIdRequest {
  moment_id: string;
}

export interface SearchMomentsByLocationRequest {
  latitude: number;
  longitude: number;
  radius: number;
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface UserMomentsHistoryRequest {
  user_id?: string;
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface SearchMomentsByTagsRequest {
  tags: string[];
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: string; // 游标分页，字符串类型
}

export interface LikeMomentRequest {
  moment_id: string;
}

export interface LikeMomentResponse {
  moment_id: string;
  likes_count: number; // 32位整数，使用number类型
  liked_by_me: boolean;
}

export interface UnlikeMomentRequest {
  moment_id: string;
}

export interface UnlikeMomentResponse {
  moment_id: string;
  likes_count: number; // 32位整数，使用number类型
  liked_by_me: boolean;
}

// 基于 hotspot.md 中的 Moment 定义
export interface Moment {
  moment_id: string; // UUID
  user_id: string; // 64位整数转换为字符串
  nickname: string;
  user_avatar?: string;
  description: string;
  latitude: number; // 地理坐标，64位浮点数转换为number
  longitude: number; // 地理坐标，64位浮点数转换为number
  location_name: string;
  created_at: string;
  likes_count: number; // 32位整数，使用number类型
  comments_count: number; // 32位整数，使用number类型
  liked_by_me: boolean;
  distance?: number; // 距离，64位浮点数转换为number
  tags: string[];
}

// 兼容性类型别名
export type MomentInfo = Moment;

// 类型别名
export type UserMomentsHistoryResponse = PaginatedResponse<MomentInfo>;
export type SearchMomentsByLocationResponse = PaginatedResponse<MomentInfo>;
export type SearchMomentsByTagsResponse = PaginatedResponse<MomentInfo>;

// ============ 位置相关类型 ============
export interface Location {
  latitude: number;
  longitude: number;
  name?: string;
  timestamp?: string;
}

// ============ 错误相关类型 ============
export type ErrorSeverity = "low" | "medium" | "high" | "critical";

export type ErrorCategory =
  | "network"
  | "auth"
  | "validation"
  | "business"
  | "system"
  | "permission";

export type ErrorRecoveryStrategy =
  | "none"
  | "retry"
  | "refresh"
  | "redirect"
  | "fallback";

export interface BaseError {
  id: string;
  code: string;
  message: string;
  category: ErrorCategory;
  severity: ErrorSeverity;
  timestamp: string;
  context?: Record<string, any>;
  cause?: Error;
  recovery?: ErrorRecoveryStrategy;
  retryable?: boolean;
  userAction?: string;
}

export interface ApiError extends BaseError {
  category: "network";
  status: number;
  endpoint?: string;
  method?: string;
  requestId?: string;
  details?: any;
}

export interface NetworkError extends BaseError {
  category: "network";
  networkCode?: string;
  timeout?: boolean;
  offline?: boolean;
  details?: any;
}

export interface AuthError extends BaseError {
  category: "auth";
  authCode:
    | "AUTH_REQUIRED"
    | "TOKEN_EXPIRED"
    | "INVALID_TOKEN"
    | "PERMISSION_DENIED"
    | "ACCOUNT_LOCKED";
  accounts?: Array<{ user_id: string; nickname: string; avatar_url?: string }>;
  redirectUrl?: string;
  details?: any;
}

export interface ValidationError extends BaseError {
  category: "validation";
  field?: string;
  value?: any;
  constraints?: string[];
  details?: any;
}

export interface BusinessError extends BaseError {
  category: "business";
  businessCode: string;
  data?: any;
  details?: any;
}

export interface SystemError extends BaseError {
  category: "system";
  systemCode?: string;
  stack?: string;
  details?: any;
}

export interface PermissionError extends BaseError {
  category: "permission";
  resource?: string;
  action?: string;
  requiredRole?: string;
  details?: any;
}

export type AppError =
  | ApiError
  | NetworkError
  | AuthError
  | ValidationError
  | BusinessError
  | SystemError
  | PermissionError;

export interface ErrorHandleResult {
  handled: boolean;
  retry?: boolean;
  redirect?: string;
  fallback?: any;
  userNotified?: boolean;
}

export interface ErrorReport {
  error: AppError;
  userAgent: string;
  url: string;
  userId?: string;
  sessionId?: string;
  additionalInfo?: Record<string, any>;
}

// ============ 加密相关类型 ============
export interface EncryptedCredential {
  version: number;
  algo: string;
  kdf: string;
  rounds: number;
  salt_b64: string;
  iv_b64: string;
  ciphertext_b64: string;
}

// ============ 消息相关类型 ============
// 基于 hotspot.md 中的 Message 定义
export interface ApiMessage {
  message_id: string; // UUID 转换为字符串
  sender_id: string; // 64位整数转换为字符串
  sender_name: string;
  content: string;
  message_type: MessageType; // 'text'|'image'|'voice'|'video'|'location'|'file'
  sent_at: string; // ISO 8601 datetime string
  group_id?: string; // UUID 转换为字符串
  conversation_id?: string; // UUID 转换为字符串
  latitude?: number;
  longitude?: number;
}

// 会话概要
export interface ConversationSummary {
  conversation_id: string; // UUID 转换为字符串
  peer_user_id: string; // 64位整数转换为字符串
  peer_nickname: string;
  last_message_preview: string;
  last_message_at: string; // ISO 8601 datetime string
  unread_count: number;
}

// 未读计数模型
export interface UnreadCount {
  group_id?: string; // UUID 转换为字符串（群组未读）
  conversation_id?: string; // UUID 转换为字符串（私聊未读）
  unread_count?: number; // 具体未读数
  total_unread_count?: number; // 汇总未读数
}

// ============ 健康检查类型 ============
export interface HealthCheckResponse {
  status: string;
  version: string;
  database: string;
  redis: string;
  message_queue: string;
}
