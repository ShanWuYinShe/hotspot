use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::types::ipnetwork::IpNetwork;
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;

/// 会话Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct SessionRow {
    pub session_id: Uuid,
    pub user_id: i64,
    pub device_type: Option<String>,
    pub device_name: Option<String>,
    pub device_fingerprint: Option<String>,
    pub ip_address: Option<IpNetwork>,
    pub user_agent: Option<String>,
    // access_token_hash：存储 access_token 的哈希值（不可逆）
    pub access_token_hash: String,
    // refresh_token_hash：存储 refresh_token 的哈希值（不可逆）
    pub refresh_token_hash: String,
    pub previous_refresh_token_hash: Option<String>,
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
    #[allow(clippy::too_many_arguments)]
    pub async fn create(
        &self,
        user_id: i64,
        session_id: Uuid,
        device_type: Option<String>,
        device_name: Option<String>,
        device_fingerprint: Option<String>,
        ip_address: Option<IpNetwork>,
        user_agent: Option<String>,
        access_token_hash: String,
        refresh_token_hash: String,
        expires_at: DateTime<Utc>,
    ) -> Result<SessionRow> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            INSERT INTO sessions (session_id, user_id, device_type, device_name, device_fingerprint, ip_address, user_agent, access_token_hash, refresh_token_hash, previous_refresh_token_hash, expires_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULL, $10)
            RETURNING session_id, user_id, device_type, device_name, device_fingerprint,
                      ip_address, user_agent,
                      access_token_hash, refresh_token_hash, previous_refresh_token_hash,
                      created_at, last_active_at, expires_at, is_revoked, revoked_at
            "#,
            session_id,
            user_id,
            device_type,
            device_name,
            device_fingerprint,
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

    /// 撤销会话（仅当会话属于指定用户时）
    /// 返回 Ok(true) 表示已撤销，Ok(false) 表示会话不存在或不属于该用户
    pub async fn revoke_if_owned_by(&self, user_id: i64, session_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            UPDATE sessions
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE session_id = $1 AND user_id = $2 AND is_revoked = FALSE
            "#,
            session_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }

    /// 按用户ID查询全部活跃会话（无分页）
    pub async fn list_all_by_user(&self, user_id: i64) -> Result<Vec<SessionRow>> {
        let rows = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT session_id, user_id, device_type, device_name, device_fingerprint,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash, previous_refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE user_id = $1 AND is_revoked = FALSE AND expires_at > NOW()
            ORDER BY last_active_at DESC
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 按 refresh_token 查找会话
    pub async fn find_by_refresh_token(&self, refresh_token: &str) -> Result<Option<SessionRow>> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT session_id, user_id, device_type, device_name, device_fingerprint,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash, previous_refresh_token_hash,
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

    /// 按 previous_refresh_token 查找会话（用于检测 Token 重用攻击）
    pub async fn find_by_previous_refresh_token(
        &self,
        refresh_token: &str,
    ) -> Result<Option<SessionRow>> {
        let row = sqlx::query_as!(
            SessionRow,
            r#"
            SELECT session_id, user_id, device_type, device_name, device_fingerprint,
                   ip_address, user_agent,
                   access_token_hash, refresh_token_hash, previous_refresh_token_hash,
                   created_at, last_active_at, expires_at, is_revoked, revoked_at
            FROM sessions
            WHERE previous_refresh_token_hash = $1 AND is_revoked = FALSE AND expires_at > NOW()
            "#,
            refresh_token
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新会话的 token（刷新时使用，实现 Token 轮换）
    /// 轮换刷新 token（带“当前 refresh_token_hash 必须匹配”的乐观锁，避免并发刷新把 token 覆盖掉）
    ///
    /// 返回：
    /// - true：本次轮换成功
    /// - false：refresh_token_hash 已被其他并发请求轮换（当前 token 已过时）
    pub async fn rotate_tokens_if_refresh_matches(
        &self,
        session_id: Uuid,
        current_refresh_token_hash: &str,
        access_token_hash: String,
        new_refresh_token_hash: String,
        expires_at: DateTime<Utc>,
    ) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            UPDATE sessions
            SET access_token_hash = $3,
                previous_refresh_token_hash = refresh_token_hash,
                refresh_token_hash = $4,
                expires_at = $5,
                last_active_at = NOW()
            WHERE session_id = $1
              AND is_revoked = FALSE
              AND refresh_token_hash = $2
            "#,
            session_id,
            current_refresh_token_hash,
            access_token_hash,
            new_refresh_token_hash,
            expires_at
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() == 1)
    }

    /// 撤销用户的所有会话
    pub async fn revoke_all_by_user(&self, user_id: i64) -> Result<Vec<Uuid>> {
        let rows = sqlx::query!(
            r#"
            UPDATE sessions
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE user_id = $1 AND is_revoked = FALSE
            RETURNING session_id
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.session_id).collect())
    }

    /// 撤销用户除指定会话外的所有会话
    pub async fn revoke_all_except(
        &self,
        user_id: i64,
        except_session_id: Uuid,
    ) -> Result<Vec<Uuid>> {
        let rows = sqlx::query!(
            r#"
            UPDATE sessions
            SET is_revoked = TRUE, revoked_at = NOW()
            WHERE user_id = $1 AND session_id != $2 AND is_revoked = FALSE
            RETURNING session_id
            "#,
            user_id,
            except_session_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.session_id).collect())
    }

    // NOTE: 初始版本不提供按 access_token_hash 查 session 的查询入口，避免遗留未用 API。
}
