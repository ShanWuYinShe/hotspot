use redis::aio::ConnectionManager as RedisConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::SYSTEM_USER_ID,
    dto::{
        request::{
            SendMessageRequest, EditMessageRequest, RecallMessageRequest, DeleteMessageRequest,
            ListMessagesRequest, MarkReadRequest, ReactToMessageRequest, ListReactionsRequest,
            MarkAllReadRequest, GetUnreadSummaryRequest, GetUnreadCountRequest,
            MarkAsReadRequest, AddReactionRequest, RemoveReactionRequest,
        },
        response::{MessageResponse, PaginatedResponse, UserResponse, ReplyInfo},
    },
    error::{AppError, Result},
    events::{event_bus::EventBus, events::AppEvent},
    models::message::Message,
    repositories::{
        message_repo::MessageRepository,
        message_read_repo::MessageReadRepository,
        group_member_repo::GroupMemberRepository,
        user_relationship_repo::UserRelationshipRepository,
    },
    types::enums::MessageType,
};

pub struct ChatMessagesService<'a> {
    db: &'a PgPool,
    #[allow(dead_code)]
    redis: &'a RedisConnectionManager,
    event_bus: Arc<EventBus>,
}

impl<'a> ChatMessagesService<'a> {
    pub fn new(db: &'a PgPool, redis: &'a RedisConnectionManager, event_bus: Arc<EventBus>) -> Self {
        Self { db, redis, event_bus }
    }

    /// 发送消息
    pub async fn send_message(&self, sender_id: i64, req: &SendMessageRequest) -> Result<String> {
        let message_repo = MessageRepository::new(self.db.clone());

        // 验证目标
        match req.target_type.as_str() {
            "user" => {
                let target_user_id = req.target_id.parse::<i64>()
                    .map_err(|_| AppError::BadRequest("无效的目标用户ID".to_string()))?;
                
                // 检查是否被对方拉黑
                let relationship_repo = UserRelationshipRepository::new(self.db.clone());
                if relationship_repo.is_blocked(target_user_id, sender_id).await? {
                    return Err(AppError::Forbidden("无法发送消息给该用户".to_string()));
                }
            }
            "group" => {
                let group_id = Uuid::parse_str(&req.target_id)
                    .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
                
                // 检查是否为群成员
                let member_repo = GroupMemberRepository::new(self.db.clone());
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
        let room_id = Uuid::parse_str(&req.target_id)
            .map_err(|_| AppError::BadRequest("无效的会话ID".to_string()))?;
        
        let message_type = MessageType::from_str(&req.message_type)
            .ok_or_else(|| AppError::BadRequest(format!("无效的消息类型: {}", req.message_type)))?;

        let message = Message {
            id: 0, // 由数据库生成
            message_id,
            room_id,
            sender_id: Some(sender_id),
            content: req.content.clone(),
            message_type,
            is_edited: false,
            created_at: chrono::Utc::now(),
            edited_at: None,
            reply_to,
            status: "sent".to_string(),
            is_deleted: false,
            deleted_at: None,
            mentions: req.mentions.clone().map(|m| m.iter().filter_map(|id| id.parse::<i64>().ok()).collect()),
            file_name: req.file_name.clone(),
            file_size: req.file_size,
            thumbnail_url: req.thumbnail_url.clone(),
            duration: req.duration,
            system_event: None,
            affected_users: None,
            latitude: req.latitude,
            longitude: req.longitude,
            location_name: req.location_name.clone(),
        };

        message_repo.create(&message).await?;

        // 更新会话最后消息时间
        use crate::repositories::chat_room_repo::ChatRoomRepository;
        let chat_room_repo = ChatRoomRepository::new(self.db.clone());
        chat_room_repo.update_last_message_at(room_id, message.created_at).await?;

        // 发出消息发送事件
        self.event_bus.publish(AppEvent::MessageSent {
            message_id,
            room_id,
            sender_id,
            content: req.content.clone(),
            message_type: req.message_type.clone(),
        });

        Ok(message_id.to_string())
    }

    /// 编辑消息
    pub async fn edit_message(&self, user_id: i64, req: &EditMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());

        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能编辑自己的消息
        if message.sender_id != Some(user_id) {
            return Err(AppError::Forbidden("无权编辑该消息".to_string()));
        }

        // 不能编辑已撤回或已删除的消息
        if message.status == "recalled" || message.deleted_at.is_some() {
            return Err(AppError::BadRequest("该消息无法编辑".to_string()));
        }

        message_repo.update_content(message_id, &req.content).await?;

        // 发出消息编辑事件
        self.event_bus.publish(AppEvent::MessageEdited {
            message_id,
            room_id: message.room_id,
            content: req.content.clone(),
            edited_at: chrono::Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 撤回消息
    pub async fn recall_message(&self, user_id: i64, req: &RecallMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());

        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能撤回自己的消息
        if message.sender_id != Some(user_id) {
            return Err(AppError::Forbidden("无权撤回该消息".to_string()));
        }

        // 检查撤回时间限制（2分钟内）
        let now = chrono::Utc::now().naive_utc();
        let elapsed = now.signed_duration_since(message.created_at.naive_utc());
        if elapsed.num_minutes() > 2 {
            return Err(AppError::BadRequest("消息发送超过2分钟，无法撤回".to_string()));
        }

        message_repo.recall(message_id).await?;

        // 发出消息撤回事件
        self.event_bus.publish(AppEvent::MessageRecalled {
            message_id,
            room_id: message.room_id,
            recalled_by: user_id,
            recalled_at: chrono::Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 删除消息
    pub async fn delete_message(&self, user_id: i64, req: &DeleteMessageRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());

        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::NotFound("消息不存在".to_string()))?;

        // 只能删除自己的消息
        if message.sender_id != Some(user_id) {
            return Err(AppError::Forbidden("无权删除该消息".to_string()));
        }

        message_repo.soft_delete(message_id).await?;

        // 发出消息删除事件
        let for_everyone = req.for_everyone.unwrap_or(false);
        self.event_bus.publish(AppEvent::MessageDeleted {
            message_id,
            room_id: message.room_id,
            deleted_by: user_id,
            for_everyone,
        });

        Ok(())
    }

    /// 获取消息历史
    pub async fn list_messages(
        &self,
        user_id: i64,
        req: &ListMessagesRequest,
    ) -> Result<PaginatedResponse<MessageResponse>> {
        let message_repo = MessageRepository::new(self.db.clone());

        let limit = req.limit.unwrap_or(50).min(100);

        let (messages, next_cursor) = match req.target_type.as_str() {
            "user" => {
                let other_user_id = req.target_id.parse::<i64>()
                    .map_err(|_| AppError::BadRequest("无效的目标用户ID".to_string()))?;
                
                message_repo
                    .list_user_messages(user_id, other_user_id, limit, req.cursor.clone())
                    .await?
            }
            "group" => {
                let group_id = Uuid::parse_str(&req.target_id)
                    .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;
                
                // 检查是否为群成员
                let member_repo = GroupMemberRepository::new(self.db.clone());
                if !member_repo.is_member(group_id, user_id).await? {
                    return Err(AppError::Forbidden("您不是该群组成员".to_string()));
                }

                message_repo
                    .list_room_messages(group_id, limit, req.cursor.clone())
                    .await?
            }
            _ => return Err(AppError::BadRequest("无效的目标类型".to_string())),
        };
        let items: Vec<MessageResponse> = messages
            .into_iter()
            .take(limit as usize)
            .map(|m| MessageResponse {
                id: m.message_id.to_string(),
                target_id: m.room_id.to_string(),
                target_type: req.target_type.clone(),
                message_type: m.message_type.as_str().to_string(),
                content: m.content.clone(),
                sender: m.sender_id.map(|id| UserResponse {
                    user_id: id.to_string(),
                    nickname: String::new(), // TODO: 从数据库查询
                    avatar_url: None,
                }),
                reply_to: m.reply_to.map(|id| ReplyInfo {
                    id: id.to_string(),
                    content: String::new(), // TODO: 从数据库查询
                    sender_name: String::new(),
                }),
                mentions: m.mentions.map(|ids| ids.iter().map(|id| id.to_string()).collect()),
                is_edited: m.is_edited,
                edited_at: m.edited_at,
                sent_at: m.created_at,
                reactions: None, // TODO: 查询反应数据
                latitude: m.latitude,
                longitude: m.longitude,
                location_name: m.location_name.clone(),
                file_name: m.file_name.clone(),
                file_size: m.file_size,
                thumbnail_url: m.thumbnail_url.clone(),
                duration: m.duration,
                system_event: None,
                affected_users: None,
            })
            .collect();

        Ok(PaginatedResponse {
            items,
            has_more: next_cursor.is_some(),
            next_cursor,
        })
    }

    /// 标记消息已读
    pub async fn mark_as_read(&self, user_id: i64, req: &MarkAsReadRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());
        let room_id = Uuid::parse_str(&req.target_id)
            .map_err(|_| AppError::BadRequest("无效的会话ID".to_string()))?;

        let up_to_id = if let Some(up_to) = &req.up_to {
            Some(Uuid::parse_str(up_to)
                .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?)
        } else {
            None
        };
        
        message_repo.mark_room_as_read(user_id, room_id, up_to_id).await?;

        // TODO: 更新会话未读计数
        // TODO: 更新Redis缓存

        Ok(())
    }

    /// 添加消息反应
    pub async fn add_reaction(&self, user_id: i64, req: &AddReactionRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());

        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 验证消息存在
        if message_repo.find_by_id(message_id).await?.is_none() {
            return Err(AppError::NotFound("消息不存在".to_string()));
        }

        message_repo.add_reaction(message_id, user_id, &req.reaction_type).await?;

        // TODO: 触发WebSocket推送反应事件

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(&self, user_id: i64, req: &RemoveReactionRequest) -> Result<()> {
        let message_repo = MessageRepository::new(self.db.clone());

        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        message_repo.remove_reaction(message_id, user_id, &req.reaction_type).await?;

        Ok(())
    }

    /// 获取未读消息摘要
    pub async fn get_unread_summary(&self, user_id: i64, include_details: bool) -> Result<serde_json::Value> {
        let read_repo = MessageReadRepository::new(self.db.clone());
        let (total, chats, groups) = read_repo.get_unread_summary(user_id).await?;

        let mut result = serde_json::json!({
            "total": total,
            "chats": chats,
            "groups": groups,
        });

        if include_details {
            let details = read_repo.get_unread_rooms_detail(user_id).await?;
            let details_json: Vec<serde_json::Value> = details.into_iter().map(|d| {
                serde_json::json!({
                    "target_id": d.target_id.to_string(),
                    "target_type": d.target_type,
                    "unread_count": d.unread_count,
                    "last_message_at": d.last_message_at,
                })
            }).collect();
            result["details"] = serde_json::json!(details_json);
        }

        Ok(result)
    }

    /// 标记所有消息已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        let read_repo = MessageReadRepository::new(self.db.clone());
        read_repo.mark_all_read(user_id).await?;
        Ok(())
    }

    /// 获取特定房间未读数
    pub async fn get_unread_count(&self, user_id: i64, room_id: Uuid) -> Result<i32> {
        let read_repo = MessageReadRepository::new(self.db.clone());
        read_repo.get_room_unread_count(room_id, user_id).await
    }

    /// 创建系统消息（辅助方法）
    /// 用于群组成员变动、系统通知等场景
    pub async fn create_system_message(
        &self,
        room_id: Uuid,
        content: String,
        system_event: String,
        affected_users: Option<Vec<i64>>,
    ) -> Result<String> {
        let message_repo = MessageRepository::new(self.db.clone());
        let message_id = Uuid::new_v4();

        let message = Message {
            id: 0,
            message_id,
            room_id,
            sender_id: Some(SYSTEM_USER_ID), // 使用系统用户ID
            content,
            message_type: MessageType::System,
            is_edited: false,
            created_at: chrono::Utc::now(),
            edited_at: None,
            reply_to: None,
            status: "sent".to_string(),
            is_deleted: false,
            deleted_at: None,
            mentions: None,
            latitude: None,
            longitude: None,
            location_name: None,
            file_name: None,
            file_size: None,
            thumbnail_url: None,
            duration: None,
            system_event: Some(system_event),
            affected_users,
        };

        message_repo.create(&message).await?;

        Ok(message_id.to_string())
    }

    /// 发起私聊（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn start_chat(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::StartChatRequest,
    ) -> Result<crate::dto::response::StartChatResponse> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }

    /// 获取私聊列表（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn list_chats(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::ListChatsRequest,
    ) -> Result<crate::dto::response::PaginatedResponse<crate::dto::response::ChatResponse>> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }

    /// 获取私聊设置（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn get_chat_settings(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::GetChatSettingsRequest,
    ) -> Result<crate::dto::response::GetChatSettingsResponse> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }

    /// 更新私聊设置（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn update_chat_settings(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::UpdateChatSettingsRequest,
    ) -> Result<crate::dto::response::UpdateChatSettingsResponse> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }

    /// 删除私聊（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn delete_chat(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::DeleteChatRequest,
    ) -> Result<crate::dto::response::DeleteChatResponse> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }

    /// 升级为群组（应该在 ChatRoomService 中，这里是占位方法）
    pub async fn upgrade_to_group(
        &self,
        _user_id: i64,
        _req: &crate::dto::request::UpgradeToGroupRequest,
    ) -> Result<crate::dto::response::UpgradeToGroupResponse> {
        Err(AppError::NotImplemented("请使用 ChatRoomService".to_string()))
    }
}
