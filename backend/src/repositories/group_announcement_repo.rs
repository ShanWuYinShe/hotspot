use anyhow::Result;
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct GroupAnnouncementRow {
    pub id: i64,
    pub announcement_id: String, // VARCHAR(36) in DB
    pub group_chat_id: String,   // VARCHAR(36) in DB
    pub title: String,
    pub content: String,
    pub created_by: i64,
    pub is_pinned: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct GroupAnnouncementRepository {
    pool: PgPool,
}

impl GroupAnnouncementRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建群公告
    pub async fn create(
        &self,
        group_chat_id: Uuid,
        title: String,
        content: String,
        created_by: i64,
        is_pinned: bool,
    ) -> Result<GroupAnnouncementRow> {
        let announcement = sqlx::query_as!(
            GroupAnnouncementRow,
            r#"
            INSERT INTO group_announcements (group_chat_id, title, content, created_by, is_pinned)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id, announcement_id, group_chat_id, title, content, created_by, is_pinned, created_at, updated_at
            "#,
            &group_chat_id.to_string(),
            title,
            content,
            created_by,
            is_pinned
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(announcement)
    }

    /// 根据 ID 查找群公告
    pub async fn find_by_id(&self, announcement_id: Uuid) -> Result<Option<GroupAnnouncementRow>> {
        let announcement = sqlx::query_as!(
            GroupAnnouncementRow,
            r#"
            SELECT id, announcement_id, group_chat_id, title, content, created_by, is_pinned, created_at, updated_at
            FROM group_announcements
            WHERE announcement_id = $1
            "#,
            announcement_id.to_string()
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(announcement)
    }

    /// 根据群组 ID 查找群公告列表
    pub async fn find_by_group_id(
        &self,
        group_chat_id: Uuid,
        limit: i64,
        _cursor: Option<String>,
    ) -> Result<Vec<GroupAnnouncementRow>> {
        // 简化实现：暂不支持 cursor 分页，直接返回所有公告
        let announcements = sqlx::query_as!(
            GroupAnnouncementRow,
            r#"
            SELECT id, announcement_id, group_chat_id, title, content, created_by, is_pinned, created_at, updated_at
            FROM group_announcements
            WHERE group_chat_id = $1
            ORDER BY is_pinned DESC, created_at DESC
            LIMIT $2
            "#,
            &group_chat_id.to_string(),
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(announcements)
    }

    /// 根据群组 ID 查找群公告列表（别名方法）
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<Vec<GroupAnnouncementRow>> {
        self.find_by_group_id(group_chat_id, limit, cursor).await
    }

    /// 更新群公告
    pub async fn update(
        &self,
        announcement_id: Uuid,
        title: Option<String>,
        content: Option<String>,
        is_pinned: Option<bool>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_announcements
            SET title = COALESCE($2, title),
                content = COALESCE($3, content),
                is_pinned = COALESCE($4, is_pinned),
                updated_at = NOW()
            WHERE announcement_id = $1
            "#,
            &announcement_id.to_string(),
            title,
            content,
            is_pinned
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除群公告
    pub async fn delete(&self, announcement_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM group_announcements
            WHERE announcement_id = $1
            "#,
            announcement_id.to_string()
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
