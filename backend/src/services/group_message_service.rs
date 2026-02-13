// src/services/group_message_service.rs
// 群聊消息Service

use std::sync::Arc;
use uuid::Uuid;

use redis::aio::ConnectionManager as RedisConnectionManager;

use crate::constants::MESSAGE_RECALL_MINUTES;
use crate::{
    dto::{
        request::{
            GroupMessagesHistoryRequest, MarkGroupMessagesReadRequest, RecallGroupMessageRequest,
            SendGroupMessageRequest, UpdateGroupMessageRequest,
        },
        response::{
            GroupMessageItem, MessageMedia, MessageReplyTo, PaginatedData, SendGroupMessageData,
            UserInfo,
        },
    },
    error::{AppError, Result},
    events::event_types::AppEvent,
    repositories::{
        chat_unread_repo::ChatUnreadRepository, group_member_repo::GroupMemberRepository,
        group_message_repo::GroupMessageRepository, group_repo::GroupRepository,
        user_repo::UserRepository,
    },
    services::media_service::MediaService,
    types::enums::{GroupRole, MessageType},
    utils::{bloom_filter::ResourceExistenceChecker, validation::Validator},
    EventBus,
};

#[derive(Clone)]
pub struct GroupMessageService {
    redis: RedisConnectionManager,
    event_bus: Arc<EventBus>,
    message_repo: GroupMessageRepository,
    member_repo: GroupMemberRepository,
    group_repo: GroupRepository,
    user_repo: UserRepository,
    unread_repo: ChatUnreadRepository,
    media_service: MediaService,
}

impl GroupMessageService {
    pub fn new(
        redis: RedisConnectionManager,
        event_bus: Arc<EventBus>,
        message_repo: GroupMessageRepository,
        member_repo: GroupMemberRepository,
        group_repo: GroupRepository,
        user_repo: UserRepository,
        unread_repo: ChatUnreadRepository,
        media_service: MediaService,
    ) -> Self {
        Self {
            redis,
            event_bus,
            message_repo,
            member_repo,
            group_repo,
            user_repo,
            unread_repo,
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

        let group_id = req
            .group_chat_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证群组是否存在
        if !self.group_repo.exists(group_id).await? {
            return Err(AppError::GroupNotFound("群组不存在".to_string()));
        }

        let message_type = req.r#type;

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
                        MessageType::Text => reply_msg.content.clone().unwrap_or_default(),
                        MessageType::Image => "[图片]".to_string(),
                        MessageType::Video => "[视频]".to_string(),
                        MessageType::Audio => "[语音]".to_string(),
                        MessageType::File => "[文件]".to_string(),
                        MessageType::Location => "[位置]".to_string(),
                        MessageType::System => reply_msg.content.clone().unwrap_or_default(),
                    };

                    reply_to_snapshot = Some(crate::events::event_types::ReplyToSnapshot {
                        message_id: reply_id,
                        user_id: reply_msg.sender_id,
                        nickname: sender.nickname.unwrap_or_else(|| "未知用户".to_string()),
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
            .find_member(group_id, user_id)
            .await?
            .ok_or_else(|| AppError::Forbidden("您不是此群组的成员".to_string()))?;

        // 检查是否被禁言
        if let Some(muted_until) = member.muted_until {
            if muted_until > chrono::Utc::now() {
                return Err(AppError::UserMuted);
            }
        }

        // 3. 根据消息类型处理内容和文件
        // - 文本消息：content 必填，file_id 不需要
        // - 媒体消息：file_id 必填，服务端转换为实际 url 存入 content
        let (content, media) = match message_type {
            MessageType::Text | MessageType::Location => {
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
        let media_meta = media.as_ref().map(|m| {
            serde_json::json!({
                "file_name": m.file_name,
                "file_size": m.file_size,
                "thumbnail_url": m.thumbnail_url,
                "duration": m.duration,
                "width": m.width,
                "height": m.height,
                "mime_type": m.mime_type
            })
        });

        let created = self
            .message_repo
            .create(
                group_id,
                user_id,
                message_type,
                Some(content.clone()),
                media_meta,
                reply_to_message_id,
                mentions.clone(),
            )
            .await?;

        // 5. 更新群聊最后活跃时间（通过更新updated_at字段）
        self.group_repo.touch_updated_at(group_id).await?;

        // 记录消息到布隆过滤器
        if let Err(e) = ResourceExistenceChecker::new(self.redis.clone())
            .record_message(&created.message_id.to_string())
            .await
        {
            tracing::warn!(message_id = %created.message_id, error = %e, "record_message bloom failed");
        }

        // 6. 发布消息事件（用于实时推送）
        // 注意：媒体消息的 content 字段存储的是 media_url
        let event_content = if matches!(
            message_type,
            MessageType::Text | MessageType::System | MessageType::Location
        ) {
            req.content.clone()
        } else {
            None
        };

        self.event_bus.publish(AppEvent::GroupMessageSent {
            message_id: created.message_id,
            group_chat_id: group_id,
            sender_id: user_id,
            message_type, // Use parsed enum
            content: event_content,
            reply_to: reply_to_snapshot,
            mentions,
            media,
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
        let group_id = req
            .group_chat_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 2. 验证用户是否是群成员
        if !self.member_repo.is_member(group_id, user_id).await? {
            return Err(AppError::Forbidden("您不是此群组的成员".to_string()));
        }

        // 3. 解析游标
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let include_system = req.include_system.unwrap_or(false);

        // 4. 查询消息历史
        let (messages, next_cursor) = self
            .message_repo
            .list_by_group(group_id, user_id, limit, cursor, include_system)
            .await?;

        let reply_ids: Vec<Uuid> = messages.iter().filter_map(|m| m.reply_to_id).collect();
        let reply_rows = self.message_repo.find_by_ids(&reply_ids).await?;
        let reply_sender_ids: Vec<i64> = reply_rows.iter().map(|m| m.sender_id).collect();
        let reply_senders = self.user_repo.find_profiles(&reply_sender_ids).await?;
        let reply_sender_map: std::collections::HashMap<i64, _> =
            reply_senders.into_iter().map(|u| (u.user_id, u)).collect();
        let reply_map: std::collections::HashMap<Uuid, _> =
            reply_rows.into_iter().map(|m| (m.message_id, m)).collect();

        let message_ids: Vec<Uuid> = messages.iter().map(|m| m.message_id).collect();
        let read_counts = self.message_repo.get_read_counts(&message_ids).await?;

        // 5. 批量获取发送者和撤回者信息
        let mut user_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        for recalled_id in messages.iter().filter_map(|m| m.recalled_by) {
            if !user_ids.contains(&recalled_id) {
                user_ids.push(recalled_id);
            }
        }
        let users = self.user_repo.find_profiles(&user_ids).await?;
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 6. 转换为DTO
        let mut items = Vec::with_capacity(messages.len());
        for m in messages {
            let user = user_map.get(&m.sender_id);

            // Construct MessageMedia if it's a media message
            let media = if matches!(
                m.r#type,
                MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File
            ) {
                let content_str = m.content.clone().unwrap_or_default();
                let signed_url = match self.media_service.sign_url(&content_str).await {
                    Ok(u) => u,
                    Err(e) => {
                        tracing::warn!("Failed to sign url for path {}: {}", content_str, e);
                        content_str
                    }
                };

                let meta = m.metadata.as_object();
                let thumbnail_url = meta
                    .and_then(|obj| obj.get("thumbnail_url"))
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string());
                let file_name = meta
                    .and_then(|obj| obj.get("file_name"))
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string());
                let file_size = meta
                    .and_then(|obj| obj.get("file_size"))
                    .and_then(|v| v.as_i64());
                let duration = meta
                    .and_then(|obj| obj.get("duration"))
                    .and_then(|v| v.as_f64());
                let width = meta
                    .and_then(|obj| obj.get("width"))
                    .and_then(|v| v.as_i64())
                    .map(|v| v as i32);
                let height = meta
                    .and_then(|obj| obj.get("height"))
                    .and_then(|v| v.as_i64())
                    .map(|v| v as i32);
                let mime_type = meta
                    .and_then(|obj| obj.get("mime_type"))
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string());

                Some(MessageMedia {
                    url: signed_url, // For media messages, content stores the URL
                    thumbnail_url,
                    file_name,
                    file_size,
                    duration,
                    width,
                    height,
                    mime_type,
                })
            } else {
                None
            };

            let mentions = m
                .mentions
                .as_array()
                .map(|arr| arr.iter().filter_map(|v| v.as_i64()).collect::<Vec<i64>>())
                .filter(|arr| !arr.is_empty());

            let recalled_by = if let Some(id) = m.recalled_by {
                if let Some(u) = user_map.get(&id) {
                    Some(UserInfo {
                        user_id: u.user_id,
                        nickname: u.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: self
                            .media_service
                            .sign_url_if_path(u.avatar_url.as_deref())
                            .await,
                    })
                } else {
                    None
                }
            } else {
                None
            };

            items.push(GroupMessageItem {
                message_id: m.message_id.to_string(),
                message_type: m.r#type,
                content: if matches!(
                    m.r#type,
                    MessageType::Text | MessageType::System | MessageType::Location
                ) {
                    m.content.clone()
                } else {
                    None
                },
                sender: UserInfo {
                    user_id: m.sender_id,
                    nickname: user
                        .and_then(|u| u.nickname.clone())
                        .unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: self
                        .media_service
                        .sign_url_if_path(user.and_then(|u| u.avatar_url.as_deref()))
                        .await,
                },
                reply_to: m.reply_to_id.and_then(|reply_id| {
                    reply_map.get(&reply_id).map(|reply_msg| {
                        let reply_sender = reply_sender_map.get(&reply_msg.sender_id);
                        let preview = if reply_msg.r#type == MessageType::Text {
                            reply_msg.content.clone()
                        } else {
                            None
                        };
                        MessageReplyTo {
                            message_id: reply_msg.message_id.to_string(),
                            user_id: reply_msg.sender_id,
                            nickname: reply_sender
                                .and_then(|u| u.nickname.clone())
                                .unwrap_or_else(|| "未知用户".to_string()),
                            content: preview.unwrap_or_default(),
                        }
                    })
                }),
                mentions,
                media,
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                is_recalled: m.is_recalled,
                recalled_by,
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                read_count: read_counts.get(&m.message_id).map(|c| *c as i32),
            });
        }

        Ok(PaginatedData {
            items,
            next_cursor,
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
            return Err(AppError::MessageAlreadyRecalled);
        }

        // 5. 更新消息
        self.message_repo
            .update_content(message_id, req.content.clone())
            .await?;

        // 6. 发布编辑事件
        self.event_bus.publish(AppEvent::GroupMessageEdited {
            message_id,
            group_chat_id: message.group_id,
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
        if message.sender_id == user_id {
            // 发送者可以撤回自己的消息（2分钟内）
            let now = chrono::Utc::now();
            let elapsed = now.signed_duration_since(message.sent_at);
            if elapsed.num_minutes() >= MESSAGE_RECALL_MINUTES {
                return Err(AppError::OperationWindowExpired);
            }
        } else {
            // 检查是否是管理员/群主
            let role = self
                .member_repo
                .get_member_role(message.group_id, user_id)
                .await?;
            if role != Some(GroupRole::Owner) && role != Some(GroupRole::Admin) {
                return Err(AppError::Forbidden("无权撤回此消息".to_string()));
            }
        }

        // 3. 验证消息状态
        if message.is_recalled {
            return Err(AppError::MessageAlreadyRecalled);
        }

        // 4. 撤回消息
        self.message_repo.recall(message_id, user_id).await?;

        // 5. 发布撤回事件
        self.event_bus.publish(AppEvent::GroupMessageRecalled {
            message_id,
            group_chat_id: message.group_id,
            recalled_by: user_id,
            recalled_at: chrono::Utc::now().to_rfc3339(),
        });

        Ok(())
    }

    /// 标记群聊消息已读
    pub async fn mark_as_read(
        &self,
        req: &MarkGroupMessagesReadRequest,
        user_id: i64,
    ) -> Result<()> {
        let group_id = req
            .group_chat_id
            .parse::<i64>()
            .map_err(|_| AppError::BadRequest("无效的群组ID".to_string()))?;

        // 验证用户是否是群成员
        if !self.member_repo.is_member(group_id, user_id).await? {
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
            .mark_as_read("group", &group_id.to_string(), user_id, up_to)
            .await?;

        Ok(())
    }

    /// 标记当前用户所有群聊会话为已读
    pub async fn mark_all_read(&self, user_id: i64) -> Result<()> {
        self.unread_repo
            .mark_all_as_read("group", user_id)
            .await?;
        Ok(())
    }
}
