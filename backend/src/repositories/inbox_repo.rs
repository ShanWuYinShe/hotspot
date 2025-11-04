use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::inbox::{InboxItem, DeviceToken},
    types::enums::InboxCategory,
};

#[derive(Debug, Clone)]
pub struct InboxRepository {
    db: PgPool,
}

impl InboxRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 创建收件箱项
    pub async fn create(&self, item: &InboxItem) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO inbox_items (
                item_id, user_id, category, item_type, 
                title, content, data, action_url, severity, is_read
            )
            VALUES ($1, $2, $3::inbox_category, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id
            "#,
            item.item_id,
            item.user_id,
            item.category as InboxCategory,
            item.item_type,
            item.title,
            item.content,
            item.data,
            item.action_url,
            item.severity,
            item.is_read,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.id)
    }

    /// 获取用户的收件箱列表（游标分页）
    /// cursor: item_id (UUID字符串)
    pub async fn list_by_user(
        &self,
        user_id: i64,
        category: Option<InboxCategory>,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<InboxItem>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let item_uuid = uuid::Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM inbox_items WHERE item_id = $1",
                item_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let items = if let Some(cat) = category {
            sqlx::query_as!(
                InboxItem,
                r#"
                SELECT id, item_id, user_id, category as "category: InboxCategory", item_type, 
                       title, content, data, action_url, severity, 
                       created_at, is_read
                FROM inbox_items
                WHERE user_id = $1 
                AND (category = $2::inbox_category OR $2::inbox_category = 'any'::inbox_category)
                AND ($3::BIGINT IS NULL OR id > $3::BIGINT)
                ORDER BY id DESC
                LIMIT $4
                "#,
                user_id,
                cat as InboxCategory,
                cursor_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
        } else {
            sqlx::query_as!(
                InboxItem,
                r#"
                SELECT id, item_id, user_id, category as "category: InboxCategory", item_type, 
                       title, content, data, action_url, severity, 
                       created_at, is_read
                FROM inbox_items
                WHERE user_id = $1
                AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
                ORDER BY id DESC
                LIMIT $3
                "#,
                user_id,
                cursor_id,
                limit
            )
            .fetch_all(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
        };

        Ok(items)
    }

    /// 标记为已读
    /// 标记已读（通过item_id）
    pub async fn mark_as_read(&self, user_id: i64, item_ids: &[uuid::Uuid]) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE inbox_items
            SET is_read = true
            WHERE user_id = $1 AND item_id = ANY($2)
            "#,
            user_id,
            item_ids
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(result.rows_affected())
    }

    /// 标记所有为已读
    pub async fn mark_all_as_read(&self, user_id: i64, category: Option<InboxCategory>) -> Result<u64> {
        let result = if let Some(cat) = category {
            sqlx::query!(
                r#"
                UPDATE inbox_items
                SET is_read = true
                WHERE user_id = $1 
                AND (category = $2::inbox_category OR $2::inbox_category = 'any'::inbox_category)
                AND is_read = false
                "#,
                user_id,
                cat as InboxCategory
            )
            .execute(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
        } else {
            sqlx::query!(
                r#"
                UPDATE inbox_items
                SET is_read = true
                WHERE user_id = $1 AND is_read = false
                "#,
                user_id
            )
            .execute(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
        };

        Ok(result.rows_affected())
    }

    /// 删除收件箱项（通过item_id）
    pub async fn delete(&self, user_id: i64, item_ids: &[uuid::Uuid]) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM inbox_items
            WHERE user_id = $1 AND item_id = ANY($2)
            "#,
            user_id,
            item_ids
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(result.rows_affected())
    }

    /// 获取未读数量
    pub async fn get_unread_count(&self, user_id: i64) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as count
            FROM inbox_items
            WHERE user_id = $1 AND is_read = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count.unwrap_or(0))
    }

    /// 获取未读数量
    pub async fn count_unread(&self, user_id: i64, category: Option<InboxCategory>) -> Result<i64> {
        let count = if let Some(cat) = category {
            sqlx::query!(
                r#"
                SELECT COUNT(*) as "count!"
                FROM inbox_items
                WHERE user_id = $1 
                AND (category = $2::inbox_category OR $2::inbox_category = 'any'::inbox_category)
                AND is_read = false
                "#,
                user_id,
                cat as InboxCategory
            )
            .fetch_one(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
            .count
        } else {
            sqlx::query!(
                r#"
                SELECT COUNT(*) as "count!"
                FROM inbox_items
                WHERE user_id = $1 AND is_read = false
                "#,
                user_id
            )
            .fetch_one(&self.db)
            .await
            .map_err(|e| AppError::from(e))?
            .count
        };

        Ok(count)
    }

    /// 注册设备Token
    pub async fn register_device_token(
        &self,
        user_id: i64,
        token: &str,
        platform: &str,
    ) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO device_tokens (user_id, token, platform)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id, platform, token) 
            DO UPDATE SET updated_at = NOW()
            RETURNING id
            "#,
            user_id,
            token,
            platform
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.id)
    }

    /// 注销设备Token
    pub async fn unregister_device_token(&self, user_id: i64, token: &str) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM device_tokens
            WHERE user_id = $1 AND token = $2
            "#,
            user_id,
            token
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取用户的所有设备Token
    pub async fn get_user_device_tokens(&self, user_id: i64) -> Result<Vec<DeviceToken>> {
        let tokens = sqlx::query_as!(
            DeviceToken,
            r#"
            SELECT id, user_id, token, platform, created_at
            FROM device_tokens
            WHERE user_id = $1
            ORDER BY created_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(tokens)
    }
}
