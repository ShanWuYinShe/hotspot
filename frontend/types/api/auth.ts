/**
 * 认证相关API类型定义
 */
import type { UserProfile } from '~/types/models/user' // Import UserProfile

// 登录请求
export interface LoginRequest {
  login_id: string
  password: string
}

// 登录响应
export interface LoginResponse {
  user_id: number
  nickname: string
  access_token: string
  refresh_token: string
  expires_at: string
}

// 注册请求
export interface RegisterRequest {
  login_id: string
  nickname: string
  password: string
}

// 注册响应
export interface RegisterResponse {
  user_id: number
  nickname: string
  access_token: string
  refresh_token: string
  expires_at: string
}

// 创建临时用户请求 - ensure this is defined
export interface CreateTempUserRequest {
  nickname?: string // As it was in types/user.ts
  device_id?: string // As it was in types/api/auth.ts - let's consolidate
}

// 临时用户响应
export interface TempUserResponse {
  user_id: number
  nickname: string
  access_token: string
  refresh_token: string
  expires_at: string
}

// 刷新令牌响应
export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_at: string // Changed from expires_in to expires_at to match backend
}

// 设置密码请求
export interface SetPasswordRequest {
  old_password?: string
  new_password: string
}

// 令牌验证响应
export interface TokenValidationResponse {
  valid: boolean
  user?: UserProfile // Changed from UserInfo to UserProfile
}

// 统一认证响应类型
export type AuthResponse = LoginResponse | RegisterResponse | TempUserResponse
