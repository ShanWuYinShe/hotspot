//! 评论仓库层实现
//!
//! 适配新 Schema: comment_id (UUID), user_id (BIGINT), 分区表 comments_partitioned

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated};

/// 评论行数据
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct CommentRow {
    pub comment_id: Uuid,
    pub moment_id: Uuid,
    pub user_id: i64,
    pub content: String,
    pub parent_id: Option<Uuid>,
    pub reply_to_comment_id: Option<Uuid>,
    pub reply_to_user_id: Option<i64>,
    pub mentions: Option<Vec<i64>>,
    pub like_count: i32,
    pub reply_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,

    // Joined fields
    pub user_nickname: Option<String>,
    pub user_avatar_url: Option<String>,
}

#[derive(Debug, Clone, FromRow)]
pub struct CommentLikeRow {
    pub user_id: i64,
    pub user_nickname: Option<String>,
    pub user_avatar_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct CommentRepository {
    pool: PgPool,
}

impl CommentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建评论
    pub async fn create(
        &self,
        moment_id: Uuid,
        user_id: i64,
        content: String,
        parent_id: Option<Uuid>,
        reply_to_comment_id: Option<Uuid>,
        reply_to_user_id: Option<i64>,
        mentions: Option<Vec<i64>>,
    ) -> Result<CommentRow> {
        let row = sqlx::query_as!(
            CommentRow,
            r#"
            INSERT INTO comments_partitioned (
                moment_id, user_id, content, parent_id, reply_to_comment_id, reply_to_user_id, mentions
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING comment_id as "comment_id!", moment_id as "moment_id!", user_id as "user_id!", content as "content!",
                      parent_id, reply_to_comment_id, reply_to_user_id, mentions,
                      0::int as "like_count!", 0::int as "reply_count!",
                      created_at as "created_at!", updated_at as "updated_at!", deleted_at,
                      NULL::text as "user_nickname?", NULL::text as "user_avatar_url?"
            "#,
            moment_id,
            user_id,
            content,
            parent_id,
            reply_to_comment_id,
            reply_to_user_id,
            mentions.as_deref()
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据 ID 查找
    pub async fn find_by_id(&self, comment_id: Uuid) -> Result<Option<CommentRow>> {
        let row = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT c.comment_id as "comment_id!", c.moment_id as "moment_id!", c.user_id as "user_id!", c.content as "content!",
                   c.parent_id, c.reply_to_comment_id, c.reply_to_user_id, c.mentions,
                   COALESCE(cs.likes_count, 0)::int as "like_count!", COALESCE(cs.replies_count, 0)::int as "reply_count!",
                   c.created_at as "created_at!", c.updated_at as "updated_at!", c.deleted_at,
                   u.nickname as "user_nickname?", u.avatar_url as "user_avatar_url?"
            FROM comments_partitioned c
            LEFT JOIN comment_stats cs ON c.comment_id = cs.comment_id
            JOIN users u ON c.user_id = u.user_id
            WHERE c.comment_id = $1 AND c.deleted_at IS NULL
            "#,
            comment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn exists(&self, comment_id: Uuid) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM comments_partitioned WHERE comment_id = $1 AND deleted_at IS NULL) as \"exists!\"",
            comment_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 列出动态下的评论（根评论）
    pub async fn list_by_moment(
        &self,
        moment_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<CommentRow> {
        let last_ts = cursor
            .and_then(|c| decode_cursor::<DateTime<Utc>>(&c).ok())
            .unwrap_or(DateTime::<Utc>::MAX_UTC);

        let mut rows = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT c.comment_id as "comment_id!", c.moment_id as "moment_id!", c.user_id as "user_id!", c.content as "content!",
                   c.parent_id, c.reply_to_comment_id, c.reply_to_user_id, c.mentions,
                   COALESCE(cs.likes_count, 0)::int as "like_count!", COALESCE(cs.replies_count, 0)::int as "reply_count!",
                   c.created_at as "created_at!", c.updated_at as "updated_at!", c.deleted_at,
                   u.nickname as "user_nickname?", u.avatar_url as "user_avatar_url?"
            FROM comments_partitioned c
            LEFT JOIN comment_stats cs ON c.comment_id = cs.comment_id
            JOIN users u ON c.user_id = u.user_id
            WHERE c.moment_id = $1 
              AND c.parent_id IS NULL 
              AND c.deleted_at IS NULL
              AND c.created_at < $2
            ORDER BY c.created_at DESC
            LIMIT $3
            "#,
            moment_id,
            last_ts,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 列出评论的子评论（回复）
    pub async fn list_replies(
        &self,
        parent_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<CommentRow> {
        let last_ts = cursor
            .and_then(|c| decode_cursor::<DateTime<Utc>>(&c).ok())
            .unwrap_or(DateTime::<Utc>::MIN_UTC);

        let mut rows = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT c.comment_id as "comment_id!", c.moment_id as "moment_id!", c.user_id as "user_id!", c.content as "content!",
                   c.parent_id, c.reply_to_comment_id, c.reply_to_user_id, c.mentions,
                   COALESCE(cs.likes_count, 0)::int as "like_count!", COALESCE(cs.replies_count, 0)::int as "reply_count!",
                   c.created_at as "created_at!", c.updated_at as "updated_at!", c.deleted_at,
                   u.nickname as "user_nickname?", u.avatar_url as "user_avatar_url?"
            FROM comments_partitioned c
            LEFT JOIN comment_stats cs ON c.comment_id = cs.comment_id
            JOIN users u ON c.user_id = u.user_id
            WHERE c.parent_id = $1
              AND c.deleted_at IS NULL
              AND c.created_at > $2
            ORDER BY c.created_at ASC
            LIMIT $3
            "#,
            parent_id,
            last_ts,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 评论点赞列表
    pub async fn list_likes(
        &self,
        comment_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<CommentLikeRow> {
        let last_ts = cursor
            .and_then(|c| decode_cursor::<DateTime<Utc>>(&c).ok())
            .unwrap_or(DateTime::<Utc>::MAX_UTC);

        let mut rows = sqlx::query_as!(
            CommentLikeRow,
            r#"
            SELECT cr.user_id as "user_id!",
                   u.nickname as "user_nickname?",
                   u.avatar_url as "user_avatar_url?",
                   cr.created_at as "created_at!"
            FROM comment_reactions cr
            JOIN users u ON cr.user_id = u.user_id
            WHERE cr.comment_id = $1
              AND cr.created_at < $2
              AND cr.type = 'like'
            ORDER BY cr.created_at DESC
            LIMIT $3
            "#,
            comment_id,
            last_ts,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.created_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    pub async fn batch_check_liked(
        &self,
        comment_ids: &[Uuid],
        user_id: i64,
    ) -> Result<std::collections::HashSet<Uuid>> {
        if comment_ids.is_empty() {
            return Ok(std::collections::HashSet::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT comment_id as "comment_id!"
            FROM comment_reactions
            WHERE user_id = $1 AND comment_id = ANY($2) AND type = 'like'
            "#,
            user_id,
            comment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.comment_id).collect())
    }

    pub async fn insert_like(&self, comment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            "INSERT INTO comment_reactions (comment_id, user_id, type) VALUES ($1, $2, 'like') ON CONFLICT DO NOTHING",
            comment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    pub async fn delete_like(&self, comment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM comment_reactions WHERE comment_id = $1 AND user_id = $2 AND type = 'like'",
            comment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除评论
    pub async fn delete(&self, comment_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE comments_partitioned SET deleted_at = NOW() WHERE comment_id = $1",
            comment_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn list_comment_ids_after(
        &self,
        last_row_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, Uuid)>> {
        #[derive(Debug, FromRow)]
        struct CommentIdMapRow {
            row_id: i64,
            comment_id: Uuid,
        }

        let rows = sqlx::query_as!(
            CommentIdMapRow,
            r#"
            SELECT row_id as "row_id!", comment_id as "comment_id!"
            FROM (
                SELECT row_number() OVER (ORDER BY created_at ASC, comment_id ASC)::bigint as row_id,
                       comment_id
                FROM comment_id_map
            ) ranked
            WHERE row_id > $1
            ORDER BY row_id ASC
            LIMIT $2
            "#,
            last_row_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.row_id, r.comment_id)).collect())
    }
}
