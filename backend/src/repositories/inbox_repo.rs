use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::inbox::{InboxItem, DeviceToken},
};

pub struct InboxRepository<'a> {
    db: &'a PgPool,
}

impl<'a> InboxRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建收件箱项
    pub async fn create(&self, item: &InboxItem) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO inbox_items (user_id, category, type, data, is_read)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id
            "#,
            item.user_id,
            item.category,
            item.item_type,
            item.data,
            item.is_read,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.id)
    }

    /// 获取用户的收件箱列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        category: Option<&str>,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<InboxItem>> {
        let items = if let Some(cat) = category {
            sqlx::query_as!(
                InboxItem,
                r#"
                SELECT id, user_id, category, type as item_type, data, is_read, created_at
                FROM inbox_items
                WHERE user_id = $1 AND category = $2
                ORDER BY created_at DESC
                LIMIT $3 OFFSET $4
                "#,
                user_id,
                cat,
                limit,
                offset
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        } else {
            sqlx::query_as!(
                InboxItem,
                r#"
                SELECT id, user_id, category, type as item_type, data, is_read, created_at
                FROM inbox_items
                WHERE user_id = $1
                ORDER BY created_at DESC
                LIMIT $2 OFFSET $3
                "#,
                user_id,
                limit,
                offset
            )
            .fetch_all(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
        };

        Ok(items)
    }

    /// 标记为已读
    pub async fn mark_as_read(&self, user_id: i64, item_ids: &[i64]) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE inbox_items
            SET is_read = true
            WHERE user_id = $1 AND id = ANY($2)
            "#,
            user_id,
            item_ids
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 标记所有为已读
    pub async fn mark_all_as_read(&self, user_id: i64, category: Option<&str>) -> Result<()> {
        if let Some(cat) = category {
            sqlx::query!(
                r#"
                UPDATE inbox_items
                SET is_read = true
                WHERE user_id = $1 AND category = $2 AND is_read = false
                "#,
                user_id,
                cat
            )
            .execute(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        } else {
            sqlx::query!(
                r#"
                UPDATE inbox_items
                SET is_read = true
                WHERE user_id = $1 AND is_read = false
                "#,
                user_id
            )
            .execute(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?;
        }

        Ok(())
    }

    /// 删除收件箱项
    pub async fn delete(&self, user_id: i64, item_ids: &[i64]) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM inbox_items
            WHERE user_id = $1 AND id = ANY($2)
            "#,
            user_id,
            item_ids
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取未读数量
    pub async fn count_unread(&self, user_id: i64, category: Option<&str>) -> Result<i64> {
        let count = if let Some(cat) = category {
            sqlx::query!(
                r#"
                SELECT COUNT(*) as "count!"
                FROM inbox_items
                WHERE user_id = $1 AND category = $2 AND is_read = false
                "#,
                user_id,
                cat
            )
            .fetch_one(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
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
            .fetch_one(self.db)
            .await
            .map_err(|e| AppError::DatabaseError(e.to_string()))?
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
            ON CONFLICT (user_id, token) DO UPDATE SET platform = $3
            RETURNING id
            "#,
            user_id,
            token,
            platform
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

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
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取用户的所有设备Token
    pub async fn get_user_device_tokens(&self, user_id: i64) -> Result<Vec<DeviceToken>> {
        let tokens = sqlx::query_as!(
            DeviceToken,
            r#"
            SELECT * FROM device_tokens
            WHERE user_id = $1
            ORDER BY created_at DESC
            "#,
            user_id
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(tokens)
    }
}

