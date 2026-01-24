use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 评论Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct CommentRow {
    pub id: i64,
    pub comment_id: Uuid,
    // cursor field removed
    pub moment_id: Uuid,
    pub user_id: i64,
    pub parent_id: Option<Uuid>,
    pub reply_to_user_id: Option<i64>,
    pub content: String,
    pub mentions: Option<Vec<i64>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct CommentRepository {
    pool: PgPool,
}

impl CommentRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 是否存在（用于仅做存在性校验的场景，避免拉取整行）
    pub async fn exists(&self, comment_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM comments
                WHERE comment_id = $1 AND is_deleted = FALSE
            ) as "exists!"
            "#,
            comment_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 创建评论
    pub async fn create(
        &self,
        moment_id: Uuid,
        user_id: i64,
        content: String,
        parent_id: Option<Uuid>,
        reply_to_user_id: Option<i64>,
        mentions: Option<Vec<i64>>,
    ) -> Result<CommentRow> {
        let row = sqlx::query_as!(
            CommentRow,
            r#"
            INSERT INTO comments (moment_id, user_id, content, parent_id, reply_to_user_id, mentions)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, comment_id, moment_id, user_id, parent_id, reply_to_user_id, content,
                      mentions, created_at, updated_at, is_deleted, deleted_at
            "#,
            moment_id,
            user_id,
            content,
            parent_id,
            reply_to_user_id,
            mentions.as_deref()
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找评论
    pub async fn find_by_id(&self, comment_id: Uuid) -> Result<Option<CommentRow>> {
        let row = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT id, comment_id, moment_id, user_id, parent_id, reply_to_user_id, content,
                   mentions, created_at, updated_at, is_deleted, deleted_at
            FROM comments
            WHERE comment_id = $1 AND is_deleted = FALSE
            "#,
            comment_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 软删除评论
    pub async fn soft_delete(&self, comment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE comments
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE comment_id = $1
            "#,
            comment_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 点赞评论
    pub async fn like_comment(&self, comment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            INSERT INTO comment_likes (comment_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (comment_id, user_id) DO NOTHING
            "#,
            comment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected() > 0)
    }

    /// 取消点赞评论
    pub async fn unlike_comment(&self, comment_id: Uuid, user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            DELETE FROM comment_likes
            WHERE comment_id = $1 AND user_id = $2
            "#,
            comment_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(result.rows_affected() > 0)
    }

    /// 批量获取评论统计数据（点赞数和回复数）
    pub async fn get_stats_batch(
        &self,
        comment_ids: &[Uuid],
    ) -> Result<std::collections::HashMap<Uuid, (i32, i32)>> {
        if comment_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT 
                c.comment_id as "comment_id!",
                COALESCE(l.likes_count, 0) as "likes_count!",
                COALESCE(r.replies_count, 0) as "replies_count!"
            FROM unnest($1::uuid[]) as c(comment_id)
            LEFT JOIN (
                SELECT comment_id, COUNT(*) as likes_count
                FROM comment_likes
                WHERE comment_id = ANY($1)
                GROUP BY comment_id
            ) l ON c.comment_id = l.comment_id
            LEFT JOIN (
                SELECT parent_id, COUNT(*) as replies_count
                FROM comments
                WHERE is_deleted = FALSE AND parent_id = ANY($1)
                GROUP BY parent_id
            ) r ON c.comment_id = r.parent_id
            "#,
            comment_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.comment_id, (r.likes_count as i32, r.replies_count as i32)))
            .collect())
    }

    /// 按瞬间ID查询评论列表
    pub async fn list_by_moment(
        &self,
        moment_id: Uuid,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<CommentRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let comments = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT id, comment_id, moment_id, user_id, parent_id, reply_to_user_id, content, 
                   mentions, created_at, updated_at, is_deleted, deleted_at
            FROM comments
            WHERE moment_id = $1 AND parent_id IS NULL AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            moment_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = comments.last().map(|c| encode_cursor(&c.id)).transpose()?;
        Ok((comments, next_cursor))
    }

    /// 获取评论的回复列表
    pub async fn list_replies(
        &self,
        parent_id: Uuid,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<CommentRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let replies = sqlx::query_as!(
            CommentRow,
            r#"
            SELECT id, comment_id, moment_id, user_id, parent_id, reply_to_user_id, content, 
                   mentions, created_at, updated_at, is_deleted, deleted_at
            FROM comments
            WHERE parent_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            parent_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = replies.last().map(|c| encode_cursor(&c.id)).transpose()?;
        Ok((replies, next_cursor))
    }

    /// 获取评论点赞列表
    pub async fn list_likes(
        &self,
        comment_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<CommentLikeRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let likes = sqlx::query_as!(
            CommentLikeRow,
            r#"
            SELECT id, user_id, created_at
            FROM comment_likes
            WHERE comment_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            comment_id,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = likes.last().map(|l| encode_cursor(&l.id)).transpose()?;
        Ok((likes, next_cursor))
    }

    /// 列出布隆过滤器用的 comment_id（按 id 分页）
    pub async fn list_comment_ids_after(
        &self,
        last_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, Uuid)>> {
        let rows = sqlx::query!(
            r#"
            SELECT id, comment_id
            FROM comments
            WHERE id > $1 AND is_deleted = FALSE
            ORDER BY id
            LIMIT $2
            "#,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.into_iter().map(|r| (r.id, r.comment_id)).collect())
    }
}

/// 评论点赞行
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct CommentLikeRow {
    pub id: i64,
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}
