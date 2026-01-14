use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::constants::INVITE_EXPIRE_DAYS;
use crate::error::Result;
use crate::types::enums::InviteStatus;
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 群组邀请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupInviteRow {
    pub id: i64,
    pub invite_id: Uuid,
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
            RETURNING id, invite_id, group_chat_id, inviter_id, invitee_id, message,
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

    /// 按 invite_id 查找邀请（对外契约使用 UUID 字符串）
    pub async fn find_by_invite_id(&self, invite_id: Uuid) -> Result<Option<GroupInviteRow>> {
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT id, invite_id, group_chat_id, inviter_id, invitee_id, message,
                   status as "status: InviteStatus", created_at, expires_at, responded_at
            FROM group_invites
            WHERE invite_id = $1
            "#,
            invite_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按 invite_id 更新邀请状态（对外契约使用 UUID 字符串）
    pub async fn update_status_by_invite_id(
        &self,
        invite_id: Uuid,
        status: InviteStatus,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_invites
            SET status = $2, responded_at = NOW()
            WHERE invite_id = $1
            "#,
            invite_id,
            status as InviteStatus
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按用户ID查询邀请列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupInviteRow>, Option<String>)> {
        let status_str = status.map(|s| s.to_string());
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT id, invite_id, group_chat_id, inviter_id, invitee_id, message,
                   status as "status: InviteStatus", created_at, expires_at, responded_at
            FROM group_invites
            WHERE invitee_id = $1 
              AND ($2::text IS NULL OR status::text = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            user_id,
            status_str,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|last| encode_cursor(&last.id))
            .transpose()?;

        Ok((rows, next_cursor))
    }
}
