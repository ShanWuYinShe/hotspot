// src/services/private_message_service.rs - 私聊消息服务

use chrono::{Duration, Utc};
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            SendPrivateMessageRequest, GetPrivateMessagesRequest, UpdatePrivateMessageRequest,
            RecallPrivateMessageRequest, DeletePrivateMessageRequest, MarkPrivateMessagesReadRequest,
        },
        response::{
            SendPrivateMessageData, PrivateMessageItem, PaginatedData, PaginationMeta, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::{
        private_message_repo::PrivateMessageRepository,
        private_chat_repo::PrivateChatRepository,
        user_repo::UserRepository,
        chat_unread_repo::ChatUnreadRepository,
    },
    types::enums::MessageType,
};

/// 私聊消息服务
pub struct PrivateMessageService {
    db: PgPool,
    message_repo: PrivateMessageRepository,
    chat_repo: PrivateChatRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
}

impl PrivateMessageService {
    pub fn new(db: PgPool) -> Self {
        Self {
            message_repo: PrivateMessageRepository::new(db.clone()),
            chat_repo: PrivateChatRepository::new(db.clone()),
            user_repo: UserRepository::new(db.clone()),
            unread_repo: ChatUnreadRepository::new(db.clone()),
            db,
        }
    }

    /// 发送私聊消息
    pub async fn send(
        &self,
        req: &SendPrivateMessageRequest,
        user_id: i64,
    ) -> Result<SendPrivateMessageData> {
        // 解析私聊ID
        let private_chat_id = Uuid::parse_str(&req.private_chat_id)
            .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

        // 验证用户是否在私聊中
        if !self.chat_repo.is_user_in_chat(private_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 解析消息类型
        let message_type = MessageType::from_str(&req.r#type)
            .ok_or_else(|| AppError::BadRequest("无效的消息类型".to_string()))?;

        // 解析回复消息ID
        let reply_to = req.reply_to.as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?;

        // 创建消息
        let message = self.message_repo.create(
            private_chat_id,
            user_id,
            message_type,
            &req.content,
            reply_to,
            req.file_name.as_deref(),
            req.file_size,
            req.thumbnail_url.as_deref(),
            req.duration,
        ).await?;

        Ok(SendPrivateMessageData {
            id: message.message_id.to_string(),
            sent_at: message.sent_at.to_rfc3339(),
        })
    }

    /// 获取私聊消息历史
    pub async fn get_history(
        &self,
        req: &GetPrivateMessagesRequest,
        user_id: i64,
    ) -> Result<PaginatedData<PrivateMessageItem>> {
        // 解析私聊ID
        let private_chat_id = Uuid::parse_str(&req.private_chat_id)
            .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

        // 验证用户是否在私聊中
        if !self.chat_repo.is_user_in_chat(private_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        let limit = req.limit.unwrap_or(20).min(100) as i64;
        let cursor = req.cursor.as_ref()
            .map(|c| Uuid::parse_str(c))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;

        // 获取消息列表
        let messages = self.message_repo.list_by_chat(private_chat_id, limit + 1, cursor).await?;
        
        let has_more = messages.len() > limit as usize;
        let messages: Vec<_> = messages.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            messages.last().map(|m| m.cursor_id.to_string())
        } else {
            None
        };

        // 批量获取发送者信息
        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let users = self.user_repo.find_by_ids(&sender_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        // 构建响应
        let items: Vec<PrivateMessageItem> = messages.into_iter().map(|m| {
            let user = user_map.get(&m.sender_id);
            PrivateMessageItem {
                id: m.message_id.to_string(),
                r#type: m.r#type.as_str().to_string(),
                content: m.content,
                sender: UserInfo {
                    user_id: m.sender_id.to_string(),
                    nickname: user.map(|u| u.nickname.clone()).unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: user.and_then(|u| u.avatar_url.clone()),
                },
                reply_to: m.reply_to_message_id.map(|id| id.to_string()),
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                is_recalled: m.is_recalled,
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
            }
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 编辑私聊消息
    pub async fn update(
        &self,
        req: &UpdatePrivateMessageRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证是否是发送者
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("只能编辑自己发送的消息".to_string()));
        }

        // 验证消息未被撤回
        if message.is_recalled {
            return Err(AppError::BadRequest("已撤回的消息无法编辑".to_string()));
        }

        // 更新消息内容
        self.message_repo.update_content(message_id, &req.content).await?;

        Ok(())
    }

    /// 撤回私聊消息
    pub async fn recall(
        &self,
        req: &RecallPrivateMessageRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证是否是发送者
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("只能撤回自己发送的消息".to_string()));
        }

        // 验证撤回时间窗口（2分钟内）
        let now = Utc::now();
        if now - message.sent_at > Duration::minutes(2) {
            return Err(AppError::BadRequest("消息发送超过2分钟，无法撤回".to_string()));
        }

        // 撤回消息
        self.message_repo.recall(message_id).await?;

        Ok(())
    }

    /// 删除私聊消息
    pub async fn delete(
        &self,
        req: &DeletePrivateMessageRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self.message_repo.find_by_id(message_id).await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 验证用户是否在私聊中
        if !self.chat_repo.is_user_in_chat(message.private_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 删除消息（仅对自己可见）
        self.message_repo.soft_delete(message_id).await?;

        Ok(())
    }

    /// 标记私聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkPrivateMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let private_chat_id = Uuid::parse_str(&req.private_chat_id)
            .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

        // 验证用户是否在私聊中
        if !self.chat_repo.is_user_in_chat(private_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 解析截止消息ID
        let up_to = req.up_to.as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 更新已读状态
        self.unread_repo.mark_as_read("private_chat", &private_chat_id.to_string(), user_id, up_to).await?;

        Ok(())
    }
}
