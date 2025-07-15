/**
 * 用户相关API类型定义
 * 与后端 user API 严格对应
 */

import type { PaginatedResponse, Pagination } from '~/types/api'

// ==================== 认证相关 ====================

export interface RegisterRequest {
  login_id: string
  nickname?: string
  password: string
}

export interface LoginRequest {
  login_id: string
  password: string
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export interface CreateTempUserRequest {
  nickname?: string
}

export interface AuthResponse {
  user_id: number
  nickname: string
  access_token: string
  refresh_token: string
  expires_at: string
}

export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_at: string
}

// ==================== 用户信息 ====================

export interface UserInfo {
  user_id: number
  nickname: string
  last_active: string
  latitude: number
  longitude: number
  distance: number
  location_name: string
  last_active_at: string
}

// ==================== 搜索请求 ====================

export interface SearchUserByLocationRequest extends Pagination {
  latitude: number
  longitude: number
  radius: number
}

export interface SearchUserByNameRequest extends Pagination {
  keyword: string
}

export interface SearchUserByIdRequest {
  user_id: number
}

// ==================== 响应类型 ====================

export type SearchUserByLocationResponse = PaginatedResponse<UserInfo>
export type SearchUserByNameResponse = PaginatedResponse<UserInfo>
export type SearchUserByIdResponse = UserInfo

// ==================== 位置更新 ====================

export interface UpdateLocationRequest {
  latitude: number
  longitude: number
  location_name: string
}

// ==================== 密码设置 ====================

export interface SetPasswordRequest {
  old_password?: string
  new_password: string
}

// ==================== 通用响应 ====================

export interface UserActionSuccessResponse {
  success: boolean
  message?: string
}

// ==================== 通知设置相关 ====================

export interface NotificationSettings {
  message_notifications: boolean
  group_notifications: boolean
  system_notifications: boolean
  location_notifications: boolean
  quiet_hours: QuietHoursSettings
  notification_radius_km: number
  sound_enabled: boolean
  vibration_enabled: boolean
}

export interface QuietHoursSettings {
  enabled: boolean
  start_time: string // "22:00" format
  end_time: string   // "08:00" format
}

// 通知级别枚举
export enum NotificationLevel {
  All = 'all',
  Mention = 'mention',
  Important = 'important',
  Muted = 'muted'
}
