/**
 * 认证服务
 */

import { post, get, del, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  AuthData,
  CheckStatusData,
  SessionItem,
  PaginatedData,
} from '@/types/api';

export interface LoginParams {
  login_id: string;
  password: string;
  client_type: 'web' | 'app';
}

export interface RegisterParams {
  login_id: string;
  nickname: string;
  password: string;
  client_type: 'web' | 'app';
  verify_token?: string;
}

export interface GuestParams {
  nickname: string;
  client_type: 'web' | 'app';
}

export interface RefreshParams {
  client_type: 'web' | 'app';
  refresh_token?: string;
}

export interface ChangePasswordParams {
  old_password: string;
  new_password: string;
}

export const authService = {
  /**
   * 用户登录
   */
  async login(params: LoginParams): Promise<ApiResponse<AuthData>> {
    return post<AuthData>(ENDPOINTS.AUTH.LOGIN, params, {
      requireAuth: false,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 用户注册
   */
  async register(params: RegisterParams): Promise<ApiResponse<AuthData>> {
    return post<AuthData>(ENDPOINTS.AUTH.REGISTER, params, {
      requireAuth: false,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 访客登录
   */
  async guest(params: GuestParams): Promise<ApiResponse<AuthData>> {
    return post<AuthData>(ENDPOINTS.AUTH.GUEST, params, {
      requireAuth: false,
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 退出登录
   */
  async logout(): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.AUTH.LOGOUT);
  },

  /**
   * 刷新 Token
   */
  async refresh(params: RefreshParams): Promise<ApiResponse<AuthData>> {
    return post<AuthData>(ENDPOINTS.AUTH.REFRESH, params, {
      requireAuth: false,
    });
  },

  /**
   * 检查登录状态
   */
  async checkStatus(): Promise<ApiResponse<CheckStatusData>> {
    return get<CheckStatusData>(ENDPOINTS.AUTH.CHECK_STATUS);
  },

  /**
   * 修改密码
   */
  async changePassword(params: ChangePasswordParams): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.AUTH.CHANGE_PASSWORD, params, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取会话列表
   */
  async getSessions(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<SessionItem>>> {
    return get<PaginatedData<SessionItem>>(ENDPOINTS.AUTH.SESSIONS, params);
  },

  /**
   * 撤销会话
   */
  async revokeSession(sessionId: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.AUTH.REVOKE_SESSION(sessionId));
  },
};
