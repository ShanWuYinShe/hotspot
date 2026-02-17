//! 举报相关响应类型

use crate::types::enums::{ReportReason, ReportStatus, ReportTargetType};
use serde::Serialize;

use super::common::id_as_string;
use super::user::UserInfo;

/// 创建举报返回数据
#[derive(Debug, Serialize)]
pub struct CreateReportData {
    pub report_id: String,
}

/// 举报内容快照 - API 文档: ReportTargetSnapshot
#[derive(Debug, Serialize)]
pub struct ReportTargetSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<String>,
}

/// 举报项 - API 文档: ReportItem (用户可见，不含 target_snapshot)
#[derive(Debug, Serialize)]
pub struct ReportItem {
    pub report_id: String,
    pub target_type: ReportTargetType,
    pub target_id: String, // 统一为 string（user_id 也转为字符串格式）
    pub reason: ReportReason,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub status: ReportStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
}

/// 管理员举报项 - API 文档: AdminReportItem (含 target_snapshot 等管理员可见字段)
#[derive(Debug, Serialize)]
pub struct AdminReportItem {
    #[serde(flatten)]
    pub report: ReportItem,
    #[serde(serialize_with = "id_as_string::serialize")]
    pub reporter_user_id: i64,
    pub reporter: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_by: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_action: Option<crate::types::enums::ReportHandleAction>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_note: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_snapshot: Option<ReportTargetSnapshot>,
}
