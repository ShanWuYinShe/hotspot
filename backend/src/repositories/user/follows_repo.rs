use std::collections::{HashMap, HashSet};

use crate::error::Result;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated, TimeCursor};
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

    /// 批量检查 follower_id 是否关注了 target_ids 中的用户
    /// 返回 follower_id 正在关注的 target_ids 子集
    pub async fn batch_check_following(
        &self,
        follower_id: i64,
        target_ids: &[i64],
    ) -> Result<HashSet<i64>> {
        if target_ids.is_empty() {
            return Ok(HashSet::new());
        }
        let rows = sqlx::query_scalar!(
            "SELECT following_id as \"id!\" FROM follows WHERE follower_id = $1 AND following_id = ANY($2)",
            follower_id,
            target_ids
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.into_iter().collect())
    }

    /// 批量检查 following_id 被 follower_ids 中哪些用户关注
    /// 返回正在关注 following_id 的 follower_ids 子集
    pub async fn batch_check_followed_by(
        &self,
        following_id: i64,
        follower_ids: &[i64],
    ) -> Result<HashSet<i64>> {
        if follower_ids.is_empty() {
            return Ok(HashSet::new());
        }
        let rows = sqlx::query_scalar!(
            "SELECT follower_id as \"id!\" FROM follows WHERE following_id = $1 AND follower_id = ANY($2)",
            following_id,
            follower_ids
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.into_iter().collect())
    }

    pub async fn list_following(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Paginated<FollowEdgeRow> {
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
    ) -> Paginated<FollowEdgeRow> {
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
    ) -> Paginated<FriendEdgeRow> {
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

    /// 批量计算共同好友数量
    /// 返回 HashMap<target_id, mutual_count>，未出现在结果中的 target 表示共同好友为 0
    pub async fn count_mutual_friends_batch(
        &self,
        viewer_id: i64,
        target_ids: &[i64],
    ) -> Result<HashMap<i64, i32>> {
        if target_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            WITH viewer_friends AS (
                SELECT f1.following_id as friend_id
                FROM follows f1
                INNER JOIN follows f2
                  ON f1.following_id = f2.follower_id
                 AND f2.following_id = f1.follower_id
                WHERE f1.follower_id = $1
            ),
            target_friends AS (
                SELECT f1.follower_id as target_id, f1.following_id as friend_id
                FROM follows f1
                INNER JOIN follows f2
                  ON f1.following_id = f2.follower_id
                 AND f2.following_id = f1.follower_id
                WHERE f1.follower_id = ANY($2)
            )
            SELECT tf.target_id as "target_id!", COUNT(DISTINCT vf.friend_id)::int as "mutual_count!"
            FROM viewer_friends vf
            INNER JOIN target_friends tf ON vf.friend_id = tf.friend_id
            GROUP BY tf.target_id
            "#,
            viewer_id,
            target_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.target_id, r.mutual_count))
            .collect())
    }

    /// 批量计算用户好友中在各群组内的人数
    /// 返回 HashMap<group_id, friend_count>
    pub async fn count_friends_in_groups_batch(
        &self,
        viewer_id: i64,
        group_ids: &[i64],
    ) -> Result<HashMap<i64, i32>> {
        if group_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            WITH viewer_friends AS (
                SELECT f1.following_id as friend_id
                FROM follows f1
                INNER JOIN follows f2
                  ON f1.following_id = f2.follower_id
                 AND f2.following_id = f1.follower_id
                WHERE f1.follower_id = $1
            )
            SELECT gm.group_id as "group_id!", COUNT(DISTINCT vf.friend_id)::int as "friend_count!"
            FROM viewer_friends vf
            INNER JOIN group_members gm ON vf.friend_id = gm.user_id
            WHERE gm.group_id = ANY($2)
            GROUP BY gm.group_id
            "#,
            viewer_id,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows
            .into_iter()
            .map(|r| (r.group_id, r.friend_count))
            .collect())
    }
}
