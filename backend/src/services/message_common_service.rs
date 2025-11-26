// src/services/message_common_service.rs
// 消息通用服务：标记全部已读、批量标记已读
//
// ⚠️ 架构决策：
// - ❌ 消息搜索由前端本地实现（基于客户端缓存的消息数据）
// - ❌ 未读汇总由前端本地统计（基于聊天列表的未读计数）
// - ✅ 批量标记已读由后端实现（跨聊天批量操作）

use uuid::Uuid;
use crate::{
    dto::{
        request::MarkAsReadBatchRequest,
        response::{BatchResultData, BatchItemResult},
    },
    error::AppError,
};
use sqlx::PgPool;

pub struct MessageCommonService {
    pool: PgPool,
}

impl MessageCommonService {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 标记所有消息已读
    pub async fn mark_all_as_read(&self, user_id: i64) -> Result<(), AppError> {
        sqlx::query!(
            r#"
            UPDATE chat_unread_counts
            SET unread_count = 0,
                last_read_at = NOW(),
                updated_at = NOW()
            WHERE user_id = $1 AND unread_count > 0
            "#,
            user_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 批量标记消息已读
    pub async fn mark_as_read_batch(
        &self,
        user_id: i64,
        req: MarkAsReadBatchRequest,
    ) -> Result<BatchResultData, AppError> {
        let total = req.items.len() as i32;
        let mut succeeded = 0;
        let mut failed = 0;
        let mut results = Vec::new();

        for (index, item) in req.items.into_iter().enumerate() {
            // 解析 chat_id
            let chat_uuid = match Uuid::parse_str(&item.chat_id) {
                Ok(id) => id,
                Err(_) => {
                    results.push(BatchItemResult {
                        index: index as i32,
                        code: 1002,
                        message: Some("无效的聊天ID格式".to_string()),
                    });
                    failed += 1;
                    continue;
                }
            };

            // 解析 up_to 消息ID
            let up_to_uuid = match Uuid::parse_str(&item.up_to) {
                Ok(id) => id,
                Err(_) => {
                    results.push(BatchItemResult {
                        index: index as i32,
                        code: 1002,
                        message: Some("无效的消息ID格式".to_string()),
                    });
                    failed += 1;
                    continue;
                }
            };

            // 根据聊天类型更新已读状态
            let result = match item.chat_type.as_str() {
                "private_chat" => {
                    self.mark_private_chat_read(user_id, chat_uuid, up_to_uuid).await
                }
                "group_chat" => {
                    self.mark_group_chat_read(user_id, chat_uuid, up_to_uuid).await
                }
                _ => {
                    results.push(BatchItemResult {
                        index: index as i32,
                        code: 1002,
                        message: Some("无效的聊天类型".to_string()),
                    });
                    failed += 1;
                    continue;
                }
            };

            match result {
                Ok(_) => {
                    results.push(BatchItemResult {
                        index: index as i32,
                        code: 0,
                        message: None,
                    });
                    succeeded += 1;
                }
                Err(e) => {
                    results.push(BatchItemResult {
                        index: index as i32,
                        code: 5000,
                        message: Some(e.to_string()),
                    });
                    failed += 1;
                }
            }
        }

        Ok(BatchResultData {
            total,
            succeeded,
            failed,
            results,
        })
    }

    /// 标记私聊消息已读
    async fn mark_private_chat_read(
        &self,
        user_id: i64,
        chat_id: Uuid,
        up_to: Uuid,
    ) -> Result<(), AppError> {
        // 获取 up_to 消息的发送时间
        let sent_at = sqlx::query_scalar!(
            r#"SELECT sent_at FROM private_messages WHERE message_id = $1"#,
            up_to
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 更新已读状态
        sqlx::query!(
            r#"
            INSERT INTO chat_unread_counts (user_id, chat_type, chat_id, last_read_at, unread_count)
            VALUES ($1, 'private_chat', $2, $3, 0)
            ON CONFLICT (user_id, chat_type, chat_id) DO UPDATE
            SET last_read_at = GREATEST(chat_unread_counts.last_read_at, $3),
                unread_count = GREATEST(0, chat_unread_counts.unread_count - (
                    SELECT COUNT(*) FROM private_messages 
                    WHERE private_chat_id = $2 
                      AND sender_id != $1 
                      AND sent_at <= $3 
                      AND sent_at > COALESCE(chat_unread_counts.last_read_at, '1970-01-01'::timestamptz)
                )),
                updated_at = NOW()
            "#,
            user_id,
            chat_id,
            sent_at
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 标记群聊消息已读
    async fn mark_group_chat_read(
        &self,
        user_id: i64,
        chat_id: Uuid,
        up_to: Uuid,
    ) -> Result<(), AppError> {
        // 获取 up_to 消息的发送时间
        let sent_at = sqlx::query_scalar!(
            r#"SELECT sent_at FROM group_messages WHERE message_id = $1"#,
            up_to
        )
        .fetch_optional(&self.pool)
        .await?
        .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 更新已读状态
        sqlx::query!(
            r#"
            INSERT INTO chat_unread_counts (user_id, chat_type, chat_id, last_read_at, unread_count)
            VALUES ($1, 'group_chat', $2, $3, 0)
            ON CONFLICT (user_id, chat_type, chat_id) DO UPDATE
            SET last_read_at = GREATEST(chat_unread_counts.last_read_at, $3),
                unread_count = GREATEST(0, chat_unread_counts.unread_count - (
                    SELECT COUNT(*) FROM group_messages 
                    WHERE group_chat_id = $2 
                      AND sender_id != $1 
                      AND sent_at <= $3 
                      AND sent_at > COALESCE(chat_unread_counts.last_read_at, '1970-01-01'::timestamptz)
                )),
                updated_at = NOW()
            "#,
            user_id,
            chat_id,
            sent_at
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }
}
