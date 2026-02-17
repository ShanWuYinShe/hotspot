//! 分享模块响应类型

use serde::Serialize;
use serde_json::Value;

use super::user::UserInfo;
use crate::types::enums::ShareType;

/// 创建分享响应 - API 文档: ShareCreated
#[derive(Debug, Serialize, Clone)]
pub struct ShareCreated {
    pub share_id: String,
    pub share_code: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

/// 分享基础信息 - API 文档: ShareBase
#[derive(Debug, Serialize, Clone)]
pub struct ShareBase {
    pub share_id: String,
    pub share_code: String,
    pub share_type: crate::types::enums::ShareType,
    pub creator: UserInfo,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

/// 分享预览 - API 文档: SharePreview
#[derive(Debug, Serialize, Clone)]
pub struct SharePreview {
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subtitle: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
}

/// 分享详情 - API 文档: ShareDetail
#[derive(Debug, Serialize, Clone)]
pub struct ShareDetail {
    pub share: ShareBase,
    pub payload: Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview: Option<SharePreview>,
}

/// 分享列表项 - API 文档: ShareListItem
#[derive(Debug, Serialize, Clone)]
pub struct ShareListItem {
    pub share_id: String,
    pub share_code: String,
    pub share_type: ShareType,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
    pub is_revoked: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview: Option<SharePreview>,
}
