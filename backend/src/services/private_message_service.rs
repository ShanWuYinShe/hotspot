//! 私聊消息服务层实现
//!
//! 适配新 Schema: message_id(UUID), chat_id(UUID), metadata(media)

use std::collections::HashMap;
use std::sync::Arc;

use chrono::{DateTime, Utc};
use tokio::try_join;
use tracing::{debug, info, warn};
use uuid::Uuid;

use crate::constants::MESSAGE_RECALL_MINUTES;
use crate::{
    dto::{
        request::{
            MarkPrivateMessagesReadRequest, PrivateMessagesHistoryRequest,
            RecallPrivateMessageRequest, SendPrivateMessageRequest, UpdatePrivateMessageRequest,
        },
        response::{
            MessageMedia, MessageReplyTo, PaginatedData, PrivateMessageItem,
            SendPrivateMessageData, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::{AppEvent, ReplyToSnapshot},
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        private_chat_repo::{PrivateChatRepository, PrivateChatRow},
        private_message_repo::PrivateMessageRepository,
        user::blocks_repo::UserBlocksRepository,
        user::follows_repo::UserFollowsRepository,
        user::settings_repo::UserSettingsRepository,
        user_repo::UserRepository,
    },
    services::{
        private_chat_permission::PrivateChatPermissionResolver,
        stranger_message_limiter::StrangerMessageLimiter,
    },
    types::enums::{MessageBlockedReason, MessageType},
    utils::validation::Validator,
    EventBus,
};

#[derive(Clone)]
pub struct PrivateMessageService {
    event_bus: Arc<EventBus>,
    chat_repo: PrivateChatRepository,
    message_repo: PrivateMessageRepository,
    unread_repo: ChatUnreadRepository,
    user_repo: UserRepository,
    media_service: crate::services::media_service::MediaService,
    permission_resolver: PrivateChatPermissionResolver,
    stranger_limiter: StrangerMessageLimiter,
}

impl PrivateMessageService {
    fn cleared_at_for_user(chat: &PrivateChatRow, user_id: i64) -> Option<DateTime<Utc>> {
        if chat.user1_id == user_id {
            chat.user1_cleared_at
        } else if chat.user2_id == user_id {
            chat.user2_cleared_at
        } else {
            None
        }
    }

    pub fn new(
        event_bus: Arc<EventBus>,
        message_repo: PrivateMessageRepository,
        chat_repo: PrivateChatRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        blocks_repo: UserBlocksRepository,
        settings_repo: UserSettingsRepository,
        follows_repo: UserFollowsRepository,
        media_service: crate::services::media_service::MediaService,
        stranger_limiter: StrangerMessageLimiter,
    ) -> Self {
        let permission_resolver =
            PrivateChatPermissionResolver::new(settings_repo, follows_repo, blocks_repo);

        Self {
            event_bus,
            chat_repo,
            message_repo,
            unread_repo,
            user_repo,
            media_service,
            permission_resolver,
            stranger_limiter,
        }
    }

    /// 发送私聊消息
    pub async fn send(
        &self,
        req: &SendPrivateMessageRequest,
        user_id: i64,
    ) -> Result<SendPrivateMessageData> {
        let started_at = std::time::Instant::now();
        let target_user_id =
            crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
        info!(
            user_id,
            target_user_id,
            message_type = ?req.r#type,
            has_reply_to = req.reply_to_message_id.is_some(),
            has_file_id = req.file_id.is_some(),
            "private_message.send started"
        );
        if user_id == target_user_id {
            return Err(AppError::CannotOperateOnSelf);
        }

        if let Some(content) = req.content.as_deref() {
            Validator::validate_message_content(content)?;
        }

        // 校验目标用户存在
        let receiver = self
            .user_repo
            .find_by_id(target_user_id)
            .await?
            .ok_or_else(|| AppError::UserNotFound("接收者不存在".to_string()))?;

        let permission = self
            .permission_resolver
            .evaluate(user_id, target_user_id, Some(&receiver))
            .await?;
        if !permission.can_message {
            let err_msg = match permission.reason {
                Some(MessageBlockedReason::AccountSuspended) => "对方账号暂不可用".to_string(),
                _ => "对方暂时不允许接收私聊".to_string(),
            };
            return Err(AppError::Forbidden(err_msg));
        }

        if !permission.is_mutual_friend {
            let allowed = self
                .stranger_limiter
                .increment_if_allowed(user_id, target_user_id)
                .await?;
            if !allowed {
                return Err(AppError::Forbidden(
                    "陌生人消息限额已用完，请稍后再试".to_string(),
                ));
            }
        }

        // 创建/获取会话
        let chat = self
            .chat_repo
            .get_or_create(user_id, target_user_id)
            .await?;
        let chat_id_str = chat.chat_id.to_string();
        if let Err(e) = self
            .unread_repo
            .unhide_private_chat(user_id, &chat_id_str)
            .await
        {
            warn!(
                user_id,
                target_user_id,
                chat_id = %chat.chat_id,
                error = %e,
                "发送私聊前取消隐藏会话失败（发送方）"
            );
        }
        if let Err(e) = self
            .unread_repo
            .unhide_private_chat(target_user_id, &chat_id_str)
            .await
        {
            warn!(
                user_id,
                target_user_id,
                chat_id = %chat.chat_id,
                error = %e,
                "发送私聊前取消隐藏会话失败（接收方）"
            );
        }

        let reply_to_message_id = req
            .reply_to_message_id
            .as_ref()
            .and_then(|id| Uuid::parse_str(id).ok());

        let message_type: MessageType = req.r#type.into();
        let (content, media, metadata) = match message_type {
            MessageType::Text => {
                let content = req
                    .content
                    .clone()
                    .ok_or_else(|| AppError::BadRequest("消息内容不能为空".to_string()))?;
                (content, None, None)
            }
            MessageType::Share => {
                // Share 类型消息使用 share_code
                let share_code = req.share_code.clone().ok_or_else(|| {
                    AppError::BadRequest("分享消息必须提供 share_code".to_string())
                })?;
                (share_code, None, None)
            }
            MessageType::System => {
                return Err(AppError::BadRequest("不允许发送系统消息".to_string()));
            }
            MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File => {
                let file_id = req
                    .file_id
                    .as_ref()
                    .ok_or_else(|| AppError::BadRequest("媒体消息必须提供 file_id".to_string()))?;
                let file_uuid = Validator::parse_uuid(file_id, "file_id")?;

                let expected_categories = match message_type {
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
                target_user_id,
                message_type,
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
                    let (preview, reply_share_code) = match row.r#type {
                        MessageType::Text | MessageType::System => {
                            (row.content.clone().unwrap_or_default(), None)
                        }
                        MessageType::Share => {
                            (row.r#type.preview_text().to_string(), row.content.clone())
                        }
                        _ => (row.r#type.preview_text().to_string(), None),
                    };

                    Some(ReplyToSnapshot {
                        message_id: reply_id,
                        user_id: row.sender_id,
                        nickname: String::new(),
                        content: preview,
                        share_code: reply_share_code,
                    })
                })
        } else {
            None
        };

        let share_code = if matches!(message_type, MessageType::Share) {
            Some(content.clone())
        } else {
            None
        };

        let event_content = if matches!(message_type, MessageType::Text | MessageType::System) {
            req.content.clone()
        } else {
            None
        };

        self.event_bus.publish(AppEvent::PrivateMessageSent {
            message_id: created.message_id,
            sender_id: user_id,
            target_user_id,
            message_type,
            content: event_content,
            share_code,
            reply_to: reply_to_snapshot,
            media,
        });
        info!(
            user_id,
            target_user_id,
            chat_id = %chat.chat_id,
            message_id = %created.message_id,
            message_type = ?message_type,
            elapsed_ms = started_at.elapsed().as_millis(),
            "private_message.send completed"
        );

        Ok(SendPrivateMessageData {
            message_id: created.message_id.to_string(),
            sent_at: created.sent_at.to_rfc3339(),
        })
    }

    /// 获取历史记录
    pub async fn get_history(
        &self,
        req: &PrivateMessagesHistoryRequest,
        user_id: i64,
    ) -> Result<PaginatedData<PrivateMessageItem>> {
        let target_user_id =
            crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
        let chat = match self
            .chat_repo
            .find_by_users(user_id, target_user_id)
            .await?
        {
            Some(c) => c,
            None => {
                return Ok(PaginatedData {
                    items: vec![],
                    next_cursor: None,
                });
            }
        };

        let pagination = req.pagination.as_ref();
        let limit = Validator::validate_limit(
            pagination.and_then(|p| p.limit),
            crate::constants::MAX_PAGE_LIMIT,
        )? as i64;
        let cleared_after = Self::cleared_at_for_user(&chat, user_id);
        let (messages, next_cursor) = self
            .message_repo
            .list_by_chat(
                chat.chat_id,
                limit,
                pagination.and_then(|p| p.cursor.clone()),
                cleared_after,
            )
            .await?;

        let chat_id_str = chat.chat_id.to_string();

        let other_last_read = self
            .unread_repo
            .get_last_read_message_id(target_user_id, &chat_id_str)
            .await?;
        let other_last_read_state: Option<(Uuid, DateTime<Utc>)> =
            if let Some(last_id) = other_last_read {
                self.message_repo
                    .find_by_id(last_id)
                    .await?
                    .map(|row| (last_id, row.sent_at))
            } else {
                None
            };
        let (other_last_read_id, other_last_read_sent_at) = match other_last_read_state {
            Some((id, ts)) => (Some(id), Some(ts)),
            None => (None, None),
        };

        let reply_ids: Vec<Uuid> = messages
            .iter()
            .filter_map(|m| m.reply_to_message_id)
            .collect();
        let reply_rows = self.message_repo.find_by_ids(&reply_ids).await?;
        let reply_sender_ids: Vec<i64> = reply_rows.iter().map(|m| m.sender_id).collect();
        let reply_map: HashMap<Uuid, _> =
            reply_rows.into_iter().map(|m| (m.message_id, m)).collect();

        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let (reply_senders, senders) = try_join!(
            self.user_repo.find_profiles(&reply_sender_ids),
            self.user_repo.find_profiles(&sender_ids)
        )?;
        let reply_sender_map: HashMap<i64, _> =
            reply_senders.into_iter().map(|u| (u.user_id, u)).collect();
        let sender_map: HashMap<i64, _> = senders.into_iter().map(|u| (u.user_id, u)).collect();

        let mut items = Vec::with_capacity(messages.len());
        for m in messages {
            let sender = sender_map.get(&m.sender_id);

            let media = if matches!(
                m.r#type,
                MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File
            ) {
                let storage_path = m.content.clone().unwrap_or_default();
                let signed_url = match self.media_service.sign_url(&storage_path).await {
                    Ok(url) => url,
                    Err(e) => {
                        warn!(
                            error = %e,
                            path = %storage_path,
                            "Failed to sign private message media URL"
                        );
                        storage_path.clone()
                    }
                };
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
                    let (preview, reply_share_code) = match reply_msg.r#type {
                        MessageType::Text | MessageType::System => {
                            (reply_msg.content.clone().unwrap_or_default(), None)
                        }
                        MessageType::Share => (
                            reply_msg.r#type.preview_text().to_string(),
                            reply_msg.content.clone(),
                        ),
                        _ => (reply_msg.r#type.preview_text().to_string(), None),
                    };
                    MessageReplyTo {
                        message_id: reply_msg.message_id.to_string(),
                        user_id: reply_msg.sender_id,
                        nickname: reply_sender
                            .and_then(|u| u.nickname.clone())
                            .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                        content: preview,
                        share_code: reply_share_code,
                    }
                })
            });

            let share_code = if matches!(m.r#type, MessageType::Share) {
                m.content.clone()
            } else {
                None
            };

            let content = if share_code.is_some() || media.is_some() {
                None
            } else {
                m.content.clone()
            };

            let is_read = if m.sender_id != user_id {
                true
            } else if let Some(ref last_ts) = other_last_read_sent_at {
                if m.sent_at < *last_ts {
                    true
                } else if m.sent_at == *last_ts {
                    other_last_read_id
                        .map(|id| id == m.message_id)
                        .unwrap_or(false)
                } else {
                    false
                }
            } else {
                false
            };

            items.push(PrivateMessageItem {
                message_id: m.message_id.to_string(),
                message_type: m.r#type,
                content,
                sender: UserInfo::new(
                    m.sender_id,
                    sender
                        .and_then(|u| u.nickname.clone())
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    self.media_service
                        .sign_url_if_path(sender.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                ),
                reply_to,
                media,
                share_code,
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                is_read,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn update(&self, req: &UpdatePrivateMessageRequest, user_id: i64) -> Result<()> {
        let started_at = std::time::Instant::now();
        Validator::validate_message_content(&req.content)?;
        let message_id = Validator::parse_uuid(&req.message_id, "消息ID")?;
        info!(
            user_id,
            message_id = %message_id,
            content_len = req.content.chars().count(),
            "private_message.update started"
        );

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
            return Err(AppError::BusinessRuleViolation(
                "仅支持编辑文本消息".to_string(),
            ));
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
        info!(
            user_id,
            message_id = %message_id,
            target_user_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "private_message.update completed"
        );

        Ok(())
    }

    pub async fn recall(&self, req: &RecallPrivateMessageRequest, user_id: i64) -> Result<()> {
        let started_at = std::time::Instant::now();
        let message_id = Validator::parse_uuid(&req.message_id, "消息ID")?;
        info!(
            user_id,
            message_id = %message_id,
            "private_message.recall started"
        );

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
        if elapsed.num_minutes() >= MESSAGE_RECALL_MINUTES {
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
        info!(
            user_id,
            message_id = %message_id,
            target_user_id = message.receiver_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "private_message.recall completed"
        );

        Ok(())
    }

    pub async fn mark_as_read(
        &self,
        req: &MarkPrivateMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let target_user_id =
            crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
        let chat = match self
            .chat_repo
            .find_by_users(user_id, target_user_id)
            .await?
        {
            Some(chat) => chat,
            None => {
                warn!(user_id, target_user_id, "私聊标记已读失败：会话不存在");
                return Err(AppError::ChatNotFound("会话不存在".to_string()));
            }
        };

        let up_to = req
            .up_to_message_id
            .as_ref()
            .map(|id| Validator::parse_uuid(id, "消息ID"))
            .transpose()?;

        if let Some(up_to_id) = up_to {
            let message = self
                .message_repo
                .find_by_id(up_to_id)
                .await?
                .ok_or_else(|| {
                    warn!(
                        user_id,
                        target_user_id,
                        up_to_message_id = %up_to_id,
                        "私聊标记已读失败：up_to_message_id 不存在"
                    );
                    AppError::ChatNotFound("消息不存在".to_string())
                })?;
            if message.chat_id != chat.chat_id {
                warn!(
                    user_id,
                    target_user_id,
                    up_to_message_id = %up_to_id,
                    chat_id = %chat.chat_id,
                    message_chat_id = %message.chat_id,
                    "私聊标记已读失败：up_to_message_id 不属于当前会话"
                );
                return Err(AppError::ChatNotFound("消息不属于当前私聊会话".to_string()));
            }
        }

        self.unread_repo
            .mark_as_read("private", &chat.chat_id.to_string(), user_id, up_to)
            .await?;
        debug!(
            user_id,
            target_user_id,
            chat_id = %chat.chat_id,
            up_to_message_id = ?up_to,
            "私聊标记已读成功"
        );

        let up_to_for_event: Option<Uuid> = match up_to {
            Some(id) => Some(id),
            None => self
                .message_repo
                .get_last_message_batch(&[chat.chat_id])
                .await?
                .get(&chat.chat_id)
                .map(|row| row.message_id),
        };

        if let Some(up_to_id) = up_to_for_event {
            self.event_bus.publish(AppEvent::PrivateMessageRead {
                target_user_id,
                reader_id: user_id,
                up_to_message_id: up_to_id,
                read_at: Utc::now().to_rfc3339(),
            });
        }

        Ok(())
    }

    /// 标记当前用户所有私聊会话为已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        self.unread_repo
            .mark_all_as_read("private", user_id)
            .await?;
        Ok(())
    }
}
