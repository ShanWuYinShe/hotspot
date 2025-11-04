use sqlx::PgPool;
use uuid::Uuid;

use crate::{error::Result, models::session::Session};

pub struct SessionRepository {
    pool: PgPool,
}

impl SessionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建会话
    pub async fn create(&self, session: &Session) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO sessions (
                session_id, user_id, device_type, device_name,
                ip_address, location, created_at, last_active_at, expires_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            "#,
            session.session_id,
            session.user_id,
            session.device_type,
            session.device_name,
            session.ip_address,
            session.location,
            session.created_at,
            session.last_active_at,
            session.expires_at
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 根据session_id查询会话
    pub async fn find_by_id(&self, session_id: Uuid) -> Result<Option<Session>> {
        let session = sqlx::query_as!(
            Session,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, location, created_at, last_active_at, expires_at
            FROM sessions
            WHERE session_id = $1 AND expires_at > NOW()
            "#,
            session_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(session)
    }

    /// 获取用户的会话列表（游标分页）
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<Session>, Option<String>)> {
        // 1. 根据业务ID查询自增主键id (如果提供了cursor)
        let cursor_id = if let Some(session_id_str) = &cursor {
            let session_id = Uuid::parse_str(session_id_str).ok();
            if let Some(sid) = session_id {
                sqlx::query!("SELECT id FROM sessions WHERE session_id = $1", sid)
                    .fetch_optional(&self.pool)
                    .await?
                    .map(|r| r.id)
            } else {
                None
            }
        } else {
            None
        };

        // 2. 使用自增主键id进行分页
        let sessions = sqlx::query_as!(
            Session,
            r#"
            SELECT id, session_id, user_id, device_type, device_name,
                   ip_address, location, created_at, last_active_at, expires_at
            FROM sessions
            WHERE user_id = $1
            AND ($2::BIGINT IS NULL OR id > $2)
            AND expires_at > NOW()
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        // 3. 返回业务ID作为游标
        let next_cursor = if sessions.len() == limit as usize {
            sessions.last().map(|s| s.session_id.to_string())
        } else {
            None
        };

        Ok((sessions, next_cursor))
    }

    /// 删除会话
    pub async fn delete(&self, session_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM sessions WHERE session_id = $1
            "#,
            session_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除用户的所有会话
    pub async fn delete_by_user(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM sessions WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新最后活跃时间
    pub async fn update_last_active(&self, session_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE sessions
            SET last_active_at = NOW()
            WHERE session_id = $1
            "#,
            session_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除过期会话（定期清理）
    pub async fn delete_expired(&self) -> Result<u64> {
        let result = sqlx::query!(
            r#"
            DELETE FROM sessions WHERE expires_at < NOW()
            "#
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected())
    }
}

