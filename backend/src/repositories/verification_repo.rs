use anyhow::Result;
use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct VerificationCode {
    pub code_id: Uuid,
}

#[derive(Clone)]
pub struct VerificationRepository {
    pool: PgPool,
}

impl VerificationRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建验证码
    pub async fn create_code(
        &self,
        target: &str,
        code: &str,
        code_type: &str,
        expires_at: DateTime<Utc>,
    ) -> Result<VerificationCode> {
        let row = sqlx::query_as!(
            VerificationCode,
            r#"
            INSERT INTO verification_codes (target, code, type, expires_at)
            VALUES ($1, $2, $3, $4)
            RETURNING code_id
            "#,
            target,
            code,
            code_type,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 查找验证码
    pub async fn find_valid(
        &self,
        target: &str,
        code: &str,
        code_type: &str,
    ) -> Result<Option<VerificationCode>> {
        let row = sqlx::query_as!(
            VerificationCode,
            r#"
            SELECT code_id
            FROM verification_codes
            WHERE target = $1 AND code = $2 AND type = $3 AND used = FALSE AND expires_at > NOW()
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            target,
            code,
            code_type
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 标记为已使用
    pub async fn mark_used(&self, code_id: Uuid) -> Result<()> {
        sqlx::query!(
            "UPDATE verification_codes SET used = TRUE WHERE code_id = $1",
            code_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
