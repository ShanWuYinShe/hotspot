/**
 * 通用业务逻辑类型定义
 * 只包含真正通用的类型，特定领域的类型应该在对应的业务文件中定义
 */

// 地理位置信息 (通用)
export interface Location {
  latitude: number
  longitude: number
  accuracy?: number
  altitude?: number
  heading?: number
  speed?: number
}

// 地址信息
export interface Address {
  country?: string
  province?: string
  city?: string
  district?: string
  street?: string
  formatted_address?: string
}

// 地理位置详情
export interface LocationDetail extends Location {
  address?: Address
  timestamp?: string
}

// 分页参数
export interface PaginationParams {
  limit?: number
  cursor?: string | number
}

// 排序参数
export interface SortParams {
  field: string
  order: 'asc' | 'desc'
}

// 搜索参数
export interface SearchParams {
  keyword?: string
  filters?: Record<string, any>
  pagination?: PaginationParams
  sort?: SortParams
}

// 文件信息
export interface FileInfo {
  name: string
  size: number
  type: string
  url?: string
  hash?: string
}

// 上传进度
export interface UploadProgress {
  loaded: number
  total: number
  percentage: number
}

// 距离信息
export interface Distance {
  value: number // 距离值（米）
  unit: 'meter' | 'kilometer'
  formatted: string // 格式化显示
}

// 时间段
export interface TimeRange {
  start: string
  end: string
}

// 统计信息
export interface Statistics {
  total: number
  active: number
  inactive: number
  growth_rate?: number
}

// 操作结果
export interface OperationResult {
  success: boolean
  message?: string
  affected_count?: number
}

// 审计信息
export interface AuditInfo {
  created_at: string
  updated_at?: string
  created_by?: number
  updated_by?: number
}

// ==================== 任务相关 ====================

// 通知类型
export enum NotificationType {
  Message = 'message',
  System = 'system',
  Checkin = 'checkin',
  GroupInvite = 'group_invite',
  FriendRequest = 'friend_request',
  Welcome = 'welcome',
  LocationUpdate = 'location_update'
}

// 群组活动类型
export enum GroupActivityType {
  MessageSent = 'message_sent',
  MessageRead = 'message_read',
  Join = 'join',
  Leave = 'leave',
  Reaction = 'reaction',
  EventCreate = 'event_create',
  EventJoin = 'event_join',
  MediaShare = 'media_share'
}
