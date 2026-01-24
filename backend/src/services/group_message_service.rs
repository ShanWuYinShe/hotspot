// src/services/group_message_service.rs
// 群聊消息Service

use sqlx::PgPool;
use std::str::FromStr; // Add FromStr
use std::sync::Arc;
use uuid::Uuid;

use redis::aio::ConnectionManager as RedisConnectionManager;

use crate::{
    dto::{
        request::{
            AddGroupMessageReactionRequest, DeleteGroupMessageRequest,
            GroupMessageReactionsRequest, GroupMessagesHistoryRequest,
            MarkGroupMessagesReadRequest, MessageReadStatusRequest, RecallGroupMessageRequest,
            RemoveGroupMessageReactionRequest, SendGroupMessageRequest, UpdateGroupMessageRequest,
        },
        response::{
            GroupMessageItem, Location, MessageLocation, MessageMedia, MessageReactionItem,
            MessageReadItem, MessageReplyTo, PaginatedData, PaginationMeta, SendGroupMessageData,
            UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository, group_member_repo::GroupMemberRepository,
        group_message_repo::GroupMessageRepository,
        message_reactions_repo::MessageReactionsRepository, user_repo::UserRepository,
    },
    services::media_service::MediaService,
    types::enums::{GroupRole, MessageType, ReactionType, UserType},
    utils::{bloom_filter::ResourceExistenceChecker, validation::Validator},
    EventBus,
};

#[derive(Clone)]
pub struct GroupMessageService {
    db: PgPool,
    redis: RedisConnectionManager,
    event_bus: Arc<EventBus>,
    message_repo: GroupMessageRepository,
    member_repo: GroupMemberRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    reactions_repo: MessageReactionsRepository,
    media_service: MediaService,
}

impl GroupMessageService {
    pub fn new(
        db: PgPool,
        redis: RedisConnectionManager,
        event_bus: Arc<EventBus>,
        message_repo: GroupMessageRepository,
        member_repo: GroupMemberRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        reactions_repo: MessageReactionsRepository,
        media_service: MediaService,
    ) -> Self {
        Self {
            db,
            redis,
            event_bus,
            message_repo,
            member_repo,
            user_repo,
            unread_repo,
            reactions_repo,
            media_service,
        }
    }

    /// 发送群聊消息
    pub async fn send(
        &self,
        req: &SendGroupMessageRequest,
        user_id: i64,
    ) -> Result<SendGroupMessageData> {
        // 1. 参数验证
        if let Some(ref content) = req.content {
            Validator::validate_length(content, "消息内容", 1, 10000)?;
        }

        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 使用 SendMessageType 的转换方法获取 MessageType
        let message_type = req.r#type.to_message_type();

        let reply_to_message_id = if let Some(ref id) = req.reply_to_message_id {
            Some(
                Uuid::parse_str(id)
                    .map_err(|_| AppError::BadRequest("无效的回复消息ID".to_string()))?,
            )
        } else {
            None
        };

        // 准备回复消息快照 (用于事件推送)
        let mut reply_to_snapshot = None;
        if let Some(reply_id) = reply_to_message_id {
            // 尝试获取被回复的消息
            if let Ok(Some(reply_msg)) = self.message_repo.find_by_id(reply_id).await {
                // 获取发送者信息
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
                        _ => "[未知消息]".to_string(),
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

        // mentions 已经是 Option<Vec<i64>>，直接使用
        let mentions = req.mentions.clone();

        // 2. 验证用户是否是群成员
        let member = self
            .member_repo
            .find_member(group_chat_id, user_id)
            .await?
            .ok_or_else(|| AppError::Forbidden("您不是此群组的成员".to_string()))?;

        // 检查是否被禁言
        if let Some(muted_until) = member.muted_until {
            if muted_until > chrono::Utc::now() {
                return Err(AppError::Forbidden("您已被禁言，无法发送消息".to_string()));
            }
        }

        // 3. 根据消息类型处理内容和文件
        // - 文本消息：content 必填，file_id 不需要
        // - 媒体消息：file_id 必填，服务端转换为实际 url 存入 content
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

        // 4. 保存到数据库
        let created = self
            .message_repo
            .create(
                group_chat_id,
                user_id,
                message_type,
                content.clone(),
                reply_to_message_id,
                mentions.clone(),
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

        // 5. 更新群聊最后活跃时间（通过更新updated_at字段）
        sqlx::query!(
            r#"UPDATE group_chats SET updated_at = NOW() WHERE group_chat_id = $1"#,
            group_chat_id
        )
        .execute(&self.db)
        .await?;

        // 记录消息到布隆过滤器
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_message(&created.message_id.to_string())
            .await
        {
            tracing::warn!(message_id = %created.message_id, error = %e, "record_message bloom failed");
        }

        // 6. 发布消息事件（用于实时推送）
        // 注意：媒体消息的 content 字段存储的是 media_url
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

        self.event_bus.publish(AppEvent::GroupMessageSent {
            message_id: created.message_id,
            group_chat_id,
            sender_id: user_id,
            message_type: message_type, // Use parsed enum
            content: event_content,
            reply_to: reply_to_snapshot,
            mentions,
            media,
            location: event_location,
        });

        Ok(SendGroupMessageData {
            message_id: created.message_id.to_string(),
        })
    }

    /// 获取群聊消息历史
    pub async fn get_history(
        &self,
        req: &GroupMessagesHistoryRequest,
        user_id: i64,
    ) -> Result<PaginatedData<GroupMessageItem>> {
        // 1. 参数验证
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 2. 验证用户是否是群成员
        if !self.member_repo.is_member(group_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 3. 解析游标
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let include_system = req.include_system.unwrap_or(false);

        // 4. 查询消息历史
        let (messages, next_cursor) = self
            .message_repo
            .list_by_group(group_chat_id, limit, cursor, include_system)
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

        let message_ids: Vec<Uuid> = messages.iter().map(|m| m.message_id).collect();
        let read_counts = self.message_repo.get_read_counts(&message_ids).await?;

        // 5. 批量获取发送者信息
        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let users = self.user_repo.find_profiles(&sender_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 6. 转换为DTO
        let mut items = Vec::with_capacity(messages.len());
        for m in messages {
            let user = user_map.get(&m.sender_id);

            // Construct MessageMedia if it's a media message
            let media = if m.r#type != MessageType::Text && m.r#type != MessageType::System {
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
                    url: signed_url, // For media messages, content stores the URL
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

            items.push(GroupMessageItem {
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
                    user_type: user.map(|u| u.user_type).unwrap_or(UserType::Normal),
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
                mentions: m.mentions.clone(),
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
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                is_recalled: m.is_recalled,
                recalled_by: m.recalled_by,
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                read_count: read_counts.get(&m.message_id).map(|c| *c as i32),
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

    /// 编辑群聊消息
    pub async fn update(&self, req: &UpdateGroupMessageRequest, user_id: i64) -> Result<()> {
        // 1. 参数验证
        Validator::validate_length(&req.content, "消息内容", 1, 10000)?;
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 2. 查询消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 3. 验证权限（只能编辑自己的消息）
        if message.sender_id != user_id {
            return Err(AppError::Forbidden("无权编辑此消息".to_string()));
        }

        // 4. 验证消息状态
        if message.is_recalled {
            return Err(AppError::BadRequest("已撤回的消息无法编辑".to_string()));
        }

        // 5. 更新消息
        self.message_repo
            .update_content(message_id, req.content.clone())
            .await?;

        // 6. 发布编辑事件
        self.event_bus.publish(AppEvent::GroupMessageEdited {
            message_id,
            group_chat_id: message.group_chat_id,
            content: req.content.clone(),
            edited_at: chrono::Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 撤回群聊消息
    pub async fn recall(&self, req: &RecallGroupMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 1. 查询消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 2. 验证权限
        let can_recall = if message.sender_id == user_id {
            // 发送者可以撤回自己的消息（2分钟内）
            let now = chrono::Utc::now();
            let elapsed = now.signed_duration_since(message.sent_at);
            elapsed.num_minutes() < 2
        } else {
            // 检查是否是管理员/群主
            let role = self
                .member_repo
                .get_member_role(message.group_chat_id, user_id)
                .await?;
            role == Some(GroupRole::Owner) || role == Some(GroupRole::Admin)
        };

        if !can_recall {
            return Err(AppError::Forbidden("无权撤回此消息".to_string()));
        }

        // 3. 验证消息状态
        if message.is_recalled {
            return Err(AppError::BadRequest("消息已被撤回".to_string()));
        }

        // 4. 撤回消息
        self.message_repo.recall(message_id, user_id).await?;

        // 5. 发布撤回事件
        self.event_bus.publish(AppEvent::GroupMessageRecalled {
            message_id,
            group_chat_id: message.group_chat_id,
            recalled_by: user_id,
            recalled_at: chrono::Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 删除群聊消息
    pub async fn delete(&self, req: &DeleteGroupMessageRequest, user_id: i64) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 1. 查询消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 2. 验证权限
        let for_everyone = req.for_everyone.unwrap_or(false);

        if for_everyone {
            // 为所有人删除（仅管理员/群主）
            let role = self
                .member_repo
                .get_member_role(message.group_chat_id, user_id)
                .await?;
            let is_admin = role == Some(GroupRole::Owner) || role == Some(GroupRole::Admin);
            if !is_admin {
                return Err(AppError::Forbidden("无权为所有人删除消息".to_string()));
            }
        } else {
            // 仅为自己删除（需要是群成员）
            if !self
                .member_repo
                .is_member(message.group_chat_id, user_id)
                .await?
            {
                return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
            }
        }

        // 3. 删除消息
        self.message_repo.soft_delete(message_id).await?;

        // 4. 发布删除事件
        self.event_bus.publish(AppEvent::GroupMessageDeleted {
            message_id,
            group_chat_id: message.group_chat_id,
            deleted_by: user_id,
            for_everyone,
        });

        Ok(())
    }

    /// 标记群聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkGroupMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let group_chat_id = Uuid::parse_str(&req.group_chat_id)
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证用户是否是群成员
        if !self.member_repo.is_member(group_chat_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 解析截止消息ID
        let up_to = req
            .up_to_message_id
            .as_ref()
            .map(|id| Uuid::parse_str(id))
            .transpose()
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 更新已读状态
        self.unread_repo
            .mark_as_read("group_chat", &group_chat_id.to_string(), user_id, up_to)
            .await?;

        Ok(())
    }

    /// 获取消息已读状态
    pub async fn get_read_status(
        &self,
        req: &MessageReadStatusRequest,
        user_id: i64,
    ) -> Result<PaginatedData<MessageReadItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if !self
            .member_repo
            .is_member(message.group_chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 查询已读状态
        let (read_status_list, next_cursor) = self
            .message_repo
            .get_read_status(message_id, limit as i64, cursor)
            .await?;

        // 批量获取用户信息
        let user_ids: Vec<i64> = read_status_list.iter().map(|(uid, _)| *uid).collect();
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 转换为DTO
        let items: Vec<MessageReadItem> = read_status_list
            .iter()
            .map(|(uid, read_at)| {
                let user = user_map.get(uid);
                MessageReadItem {
                    user: UserInfo {
                        user_id: *uid,
                        nickname: user
                            .map(|u| u.nickname.clone())
                            .unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: user.and_then(|u| u.avatar_url.clone()),
                        user_type: user.map(|u| u.user_type).unwrap_or(UserType::Normal),
                    },
                    read_at: read_at.to_rfc3339(),
                }
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

    /// 添加消息反应
    pub async fn add_reaction(
        &self,
        req: &AddGroupMessageReactionRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if !self
            .member_repo
            .is_member(message.group_chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 添加反应 - r#type 已是强类型枚举
        self.reactions_repo
            .add(message_id, user_id, req.r#type.as_str())
            .await?;

        self.event_bus.publish(AppEvent::GroupMessageReaction {
            message_id,
            group_chat_id: message.group_chat_id,
            user_id,
            reaction_type: req.r#type,
            action: "add".to_string(),
        });

        Ok(())
    }

    /// 移除消息反应
    pub async fn remove_reaction(
        &self,
        req: &RemoveGroupMessageReactionRequest,
        user_id: i64,
    ) -> Result<()> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if !self
            .member_repo
            .is_member(message.group_chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 移除反应 - r#type 已是强类型枚举
        self.reactions_repo
            .remove(message_id, user_id, req.r#type.as_str())
            .await?;

        self.event_bus.publish(AppEvent::GroupMessageReaction {
            message_id,
            group_chat_id: message.group_chat_id,
            user_id,
            reaction_type: req.r#type,
            action: "remove".to_string(),
        });

        Ok(())
    }

    /// 获取消息反应列表
    pub async fn get_reactions_list(
        &self,
        req: &GroupMessageReactionsRequest,
        user_id: i64,
    ) -> Result<PaginatedData<MessageReactionItem>> {
        let message_id = Uuid::parse_str(&req.message_id)
            .map_err(|_| AppError::BadRequest("无效的消息ID".to_string()))?;

        // 获取消息元数据
        let message = self
            .message_repo
            .get_meta(message_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("消息不存在".to_string()))?;

        // 验证用户是否是群成员
        if !self
            .member_repo
            .is_member(message.group_chat_id, user_id)
            .await?
        {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
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
