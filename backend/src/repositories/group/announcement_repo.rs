use crate::error::Result;
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct GroupAnnouncementRow {
    pub announcement_id: Uuid,
    pub group_id: i64,
    pub title: String,
    pub content: String,
    pub publisher_id: i64,
    pub is_pinned: bool,
    pub created_at: DateTime<Utc>,
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
        group_id: i64,
        title: String,
        content: String,
        publisher_id: i64,
        is_pinned: bool,
    ) -> Result<GroupAnnouncementRow> {
        let announcement = sqlx::query_as!(
            GroupAnnouncementRow,
            r#"
            INSERT INTO group_announcements (group_id, title, content, publisher_id, is_pinned)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING announcement_id, group_id, title, content, publisher_id,
                      COALESCE(is_pinned, FALSE) as "is_pinned!", created_at
            "#,
            group_id,
            title,
            content,
            publisher_id,
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
            SELECT announcement_id, group_id, title, content, publisher_id,
                   COALESCE(is_pinned, FALSE) as "is_pinned!", created_at
            FROM group_announcements
            WHERE announcement_id = $1
            "#,
            announcement_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(announcement)
    }

    /// 根据群组 ID 查找群公告列表
    pub async fn find_by_group_id(
        &self,
        group_id: i64,
        limit: i64,
        _cursor: Option<String>,
    ) -> Result<Vec<GroupAnnouncementRow>> {
        // 简化实现：暂不支持 cursor 分页，直接返回所有公告
        let announcements = sqlx::query_as!(
            GroupAnnouncementRow,
            r#"
            SELECT announcement_id, group_id, title, content, publisher_id,
                   COALESCE(is_pinned, FALSE) as "is_pinned!", created_at
            FROM group_announcements
            WHERE group_id = $1 AND deleted_at IS NULL
            ORDER BY is_pinned DESC, created_at DESC
            LIMIT $2
            "#,
            group_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(announcements)
    }

    /// 根据群组 ID 查找群公告列表（别名方法）
    pub async fn list_by_group(
        &self,
        group_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<Vec<GroupAnnouncementRow>> {
        self.find_by_group_id(group_id, limit, cursor).await
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
            announcement_id,
            title,
            content,
            is_pinned
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除群公告 (软删除)
    pub async fn delete(&self, announcement_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_announcements
            SET deleted_at = NOW()
            WHERE announcement_id = $1
            "#,
            announcement_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
