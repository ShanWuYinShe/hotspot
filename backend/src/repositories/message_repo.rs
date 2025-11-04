use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::message::Message,
    types::enums::MessageType,
};

pub struct MessageRepository {
    db: PgPool,
}

impl MessageRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 创建消息
    pub async fn create(&self, message: &Message) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO messages (
                message_id, sender_id, room_id, content, 
                message_type, reply_to, status
            )
            VALUES ($1, $2, $3, $4, $5::message_type, $6, $7)
            RETURNING message_id
            "#,
            message.message_id,
            message.sender_id,
            message.room_id,
            message.content,
            message.message_type as MessageType,
            message.reply_to,
            message.status,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.message_id)
    }

    /// 根据ID查找消息
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<Message>> {
        let message = sqlx::query_as!(
            Message,
            r#"
            SELECT id, message_id, room_id, sender_id, content, message_type as "message_type: MessageType", is_edited,
                   created_at, edited_at, reply_to, status, is_deleted, deleted_at,
                   mentions, file_name, file_size, thumbnail_url, duration,
                   system_event, affected_users, latitude, longitude, location_name
            FROM messages 
            WHERE message_id = $1
            "#,
            message_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(message)
    }

    /// 更新消息内容
    pub async fn update_content(&self, message_id: Uuid, content: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET content = $2, edited_at = NOW()
            WHERE message_id = $1
            "#,
            message_id,
            content
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 更新消息状态
    pub async fn update_status(&self, message_id: Uuid, status: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET status = $2
            WHERE message_id = $1
            "#,
            message_id,
            status
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid) -> Result<()> {
        self.update_status(message_id, "recalled").await
    }

    /// 删除消息（软删除）
    pub async fn soft_delete(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE messages
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE message_id = $1
            "#,
            message_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取聊天室消息历史（统一私聊和群聊，使用游标分页）
    /// cursor: message_id (UUID字符串)
    pub async fn list_room_messages(
        &self,
        room_id: Uuid,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<Message>, Option<String>)> {
        // 1. 根据业务ID查询自增主键id
        let cursor_id = if let Some(msg_id_str) = cursor {
            let msg_id = Uuid::parse_str(&msg_id_str).ok();
            if let Some(mid) = msg_id {
                sqlx::query!("SELECT id FROM messages WHERE message_id = $1", mid)
                    .fetch_optional(&self.db)
                    .await?
                    .map(|r| r.id)
            } else {
                None
            }
        } else {
            None
        };

        // 2. 查询limit+1条记录
        let messages = sqlx::query_as!(
            Message,
            r#"
            SELECT id, message_id, room_id, sender_id, content, message_type as "message_type: MessageType", is_edited,
                   created_at, edited_at, reply_to, status, is_deleted, deleted_at,
                   mentions, file_name, file_size, thumbnail_url, duration,
                   system_event, affected_users, latitude, longitude, location_name
            FROM messages
            WHERE room_id = $1
              AND is_deleted = FALSE
              AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id
            LIMIT $3
            "#,
            room_id,
            cursor_id,
            (limit + 1) as i64
        )
        .fetch_all(&self.db)
        .await?;

        // 3. 判断has_more并返回next_cursor
        let has_more = messages.len() > limit as usize;
        let items: Vec<Message> = messages.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items.last().map(|m| m.message_id.to_string())
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    // list_group_messages 已合并到 list_room_messages

    /// 标记聊天室消息已读（使用 message_read_records 表）
    pub async fn mark_room_as_read(
        &self,
        user_id: i64,
        room_id: Uuid,
        up_to_message_id: Option<Uuid>,
    ) -> Result<()> {
        // 如果没有指定 up_to_message_id，查询该房间最新的消息ID
        let last_read_message_id = if let Some(msg_id) = up_to_message_id {
            msg_id
        } else {
            // 查询房间最新的消息ID
            let latest = sqlx::query!(
                r#"
                SELECT message_id
                FROM messages
                WHERE room_id = $1 AND is_deleted = FALSE
                ORDER BY created_at DESC
                LIMIT 1
                "#,
                room_id
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            match latest {
                Some(rec) => rec.message_id,
                None => return Ok(()), // 没有消息，直接返回
            }
        };
        
        sqlx::query!(
            r#"
            INSERT INTO message_read_records (user_id, room_id, last_read_message_id, read_at)
            VALUES ($1, $2, $3, NOW())
            ON CONFLICT (user_id, room_id) 
            DO UPDATE SET last_read_message_id = $3, read_at = NOW()
            "#,
            user_id,
            room_id,
            last_read_message_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 批量标记已读（废弃 - 使用 mark_room_as_read 代替）
    pub async fn mark_multiple_as_read(
        &self,
        _user_id: i64,
        _message_ids: &[Uuid],
    ) -> Result<()> {
        // 这个方法已废弃，应该使用 mark_room_as_read
        Err(AppError::BadRequest("Use mark_room_as_read instead".to_string()))
    }

    /// 添加消息反应
    pub async fn add_reaction(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO message_reactions (message_id, user_id, reaction_type)
            VALUES ($1, $2, $3)
            ON CONFLICT (message_id, user_id, reaction_type) DO NOTHING
            "#,
            message_id,
            user_id,
            reaction
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(
        &self,
        message_id: Uuid,
        user_id: i64,
        reaction: &str,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM message_reactions
            WHERE message_id = $1 AND user_id = $2 AND reaction_type = $3
            "#,
            message_id,
            user_id,
            reaction
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 获取聊天室未读消息数量（统一私聊和群聊，使用 room_members 表的 unread_count）
    pub async fn count_unread_room_messages(
        &self,
        user_id: i64,
        room_id: Uuid,
    ) -> Result<i32> {
        let rec = sqlx::query!(
            r#"
            SELECT unread_count
            FROM room_members
            WHERE user_id = $1 AND room_id = $2 AND is_deleted = FALSE
            "#,
            user_id,
            room_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.map(|r| r.unread_count).unwrap_or(0))
    }

    /// 获取两个用户之间的私聊消息（包装方法）
    pub async fn list_user_messages(
        &self,
        user1_id: i64,
        user2_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<Message>, Option<String>)> {
        // 首先查找两个用户之间的私聊室
        use crate::repositories::chat_room_repo::ChatRoomRepository;
        let chat_room_repo = ChatRoomRepository::new(self.db.clone());
        
        if let Some(room) = chat_room_repo.find_direct_chat(user1_id, user2_id).await? {
            // 如果私聊室存在，返回消息列表
            self.list_room_messages(room.room_id, limit, cursor).await
        } else {
            // 如果私聊室不存在，返回空列表
            Ok((Vec::new(), None))
        }
    }

}
