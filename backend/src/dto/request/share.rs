//! 分享模块

use serde::Deserialize;

use crate::{dto::request::common::PaginationParams, types::enums::ShareType};

#[derive(Debug, Deserialize)]
pub struct CreateShareRequest {
    pub share_type: ShareType,
    /// Payload shape depends on share_type; should validate against SharePayload schema.
    /// Kept as Value as pragmatic tradeoff; full discriminated union typing is complex.
    #[serde(flatten)]
    pub payload: serde_json::Value,
    /// 可选，默认由系统配置；若提供须为正整数且不超过系统允许上限
    pub expires_in_seconds: Option<u32>,
}

#[derive(Debug, Deserialize)]
pub struct GetShareRequest {
    pub share_code: String,
}

/// 分享撤销请求 - POST /api/v1/share/revoke
#[derive(Debug, Deserialize)]
pub struct RevokeShareRequest {
    pub share_id: String,
}

#[derive(Debug, Deserialize)]
pub struct ShareListRequest {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_type: Option<ShareType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pagination: Option<PaginationParams>,
}
