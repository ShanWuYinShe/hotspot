// src/services/private_message_service.rs
// 私聊消息Service

use chrono::{Duration, Utc};
use std::str::FromStr; // Add FromStr
use std::sync::Arc;
use uuid::Uuid;

use redis::aio::ConnectionManager as RedisConnectionManager;

use crate::{
    constants::MESSAGE_RECALL_MINUTES,
    dto::{
        request::{
            AddPrivateMessageReactionRequest, DeletePrivateMessageRequest,
            MarkPrivateMessagesReadRequest, PrivateMessageReactionsRequest,
            PrivateMessagesHistoryRequest, RecallPrivateMessageRequest,
            RemovePrivateMessageReactionRequest, SendPrivateMessageRequest,
            UpdatePrivateMessageRequest,
        },
        response::{
            Location, MessageLocation, MessageMedia, MessageReactionItem, MessageReplyTo,
            PaginatedData, PaginationMeta, PrivateMessageItem, SendPrivateMessageData, UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        message_reactions_repo::MessageReactionsRepository,
        private_chat_repo::{PrivateChatRepository, PrivateChatRow},
        private_message_repo::PrivateMessageRepository,
        user_blocks_repo::UserBlocksRepository,
        user_follows_repo::UserFollowsRepository,
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    services::media_service::MediaService,
    types::enums::{PrivateChatPermission, MessageType, ReactionType},
    utils::{bloom_filter::ResourceExistenceChecker, validation::Validator},
    EventBus,
};

/// 私聊消息服务
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
    reactions_repo: MessageReactionsRepository,
    media_service: MediaService,
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
        reactions_repo: MessageReactionsRepository,
        media_service: MediaService,
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

    // 查找或创建私聊并验证权限
    async fn ensure_chat_and_check_permission(
        &self,
        user_id: i64,
        target_user_id: i64,
    ) -> Result<PrivateChatRow> {
        use crate::types::enums::UserType;

        // 1. 查找现有私聊 (无需权限验证，因为已存在)
        if let Some(chat) = self
            .chat_repo
            .find_by_users(user_id, target_user_id)
            .await?
        {
            return Ok(chat);
        }

        // 2. 创建新私聊前进行权限验证

        // 检查发送者的用户类型：管理员和系统账号不受任何限制
        let sender = self.user_repo.find_by_id(user_id).await?;
        let is_privileged_user = sender
            .as_ref()
            .map(|u| matches!(u.user_type, UserType::Admin | UserType::System))
            .unwrap_or(false);

        // 只对普通用户（temp/normal）进行权限检查
        if !is_privileged_user {
            // 检查目标用户是否允许接收私聊
            let target_settings = self.settings_repo.find_by_user_id(target_user_id).await?;
            if let Some(settings) = target_settings {
                if settings.allow_private_chat == PrivateChatPermission::Nobody {
                    return Err(AppError::PrivateChatNotAllowed(
                        "对方不允许接收私聊".to_string(),
                    ));
                } else if settings.allow_private_chat == PrivateChatPermission::Friends {
                    // 检查是否是好友（互相关注）
                    let is_following = self
                        .follows_repo
                        .exists(user_id, target_user_id)
                        .await
                        .unwrap_or(false);
                    let is_followed_by = self
                        .follows_repo
                        .exists(target_user_id, user_id)
                        .await
                        .unwrap_or(false);
                    let is_mutual_follow = is_following && is_followed_by;

                    if !is_mutual_follow {
                        return Err(AppError::PrivateChatNotAllowed(
                            "对方仅允许好友发起私聊".to_string(),
                        ));
                    }
                }
            }

            // 检查当前用户是否被目标用户拉黑
            if self.blocks_repo.is_blocked(target_user_id, user_id).await? {
                return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
            }
        }

        // 创建新私聊
        let chat = self.chat_repo.create(user_id, target_user_id).await?;
        Ok(chat)
    }

    /// 发送私聊消息
    pub async fn send(
        &self,
        req: &SendPrivateMessageRequest,
        user_id: i64,
    ) -> Result<SendPrivateMessageData> {
        use crate::types::enums::UserType;

        let target_user_id = req.target_user_id;
        if user_id == target_user_id {
            return Err(AppError::BadRequest("不能与自己发送私聊消息".to_string()));
        }

        // 查找或创建私聊 (带完整权限检查)
        let chat = self
            .ensure_chat_and_check_permission(user_id, target_user_id)
            .await?;

        // 验证用户是否在私聊中（双重保险）
        let other_user_id = if chat.user_a_id == user_id {
            chat.user_b_id
        } else if chat.user_b_id == user_id {
            chat.user_a_id
        } else {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        };

        // 获取发送者的用户类型
        let sender = self.user_repo.find_by_id(user_id).await?;
        let is_privileged_user = sender
            .as_ref()
            .map(|u| matches!(u.user_type, UserType::Admin | UserType::System))
            .unwrap_or(false);

        // 只对普通用户（temp/normal）检查陌生人消息限制
        if !is_privileged_user {
            let is_mutual_follow = self
                .follows_repo
                .are_mutual_follows(user_id, other_user_id)
                .await?;

            if !is_mutual_follow {
                // 获取当前用户的消息限制状态
                let (can_message_freely, sent_count) =
                    self.chat_repo.get_message_limit(chat.id, user_id).await?;

                if !can_message_freely {
                    // 检查是否关注了对方
                    let is_following = self.follows_repo.exists(user_id, other_user_id).await?;
                    let limit = if is_following { 3 } else { 1 };

                    if sent_count >= limit {
                        return Err(AppError::MessageLimitReached(
                            "消息数量已达上限，等待对方回复".to_string(),
                        ));
                    }
                }
            }
        }

        // 使用 SendMessageType 的转换方法获取 MessageType
        let message_type = req.r#type.to_message_type();

        // 解析回复消息ID
        let reply_to = req
            .reply_to_message_id
            .as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?;

        // 准备回复消息快照 (用于事件推送)
        let mut reply_to_snapshot = None;
        if let Some(reply_id) = reply_to {
            // 尝试获取被回复的消息
            if let Ok(Some(reply_msg)) = self.message_repo.find_by_id(reply_id).await {
                // 获取发送者信息
                // 注意：这里可能会有额外的数据库查询，但为了实时体验是值得的
                // 暂时简单的用 user_repo.find_by_id (如果不暴露 profile 方法)
                // 或者复用 find_profiles
                let sender_profile = self
                    .user_repo
                    .find_profiles(&[reply_msg.sender_id])
                    .await?
                    .into_iter()
                    .next();

                if let Some(sender) = sender_profile {
                    let preview_content = match reply_msg.r#type {
                        MessageType::Text => reply_msg.content,
                        MessageType::Image => "[图片]".to_string(),
                        MessageType::Video => "[视频]".to_string(),
                        MessageType::Audio => "[语音]".to_string(),
                        MessageType::File => "[文件]".to_string(),
                        MessageType::System => reply_msg.content,
                        MessageType::Location => "[位置]".to_string(),
                    };

                    reply_to_snapshot = Some(crate::events::event_types::ReplyToSnapshot {
                        message_id: reply_id,
                        user_id: reply_msg.sender_id,
                        nickname: sender.nickname,
                        content: preview_content,
                    });
                }
            }
        }

        // 根据消息类型处理内容和文件
        let (content, media) = match message_type {
            MessageType::Text => {
                let content = req
                    .content
                    .clone()
                    .ok_or_else(|| AppError::BadRequest("文本消息必须提供 content".to_string()))?;
                (content, None)
            }
            MessageType::System => {
                let content = req.content.clone().unwrap_or_default();
                (content, None)
            }
            _ => {
                // 媒体消息：需要 file_id
                let file_id_str = req
                    .file_id
                    .as_ref()
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
                let file_data = self
                    .media_service
                    .validate_message_file(file_id, user_id, &expected_categories)
                    .await?;

                // 标记文件已使用
                self.media_service.mark_as_used(file_id).await?;

                // 为事件和响应生成签名 URL
                let signed_url = match self.media_service.sign_url(&file_data.storage_path).await {
                    Ok(u) => u,
                    Err(e) => {
                        tracing::warn!(
                            "Failed to sign url for path {}: {}",
                            file_data.storage_path,
                            e
                        );
                        file_data.storage_path.clone()
                    }
                };

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

                // 媒体消息的 content 存储实际 storage_path (Key)
                (file_data.storage_path, Some(media))
            }
        };

        // 创建消息
        let message = self
            .message_repo
            .create(
                chat.id,
                user_id,
                message_type,
                &content,
                reply_to,
                media.as_ref().and_then(|m| m.file_name.as_deref()),
                media.as_ref().and_then(|m| m.file_size),
                media.as_ref().and_then(|m| m.thumbnail_url.as_deref()),
                media.as_ref().and_then(|m| m.duration.map(|d| d as i32)),
                req.location
                    .as_ref()
                    .map(|msg_loc| msg_loc.location.latitude),
                req.location
                    .as_ref()
                    .map(|msg_loc| msg_loc.location.longitude),
                req.location
                    .as_ref()
                    .map(|msg_loc| msg_loc.location.location_name.as_str()),
                req.location
                    .as_ref()
                    .and_then(|msg_loc| msg_loc.address.as_deref()),
            )
            .await?;

        // 记录消息到布隆过滤器
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_message(&message.message_id.to_string())
            .await
        {
            tracing::warn!(message_id = %message.message_id, error = %e, "record_message bloom failed");
        }

        // 发送成功后的处理
        // 只对普通用户（temp/normal）进行消息计数管理
        if !is_privileged_user {
            let is_mutual_follow = self
                .follows_repo
                .are_mutual_follows(user_id, other_user_id)
                .await?;

            if !is_mutual_follow {
                let (can_message_freely, _) =
                    self.chat_repo.get_message_limit(chat.id, user_id).await?;

                if !can_message_freely {
                    // 增加发送消息计数
                    self.chat_repo
                        .increment_sent_count(chat.id, user_id)
                        .await?;
                }
            }
        }

        // 无论发送者是谁，只要回复了对方的消息，就解锁对方的消息限制
        // （即使是管理员回复普通用户，也应该解锁对方的限制）
        let other_has_sent = self
            .message_repo
            .has_messages_from_user(chat.id, other_user_id)
            .await?;
        if other_has_sent {
            // 对方发过消息给我，我回复了，解锁对方
            self.chat_repo
                .unlock_user_messaging_limit(chat.id, other_user_id)
                .await?;
        }

        // 发布私聊消息事件
        let event_content = if message_type == MessageType::Text {
            req.content.clone()
        } else {
            None
        };

        let event_location = req.location.as_ref().map(|loc| crate::dto::response::MessageLocation {
            location: crate::dto::response::Location {
                latitude: loc.location.latitude,
                longitude: loc.location.longitude,
                location_name: loc.location.location_name.clone(),
            },
            address: loc.address.clone(),
        });

        self.event_bus.publish(AppEvent::PrivateMessageSent {
            message_id: message.message_id,
            sender_id: user_id,
            target_user_id: other_user_id, // 接收者
            message_type,    // Use parsed enum
            content: event_content,
            reply_to: reply_to_snapshot,
            media,
            location: event_location,
        });

        Ok(SendPrivateMessageData {
            message_id: message.message_id.to_string(),
        })
    }

    /// 获取私聊消息历史
    pub async fn get_history(
        &self,
        req: &PrivateMessagesHistoryRequest,
        user_id: i64,
    ) -> Result<PaginatedData<PrivateMessageItem>> {
        let target_user_id = req.target_user_id;

        // 查找私聊
        let chat = self
            .chat_repo
            .find_by_users(user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        // 获取消息列表
        let (messages, next_cursor) = self
            .message_repo
            .list_by_chat(chat.id, limit, cursor)
            .await?;

        let reply_ids: Vec<Uuid> = messages
            .iter()
            .filter_map(|m| m.reply_to_message_id)
            .collect();
        let reply_rows = self.message_repo.find_by_ids(&reply_ids).await?;
        let reply_sender_ids: Vec<i64> = reply_rows.iter().map(|m| m.sender_id).collect();
        let reply_senders = self.user_repo.find_profiles(&reply_sender_ids).await?;
        let reply_sender_map: std::collections::HashMap<i64, _> =
            reply_senders.into_iter().map(|u| (u.user_id, u)).collect();
        let reply_map: std::collections::HashMap<Uuid, _> =
            reply_rows.into_iter().map(|m| (m.message_id, m)).collect();

        // 批量获取发送者信息
        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let users = self.user_repo.find_profiles(&sender_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 构建响应
        let mut items = Vec::with_capacity(messages.len());
        for m in messages {
            let user = user_map.get(&m.sender_id);

            let media = if m.r#type.is_media() {
                let signed_url = match self.media_service.sign_url(&m.content).await {
                    Ok(u) => u,
                    Err(e) => {
                        tracing::warn!("Failed to sign url for path {}: {}", m.content, e);
                        m.content.clone()
                    }
                };

                // 如果有 thumbnail_url，也需要签名
                let thumbnail_url = if let Some(ref thumb) = m.thumbnail_url {
                    self.media_service.sign_url_if_path(Some(thumb)).await
                } else {
                    None
                };

                Some(MessageMedia {
                    url: signed_url,
                    thumbnail_url,
                    file_name: m.file_name.clone(),
                    file_size: m.file_size,
                    duration: m.duration.map(|d| d as f64),
                    width: None,
                    height: None,
                    mime_type: None,
                })
            } else {
                None
            };

            items.push(PrivateMessageItem {
                message_id: m.message_id.to_string(),
                message_type: m.r#type,
                content: Some(m.content.clone()),
                sender: UserInfo {
                    user_id: m.sender_id,
                    nickname: user
                        .map(|u| u.nickname.clone())
                        .unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(user.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                    user_type: user
                        .map(|u| u.user_type)
                        .unwrap_or(crate::types::enums::UserType::Normal),
                },
                media,
                location: if let (Some(lat), Some(lng), Some(name)) = (
                    m.location_latitude,
                    m.location_longitude,
                    m.location_name.clone(),
                ) {
                    Some(MessageLocation {
                        location: Location {
                            latitude: lat,
                            longitude: lng,
                            location_name: name,
                        },
                        address: m.location_address.clone(),
                    })
                } else {
                    None
                },
                reply_to: m.reply_to_message_id.and_then(|reply_id| {
                    reply_map.get(&reply_id).map(|reply_msg| {
                        let reply_sender = reply_sender_map.get(&reply_msg.sender_id);
                        let preview = if reply_msg.r#type == MessageType::Text {
                            reply_msg.content.clone()
                        } else {
                            String::new()
                        };
                        MessageReplyTo {
                            message_id: reply_msg.message_id.to_string(),
                            user_id: reply_msg.sender_id,
                            nickname: reply_sender
                                .map(|u| u.nickname.clone())
                                .unwrap_or_else(|| "未知用户".to_string()),
                            content: preview,
                        }
                    })
                }),
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

    /// 获取私聊对方ID
    async fn get_other_user_id(&self, chat_id: i64, user_id: i64) -> Result<i64> {
        let chat = self
            .chat_repo
            .find_by_id(chat_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        if chat.user_a_id == user_id {
            Ok(chat.user_b_id)
        } else if chat.user_b_id == user_id {
            Ok(chat.user_a_id)
        } else {
            Err(AppError::Forbidden("您不是此私聊的成员".to_string()))
        }
    }

    /// 编辑私聊消息
    pub async fn update(&self, req: &UpdatePrivateMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证是否是发送者
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("只能编辑自己发送的消息".to_string()));
        }

        // 验证消息未被撤回
        if message.is_recalled {
            return Err(AppError::MessageAlreadyRecalled);
        }

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.chat_id, user_id).await?;

        // 更新消息内容
        self.message_repo
            .update_content(message_id, &req.content)
            .await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageEdited {
            message_id,
            sender_id: user_id,
            target_user_id: other_user_id,
            content: req.content.clone(),
            edited_at: Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 撤回私聊消息
    pub async fn recall(&self, req: &RecallPrivateMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证是否是发送者
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("只能撤回自己发送的消息".to_string()));
        }

        // 验证消息未被撤回
        if message.is_recalled {
            return Err(AppError::MessageAlreadyRecalled);
        }

        // 验证撤回时间窗口
        let now = Utc::now();
        if now - message.sent_at > Duration::minutes(MESSAGE_RECALL_MINUTES) {
            return Err(AppError::OperationWindowExpired);
        }

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.chat_id, user_id).await?;

        // 撤回消息
        self.message_repo.recall(message_id).await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageRecalled {
            message_id,
            sender_id: user_id,
            target_user_id: other_user_id,
            recalled_by: user_id,
            recalled_at: now.to_rfc3339(),
        });

        Ok(())
    }

    /// 删除私聊消息
    pub async fn delete(&self, req: &DeletePrivateMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否在私聊中
        if !self
            .chat_repo
            .is_user_in_chat(message.chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 获取对方ID（用于事件，虽然此接口不发事件，但保持一致性）
        let other_user_id = self.get_other_user_id(message.chat_id, user_id).await?;

        // 删除消息（仅对自己可见，不发布事件）
        self.message_repo.soft_delete(message_id).await?;

        // 这里原逻辑不发事件，但 Spec 现在的事件定义包含 PrivateMessageDeleted
        // 如果要支持多端同步删除，应该发事件。
        // 但 API Spec 对于删除接口说明：
        // "本操作仅影响当前用户视角" -> 通常意味着软删除或标记删除，且不通知对方。
        // 但多端同步（我自己的其他设备）需要知道。
        // 所以我们还是发一个事件，但 target_user_id 应该怎么填？
        // 如果是仅对自己删除，target_user_id 是否重要？
        // 我们的事件定义 `PrivateMessageDeleted` 包含 `target_user_id`。
        // 为了支持多端同步，应该填 `other_user_id`，这样客户端能定位到会话。

        self.event_bus.publish(AppEvent::PrivateMessageDeleted {
            message_id,
            target_user_id: other_user_id,
            deleted_by: user_id,
        });

        Ok(())
    }

    /// 标记私聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkPrivateMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let target_user_id = req.target_user_id;

        // 查找私聊
        let chat = self
            .chat_repo
            .find_by_users(user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        // 解析截止消息ID
        let up_to = req
            .up_to_message_id
            .as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 更新已读状态
        self.unread_repo
            .mark_as_read("private_chat", &chat.id.to_string(), user_id, up_to)
            .await?;

        // 发布已读事件（通知消息发送者）
        if let Some(up_to_id) = up_to {
            self.event_bus.publish(AppEvent::PrivateMessageRead {
                target_user_id, // 被读消息的所属会话对方ID (即发送者)
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
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self
            .message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否在私聊中
        if !self
            .chat_repo
            .is_user_in_chat(message.chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.chat_id, user_id).await?;

        // 添加反应 - r#type 已是强类型枚举
        self.reactions_repo
            .add(message_id, user_id, req.r#type.as_str())
            .await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            target_user_id: other_user_id,
            user_id,
            reaction_type: req.r#type,
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
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self
            .message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否在私聊中
        if !self
            .chat_repo
            .is_user_in_chat(message.chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        // 获取对方ID
        let other_user_id = self.get_other_user_id(message.chat_id, user_id).await?;

        // 移除反应 - r#type 已是强类型枚举
        self.reactions_repo
            .remove(message_id, user_id, req.r#type.as_str())
            .await?;

        // 发布事件
        self.event_bus.publish(AppEvent::PrivateMessageReaction {
            message_id,
            target_user_id: other_user_id,
            user_id,
            reaction_type: req.r#type,
            action: "remove".to_string(),
        });

        Ok(())
    }

    /// 获取消息反应列表
    pub async fn get_reactions_list(
        &self,
        req: &PrivateMessageReactionsRequest,
        user_id: i64,
    ) -> Result<PaginatedData<MessageReactionItem>> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息
        let message = self
            .message_repo
            .find_by_id(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否在私聊中
        if !self
            .chat_repo
            .is_user_in_chat(message.chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此私聊的成员".to_string()));
        }

        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let reaction_type = req.r#type.as_ref().map(|r| r.as_str());

        let (reactions, next_cursor) = self
            .reactions_repo
            .list_by_message(message_id, reaction_type, limit, cursor)
            .await?;

        // 批量获取用户信息
        let user_ids: Vec<i64> = reactions.iter().map(|r| r.user_id).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let items: Vec<MessageReactionItem> = reactions
            .into_iter()
            .filter_map(|r| {
                let user = user_map.get(&r.user_id)?;
                Some(MessageReactionItem {
                    user: UserInfo {
                        user_id: user.user_id,
                        nickname: user.nickname.clone(),
                        avatar_url: user.avatar_url.clone(),
                        user_type: user.user_type,
                    },
                    reaction_type: ReactionType::from_str(&r.r#type).unwrap_or(ReactionType::Like),
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
