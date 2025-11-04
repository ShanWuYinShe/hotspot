use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;
use crate::types::enums::ReportStatus;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Report {
    pub id: i64,
    pub report_id: Uuid,
    pub reporter_id: i64,
    pub target_type: String, // "user", "moment", "comment", "message", "group"
    pub target_id: String,   // 可以是user_id, moment_id, comment_id等
    pub reason: String,
    pub description: Option<String>,
    pub status: ReportStatus,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub handled_at: Option<chrono::DateTime<chrono::Utc>>,
    pub handler_id: Option<i64>,
    pub handler_note: Option<String>,
}

impl Report {
    pub fn new(
        reporter_id: i64,
        target_type: String,
        target_id: String,
        reason: String,
        description: Option<String>,
    ) -> Self {
        let now = chrono::Utc::now();
        Self {
            id: 0, // 由数据库自动生成
            report_id: Uuid::new_v4(),
            reporter_id,
            target_type,
            target_id,
            reason,
            description,
            status: ReportStatus::Pending,
            created_at: now,
            handled_at: None,
            handler_id: None,
            handler_note: None,
        }
    }
}

