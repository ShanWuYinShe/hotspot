use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, TimeCursor};
use chrono::{DateTime, Utc};
use sqlx::PgPool;

#[derive(Debug, Clone)]
pub struct UserFollowsRepository {
    pool: PgPool,
}

#[derive(Debug, Clone)]
pub struct FollowEdgeRow {
    pub user_id: i64,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct FriendEdgeRow {
    pub user_id: i64,
    pub followed_at: DateTime<Utc>,
    pub followed_back_at: DateTime<Utc>,
}

impl UserFollowsRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn follow(&self, follower_id: i64, following_id: i64) -> Result<()> {
        sqlx::query!(
            "INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
            follower_id, following_id
        ).execute(&self.pool).await?;
        Ok(())
    }

    pub async fn unfollow(&self, follower_id: i64, following_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM follows WHERE follower_id = $1 AND following_id = $2",
            follower_id,
            following_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    pub async fn is_following(&self, follower_id: i64, following_id: i64) -> Result<bool> {
        let row = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)",
            follower_id,
            following_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(row.unwrap_or(false))
    }

    pub async fn list_following(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<FollowEdgeRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT following_id as "user_id!", created_at as "created_at!"
            FROM follows
            WHERE follower_id = $1
              AND (created_at, following_id) < ($2, $3)
            ORDER BY created_at DESC, following_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<FollowEdgeRow> = rows
            .into_iter()
            .take(limit as usize)
            .map(|r| FollowEdgeRow {
                user_id: r.user_id,
                created_at: r.created_at,
            })
            .collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn list_followers(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<FollowEdgeRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT follower_id as "user_id!", created_at as "created_at!"
            FROM follows
            WHERE following_id = $1
              AND (created_at, follower_id) < ($2, $3)
            ORDER BY created_at DESC, follower_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<FollowEdgeRow> = rows
            .into_iter()
            .take(limit as usize)
            .map(|r| FollowEdgeRow {
                user_id: r.user_id,
                created_at: r.created_at,
            })
            .collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.created_at, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    pub async fn list_friends(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<FriendEdgeRow>, Option<String>)> {
        let (last_ts, last_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT f1.following_id as "user_id!",
                   f1.created_at as "followed_at!",
                   f2.created_at as "followed_back_at!"
            FROM follows f1
            JOIN follows f2
              ON f1.following_id = f2.follower_id
             AND f2.following_id = f1.follower_id
            WHERE f1.follower_id = $1
              AND (f2.created_at, f1.following_id) < ($2, $3)
            ORDER BY f2.created_at DESC, f1.following_id DESC
            LIMIT $4
            "#,
            user_id,
            last_ts,
            last_id,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        let items: Vec<FriendEdgeRow> = rows
            .into_iter()
            .take(limit as usize)
            .map(|r| FriendEdgeRow {
                user_id: r.user_id,
                followed_at: r.followed_at,
                followed_back_at: r.followed_back_at,
            })
            .collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.followed_back_at, r.user_id)))
                .transpose()?
        } else {
            None
        };

        Ok((items, next_cursor))
    }
}
