import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { 
  AuthTokenPairWeb, 
  ClientType, 
  VerificationCodeType,
  SessionItem,
  PaginatedData,
} from '@/types/api';

const CLIENT_TYPE: ClientType = 'web';

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type: ClientType;
}

export interface RegisterRequest {
  login_id: string;
  nickname: string;
  password: string;
  client_type: ClientType;
  verify_token?: string;
}

export interface SendVerificationCodeRequest {
  phone: string;
  type: VerificationCodeType;
  captcha_token?: string;
}

export interface VerificationCodeSent {
  code_id: string;
  expires_at: string;
  retry_after?: number;
}

export interface VerifyCodeRequest {
  code_id: string;
  code: string;
}

export interface VerificationResult {
  verified: boolean;
  verify_token: string;
  expires_at: string;
}

export interface AuthStatus {
  user_id: number;
  nickname: string;
  is_valid: boolean;
  expires_at: string;
}

export interface RequestPasswordResetRequest {
  login_id: string;
}

export interface ResetRequested {
  code_id: string;
  expires_at: string;
}

export interface VerifyResetCodeRequest {
  code_id: string;
  code: string;
}

export interface ResetVerified {
  verify_token: string;
  expires_at: string;
}

export interface ResetPasswordRequest {
  verify_token: string;
  new_password: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface SessionsRequest {
  limit?: number;
  cursor?: string;
}

export interface RevokeSessionRequest {
  session_id: string;
}

export const AuthService = {
  login: (data: Omit<LoginRequest, 'client_type'>) => 
    apiClient.post<AuthTokenPairWeb>(ENDPOINTS.AUTH.LOGIN, { ...data, client_type: CLIENT_TYPE }),
    
  register: (data: Omit<RegisterRequest, 'client_type'>) => 
    apiClient.post<AuthTokenPairWeb>(ENDPOINTS.AUTH.REGISTER, { ...data, client_type: CLIENT_TYPE }),

  logout: () => apiClient.post(ENDPOINTS.AUTH.LOGOUT),

  guest: (data: { nickname: string }) =>
    apiClient.post<AuthTokenPairWeb>(ENDPOINTS.AUTH.GUEST, { ...data, client_type: CLIENT_TYPE }),

  refresh: () =>
    apiClient.post<AuthTokenPairWeb>(ENDPOINTS.AUTH.REFRESH_TOKEN, { client_type: CLIENT_TYPE }),

  checkStatus: () =>
    apiClient.post<AuthStatus>(ENDPOINTS.AUTH.CHECK_STATUS),

  sendVerificationCode: (data: SendVerificationCodeRequest) =>
    apiClient.post<VerificationCodeSent>(ENDPOINTS.AUTH.SEND_VERIFICATION_CODE, data),

  verifyCode: (data: VerifyCodeRequest) =>
    apiClient.post<VerificationResult>(ENDPOINTS.AUTH.VERIFY_CODE, data),

  requestPasswordReset: (data: RequestPasswordResetRequest) =>
    apiClient.post<ResetRequested>(ENDPOINTS.AUTH.REQUEST_PASSWORD_RESET, data),

  verifyResetCode: (data: VerifyResetCodeRequest) =>
    apiClient.post<ResetVerified>(ENDPOINTS.AUTH.VERIFY_RESET_CODE, data),

  resetPassword: (data: ResetPasswordRequest) =>
    apiClient.post(ENDPOINTS.AUTH.RESET_PASSWORD, data),

  changePassword: (data: ChangePasswordRequest) =>
    apiClient.post(ENDPOINTS.AUTH.CHANGE_PASSWORD, data),

  sessions: (params?: SessionsRequest) =>
    apiClient.post<PaginatedData<SessionItem>>(ENDPOINTS.AUTH.SESSIONS, params ?? {}),

  revokeSession: (data: RevokeSessionRequest) =>
    apiClient.post(ENDPOINTS.AUTH.REVOKE_SESSION, data),
};
