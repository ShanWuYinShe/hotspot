use crate::repositories::{
    GroupRepository, RepositoryError,
    message_repository::{ConversationRow, GroupMessageRow, MessageRow},
};
use crate::utils::response::AppError;
use crate::utils::result_ext::ResultExt;
use std::sync::Arc;
use tracing::error;
use uuid::Uuid;

/// 消息服务
pub struct MessageService {
    repo: Arc<crate::repositories::PgMessageRepository>,
    group_repo: Arc<crate::repositories::PgGroupRepository>,
}

impl MessageService {
    /// 创建新的消息服务
    pub fn new(
        db: crate::config::db::Database,
        group_repo: Arc<crate::repositories::PgGroupRepository>,
    ) -> Self {
        let repo = Arc::new(crate::repositories::PgMessageRepository::new(
            db.pg_pool.clone(),
        ));
        Self { repo, group_repo }
    }

    // =============== 验证函数 ===============

    /// 检查用户是否在会话中
    pub async fn check_user_in_conversation(
        &self,
        conversation_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.repo
            .check_user_in_conversation(conversation_id, user_id)
            .await
            .map_repository_error("检查用户会话权限")
    }

    /// 检查用户是否在群组中
    pub async fn check_user_in_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.repo
            .check_user_in_group(group_id, user_id)
            .await
            .map_repository_error("检查用户群组权限")
    }

    /// 验证用户对私信会话的访问权限
    async fn ensure_user_access_to_conversation(
        &self,
        conversation_id: Uuid,
        user_id: i64,
    ) -> Result<(), AppError> {
        if !self
            .check_user_in_conversation(conversation_id, user_id)
            .await?
        {
            return Err(AppError::BadRequest("您不在该私信会话中".to_string()));
        }
        Ok(())
    }

    /// 验证用户对群组的访问权限
    async fn ensure_user_access_to_group(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<(), AppError> {
        if !self.check_user_in_group(group_id, user_id).await? {
            return Err(AppError::BadRequest("您不是群组成员".to_string()));
        }
        Ok(())
    }

    // =============== 操作函数 ===============

    /// 保存私信消息
    pub async fn save_direct_message(
        &self,
        conversation_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        content: &str,
        content_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> Result<Uuid, AppError> {
        self.repo
            .save_direct_message(
                conversation_id,
                sender_id,
                recipient_id,
                content,
                content_type,
                latitude,
                longitude,
            )
            .await
            .map_repository_error("保存私信消息")
    }

    /// 保存群聊消息
    pub async fn save_group_message(
        &self,
        group_id: Uuid,
        sender_id: i64,
        content: &str,
        message_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> Result<Uuid, AppError> {
        self.repo
            .save_group_message(
                group_id,
                sender_id,
                content,
                message_type,
                latitude,
                longitude,
            )
            .await
            .map_repository_error("保存群聊消息")
    }

    // =============== 业务函数 ===============

    /// 发送私信
    pub async fn send_direct_message(
        &self,
        conversation_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        content: &str,
        message_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> Result<MessageRow, AppError> {
        // 验证访问权限
        self.ensure_user_access_to_conversation(conversation_id, sender_id)
            .await?;

        // 保存消息
        let message_id = self
            .save_direct_message(
                conversation_id,
                sender_id,
                recipient_id,
                content,
                message_type,
                latitude,
                longitude,
            )
            .await?;

        // 获取消息详情
        self.get_direct_message_by_id(message_id).await
    }

    /// 发送群组消息
    pub async fn send_group_message(
        &self,
        group_id: Uuid,
        sender_id: i64,
        content: &str,
        message_type: &str,
        latitude: f64,
        longitude: f64,
    ) -> Result<GroupMessageRow, AppError> {
        // 验证访问权限
        self.ensure_user_access_to_group(group_id, sender_id)
            .await?;

        // 保存消息
        let message_id = self
            .save_group_message(
                group_id,
                sender_id,
                content,
                message_type,
                latitude,
                longitude,
            )
            .await?;

        // 获取消息详情
        self.get_group_message_by_id(message_id).await
    }

    /// 创建私信会话与用户
    #[allow(dead_code)]
    pub async fn create_dm_session_with_user(
        &self,
        user_id: i64,
        current_user_id: i64,
    ) -> Result<Uuid, AppError> {
        // 创建或获取会话ID
        self.get_or_create_conversation(current_user_id, user_id)
            .await
    }

    /// 获取或创建私信会话
    pub async fn get_or_create_conversation(
        &self,
        user1_id: i64,
        user2_id: i64,
    ) -> Result<Uuid, AppError> {
        self.repo
            .get_or_create_conversation(user1_id, user2_id)
            .await
            .map_repository_error("获取或创建私信会话")
    }

    /// 获取或创建私信会话的别名方法
    pub async fn get_or_create_direct_conversation(
        &self,
        user1_id: i64,
        user2_id: i64,
    ) -> Result<Uuid, AppError> {
        self.get_or_create_conversation(user1_id, user2_id).await
    }

    /// 获取私信会话列表
    pub async fn get_conversations(&self, user_id: i64) -> Result<Vec<ConversationRow>, AppError> {
        self.repo
            .get_conversations(user_id)
            .await
            .map_repository_error("获取会话列表")
    }

    /// 获取私聊会话中的另一方用户ID
    #[allow(dead_code)]
    pub async fn get_other_user_in_conversation(
        &self,
        conversation_id: Uuid,
        current_user_id: i64,
    ) -> Result<i64, AppError> {
        // 权限检查：确保当前用户是此会话的参与者
        let is_member = self
            .repo
            .check_user_in_conversation(conversation_id, current_user_id)
            .await
            .map_repository_error("检查用户会话权限")?;

        if !is_member {
            return Err(AppError::Forbidden("无权访问此会话".to_string()));
        }

        self.repo
            .get_other_user_in_conversation(conversation_id, current_user_id)
            .await
            .map_err(|e| match e {
                RepositoryError::NotFound => AppError::NotFound("会话或用户不存在".to_string()),
                _ => AppError::InternalServerError("获取对方用户失败".to_string()),
            })
    }

    /// 获取私信消息列表（游标分页，规范实现）
    pub async fn get_direct_messages(
        &self,
        conversation_id: Uuid,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> Result<(Vec<MessageRow>, bool, Option<i64>), AppError> {
        // 使用仓库获取消息
        self.repo
            .get_direct_messages(conversation_id, user_id, cursor, limit)
            .await
            .map_err(|e| {
                error!("获取私信消息失败: {:?}", e);
                AppError::InternalServerError("获取私信消息失败".to_string())
            })
    }

    /// 获取群组消息
    pub async fn get_group_messages(
        &self,
        group_id: Uuid,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> Result<(Vec<GroupMessageRow>, bool, Option<i64>), AppError> {
        // 使用仓库获取消息
        self.repo
            .get_group_messages(group_id, user_id, cursor, limit)
            .await
            .map_err(|e| {
                error!("获取群组消息失败: {:?}", e);
                AppError::InternalServerError("获取群组消息失败".to_string())
            })
    }

    /// 标记消息已读
    pub async fn mark_direct_messages_read(
        &self,
        conversation_id: Uuid,
        user_id: i64,
        message_id: Option<Uuid>,
    ) -> Result<(), AppError> {
        // 使用仓库标记消息已读
        self.repo
            .mark_direct_messages_read(conversation_id, user_id, message_id)
            .await
            .map_err(|e| {
                error!("标记消息已读失败: {:?}", e);
                AppError::InternalServerError("标记消息已读失败".to_string())
            })
    }

    /// 标记群组消息已读
    pub async fn mark_group_messages_read(
        &self,
        group_id: Uuid,
        user_id: i64,
        message_id: Option<Uuid>,
    ) -> Result<(), AppError> {
        // 首先验证用户是否在群组中
        let is_member = self.group_repo
            .check_user_in_group(group_id, user_id)
            .await
            .map_repository_error("检查用户群组权限")?;

        if !is_member {
            return Err(AppError::Forbidden("您不在该群组中".to_string()));
        }

        self.repo
            .mark_group_messages_read(group_id, user_id, message_id)
            .await
            .map_repository_error("标记群组消息已读")
    }

    /// 获取用户未读消息数量
    pub async fn get_direct_message_unread_count(&self, user_id: i64) -> Result<i64, AppError> {
        self.repo
            .get_direct_message_unread_count(user_id)
            .await
            .map_err(|e| {
                error!("获取未读消息数量失败: {:?}", e);
                AppError::InternalServerError("获取未读消息数量失败".to_string())
            })
    }

    /// 获取群组未读消息数量
    pub async fn get_group_messages_unread_count(&self, user_id: i64) -> Result<i64, AppError> {
        self.repo
            .get_group_messages_unread_count(user_id)
            .await
            .map_repository_error("获取群组未读消息数量")
    }

    /// 删除群组消息
    pub async fn delete_group_message(
        &self,
        message_id: Uuid,
        user_id: i64,
    ) -> Result<Uuid, AppError> {
        // 检查消息是否存在以及用户是否有权限删除（发送者、群主、管理员）
        let message = self
            .repo
            .get_group_message_by_id(message_id)
            .await
            .map_repository_error("获取群组消息详情以进行删除")?;

        let sender_id = message.sender_id;
        let group_id = message.group_id;

        if sender_id != user_id {
            // 如果不是发送者，检查是否是群主或管理员
            let is_admin = self
                .group_repo
                .check_user_is_group_admin(group_id, user_id)
                .await
                .map_err(AppError::from)?;
            let is_owner = self
                .group_repo
                .check_user_is_group_owner(group_id, user_id)
                .await
                .map_err(AppError::from)?;
            if !is_admin && !is_owner {
                return Err(AppError::Forbidden("无权限删除此消息".to_string()));
            }
        }

        self.repo
            .delete_group_message(message_id, user_id)
            .await
            .map_repository_error("删除群组消息")?;

        Ok(group_id) // 返回群组ID，用于后续通知
    }

    /// 删除私聊消息
    pub async fn delete_direct_message(
        &self,
        message_id: Uuid,
        user_id: i64,
    ) -> Result<Uuid, AppError> {
        // 检查消息是否存在以及用户是否有权限删除（发送者）
        let message = self
            .repo
            .get_direct_message_by_id(message_id)
            .await
            .map_repository_error("获取私聊消息详情以进行删除")?;

        let sender_id = message.sender_id;
        let conversation_id = message.conversation_id;

        if sender_id != user_id {
            return Err(AppError::Forbidden("无权限删除此消息".to_string()));
        }

        self.repo
            .delete_direct_message(message_id, user_id)
            .await
            .map_repository_error("删除私聊消息")?;

        Ok(conversation_id) // 返回会话ID，用于后续通知
    }

    // =============== 仓库访问方法 ===============

    /// 根据ID获取私信消息详情
    pub async fn get_direct_message_by_id(&self, message_id: Uuid) -> Result<MessageRow, AppError> {
        self.repo
            .get_direct_message_by_id(message_id)
            .await
            .map_repository_error("获取私信消息详情")
    }

    /// 根据ID获取群组消息详情
    pub async fn get_group_message_by_id(
        &self,
        message_id: Uuid,
    ) -> Result<GroupMessageRow, AppError> {
        self.repo
            .get_group_message_by_id(message_id)
            .await
            .map_repository_error("获取群组消息详情")
    }
}
