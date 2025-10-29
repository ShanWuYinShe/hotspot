use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::comment::Comment,
};

pub struct CommentRepository<'a> {
    db: &'a PgPool,
}

impl<'a> CommentRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建评论
    pub async fn create(&self, comment: &Comment) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO comments (
                comment_id, user_id, target_id, target_type,
                content, parent_id
            )
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING comment_id
            "#,
            comment.comment_id,
            comment.user_id,
            comment.target_id,
            comment.target_type,
            comment.content,
            comment.parent_id,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.comment_id)
    }

    /// 根据ID查找评论
    pub async fn find_by_id(&self, comment_id: Uuid) -> Result<Option<Comment>> {
        let comment = sqlx::query_as!(
            Comment,
            r#"
            SELECT * FROM comments WHERE comment_id = $1 AND deleted_at IS NULL
            "#,
            comment_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(comment)
    }

    /// 软删除评论
    pub async fn soft_delete(&self, comment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE comments
            SET deleted_at = NOW()
            WHERE comment_id = $1
            "#,
            comment_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取目标的评论列表（顶级评论）
    pub async fn list_by_target(
        &self,
        target_id: &str,
        target_type: &str,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Comment>> {
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT * FROM comments
            WHERE 
                target_id = $1 
                AND target_type = $2 
                AND parent_id IS NULL
                AND deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $3 OFFSET $4
            "#,
            target_id,
            target_type,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(comments)
    }

    /// 获取评论的回复列表
    pub async fn list_replies(
        &self,
        parent_id: Uuid,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Comment>> {
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT * FROM comments
            WHERE parent_id = $1 AND deleted_at IS NULL
            ORDER BY created_at ASC
            LIMIT $2 OFFSET $3
            "#,
            parent_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(comments)
    }

    /// 点赞评论
    pub async fn like_comment(&self, comment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO comment_likes (comment_id, user_id)
            VALUES ($1, $2)
            ON CONFLICT (comment_id, user_id) DO NOTHING
            "#,
            comment_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 取消点赞
    pub async fn unlike_comment(&self, comment_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM comment_likes
            WHERE comment_id = $1 AND user_id = $2
            "#,
            comment_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 获取点赞数
    pub async fn get_like_count(&self, comment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comment_likes
            WHERE comment_id = $1
            "#,
            comment_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }

    /// 获取回复数
    pub async fn get_reply_count(&self, comment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE parent_id = $1 AND deleted_at IS NULL
            "#,
            comment_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }

    /// 检查是否已点赞
    pub async fn is_liked(&self, comment_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM comment_likes
                WHERE comment_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            comment_id,
            user_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.exists)
    }

    /// 获取用户的评论列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Comment>> {
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT * FROM comments
            WHERE user_id = $1 AND deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(comments)
    }

    /// 获取目标的评论总数
    pub async fn count_by_target(&self, target_id: &str, target_type: &str) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE target_id = $1 AND target_type = $2 AND deleted_at IS NULL
            "#,
            target_id,
            target_type
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }
}

