// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	SessionStatus,
	AuthTokenPair,
	RegisterRequest,
	LoginRequest,
	GuestRequest,
	SendRegisterCodeRequest,
	RequestPasswordResetRequest,
	ResetPasswordRequest,
	ChangePasswordRequest,
	RevokeSessionRequest,
	SessionList,
	RevokeOtherSessionsResult,
} from '@/api/types';

export async function register(params: RegisterRequest)           { return request<AuthTokenPair>(Endpoints.Auth.Register, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function login(params: LoginRequest)              { return request<AuthTokenPair>(Endpoints.Auth.Login, params); }
export async function guest(params?: GuestRequest)             { return request<AuthTokenPair>(Endpoints.Auth.Guest, params ?? {}, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function checkAuthStatus()                   { return request<SessionStatus>(Endpoints.Auth.Session); }
export async function logout()                            { return request<void>(Endpoints.Auth.Signout); }
export async function requestPasswordReset(params: RequestPasswordResetRequest) { return request<void>(Endpoints.Auth.RequestPasswordReset, params); }
export async function resetPassword(params: ResetPasswordRequest)      { return request<AuthTokenPair>(Endpoints.Auth.ResetPassword, params); }
// spec: api_spec_public.md → auth/send_register_code（手机号/邮箱型 login_id 的所有权验证，防枚举恒成功）
export async function sendRegisterCode(params: SendRegisterCodeRequest) { return request<void>(Endpoints.Auth.SendRegisterCode, params); }
// spec: api_spec_auth.md → auth/change_password（成功后服务端自动吊销其余会话；10401=旧密码错误）
export async function changePassword(params: ChangePasswordRequest) { return request<void>(Endpoints.Auth.ChangePassword, params); }
// spec: api_spec_auth.md → auth/fetch_sessions（全量活跃会话，当前会话置顶，无分页）
export async function fetchSessions() { return request<SessionList>(Endpoints.Auth.FetchSessions); }
// spec: api_spec_auth.md → auth/revoke_session（不允许撤销当前会话，10400；当前会话走 signout）
export async function revokeSession(params: RevokeSessionRequest) { return request<void>(Endpoints.Auth.RevokeSession, params); }
// spec: api_spec_auth.md → auth/revoke_other_sessions（撤销当前会话之外的所有其他活跃会话）
export async function revokeOtherSessions() { return request<RevokeOtherSessionsResult>(Endpoints.Auth.RevokeOtherSessions); }
