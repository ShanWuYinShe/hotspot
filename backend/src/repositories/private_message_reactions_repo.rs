//! 私聊消息反应仓库
//!
//! 管理私聊消息的反应（点赞、表情等）

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 私聊消息反应行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessageReactionRow {
    pub id: i64,
    pub reaction_id: Uuid,
    pub message_id: Uuid,
    pub user_id: i64,
    pub r#type: String, // 'like', 'love', 'laugh', 'angry', 'sad', 'wow'
    pub created_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct PrivateMessageReactionsRepository {
    pool: PgPool,
}

impl PrivateMessageReactionsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 添加私聊消息反应
    pub async fn add(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction_type: &str,
    ) -> Result<PrivateMessageReactionRow> {
        // 强约束：重复添加反应应返回冲突，而不是"更新/保持原值"
        let row = sqlx::query_as!(
            PrivateMessageReactionRow,
            r#"
            INSERT INTO private_message_reactions (message_id, user_id, type)
            VALUES ($1, $2, $3::TEXT::reaction_type)
            ON CONFLICT (message_id, user_id, type) DO NOTHING
            RETURNING id, reaction_id, message_id, user_id, 
                      type::TEXT as "type!", created_at
            "#,
            message_id,
            user_id,
            reaction_type
        )
        .fetch_optional(&self.pool)
        .await?;

        row.ok_or_else(|| AppError::Conflict("已添加过该反应".to_string()))
    }

    /// 移除私聊消息反应
    pub async fn remove(&self, message_id: Uuid, user_id: i64, reaction_type: &str) -> Result<()> {
        let result = sqlx::query!(
            r#"
            DELETE FROM private_message_reactions
            WHERE message_id = $1 AND user_id = $2 AND type = $3::TEXT::reaction_type
            "#,
            message_id,
            user_id,
            reaction_type
        )
        .execute(&self.pool)
        .await?;

        if result.rows_affected() == 0 {
            return Err(AppError::Conflict("尚未添加该反应".to_string()));
        }

        Ok(())
    }

    /// 获取消息的所有反应（按类型过滤）
    pub async fn list_by_message(
        &self,
        message_id: Uuid,
        reaction_type: Option<&str>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateMessageReactionRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        // 查询 limit + 1 条以判断是否有更多数据
        let rows = if let Some(r_type) = reaction_type {
            sqlx::query_as!(
                PrivateMessageReactionRow,
                r#"
                SELECT id, reaction_id, message_id, user_id, 
                       type::TEXT as "type!", created_at
                FROM private_message_reactions
                WHERE message_id = $1 AND type = $2::TEXT::reaction_type AND id < $3
                ORDER BY id DESC
                LIMIT $4
                "#,
                message_id,
                r_type,
                last_id,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                PrivateMessageReactionRow,
                r#"
                SELECT id, reaction_id, message_id, user_id, 
                       type::TEXT as "type!", created_at
                FROM private_message_reactions
                WHERE message_id = $1 AND id < $2
                ORDER BY id DESC
                LIMIT $3
                "#,
                message_id,
                last_id,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?
        };

        crate::utils::pagination::paginate_with_limit_plus_one(rows, limit as i32, |r| {
            encode_cursor(&r.id)
        })
    }
}
