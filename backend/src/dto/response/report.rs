//! 举报相关响应类型

use crate::types::enums::{ReportReason, ReportStatus, ReportTargetType};
use serde::{Deserialize, Serialize};

use super::common::id_as_string;
use super::user::UserInfo;

/// 创建举报返回数据
#[derive(Debug, Serialize)]
pub struct CreateReportData {
    pub report_id: String,
}

/// 举报内容快照 - API 文档: ReportTargetSnapshot
///
/// 结构说明：
/// - moment: 瞬间快照（描述、图片链接、作者ID）
/// - comment: 评论快照（内容、作者ID、所属瞬间ID）
/// - message: 消息快照（内容、媒体链接、发送者ID）
/// - 注意：根据举报目标类型，只有对应类型的快照字段会填充
#[derive(Debug, Serialize, Deserialize)]
pub struct ReportTargetSnapshot {
    // 瞬间快照
    #[serde(skip_serializing_if = "Option::is_none")]
    pub moment: Option<MomentSnapshot>,
    // 评论快照
    #[serde(skip_serializing_if = "Option::is_none")]
    pub comment: Option<CommentSnapshot>,
    // 消息快照
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<MessageSnapshot>,
}

/// 瞬间快照
#[derive(Debug, Serialize, Deserialize)]
pub struct MomentSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub image_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub author_id: Option<String>,
}

/// 评论快照
#[derive(Debug, Serialize, Deserialize)]
pub struct CommentSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub author_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub moment_id: Option<String>,
}

/// 消息快照
#[derive(Debug, Serialize, Deserialize)]
pub struct MessageSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sender_id: Option<String>,
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
    #[serde(rename = "handle_note", skip_serializing_if = "Option::is_none")]
    pub handled_note: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_snapshot: Option<ReportTargetSnapshot>,
}
