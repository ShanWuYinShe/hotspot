use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    types::RoomType,
};

pub struct MessageReadRepository {
    pool: PgPool,
}

impl MessageReadRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 标记房间消息已读（更新room_members表的unread_count和last_read_at）
    pub async fn mark_room_read(
        &self,
        room_id: Uuid,
        user_id: i64,
        up_to_message_id: Option<Uuid>,
    ) -> Result<()> {
        if let Some(msg_id) = up_to_message_id {
            // 标记到特定消息ID之前的所有消息为已读
            // 1. 获取该消息的自增ID
            // 2. 计算该ID之后还有多少未读消息
            // 3. 更新unread_count为剩余未读数
            sqlx::query!(
                r#"
                UPDATE room_members
                SET 
                    unread_count = GREATEST(0, (
                        SELECT COUNT(*)
                        FROM messages
                        WHERE room_id = $1 
                          AND id > (SELECT id FROM messages WHERE message_id = $3 AND room_id = $1)
                          AND deleted_at IS NULL
                    )::INT),
                    last_read_at = NOW()
                WHERE room_id = $1 AND user_id = $2
                "#,
                room_id,
                user_id,
                msg_id
            )
            .execute(&self.pool)
            .await?;
        } else {
            // 标记全部已读
            sqlx::query!(
                r#"
                UPDATE room_members
                SET unread_count = 0, last_read_at = NOW()
                WHERE room_id = $1 AND user_id = $2
                "#,
                room_id,
                user_id
            )
            .execute(&self.pool)
            .await?;
        }
        Ok(())
    }

    /// 增加房间未读数（新消息到达时调用）
    pub async fn increment_unread(
        &self,
        room_id: Uuid,
        exclude_user_id: i64,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET unread_count = unread_count + 1
            WHERE room_id = $1 AND user_id != $2 AND is_deleted = FALSE
            "#,
            room_id,
            exclude_user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 获取特定房间的未读数
    pub async fn get_room_unread_count(
        &self,
        room_id: Uuid,
        user_id: i64,
    ) -> Result<i32> {
        let result = sqlx::query!(
            r#"
            SELECT unread_count
            FROM room_members
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            room_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(result.map(|r| r.unread_count).unwrap_or(0))
    }

    /// 获取用户所有未读消息汇总
    pub async fn get_unread_summary(
        &self,
        user_id: i64,
    ) -> Result<(i32, i32, i32)> {
        let result = sqlx::query!(
            r#"
            SELECT 
                SUM(rm.unread_count) as "total_unread!",
                SUM(CASE WHEN cr.room_type = 'direct' THEN rm.unread_count ELSE 0 END) as "chat_unread!",
                SUM(CASE WHEN cr.room_type IN ('public_group', 'private_group') THEN rm.unread_count ELSE 0 END) as "group_unread!"
            FROM room_members rm
            INNER JOIN chat_rooms cr ON rm.room_id = cr.room_id
            WHERE rm.user_id = $1 AND rm.is_deleted = FALSE AND cr.is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok((
            result.total_unread as i32,
            result.chat_unread as i32,
            result.group_unread as i32,
        ))
    }

    /// 获取用户所有未读房间的详情列表
    pub async fn get_unread_rooms_detail(
        &self,
        user_id: i64,
    ) -> Result<Vec<UnreadRoomDetail>> {
        let results = sqlx::query_as!(
            UnreadRoomDetail,
            r#"
            SELECT 
                cr.room_id as "target_id!",
                cr.room_type::text as "target_type!",
                rm.unread_count,
                cr.last_message_at
            FROM room_members rm
            INNER JOIN chat_rooms cr ON rm.room_id = cr.room_id
            WHERE rm.user_id = $1 
              AND rm.unread_count > 0
              AND rm.is_deleted = FALSE 
              AND cr.is_deleted = FALSE
            ORDER BY cr.last_message_at DESC NULLS LAST
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(results)
    }

    /// 标记所有消息为已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET unread_count = 0, last_read_at = NOW()
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}

#[derive(Debug)]
pub struct UnreadRoomDetail {
    pub target_id: Uuid,
    pub target_type: String,
    pub unread_count: i32,
    pub last_message_at: Option<DateTime<Utc>>,
}


