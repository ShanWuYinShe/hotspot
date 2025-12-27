import { BaseService } from './base.service'
import { AUTH } from '~/config/api'
import type {
  AuthData,
  RegisterRequest,
  LoginRequest,
  GuestRequest,
  RequestPasswordResetRequest,
  RequestPasswordResetData,
  VerifyResetCodeRequest,
  VerifyResetCodeData,
  ResetPasswordRequest,
  ChangePasswordRequest,
  CheckStatusData,
  SessionsRequest,
  RevokeSessionRequest,
  SessionItem,
  PaginatedData
} from '~/types/api'

class AuthService extends BaseService {
  async register(data: RegisterRequest) {
    return this.publicPost<AuthData>(AUTH.REGISTER, data)
  }

  async login(data: LoginRequest) {
    return this.publicPost<AuthData>(AUTH.LOGIN, data)
  }

  async logout() {
    return this.post<void>(AUTH.LOGOUT, {})
  }

  async guest(data: GuestRequest) {
    return this.publicPost<AuthData>(AUTH.GUEST, data)
  }

  async checkStatus() {
    return this.post<CheckStatusData>(AUTH.CHECK_STATUS, {})
  }

  async requestPasswordReset(data: RequestPasswordResetRequest) {
    return this.publicPost<RequestPasswordResetData>(AUTH.REQUEST_PASSWORD_RESET, data)
  }

  async verifyResetCode(data: VerifyResetCodeRequest) {
    return this.publicPost<VerifyResetCodeData>(AUTH.VERIFY_RESET_CODE, data)
  }

  async resetPassword(data: ResetPasswordRequest) {
    return this.publicPost<void>(AUTH.RESET_PASSWORD, data)
  }

  // ============ 当前用户认证相关 API（/me/ 路径）============

  async changePassword(data: ChangePasswordRequest) {
    return this.post<void>(AUTH.ME.CHANGE_PASSWORD, data)
  }

  async getSessions(params: SessionsRequest = {}) {
    return this.post<PaginatedData<SessionItem>>(AUTH.ME.SESSIONS, params)
  }

  async revokeSession(data: RevokeSessionRequest) {
    return this.post<void>(AUTH.ME.REVOKE_SESSION, data)
  }
}

export const authService = new AuthService()
export function useAuthService() { return authService }
