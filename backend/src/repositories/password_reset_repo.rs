use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::error::Result;

#[derive(Debug, Clone)]
pub struct PasswordResetRow {
    pub reset_id: Uuid,
    pub user_id: i64,
    pub token: String,
    pub expires_at: DateTime<Utc>,
    pub used: Option<bool>,
    pub created_at: DateTime<Utc>,
}

#[derive(Clone)]
pub struct PasswordResetRepository {
    pool: PgPool,
}

impl PasswordResetRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建重置记录
    pub async fn create(
        &self,
        user_id: i64,
        token: String,
        expires_at: DateTime<Utc>,
    ) -> Result<Uuid> {
        let row = sqlx::query!(
            r#"
            INSERT INTO password_resets (user_id, token, expires_at)
            VALUES ($1, $2, $3)
            RETURNING reset_id
            "#,
            user_id,
            token,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row.reset_id)
    }

    /// 查找有效记录
    pub async fn find_valid(&self, token: &str) -> Result<Option<PasswordResetRow>> {
        let row = sqlx::query_as!(
            PasswordResetRow,
            r#"
            SELECT reset_id, user_id, token, expires_at, used, created_at
            FROM password_resets
            WHERE token = $1 AND used = FALSE AND expires_at > NOW()
            "#,
            token
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按 reset_id 查找记录
    pub async fn find_by_id(&self, reset_id: Uuid) -> Result<Option<PasswordResetRow>> {
        let row = sqlx::query_as!(
            PasswordResetRow,
            r#"
            SELECT reset_id, user_id, token, expires_at, used, created_at
            FROM password_resets
            WHERE reset_id = $1
            "#,
            reset_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新重置 token
    pub async fn update_token(&self, reset_id: Uuid, token: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE password_resets
            SET token = $2
            WHERE reset_id = $1 AND used = FALSE
            "#,
            reset_id,
            token
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 标记已使用
    pub async fn mark_used(&self, reset_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE password_resets SET used = TRUE WHERE reset_id = $1",
            reset_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
