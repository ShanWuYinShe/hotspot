use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::InviteStatus;

/// 群组申请Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupApplicationRow {
    pub id: i64,
    pub cursor_id: Uuid,
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
            RETURNING id, cursor_id, group_chat_id, applicant_id, message,
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

    /// 按ID查找申请
    pub async fn find_by_id(&self, id: i64) -> Result<Option<GroupApplicationRow>> {
        let row = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT id, cursor_id, group_chat_id, applicant_id, message,
                   status as "status: InviteStatus", created_at, handled_by, handled_at
            FROM group_applications
            WHERE id = $1
            "#,
            id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新申请状态
    pub async fn update_status(
        &self,
        id: i64,
        status: InviteStatus,
        handled_by: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_applications
            SET status = $2, handled_by = $3, handled_at = NOW()
            WHERE id = $1
            "#,
            id,
            status as InviteStatus,
            handled_by
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
                    r#"SELECT id FROM group_applications WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按群组ID查询申请列表
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        status: Option<InviteStatus>,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<Vec<GroupApplicationRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let rows = sqlx::query_as!(
            GroupApplicationRow,
            r#"
            SELECT id, cursor_id, group_chat_id, applicant_id, message,
                   status as "status: InviteStatus", created_at, handled_by, handled_at
            FROM group_applications
            WHERE group_chat_id = $1 
              AND ($2::invitation_status IS NULL OR status = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            group_chat_id,
            status as Option<InviteStatus>,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(rows)
    }

    /// 删除申请（取消申请时使用）
    pub async fn delete(&self, id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM group_applications WHERE id = $1",
            id
        )
        .execute(&self.pool)
        .await?;
        
        Ok(())
    }
}
