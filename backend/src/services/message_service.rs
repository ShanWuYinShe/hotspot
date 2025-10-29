use redis::Client as RedisClient;
use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            SendMessageRequest, EditMessageRequest, RecallMessageRequest, DeleteMessageRequest,
            ListMessagesRequest, MarkAsReadRequest, AddReactionRequest, RemoveReactionRequest,
        },
        response::{MessageResponse, PaginatedResponse},
    },
    error::{AppError, Result},
    models::message::Message,
    repositories::{
        message_repo::MessageRepository,
        conversation_repo::ConversationRepository,
        group_member_repo::GroupMemberRepository,
        user_relationship_repo::UserRelationshipRepository,
    },
};

pub struct MessageService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisClient,
}

impl<'a> MessageService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisClient) -> Self {
        Self { db, redis }
    }

    /// 发送消息
    pub async fn send_message(&self, sender_id: i64, req: &SendMessageRequest) -> Result<String> {
        let message_repo = MessageRepository::new(self.db);

        // 验证目标
        match req.target_type.as_str() {
            "user" => {
                let target_user_id = req.target_id.parse::<i64>()
                    .map_err(|_| AppError::BadRequest("无效的目标用户ID".to_string()))?;
                
                // 检查是否被对方拉黑
                let relationship_repo = UserRelationshipRepository::new(self.db);
                if relationship_repo.is_blocked(target_user_id, sender_id).await? {
                    return Err(AppError::Forbidden("无法发送消息给该用户".to_string()));
                }
            }
            "group" => {
                let group_id = Uuid::parse_str(&req.target_id)
                    .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
                
                // 检查是否为群成员
                let member_repo = GroupMemberRepository::new(self.db);
                if !member_repo.is_member(group_id, sender_id).await? {
                    return Err(AppError::Forbidden("您不是该群组成员".to_string()));
                }

                // 检查是否被禁言
                if member_repo.is_muted(group_id, sender_id).await? {
                    return Err(AppError::Forbidden("您已被禁言".to_string()));
                }
            }
            _ => return Err(AppError::BadRequest("无效的目标类型".to_string())),
        }

        // 验证回复消息是否存在
        let reply_to = if let Some(reply_id_str) = &req.reply_to {
            let reply_id = Uuid::parse_str(reply_id_str)
                .map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?;
            
            if message_repo.find_by_id(reply_id).await?.is_none() {
                return Err(AppError::NotFound("回复的消息不存在".to_string()));
            }
            Some(reply_id)
        } else {
            None
        };

        // 创建消息
        let message_id = Uuid::new_v4();
        let message = Message {
            message_id,
            sender_id,
            target_id: req.target_id.clone(),
            target_type: req.target_type.clone(),
            content: req.content.clone(),
            message_type: req.message_type.clone(),
            reply_to,
            metadata: req.metadata.clone(),
            status: "sent".to_string(),
            created_at: chrono::Utc::now().naive_utc(),
            edited_at: None,
            deleted_at: None,
        };

        message_repo.create(&message).await?;

        // TODO: 更新会话记录
        // TODO: 触发WebSocket推送
        // TODO: 更新未读计数（Redis）

        Ok(message_id.to_string())
    }

    /// 编辑消息
    pub async fn edit_message(&self, user_id: i64, req: &EditMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能编辑自己的消息
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权编辑该消息".to_string()));
        }

        // 不能编辑已撤回或已删除的消息
        if message.status == "recalled" || message.deleted_at.is_some() {
            return Err(AppError::BadRequest("该消息无法编辑".to_string()));
        }

        message_repo.update_content(message_id, &req.content).await?;

        // TODO: 触发WebSocket推送消息编辑事件

        Ok(())
    }

    /// 撤回消息
    pub async fn recall_message(&self, user_id: i64, req: &RecallMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能撤回自己的消息
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权撤回该消息".to_string()));
        }

        // 检查撤回时间限制（2分钟内）
        let now = chrono::Utc::now().naive_utc();
        let elapsed = now.signed_duration_since(message.created_at);
        if elapsed.num_minutes() > 2 {
            return Err(AppError::BadRequest("消息发送超过2分钟，无法撤回".to_string()));
        }

        message_repo.recall(message_id).await?;

        // TODO: 触发WebSocket推送消息撤回事件

        Ok(())
    }

    /// 删除消息
    pub async fn delete_message(&self, user_id: i64, req: &DeleteMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能删除自己的消息
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权删除该消息".to_string()));
        }

        message_repo.soft_delete(message_id).await?;

        Ok(())
    }

    /// 获取消息历史
    pub async fn list_messages(
        &self,
        user_id: i64,
        req: &ListMessagesRequest,
    ) -> Result<PaginatedResponse<MessageResponse>> {
        let message_repo = MessageRepository::new(self.db);

        let limit = req.limit.unwrap_or(50).min(100);
        let before_timestamp = req.before_timestamp;

        let messages = match req.target_type.as_str() {
            "user" => {
                let other_user_id = req.target_id.parse::<i64>()
                    .map_err(|_| AppError::BadRequest("无效的目标用户ID".to_string()))?;
                
                message_repo
                    .list_user_messages(user_id, other_user_id, limit + 1, before_timestamp)
                    .await?
            }
            "group" => {
                let group_id = Uuid::parse_str(&req.target_id)
                    .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
                
                // 检查是否为群成员
                let member_repo = GroupMemberRepository::new(self.db);
                if !member_repo.is_member(group_id, user_id).await? {
                    return Err(AppError::Forbidden("您不是该群组成员".to_string()));
                }

                message_repo
                    .list_group_messages(group_id, limit + 1, before_timestamp)
                    .await?
            }
            _ => return Err(AppError::BadRequest("无效的目标类型".to_string())),
        };

        let has_more = messages.len() > limit as usize;
        let items: Vec<MessageResponse> = messages
            .into_iter()
            .take(limit as usize)
            .map(|m| MessageResponse {
                id: m.message_id.to_string(),
                sender_id: m.sender_id.to_string(),
                target_id: m.target_id,
                target_type: m.target_type,
                content: m.content,
                message_type: m.message_type,
                reply_to: m.reply_to.map(|id| id.to_string()),
                metadata: m.metadata,
                status: m.status,
                created_at: m.created_at.and_utc().timestamp(),
                edited_at: m.edited_at.map(|dt| dt.and_utc().timestamp()),
            })
            .collect();

        let next_cursor = if has_more && !items.is_empty() {
            Some(items.last().unwrap().created_at)
        } else {
            None
        };

        Ok(PaginatedResponse {
            items,
            has_more,
            next_cursor,
        })
    }

    /// 标记消息已读
    pub async fn mark_as_read(&self, user_id: i64, req: &MarkAsReadRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_ids: Result<Vec<Uuid>> = req.message_ids
            .iter()
            .map(|id_str| {
                Uuid::parse_str(id_str)
                    .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))
            })
            .collect();

        let message_ids = message_ids?;
        message_repo.mark_multiple_as_read(user_id, &message_ids).await?;

        // TODO: 更新会话未读计数
        // TODO: 更新Redis缓存

        Ok(())
    }

    /// 添加消息反应
    pub async fn add_reaction(&self, user_id: i64, req: &AddReactionRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 验证消息存在
        if message_repo.find_by_id(message_id).await?.is_none() {
            return Err(AppError::NotFound("消息不存在".to_string()));
        }

        message_repo.add_reaction(message_id, user_id, &req.reaction).await?;

        // TODO: 触发WebSocket推送反应事件

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(&self, user_id: i64, req: &RemoveReactionRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db);

        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        message_repo.remove_reaction(message_id, user_id, &req.reaction).await?;

        Ok(())
    }

    /// 获取未读消息摘要
    pub async fn get_unread_summary(&self, user_id: i64) -> Result<serde_json::Value> {
        let conversation_repo = ConversationRepository::new(self.db);

        let total_unread = conversation_repo.get_total_unread_count(user_id).await?;

        Ok(serde_json::json!({
            "total_unread": total_unread,
            // TODO: 添加更多详细信息
        }))
    }
}
