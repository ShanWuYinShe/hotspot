use chrono::{DateTime, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::chat_room::ChatRoom,
    types::enums::RoomType,
};

pub struct ChatRoomRepository {
    pool: PgPool,
}

impl ChatRoomRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建聊天室
    pub async fn create(&self, room: &ChatRoom) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO chat_rooms (
                room_id, room_type, name, description, avatar_url,
                owner_id, password_hash, max_members, created_at, last_message_at
            )
            VALUES ($1, $2::room_type, $3, $4, $5, $6, $7, $8, $9, $10)
            "#,
            room.room_id,
            room.room_type as RoomType,
            room.name,
            room.description,
            room.avatar_url,
            room.owner_id,
            room.password_hash,
            room.max_members,
            room.created_at,
            room.last_message_at,
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 根据room_id查找聊天室
    pub async fn find_by_room_id(&self, room_id: Uuid) -> Result<Option<ChatRoom>> {
        let room = sqlx::query_as!(
            ChatRoom,
            r#"
            SELECT id, room_id, room_type as "room_type: RoomType", name, description,
                   avatar_url, owner_id, password_hash, max_members, created_at,
                   last_message_at, is_deleted, deleted_at
            FROM chat_rooms
            WHERE room_id = $1 AND is_deleted = FALSE
            "#,
            room_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(room)
    }

    /// 根据id(自增主键)查找聊天室
    pub async fn find_by_id(&self, id: i64) -> Result<Option<ChatRoom>> {
        let room = sqlx::query_as!(
            ChatRoom,
            r#"
            SELECT id, room_id, room_type as "room_type: RoomType", name, description,
                   avatar_url, owner_id, password_hash, max_members, created_at,
                   last_message_at, is_deleted, deleted_at
            FROM chat_rooms
            WHERE id = $1 AND is_deleted = FALSE
            "#,
            id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(room)
    }

    /// 查找两个用户之间的私聊
    pub async fn find_direct_chat(&self, user1_id: i64, user2_id: i64) -> Result<Option<ChatRoom>> {
        // 通过room_members表查找两个用户共同的私聊室
        let room = sqlx::query_as!(
            ChatRoom,
            r#"
            SELECT DISTINCT cr.id, cr.room_id, cr.room_type as "room_type: RoomType", 
                   cr.name, cr.description, cr.avatar_url, cr.owner_id,
                   cr.password_hash, cr.max_members, cr.created_at, cr.last_message_at,
                   cr.is_deleted, cr.deleted_at
            FROM chat_rooms cr
            INNER JOIN room_members rm1 ON cr.room_id = rm1.room_id AND rm1.user_id = $1 AND rm1.is_deleted = FALSE
            INNER JOIN room_members rm2 ON cr.room_id = rm2.room_id AND rm2.user_id = $2 AND rm2.is_deleted = FALSE
            WHERE cr.room_type = 'direct'::room_type AND cr.is_deleted = FALSE
            LIMIT 1
            "#,
            user1_id,
            user2_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(room)
    }

    /// 按名称搜索群组(公开群和私密群,不包括私聊)
    pub async fn search_by_name(
        &self,
        name: &str,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<ChatRoom>, Option<String>)> {
        // 1. 根据业务ID查询自增主键id
        let cursor_id = if let Some(room_id_str) = cursor {
            let room_id = Uuid::parse_str(&room_id_str).ok();
            if let Some(rid) = room_id {
                sqlx::query!("SELECT id FROM chat_rooms WHERE room_id = $1", rid)
                    .fetch_optional(&self.pool)
                    .await?
                    .map(|r| r.id)
            } else {
                None
            }
        } else {
            None
        };

        // 2. 查询limit+1条记录
        let search_pattern = format!("%{}%", name);
        let rooms = sqlx::query_as!(
            ChatRoom,
            r#"
            SELECT id, room_id, room_type as "room_type: RoomType", name, description,
                   avatar_url, owner_id, password_hash, max_members, created_at,
                   last_message_at, is_deleted, deleted_at
            FROM chat_rooms
            WHERE name ILIKE $1
              AND room_type != 'direct'::room_type
              AND is_deleted = FALSE
              AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id
            LIMIT $3
            "#,
            search_pattern,
            cursor_id,
            (limit + 1) as i64
        )
        .fetch_all(&self.pool)
        .await?;

        // 3. 判断has_more并返回next_cursor
        let has_more = rooms.len() > limit as usize;
        let items: Vec<ChatRoom> = rooms.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            items.last().map(|r| r.room_id.to_string())
        } else {
            None
        };

        Ok((items, next_cursor))
    }

    /// 更新聊天室资料
    pub async fn update_profile(
        &self,
        room_id: Uuid,
        name: Option<&str>,
        description: Option<&str>,
        avatar_url: Option<&str>,
    ) -> Result<()> {
        // 动态构建SQL
        if let Some(n) = name {
            sqlx::query!(
                "UPDATE chat_rooms SET name = $2 WHERE room_id = $1",
                room_id,
                n
            )
            .execute(&self.pool)
            .await?;
        }

        if let Some(d) = description {
            sqlx::query!(
                "UPDATE chat_rooms SET description = $2 WHERE room_id = $1",
                room_id,
                d
            )
            .execute(&self.pool)
            .await?;
        }

        if let Some(a) = avatar_url {
            sqlx::query!(
                "UPDATE chat_rooms SET avatar_url = $2 WHERE room_id = $1",
                room_id,
                a
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    /// 更新最后消息时间
    pub async fn update_last_message_at(&self, room_id: Uuid, timestamp: DateTime<Utc>) -> Result<()> {
        sqlx::query!(
            "UPDATE chat_rooms SET last_message_at = $2 WHERE room_id = $1",
            room_id,
            timestamp
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除聊天室
    pub async fn soft_delete(&self, room_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE chat_rooms
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE room_id = $1
            "#,
            room_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 更新聊天室隐私设置
    pub async fn update_privacy(
        &self,
        room_id: Uuid,
        room_type: Option<RoomType>,
        password_hash: Option<String>,
        max_members: Option<i32>,
    ) -> Result<()> {
        // 构建动态更新查询
        if let Some(rt) = room_type {
            sqlx::query!(
                r#"UPDATE chat_rooms SET room_type = $2::room_type WHERE room_id = $1"#,
                room_id,
                rt as RoomType
            )
            .execute(&self.pool)
            .await?;
        }

        if let Some(ph) = password_hash {
            sqlx::query!(
                "UPDATE chat_rooms SET password_hash = $2 WHERE room_id = $1",
                room_id,
                ph
            )
            .execute(&self.pool)
            .await?;
        }

        if let Some(mm) = max_members {
            sqlx::query!(
                "UPDATE chat_rooms SET max_members = $2 WHERE room_id = $1",
                room_id,
                mm
            )
            .execute(&self.pool)
            .await?;
        }

        Ok(())
    }

    /// 获取用户的聊天室列表（游标分页）
    /// cursor: room_id (UUID字符串)
    pub async fn list_user_rooms(
        &self,
        user_id: i64,
        room_type: Option<RoomType>,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<ChatRoom>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let room_id_val = Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM chat_rooms WHERE room_id = $1 AND is_deleted = FALSE",
                room_id_val
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 根据是否有 room_type 过滤
        let rooms = if let Some(rt) = room_type {
            sqlx::query_as!(
                ChatRoom,
                r#"
                SELECT DISTINCT c.id, c.room_id, c.room_type as "room_type: RoomType", c.name, c.description,
                       c.avatar_url, c.owner_id, c.password_hash, c.max_members,
                       c.created_at, c.last_message_at, c.is_deleted, c.deleted_at
                FROM chat_rooms c
                INNER JOIN room_members rm ON c.room_id = rm.room_id
                WHERE rm.user_id = $1
                AND c.is_deleted = FALSE
                AND rm.is_deleted = FALSE
                AND c.room_type = $2::room_type
                AND ($3::BIGINT IS NULL OR c.id > $3::BIGINT)
                ORDER BY c.id ASC
                LIMIT $4
                "#,
                user_id,
                rt as RoomType,
                cursor_id,
                limit
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                ChatRoom,
                r#"
                SELECT DISTINCT c.id, c.room_id, c.room_type as "room_type: RoomType", c.name, c.description,
                       c.avatar_url, c.owner_id, c.password_hash, c.max_members,
                       c.created_at, c.last_message_at, c.is_deleted, c.deleted_at
                FROM chat_rooms c
                INNER JOIN room_members rm ON c.room_id = rm.room_id
                WHERE rm.user_id = $1
                AND c.is_deleted = FALSE
                AND rm.is_deleted = FALSE
                AND ($2::BIGINT IS NULL OR c.id > $2::BIGINT)
                ORDER BY c.id ASC
                LIMIT $3
                "#,
                user_id,
                cursor_id,
                limit
            )
            .fetch_all(&self.pool)
            .await?
        };

        Ok(rooms)
    }
}


