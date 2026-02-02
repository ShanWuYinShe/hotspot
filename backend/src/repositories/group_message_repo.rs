use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use std::collections::HashMap;
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MessageType;

use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 群聊消息Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMessageRow {
    pub id: i64,
    pub message_id: Uuid,
    // cursor field removed
    pub group_chat_id: Uuid,
    pub sender_id: i64,
    pub r#type: MessageType,
    pub content: String,
    pub reply_to_message_id: Option<Uuid>,
    pub mentions: Option<Vec<i64>>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
    pub sent_at: DateTime<Utc>,
    pub edited_at: Option<DateTime<Utc>>,
    pub is_recalled: bool,
    pub recalled_by: Option<i64>,
    pub recalled_at: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct GroupMessageMeta {
    pub group_chat_id: Uuid,
    pub sender_id: i64,
    pub is_recalled: bool,
    pub sent_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
pub struct GroupMessageRepository {
    pool: PgPool,
}

impl GroupMessageRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取消息元数据（用于权限检查）
    pub async fn get_meta(&self, message_id: Uuid) -> Result<Option<GroupMessageMeta>> {
        let row = sqlx::query_as!(
            GroupMessageMeta,
            r#"
            SELECT group_chat_id, sender_id, is_recalled, sent_at
            FROM group_messages
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 按消息ID批量获取消息
    pub async fn find_by_ids(&self, message_ids: &[Uuid]) -> Result<Vec<GroupMessageRow>> {
        if message_ids.is_empty() {
            return Ok(vec![]);
        }
        let rows = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE message_id = ANY($1) AND is_deleted = FALSE
            "#,
            message_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows)
    }

    /// 按消息ID查找消息
    pub async fn find_by_id(&self, message_id: Uuid) -> Result<Option<GroupMessageRow>> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE message_id = $1 AND is_deleted = FALSE
            "#,
            message_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 发送群聊消息
    #[allow(clippy::too_many_arguments)]
    pub async fn create(
        &self,
        group_chat_id: Uuid,
        sender_id: i64,
        r#type: MessageType,
        content: String,
        reply_to_message_id: Option<Uuid>,
        mentions: Option<Vec<i64>>,
        file_name: Option<&str>,
        file_size: Option<i64>,
        thumbnail_url: Option<&str>,
        duration: Option<i32>,
    ) -> Result<GroupMessageRow> {
        let row = sqlx::query_as!(
            GroupMessageRow,
            r#"
            INSERT INTO group_messages (group_chat_id, sender_id, type, content, reply_to_message_id, mentions, file_name, file_size, thumbnail_url, duration)
            VALUES ($1, $2, $3::message_type, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id, message_id, group_chat_id, sender_id,
                      type as "type: MessageType", content, reply_to_message_id, mentions,
                      file_name, file_size, thumbnail_url, duration,
                      sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            "#,
            group_chat_id,
            sender_id,
            r#type as MessageType,
            content,
            reply_to_message_id,
            mentions.as_deref(),
            file_name,
            file_size,
            thumbnail_url,
            duration
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// cursor: 游标（cursor 的 UUID），用于分页
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        limit: i64,
        cursor: Option<String>,
        include_system: bool,
    ) -> Result<(Vec<GroupMessageRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        // 查询 limit + 1 条以判断是否有更多数据
        let rows = sqlx::query_as!(
            GroupMessageRow,
            r#"
            SELECT id, message_id, group_chat_id, sender_id,
                   type as "type: MessageType", content, reply_to_message_id, mentions,
                   file_name, file_size, thumbnail_url, duration,
                   sent_at, edited_at, is_recalled, recalled_by, recalled_at, is_deleted, deleted_at
            FROM group_messages
            WHERE group_chat_id = $1 AND id < $2 AND is_deleted = FALSE
              AND ($4::bool = TRUE OR type != 'system'::message_type)
            ORDER BY id DESC
            LIMIT $3
            "#,
            group_chat_id,
            last_id,
            limit + 1,
            include_system
        )
        .fetch_all(&self.pool)
        .await?;

        crate::utils::pagination::paginate_with_limit_plus_one(rows, limit as i32, |r| {
            encode_cursor(&r.id)
        })
    }

    /// 更新消息内容
    pub async fn update_content(&self, message_id: Uuid, content: String) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET content = $2, edited_at = NOW()
            WHERE message_id = $1
            "#,
            message_id,
            content
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 撤回消息
    pub async fn recall(&self, message_id: Uuid, recalled_by: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET is_recalled = TRUE, recalled_by = $2, recalled_at = NOW(), content = '[消息已撤回]'
            WHERE message_id = $1
            "#,
            message_id,
            recalled_by
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 软删除消息
    pub async fn soft_delete(&self, message_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_messages
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE message_id = $1
            "#,
            message_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 统计群组消息总数
    pub async fn count_by_group(&self, group_chat_id: Uuid) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_messages
            WHERE group_chat_id = $1 AND is_deleted = FALSE
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(result.count)
    }

    /// 统计今日活跃成员数量（按发言用户去重）
    pub async fn count_active_members_today(&self, group_chat_id: Uuid) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            SELECT COUNT(DISTINCT sender_id) as "count!"
            FROM group_messages
            WHERE group_chat_id = $1
              AND is_deleted = FALSE
              AND sent_at >= CURRENT_DATE
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?;
        Ok(result.count)
    }

    /// 批量获取消息已读数量
    pub async fn get_read_counts(&self, message_ids: &[Uuid]) -> Result<HashMap<Uuid, i64>> {
        if message_ids.is_empty() {
            return Ok(HashMap::new());
        }
        let rows = sqlx::query!(
            r#"
            SELECT message_id, COUNT(*) as "count!"
            FROM group_message_read_status
            WHERE message_id = ANY($1)
            GROUP BY message_id
            "#,
            message_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map = HashMap::new();
        for row in rows {
            map.insert(row.message_id, row.count);
        }
        Ok(map)
    }

    /// 获取消息已读用户列表（分页）
    pub async fn get_read_status(
        &self,
        message_id: Uuid,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<(i64, DateTime<Utc>)>, Option<String>)> {
        // cursor: (read_at, user_id)
        use crate::utils::pagination::TimeCursor;
        let (last_read_at, last_user_id) = if let Some(c) = cursor {
            decode_cursor::<TimeCursor>(&c)?
        } else {
            (DateTime::<Utc>::MAX_UTC, i64::MAX)
        };

        let rows = sqlx::query!(
            r#"
            SELECT user_id, read_at
            FROM group_message_read_status
            WHERE message_id = $1 AND (read_at, user_id) < ($2, $3)
            ORDER BY read_at DESC, user_id DESC
            LIMIT $4
            "#,
            message_id,
            last_read_at,
            last_user_id,
            limit + 1 // 使用 limit + 1 来判断是否有更多数据
        )
        .fetch_all(&self.pool)
        .await?;

        // 判断是否有更多数据
        let has_more = rows.len() > limit as usize;
        let items: Vec<_> = rows.into_iter().take(limit as usize).collect();

        let next_cursor = if has_more {
            items
                .last()
                .map(|r| encode_cursor(&(r.read_at, r.user_id)))
                .transpose()?
        } else {
            None
        };

        let result = items.into_iter().map(|r| (r.user_id, r.read_at)).collect();
        Ok((result, next_cursor))
    }

    /// 列出布隆过滤器用的 message_id（按 id 分页）
    pub async fn list_message_ids_after(
        &self,
        last_id: i64,
        limit: i64,
    ) -> Result<Vec<(i64, Uuid)>> {
        let rows = sqlx::query!(
            r#"
            SELECT id, message_id
            FROM group_messages
            WHERE id > $1 AND is_deleted = FALSE
            ORDER BY id
            LIMIT $2
            "#,
            last_id,
            limit
        )
        .fetch_all(&self.pool)
        .await?;
        Ok(rows.into_iter().map(|r| (r.id, r.message_id)).collect())
    }
}
