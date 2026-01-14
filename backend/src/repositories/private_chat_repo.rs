use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 私聊Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatRow {
    pub id: i64,
    // private_chat_id 已移除
    pub user_a_id: i64,
    pub user_b_id: i64,
    // 陌生人消息限制字段
    pub user_a_can_message_freely: bool,
    pub user_b_can_message_freely: bool,
    pub user_a_sent_count: i32,
    pub user_b_sent_count: i32,
    pub created_at: DateTime<Utc>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 私聊设置Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateChatSettingsRow {
    pub chat_id: i64, // 关联内部 ID
    pub user_id: i64,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_archived: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct PrivateChatRepository {
    pool: PgPool,
}

impl PrivateChatRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建私聊
    pub async fn create(&self, user_a_id: i64, user_b_id: i64) -> Result<PrivateChatRow> {
        // 确保 user_a_id < user_b_id 以保持一致性（虽然表定义不强制，但业务上最好保持有序）
        let (u1, u2) = if user_a_id < user_b_id {
            (user_a_id, user_b_id)
        } else {
            (user_b_id, user_a_id)
        };

        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            INSERT INTO private_chats (user_a_id, user_b_id)
            VALUES ($1, $2)
            RETURNING id, user_a_id, user_b_id,
                      user_a_can_message_freely, user_b_can_message_freely,
                      user_a_sent_count, user_b_sent_count,
                      created_at, is_deleted, deleted_at
            "#,
            u1,
            u2
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID获取私聊
    pub async fn find_by_id(&self, chat_id: i64) -> Result<Option<PrivateChatRow>> {
        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, user_a_id, user_b_id,
                   user_a_can_message_freely, user_b_can_message_freely,
                   user_a_sent_count, user_b_sent_count,
                   created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE id = $1 AND is_deleted = FALSE
            "#,
            chat_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按双方用户获取私聊
    pub async fn find_by_users(
        &self,
        user_id_1: i64,
        user_id_2: i64,
    ) -> Result<Option<PrivateChatRow>> {
        // SQL 中使用 OR 或者标准化顺序查询
        // 这里假设表中存储时没有强制顺序（虽然create里做了），所以用 OR 查询更保险
        let row = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, user_a_id, user_b_id,
                   user_a_can_message_freely, user_b_can_message_freely,
                   user_a_sent_count, user_b_sent_count,
                   created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE ((user_a_id = $1 AND user_b_id = $2) OR (user_a_id = $2 AND user_b_id = $1))
              AND is_deleted = FALSE
            "#,
            user_id_1,
            user_id_2
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 检查用户是否在私聊中
    pub async fn is_user_in_chat(&self, chat_id: i64, user_id: i64) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM private_chats
                WHERE id = $1
                  AND (user_a_id = $2 OR user_b_id = $2)
                  AND is_deleted = FALSE
            ) as "exists!"
            "#,
            chat_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(exists)
    }

    /// 软删除私聊
    pub async fn soft_delete(&self, chat_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE private_chats
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE id = $1
            "#,
            chat_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取用户的所有私聊列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
        include_archived: bool,
    ) -> Result<(Vec<PrivateChatRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            PrivateChatRow,
            r#"
            SELECT id, user_a_id, user_b_id,
                user_a_can_message_freely, user_b_can_message_freely,
                user_a_sent_count, user_b_sent_count,
                created_at, is_deleted, deleted_at
            FROM private_chats
            WHERE (user_a_id = $1 OR user_b_id = $1)
              AND id < $2
              AND is_deleted = FALSE
              AND (
                $4::bool = TRUE
                OR NOT EXISTS (
                    SELECT 1
                    FROM private_chat_settings
                    WHERE private_chat_settings.chat_id = private_chats.id
                      AND private_chat_settings.user_id = $1
                      AND private_chat_settings.is_archived = TRUE
                )
              )
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit,
            include_archived
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|last| encode_cursor(&last.id))
            .transpose()?;

        Ok((rows, next_cursor))
    }

    /// 获取私聊设置
    pub async fn get_settings(
        &self,
        chat_id: i64,
        user_id: i64,
    ) -> Result<Option<PrivateChatSettingsRow>> {
        let row = sqlx::query_as!(
            PrivateChatSettingsRow,
            r#"
            SELECT chat_id, user_id, is_muted, is_pinned, is_archived, created_at, updated_at
            FROM private_chat_settings
            WHERE chat_id = $1 AND user_id = $2
            "#,
            chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 创建或更新私聊设置
    pub async fn upsert_settings(
        &self,
        chat_id: i64,
        user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
        is_archived: Option<bool>,
    ) -> Result<()> {
        // 使用 ON CONFLICT 实现 upsert，避免 if-else 分支
        sqlx::query!(
            r#"
            INSERT INTO private_chat_settings (chat_id, user_id, is_muted, is_pinned, is_archived)
            VALUES ($1, $2, COALESCE($3, FALSE), COALESCE($4, FALSE), COALESCE($5, FALSE))
            ON CONFLICT (chat_id, user_id) DO UPDATE
            SET is_muted = COALESCE($3, private_chat_settings.is_muted),
                is_pinned = COALESCE($4, private_chat_settings.is_pinned),
                is_archived = COALESCE($5, private_chat_settings.is_archived),
                updated_at = NOW()
            "#,
            chat_id,
            user_id,
            is_muted,
            is_pinned,
            is_archived
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取用户的消息限制信息
    /// 返回 (can_message_freely, sent_count)
    pub async fn get_message_limit(&self, chat_id: i64, user_id: i64) -> Result<(bool, i32)> {
        let row = sqlx::query!(
            r#"
            SELECT user_a_id, user_b_id,
                   user_a_can_message_freely, user_b_can_message_freely,
                   user_a_sent_count, user_b_sent_count
            FROM private_chats
            WHERE id = $1 AND is_deleted = FALSE
            "#,
            chat_id
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| crate::error::AppError::NotFound("私聊不存在".to_string()))?;

        if row.user_a_id == user_id {
            Ok((row.user_a_can_message_freely, row.user_a_sent_count))
        } else if row.user_b_id == user_id {
            Ok((row.user_b_can_message_freely, row.user_b_sent_count))
        } else {
            Err(crate::error::AppError::Forbidden(
                "用户不在此私聊中".to_string(),
            ))
        }
    }

    /// 增加用户发送消息计数
    pub async fn increment_sent_count(&self, chat_id: i64, user_id: i64) -> Result<i32> {
        let row = sqlx::query!(
            r#"
            UPDATE private_chats
            SET user_a_sent_count = CASE WHEN user_a_id = $2 THEN user_a_sent_count + 1 ELSE user_a_sent_count END,
                user_b_sent_count = CASE WHEN user_b_id = $2 THEN user_b_sent_count + 1 ELSE user_b_sent_count END
            WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2) AND is_deleted = FALSE
            RETURNING user_a_id, user_a_sent_count, user_b_sent_count
            "#,
            chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        match row {
            Some(r) => {
                if r.user_a_id == user_id {
                    Ok(r.user_a_sent_count)
                } else {
                    Ok(r.user_b_sent_count)
                }
            }
            None => Err(crate::error::AppError::NotFound(
                "私聊不存在或您不在此私聊中".to_string(),
            )),
        }
    }

    /// 解锁指定用户的消息限制
    /// 当 A 回复 B 时，Service 层会调用此方法解锁 B (user_id = B)
    pub async fn unlock_user_messaging_limit(&self, chat_id: i64, user_id: i64) -> Result<()> {
        let result = sqlx::query!(
            r#"
            UPDATE private_chats
            SET user_a_can_message_freely = CASE WHEN user_a_id = $2 THEN TRUE ELSE user_a_can_message_freely END,
                user_b_can_message_freely = CASE WHEN user_b_id = $2 THEN TRUE ELSE user_b_can_message_freely END
            WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2) AND is_deleted = FALSE
            "#,
            chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        if result.rows_affected() == 0 {
            return Err(crate::error::AppError::NotFound(
                "私聊不存在或该用户不在此私聊中".to_string(),
            ));
        }

        Ok(())
    }
}
