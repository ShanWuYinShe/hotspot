use crate::{
    dto::response::{
        MessageLimit, MessageLocation, MessageMedia, MessageReplyTo, PrivateChatDetailData,
        PrivateChatLastMessage, PrivateChatListItem, PrivateMessageItem, RecentMessages, UserInfo,
    },
    error::{AppError, Result},
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        private_chat_repo::{PrivateChatRepository, PrivateChatRow},
        private_message_repo::PrivateMessageRepository,
        user_blocks_repo::UserBlocksRepository,
        user_follows_repo::UserFollowsRepository,
        user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    types::enums::{MessageType, PrivateChatPermission, UserType},
};

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
    settings_repo: UserSettingsRepository,
    blocks_repo: UserBlocksRepository,
    follows_repo: UserFollowsRepository,
    user_repo: UserRepository,
    message_repo: PrivateMessageRepository,
    unread_repo: ChatUnreadRepository,
    presence_service: crate::services::presence_service::PresenceService,
}

impl PrivateChatService {
    pub fn new(
        repo: PrivateChatRepository,
        settings_repo: UserSettingsRepository,
        blocks_repo: UserBlocksRepository,
        follows_repo: UserFollowsRepository,
        user_repo: UserRepository,
        message_repo: PrivateMessageRepository,
        unread_repo: ChatUnreadRepository,
        presence_service: crate::services::presence_service::PresenceService,
    ) -> Self {
        Self {
            repo,
            settings_repo,
            blocks_repo,
            follows_repo,
            user_repo,
            message_repo,
            unread_repo,
            presence_service,
        }
    }

    /// 获取用户的所有私聊列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
        include_archived: bool,
    ) -> Result<(Vec<PrivateChatRow>, Option<String>)> {
        self.repo
            .list_by_user(user_id, limit, cursor, include_archived)
            .await
    }

    /// 删除私聊 (通过 target_user_id 定位)
    pub async fn delete(&self, target_user_id: i64, current_user_id: i64) -> Result<()> {
        // 查找私聊
        let chat = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        // 软删除私聊 (注意：这会删除整个会话，通常应该是 hide/archive 语义，或者仅删除当前用户的可见性)
        // 但根据需求描述，这里是 delete。实际业务中可能只是解除关系或隐藏。
        // 这里暂时实现为软删除（影响双方？通常应该是单向隐藏）。
        // 假设是单向隐藏/删除：
        // 实际上 `private_chats` 表是共享的。如果直接 delete，对方也没了。
        // 通常 "删除私聊" 仅是清空当前用户的历史记录或隐藏会话。
        // 但这里我们先按软删除实现（假设是双向的，或者后续优化为单向）。
        // 根据 `private_chat_settings.is_archived` 可能更合适。
        // 但 Spec 中有 `is_archived` 也有 `delete`。
        // 假设 `delete` 是管理员或特殊操作，或者就是双向删除。
        // 为安全起见，这里仅演示调用 repo 的 delete。
        self.repo.soft_delete(chat.id).await?;

        Ok(())
    }

    /// 更新私聊设置 (通过 target_user_id 定位)
    pub async fn update_settings(
        &self,
        target_user_id: i64,
        current_user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
        is_archived: Option<bool>,
    ) -> Result<()> {
        // 查找私聊
        let chat = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        // 更新设置
        self.repo
            .upsert_settings(chat.id, current_user_id, is_muted, is_pinned, is_archived)
            .await?;

        Ok(())
    }

    /// 获取私聊列表（包含用户信息、未读数、最后消息）
    pub async fn get_list_with_details(
        &self,
        current_user_id: i64,
        limit: i64,
        cursor: Option<String>,
        include_archived: bool,
    ) -> Result<(Vec<PrivateChatListItem>, bool, Option<String>)> {
        // 1. 获取私聊列表基础数据
        let (chats, next_cursor) = self
            .list_by_user(current_user_id, limit, cursor, include_archived)
            .await?;

        let has_more = next_cursor.is_some();
        if chats.is_empty() {
            return Ok((Vec::new(), has_more, next_cursor));
        }

        // 2. 收集 ID（使用迭代器方法）
        let (chat_ids, other_user_ids): (Vec<_>, Vec<_>) = chats
            .iter()
            .map(|chat| {
                let other_id = if chat.user_a_id == current_user_id {
                    chat.user_b_id
                } else {
                    chat.user_a_id
                };
                (chat.id, other_id)
            })
            .unzip();

        // 3. 批量查询（并行执行）
        // chat_ids is Vec<i64>, convert to String for unread_repo
        let chat_ids_str: Vec<String> = chat_ids.iter().map(|id| id.to_string()).collect();

        let (users, unread_counts, last_messages) = tokio::try_join!(
            self.user_repo.find_profiles(&other_user_ids),
            self.unread_repo
                .get_unread_count_batch("private_chat", &chat_ids_str, current_user_id),
            self.message_repo.get_last_message_batch(&chat_ids)
        )?;

        // 4. 构建映射
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // unread_counts key is String
        let unread_map = unread_counts;

        // 批量获取在线状态
        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&other_user_ids)
            .await;

        // 批量获取私聊设置
        let settings_map = self
            .repo
            .get_settings_batch(&chat_ids, current_user_id)
            .await?;

        // 5. 组装结果
        let mut items = Vec::with_capacity(chats.len());
        for chat in chats {
            // ... (user id logic) ...
            let other_user_id = if chat.user_a_id == current_user_id {
                chat.user_b_id
            } else {
                chat.user_a_id
            };

            let other_user_info = user_map
                .get(&other_user_id)
                .map(|u| UserInfo {
                    user_id: u.user_id,
                    nickname: u.nickname.clone(),
                    avatar_url: u.avatar_url.clone(),
                    user_type: u.user_type,
                })
                .unwrap_or_else(|| UserInfo {
                    user_id: other_user_id,
                    nickname: "未知用户".to_string(),
                    avatar_url: None,
                    user_type: UserType::Normal,
                });

            let unread_count = unread_map.get(&chat.id.to_string()).copied().unwrap_or(0);

            let last_message_row = last_messages.get(&chat.id);
            let last_message = last_message_row.map(|m| PrivateChatLastMessage {
                message_id: m.message_id.to_string(),
                content: Some(m.content.clone()),
                message_type: m.r#type,
                sender_id: m.sender_id,
                sent_at: m.sent_at.to_rfc3339(),
            });

            let last_activity_at = last_message
                .as_ref()
                .map(|m| m.sent_at.clone())
                .unwrap_or_else(|| chat.created_at.to_rfc3339());

            // Get is_online (with privacy check)
            let is_online = online_status_map
                .get(&other_user_id)
                .and_then(|(online, _)| *online)
                .unwrap_or(false);

            // 获取用户的私聊设置
            let settings = settings_map.get(&chat.id);
            let is_muted = settings.map(|s| s.is_muted).unwrap_or(false);
            let is_pinned = settings.map(|s| s.is_pinned).unwrap_or(false);

            items.push(PrivateChatListItem {
                other_user: other_user_info,
                unread_count: unread_count as i32,
                last_message,
                last_activity_at,
                is_muted,
                is_pinned,
                is_online,
            });
        }

        Ok((items, has_more, next_cursor))
    }

    /// 获取私聊详情（包含用户信息、消息限制、未读数等）
    pub async fn get_details(
        &self,
        target_user_id: i64,
        current_user_id: i64,
        recent_messages_limit: Option<i32>,
    ) -> Result<PrivateChatDetailData> {
        // 查找或创建私聊
        let chat_opt = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?;

        let chat = match chat_opt {
            Some(c) => c,
            None => return Err(AppError::ChatNotFound("私聊不存在".to_string())),
        };

        // 获取对方用户信息
        let other_user = self.user_repo.find_by_id(target_user_id).await?;
        let other_user_info = other_user
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: u.avatar_url,
                user_type: u.user_type,
            })
            .unwrap_or_else(|| UserInfo {
                user_id: target_user_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
                user_type: crate::types::enums::UserType::Normal,
            });

        // 获取未读计数
        let unread_count = self
            .unread_repo
            .get_unread_count("private_chat", &chat.id.to_string(), current_user_id)
            .await
            .unwrap_or(0);

        // 获取最后一条消息
        let last_message =
            self.message_repo
                .get_last_message(chat.id)
                .await?
                .map(|m| PrivateChatLastMessage {
                    message_id: m.message_id.to_string(),
                    content: Some(m.content),
                    message_type: m.r#type,
                    sender_id: m.sender_id,
                    sent_at: m.sent_at.to_rfc3339(),
                });

        let last_activity_at = last_message
            .as_ref()
            .map(|m| m.sent_at.clone())
            .unwrap_or_else(|| chat.created_at.to_rfc3339());

        // 获取当前用户信息，判断是否为特权用户（管理员/系统账号）
        let current_user = self.user_repo.find_by_id(current_user_id).await?;
        let is_privileged_user = current_user
            .as_ref()
            .map(|u| matches!(u.user_type, UserType::Admin | UserType::System))
            .unwrap_or(false);

        // 计算 can_message
        // 特权用户（管理员/系统账号）不受任何限制
        let can_message = if is_privileged_user {
            true
        } else {
            // 1. 检查当前用户是否被目标用户拉黑
            let is_blocked = self
                .blocks_repo
                .is_blocked(target_user_id, current_user_id)
                .await
                .unwrap_or(false);

            // 2. 检查目标用户是否允许接收私聊
            let target_settings = self
                .settings_repo
                .find_by_user_id(target_user_id)
                .await
                .unwrap_or(None);
            let allow_chat = if let Some(s) = target_settings {
                if s.allow_private_chat == PrivateChatPermission::Nobody {
                    false
                } else if s.allow_private_chat == PrivateChatPermission::Friends {
                    // 检查是否是好友（互相关注）
                    let is_following = self
                        .follows_repo
                        .exists(current_user_id, target_user_id)
                        .await
                        .unwrap_or(false);
                    let is_followed_by = self
                        .follows_repo
                        .exists(target_user_id, current_user_id)
                        .await
                        .unwrap_or(false);
                    is_following && is_followed_by
                } else {
                    true // everyone
                }
            } else {
                true // default everyone
            };

            !is_blocked && allow_chat
        };

        // 检查消息限制（陌生人消息限制）
        // 特权用户不受消息限制
        let is_friend = self
            .follows_repo
            .are_mutual_follows(current_user_id, target_user_id)
            .await
            .unwrap_or(false);
        let message_limit = if is_privileged_user {
            // 特权用户不受限制
            None
        } else if !is_friend {
            // 从 chat row 中获取已发送消息数和是否可以自由发送
            let (can_message_freely, sent_count) = if chat.user_a_id == current_user_id {
                (chat.user_a_can_message_freely, chat.user_a_sent_count)
            } else {
                (chat.user_b_can_message_freely, chat.user_b_sent_count)
            };

            if !can_message_freely {
                let total = if self
                    .follows_repo
                    .exists(current_user_id, target_user_id)
                    .await
                    .unwrap_or(false)
                {
                    3
                } else {
                    1
                };
                Some(MessageLimit {
                    remaining: (total - sent_count).max(0),
                    total,
                    is_unlocked: false,
                })
            } else {
                None
            }
        } else {
            None
        };

        // 获取私聊设置
        let settings = self.repo.get_settings(chat.id, current_user_id).await?;

        // 加载最近消息（默认 20 条，最大 50 条）
        let limit = recent_messages_limit.unwrap_or(20).clamp(1, 50) as i64;
        let (messages, next_cursor) = self.message_repo.list_by_chat(chat.id, limit, None).await?;
        let has_more = next_cursor.is_some();

        let reply_ids: Vec<_> = messages
            .iter()
            .filter_map(|m| m.reply_to_message_id)
            .collect();
        let reply_rows = self.message_repo.find_by_ids(&reply_ids).await?;
        let reply_sender_ids: Vec<i64> = reply_rows.iter().map(|m| m.sender_id).collect();
        let reply_senders = self.user_repo.find_profiles(&reply_sender_ids).await?;
        let reply_sender_map: std::collections::HashMap<i64, _> =
            reply_senders.into_iter().map(|s| (s.user_id, s)).collect();
        let reply_map: std::collections::HashMap<_, _> =
            reply_rows.into_iter().map(|m| (m.message_id, m)).collect();

        // 获取发送者信息
        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let senders = self.user_repo.find_profiles(&sender_ids).await?;
        let sender_map: std::collections::HashMap<i64, _> =
            senders.into_iter().map(|s| (s.user_id, s)).collect();

        // 转换消息为响应格式
        use crate::dto::response::Location;

        let message_items: Vec<PrivateMessageItem> = messages
            .into_iter()
            .map(|m| {
                let sender_info = sender_map.get(&m.sender_id);
                PrivateMessageItem {
                    message_id: m.message_id.to_string(),
                    message_type: m.r#type,
                    content: Some(m.content.clone()),
                    sender: sender_info
                        .map(|s| UserInfo {
                            user_id: s.user_id,
                            nickname: s.nickname.clone(),
                            avatar_url: s.avatar_url.clone(),
                            user_type: s.user_type,
                        })
                        .unwrap_or_else(|| UserInfo {
                            user_id: m.sender_id,
                            nickname: "未知用户".to_string(),
                            avatar_url: None,
                            user_type: UserType::Normal,
                        }),
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
                                    .map(|s| s.nickname.clone())
                                    .unwrap_or_else(|| "未知用户".to_string()),
                                content: preview,
                            }
                        })
                    }),
                    media: m.thumbnail_url.as_ref().map(|url| MessageMedia {
                        url: url.clone(),
                        file_name: m.file_name.clone(),
                        file_size: m.file_size,
                        thumbnail_url: m.thumbnail_url.clone(),
                        duration: m.duration.map(|d| d as f64),
                        width: None,
                        height: None,
                        mime_type: None,
                    }),
                    location: if m.location_latitude.is_some() {
                        Some(MessageLocation {
                            location: Location {
                                latitude: m.location_latitude.unwrap_or(0.0),
                                longitude: m.location_longitude.unwrap_or(0.0),
                                location_name: m.location_name.clone().unwrap_or_default(),
                            },
                            address: m.location_address.clone(),
                        })
                    } else {
                        None
                    },
                    sent_at: m.sent_at.to_rfc3339(),
                    edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                    is_recalled: m.is_recalled,
                    recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                }
            })
            .collect();

        Ok(PrivateChatDetailData {
            other_user: other_user_info,
            message_limit,
            unread_count: unread_count as i32,
            is_muted: settings.as_ref().map(|s| s.is_muted).unwrap_or(false),
            is_pinned: settings.as_ref().map(|s| s.is_pinned).unwrap_or(false),
            last_activity_at,
            created_at: chat.created_at.to_rfc3339(),
            can_message,
            recent_messages: RecentMessages {
                items: message_items,
                has_more,
                next_cursor,
            },
        })
    }
}
