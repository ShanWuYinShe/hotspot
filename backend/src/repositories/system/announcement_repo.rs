use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor};

/// 公告Row - 严格按照数据库 schema
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct AnnouncementRow {
    pub announcement_id: Uuid,
    pub title: String,
    pub content: String,
    pub r#type: String,                   // database uses 'type' field
    pub target_user_type: Option<String>, // all, user, admin
    pub severity: String,
    pub action_url: Option<String>,
    pub send_push: bool,
    pub start_at: DateTime<Utc>,
    pub end_at: Option<DateTime<Utc>>,
    pub is_published: bool,
    pub publisher_id: Option<i64>,
    pub push_sent_at: Option<DateTime<Utc>>,
    pub view_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// 带发布者信息的公告Row
#[derive(Debug, Clone, FromRow)]
pub struct AnnouncementDetailRow {
    pub announcement_id: Uuid,
    pub title: String,
    pub content: String,
    pub severity: String,
    pub action_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, FromRow)]
pub struct AdminAnnouncementRow {
    pub announcement_id: Uuid,
    pub title: String,
    pub content: String,
    pub severity: String,
    pub action_url: Option<String>,
    pub send_push: bool,
    pub push_sent_at: Option<DateTime<Utc>>,
    pub view_count: i32,
    pub publisher_id: Option<i64>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct AnnouncementRepository {
    pool: PgPool,
}

impl AnnouncementRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建公告
    pub async fn create(
        &self,
        title: &str,
        content: &str,
        severity: &str,
        action_url: Option<&str>,
        send_push: bool,
        publisher_id: Option<i64>,
    ) -> Result<AnnouncementRow> {
        let row = sqlx::query_as!(
            AnnouncementRow,
            r#"
            INSERT INTO announcements (title, content, severity, action_url, send_push, publisher_id, is_published)
            VALUES ($1, $2, $3::text::announcement_severity, $4, $5, $6, TRUE)
            RETURNING announcement_id as "announcement_id!",
                      title as "title!",
                      content as "content!",
                      COALESCE(type, 'system') as "type!",
                      target_user_type as "target_user_type?",
                      COALESCE(severity, 'info')::text as "severity!",
                      action_url as "action_url?",
                      COALESCE(send_push, FALSE) as "send_push!",
                      start_at as "start_at!",
                      end_at as "end_at?",
                      COALESCE(is_published, FALSE) as "is_published!",
                      publisher_id as "publisher_id?",
                      push_sent_at as "push_sent_at?",
                      COALESCE(view_count, 0) as "view_count!",
                      created_at as "created_at!",
                      updated_at as "updated_at!"
            "#,
            title,
            content,
            severity,
            action_url,
            send_push,
            publisher_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找公告
    /// 轻量级检查公告是否存在
    pub async fn exists(&self, announcement_id: Uuid) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM announcements WHERE announcement_id = $1)",
            announcement_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.unwrap_or(false))
    }

    /// 更新公告（仅更新非 None 的字段，None 保持原值）
    pub async fn update(
        &self,
        announcement_id: Uuid,
        title: Option<&str>,
        content: Option<&str>,
        severity: Option<&str>,
        action_url: Option<&str>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE announcements SET
                title = COALESCE($2, title),
                content = COALESCE($3, content),
                severity = COALESCE($4::text::announcement_severity, severity),
                action_url = COALESCE($5, action_url),
                updated_at = NOW()
            WHERE announcement_id = $1
            "#,
            announcement_id,
            title,
            content,
            severity,
            action_url
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除公告
    pub async fn delete(&self, announcement_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM announcements WHERE announcement_id = $1",
            announcement_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 列出公告（分页，可选按 severity 筛选）
    pub async fn list(
        &self,
        severity: Option<&str>,
        limit: i64,
        cursor: Option<&str>,
    ) -> crate::utils::pagination::Paginated<AnnouncementDetailRow> {
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());

        let start_time = cursor_time.unwrap_or_else(Utc::now);
        let query_limit = limit + 1;

        let mut rows = if let Some(sev) = severity {
            sqlx::query_as!(
                AnnouncementDetailRow,
                r#"
                SELECT 
                    a.announcement_id as "announcement_id!",
                    a.title as "title!",
                    a.content as "content!",
                    COALESCE(a.severity, 'info')::text as "severity!",
                    a.action_url as "action_url?",
                    a.created_at as "created_at!"
                FROM announcements a
                WHERE a.created_at < $1
                  AND (COALESCE(a.severity, 'info')::text = $2)
                ORDER BY a.created_at DESC
                LIMIT $3
                "#,
                start_time,
                sev,
                query_limit
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                AnnouncementDetailRow,
                r#"
                SELECT 
                    a.announcement_id as "announcement_id!",
                    a.title as "title!",
                    a.content as "content!",
                    COALESCE(a.severity, 'info')::text as "severity!",
                    a.action_url as "action_url?",
                    a.created_at as "created_at!"
                FROM announcements a
                WHERE a.created_at < $1
                ORDER BY a.created_at DESC
                LIMIT $2
                "#,
                start_time,
                query_limit
            )
            .fetch_all(&self.pool)
            .await?
        };

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    pub async fn list_admin(
        &self,
        severity: Option<&str>,
        limit: i64,
        cursor: Option<&str>,
    ) -> crate::utils::pagination::Paginated<AdminAnnouncementRow> {
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());
        let start_time = cursor_time.unwrap_or_else(Utc::now);
        let query_limit = limit + 1;

        let mut rows = if let Some(sev) = severity {
            sqlx::query_as!(
                AdminAnnouncementRow,
                r#"
                SELECT 
                    a.announcement_id as "announcement_id!",
                    a.title as "title!",
                    a.content as "content!",
                    COALESCE(a.severity, 'info')::text as "severity!",
                    a.action_url as "action_url?",
                    COALESCE(a.send_push, FALSE) as "send_push!",
                    a.push_sent_at as "push_sent_at?",
                    COALESCE(a.view_count, 0) as "view_count!",
                    a.publisher_id as "publisher_id?",
                    a.created_at as "created_at!"
                FROM announcements a
                WHERE a.created_at < $1
                  AND (COALESCE(a.severity, 'info')::text = $2)
                ORDER BY a.created_at DESC
                LIMIT $3
                "#,
                start_time,
                sev,
                query_limit
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                AdminAnnouncementRow,
                r#"
                SELECT 
                    a.announcement_id as "announcement_id!",
                    a.title as "title!",
                    a.content as "content!",
                    COALESCE(a.severity, 'info')::text as "severity!",
                    a.action_url as "action_url?",
                    COALESCE(a.send_push, FALSE) as "send_push!",
                    a.push_sent_at as "push_sent_at?",
                    COALESCE(a.view_count, 0) as "view_count!",
                    a.publisher_id as "publisher_id?",
                    a.created_at as "created_at!"
                FROM announcements a
                WHERE a.created_at < $1
                ORDER BY a.created_at DESC
                LIMIT $2
                "#,
                start_time,
                query_limit
            )
            .fetch_all(&self.pool)
            .await?
        };

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }
}
