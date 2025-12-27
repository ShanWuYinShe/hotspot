use crate::error::Result;
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct FeedRepository {
    pool: PgPool,
}

impl FeedRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 记录反馈
    pub async fn record_feedback(
        &self,
        user_id: i64,
        feed_item_id: Uuid,
        action: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO feed_feedback (user_id, feed_item_id, action)
            VALUES ($1, $2, $3)
            "#,
            user_id,
            feed_item_id,
            action
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
