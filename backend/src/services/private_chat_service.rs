use crate::{
    dto::response::{
        MessageLimit, MessageMedia, MessageReplyTo, PrivateChatDetailData, PrivateChatLastMessage,
        PrivateChatListItem, PrivateMessageItem, RecentMessages, UserInfo,
    },
    error::{AppError, Result},
    repositories::{
        chat_unread_repo::ChatUnreadRepository,
        private_chat_repo::{PrivateChatRepository, PrivateChatRow},
        private_message_repo::PrivateMessageRepository,
        user_blocks_repo::UserBlocksRepository,
        user_repo::UserRepository,
    },
    types::enums::MessageType,
};

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
    blocks_repo: UserBlocksRepository,
    user_repo: UserRepository,
    message_repo: PrivateMessageRepository,
    unread_repo: ChatUnreadRepository,
    presence_service: crate::services::presence_service::PresenceService,
}

impl PrivateChatService {
    pub fn new(
        repo: PrivateChatRepository,
        _settings_repo: crate::repositories::user_settings_repo::UserSettingsRepository,
        blocks_repo: UserBlocksRepository,
        _follows_repo: crate::repositories::user_follows_repo::UserFollowsRepository,
        user_repo: UserRepository,
        message_repo: PrivateMessageRepository,
        unread_repo: ChatUnreadRepository,
        presence_service: crate::services::presence_service::PresenceService,
        _cache_manager: crate::utils::cache::CacheManager,
    ) -> Self {
        Self {
            repo,
            blocks_repo,
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
        _include_archived: bool,
    ) -> Result<(Vec<PrivateChatRow>, Option<String>)> {
        self.repo.list_by_user(user_id, limit, cursor).await
    }

    /// 删除私聊 (通过 target_user_id 定位)
    pub async fn delete(&self, target_user_id: i64, current_user_id: i64) -> Result<()> {
        let _chat = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        Ok(())
    }

    /// 更新私聊设置 (通过 target_user_id 定位)
    pub async fn update_settings(
        &self,
        _target_user_id: i64,
        _current_user_id: i64,
        _is_muted: Option<bool>,
        _is_pinned: Option<bool>,
        _is_archived: Option<bool>,
    ) -> Result<()> {
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
        let (chats, next_cursor) = self
            .list_by_user(current_user_id, limit, cursor, include_archived)
            .await?;

        let has_more = next_cursor.is_some();
        if chats.is_empty() {
            return Ok((Vec::new(), has_more, next_cursor));
        }

        let (chat_ids, other_user_ids): (Vec<_>, Vec<_>) = chats
            .iter()
            .map(|chat| {
                let other_id = if chat.user1_id == current_user_id {
                    chat.user2_id
                } else {
                    chat.user1_id
                };
                (chat.chat_id, other_id)
            })
            .unzip();

        let chat_ids_str: Vec<String> = chat_ids.iter().map(|id| id.to_string()).collect();

        // Include current_user_id for last message sender (when current user sent the last msg)
        let mut user_ids_to_fetch: Vec<i64> = other_user_ids.iter().copied().collect();
        user_ids_to_fetch.push(current_user_id);

        let (users, unread_counts, last_messages) = tokio::try_join!(
            self.user_repo.find_profiles(&user_ids_to_fetch),
            self.unread_repo
                .get_unread_count_batch("private", &chat_ids_str, current_user_id),
            self.message_repo.get_last_message_batch(&chat_ids)
        )?;

        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        let online_status_map = self
            .presence_service
            .batch_get_online_status_with_privacy(&other_user_ids)
            .await;

        let mut items = Vec::with_capacity(chats.len());
        for chat in chats {
            let other_user_id = if chat.user1_id == current_user_id {
                chat.user2_id
            } else {
                chat.user1_id
            };

            let other_user_info = user_map
                .get(&other_user_id)
                .map(|u| UserInfo {
                    user_id: u.user_id,
                    nickname: u.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                    avatar_url: u.avatar_url.clone(),
                })
                .unwrap_or_else(|| UserInfo {
                    user_id: other_user_id,
                    nickname: "未知用户".to_string(),
                    avatar_url: None,
                });

            let unread_count = unread_counts
                .get(&chat.chat_id.to_string())
                .copied()
                .unwrap_or(0);

            let last_message_row = last_messages.get(&chat.chat_id);
            let last_message = last_message_row.and_then(|m| {
                user_map.get(&m.sender_id).map(|u| PrivateChatLastMessage {
                    message_id: m.message_id.to_string(),
                    content: m.content.clone(),
                    message_type: m.r#type,
                    sender: UserInfo {
                        user_id: u.user_id,
                        nickname: u.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                        avatar_url: u.avatar_url.clone(),
                    },
                    sent_at: m.sent_at.to_rfc3339(),
                })
            });

            let last_activity_at = last_message
                .as_ref()
                .map(|m| m.sent_at.clone())
                .unwrap_or_else(|| chat.updated_at.to_rfc3339());

            let is_online = online_status_map
                .get(&other_user_id)
                .and_then(|(online, _)| *online)
                .unwrap_or(false);

            items.push(PrivateChatListItem {
                other_user: other_user_info,
                unread_count: unread_count as i32,
                last_message,
                last_activity_at,
                is_muted: false,
                is_pinned: false,
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
        let chat = self
            .repo
            .get_or_create(current_user_id, target_user_id)
            .await?;

        let other_user = self.user_repo.find_by_id(target_user_id).await?;
        let other_user_info = other_user
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname.unwrap_or_else(|| "未知用户".to_string()),
                avatar_url: u.avatar_url,
            })
            .unwrap_or_else(|| UserInfo {
                user_id: target_user_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
            });

        let unread_count = self
            .unread_repo
            .get_unread_count("private", &chat.chat_id.to_string(), current_user_id)
            .await
            .unwrap_or(0);

        let is_blocked = self
            .blocks_repo
            .is_blocked(target_user_id, current_user_id)
            .await
            .unwrap_or(false);
        let can_message = !is_blocked;

        let message_limit: Option<MessageLimit> = None;

        let limit = recent_messages_limit.unwrap_or(20).clamp(1, 50) as i64;
        let (messages, next_cursor) = self
            .message_repo
            .list_by_chat(chat.chat_id, current_user_id, limit, None)
            .await?;

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

        let sender_ids: Vec<i64> = messages.iter().map(|m| m.sender_id).collect();
        let senders = self.user_repo.find_profiles(&sender_ids).await?;
        let sender_map: std::collections::HashMap<i64, _> =
            senders.into_iter().map(|s| (s.user_id, s)).collect();

        let last_message = messages.first().map(|m| {
            let sender_info = sender_map.get(&m.sender_id).map(|s| UserInfo {
                user_id: s.user_id,
                nickname: s.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                avatar_url: s.avatar_url.clone(),
            }).unwrap_or_else(|| UserInfo {
                user_id: m.sender_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
            });
            PrivateChatLastMessage {
                message_id: m.message_id.to_string(),
                content: m.content.clone(),
                message_type: m.r#type,
                sender: sender_info,
                sent_at: m.sent_at.to_rfc3339(),
            }
        });
        let last_activity_at = last_message
            .as_ref()
            .map(|m| m.sent_at.clone())
            .unwrap_or_else(|| chat.updated_at.to_rfc3339());

        let message_items: Vec<PrivateMessageItem> = messages
            .into_iter()
            .map(|m| {
                let sender_info = sender_map.get(&m.sender_id);
                PrivateMessageItem {
                    message_id: m.message_id.to_string(),
                    message_type: m.r#type,
                    content: m.content.clone(),
                    sender: sender_info
                        .map(|s| UserInfo {
                            user_id: s.user_id,
                            nickname: s.nickname.clone().unwrap_or_else(|| "未知用户".to_string()),
                            avatar_url: s.avatar_url.clone(),
                        })
                        .unwrap_or_else(|| UserInfo {
                            user_id: m.sender_id,
                            nickname: "未知用户".to_string(),
                            avatar_url: None,
                        }),
                    reply_to: m.reply_to_message_id.and_then(|reply_id| {
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
                                    .and_then(|s| s.nickname.clone())
                                    .unwrap_or_else(|| "未知用户".to_string()),
                                content: preview,
                            }
                        })
                    }),
                    media: if matches!(
                        m.r#type,
                        MessageType::Image
                            | MessageType::Video
                            | MessageType::Audio
                            | MessageType::File
                    ) {
                        Some(MessageMedia {
                            url: m.content.clone().unwrap_or_default(),
                            file_name: m.file_name.clone(),
                            file_size: m.file_size,
                            thumbnail_url: m.thumbnail_url.clone(),
                            duration: m.duration.map(|d| d as f64),
                            width: None,
                            height: None,
                            mime_type: None,
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
            is_muted: false,
            is_pinned: false,
            last_activity_at,
            created_at: chat.created_at.to_rfc3339(),
            can_message,
            recent_messages: RecentMessages {
                items: message_items,
                next_cursor,
            },
        })
    }
}
