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
}

