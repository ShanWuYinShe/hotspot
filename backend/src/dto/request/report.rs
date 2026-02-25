//! 举报审核 API

use serde::Deserialize;

use super::common::PaginationParams;
use crate::types::enums::{ReportHandleAction, ReportReason, ReportStatus, ReportTargetType};

#[derive(Debug, Deserialize)]
pub struct CreateReportRequest {
    pub target_type: ReportTargetType,
    pub target_id: String, // 统一为 string（user_id 也转为字符串格式）
    pub reason: ReportReason,
    pub description: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyReportsRequest {
    pub status: Option<ReportStatus>,
    pub target_type: Option<ReportTargetType>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct AdminReportSearchRequest {
    pub status: Option<ReportStatus>,
    pub target_type: Option<ReportTargetType>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct AdminReportHandleRequest {
    pub report_id: String,
    pub action: ReportHandleAction,
    pub note: Option<String>,
    pub ban_duration_days: Option<u32>,
}
