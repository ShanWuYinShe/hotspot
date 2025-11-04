use chrono::{DateTime, Utc};
use sqlx::FromRow;
use uuid::Uuid;
use crate::types::enums::InviteStatus;

#[derive(Debug, Clone, FromRow)]
pub struct GroupInvite {
    pub invite_id: Uuid,
    pub group_id: Uuid,
    pub inviter_id: i64,
    pub invitee_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus,
    pub created_at: DateTime<Utc>,
    pub responded_at: Option<DateTime<Utc>>,
}

