import { BaseService } from './base.service'
import type {
  AuthResponse,
  RegisterRequest,
  LoginRequest,
  CreateTempRequest,
  RefreshTokenRequest,
  SetPasswordRequest,
  SetPasswordResponse,
  RequestPasswordResetRequest,
  RequestPasswordResetResponse,
  VerifyResetCodeRequest,
  VerifyResetCodeResponse,
  ResetPasswordRequest,
  CheckStatusResponse,
  ListSessionsRequest,
  ListSessionsResponse,
  RevokeSessionRequest,
  SessionItem,
  PaginatedData
} from '~/types/api'

/**
 * 认证服务
 * 根据 api_spec.mdc 3.1 认证 API 实现
 */

export class AuthService extends BaseService {
  protected readonly baseUrl = '/api/v1'

  /**
   * 3.1.1 用户注册
   */
  async register(data: RegisterRequest): Promise<AuthResponse> {
    return this.publicPost<AuthResponse>('/public/auth/register', data)
  }

  /**
   * 3.1.2 用户登录
   */
  async login(data: LoginRequest): Promise<AuthResponse> {
    return this.publicPost<AuthResponse>('/public/auth/login', data)
  }

  /**
   * 3.1.3 用户登出
   */
  async logout(): Promise<void> {
    return this.post<void>('/protected/auth/logout', {})
  }

  /**
   * 3.1.4 创建临时用户
   */
  async createTempUser(data: CreateTempRequest): Promise<AuthResponse> {
    return this.publicPost<AuthResponse>('/public/auth/create_temp_user', data)
  }

  /**
   * 3.1.5 刷新Token
   */
  async refreshToken(data: RefreshTokenRequest): Promise<AuthResponse> {
    return this.post<AuthResponse>('/protected/auth/refresh_token', data)
  }

  /**
   * 3.1.6 检查认证状态
   */
  async checkStatus(): Promise<CheckStatusResponse> {
    return this.post<CheckStatusResponse>('/protected/auth/check_status', {})
  }

  /**
   * 3.1.7 设置密码
   */
  async setPassword(data: SetPasswordRequest): Promise<SetPasswordResponse> {
    return this.post<SetPasswordResponse>('/protected/auth/set_password', data)
  }

  /**
   * 3.1.8 请求密码重置
   */
  async requestPasswordReset(data: RequestPasswordResetRequest): Promise<RequestPasswordResetResponse> {
    return this.publicPost<RequestPasswordResetResponse>('/public/auth/request_password_reset', data)
  }

  /**
   * 3.1.9 验证重置码
   */
  async verifyResetCode(data: VerifyResetCodeRequest): Promise<VerifyResetCodeResponse> {
    return this.publicPost<VerifyResetCodeResponse>('/public/auth/verify_reset_code', data)
  }

  /**
   * 3.1.10 重置密码
   */
  async resetPassword(data: ResetPasswordRequest): Promise<void> {
    return this.publicPost<void>('/public/auth/reset_password', data)
  }

  /**
   * 3.1.11 获取活跃会话列表
   */
  async getSessions(params: ListSessionsRequest = {}): Promise<PaginatedData<SessionItem>> {
    return this.post<PaginatedData<SessionItem>>('/protected/auth/get_sessions', params)
  }

  /**
   * 3.1.12 撤销会话
   */
  async revokeSession(data: RevokeSessionRequest): Promise<void> {
    return this.post<void>('/protected/auth/revoke_session', data)
  }
}

export const authService = new AuthService()

export function useAuthService() {
  return authService
}


