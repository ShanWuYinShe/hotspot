/**
 * 用户业务类型定义
 * 基于后端API响应的用户信息结构
 */

import type { AuditInfo, Location } from './common'

// 基础用户信息（来自后端API）
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

// 用户配置文件（扩展基础信息）
export interface UserProfile extends UserInfo {
  login_id?: string // 敏感信息，通常不在公开配置中
  email?: string
  avatar_url?: string
  bio?: string
  is_temp?: boolean
  created_at?: string
  updated_at?: string
}

// 用户活动状态
export interface UserActivityStatus {
  user_id: number
  is_online: boolean
  last_active_at: string
  current_location?: Location & { name?: string; timestamp?: string }
}

// 用户列表项（用于搜索结果、附近用户等）
export interface UserListItem extends UserInfo {
  is_friend?: boolean
  common_groups_count?: number
  avatar_url?: string
}

// 用户详细信息
export interface UserDetail extends UserProfile, Omit<UserActivityStatus, 'user_id'> {
  settings?: UserSettings
}

// 用户设置
export interface UserSettings {
  user_id: number
  notifications_enabled: boolean
  theme: 'light' | 'dark' | 'system'
  language: string
  location_sharing_enabled: boolean
}

// 地图上的用户位置标记
export interface UserLocationPin {
  user_id: number
  nickname: string
  avatar_url?: string
  latitude: number
  longitude: number
  timestamp: string
}
