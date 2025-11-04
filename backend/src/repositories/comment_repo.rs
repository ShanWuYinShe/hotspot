use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::comment::Comment,
};

pub struct CommentRepository {
    db: PgPool,
}

impl CommentRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 创建评论
    pub async fn create(&self, comment: &Comment) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO comments (
                comment_id, user_id, moment_id,
                content, parent_id, reply_to_user_id
            )
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING comment_id
            "#,
            comment.comment_id,
            comment.user_id,
            comment.moment_id,
            comment.content,
            comment.parent_id,
            comment.reply_to_user_id,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.comment_id)
    }

    /// 根据ID查找评论
    pub async fn find_by_id(&self, comment_id: Uuid) -> Result<Option<Comment>> {
        let comment = sqlx::query_as!(
            Comment,
            r#"
            SELECT comment_id, moment_id, user_id, content, parent_id, 
                   reply_to_user_id, created_at, updated_at,
                   is_deleted, deleted_at
            FROM comments
            WHERE comment_id = $1 AND is_deleted = FALSE
            "#,
            comment_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(comment)
    }

    /// 删除评论（软删除）
    pub async fn soft_delete(&self, comment_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE comments
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE comment_id = $1
            "#,
            comment_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取瞬间的评论列表（顶级评论）（游标分页）
    /// cursor: comment_id (UUID字符串)
    pub async fn list_by_moment(
        &self,
        moment_id: Uuid,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Comment>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let comment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM comments WHERE comment_id = $1 AND is_deleted = FALSE",
                comment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT comment_id, moment_id, user_id, content, parent_id, 
                   reply_to_user_id, created_at, updated_at,
                   is_deleted, deleted_at
            FROM comments
            WHERE 
                moment_id = $1 
                AND parent_id IS NULL
                AND is_deleted = FALSE
                AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            moment_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(comments)
    }

    /// 获取评论的回复列表（游标分页）
    /// cursor: comment_id (UUID字符串)
    pub async fn list_replies(
        &self,
        parent_id: Uuid,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Comment>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let comment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM comments WHERE comment_id = $1 AND is_deleted = FALSE",
                comment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页（回复按创建时间升序，id也是升序）
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT comment_id, moment_id, user_id, content, parent_id, 
                   reply_to_user_id, created_at, updated_at,
                   is_deleted, deleted_at
            FROM comments
            WHERE parent_id = $1 
            AND is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id ASC
            LIMIT $3
            "#,
            parent_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count)
    }

    /// 获取回复数
    pub async fn get_reply_count(&self, comment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE parent_id = $1
            "#,
            comment_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.exists)
    }

    /// 获取用户的评论列表（游标分页）
    /// cursor: comment_id (UUID字符串)
    pub async fn list_by_user(
        &self,
        user_id: i64,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Comment>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let comment_uuid = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM comments WHERE comment_id = $1 AND is_deleted = FALSE",
                comment_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let comments = sqlx::query_as!(
            Comment,
            r#"
            SELECT comment_id, moment_id, user_id, content, parent_id, 
                   reply_to_user_id, created_at, updated_at,
                   is_deleted, deleted_at
            FROM comments
            WHERE user_id = $1 
            AND is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(comments)
    }

    /// 获取瞬间的评论总数
    pub async fn count_by_moment(&self, moment_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM comments
            WHERE moment_id = $1
            "#,
            moment_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count)
    }
}
