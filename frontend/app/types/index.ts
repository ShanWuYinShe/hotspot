/**
 * 核心类型定义
 */

// 用户类型
export interface User {
  user_id: number
  login_id?: string
  nickname: string
  avatar_url?: string
  email?: string
  bio?: string
  is_temp?: boolean
  created_at?: string
  last_active_at?: string
  latitude?: number
  longitude?: number
  location_name?: string
  distance?: number
}

// 群组类型
export interface Group {
  group_id: string
  name: string
  description?: string
  owner_id: number
  creator_name: string
  created_at: string
  last_active_at: string
  latitude: number
  longitude: number
  member_count: number
  distance?: number
  location_name: string
  is_password_required: boolean
  is_member: boolean
  user_role: string
  avatar_url?: string
}

// 消息类型
export interface Message {
  message_id: string
  sender_id: number
  sender_name: string
  content: string
  message_type: 'text' | 'image' | 'location'
  sent_at: string
  group_id?: string
  conversation_id?: string
  latitude?: number
  longitude?: number
}

// 认证响应类型
export interface AuthResponse {
  user_id: number
  login_id?: string
  nickname: string
  avatar_url?: string
  email?: string
  access_token: string
  refresh_token: string
  expires_at: string
}

// API响应类型
export interface ApiResponse<T = any> {
  data: T
  message?: string
  success: boolean
}

// 分页响应类型
export interface PaginatedResponse<T = any> {
  items: T[]
  has_more: boolean
  next_cursor?: number
}

// 认证响应类型
export interface AuthResponse {
  user_id: number
  login_id?: string
  nickname: string
  avatar_url?: string
  email?: string
  access_token: string
  is_temp?: boolean
}

// 位置类型
export interface Location {
  latitude: number
  longitude: number
  name?: string
  timestamp?: string
}
