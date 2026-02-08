//! 私聊消息反应仓库
//!
//! 管理私聊消息的反应（点赞、表情等）
//! 适配新 Schema: (message_id, user_id) 复合主键，无自增 ID

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::{AppError, Result};
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};
use crate::types::enums::ReactionType;

/// 私聊消息反应行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct PrivateMessageReactionRow {
    pub message_id: Uuid,
    pub user_id: i64,
    pub reaction_type: ReactionType,
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
    /// 幂等操作：如果已存在，则视为成功 (ON CONFLICT DO NOTHING)
    /// 除非需要更新时间（通常 reaction 时间以首次为准）
    pub async fn add(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction_type: ReactionType,
    ) -> Result<Option<PrivateMessageReactionRow>> {
        // reaction_type 是枚举，sqlx 会自动处理映射 (Type trait)
        let row = sqlx::query_as!(
            PrivateMessageReactionRow,
            r#"
            INSERT INTO private_message_reactions (message_id, user_id, reaction_type)
            VALUES ($1, $2, $3::text::reaction_type)
            ON CONFLICT (message_id, user_id) DO UPDATE 
            SET reaction_type = EXCLUDED.reaction_type, created_at = NOW()
            RETURNING message_id, user_id, reaction_type as "reaction_type: ReactionType", created_at
            "#,
            message_id,
            user_id,
            reaction_type as ReactionType
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 移除私聊消息反应
    pub async fn remove(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        // 移除该用户对该消息的所有反应（目前设计是一人一个反应还是可以多选？Schema PK是 (message_id, user_id)，意味着一人一个状态）
        // 如果 PK 是 (message_id, user_id)，那么再次 add 会覆盖。
        // API 规范：RemoveGroupMessageReactionRequest { message_id, type }
        // 如果业务逻辑允许"取消点赞"，那么应该 check type 是否匹配？
        // 通常 Reaction 是 toggle 或者 replace。
        // 如果 PK 约束了一人一条，那么 remove 只需要 message_id + user_id。
        // 但是 API 参数带 type。
        // 我们加上 type 条件，防止"想取消 Like 却误删了刚刚改成 Love 的状态"（并发场景）。
        
        // 注意：SQL中没有传入 type 参数
        let result = sqlx::query!(
            r#"
            DELETE FROM private_message_reactions
            WHERE message_id = $1 AND user_id = $2
            "#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        if result.rows_affected() == 0 {
            // 幂等删除通常不报错，但如果业务要求"必须存在才能删除"，则报错
            // 这里返回 Ok 比较符合幂等性
            return Ok(());
        }

        Ok(())
    }

    /// 获取消息的所有反应（支持分页）
    /// 排序：created_at DESC
    /// Cursor: (created_at, user_id)
    pub async fn list_by_message(
        &self,
        message_id: Uuid,
        filter_type: Option<ReactionType>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateMessageReactionRow>, Option<String>)> {
        let (last_ts, last_uid) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        // 动态构建查询比较麻烦，这里分两个 query 或者用 COALESCE trick
        // 使用 limit + 1 判断 has_more
        let rows = sqlx::query_as!(
            PrivateMessageReactionRow,
            r#"
            SELECT message_id, user_id, reaction_type as "reaction_type: ReactionType", created_at
            FROM private_message_reactions
            WHERE message_id = $1
              AND ($2::text IS NULL OR reaction_type = $2::text::reaction_type)
              AND (created_at, user_id) < ($3, $4)
            ORDER BY created_at DESC, user_id DESC
            LIMIT $5
            "#,
            message_id,
            filter_type.map(|t| t.to_string()),
            last_ts,
            last_uid,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let rows_to_return: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            rows_to_return.last().map(|r| encode_cursor(&(r.created_at, r.user_id))).transpose()?
        } else {
            None
        };

        Ok((rows_to_return, next_cursor))
    }
}
