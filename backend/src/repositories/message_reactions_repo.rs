//! 消息反应仓库
//!
//! 管理私聊和群聊消息的反应（点赞、表情等）
//! ReactionType 定义在 types::enums 中

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};

/// 消息反应行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct MessageReactionRow {
    pub id: i64,
    pub reaction_id: Uuid,
    pub cursor_id: Uuid,
    pub message_id: Uuid,
    pub user_id: i64,
    pub r#type: String, // 'like', 'love', 'laugh', 'angry', 'sad', 'wow'
    pub created_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct MessageReactionsRepository {
    pool: PgPool,
}

impl MessageReactionsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 添加消息反应
    pub async fn add(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction_type: &str,
    ) -> Result<MessageReactionRow> {
        // 使用 unchecked 版本避免 sqlx 类型检查问题
        let row = sqlx::query_as::<_, MessageReactionRow>(
            r#"
            INSERT INTO message_reactions (message_id, user_id, type)
            VALUES ($1, $2, $3::reaction_type)
            ON CONFLICT (message_id, user_id, type) DO UPDATE 
            SET created_at = message_reactions.created_at
            RETURNING id, reaction_id, cursor_id, message_id, user_id, 
                      type::TEXT as type, created_at
            "#
        )
        .bind(message_id)
        .bind(user_id)
        .bind(reaction_type)
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 移除消息反应
    pub async fn remove(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction_type: &str,
    ) -> Result<()> {
        let result = sqlx::query(
            r#"
            DELETE FROM message_reactions
            WHERE message_id = $1 AND user_id = $2 AND type = $3::reaction_type
            "#
        )
        .bind(message_id)
        .bind(user_id)
        .bind(reaction_type)
        .execute(&self.pool)
        .await?;

        if result.rows_affected() == 0 {
            return Err(AppError::NotFound("反应不存在".to_string()));
        }

        Ok(())
    }

    /// 获取消息的所有反应
    pub async fn list_by_message(
        &self,
        message_id: Uuid,
        reaction_type: Option<&str>,
        limit: i64,
        cursor: Option<Uuid>,
    ) -> Result<Vec<MessageReactionRow>> {
        let cursor_pk = if let Some(cursor_id) = cursor {
            let row: Option<(i64,)> = sqlx::query_as(
                "SELECT id FROM message_reactions WHERE cursor_id = $1"
            )
            .bind(cursor_id)
            .fetch_optional(&self.pool)
            .await?;
            row.map(|r| r.0).unwrap_or(i64::MAX)
        } else {
            i64::MAX
        };

        let rows = if let Some(r_type) = reaction_type {
            sqlx::query_as::<_, MessageReactionRow>(
                r#"
                SELECT id, reaction_id, cursor_id, message_id, user_id, 
                       type::TEXT as type, created_at
                FROM message_reactions
                WHERE message_id = $1 AND type = $2::reaction_type AND id < $3
                ORDER BY id DESC
                LIMIT $4
                "#
            )
            .bind(message_id)
            .bind(r_type)
            .bind(cursor_pk)
            .bind(limit)
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as::<_, MessageReactionRow>(
                r#"
                SELECT id, reaction_id, cursor_id, message_id, user_id, 
                       type::TEXT as type, created_at
                FROM message_reactions
                WHERE message_id = $1 AND id < $2
                ORDER BY id DESC
                LIMIT $3
                "#
            )
            .bind(message_id)
            .bind(cursor_pk)
            .bind(limit)
            .fetch_all(&self.pool)
            .await?
        };

        Ok(rows)
    }
}

