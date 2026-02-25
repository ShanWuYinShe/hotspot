//! 认证相关响应类型

use serde::Serialize;

use super::common::id_as_string;

/// 认证令牌数据 - API 文档: AuthTokenPairWeb/AuthTokenPairApp
#[derive(Debug, Serialize, Clone)]
pub struct AuthData {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub access_token: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    pub access_token_expires_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token_expires_at: Option<String>,
}

/// 认证状态检查数据 - API 文档: CheckStatusData
#[derive(Debug, Serialize)]
pub struct CheckStatusData {
    pub is_valid: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

/// 撤销所有会话返回数据 - API 文档: RevokeAllResult
#[derive(Debug, Serialize)]
pub struct RevokeAllResult {
    pub revoked_count: i64,
}

/// 会话列表返回数据 - API 文档: { items: SessionItem[] }
/// 直接返回全部活跃会话，无需分页
#[derive(Debug, Serialize)]
pub struct SessionsListData {
    pub items: Vec<SessionItem>,
}

/// 会话信息项 - API 文档: SessionItem
///
/// session_id 用于展示和撤销操作（调用 revoke 接口时传入）
#[derive(Debug, Serialize)]
pub struct SessionItem {
    /// 会话标识，用于展示和撤销操作
    pub session_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_name: Option<String>,
    /// IP 地址（已脱敏，如 "192.168.xxx.xxx"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_location: Option<String>,
    pub created_at: String,
    pub last_active_at: String,
    pub is_current: bool,
}
