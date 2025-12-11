use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserFollowRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub follower_id: i64,
    pub following_id: i64,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct UserFollowsRepository {
    pool: PgPool,
}

impl UserFollowsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(&self, follower_id: i64, following_id: i64) -> Result<UserFollowRow> {
        let row = sqlx::query_as!(
            UserFollowRow,
            r#"
            INSERT INTO user_follows (follower_id, following_id)
            VALUES ($1, $2)
            ON CONFLICT (follower_id, following_id) DO NOTHING
            RETURNING id, cursor_id, follower_id, following_id, created_at
            "#,
            follower_id,
            following_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    pub async fn delete(&self, follower_id: i64, following_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM user_follows
            WHERE follower_id = $1 AND following_id = $2
            "#,
            follower_id,
            following_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn exists(&self, follower_id: i64, following_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM user_follows
                WHERE follower_id = $1 AND following_id = $2
            ) as "exists!"
            "#,
            follower_id,
            following_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    pub async fn count_followers(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_follows
            WHERE following_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    pub async fn count_following(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_follows
            WHERE follower_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 解析 UUID cursor 为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM user_follows WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 获取用户关注列表（分页）
    pub async fn list_following(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<i64>, Option<Uuid>)> {
        #[derive(sqlx::FromRow)]
        struct FollowingRow {
            following_id: i64,
            cursor_id: Uuid,
        }

        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<FollowingRow> = sqlx::query_as!(
            FollowingRow,
            r#"
            SELECT following_id, cursor_id
            FROM user_follows
            WHERE follower_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let user_ids = rows.iter().map(|r| r.following_id).collect();
        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((user_ids, next_cursor))
    }

    /// 获取用户粉丝列表（分页）
    pub async fn list_followers(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<i64>, Option<Uuid>)> {
        #[derive(sqlx::FromRow)]
        struct FollowerRow {
            follower_id: i64,
            cursor_id: Uuid,
        }

        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<FollowerRow> = sqlx::query_as!(
            FollowerRow,
            r#"
            SELECT follower_id, cursor_id
            FROM user_follows
            WHERE following_id = $1 AND id < $2
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let user_ids = rows.iter().map(|r| r.follower_id).collect();
        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((user_ids, next_cursor))
    }

    /// 检查是否互相关注（好友）
    pub async fn are_mutual_follows(&self, user_a_id: i64, user_b_id: i64) -> Result<bool> {
        let a_follows_b = self.exists(user_a_id, user_b_id).await?;
        let b_follows_a = self.exists(user_b_id, user_a_id).await?;
        Ok(a_follows_b && b_follows_a)
    }

    /// 统计互相关注（好友）数量
    pub async fn count_mutual_follows(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_follows f1
            INNER JOIN user_follows f2 ON f1.following_id = f2.follower_id AND f1.follower_id = f2.following_id
            WHERE f1.follower_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }

    /// 获取互相关注（好友）列表（分页）
    pub async fn list_mutual_follows(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<MutualFollowRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<MutualFollowRow> = sqlx::query_as!(
            MutualFollowRow,
            r#"
            SELECT f1.following_id as user_id, f1.cursor_id, f1.created_at as followed_at, f2.created_at as followed_back_at
            FROM user_follows f1
            INNER JOIN user_follows f2 ON f1.following_id = f2.follower_id AND f1.follower_id = f2.following_id
            WHERE f1.follower_id = $1 AND f1.id < $2
            ORDER BY f1.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((rows, next_cursor))
    }

    /// 获取用户关注列表（包含详细信息）
    pub async fn list_following_detail(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<FollowingDetailRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<FollowingDetailRow> = sqlx::query_as!(
            FollowingDetailRow,
            r#"
            SELECT 
                f.following_id,
                f.cursor_id,
                f.created_at,
                EXISTS(
                    SELECT 1 FROM user_follows f2 
                    WHERE f2.follower_id = f.following_id AND f2.following_id = f.follower_id
                ) as "is_mutual!"
            FROM user_follows f
            WHERE f.follower_id = $1 AND f.id < $2
            ORDER BY f.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((rows, next_cursor))
    }

    /// 获取用户粉丝列表（包含详细信息）
    pub async fn list_followers_detail(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<FollowerDetailRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows: Vec<FollowerDetailRow> = sqlx::query_as!(
            FollowerDetailRow,
            r#"
            SELECT 
                f.follower_id,
                f.cursor_id,
                f.created_at,
                EXISTS(
                    SELECT 1 FROM user_follows f2 
                    WHERE f2.follower_id = f.following_id AND f2.following_id = f.follower_id
                ) as "is_following_back!"
            FROM user_follows f
            WHERE f.following_id = $1 AND f.id < $2
            ORDER BY f.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows.last().map(|r| r.cursor_id);
        Ok((rows, next_cursor))
    }
}

/// 互相关注（好友）查询结果
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct MutualFollowRow {
    pub user_id: i64,
    pub cursor_id: Uuid,
    pub followed_at: DateTime<Utc>,      // 我关注对方的时间
    pub followed_back_at: DateTime<Utc>, // 对方关注我的时间
}

/// 关注列表查询结果（包含额外信息）
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct FollowingDetailRow {
    pub following_id: i64,
    pub cursor_id: Uuid,
    pub created_at: DateTime<Utc>,       // 关注时间
    pub is_mutual: bool,                 // 是否互相关注
}

/// 粉丝列表查询结果（包含额外信息）
#[derive(Debug, Clone, sqlx::FromRow)]
pub struct FollowerDetailRow {
    pub follower_id: i64,
    pub cursor_id: Uuid,
    pub created_at: DateTime<Utc>,       // 对方关注我的时间
    pub is_following_back: bool,         // 我是否回关
}

