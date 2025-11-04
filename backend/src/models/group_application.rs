use chrono::{DateTime, Utc};
use serde::Serialize;
use sqlx::FromRow;
use uuid::Uuid;
use crate::types::enums::InviteStatus;

#[derive(Debug, Clone, FromRow, Serialize)]
pub struct GroupApplication {
    #[serde(skip_serializing)]
    pub id: i64,
    pub application_id: Uuid,
    pub group_id: Uuid,
    pub applicant_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus, // pending, approved, rejected
    pub created_at: DateTime<Utc>,
    pub reviewed_at: Option<DateTime<Utc>>,
    pub reviewer_id: Option<i64>,
    pub review_reason: Option<String>,
}

