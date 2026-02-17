//! 通知仓库层实现
//!
//! 适配新 Schema: notification_id (UUID), user_id (BIGINT), 分区表 notifications_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
pub use crate::types::enums::NotificationType;

/// 通知行数据（列表查询用）
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct NotificationListRow {
    pub notification_id: Uuid,
    pub notification_type: String,
    pub payload: serde_json::Value,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

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
        extra_payload: Option<serde_json::Value>,
    ) -> Result<NotificationRow> {
        let mut payload = serde_json::json!({
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
        if let Some(extra) = extra_payload {
            if let Some(obj) = payload.as_object_mut() {
                if let Some(extra_obj) = extra.as_object() {
                    for (k, v) in extra_obj {
                        obj.insert(k.clone(), v.clone());
                    }
                }
            }
        }

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

    /// 分页列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        notification_type: Option<NotificationType>,
        cursor: Option<&str>,
        limit: i64,
    ) -> Result<Vec<NotificationListRow>> {
        let limit = limit.clamp(1, crate::constants::MAX_PAGE_LIMIT as i64);
        let type_filter = notification_type.map(|t| t.to_string());

        #[derive(serde::Deserialize)]
        struct CursorPayload {
            created_at: chrono::DateTime<chrono::Utc>,
            notification_id: Uuid,
        }

        let rows = if let Some(c) = cursor {
            let decoded: CursorPayload = crate::utils::pagination::decode_cursor(c)
                .map_err(|e| crate::error::AppError::CursorFormatError(e.to_string()))?;

            if let Some(ref t) = type_filter {
                sqlx::query_as!(
                    NotificationListRow,
                    r#"
                    SELECT
                        notification_id as "notification_id!",
                        notification_type as "notification_type!",
                        payload as "payload!",
                        notification_is_read($1, notification_id, notification_type, created_at) as "is_read!",
                        created_at as "created_at!"
                    FROM notifications_partitioned
                    WHERE user_id = $1
                      AND notification_type = $2
                      AND (created_at, notification_id) < ($3, $4)
                    ORDER BY created_at DESC, notification_id DESC
                    LIMIT $5
                    "#,
                    user_id,
                    t,
                    decoded.created_at,
                    decoded.notification_id,
                    limit + 1
                )
                .fetch_all(&self.pool)
                .await?
            } else {
                sqlx::query_as!(
                    NotificationListRow,
                    r#"
                    SELECT
                        notification_id as "notification_id!",
                        notification_type as "notification_type!",
                        payload as "payload!",
                        notification_is_read($1, notification_id, notification_type, created_at) as "is_read!",
                        created_at as "created_at!"
                    FROM notifications_partitioned
                    WHERE user_id = $1
                      AND (created_at, notification_id) < ($2, $3)
                    ORDER BY created_at DESC, notification_id DESC
                    LIMIT $4
                    "#,
                    user_id,
                    decoded.created_at,
                    decoded.notification_id,
                    limit + 1
                )
                .fetch_all(&self.pool)
                .await?
            }
        } else if let Some(ref t) = type_filter {
            sqlx::query_as!(
                NotificationListRow,
                r#"
                SELECT
                    notification_id as "notification_id!",
                    notification_type as "notification_type!",
                    payload as "payload!",
                    notification_is_read($1, notification_id, notification_type, created_at) as "is_read!",
                    created_at as "created_at!"
                FROM notifications_partitioned
                WHERE user_id = $1 AND notification_type = $2
                ORDER BY created_at DESC, notification_id DESC
                LIMIT $3
                "#,
                user_id,
                t,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                NotificationListRow,
                r#"
                SELECT
                    notification_id as "notification_id!",
                    notification_type as "notification_type!",
                    payload as "payload!",
                    notification_is_read($1, notification_id, notification_type, created_at) as "is_read!",
                    created_at as "created_at!"
                FROM notifications_partitioned
                WHERE user_id = $1
                ORDER BY created_at DESC, notification_id DESC
                LIMIT $2
                "#,
                user_id,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        };

        Ok(rows)
    }

    /// 未读数量
    pub async fn unread_count(&self, user_id: i64) -> Result<(i64, i64, i64, i64, i64)> {
        struct UnreadCountRow {
            total: i64,
            social: i64,
            interaction: i64,
            group: i64,
            system: i64,
        }

        let row = sqlx::query_as!(
            UnreadCountRow,
            r#"
            WITH unread AS (
                SELECT notification_type
                FROM notifications_partitioned n
                WHERE n.user_id = $1
                  AND NOT notification_is_read($1, n.notification_id, n.notification_type, n.created_at)
            )
            SELECT
                COUNT(*)::bigint as "total!",
                COUNT(*) FILTER (WHERE notification_type IN ('new_follower', 'mutual_follow'))::bigint as "social!",
                COUNT(*) FILTER (WHERE notification_type IN ('moment_liked', 'moment_commented', 'comment_liked', 'comment_reply', 'mentioned'))::bigint as "interaction!",
                COUNT(*) FILTER (WHERE notification_type IN ('group_invite_received', 'group_application_accepted'))::bigint as "group!",
                COUNT(*) FILTER (WHERE notification_type = 'system_announcement')::bigint as "system!"
            FROM unread
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok((
            row.total,
            row.social,
            row.interaction,
            row.group,
            row.system,
        ))
    }

    /// 标记指定通知已读
    pub async fn mark_as_read(&self, user_id: i64, notification_ids: &[Uuid]) -> Result<()> {
        if notification_ids.is_empty() {
            return Ok(());
        }
        sqlx::query!(
            r#"
            INSERT INTO notification_read_items (user_id, notification_id)
            SELECT $1, unnest($2::uuid[])
            ON CONFLICT (user_id, notification_id) DO NOTHING
            "#,
            user_id,
            notification_ids
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 标记全部已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO notification_read_state (user_id, last_read_at, updated_at)
            VALUES ($1, NOW(), NOW())
            ON CONFLICT (user_id) DO UPDATE SET last_read_at = NOW(), updated_at = NOW()
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
