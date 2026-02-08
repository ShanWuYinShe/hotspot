//! 群聊消息反应仓库
//!
//! 适配新 Schema: (message_id, user_id) 复合主键

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};
use crate::types::enums::ReactionType;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMessageReactionRow {
    pub message_id: Uuid,
    pub user_id: i64,
    pub reaction_type: ReactionType,
    pub created_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct GroupMessageReactionsRepository {
    pool: PgPool,
}

impl GroupMessageReactionsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 添加群聊消息反应
    pub async fn add(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction_type: ReactionType,
    ) -> Result<Option<GroupMessageReactionRow>> {
        let row = sqlx::query_as!(
            GroupMessageReactionRow,
            r#"
            INSERT INTO group_message_reactions (message_id, user_id, reaction_type)
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

    /// 移除群聊消息反应
    pub async fn remove(&self, message_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM group_message_reactions
            WHERE message_id = $1 AND user_id = $2
            "#,
            message_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 列出消息反应
    pub async fn list_by_message(
        &self,
        message_id: Uuid,
        filter_type: Option<ReactionType>,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupMessageReactionRow>, Option<String>)> {
        let (last_ts, last_uid) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query_as!(
            GroupMessageReactionRow,
            r#"
            SELECT message_id, user_id, reaction_type as "reaction_type: ReactionType", created_at
            FROM group_message_reactions
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
