//! 通知模块

use serde::Deserialize;

use crate::types::enums::NotificationType;

/// 通知列表请求
#[derive(Debug, Deserialize)]
pub struct NotificationListRequest {
    #[serde(rename = "type")]
    pub notification_type: Option<NotificationType>,
    pub cursor: Option<String>,
    pub limit: Option<i64>,
}

/// 标记通知已读请求
#[derive(Debug, Deserialize)]
pub struct MarkNotificationsReadRequest {
    pub notification_ids: Vec<String>,
}
