use sqlx::PgPool;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct FollowRequestRepository {
    pool: PgPool,
}

impl FollowRequestRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 检查请求是否存在
    pub async fn exists(&self, requester_id: i64, target_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM follow_requests
                WHERE requester_id = $1 AND target_id = $2 AND status = 'pending'
            ) as "exists!"
            "#,
            requester_id,
            target_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }
}
