use crate::utils::response::{PaginatedResponse, Pagination};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthUser {
    pub user_id: i64,
    pub nickname: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RegisterRequest {
    /// 登录ID（邮箱/手机号，必填，唯一，长度1-100）
    pub login_id: String,
    /// 用户昵称（可选，长度1-32，默认“用户”）
    pub nickname: Option<String>,
    /// 明文密码（必填，长度8-64，服务端自动加密存储）
    pub password: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LoginRequest {
    /// 登录ID（邮箱/手机号，必填）
    pub login_id: String,
    /// 明文密码（必填，长度8-64，服务端自动校验）
    pub password: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct RefreshTokenRequest {
    /// 刷新令牌（必填，登录/注册/刷新时获取）
    pub refresh_token: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateTempUserRequest {
    /// 临时用户昵称（可选，长度1-32，默认“临时用户”）
    pub nickname: Option<String>,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct AuthResponse {
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RefreshTokenResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserInfo {
    pub user_id: i64,
    pub nickname: String,
    pub last_active: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub distance: f64,
    pub location_name: String,
    pub last_active_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchUserByLocationRequest {
    /// 纬度（必填，-90~90）
    pub latitude: f64,
    /// 经度（必填，-180~180）
    pub longitude: f64,
    /// 搜索半径（必填，单位米，建议100~10000）
    pub radius: f64,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchUserByNameRequest {
    /// 搜索关键词（必填，长度1-50）
    pub keyword: String,
    #[serde(flatten)]
    pub pagination: Pagination,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SearchUserByIdRequest {
    /// 用户ID（必填，正整数）
    pub user_id: i64,
}

pub type SearchUserByLocationResponse = PaginatedResponse<UserInfo>;
pub type SearchUserByNameResponse = PaginatedResponse<UserInfo>;
pub type SearchUserByIdResponse = UserInfo;

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UpdateLocationRequest {
    /// 纬度（必填，-90~90）
    pub latitude: f64,
    /// 经度（必填，-180~180）
    pub longitude: f64,
    /// 位置名称（必填，长度1-100，客户端无法获取时用“南天门”）
    pub location_name: String,
}

#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SetPasswordRequest {
    /// 原密码（明文，首次设置可为空，修改时必填，长度8-64）
    pub old_password: Option<String>,
    /// 新密码（明文，必填，长度8-64，服务端自动加密存储）
    pub new_password: String,
}
