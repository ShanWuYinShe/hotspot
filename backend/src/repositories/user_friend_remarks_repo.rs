use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserFriendRemarkRow {
    pub id: i64,
    pub user_id: i64,
    pub friend_id: i64,
    pub remark: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct UserFriendRemarksRepository {
    pool: PgPool,
}

impl UserFriendRemarksRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn upsert(&self, user_id: i64, friend_id: i64, remark: String) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_friend_remarks (user_id, friend_id, remark)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id, friend_id)
            DO UPDATE SET remark = $3, updated_at = NOW()
            "#,
            user_id,
            friend_id,
            remark
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn delete(&self, user_id: i64, friend_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM user_friend_remarks
            WHERE user_id = $1 AND friend_id = $2
            "#,
            user_id,
            friend_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    pub async fn get(&self, user_id: i64, friend_id: i64) -> Result<Option<String>> {
        let result = sqlx::query!(
            r#"
            SELECT remark
            FROM user_friend_remarks
            WHERE user_id = $1 AND friend_id = $2
            "#,
            user_id,
            friend_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(result.and_then(|r| r.remark))
    }

    /// 批量获取好友备注
    pub async fn get_batch(&self, user_id: i64, friend_ids: &[i64]) -> Result<std::collections::HashMap<i64, String>> {
        if friend_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT friend_id, remark
            FROM user_friend_remarks
            WHERE user_id = $1 AND friend_id = ANY($2)
            "#,
            user_id,
            friend_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter()
            .filter_map(|r| r.remark.map(|remark| (r.friend_id, remark)))
            .collect())
    }
}

