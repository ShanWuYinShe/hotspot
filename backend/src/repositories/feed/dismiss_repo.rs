use sqlx::PgPool;

use crate::{error::Result, types::enums::DismissTargetType};

#[derive(Clone)]
pub struct FeedDismissRepository {
    pool: PgPool,
}

impl FeedDismissRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn upsert(
        &self,
        user_id: i64,
        target_type: DismissTargetType,
        target_id: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO feed_dismissals (user_id, target_type, target_id)
            VALUES ($1, $2::text::dismiss_target_type, $3)
            ON CONFLICT (user_id, target_type, target_id)
            DO UPDATE SET created_at = NOW()
            "#,
            user_id,
            target_type as DismissTargetType,
            target_id,
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
