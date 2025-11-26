use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;

/// 好友申请Row
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct FriendRequestRow {
    pub id: i64,
    pub request_id: Uuid,
    pub from_user_id: i64,
    pub to_user_id: i64,
    pub message: Option<String>,
    pub status: String, // 'pending', 'accepted', 'rejected'
    pub created_at: DateTime<Utc>,
    pub responded_at: Option<DateTime<Utc>>,
}

/// 好友申请Repository（替代原来的UserRelationshipRepository）
pub struct UserRelationshipRepository {
    pool: PgPool,
}

impl UserRelationshipRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM user_friend_requests WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 创建好友申请
    pub async fn create(
        &self,
        from_user_id: i64,
        to_user_id: i64,
        message: Option<String>,
    ) -> Result<Uuid> {
        let result = sqlx::query!(
            r#"
            INSERT INTO user_friend_requests (from_user_id, to_user_id, message)
            VALUES ($1, $2, $3)
            RETURNING request_id
            "#,
            from_user_id,
            to_user_id,
            message
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.request_id)
    }

    /// 通过ID查找好友申请
    pub async fn find_by_id(&self, request_id: Uuid) -> Result<Option<FriendRequestRow>> {
        let request = sqlx::query_as!(
            FriendRequestRow,
            r#"
            SELECT id, request_id, from_user_id, to_user_id, message, 
                   status::text as "status!", created_at, responded_at
            FROM user_friend_requests
            WHERE request_id = $1
            "#,
            request_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(request)
    }

    /// 更新好友申请状态
    pub async fn update_status(&self, request_id: Uuid, status: &str) -> Result<()> {
        sqlx::query(
            r#"
            UPDATE user_friend_requests
            SET status = $2::invitation_status, responded_at = NOW()
            WHERE request_id = $1
            "#
        )
        .bind(request_id)
        .bind(status)
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除好友申请
    pub async fn delete(&self, request_id: Uuid) -> Result<()> {
        sqlx::query!(
            "DELETE FROM user_friend_requests WHERE request_id = $1",
            request_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 查找好友申请（从某用户到某用户）
    pub async fn find_relationship(
        &self,
        from_user_id: i64,
        to_user_id: i64,
    ) -> Result<Option<FriendRequestRow>> {
        let request = sqlx::query_as!(
            FriendRequestRow,
            r#"
            SELECT id, request_id, from_user_id, to_user_id, message,
                   status::text as "status!", created_at, responded_at
            FROM user_friend_requests
            WHERE from_user_id = $1 AND to_user_id = $2
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            from_user_id,
            to_user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(request)
    }

    /// 列出从某用户发出的好友申请
    pub async fn list_from_user(
        &self,
        from_user_id: i64,
        status: Option<&str>,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<FriendRequestRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let requests = sqlx::query_as!(
            FriendRequestRow,
            r#"
            SELECT id, request_id, from_user_id, to_user_id, message,
                   status::text as "status!", created_at, responded_at
            FROM user_friend_requests
            WHERE from_user_id = $1 
              AND ($2::text IS NULL OR status::text = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            from_user_id,
            status,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = requests.last().map(|r| r.request_id);
        Ok((requests, next_cursor))
    }

    /// 列出发送给某用户的好友申请
    pub async fn list_to_user(
        &self,
        to_user_id: i64,
        status: Option<&str>,
        limit: i32,
        cursor: Option<Uuid>,
    ) -> Result<(Vec<FriendRequestRow>, Option<Uuid>)> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let requests = sqlx::query_as!(
            FriendRequestRow,
            r#"
            SELECT id, request_id, from_user_id, to_user_id, message,
                   status::text as "status!", created_at, responded_at
            FROM user_friend_requests
            WHERE to_user_id = $1 
              AND ($2::text IS NULL OR status::text = $2)
              AND id < $3
            ORDER BY id DESC
            LIMIT $4
            "#,
            to_user_id,
            status,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = requests.last().map(|r| r.request_id);
        Ok((requests, next_cursor))
    }

    /// 删除好友关系（删除双向的已接受的好友申请）
    pub async fn delete_friendship(&self, user_a_id: i64, user_b_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM user_friend_requests
            WHERE status = 'accepted'
            AND ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
            "#,
            user_a_id,
            user_b_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 检查两个用户是否是好友
    pub async fn are_friends(&self, user_a_id: i64, user_b_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM user_friend_requests
                WHERE status = 'accepted'
                AND ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
            ) as "exists!"
            "#,
            user_a_id,
            user_b_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }

    /// 统计好友数量
    pub async fn count_friends(&self, user_id: i64) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM user_friend_requests
            WHERE status = 'accepted'
            AND (from_user_id = $1 OR to_user_id = $1)
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.count)
    }
}
