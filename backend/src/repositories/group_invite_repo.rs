use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::constants::INVITE_EXPIRE_DAYS;
use crate::error::Result;
use crate::types::enums::InviteStatus;

/// 群组邀请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupInviteRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub group_chat_id: Uuid,
    pub inviter_id: i64,
    pub invitee_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub responded_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct GroupInviteRepository {
    pool: PgPool,
}

impl GroupInviteRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建邀请
    pub async fn create(
        &self,
        group_chat_id: Uuid,
        inviter_id: i64,
        invitee_id: i64,
        message: Option<String>,
    ) -> Result<GroupInviteRow> {
        let expires_at = Utc::now() + Duration::days(INVITE_EXPIRE_DAYS);
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            INSERT INTO group_invites (group_chat_id, inviter_id, invitee_id, message, expires_at)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id, cursor_id, group_chat_id, inviter_id, invitee_id, message,
                      status as "status: InviteStatus", created_at, expires_at, responded_at
            "#,
            group_chat_id,
            inviter_id,
            invitee_id,
            message,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按ID查找邀请
    pub async fn find_by_id(&self, id: i64) -> Result<Option<GroupInviteRow>> {
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT id, cursor_id, group_chat_id, inviter_id, invitee_id, message,
                   status as "status: InviteStatus", created_at, expires_at, responded_at
            FROM group_invites
            WHERE id = $1
            "#,
            id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新邀请状态
    pub async fn update_status(&self, id: i64, status: InviteStatus) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_invites
            SET status = $2, responded_at = NOW()
            WHERE id = $1
            "#,
            id,
            status as InviteStatus
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
                    r#"SELECT id FROM group_invites WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按用户ID查询邀请列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<GroupInviteRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT id, cursor_id, group_chat_id, inviter_id, invitee_id, message,
                   status as "status: InviteStatus", created_at, expires_at, responded_at
            FROM group_invites
            WHERE invitee_id = $1 
              AND ($2::invitation_status IS NULL OR status = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            user_id,
            status as Option<InviteStatus>,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }
}
