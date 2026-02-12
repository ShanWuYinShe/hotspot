//! 通知仓库层实现
//!
//! 适配新 Schema: notification_id (UUID), user_id (BIGINT), 分区表 notifications_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
pub use crate::types::enums::NotificationType;

/// 通知行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct NotificationRow {
    pub notification_id: Uuid,
    pub user_id: i64,
    pub r#type: NotificationType,
    pub title: Option<String>,
    pub content: Option<String>,
    pub sender_id: Option<i64>,
    pub sender_nickname: Option<String>,
    pub sender_avatar_url: Option<String>,
    pub target_type: Option<String>,
    pub target_id: Option<String>,
    pub reference_id: Option<String>,
    pub target_preview: Option<String>,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct NotificationRepository {
    pool: PgPool,
}

impl NotificationRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建通知
    pub async fn create(
        &self,
        user_id: i64,
        r#type: NotificationType,
        title: String,
        content: String,
        sender_id: Option<i64>,
        sender_nickname: Option<String>,
        sender_avatar_url: Option<String>,
        target_type: Option<String>,
        target_id: Option<String>,
        reference_id: Option<String>,
        target_preview: Option<String>,
    ) -> Result<NotificationRow> {
        let payload = serde_json::json!({
            "title": title,
            "content": content,
            "sender_id": sender_id,
            "sender_nickname": sender_nickname,
            "sender_avatar_url": sender_avatar_url,
            "target_type": target_type,
            "target_id": target_id,
            "reference_id": reference_id,
            "target_preview": target_preview
        });

        let row = sqlx::query_as!(
            NotificationRow,
            r#"
            WITH inserted AS (
                INSERT INTO notifications_partitioned (user_id, notification_type, payload)
                VALUES ($1, $2, $3)
                RETURNING notification_id, user_id, notification_type, payload, created_at
            )
            SELECT
                notification_id as "notification_id!",
                user_id as "user_id!",
                notification_type as "type!: NotificationType",
                payload->>'title' as "title?",
                payload->>'content' as "content?",
                (payload->>'sender_id')::bigint as "sender_id?",
                payload->>'sender_nickname' as "sender_nickname?",
                payload->>'sender_avatar_url' as "sender_avatar_url?",
                payload->>'target_type' as "target_type?",
                payload->>'target_id' as "target_id?",
                payload->>'reference_id' as "reference_id?",
                payload->>'target_preview' as "target_preview?",
                notification_is_read($1, notification_id, notification_type, created_at) as "is_read!",
                created_at as "created_at!"
            FROM inserted
            "#,
            user_id,
            r#type.to_string(),
            payload
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }
}
