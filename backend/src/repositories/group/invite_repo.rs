use chrono::{DateTime, Duration, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::constants::INVITE_EXPIRE_DAYS;
use crate::error::Result;
use crate::types::enums::ApplicationStatus;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated, TimeIdCursor};

/// 群组邀请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupInviteRow {
    pub invite_id: Uuid,
    pub group_id: i64,
    pub inviter_id: i64,
    pub invitee_id: i64,
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: DateTime<Utc>,
    pub expires_at: Option<DateTime<Utc>>,
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
        group_id: i64,
        inviter_id: i64,
        invitee_id: i64,
        message: Option<String>,
    ) -> Result<GroupInviteRow> {
        let expires_at = Utc::now() + Duration::days(INVITE_EXPIRE_DAYS);
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            INSERT INTO group_requests (group_id, user_id, operator_id, type, message, expires_at)
            VALUES ($1, $3, $2, 'invite', $4, $5)
            RETURNING request_id as "invite_id!", group_id, operator_id as "inviter_id!", user_id as "invitee_id!", message,
                      COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, expires_at, handled_at as responded_at
            "#,
            group_id,
            inviter_id,
            invitee_id,
            message,
            expires_at
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 查找指定群组+被邀请人的 pending 状态邀请（用于重复邀请去重）
    pub async fn find_pending_by_group_and_invitee(
        &self,
        group_id: i64,
        invitee_id: i64,
    ) -> Result<Option<GroupInviteRow>> {
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT request_id as "invite_id!", group_id, operator_id as "inviter_id!", user_id as "invitee_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, expires_at, handled_at as responded_at
            FROM group_requests
            WHERE group_id = $1 AND user_id = $2 AND type = 'invite'
                  AND COALESCE(status, 'pending'::application_status) = 'pending'
                  AND (expires_at IS NULL OR expires_at > NOW())
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            group_id,
            invitee_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按 invite_id 查找邀请（对外契约使用 UUID 字符串）
    pub async fn find_by_invite_id(&self, invite_id: Uuid) -> Result<Option<GroupInviteRow>> {
        let row = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT request_id as "invite_id!", group_id, operator_id as "inviter_id!", user_id as "invitee_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, expires_at, handled_at as responded_at
            FROM group_requests
            WHERE request_id = $1 AND type = 'invite'
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
        status: ApplicationStatus,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_requests
            SET status = $2, handled_at = NOW()
            WHERE request_id = $1 AND type = 'invite'
            "#,
            invite_id,
            status as ApplicationStatus
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按用户ID查询邀请列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        status: Option<ApplicationStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<GroupInviteRow> {
        let status_str = status.map(|s| s.to_string());
        let (cursor_time, cursor_id) = if let Some(c) = cursor {
            let (time, id) = decode_cursor::<TimeIdCursor>(&c)?;
            (Some(time), Some(id))
        } else {
            (None, None)
        };

        let mut rows = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT request_id as "invite_id!", group_id, operator_id as "inviter_id!", user_id as "invitee_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, expires_at, handled_at as responded_at
            FROM group_requests
            WHERE user_id = $1
              AND type = 'invite'
              AND ($2::text IS NULL OR status::text = $2)
              AND (
                  $3::timestamptz IS NULL
                  OR created_at < $3
                  OR (created_at = $3 AND request_id < $4)
              )
            ORDER BY created_at DESC, request_id DESC
            LIMIT $5
            "#,
            user_id,
            status_str,
            cursor_time,
            cursor_id,
            (limit as i64) + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&(r.created_at, r.invite_id)))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 按群组查询发出的邀请列表
    pub async fn list_by_group(
        &self,
        group_id: i64,
        status: Option<ApplicationStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<GroupInviteRow> {
        let status_str = status.map(|s| s.to_string());
        let (cursor_time, cursor_id) = if let Some(c) = cursor {
            let (time, id) = decode_cursor::<TimeIdCursor>(&c)?;
            (Some(time), Some(id))
        } else {
            (None, None)
        };

        let mut rows = sqlx::query_as!(
            GroupInviteRow,
            r#"
            SELECT request_id as "invite_id!", group_id, operator_id as "inviter_id!", user_id as "invitee_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, expires_at, handled_at as responded_at
            FROM group_requests
            WHERE group_id = $1
              AND type = 'invite'
              AND ($2::text IS NULL OR status::text = $2)
              AND (
                  $3::timestamptz IS NULL
                  OR created_at < $3
                  OR (created_at = $3 AND request_id < $4)
              )
            ORDER BY created_at DESC, request_id DESC
            LIMIT $5
            "#,
            group_id,
            status_str,
            cursor_time,
            cursor_id,
            (limit as i64) + 1
        )
        .fetch_all(&self.pool)
        .await?;

        let has_more = rows.len() > limit as usize;
        if has_more {
            rows.truncate(limit as usize);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&(r.created_at, r.invite_id)))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 撤销 pending 状态的邀请
    pub async fn revoke_pending_invite(&self, invite_id: Uuid) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            UPDATE group_requests
            SET status = $2, handled_at = NOW()
            WHERE request_id = $1
              AND type = 'invite'
              AND COALESCE(status, 'pending'::application_status) = 'pending'
            "#,
            invite_id,
            ApplicationStatus::Rejected as ApplicationStatus
        )
        .execute(&self.pool)
        .await?;

        Ok(result.rows_affected() > 0)
    }
}
