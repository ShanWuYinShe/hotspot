/**
 * 认证服务
 */

import { post } from '../client';
import { AUTH } from '../endpoints';
import type {
	ApiResponse,
	AuthData,
	LoginRequest,
	RegisterRequest,
	GuestRequest,
	RefreshTokenRequest,
	SessionItem,
	PaginatedData
} from '$lib/types/api';

export const authService = {
	/**
	 * 用户登录
	 */
	login(data: LoginRequest) {
		return post<AuthData>(AUTH.LOGIN, data);
	},

	/**
	 * 用户注册
	 */
	register(data: RegisterRequest) {
		return post<AuthData>(AUTH.REGISTER, data);
	},

	/**
	 * 创建访客用户
	 */
	guest(data: GuestRequest) {
		return post<AuthData>(AUTH.GUEST, data);
	},

	/**
	 * 用户登出
	 */
	logout() {
		return post<void>(AUTH.LOGOUT);
	},

	/**
	 * 刷新 Token
	 */
	refreshToken(data: RefreshTokenRequest) {
		return post<{
			access_token: string;
			access_token_expires_at: string;
			refresh_token?: string;
			refresh_token_expires_at?: string;
		}>(AUTH.REFRESH_TOKEN, data, { skipAuth: true });
	},

	/**
	 * 检查认证状态
	 */
	checkStatus() {
		return post<{
			is_authenticated: boolean;
			user_id?: number;
			nickname?: string;
		}>(AUTH.CHECK_STATUS);
	},

	/**
	 * 发送验证码
	 */
	sendVerificationCode(data: { login_id: string; purpose: 'register' | 'reset_password' }) {
		return post<{ expires_at: string }>(AUTH.SEND_VERIFICATION_CODE, data, { skipAuth: true });
	},

	/**
	 * 校验验证码
	 */
	verifyCode(data: { login_id: string; code: string; purpose: 'register' | 'reset_password' }) {
		return post<{ verify_token: string }>(AUTH.VERIFY_CODE, data, { skipAuth: true });
	},

	/**
	 * 请求重置密码
	 */
	requestPasswordReset(data: { login_id: string }) {
		return post<{ expires_at: string }>(AUTH.REQUEST_PASSWORD_RESET, data, { skipAuth: true });
	},

	/**
	 * 校验重置码
	 */
	verifyResetCode(data: { login_id: string; code: string }) {
		return post<{ reset_token: string }>(AUTH.VERIFY_RESET_CODE, data, { skipAuth: true });
	},

	/**
	 * 重置密码
	 */
	resetPassword(data: { reset_token: string; new_password: string }) {
		return post<void>(AUTH.RESET_PASSWORD, data, { skipAuth: true });
	},

	/**
	 * 修改密码（已登录用户）
	 */
	changePassword(data: { old_password: string; new_password: string }) {
		return post<void>(AUTH.ME.CHANGE_PASSWORD, data);
	},

	/**
	 * 获取会话列表
	 */
	getSessions(params?: { limit?: number; cursor?: string }) {
		return post<PaginatedData<SessionItem>>(AUTH.ME.SESSIONS, params);
	},

	/**
	 * 撤销会话
	 */
	revokeSession(data: { session_id: string }) {
		return post<void>(AUTH.ME.REVOKE_SESSION, data);
	}
};
