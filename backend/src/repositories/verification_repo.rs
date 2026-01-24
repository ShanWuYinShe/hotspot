use anyhow::Result;
use chrono::{DateTime, Utc};
use sqlx::PgPool;

#[derive(Debug, Clone)]
pub struct VerificationCode {
    #[allow(dead_code)]
    pub id: i64,
    pub code_id: String,
    pub phone: String,
    pub code: String,
    pub r#type: String, // Keep r#type for Rust side
    pub expires_at: DateTime<Utc>,
    pub attempts: i32,
    pub max_attempts: i32,
    #[allow(dead_code)]
    pub verified_at: Option<DateTime<Utc>>,
    #[allow(dead_code)]
    pub created_at: DateTime<Utc>,
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
        phone: &str,
        code: &str,
        code_type: &str,
        expires_at: DateTime<Utc>,
    ) -> Result<VerificationCode> {
        let row = sqlx::query_as!(
            VerificationCode,
            r#"
            INSERT INTO verification_codes (phone, code, type, expires_at)
            VALUES ($1, $2, $3, $4)
            RETURNING id, code_id, phone, code, type::text as "type!", expires_at, attempts, max_attempts, verified_at, created_at
            "#,
            phone,
            code,
            code_type as _,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 根据 code_id 查找验证码
    pub async fn find_by_code_id(&self, code_id: &str) -> Result<Option<VerificationCode>> {
        let row = sqlx::query_as!(
            VerificationCode,
            r#"
            SELECT id, code_id, phone, code, type::text as "type!", expires_at, attempts, max_attempts, verified_at, created_at
            FROM verification_codes
            WHERE code_id = $1
            "#,
            code_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 标记为已验证
    pub async fn mark_verified(&self, code_id: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE verification_codes
            SET verified_at = NOW()
            WHERE code_id = $1
            "#,
            code_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 增加尝试次数
    pub async fn increment_attempts(&self, code_id: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE verification_codes
            SET attempts = attempts + 1
            WHERE code_id = $1
            "#,
            code_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 检查最近是否发送过验证码（用于限流）
    pub async fn check_recent_code(
        &self,
        phone: &str,
        code_type: &str,
        seconds: i64,
    ) -> Result<bool> {
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1
                FROM verification_codes
                WHERE phone = $1 AND type::text = $2
                  AND created_at > NOW() - INTERVAL '1 second' * $3
            ) as "exists!"
            "#,
            phone,
            code_type,
            seconds as f64
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(exists)
    }

    /// 获取当前手机号今天发送的验证码数量（用于每日限流）
    pub async fn count_today_codes(&self, phone: &str, code_type: &str) -> Result<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM verification_codes
            WHERE phone = $1 AND type::text = $2
              AND created_at >= CURRENT_DATE
            "#,
            phone,
            code_type
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(count)
    }
}
