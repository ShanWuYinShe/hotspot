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
} from '@/lib/types/api';

export const authService = {
  login(data: LoginRequest) {
    return post<Record<string, never>>(AUTH.LOGIN, data);
  },

  register(data: RegisterRequest) {
    return post<Record<string, never>>(AUTH.REGISTER, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  guest(data: GuestRequest) {
    return post<Record<string, never>>(AUTH.GUEST, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

  logout() {
    return post<void>(AUTH.LOGOUT);
  },

  checkStatus() {
    return post<AuthStatus>(AUTH.CHECK_STATUS);
  },

  changePassword(data: { old_password: string; new_password: string }) {
    return post<void>(AUTH.ME.CHANGE_PASSWORD, data, { idempotencyKey: generateIdempotencyKey() });
  },

  getSessions() {
    return post<{ items: SessionItem[] }>(AUTH.ME.SESSIONS_LIST);
  },

  revokeSession(data: { session_id: string }) {
    return post<void>(AUTH.ME.REVOKE_SESSION, data);
  },

  revokeAllSessions() {
    return post<RevokeAllResult>(AUTH.ME.REVOKE_ALL_SESSIONS);
  },

  requestPasswordReset(data: { login_id: string; client_type: ClientType }) {
    return post<void>(AUTH.REQUEST_PASSWORD_RESET, data, {
      idempotencyKey: generateIdempotencyKey()
    });
  },

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
