use serde::{Deserialize, Serialize};
use sqlx::FromRow;
use uuid::Uuid;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Report {
    pub id: Uuid,
    pub reporter_id: i64,
    pub target_type: String, // "user", "moment", "comment", "message"
    pub target_id: String,   // 可以是user_id, moment_id, comment_id等
    pub reason: String,
    pub description: Option<String>,
    pub status: String, // "pending", "reviewing", "resolved", "rejected"
    pub reviewed_by: Option<i64>,
    pub review_note: Option<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
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
            id: Uuid::new_v4(),
            reporter_id,
            target_type,
            target_id,
            reason,
            description,
            status: "pending".to_string(),
            reviewed_by: None,
            review_note: None,
            created_at: now,
            updated_at: now,
        }
    }
}

