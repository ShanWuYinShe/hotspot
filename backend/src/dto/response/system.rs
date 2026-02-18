//! 系统健康检查与版本元信息

use crate::types::enums::SystemHealthStatus;
use serde::Serialize;

/// API 版本状态 - API 文档: 'active' | 'deprecated' | 'sunset'
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum ApiVersionStatus {
    Active,
    Deprecated,
    Sunset,
}

impl ApiVersionStatus {
    pub const fn variants() -> [Self; 3] {
        [Self::Active, Self::Deprecated, Self::Sunset]
    }
}

/// API 版本元信息 - API 文档: ApiVersionMeta
#[derive(Debug, Serialize)]
pub struct ApiVersionMeta {
    pub version: String,
    pub status: ApiVersionStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub deprecated_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sunset_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recommended_version: Option<String>,
}

/// 健康检查数据 - API 文档: HealthStatus
#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: SystemHealthStatus,
    pub timestamp: String,
}
