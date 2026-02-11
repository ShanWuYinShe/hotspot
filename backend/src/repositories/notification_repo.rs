//! 通知仓库层实现
//!
//! 适配新 Schema: notification_id (UUID), user_id (BIGINT), 分区表 notifications_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::str::FromStr;
use uuid::Uuid;

use crate::error::Result;
pub use crate::types::enums::NotificationType;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeIdCursor};

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

#[derive(Debug, Clone, FromRow)]
struct UnreadBreakdownRow {
    pub total: i64,
    pub breakdown: serde_json::Value,
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

    /// 列出用户的通知
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
        type_filters: Option<Vec<String>>,
    ) -> Result<(Vec<NotificationRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeIdCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, Uuid::max())
        };

        let rows = sqlx::query_as!(
            NotificationRow,
            r#"
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
            FROM notifications_partitioned
            WHERE user_id = $1
              AND (created_at, notification_id) < ($2, $3)
              AND ($4::text[] IS NULL OR notification_type = ANY($4))
            ORDER BY created_at DESC, notification_id DESC
            LIMIT $5
            "#,
            user_id,
            last_ts,
            last_id,
            type_filters.as_deref(),
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.notification_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn count_unread(&self, user_id: i64) -> Result<i64> {
        let row = sqlx::query_as!(
            UnreadBreakdownRow,
            "SELECT total::bigint as \"total!\", breakdown as \"breakdown!\" FROM get_unread_notifications_breakdown($1)",
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.total)
    }

    pub async fn count_unread_breakdown(
        &self,
        user_id: i64,
    ) -> Result<Vec<(NotificationType, i64)>> {
        let row = sqlx::query_as!(
            UnreadBreakdownRow,
            "SELECT total::bigint as \"total!\", breakdown as \"breakdown!\" FROM get_unread_notifications_breakdown($1)",
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        let mut result = Vec::new();
        if let Some(obj) = row.breakdown.as_object() {
            for (key, value) in obj {
                if let Ok(r#type) = NotificationType::from_str(key) {
                    let count = value.as_i64().unwrap_or(0);
                    result.push((r#type, count));
                }
            }
        }

        Ok(result)
    }

    pub async fn mark_read(&self, user_id: i64, notification_ids: &[Uuid]) -> Result<()> {
        sqlx::query!(
            "SELECT mark_notifications_read($1, NULL, $2, NOW())",
            user_id,
            notification_ids
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn mark_type_read(&self, user_id: i64, r#type: NotificationType) -> Result<()> {
        sqlx::query!(
            "SELECT mark_notifications_read($1, $2, NULL, NOW())",
            user_id,
            r#type.to_string()
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            "SELECT mark_notifications_read($1, NULL, NULL, NOW())",
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
