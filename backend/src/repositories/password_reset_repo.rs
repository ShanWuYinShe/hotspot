use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct PasswordResetRepository {
    pool: PgPool,
}

#[derive(Debug, Clone)]
pub struct PasswordResetTokenRow {
    pub id: i64,
    pub user_id: i64,
    pub code_hash: String,
}

impl PasswordResetRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    pub async fn create(
        &self,
        user_id: i64,
        code: String,
        code_hash: String,
        expires_at: DateTime<Utc>,
    ) -> Result<Uuid> {
        let reset_id = Uuid::new_v4();

        sqlx::query!(
            r#"
            INSERT INTO password_reset_tokens (reset_id, user_id, code, code_hash, expires_at)
            VALUES ($1, $2, $3, $4, $5)
            "#,
            reset_id,
            user_id,
            code,
            code_hash,
            expires_at
        )
        .execute(&self.pool)
        .await
        ?;

        Ok(reset_id)
    }

    pub async fn find_valid_by_reset_id(
        &self,
        reset_id: Uuid,
    ) -> Result<Option<PasswordResetTokenRow>> {
        let row = sqlx::query_as!(
            PasswordResetTokenRow,
            r#"
            SELECT
              id,
              user_id,
              code_hash
            FROM password_reset_tokens
            WHERE reset_id = $1
              AND is_used = FALSE
              AND is_expired = FALSE
              AND expires_at > NOW()
            "#,
            reset_id
        )
        .fetch_optional(&self.pool)
        .await
        ?;

        Ok(row)
    }

    pub async fn set_verified(
        &self,
        id: i64,
        reset_token_hash: String,
        verified_at: DateTime<Utc>,
        expires_at: DateTime<Utc>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE password_reset_tokens
            SET reset_token_hash = $2,
                verified_at = $3,
                expires_at = $4
            WHERE id = $1
            "#,
            id,
            reset_token_hash,
            verified_at,
            expires_at
        )
        .execute(&self.pool)
        .await
        ?;

        Ok(())
    }

    pub async fn find_valid_by_reset_token_hash(
        &self,
        reset_token_hash: &str,
    ) -> Result<Option<PasswordResetTokenRow>> {
        let row = sqlx::query_as!(
            PasswordResetTokenRow,
            r#"
            SELECT
              id,
              user_id,
              code_hash
            FROM password_reset_tokens
            WHERE reset_token_hash = $1
              AND is_used = FALSE
              AND is_expired = FALSE
              AND expires_at > NOW()
              AND verified_at IS NOT NULL
            "#,
            reset_token_hash
        )
        .fetch_optional(&self.pool)
        .await
        ?;

        Ok(row)
    }

    pub async fn mark_used(&self, id: i64, used_at: DateTime<Utc>) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE password_reset_tokens
            SET is_used = TRUE,
                used_at = $2
            WHERE id = $1
            "#,
            id,
            used_at
        )
        .execute(&self.pool)
        .await
        ?;

        Ok(())
    }
}
