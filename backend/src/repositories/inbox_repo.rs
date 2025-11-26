use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::{NotificationCategory, NotificationSeverity};

/// 收件箱项Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct InboxItemRow {
    pub id: i64,
    pub inbox_item_id: Uuid,
    pub user_id: i64,
    pub category: NotificationCategory,
    pub r#type: String,
    pub title: Option<String>,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub action_url: Option<String>,
    pub severity: Option<NotificationSeverity>,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct InboxRepository {
    pool: PgPool,
}

impl InboxRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM inbox_items WHERE inbox_item_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 创建收件箱项
    pub async fn create(
        &self,
        user_id: i64,
        category: NotificationCategory,
        r#type: String,
        title: Option<String>,
        content: String,
        data: Option<serde_json::Value>,
        action_url: Option<String>,
        severity: Option<NotificationSeverity>,
    ) -> Result<InboxItemRow> {
        let row = sqlx::query_as!(
            InboxItemRow,
            r#"
            INSERT INTO inbox_items (user_id, category, type, title, content, data, action_url, severity)
            VALUES ($1, $2::notification_category, $3, $4, $5, $6, $7, $8::notification_severity)
            RETURNING id, inbox_item_id, user_id,
                      category as "category: NotificationCategory",
                      type as "type!", title, content, data, action_url,
                      severity as "severity: NotificationSeverity",
                      is_read, created_at
            "#,
            user_id,
            category as NotificationCategory,
            r#type,
            title,
            content,
            data,
            action_url,
            severity as Option<NotificationSeverity>
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找收件箱项
    pub async fn find_by_id(&self, inbox_item_id: Uuid) -> Result<Option<InboxItemRow>> {
        let row = sqlx::query_as!(
            InboxItemRow,
            r#"
            SELECT id, inbox_item_id, user_id,
                   category as "category: NotificationCategory",
                   type as "type!", title, content, data, action_url,
                   severity as "severity: NotificationSeverity",
                   is_read, created_at
            FROM inbox_items
            WHERE inbox_item_id = $1
            "#,
            inbox_item_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 标记为已读 (单个)
    pub async fn mark_as_read(&self, inbox_item_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE inbox_items
            SET is_read = TRUE
            WHERE inbox_item_id = $1
            "#,
            inbox_item_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 批量标记为已读
    pub async fn mark_batch_as_read(&self, user_id: i64, inbox_item_ids: &[Uuid]) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE inbox_items
            SET is_read = TRUE
            WHERE user_id = $1 AND inbox_item_id = ANY($2)
            "#,
            user_id,
            inbox_item_ids
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    /// 标记所有为已读
    pub async fn mark_all_as_read(&self, user_id: i64, category: Option<NotificationCategory>) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE inbox_items
            SET is_read = TRUE
            WHERE user_id = $1 
              AND ($2::notification_category IS NULL OR category = $2::notification_category)
              AND is_read = FALSE
            "#,
            user_id,
            category as Option<NotificationCategory>
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    /// 删除收件箱项 (单个)
    pub async fn delete(&self, inbox_item_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM inbox_items
            WHERE inbox_item_id = $1
            "#,
            inbox_item_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 批量删除
    pub async fn delete_batch(&self, user_id: i64, inbox_item_ids: &[Uuid]) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM inbox_items
            WHERE user_id = $1 AND inbox_item_id = ANY($2)
            "#,
            user_id,
            inbox_item_ids
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    /// 获取未读通知数
    pub async fn get_unread_count(&self, user_id: i64) -> Result<i64> {
        let count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM inbox_items
            WHERE user_id = $1 AND is_read = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?
        .count;

        Ok(count)
    }

    /// 获取收件箱列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        category: Option<NotificationCategory>,
        is_read: Option<bool>,
        limit: i32,
        cursor: Option<Uuid>
    ) -> Result<Vec<InboxItemRow>> {
        // 使用 resolve_cursor 将 UUID 转换为自增主键
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        // 使用单一 SQL 查询，通过 IS NULL OR 处理可选过滤条件
        let items = sqlx::query_as!(
            InboxItemRow,
            r#"
            SELECT id, inbox_item_id, user_id,
                   category as "category: NotificationCategory",
                   type as "type!", title, content, data, action_url,
                   severity as "severity: NotificationSeverity",
                   is_read, created_at
            FROM inbox_items
            WHERE user_id = $1 AND id < $2
              AND ($3::notification_category IS NULL OR category = $3::notification_category)
              AND ($4::boolean IS NULL OR is_read = $4)
            ORDER BY id DESC
            LIMIT $5
            "#,
            user_id,
            cursor_pk,
            category as Option<NotificationCategory>,
            is_read,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(items)
    }
}
