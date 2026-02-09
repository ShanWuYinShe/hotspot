/**
 * 认证服务
 */

import { post } from '../client';
import { AUTH } from '../endpoints';
import type {
	AuthData,
	LoginRequest,
	RegisterRequest,
	GuestRequest,
	RefreshTokenRequest,
	SessionItem,
	AuthStatus,
	RevokeAllResult
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
		return post<AuthStatus>(AUTH.CHECK_STATUS);
	},

	/**
	 * 修改密码（已登录用户）
	 */
	changePassword(data: { old_password: string; new_password: string }) {
		return post<void>(AUTH.ME.CHANGE_PASSWORD, data);
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
	}
};
