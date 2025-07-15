/**
 * 认证服务 - 简化版本
 * 处理用户认证相关的API请求
 */

import { useSafeApiRequest, useApiRequest } from '../core/api.service'
import type { 
  LoginRequest, 
  LoginResponse, 
  RegisterRequest, 
  RegisterResponse,
  TempUserResponse,
  RefreshTokenResponse,
  SetPasswordRequest
} from '../../types/api/auth'

// 认证相关的端点配置
const AUTH_ENDPOINTS = {
  login: '/user/login',
  register: '/user/register',
  createTempUser: '/user/temp',
  upgradeTempUser: '/user/temp/upgrade',
  refreshToken: '/user/refresh_token',
  setPassword: '/user/set_password',
  checkAuthStatus: '/user/auth/status',
} as const

/**
 * 用户登录
 */
export async function login(data: LoginRequest): Promise<LoginResponse> {
  const apiRequest = useApiRequest()
  const result = await apiRequest<LoginResponse>(AUTH_ENDPOINTS.login, data, {
    authRequired: false
  })
  if (!result) {
    throw new Error('登录失败')
  }
  return result
}

/**
 * 用户注册
 */
export async function register(data: RegisterRequest): Promise<RegisterResponse> {
  const apiRequest = useApiRequest()
  const result = await apiRequest<RegisterResponse>(AUTH_ENDPOINTS.register, data, {
    authRequired: false
  })
  if (!result) {
    throw new Error('注册失败')
  }
  return result
}

/**
 * 创建临时用户
 */
export async function createTempUser(nickname?: string): Promise<TempUserResponse> {
  const apiRequest = useApiRequest()
  const data = nickname ? { nickname } : {}
  
  const result = await apiRequest<TempUserResponse>(AUTH_ENDPOINTS.createTempUser, data, {
    authRequired: false
  })
  if (!result) {
    throw new Error('创建临时用户失败')
  }
  return result
}

/**
 * 升级临时用户为正式用户
 */
export async function upgradeTempUser(data: {
  temp_user_id: number
  login_id: string
  nickname: string
  password: string
}): Promise<LoginResponse> {
  const apiRequest = useApiRequest()
  const result = await apiRequest<LoginResponse>(AUTH_ENDPOINTS.upgradeTempUser, data, {
    authRequired: true
  })
  if (!result) {
    throw new Error('升级用户失败')
  }
  return result
}

/**
 * 刷新令牌
 */
export async function refreshToken(refreshTokenValue: string): Promise<RefreshTokenResponse> {
  const apiRequest = useApiRequest()
  const result = await apiRequest<RefreshTokenResponse>(
    AUTH_ENDPOINTS.refreshToken, 
    { refresh_token: refreshTokenValue }, 
    { authRequired: false }
  )
  if (!result) {
    throw new Error('刷新令牌失败')
  }
  return result
}

/**
 * 设置/修改密码
 */
export async function setPassword(data: SetPasswordRequest): Promise<void> {
  const apiRequest = useApiRequest()
  const result = await apiRequest<void>(AUTH_ENDPOINTS.setPassword, data)
  if (result === null) {
    throw new Error('设置密码失败')
  }
}

/**
 * 检查认证状态
 */
export async function checkAuthStatus(): Promise<any> {
  const safeRequest = useSafeApiRequest()
  return safeRequest<any>(AUTH_ENDPOINTS.checkAuthStatus, {})
}
