// ==================== 认证API类型 ====================
import { ClientType, DeviceId } from './common';


export interface AuthTokenPairApp {
  client_type: 'app'; // 判别字段（spec auth.md → AuthTokenPair）
  user_id: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

// HarmonyOS 使用 App 类型
export interface AuthData extends AuthTokenPairApp {}

export interface RegisterRequest {
  login_id: string;
  // 手机号/邮箱型 login_id 必填（所有权验证，经 auth/send_register_code 发送）；用户名型不传
  verification_code?: string | null;
  nickname: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId;
}

export interface SendRegisterCodeRequest {
  login_id: string;
  client_type: ClientType;
}

export interface LoginRequest {
  login_id: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId;
}

export interface GuestRequest {
  nickname: string | null; // 可选；null 时服务端自动生成默认昵称（spec auth.md → guest）
  client_type: ClientType;
  device_id: DeviceId;
}

export interface RefreshTokenRequest {
  client_type: ClientType;
  device_id: DeviceId;
  refresh_token?: string;
}

export interface SessionStatus {
  user_id: string;
  expires_at: string;
}

export interface SessionItem {
  session_id: string;
  device_id: DeviceId;
  device_type: string | null;
  device_name: string | null;
  ip_address: string | null;
  ip_location: string | null;
  created_at: string;
  last_active_at: string;
  is_current: boolean;
}

export interface SessionsRequest {}

export interface RevokeSessionRequest {
  session_id: string;
}

export interface RevokeAllResult {
  revoked_count: number;
}

export interface RequestPasswordResetRequest {
  login_id: string;
  client_type: ClientType;
}

export interface ResetPasswordRequest {
  login_id: string;
  verification_code: string;
  new_password: string;
  client_type: ClientType;
  device_id: DeviceId; // App 必传（spec auth.md → reset_password）
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}
