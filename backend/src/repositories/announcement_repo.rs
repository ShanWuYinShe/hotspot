use chrono::{DateTime, Utc};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};
use crate::utils::pagination::{decode_cursor, encode_cursor};

#[derive(Debug, Clone, FromRow)]
pub struct AnnouncementRow {
    pub id: i64,
    pub announcement_id: Uuid,
    pub title: String,
    pub content: String,
    pub severity: String,
    pub action_url: Option<String>,
    pub created_by: i64,
    pub created_at: DateTime<Utc>,
    pub send_push: bool,
    // Join fields (optional)
    pub creator_nickname: Option<String>,
    pub creator_avatar_url: Option<String>,
}

#[derive(Clone)]
pub struct AnnouncementRepository {
    pool: PgPool,
}

impl AnnouncementRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(
        &self,
        title: &str,
        content: &str,
        severity: &str,
        action_url: Option<&str>,
        send_push: bool,
        created_by: i64,
    ) -> Result<AnnouncementRow> {
        let row = sqlx::query_as!(
            AnnouncementRow,
            r#"
            INSERT INTO system_announcements (title, content, severity, action_url, send_push, created_by)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, announcement_id, title, content, severity, action_url, created_by, created_at, send_push,
            NULL::text as creator_nickname, NULL::text as creator_avatar_url
            "#,
            title,
            content,
            severity,
            action_url,
            send_push,
            created_by
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::Database)?;

        Ok(row)
    }

    pub async fn list(
        &self,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<AnnouncementRow>, Option<String>)> {
        let (last_created_at, last_id) = if let Some(cursor_str) = cursor {
            decode_cursor::<(DateTime<Utc>, i64)>(&cursor_str)?
        } else {
            (Utc::now(), i64::MAX)
        };

        let rows = sqlx::query_as!(
            AnnouncementRow,
            r#"
            SELECT 
                a.id, a.announcement_id, a.title, a.content, a.severity, a.action_url, a.created_by, a.created_at, a.send_push,
                u.nickname as creator_nickname, u.avatar_url as creator_avatar_url
            FROM system_announcements a
            LEFT JOIN users u ON a.created_by = u.id
            WHERE a.is_deleted = FALSE
            AND (a.created_at, a.id) < ($1, $2)
            ORDER BY a.created_at DESC, a.id DESC
            LIMIT $3
            "#,
            last_created_at,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::Database)?;

        let mut items = rows;
        let mut next_cursor = None;
        if items.len() > limit as usize {
            items.pop();
            if let Some(last) = items.last() {
                next_cursor = Some(encode_cursor(&(last.created_at, last.id))?);
            }
        }

        Ok((items, next_cursor))
    }

    pub async fn delete(&self, announcement_id: Uuid) -> Result<()> {
        let result = sqlx::query!(
            r#"
            UPDATE system_announcements
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE announcement_id = $1 AND is_deleted = FALSE
            "#,
            announcement_id
        )
        .execute(&self.pool)
        .await
        .map_err(AppError::Database)?;

        if result.rows_affected() == 0 {
            return Err(AppError::NotFound("公告不存在".to_string()));
        }

        Ok(())
    }

    pub async fn find_by_id(&self, announcement_id: Uuid) -> Result<Option<AnnouncementRow>> {
        let row = sqlx::query_as!(
            AnnouncementRow,
            r#"
            SELECT 
                a.id, a.announcement_id, a.title, a.content, a.severity, a.action_url, a.created_by, a.created_at, a.send_push,
                u.nickname as creator_nickname, u.avatar_url as creator_avatar_url
            FROM system_announcements a
            LEFT JOIN users u ON a.created_by = u.user_id
            WHERE a.announcement_id = $1 AND a.is_deleted = FALSE
            "#,
            announcement_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::Database)?;

        Ok(row)
    }
}
