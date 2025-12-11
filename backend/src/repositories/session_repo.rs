use sqlx::{PgPool, FromRow};
use sqlx::types::ipnetwork::IpNetwork;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 会话Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct SessionRow {
    pub id: i64,
    pub session_id: Uuid,
    pub user_id: i64,
    pub device_type: Option<String>,
    pub device_name: Option<String>,
    pub ip_address: Option<IpNetwork>,
    pub user_agent: Option<String>,
    pub access_token_hash: String,
    pub refresh_token_hash: String,
    pub created_at: DateTime<Utc>,
    pub last_active_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub is_revoked: bool,
    pub revoked_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct SessionRepository {
    pool: PgPool,
}

impl SessionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建会话
    pub async fn create(
        &self,
        user_id: i64,
        device_type: Option<String>,
        device_name: Option<String>,
        ip_address: Option<IpNetwork>,
        user_agent: Option<String>,
        access_token_hash: String,
        refresh_token_hash: String,
        expires_at: DateTime<Utc>,
    ) -> Result<SessionRow> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            INSERT INTO sessions (user_id, device_type, device_name, ip_address, user_agent, access_token_hash, refresh_token_hash, expires_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING id, session_id, user_id, device_type, device_name, 
                      ip_address, user_agent,
                      access_token_hash, refresh_token_hash,
                      created_at, last_active_at, expires_at, is_revoked, revoked_at
            "#,
            user_id,
            device_type,
            device_name,
            ip_address as Option<IpNetwork>,
            user_agent,
            access_token_hash,
            refresh_token_hash,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找会话
    pub async fn find_by_id(&self, session_id: Uuid) -> Result<Option<SessionRow>> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE session_id = $1 AND is_revoked = FALSE
            "#,
            session_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 撤销会话
    pub async fn revoke(&self, session_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE sessions
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE session_id = $1
            "#,
            session_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM sessions WHERE session_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按用户ID查询会话列表
    pub async fn list_by_user(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<SessionRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let sessions = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE user_id = $1 AND id < $2 AND is_revoked = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(sessions)
    }

    /// 删除会话（撤销）
    pub async fn delete(&self, session_id: Uuid) -> Result<()> {
        self.revoke(session_id).await
    }

    /// 按 refresh_token 查找会话
    pub async fn find_by_refresh_token(&self, refresh_token: &str) -> Result<Option<SessionRow>> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE refresh_token_hash = $1 AND is_revoked = FALSE AND expires_at > NOW()
            "#,
            refresh_token
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新会话的 token（刷新时使用）
    pub async fn update_tokens(
        &self,
        session_id: Uuid,
        access_token: String,
        refresh_token: String,
        expires_at: DateTime<Utc>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE sessions
            SET access_token_hash = $2,
                refresh_token_hash = $3,
                expires_at = $4,
                last_active_at = NOW()
            WHERE session_id = $1 AND is_revoked = FALSE
            "#,
            session_id,
            access_token,
            refresh_token,
            expires_at
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 撤销用户的所有会话
    pub async fn revoke_all_by_user(&self, user_id: i64) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            UPDATE sessions
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE user_id = $1 AND is_revoked = FALSE
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }

    /// 按 access_token 查找会话
    pub async fn find_by_access_token(&self, access_token: &str) -> Result<Option<SessionRow>> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE access_token_hash = $1 AND is_revoked = FALSE
            "#,
            access_token
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }
}
