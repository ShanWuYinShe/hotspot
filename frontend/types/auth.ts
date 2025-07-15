/**
 * 认证相关类型定义
 */

// 基础用户信息
export interface User {
  user_id: number;
  login_id: string;
  nickname: string;
  avatar_url?: string;
  email?: string;
  created_at: string;
  updated_at: string;
}

// 认证状态
export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  expiresAt: number | null;
}

// 登录凭据
export interface LoginCredentials {
  login_id: string;
  password: string;
}

// 注册信息
export interface RegisterInfo {
  login_id: string;
  nickname: string;
  password: string;
  email?: string;
}

// 令牌信息
export interface TokenInfo {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

// 权限信息
export interface Permission {
  id: string;
  name: string;
  description?: string;
}

// 角色信息
export interface Role {
  id: string;
  name: string;
  permissions: Permission[];
}

// 用户会话
export interface UserSession {
  sessionId: string;
  userId: number;
  deviceId?: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  lastActiveAt: string;
  expiresAt: string;
}
