use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group_application::GroupApplication,
    types::enums::InviteStatus,
};

pub struct GroupApplicationRepository {
    pool: PgPool,
}

impl GroupApplicationRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建申请
    pub async fn create(
        &self,
        group_id: Uuid,
        applicant_id: i64,
        message: Option<String>,
    ) -> Result<Uuid> {
        let application = sqlx::query!(
            r#"
            INSERT INTO room_applications (room_id, applicant_id, message, status)
            VALUES ($1, $2, $3, 'pending')
            RETURNING application_id
            "#,
            group_id,
            applicant_id,
            message
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(application.application_id)
    }

    /// 根据ID查找申请
    pub async fn find_by_id(&self, application_id: Uuid) -> Result<Option<GroupApplication>> {
        let application = sqlx::query_as!(
            GroupApplication,
            r#"
            SELECT id, application_id, room_id as group_id, applicant_id, message, status as "status: InviteStatus", 
                   created_at, reviewed_at, reviewer_id, review_reason
            FROM room_applications
            WHERE application_id = $1
            "#,
            application_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(application)
    }

    /// 查询群组的申请列表（管理员查看）
    pub async fn list_by_group(
        &self,
        group_id: Uuid,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupApplication>, Option<String>)> {
        let cursor_time = cursor.and_then(|c| chrono::DateTime::parse_from_rfc3339(&c).ok());

        let applications = if let Some(status) = status {
            sqlx::query_as!(
                GroupApplication,
                r#"
                SELECT id, application_id, room_id as group_id, applicant_id, message, status as "status: InviteStatus",
                       created_at, reviewed_at, reviewer_id, review_reason
                FROM room_applications
                WHERE room_id = $1 
                  AND (status = $2::invite_status OR $2::invite_status = 'any'::invite_status)
                  AND ($3::TIMESTAMPTZ IS NULL OR created_at < $3::TIMESTAMPTZ)
                ORDER BY created_at DESC
                LIMIT $4
                "#,
                group_id,
                status as InviteStatus,
                cursor_time.map(|t| t.with_timezone(&chrono::Utc)),
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                GroupApplication,
                r#"
                SELECT id, application_id, room_id as group_id, applicant_id, message, status as "status: InviteStatus",
                       created_at, reviewed_at, reviewer_id, review_reason
                FROM room_applications
                WHERE room_id = $1
                  AND ($2::TIMESTAMPTZ IS NULL OR created_at < $2::TIMESTAMPTZ)
                ORDER BY created_at DESC
                LIMIT $3
                "#,
                group_id,
                cursor_time.map(|t| t.with_timezone(&chrono::Utc)),
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let next_cursor = if applications.len() == limit as usize {
            applications.last().map(|a| a.created_at.to_rfc3339())
        } else {
            None
        };

        Ok((applications, next_cursor))
    }

    /// 审核申请
    pub async fn review(
        &self,
        application_id: Uuid,
        reviewer_id: i64,
        status: InviteStatus,
        reason: Option<String>,
    ) -> Result<()> {
        let result = sqlx::query!(
            r#"
            UPDATE room_applications
            SET status = $2::invite_status, reviewed_at = NOW(), reviewer_id = $3, review_reason = $4
            WHERE application_id = $1
            "#,
            application_id,
            status as InviteStatus,
            reviewer_id,
            reason
        )
        .execute(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        if result.rows_affected() == 0 {
            return Err(AppError::NotFound("资源不存在".to_string()));
        }

        Ok(())
    }

    /// 检查是否已有待处理的申请
    pub async fn has_pending_application(&self, group_id: Uuid, applicant_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM room_applications
                WHERE room_id = $1 AND applicant_id = $2 AND status = 'pending'
            ) as "exists!"
            "#,
            group_id,
            applicant_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(result.exists)
    }
}

