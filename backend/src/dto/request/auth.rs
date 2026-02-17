//! 认证模块请求

use serde::Deserialize;

use crate::types::enums::ClientType;

#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub login_id: String,
    pub nickname: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct LogoutRequest {}

#[derive(Debug, Deserialize)]
pub struct GuestRequest {
    pub nickname: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    /// app：在 body 传递；web：必须不传（从 HttpOnly cookie 读取）
    pub refresh_token: Option<String>,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct CheckStatusRequest {}

#[derive(Debug, Deserialize)]
pub struct ChangePasswordRequest {
    pub old_password: String,
    pub new_password: String,
}

#[derive(Debug, Deserialize)]
pub struct RequestPasswordResetRequest {
    /// 注册时使用的邮箱或手机号
    pub login_id: String,
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct ResetPasswordRequest {
    pub login_id: String,
    pub verification_code: String,
    pub new_password: String,
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct SessionsRequest {}

#[derive(Debug, Deserialize)]
pub struct RevokeSessionRequest {
    /// 会话标识（来自 SessionItem.session_id）
    pub session_id: String,
}

/// 撤销所有会话请求 - POST /api/v1/auth/me/sessions/revoke_all
#[derive(Debug, Deserialize)]
pub struct RevokeAllSessionsRequest {}
