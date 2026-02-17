//! 好友备注仓库
//!
//! 用于管理用户给好友设置的备注名称

use sqlx::PgPool;

use crate::error::Result;

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

    /// 批量获取好友备注
    pub async fn get_batch(
        &self,
        user_id: i64,
        friend_ids: &[i64],
    ) -> Result<std::collections::HashMap<i64, String>> {
        if friend_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT friend_id, remark as "remark!"
            FROM user_friend_remarks
            WHERE user_id = $1 AND friend_id = ANY($2)
            "#,
            user_id,
            friend_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.friend_id, r.remark)).collect())
    }
}
