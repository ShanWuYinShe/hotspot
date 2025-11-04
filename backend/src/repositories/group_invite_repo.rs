use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group_invite::GroupInvite,
    types::enums::InviteStatus,
};

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
        group_id: Uuid,
        inviter_id: i64,
        invitee_id: i64,
        message: Option<String>,
    ) -> Result<Uuid> {
        let invite = sqlx::query!(
            r#"
            INSERT INTO room_invites (room_id, inviter_id, invitee_id, message, status)
            VALUES ($1, $2, $3, $4, 'pending')
            RETURNING invite_id
            "#,
            group_id,
            inviter_id,
            invitee_id,
            message
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(invite.invite_id)
    }

    /// 根据ID查找邀请
    pub async fn find_by_id(&self, invite_id: Uuid) -> Result<Option<GroupInvite>> {
        let invite = sqlx::query_as!(
            GroupInvite,
            r#"
            SELECT invite_id, room_id as group_id, inviter_id, invitee_id, message, status as "status: InviteStatus", created_at, responded_at
            FROM room_invites
            WHERE invite_id = $1
            "#,
            invite_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(invite)
    }

    /// 查询用户收到的邀请列表（使用双ID cursor分页）
    pub async fn list_by_invitee(
        &self,
        invitee_id: i64,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupInvite>, Option<String>)> {
        // 解析cursor（业务ID）并查询对应的自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let business_id = Uuid::parse_str(&cursor_str).ok();
            
            let result = sqlx::query!(
                "SELECT id FROM room_invites WHERE invite_id = $1",
                business_id
            )
            .fetch_optional(&self.pool)
            .await?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        let invites = if let Some(status) = status {
            sqlx::query_as!(
                GroupInvite,
                r#"
                SELECT invite_id, room_id as group_id, inviter_id, invitee_id, message, status as "status: InviteStatus", created_at, responded_at
                FROM room_invites
                WHERE invitee_id = $1 
                  AND (status = $2::invite_status OR $2::invite_status = 'any'::invite_status)
                  AND ($3::BIGINT IS NULL OR id > $3::BIGINT)
                ORDER BY id DESC
                LIMIT $4
                "#,
                invitee_id,
                status as InviteStatus,
                cursor_id,
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                GroupInvite,
                r#"
                SELECT invite_id, room_id as group_id, inviter_id, invitee_id, message, status as "status: InviteStatus", created_at, responded_at
                FROM room_invites
                WHERE invitee_id = $1
                  AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
                ORDER BY id DESC
                LIMIT $3
                "#,
                invitee_id,
                cursor_id,
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let next_cursor = if invites.len() == limit as usize {
            invites.last().map(|i| i.invite_id.to_string())
        } else {
            None
        };

        Ok((invites, next_cursor))
    }

    /// 更新邀请状态
    pub async fn update_status(&self, invite_id: Uuid, status: InviteStatus) -> Result<()> {
        let result = sqlx::query!(
            r#"
            UPDATE room_invites
            SET status = $2::invite_status, responded_at = NOW()
            WHERE invite_id = $1
            "#,
            invite_id,
            status as InviteStatus
        )
        .execute(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        if result.rows_affected() == 0 {
            return Err(AppError::NotFound("资源不存在".to_string()));
        }

        Ok(())
    }

    /// 检查是否已有待处理的邀请
    pub async fn has_pending_invite(&self, group_id: Uuid, invitee_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM room_invites
                WHERE room_id = $1 AND invitee_id = $2 AND status = 'pending'
            ) as "exists!"
            "#,
            group_id,
            invitee_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(result.exists)
    }
}

