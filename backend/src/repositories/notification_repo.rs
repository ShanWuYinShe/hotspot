use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "notification_type", rename_all = "snake_case")]
pub enum NotificationType {
    Like,
    Comment,
    Reply,
    Follow,
    Mention,
    System,
    GroupInvite,
    GroupApply,
    GroupKick,
    GroupQuit,
    GroupDismiss,
    GroupAdmin,
}

impl std::str::FromStr for NotificationType {
    type Err = String;
    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        match s {
            "like" => Ok(NotificationType::Like),
            "comment" => Ok(NotificationType::Comment),
            "reply" => Ok(NotificationType::Reply),
            "follow" => Ok(NotificationType::Follow),
            "mention" => Ok(NotificationType::Mention),
            "system" => Ok(NotificationType::System),
            "group_invite" => Ok(NotificationType::GroupInvite),
            "group_apply" => Ok(NotificationType::GroupApply),
            "group_kick" => Ok(NotificationType::GroupKick),
            "group_quit" => Ok(NotificationType::GroupQuit),
            "group_dismiss" => Ok(NotificationType::GroupDismiss),
            "group_admin" => Ok(NotificationType::GroupAdmin),
            _ => Err(format!("Invalid notification type: {}", s)),
        }
    }
}

impl std::fmt::Display for NotificationType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let s = match self {
            NotificationType::Like => "like",
            NotificationType::Comment => "comment",
            NotificationType::Reply => "reply",
            NotificationType::Follow => "follow",
            NotificationType::Mention => "mention",
            NotificationType::System => "system",
            NotificationType::GroupInvite => "group_invite",
            NotificationType::GroupApply => "group_apply",
            NotificationType::GroupKick => "group_kick",
            NotificationType::GroupQuit => "group_quit",
            NotificationType::GroupDismiss => "group_dismiss",
            NotificationType::GroupAdmin => "group_admin",
        };
        write!(f, "{}", s)
    }
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct NotificationRow {
    pub id: i64,
    pub notification_id: Uuid,
    pub user_id: i64,
    pub r#type: NotificationType,
    pub title: String,
    pub content: String,
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
    #[allow(clippy::too_many_arguments)]
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
        let row = sqlx::query_as!(
            NotificationRow,
            r#"
            INSERT INTO notifications (
                user_id, type, title, content, 
                sender_id, sender_nickname, sender_avatar_url,
                target_type, target_id, reference_id, target_preview
            )
            VALUES ($1, $2::notification_type, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING 
                id, notification_id, user_id, 
                type as "type: NotificationType", 
                title, content, 
                sender_id, sender_nickname, sender_avatar_url,
                target_type, target_id, reference_id, target_preview,
                is_read, created_at
            "#,
            user_id,
            r#type as NotificationType,
            title,
            content,
            sender_id,
            sender_nickname,
            sender_avatar_url,
            target_type,
            target_id,
            reference_id,
            target_preview
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 获取通知列表（分页）
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
        type_filters: Option<Vec<String>>,
    ) -> Result<(Vec<NotificationRow>, Option<String>)> {
        let (last_created_at, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query_as!(
            NotificationRow,
            r#"
            SELECT 
                id, notification_id, user_id, 
                type as "type: NotificationType", 
                title, content, 
                sender_id, sender_nickname, sender_avatar_url,
                target_type, target_id, reference_id, target_preview,
                is_read, created_at
            FROM notifications
            WHERE user_id = $1 
              AND (created_at, id) < ($2, $3)
              AND ($4::text[] IS NULL OR type::text = ANY($4))
            ORDER BY created_at DESC, id DESC
            LIMIT $5
            "#,
            user_id,
            last_created_at,
            last_id,
            type_filters.as_deref(),
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|r| encode_cursor(&(r.created_at, r.id)))
            .transpose()?;

        Ok((rows, next_cursor))
    }

    /// 标记通知已读（单个/批量）
    pub async fn mark_read(&self, user_id: i64, notification_ids: &[Uuid]) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE notifications
            SET is_read = TRUE
            WHERE user_id = $1 AND notification_id = ANY($2)
            "#,
            user_id,
            notification_ids
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 标记指定类型的所有通知为已读
    pub async fn mark_type_read(&self, user_id: i64, r#type: NotificationType) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE notifications
            SET is_read = TRUE
            WHERE user_id = $1 AND type = $2::notification_type AND is_read = FALSE
            "#,
            user_id,
            r#type as NotificationType
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 标记所有通知已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE notifications
            SET is_read = TRUE
            WHERE user_id = $1 AND is_read = FALSE
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 获取未读计数
    pub async fn count_unread(&self, user_id: i64) -> Result<i64> {
        let count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM notifications
            WHERE user_id = $1 AND is_read = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count;
        Ok(count)
    }

    /// 获取未读计数细分
    pub async fn count_unread_breakdown(
        &self,
        user_id: i64,
    ) -> Result<Vec<(NotificationType, i64)>> {
        let rows = sqlx::query!(
            r#"
            SELECT type as "type: NotificationType", COUNT(*) as "count!"
            FROM notifications
            WHERE user_id = $1 AND is_read = FALSE
            GROUP BY type
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.r#type, r.count)).collect())
    }
}
