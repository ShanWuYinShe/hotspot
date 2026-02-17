use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::ApplicationStatus;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated, TimeIdCursor};

/// 群组申请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupApplicationRow {
    pub application_id: Uuid,
    pub group_id: i64,
    pub applicant_id: i64,
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: DateTime<Utc>,
    pub handled_by: Option<i64>,
    pub handled_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct GroupApplicationRepository {
    pool: PgPool,
}

impl GroupApplicationRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建入群申请（approval 群组用户申请加入时调用）
    pub async fn create(
        &self,
        group_id: i64,
        applicant_id: i64,
        message: Option<String>,
    ) -> Result<GroupApplicationRow> {
        let row = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            INSERT INTO group_requests (group_id, user_id, operator_id, type, message)
            VALUES ($1, $2, $2, 'apply', $3)
            RETURNING request_id as "application_id!", group_id, user_id as "applicant_id!", message,
                      COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, handled_by, handled_at
            "#,
            group_id,
            applicant_id,
            message
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按 application_id 查找申请（对外契约使用 UUID 字符串）
    pub async fn find_by_application_id(
        &self,
        application_id: Uuid,
    ) -> Result<Option<GroupApplicationRow>> {
        let row = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT request_id as "application_id!", group_id, user_id as "applicant_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, handled_by, handled_at
            FROM group_requests
            WHERE request_id = $1 AND type = 'apply'
            "#,
            application_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按 application_id 更新申请状态（对外契约使用 UUID 字符串）
    pub async fn update_status_by_application_id(
        &self,
        application_id: Uuid,
        status: ApplicationStatus,
        handled_by: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_requests
            SET status = $2, handled_by = $3, handled_at = NOW()
            WHERE request_id = $1 AND type = 'apply'
            "#,
            application_id,
            status as ApplicationStatus,
            handled_by
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按群组ID查询申请列表
    pub async fn list_by_group(
        &self,
        group_id: i64,
        status: Option<ApplicationStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<GroupApplicationRow> {
        let status_str = status.map(|s| s.to_string());
        let (cursor_time, cursor_id) = if let Some(c) = cursor {
            let (time, id) = decode_cursor::<TimeIdCursor>(&c)?;
            (Some(time), Some(id))
        } else {
            (None, None)
        };

        let mut rows = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT request_id as "application_id!", group_id, user_id as "applicant_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, handled_by, handled_at
            FROM group_requests
            WHERE group_id = $1
              AND type = 'apply'
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
                .map(|r| encode_cursor(&(r.created_at, r.application_id)))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 按申请人 user_id 查询申请列表（我的申请列表）
    pub async fn list_by_applicant(
        &self,
        applicant_id: i64,
        status: Option<ApplicationStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Paginated<GroupApplicationRow> {
        let status_str = status.map(|s| s.to_string());
        let (cursor_time, cursor_id) = if let Some(c) = cursor {
            let (time, id) = decode_cursor::<TimeIdCursor>(&c)?;
            (Some(time), Some(id))
        } else {
            (None, None)
        };

        let mut rows = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT request_id as "application_id!", group_id, user_id as "applicant_id!", message,
                   COALESCE(status, 'pending'::application_status) as "status!: ApplicationStatus", created_at, handled_by, handled_at
            FROM group_requests
            WHERE user_id = $1
              AND type = 'apply'
              AND ($2::text IS NULL OR status::text = $2)
              AND (
                  $3::timestamptz IS NULL
                  OR created_at < $3
                  OR (created_at = $3 AND request_id < $4)
              )
            ORDER BY created_at DESC, request_id DESC
            LIMIT $5
            "#,
            applicant_id,
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
                .map(|r| encode_cursor(&(r.created_at, r.application_id)))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 按 application_id 删除申请（取消申请时使用；对外契约使用 UUID 字符串）
    pub async fn delete_by_application_id(&self, application_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM group_requests WHERE request_id = $1 AND type = 'apply'",
            application_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
