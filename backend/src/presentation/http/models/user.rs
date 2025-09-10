/// 用户相关HTTP模型
/// 
/// 迁移自models/api/user.rs，适配新架构
use super::{PaginatedResponse, LocationInfo, TimestampInfo};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use utoipa::ToSchema;
use crate::utils::JsSafeI64;

/// 认证用户信息
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct AuthUser {
    pub user_id: JsSafeI64,
    pub nickname: String,
}

/// 用户注册请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct RegisterRequest {
    /// 登录ID（邮箱/手机号，必填，唯一，长度1-100）
    pub login_id: String,
    /// 用户昵称（可选，长度1-32，默认"用户"）
    pub nickname: Option<String>,
    /// 明文密码（必填，长度8-64，服务端自动加密存储）
    pub password: String,
}

/// 用户登录请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct LoginRequest {
    /// 登录ID（邮箱/手机号，必填）
    pub login_id: String,
    /// 明文密码（必填，长度8-64，服务端自动校验）
    pub password: String,
}

/// 刷新令牌请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct RefreshTokenRequest {
    /// 刷新令牌（必填，登录/注册/刷新时获取）
    pub refresh_token: String,
}

/// 创建临时用户请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct CreateTempUserRequest {
    /// 临时用户昵称（可选，长度1-32，默认"临时用户"）
    pub nickname: Option<String>,
}

/// 认证响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct AuthResponse {
    pub user_id: JsSafeI64,
    pub nickname: String,
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
    pub is_temp: Option<bool>,
}

/// 刷新令牌响应
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct RefreshTokenResponse {
    pub access_token: String,
    pub refresh_token: String,
    pub expires_at: DateTime<Utc>,
}

/// 用户信息
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserInfo {
    pub user_id: JsSafeI64,
    pub nickname: String,
    pub last_active: DateTime<Utc>,
    pub latitude: f64,
    pub longitude: f64,
    pub distance: f64,
    pub location_name: String,
    pub last_active_at: DateTime<Utc>,
}

/// 用户资料
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserProfile {
    pub user_id: JsSafeI64,
    pub nickname: String,
    pub login_id: String,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    #[serde(flatten)]
    pub location: LocationInfo,
}

/// 更新用户资料请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UpdateUserProfileRequest {
    /// 用户昵称
    pub nickname: Option<String>,
    /// 位置信息
    #[serde(flatten)]
    pub location: Option<LocationInfo>,
}

/// 附近用户查询请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct NearbyUsersRequest {
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 搜索半径（米）
    pub radius: f64,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: super::Pagination,
}

/// 附近用户响应
pub type NearbyUsersResponse = PaginatedResponse<UserInfo>;

/// 用户搜索请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct SearchUsersRequest {
    /// 搜索关键词
    pub keyword: String,
    /// 分页参数
    #[serde(flatten)]
    pub pagination: super::Pagination,
}

/// 用户搜索结果（轻量级，仅包含必要字段）
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserSearchResult {
    pub user_id: JsSafeI64,
    pub nickname: String,
    pub avatar_url: Option<String>,
    pub last_active_at: DateTime<Utc>,
    /// 是否为临时用户
    pub is_temp: bool,
}

/// 用户搜索响应
pub type SearchUsersResponse = PaginatedResponse<UserSearchResult>;

/// 用户设置
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UserSettings {
    pub user_id: JsSafeI64,
    /// 通知设置
    pub notifications_enabled: bool,
    /// 位置共享设置
    pub location_sharing_enabled: bool,
    /// 隐私设置
    pub privacy_level: String,
    #[serde(flatten)]
    pub timestamps: TimestampInfo,
}

/// 更新用户设置请求
#[derive(Debug, Serialize, Deserialize, ToSchema)]
pub struct UpdateUserSettingsRequest {
    /// 通知设置
    pub notifications_enabled: Option<bool>,
    /// 位置共享设置
    pub location_sharing_enabled: Option<bool>,
    /// 隐私设置
    pub privacy_level: Option<String>,
}
