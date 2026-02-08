//! 私聊消息服务层实现
//!
//! 适配新 Schema: message_id(UUID), chat_id(UUID), metadata(media)

use std::collections::HashMap;
use std::sync::Arc;

use chrono::Utc;
use redis::aio::ConnectionManager as RedisConnectionManager;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AddPrivateMessageReactionRequest, DeletePrivateMessageRequest, MarkPrivateMessagesReadRequest,
            PrivateMessageReactionsRequest, PrivateMessagesHistoryRequest, RecallPrivateMessageRequest,
            RemovePrivateMessageReactionRequest, SendPrivateMessageRequest, UpdatePrivateMessageRequest,
        },
        response::{
            MessageMedia, MessageReactionItem, MessageReplyTo, PaginatedData, PaginationMeta,
            PrivateMessageItem, SendPrivateMessageData, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::{AppEvent, ReplyToSnapshot},
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        private_chat_repo::PrivateChatRepository,
        private_message_reactions_repo::PrivateMessageReactionsRepository,
        private_message_repo::PrivateMessageRepository,
        user_blocks_repo::UserBlocksRepository,
        user_follows_repo::UserFollowsRepository,
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    types::enums::{MessageType, ReactionType, UserType},
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct PrivateMessageService {
    event_bus: Arc<EventBus>,
    redis: RedisConnectionManager,
    message_repo: PrivateMessageRepository,
    chat_repo: PrivateChatRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    follows_repo: UserFollowsRepository,
    blocks_repo: UserBlocksRepository,
    settings_repo: UserSettingsRepository,
    reactions_repo: PrivateMessageReactionsRepository,
    media_service: crate::services::media_service::MediaService,
}

impl PrivateMessageService {
    pub fn new(
        event_bus: Arc<EventBus>,
        redis: RedisConnectionManager,
        message_repo: PrivateMessageRepository,
        chat_repo: PrivateChatRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        follows_repo: UserFollowsRepository,
        blocks_repo: UserBlocksRepository,
        settings_repo: UserSettingsRepository,
        reactions_repo: PrivateMessageReactionsRepository,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        Self {
            event_bus,
            redis,
            message_repo,
            chat_repo,
            user_repo,
            unread_repo,
            follows_repo,
            blocks_repo,
            settings_repo,
            reactions_repo,
            media_service,
        }
    }

    /// 发送私聊消息
    pub async fn send(&self, req: &SendPrivateMessageRequest, user_id: i64) -> Result<SendPrivateMessageData> {
        if user_id == req.target_user_id {
            return Err(AppError::CannotOperateOnSelf);
        }

        if let Some(content) = req.content.as_deref() {
            Validator::validate_length(content, "消息内容", 1, 10000)?;
        }

        // 校验目标用户存在
        let _receiver = self
            .user_repo
            .find_by_id(req.target_user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("接收者不存在".to_string()))?;

        // 黑名单检查
        if self
            .blocks_repo
            .is_blocked(req.target_user_id, user_id)
            .await?
        {
            return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
        }

        // 创建/获取会话
        let chat = self.chat_repo.get_or_create(user_id, req.target_user_id).await?;

        let reply_to_message_id = req
            .reply_to_message_id
            .as_ref()
            .and_then(|id| Uuid::parse_str(id).ok());

        let (content, media, metadata) = match req.r#type {
            MessageType::Text | MessageType::System | MessageType::Location => {
                let content = req
                    .content
                    .clone()
                    .ok_or_else(|| AppError::BadRequest("消息内容不能为空".to_string()))?;
                (content, None, None)
            }
            MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File => {
                let file_id = req
                    .file_id
                    .as_ref()
                    .ok_or_else(|| AppError::BadRequest("媒体消息必须提供 file_id".to_string()))?;
                let file_uuid = Uuid::parse_str(file_id)
                    .map_err(|_| AppError::BadRequest("无效的 file_id".to_string()))?;

                let expected_categories = match req.r#type {
                    MessageType::Image => vec!["chat_image"],
                    MessageType::Video => vec!["chat_video"],
                    MessageType::Audio => vec!["chat_audio"],
                    MessageType::File => vec!["chat_file"],
                    _ => vec![],
                };

                let file_data = self
                    .media_service
                    .validate_message_file(file_uuid, user_id, &expected_categories)
                    .await?;
                self.media_service.mark_as_used(file_uuid).await?;

                let signed_url = self
                    .media_service
                    .sign_url(&file_data.storage_path)
                    .await
                    .unwrap_or_else(|_| file_data.storage_path.clone());

                let media = MessageMedia {
                    url: signed_url,
                    thumbnail_url: file_data.thumbnail_url.clone(),
                    file_name: file_data.filename.clone(),
                    file_size: Some(file_data.file_size),
                    duration: file_data.duration.map(|d| d as f64),
                    width: file_data.width,
                    height: file_data.height,
                    mime_type: Some(file_data.mime_type.clone()),
                };

                let meta = serde_json::json!({
                    "thumbnail_url": file_data.thumbnail_url,
                    "file_name": file_data.filename,
                    "size": file_data.file_size,
                    "duration": file_data.duration,
                    "width": file_data.width,
                    "height": file_data.height,
                    "mime_type": file_data.mime_type,
                });

                (file_data.storage_path, Some(media), Some(meta))
            }
        };

        let created = self
            .message_repo
            .create(
                chat.chat_id,
                user_id,
                req.target_user_id,
                req.r#type,
                Some(content.clone()),
                metadata,
                reply_to_message_id,
            )
            .await?;

        let reply_to_snapshot = if let Some(reply_id) = reply_to_message_id {
            self.message_repo
                .find_by_ids(&[reply_id])
                .await?
                .into_iter()
                .next()
                .and_then(|row| {
                    let preview = match row.r#type {
                        MessageType::Text => row.content.unwrap_or_default(),
                        MessageType::Image => "[图片]".to_string(),
                        MessageType::Video => "[视频]".to_string(),
                        MessageType::Audio => "[语音]".to_string(),
                        MessageType::File => "[文件]".to_string(),
                        MessageType::Location => "[位置]".to_string(),
                        MessageType::System => row.content.unwrap_or_default(),
                    };

                    Some(ReplyToSnapshot {
                        message_id: reply_id,
                        user_id: row.sender_id,
                        nickname: String::new(),
                        content: preview,
                    })
                })
        } else {
            None
        };

        let event_content = match req.r#type {
            MessageType::Text | MessageType::Location | MessageType::System => req.content.clone(),
            _ => None,
        };

        self.event_bus.publish(AppEvent::PrivateMessageSent {
            message_id: created.message_id,
            sender_id: user_id,
            target_user_id: req.target_user_id,
            message_type: req.r#type,
            content: event_content,
            reply_to: reply_to_snapshot,
            media,
        });

        Ok(SendPrivateMessageData {
            message_id: created.message_id.to_string(),
        })
    }

    /// 获取历史记录
    pub async fn get_history(
        &self,
        req: &PrivateMessagesHistoryRequest,
        user_id: i64,
    ) -> Result<PaginatedData<PrivateMessageItem>> {
        let chat = self
            .chat_repo
            .find_by_users(user_id, req.target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("会话不存在".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let (messages, next_cursor) = self
            .message_repo
            .list_by_chat(chat.chat_id, user_id, limit, req.cursor.clone())
            .await?;

        let reply_ids: Vec<Uuid> = messages.iter().filter_map(|m| m.reply_to_message_id).collect();
        let reply_rows = self.message_repo.find_by_ids(&reply_ids).await?;
        let reply_sender_ids: Vec<i64> = reply_rows.iter().map(|m| m.sender_id).collect();
        let reply_senders = self.user_repo.find_profiles(&reply_sender_ids).await?;
        let reply_sender_map: HashMap<i64, _> =
            reply_senders.into_iter().map(|u| (u.user_id, u)).collect();
        let reply_map: HashMap<Uuid, _> = reply_rows.into_iter().map(|m| (m.message_id, m)).collect();

        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let senders = self.user_repo.find_profiles(&sender_ids).await?;
        let sender_map: HashMap<i64, _> = senders.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(messages.len());
        for m in messages {
            let sender = sender_map.get(&m.sender_id);

            let media = if matches!(m.r#type, MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File) {
                let path = m.content.clone().unwrap_or_default();
                let signed_url = self
                    .media_service
                    .sign_url(&path)
                    .await
                    .unwrap_or(path);
                Some(MessageMedia {
                    url: signed_url,
                    thumbnail_url: m.thumbnail_url.clone(),
                    file_name: m.file_name.clone(),
                    file_size: m.file_size,
                    duration: m.duration,
                    width: m.width,
                    height: m.height,
                    mime_type: m.mime_type.clone(),
                })
            } else {
                None
            };

            let reply_to = m.reply_to_message_id.and_then(|reply_id| {
                reply_map.get(&reply_id).map(|reply_msg| {
                    let reply_sender = reply_sender_map.get(&reply_msg.sender_id);
                    let preview = if reply_msg.r#type == MessageType::Text {
                        reply_msg.content.clone().unwrap_or_default()
                    } else {
                        String::new()
                    };
                    MessageReplyTo {
                        message_id: reply_msg.message_id.to_string(),
                        user_id: reply_msg.sender_id,
                        nickname: reply_sender
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        content: preview,
                    }
                })
            });

            items.push(PrivateMessageItem {
                message_id: m.message_id.to_string(),
                message_type: m.r#type,
                content: if media.is_some() { None } else { m.content.clone() },
                sender: UserInfo {
                    user_id: m.sender_id,
                    nickname: sender
                        .and_then(|u| u.nickname.clone())
                        .unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(sender.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                    user_type: sender.map(|u| u.user_type).unwrap_or(UserType::Normal),
                },
                reply_to,
                media,
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                is_recalled: m.is_recalled,
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
            });
        }

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn update(&self, req: &UpdatePrivateMessageRequest, user_id: i64) -> Result<()> {
        Validator::validate_length(&req.content, "消息内容", 1, 10000)?;
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权编辑此消息".to_string()));
        }

        if message.is_recalled {
            return Err(AppError::MessageAlreadyRecalled);
        }

        if message.r#type != MessageType::Text {
            return Err(AppError::BadRequest("仅支持编辑文本消息".to_string()));
        }

        self.message_repo
            .update_content(message_id, req.content.clone())
            .await?;

        let target_user_id = if message.sender_id == user_id {
            message.receiver_id
        } else {
            message.sender_id
        };

        self.event_bus.publish(AppEvent::PrivateMessageEdited {
            message_id,
            sender_id: user_id,
            target_user_id,
            content: req.content.clone(),
            edited_at: Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    pub async fn recall(&self, req: &RecallPrivateMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权撤回此消息".to_string()));
        }

        if message.is_recalled {
            return Err(AppError::MessageAlreadyRecalled);
        }

        let elapsed = Utc::now().signed_duration_since(message.sent_at);
        if elapsed.num_minutes() >= 2 {
            return Err(AppError::OperationWindowExpired);
        }

        self.message_repo.recall(message_id).await?;

        self.event_bus.publish(AppEvent::PrivateMessageRecalled {
            message_id,
            sender_id: message.sender_id,
            target_user_id: message.receiver_id,
            recalled_by: user_id,
            recalled_at: Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    pub async fn delete(&self, req: &DeletePrivateMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        self.message_repo.delete_for_user(message_id, user_id).await?;

        let target_user_id = if message.sender_id == user_id {
            message.receiver_id
        } else {
            message.sender_id
        };

        self.event_bus.publish(AppEvent::PrivateMessageDeleted {
            message_id,
            target_user_id,
            deleted_by: user_id,
        });

        Ok(())
    }

    pub async fn mark_as_read(&self, req: &MarkPrivateMessagesReadRequest, user_id: i64) -> Result<()> {
        let chat = self
            .chat_repo
            .find_by_users(user_id, req.target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("会话不存在".to_string()))?;

        let up_to = req
            .up_to_message_id
            .as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        if let Some(up_to_id) = up_to {
            self.unread_repo
                .mark_as_read("private", &chat.chat_id.to_string(), user_id, Some(up_to_id))
                .await?;

            self.event_bus.publish(AppEvent::PrivateMessageRead {
                target_user_id: req.target_user_id,
                reader_id: user_id,
                up_to: up_to_id,
                read_at: Utc::now().to_rfc3339(),
            });
        }

        Ok(())
    }

    pub async fn add_reaction(&self, req: &AddPrivateMessageReactionRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        self.reactions_repo.add(message_id, user_id, req.r#type).await?;

        let target_user_id = if message.sender_id == user_id {
            message.receiver_id
        } else {
            message.sender_id
        };

        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            target_user_id,
            user_id,
            reaction_type: req.r#type,
            action: "add".to_string(),
        });

        Ok(())
    }

    pub async fn remove_reaction(&self, req: &RemovePrivateMessageReactionRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        self.reactions_repo.remove(message_id, user_id).await?;

        let target_user_id = if message.sender_id == user_id {
            message.receiver_id
        } else {
            message.sender_id
        };

        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            target_user_id,
            user_id,
            reaction_type: req.r#type,
            action: "remove".to_string(),
        });

        Ok(())
    }

    pub async fn get_reactions_list(
        &self,
        req: &PrivateMessageReactionsRequest,
        _user_id: i64,
    ) -> Result<PaginatedData<MessageReactionItem>> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let (reactions, next_cursor) = self
            .reactions_repo
            .list_by_message(message_id, req.r#type, limit, req.cursor.clone())
            .await?;

        let user_ids: Vec<i64> = reactions.iter().map(|r| r.user_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: HashMap<i64, _> = users.into_iter().map(|u| (u.user_id, u)).collect();

        let items = reactions
            .into_iter()
            .filter_map(|r| {
                let user = user_map.get(&r.user_id)?;
                Some(MessageReactionItem {
                        user: UserInfo {
                            user_id: user.user_id,
                            nickname: user.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                            avatar_url: user.avatar_url.clone(),
                            user_type: user.user_type,
                        },
                    reaction_type: r.reaction_type,
                    created_at: r.created_at.to_rfc3339(),
                })
            })
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }
}
