/**
 * 通用数据类型定义
 * 包含项目中使用的通用类型和枚举
 * 注意：基础API类型已迁移到 BackendApiTypes.ets
 */

// 地理坐标
export interface Coordinates {
  latitude: number;         // 纬度，范围：[-90, 90]
  longitude: number;        // 经度，范围：[-180, 180]
}

// 分页参数
export interface PaginationParams {
  cursor?: string;          // 游标分页
  limit?: number;           // 每页数量，范围：1-100，默认20
}

// 消息类型枚举
export enum MessageType {
  TEXT = 'text',
  IMAGE = 'image',
  VOICE = 'voice',
  VIDEO = 'video',
  LOCATION = 'location',
  FILE = 'file'
}

// 房间类型枚举
export enum RoomType {
  GROUP = 'group',
  CONVERSATION = 'conversation'
}

// 通知类型枚举
export enum NotificationType {
  FRIEND_REQUEST = 'friend_request',
  GROUP_INVITATION = 'group_invitation',
  SYSTEM = 'system',
  ANNOUNCEMENT = 'announcement',
  NEW_MOMENT = 'new_moment',
  GROUP_DIGEST = 'group_digest'
}

// 群组事件类型
export enum GroupEventType {
  MEMBER_JOINED = 'member_joined',
  MEMBER_LEFT = 'member_left',
  MEMBER_KICKED = 'member_kicked',
  OWNER_TRANSFERRED = 'owner_transferred',
  GROUP_UPDATED = 'group_updated'
}

// 用户角色
export enum UserRole {
  OWNER = 'owner',
  ADMIN = 'admin',
  MEMBER = 'member'
}

// 通知级别
export enum NotificationLevel {
  ALL = 'all',
  MENTION = 'mention',
  IMPORTANT = 'important',
  MUTED = 'muted'
}

// 密码状态结果
export interface PasswordStatusResult {
  has_password: boolean;    // 是否已设置密码
}

// 错误响应详情
export interface ErrorDetail {
  field?: string;          // 错误字段
  message: string;         // 错误消息
  code?: string;          // 错误代码
}

// 文件上传响应
export interface FileUploadResponse {
  file_id: string;        // 文件ID
  file_url: string;       // 文件访问URL
  file_name: string;      // 文件名
  file_size: number;      // 文件大小（字节）
  mime_type: string;      // MIME类型
  upload_time: string;    // 上传时间（ISO 8601）
}

// 系统配置
export interface SystemConfig {
  max_file_size: number;      // 最大文件大小（字节）
  max_image_count: number;    // 最大图片数量
  supported_formats: string[]; // 支持的文件格式
  api_version: string;        // API版本
}

// 应用状态
export interface AppState {
  is_online: boolean;         // 是否在线
  network_type: string;       // 网络类型
  location_enabled: boolean;  // 位置服务是否启用
  notification_enabled: boolean; // 通知是否启用
}

// 设备信息
export interface DeviceInfo {
  device_id: string;         // 设备ID
  device_type: string;       // 设备类型
  os_version: string;        // 操作系统版本
  app_version: string;       // 应用版本
  screen_width: number;      // 屏幕宽度
  screen_height: number;     // 屏幕高度
  screen_density: number;    // 屏幕密度
}