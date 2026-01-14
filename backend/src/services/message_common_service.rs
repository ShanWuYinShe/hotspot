// src/services/message_common_service.rs
// 消息通用服务：标记全部已读
//
// ⚠️ 架构决策：
// - ❌ 消息搜索由客户端本地实现（基于客户端缓存的消息数据）
// - ❌ 未读汇总由客户端本地统计（基于聊天列表的未读计数）
// - ✅ 标记全部已读由后端实现（mark_all_as_read）

use crate::error::AppError;
use crate::repositories::chat_unread_repo::ChatUnreadRepository;

#[derive(Clone)]
pub struct MessageCommonService {
    unread_repo: ChatUnreadRepository,
}

impl MessageCommonService {
    pub fn new(unread_repo: ChatUnreadRepository) -> Self {
        Self { unread_repo }
    }

    /// 标记所有消息已读
    pub async fn mark_all_as_read(&self, user_id: i64) -> Result<(), AppError> {
        self.unread_repo.mark_all_as_read_for_user(user_id).await
    }
}
