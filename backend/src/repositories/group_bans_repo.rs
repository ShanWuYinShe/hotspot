use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

/// 群组封禁记录 Row（用于JOIN查询，包含用户信息）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupBanRow {
    pub id: i64,
    pub ban_id: Uuid,
    pub cursor_id: Uuid,
    pub group_chat_id: Uuid,
    pub user_id: i64,
    pub banned_by: i64,
    pub reason: Option<String>,
    pub banned_at: DateTime<Utc>,
    pub unbanned_at: Option<DateTime<Utc>>,
    pub unbanned_by: Option<i64>,
    // Joined fields from users table
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub user_type: Option<crate::types::enums::UserType>,
}

#[derive(Debug, Clone)]
pub struct GroupBansRepository {
    pool: PgPool,
}

impl GroupBansRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 添加封禁记录
    pub async fn ban(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        banned_by: i64,
        reason: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO group_bans (group_chat_id, user_id, banned_by, reason)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (group_chat_id, user_id) DO UPDATE SET
                banned_by = $3,
                reason = $4,
                banned_at = NOW(),
                unbanned_at = NULL,
                unbanned_by = NULL
            "#,
            group_chat_id,
            user_id,
            banned_by,
            reason
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 解除封禁
    pub async fn unban(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        unbanned_by: i64,
    ) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            UPDATE group_bans
            SET unbanned_at = NOW(), unbanned_by = $3
            WHERE group_chat_id = $1 AND user_id = $2 AND unbanned_at IS NULL
            "#,
            group_chat_id,
            user_id,
            unbanned_by
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    /// 检查用户是否被封禁
    pub async fn is_banned(&self, group_chat_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_bans 
                WHERE group_chat_id = $1 AND user_id = $2 AND unbanned_at IS NULL
            ) as "exists!"
            "#,
            group_chat_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 获取群组封禁列表
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<GroupBanRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;

        let rows = sqlx::query_as!(
            GroupBanRow,
            r#"
            SELECT b.id, b.ban_id, b.cursor_id, b.group_chat_id, b.user_id, 
                   b.banned_by, b.reason, b.banned_at, b.unbanned_at, b.unbanned_by,
                   u.nickname, u.avatar_url, u.user_type as "user_type: crate::types::enums::UserType"
            FROM group_bans b
            LEFT JOIN users u ON u.user_id = b.user_id
            WHERE b.group_chat_id = $1 AND b.unbanned_at IS NULL AND b.id < $2
            ORDER BY b.id DESC
            LIMIT $3
            "#,
            group_chat_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM group_bans WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("Invalid cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }
}

