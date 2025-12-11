// src/services/message_common_service.rs
// 消息通用服务：标记全部已读
//
// ⚠️ 架构决策：
// - ❌ 消息搜索由客户端本地实现（基于客户端缓存的消息数据）
// - ❌ 未读汇总由客户端本地统计（基于聊天列表的未读计数）
// - ✅ 标记全部已读由后端实现（mark_all_as_read）

use crate::error::AppError;
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
}
