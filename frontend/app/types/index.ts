/**
 * API 类型定义 - 与后端完全对齐
 * 基于后端 Rust API 模型生成
 */

// DateTime类型使用string表示（ISO 8601格式）
type DateTime = string;

// ============ 基础响应类型 ============
export interface ApiResponse<T = any> {
  code: number; // 0表示成功，非0表示失败
  msg: string; // 状态消息
  data: T | null; // 成功时为数据对象，失败时为null
}

// ============ 分页相关类型 ============
export interface Pagination {
  limit?: number; // 每页限制数量，默认20，最大50
  cursor?: number; // 分页游标
}

export interface PaginationMeta {
  has_more: boolean; // 是否有更多数据
  next_cursor?: number; // 下一页游标
}

export interface PaginatedResponse<T> {
  items: T[];
  has_more: boolean;
  next_cursor?: number;
}

// ============ 用户相关类型 ============
export interface AuthUser {
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
}

export interface RegisterRequest {
  login_id: string; // 邮箱/手机号，必填，唯一，长度1-100
  nickname?: string; // 可选，长度1-32，默认"用户"
  password: string; // 必填，长度8-64
}

export interface LoginRequest {
  login_id: string; // 邮箱/手机号，必填
  password: string; // 必填，长度8-64
}

export interface RefreshTokenRequest {
  refresh_token: string; // 刷新令牌，必填
}

export interface CreateTempUserRequest {
  nickname?: string; // 可选，长度1-32，默认"临时用户"
}

export interface AuthResponse {
  user_id: string; // 修正：使用 string 避免大整数精度问题
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

export interface UserInfo {
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
  last_active: string; // DateTime<Utc>
  latitude: number;
  longitude: number;
  distance: number;
  location_name: string;
  last_active_at: string; // DateTime<Utc>
  // 可选扩展字段，用于前端展示
  avatar_url?: string;
  bio?: string;
  created_at?: string;
  friendship_status?: 'none' | 'pending_sent' | 'pending_received' | 'accepted';
  requesting?: boolean; // 前端临时状态
}

export interface SearchUserByLocationRequest {
  latitude: number; // 纬度，-90~90
  longitude: number; // 经度，-180~180
  radius?: number; // 搜索半径，单位米，建议100~10000
  radius_meters?: number; // 兼容后端字段名
  limit?: number;
  cursor?: number;
}

export interface SearchUserByNameRequest {
  keyword: string; // 搜索关键词，长度1-50
  limit?: number;
  cursor?: number;
}

export interface SearchUserByIdRequest {
  user_id: string; // 用户ID，使用 string 避免大整数精度问题
}

export interface UpdateLocationRequest {
  latitude: number; // 纬度，-90~90
  longitude: number; // 经度，-180~180
  location_name: string; // 位置名称，长度1-100
}

export interface SetPasswordRequest {
  old_password?: string; // 原密码，首次设置可为空
  new_password: string; // 新密码，长度8-64
}

export interface UpdateNicknameRequest {
  nickname: string; // 用户昵称，长度1-32
}

export interface UpgradeAccountRequest {
  login_id: string; // 邮箱/手机号，唯一，长度1-100
  password: string; // 密码，长度8-64
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
  user_id: string; // 修正：使用 string 避免大整数精度问题
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

export interface GetGroupRequest {
  group_id: string;
}

export interface GroupInfo {
  group_id: string; // UUID as string
  name: string;
  description?: string;
  avatar_url?: string;
  owner_id: string; // 修正：使用 string 避免大整数精度问题
  creator_name: string;
  created_at: string; // DateTime<Utc>
  last_active_at: string; // DateTime<Utc>
  latitude: number;
  longitude: number;
  member_count: number;
  distance: number;
  location_name: string;
  is_password_required: boolean;
  is_member: boolean;
  user_role: string;
}

export interface GetGroupResponse {
  group: GroupInfo;
}

export interface GroupMembersRequest {
  group_id: string;
  cursor?: number;
  limit?: number;
}

export interface GroupMemberInfo {
  id: string; // 修正：使用 string 避免大整数精度问题
  user_id: string; // 修正：使用 string 避免大整数精度问题
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
  cursor?: number;
  limit?: number;
}

export interface SearchGroupByNameRequest {
  keyword: string;
  cursor?: number;
  limit?: number;
}

export interface SearchGroupByIdRequest {
  group_id: string;
}

export interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string; // 修正：使用 string 避免大整数精度问题
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
  group_id: string;
  content: string;
  message_type: string; // 默认"text"，可以是"image"等
  latitude: number;
  longitude: number;
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
  cursor?: number;
}

export interface SendDirectMessageRequest {
  recipient_id: string; // 修正：使用 string 避免大整数精度问题
  content: string;
  message_type: string; // 默认"text"，可以是"image"等
  latitude: number;
  longitude: number;
}

export interface SendDirectMessageResponse {
  message_id: string;
  conversation_id: string;
  recipient_id: string; // 修正：使用 string 避免大整数精度问题
  message_type: string;
  sent_at: string; // DateTime<Utc>
}

export interface DirectMessageHistoryRequest {
  conversation_id?: string;
  user_id?: string; // 修正：使用 string 避免大整数精度问题
  limit?: number;
  cursor?: number;
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
  unread_count: number;
}

export interface DirectMessageHistoryItem {
  message_id: string;
  conversation_id: string;
  sender_id: string; // 修正：使用 string 避免大整数精度问题
  sender_name: string;
  recipient_id: string; // 修正：使用 string 避免大整数精度问题
  content: string;
  message_type: string;
  sent_at: string; // DateTime<Utc>
  read_at?: string; // DateTime<Utc>
  latitude: number;
  longitude: number;
}

export interface GroupMessageHistoryItem {
  message_id: string;
  group_id: string;
  sender_id: string; // 修正：使用 string 避免大整数精度问题
  sender_name: string;
  content: string;
  message_type: string;
  sent_at: string; // DateTime<Utc>
  latitude: number;
  longitude: number;
}

export interface DeleteMessageResponse {
  success: boolean;
}

export interface ConversationsRequest {
  limit?: number;
  cursor?: number;
}

export interface ConversationItem {
  conversation_id: string;
  peer_id: string; // 修正：使用 string 避免大整数精度问题
  peer_name: string;
  last_message_preview: string;
  last_message_time: string; // DateTime<Utc>
  unread_count: number;
}

// 类型别名
export type GroupMessageHistoryResponse =
  PaginatedResponse<GroupMessageHistoryItem>;
export type DirectMessageHistoryResponse =
  PaginatedResponse<DirectMessageHistoryItem>;
export type ConversationsResponse = PaginatedResponse<ConversationItem>;

// ============ 动态相关类型 ============
export interface CreateMomentRequest {
  description: string;
  latitude: number;
  longitude: number;
  location_name: string;
  tags?: string[];
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
  cursor?: number;
}

export interface UserMomentsHistoryRequest {
  user_id?: string; // 修正：使用 string 避免大整数精度问题
  limit?: number;
  cursor?: number;
}

export interface SearchMomentsByTagsRequest {
  tags: string[];
  latitude: number;
  longitude: number;
  limit?: number;
  cursor?: number;
}

export interface LikeMomentRequest {
  moment_id: string;
}

export interface LikeMomentResponse {
  success: boolean;
  likes_count: number;
}

export interface UnlikeMomentRequest {
  moment_id: string;
}

export interface UnlikeMomentResponse {
  success: boolean;
  likes_count: number;
}

export interface MomentInfo {
  id: string; // 修正：使用 string 避免大整数精度问题
  moment_id: string;
  user_id: string; // 修正：使用 string 避免大整数精度问题
  nickname: string;
  description?: string;
  latitude: number;
  longitude: number;
  location_name: string;
  distance: number;
  created_at: string; // DateTime<Utc>
  tags: string[];
  likes_count: number;
  comments_count: number;
  liked_by_me: boolean;
}

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

// ============ 健康检查类型 ============
export interface HealthCheckResponse {
  status: string;
  version: string;
  database: string;
  redis: string;
  message_queue: string;
}
