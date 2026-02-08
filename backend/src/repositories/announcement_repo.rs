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
    pub r#type: String,  // database uses 'type' field
    pub target_user_type: Option<String>,  // all, user, admin
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
    pub r#type: String,
    pub target_user_type: Option<String>,
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
    pub publisher_nickname: Option<String>,
    pub publisher_avatar_url: Option<String>,
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
    pub async fn find_by_id(&self, announcement_id: Uuid) -> Result<Option<AnnouncementRow>> {
        let row = sqlx::query_as!(
            AnnouncementRow,
            r#"
            SELECT announcement_id as "announcement_id!",
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
            FROM announcements
            WHERE announcement_id = $1
            "#,
            announcement_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
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

    /// 列出公告（分页）
    pub async fn list(
        &self,
        limit: i64,
        cursor: Option<&str>,
    ) -> Result<(Vec<AnnouncementDetailRow>, Option<String>)> {
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());

        let start_time = cursor_time.unwrap_or_else(Utc::now);
        let query_limit = limit + 1;

        let rows = sqlx::query_as!(
            AnnouncementDetailRow,
            r#"
            SELECT 
                a.announcement_id as "announcement_id!",
                a.title as "title!",
                a.content as "content!",
                COALESCE(a.type, 'system') as "type!",
                a.target_user_type as "target_user_type?",
                COALESCE(a.severity, 'info')::text as "severity!",
                a.action_url as "action_url?",
                COALESCE(a.send_push, FALSE) as "send_push!",
                a.start_at as "start_at!",
                a.end_at as "end_at?",
                COALESCE(a.is_published, FALSE) as "is_published!",
                a.publisher_id as "publisher_id?",
                a.push_sent_at as "push_sent_at?",
                COALESCE(a.view_count, 0) as "view_count!",
                a.created_at as "created_at!",
                a.updated_at as "updated_at!",
                u.nickname as "publisher_nickname?",
                u.avatar_url as "publisher_avatar_url?"
            FROM announcements a
            LEFT JOIN users u ON a.publisher_id = u.user_id
            WHERE a.created_at < $1
            ORDER BY a.created_at DESC
            LIMIT $2
            "#,
            start_time,
            query_limit
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items = if has_more {
            rows[..limit as usize].to_vec()
        } else {
            rows
        };

        let next_cursor = if has_more {
            items.last().map(|r| encode_cursor(&r.created_at)).transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 列出系统公告（对用户可见）
    pub async fn list_public(
        &self,
        target_user_type: Option<&str>,
        limit: i64,
    ) -> Result<Vec<AnnouncementDetailRow>> {
        let rows = sqlx::query_as!(
            AnnouncementDetailRow,
            r#"
            SELECT 
                a.announcement_id as "announcement_id!",
                a.title as "title!",
                a.content as "content!",
                COALESCE(a.type, 'system') as "type!",
                a.target_user_type as "target_user_type?",
                COALESCE(a.severity, 'info')::text as "severity!",
                a.action_url as "action_url?",
                COALESCE(a.send_push, FALSE) as "send_push!",
                a.start_at as "start_at!",
                a.end_at as "end_at?",
                COALESCE(a.is_published, FALSE) as "is_published!",
                a.publisher_id as "publisher_id?",
                a.push_sent_at as "push_sent_at?",
                COALESCE(a.view_count, 0) as "view_count!",
                a.created_at as "created_at!",
                a.updated_at as "updated_at!",
                u.nickname as "publisher_nickname?",
                u.avatar_url as "publisher_avatar_url?"
            FROM announcements a
            LEFT JOIN users u ON a.publisher_id = u.user_id
            WHERE a.is_published = TRUE 
              AND a.start_at <= NOW()
              AND (a.end_at IS NULL OR a.end_at > NOW())
              AND (a.target_user_type = $1 OR $1 IS NULL)
            ORDER BY a.start_at DESC
            LIMIT $2
            "#,
            target_user_type,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }
}
