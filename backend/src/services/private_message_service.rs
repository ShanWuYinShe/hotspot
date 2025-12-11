// src/services/private_message_service.rs - 私聊消息服务

use chrono::{Duration, Utc};
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::MESSAGE_RECALL_MINUTES,
    dto::{
        request::{
            SendPrivateMessageRequest, GetPrivateMessagesRequest, UpdatePrivateMessageRequest,
            RecallPrivateMessageRequest, DeletePrivateMessageRequest, MarkPrivateMessagesReadRequest,
            AddPrivateMessageReactionRequest, RemovePrivateMessageReactionRequest,
            GetPrivateMessageReactionsListRequest,
        },
        response::{
            SendPrivateMessageData, PrivateMessageItem, PaginatedData, PaginationMeta, UserInfo,
            MessageReactionItem,
        },
    },
    error::{AppError, Result},
    events::events::AppEvent,
    repositories::{
        private_message_repo::PrivateMessageRepository,
        private_chat_repo::PrivateChatRepository,
        user_repo::UserRepository,
        chat_unread_repo::ChatUnreadRepository,
        user_follows_repo::UserFollowsRepository,
        message_reactions_repo::MessageReactionsRepository,
        file_repo::FileRepository,
    },
    types::enums::MessageType,
    utils::validation::Validator,
    EventBus,
};

/// 私聊消息服务
pub struct PrivateMessageService {
    #[allow(dead_code)]
    db: PgPool,
    event_bus: Arc<EventBus>,
    message_repo: PrivateMessageRepository,
    chat_repo: PrivateChatRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    follows_repo: UserFollowsRepository,
    reactions_repo: MessageReactionsRepository,
    file_repo: FileRepository,
}

impl PrivateMessageService {
    pub fn new(db: PgPool, event_bus: Arc<EventBus>) -> Self {
        Self {
            message_repo: PrivateMessageRepository::new(db.clone()),
            chat_repo: PrivateChatRepository::new(db.clone()),
            user_repo: UserRepository::new(db.clone()),
            unread_repo: ChatUnreadRepository::new(db.clone()),
            follows_repo: UserFollowsRepository::new(db.clone()),
            reactions_repo: MessageReactionsRepository::new(db.clone()),
            file_repo: FileRepository::new(db.clone()),
            db,
            event_bus,
        }
    }

    /// 发送私聊消息
    pub async fn send(
        &self,
        req: &SendPrivateMessageRequest,
        user_id: i64,
    ) -> Result<SendPrivateMessageData> {
        // 解析私聊ID
        let private_chat_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

        // 获取私聊信息
        let chat = self.chat_repo.find_by_id(private_chat_id).await?
            .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;

        // 验证用户是否在私聊中
        let other_user_id = if chat.user_a_id == user_id {
            chat.user_b_id
        } else if chat.user_b_id == user_id {
            chat.user_a_id
        } else {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        };

        // 检查陌生人消息限制
        let is_mutual_follow = self.follows_repo.are_mutual_follows(user_id, other_user_id).await?;
        
        if !is_mutual_follow {
            // 获取当前用户的消息限制状态
            let (can_message_freely, sent_count) = self.chat_repo.get_message_limit(private_chat_id, user_id).await?;
            
            if !can_message_freely {
                // 检查是否关注了对方
                let is_following = self.follows_repo.exists(user_id, other_user_id).await?;
                let limit = if is_following { 3 } else { 1 };
                
                if sent_count >= limit {
                    return Err(AppError::MessageLimitReached(
                        "消息数量已达上限，等待对方回复".to_string()
                    ));
                }
            }
        }

        // 解析消息类型
        let message_type = MessageType::from_str(&req.r#type)
            .ok_or_else(|| AppError::BadRequest("无效的消息类型".to_string()))?;

        // 解析回复消息ID
        let reply_to = req.reply_to.as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?;

        // 根据消息类型处理内容和文件
        // - 文本消息：content 必填，file_id 不需要
        // - 媒体消息：file_id 必填，服务端转换为实际 url 存入 content
        let (content, file_name, file_size, thumbnail_url, duration, media_url) = match message_type {
            MessageType::Text => {
                let content = req.content.clone()
                    .ok_or_else(|| AppError::BadRequest("文本消息必须提供 content".to_string()))?;
                (content, None, None, None, None, None)
            }
            MessageType::System => {
                let content = req.content.clone().unwrap_or_default();
                (content, None, None, None, None, None)
            }
            _ => {
                // 媒体消息：需要 file_id
                let file_id_str = req.file_id.as_ref()
                    .ok_or_else(|| AppError::BadRequest("媒体消息必须提供 file_id".to_string()))?;
                let file_id = Uuid::parse_str(file_id_str)
                    .map_err(|_| AppError::BadRequest("无效的 file_id".to_string()))?;
                
                // 验证文件所有权并获取文件信息
                let expected_categories = match message_type {
                    MessageType::Image => vec!["chat_image"],
                    MessageType::Video => vec!["chat_video"],
                    MessageType::Audio => vec!["chat_audio"],
                    MessageType::File => vec!["chat_file"],
                    _ => vec![],
                };
                let file_data = self.file_repo.get_file_for_message(file_id, user_id, &expected_categories).await?;
                
                // 标记文件已使用
                self.file_repo.mark_as_used(file_id).await?;
                
                // 媒体消息的 content 存储实际 url
                (
                    file_data.url.clone(),  // content = media_url
                    None,  // file_name 暂不存储
                    Some(file_data.file_size),
                    file_data.thumbnail_url.clone(),
                    file_data.duration,
                    Some(file_data.url),
                )
            }
        };

        // 创建消息
        let message = self.message_repo.create(
            private_chat_id,
            user_id,
            message_type,
            &content,
            reply_to,
            file_name.as_deref(),
            file_size,
            thumbnail_url.as_deref(),
            duration,
        ).await?;

        // 发送成功后的处理
        if !is_mutual_follow {
            let (can_message_freely, _) = self.chat_repo.get_message_limit(private_chat_id, user_id).await?;
            
            if !can_message_freely {
                // 增加发送消息计数
                self.chat_repo.increment_sent_count(private_chat_id, user_id).await?;
            }
            
            // 解锁对方的消息限制（我回复了对方，解锁对方）
            // 检查这是否是回复（即对方之前发过消息给我）
            let other_has_sent = self.message_repo.has_messages_from_user(private_chat_id, other_user_id).await?;
            if other_has_sent {
                // 对方发过消息给我，我回复了，解锁对方
                self.chat_repo.unlock_messaging(private_chat_id, other_user_id).await?;
            }
        }

        // 发布私聊消息事件
        // 注意：媒体消息的 content 字段存储的是 media_url
        let event_content = if message_type == MessageType::Text {
            req.content.clone()
        } else {
            None
        };
        
        self.event_bus.publish(AppEvent::PrivateMessageSent {
            message_id: message.message_id,
            private_chat_id,
            sender_id: user_id,
            recipient_id: other_user_id,
            message_type: req.r#type.clone(),
            content: event_content,
            reply_to,
            media_url,
            thumbnail_url,
            file_name,
            file_size,
            duration,
            width: None,
            height: None,
        });

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
        let private_chat_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的私聊ID".to_string()))?;

        // 验证用户是否在私聊中
        if !self.chat_repo.is_user_in_chat(private_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

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
                content: Some(m.content),
                sender: UserInfo {
                    user_id: m.sender_id,
                    nickname: user.map(|u| u.nickname.clone()).unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: user.and_then(|u| u.avatar_url.clone()),
                    user_type: user.map(|u| u.user_type).unwrap_or(crate::types::enums::UserType::Normal),
                },
                reply_to: m.reply_to_message_id.map(|id| id.to_string()),
                media_url: None,
                thumbnail_url: None,
                file_name: None,
                file_size: None,
                duration: None,
                width: None,
                height: None,
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

    /// 获取私聊对方ID
    async fn get_other_user_id(&self, private_chat_id: Uuid, user_id: i64) -> Result<i64> {
        let chat = self.chat_repo.find_by_id(private_chat_id).await?
            .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;
        
        if chat.user_a_id == user_id {
            Ok(chat.user_b_id)
        } else if chat.user_b_id == user_id {
            Ok(chat.user_a_id)
        } else {
            Err(AppError::Forbidden("您不是此私聊的成员".to_string()))
        }
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

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.private_chat_id, user_id).await?;

        // 更新消息内容
        self.message_repo.update_content(message_id, &req.content).await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageEdited {
            message_id,
            private_chat_id: message.private_chat_id,
            recipient_id: other_user_id,
            content: req.content.clone(),
            edited_at: Utc::now().to_rfc3339(),
        });

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

        // 验证撤回时间窗口
        let now = Utc::now();
        if now - message.sent_at > Duration::minutes(MESSAGE_RECALL_MINUTES) {
            return Err(AppError::BadRequest(format!("消息发送超过{}分钟，无法撤回", MESSAGE_RECALL_MINUTES)));
        }

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.private_chat_id, user_id).await?;

        // 撤回消息
        self.message_repo.recall(message_id).await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageRecalled {
            message_id,
            private_chat_id: message.private_chat_id,
            recipient_id: other_user_id,
            recalled_by: user_id,
            recalled_at: now.to_rfc3339(),
        });

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

        // 删除消息（仅对自己可见，不发布事件）
        self.message_repo.soft_delete(message_id).await?;

        Ok(())
    }

    /// 标记私聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkPrivateMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let private_chat_id = Uuid::parse_str(&req.id)
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

        // 获取对方ID
        let other_user_id = self.get_other_user_id(private_chat_id, user_id).await?;

        // 更新已读状态
        self.unread_repo.mark_as_read("private_chat", &private_chat_id.to_string(), user_id, up_to).await?;

        // 发布已读事件（通知消息发送者）
        if let Some(up_to_id) = up_to {
            self.event_bus.publish(AppEvent::PrivateMessageRead {
                private_chat_id,
                sender_id: other_user_id,  // 消息发送者
                reader_id: user_id,
                up_to: up_to_id,
                read_at: Utc::now().to_rfc3339(),
            });
        }

        Ok(())
    }

    /// 添加消息反应
    pub async fn add_reaction(
        &self,
        req: &AddPrivateMessageReactionRequest,
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

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.private_chat_id, user_id).await?;

        // 添加反应
        self.reactions_repo.add(message_id, user_id, &req.r#type).await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            private_chat_id: message.private_chat_id,
            other_user_id,
            user_id,
            reaction_type: req.r#type.clone(),
            action: "add".to_string(),
        });

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(
        &self,
        req: &RemovePrivateMessageReactionRequest,
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

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.private_chat_id, user_id).await?;

        // 移除反应
        self.reactions_repo.remove(message_id, user_id, &req.r#type).await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            private_chat_id: message.private_chat_id,
            other_user_id,
            user_id,
            reaction_type: req.r#type.clone(),
            action: "remove".to_string(),
        });

        Ok(())
    }

    /// 获取消息反应列表
    pub async fn get_reactions_list(
        &self,
        req: &GetPrivateMessageReactionsListRequest,
    ) -> Result<PaginatedData<MessageReactionItem>> {
        let message_id = Uuid::parse_str(&req.id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 验证消息存在
        if self.message_repo.find_by_id(message_id).await?.is_none() {
            return Err(AppError::NotFound("消息不存在".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let reaction_type = req.r#type.as_deref();

        let reactions = self.reactions_repo
            .list_by_message(message_id, reaction_type, limit + 1, cursor)
            .await?;

        let has_more = reactions.len() > limit as usize;
        let reactions_to_return: Vec<_> = reactions.into_iter().take(limit as usize).collect();
        let next_cursor = if has_more {
            reactions_to_return.last().map(|r| r.cursor_id.to_string())
        } else {
            None
        };

        // 批量获取用户信息
        let user_ids: Vec<i64> = reactions_to_return.iter().map(|r| r.user_id).collect();
        let users = self.user_repo.find_by_ids(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> = users.into_iter()
            .map(|u| (u.user_id, u))
            .collect();

        let items: Vec<MessageReactionItem> = reactions_to_return.into_iter().filter_map(|r| {
            let user = user_map.get(&r.user_id)?;
            Some(MessageReactionItem {
                user: UserInfo {
                    user_id: user.user_id,
                    nickname: user.nickname.clone(),
                    avatar_url: user.avatar_url.clone(),
                    user_type: user.user_type,
                },
                r#type: r.r#type,
                created_at: r.created_at.to_rfc3339(),
            })
        }).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }
}
