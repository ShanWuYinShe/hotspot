/**
 * 认证服务
 *
 * spec: Web 端 auth 响应体为 AuthTokenPairWeb = Record<string, never>（空对象）
 * Token 通过 Set-Cookie 传递，前端不可见
 */

import { post, generateIdempotencyKey } from '../client';
import { AUTH } from '../endpoints';
import type {
  ClientType,
  LoginRequest,
  RegisterRequest,
  GuestRequest,
  SessionItem,
  AuthStatus,
  RevokeAllResult
} from '$lib/types/api';

export const authService = {
  /**
   * 用户登录
   * spec: Web 端响应体为空对象，token 通过 Set-Cookie 设置
   */
  login(data: LoginRequest) {
    return post<Record<string, never>>(AUTH.LOGIN, data);
  },

  /**
   * 用户注册
   * spec: Web 端响应体为空对象，token 通过 Set-Cookie 设置
   */
  register(data: RegisterRequest) {
    return post<Record<string, never>>(AUTH.REGISTER, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 创建访客用户
   * spec: Web 端响应体为空对象，token 通过 Set-Cookie 设置
   */
  guest(data: GuestRequest) {
    return post<Record<string, never>>(AUTH.GUEST, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 用户登出
   * spec: Web 端通过 Set-Cookie Max-Age=0 清除 token Cookie
   */
  logout() {
    return post<void>(AUTH.LOGOUT);
  },

  /**
   * 检查认证状态
   * spec: Token 可选，未登录时返回 is_valid: false
   */
  checkStatus() {
    return post<AuthStatus>(AUTH.CHECK_STATUS);
  },

  /**
   * 修改密码（已登录用户）
   */
  changePassword(data: { old_password: string; new_password: string }) {
    return post<void>(AUTH.ME.CHANGE_PASSWORD, data, { idempotencyKey: generateIdempotencyKey() });
  },

  /**
   * 获取会话列表（非分页，返回 { items }）
   */
  getSessions() {
    return post<{ items: SessionItem[] }>(AUTH.ME.SESSIONS_LIST);
  },

  /**
   * 撤销会话
   */
  revokeSession(data: { session_id: string }) {
    return post<void>(AUTH.ME.REVOKE_SESSION, data);
  },

  /**
   * 撤销所有其他会话
   */
  revokeAllSessions() {
    return post<RevokeAllResult>(AUTH.ME.REVOKE_ALL_SESSIONS);
  },

  /**
   * 请求密码重置（发送验证码）
   * spec: client_type 必填，用于确定验证码投递方式
   */
  requestPasswordReset(data: { login_id: string; client_type: ClientType }) {
    return post<void>(AUTH.REQUEST_PASSWORD_RESET, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  /**
   * 重置密码（使用验证码）
   * spec: 重置成功后自动登录，Web 端 token 通过 Set-Cookie 设置
   */
  resetPassword(data: {
    login_id: string;
    verification_code: string;
    new_password: string;
    client_type: ClientType;
  }) {
    return post<Record<string, never>>(AUTH.RESET_PASSWORD, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  }
};
