use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::InviteStatus;
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 群组申请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupApplicationRow {
    pub id: i64,
    pub application_id: Uuid,
    pub group_chat_id: Uuid,
    pub applicant_id: i64,
    pub message: Option<String>,
    pub status: InviteStatus,
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

    /// 创建申请
    pub async fn create(
        &self,
        group_chat_id: Uuid,
        applicant_id: i64,
        message: Option<String>,
    ) -> Result<GroupApplicationRow> {
        let row = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            INSERT INTO group_applications (group_chat_id, applicant_id, message)
            VALUES ($1, $2, $3)
            RETURNING id, application_id, group_chat_id, applicant_id, message,
                      status as "status: InviteStatus", created_at, handled_by, handled_at
            "#,
            group_chat_id,
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
            SELECT id, application_id, group_chat_id, applicant_id, message,
                   status as "status: InviteStatus", created_at, handled_by, handled_at
            FROM group_applications
            WHERE application_id = $1
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
        status: InviteStatus,
        handled_by: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_applications
            SET status = $2, handled_by = $3, handled_at = NOW()
            WHERE application_id = $1
            "#,
            application_id,
            status as InviteStatus,
            handled_by
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 按群组ID查询申请列表
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupApplicationRow>, Option<String>)> {
        let status_str = status.map(|s| s.to_string());
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT id, application_id, group_chat_id, applicant_id, message,
                   status as "status: InviteStatus", created_at, handled_by, handled_at
            FROM group_applications
            WHERE group_chat_id = $1 
              AND ($2::text IS NULL OR status::text = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            group_chat_id,
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

    /// 按 application_id 删除申请（取消申请时使用；对外契约使用 UUID 字符串）
    pub async fn delete_by_application_id(&self, application_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM group_applications WHERE application_id = $1",
            application_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
