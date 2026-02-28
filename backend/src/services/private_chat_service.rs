use crate::{
    dto::request::DeletePrivateChatRequest,
    dto::response::{
        MessageMedia, MessageReplyTo, PaginatedData, PrivateChatLastMessage, PrivateChatListItem,
        PrivateChatOpenResult, PrivateMessageItem, UserInfo,
    },
    error::{AppError, Result},
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
    types::enums::{MessageBlockedReason, MessageType, UserPresenceStatus},
};

use chrono::{DateTime, Utc};
use tracing::{info, warn};
use uuid::Uuid;

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
    permission_resolver: PrivateChatPermissionResolver,
    user_repo: UserRepository,
    message_repo: PrivateMessageRepository,
    unread_repo: ChatUnreadRepository,
    presence_service: crate::services::presence_service::PresenceService,
    media_service: crate::services::media_service::MediaService,
    stranger_limiter: StrangerMessageLimiter,
}

impl PrivateChatService {
    fn cleared_at_for_user(chat: &PrivateChatRow, current_user_id: i64) -> Option<DateTime<Utc>> {
        if chat.user1_id == current_user_id {
            chat.user1_cleared_at
        } else if chat.user2_id == current_user_id {
            chat.user2_cleared_at
        } else {
            None
        }
    }

    pub fn new(
        repo: PrivateChatRepository,
        settings_repo: UserSettingsRepository,
        blocks_repo: UserBlocksRepository,
        follows_repo: UserFollowsRepository,
        user_repo: UserRepository,
        message_repo: PrivateMessageRepository,
        unread_repo: ChatUnreadRepository,
        presence_service: crate::services::presence_service::PresenceService,
        media_service: crate::services::media_service::MediaService,
        stranger_limiter: StrangerMessageLimiter,
        _cache_manager: crate::utils::cache::CacheManager,
    ) -> Self {
        let permission_resolver =
            PrivateChatPermissionResolver::new(settings_repo, follows_repo, blocks_repo);

        Self {
            repo,
            permission_resolver,
            user_repo,
            message_repo,
            unread_repo,
            presence_service,
            media_service,
            stranger_limiter,
        }
    }

    /// 获取用户的所有私聊列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateChatRow>, Option<String>)> {
        self.repo.list_by_user(user_id, limit, cursor).await
    }

    /// 更新私聊设置 (通过 target_user_id 定位)
    pub async fn update_settings(
        &self,
        target_user_id: i64,
        current_user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        info!(
            current_user_id,
            target_user_id,
            has_is_muted = is_muted.is_some(),
            has_is_pinned = is_pinned.is_some(),
            "private_chat.update_settings started"
        );
        let chat = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
            .ok_or_else(|| AppError::ChatNotFound("私聊不存在".to_string()))?;

        if is_muted.is_some() || is_pinned.is_some() {
            self.unread_repo
                .update_private_chat_settings(
                    current_user_id,
                    &chat.chat_id.to_string(),
                    is_muted,
                    is_pinned,
                )
                .await?;
        }

        info!(
            current_user_id,
            target_user_id,
            chat_id = %chat.chat_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "private_chat.update_settings completed"
        );

        Ok(())
    }

    /// 删除我的私聊会话（单边删除）：隐藏会话入口并清空当前用户历史视图
    pub async fn delete_chat(
        &self,
        current_user_id: i64,
        req: &DeletePrivateChatRequest,
    ) -> Result<()> {
        let started_at = std::time::Instant::now();
        let target_user_id =
            crate::utils::validation::Validator::parse_user_id(&req.target_user_id)?;
        info!(
            current_user_id,
            target_user_id, "private_chat.delete started"
        );
        let chat = match self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
        {
            Some(chat) => chat,
            None => {
                warn!(current_user_id, target_user_id, "删除私聊失败：会话不存在");
                return Err(AppError::ChatNotFound("私聊不存在".to_string()));
            }
        };

        self.repo
            .mark_cleared_by_user(chat.chat_id, current_user_id)
            .await?;
        self.unread_repo
            .hide_private_chat(current_user_id, &chat.chat_id.to_string())
            .await?;

        info!(
            current_user_id,
            target_user_id,
            chat_id = %chat.chat_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "删除私聊成功（单边清理）"
        );
        Ok(())
    }

    /// 获取私聊列表（包含用户信息、未读数、最后消息）
    pub async fn get_list_with_details(
        &self,
        current_user_id: i64,
        limit: i64,
        cursor: Option<String>,
    ) -> Result<(Vec<PrivateChatListItem>, bool, Option<String>)> {
        let (chats, next_cursor) = self.list_by_user(current_user_id, limit, cursor).await?;

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

        let (users, unread_counts, last_messages, settings_map) = tokio::try_join!(
            self.user_repo.find_profiles(&user_ids_to_fetch),
            self.unread_repo
                .get_unread_count_batch("private", &chat_ids_str, current_user_id),
            self.message_repo.get_last_message_batch(&chat_ids),
            self.unread_repo
                .get_private_chat_settings_batch(&chat_ids_str, current_user_id)
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

            let other_user_info = if let Some(u) = user_map.get(&other_user_id) {
                let avatar_url = self
                    .media_service
                    .sign_url_if_path(u.avatar_url.as_deref())
                    .await;
                UserInfo::new(
                    u.user_id,
                    u.nickname
                        .clone()
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                )
            } else {
                UserInfo::new(other_user_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
            };

            let unread_count = unread_counts
                .get(&chat.chat_id.to_string())
                .copied()
                .unwrap_or(0);

            let (is_muted, is_pinned) = settings_map
                .get(&chat.chat_id.to_string())
                .copied()
                .unwrap_or((false, false));

            let last_message_row = last_messages.get(&chat.chat_id);
            let last_message = if let Some(m) = last_message_row {
                if let Some(u) = user_map.get(&m.sender_id) {
                    let avatar_url = self
                        .media_service
                        .sign_url_if_path(u.avatar_url.as_deref())
                        .await;
                    let is_media_message = matches!(
                        m.r#type,
                        MessageType::Image
                            | MessageType::Video
                            | MessageType::Audio
                            | MessageType::File
                    );
                    let share_code = if matches!(m.r#type, MessageType::Share) {
                        m.content.clone()
                    } else {
                        None
                    };
                    Some(PrivateChatLastMessage {
                        message_id: m.message_id.to_string(),
                        content: if share_code.is_some() || is_media_message {
                            None
                        } else {
                            m.content.clone()
                        },
                        message_type: m.r#type,
                        share_code,
                        sender: UserInfo::new(
                            u.user_id,
                            u.nickname.clone().unwrap_or_else(|| {
                                crate::constants::UNKNOWN_USER_NICKNAME.to_string()
                            }),
                            avatar_url,
                        ),
                        sent_at: m.sent_at.to_rfc3339(),
                    })
                } else {
                    None
                }
            } else {
                None
            };

            let last_activity_at = last_message
                .as_ref()
                .map(|m| m.sent_at.clone())
                .unwrap_or_else(|| chat.updated_at.to_rfc3339());

            let (is_online, _last_online) = online_status_map
                .get(&other_user_id)
                .cloned()
                .unwrap_or((None, None));
            let presence_status = is_online.map(|online| {
                if online {
                    UserPresenceStatus::Online
                } else {
                    UserPresenceStatus::Offline
                }
            });

            let permission = self
                .permission_resolver
                .evaluate(current_user_id, other_user_id, user_map.get(&other_user_id))
                .await?;
            let mut can_message = permission.can_message;
            let mut message_blocked_reason = permission.reason;
            if can_message && !permission.is_mutual_friend {
                if self
                    .stranger_limiter
                    .is_over_limit(current_user_id, other_user_id)
                    .await?
                {
                    can_message = false;
                    message_blocked_reason = Some(MessageBlockedReason::StrangerQuotaExceeded);
                }
            }

            items.push(PrivateChatListItem {
                other_user: other_user_info,
                unread_count: unread_count as i32,
                last_message,
                last_activity_at,
                is_muted,
                is_pinned,
                presence_status,
                can_message: Some(can_message),
                message_blocked_reason,
            });
        }

        Ok((items, has_more, next_cursor))
    }

    /// 获取私聊详情（包含用户信息、未读数等）
    pub async fn get_details(
        &self,
        target_user_id: i64,
        current_user_id: i64,
    ) -> Result<PrivateChatOpenResult> {
        let chat = self
            .repo
            .get_or_create(current_user_id, target_user_id)
            .await?;

        let other_user = self.user_repo.find_by_id(target_user_id).await?;
        let other_user_info = if let Some(u) = &other_user {
            let avatar_url = self
                .media_service
                .sign_url_if_path(u.avatar_url.as_deref())
                .await;
            UserInfo::new(
                u.user_id,
                u.nickname
                    .clone()
                    .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                avatar_url,
            )
        } else {
            UserInfo::new(
                target_user_id,
                crate::constants::UNKNOWN_USER_NICKNAME,
                None,
            )
        };

        let chat_id_str = chat.chat_id.to_string();

        let unread_count = self
            .unread_repo
            .get_unread_count("private", &chat_id_str, current_user_id)
            .await
            .unwrap_or(0);

        let (is_muted, is_pinned) = self
            .unread_repo
            .get_private_chat_settings(&chat_id_str, current_user_id)
            .await
            .unwrap_or((false, false));

        let other_last_read = self
            .unread_repo
            .get_last_read_message_id(target_user_id, &chat_id_str)
            .await?;

        let permission = self
            .permission_resolver
            .evaluate(current_user_id, target_user_id, other_user.as_ref())
            .await?;
        let mut can_message = permission.can_message;
        let mut message_blocked_reason = permission.reason;

        if can_message && !permission.is_mutual_friend {
            if self
                .stranger_limiter
                .is_over_limit(current_user_id, target_user_id)
                .await?
            {
                can_message = false;
                message_blocked_reason = Some(MessageBlockedReason::StrangerQuotaExceeded);
            }
        }

        let limit = crate::constants::DEFAULT_PAGE_LIMIT;
        let cleared_after = Self::cleared_at_for_user(&chat, current_user_id);
        let (messages, next_cursor) = self
            .message_repo
            .list_by_chat(chat.chat_id, limit, None, cleared_after)
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

        let last_message = if let Some(m) = messages.first() {
            let sender_info = if let Some(s) = sender_map.get(&m.sender_id) {
                let avatar_url = self
                    .media_service
                    .sign_url_if_path(s.avatar_url.as_deref())
                    .await;
                UserInfo::new(
                    s.user_id,
                    s.nickname
                        .clone()
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                )
            } else {
                UserInfo::new(m.sender_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
            };
            let is_media_message = matches!(
                m.r#type,
                MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File
            );
            let share_code = if matches!(m.r#type, MessageType::Share) {
                m.content.clone()
            } else {
                None
            };
            Some(PrivateChatLastMessage {
                message_id: m.message_id.to_string(),
                content: if share_code.is_some() || is_media_message {
                    None
                } else {
                    m.content.clone()
                },
                message_type: m.r#type,
                share_code,
                sender: sender_info,
                sent_at: m.sent_at.to_rfc3339(),
            })
        } else {
            None
        };
        let last_activity_at = last_message
            .as_ref()
            .map(|m| m.sent_at.clone())
            .unwrap_or_else(|| chat.updated_at.to_rfc3339());

        let (is_online, _last_online) = self
            .presence_service
            .get_online_status_with_privacy(target_user_id)
            .await;
        let presence_status = is_online.map(|online| {
            if online {
                UserPresenceStatus::Online
            } else {
                UserPresenceStatus::Offline
            }
        });

        let chat_item = PrivateChatListItem {
            other_user: other_user_info.clone(),
            unread_count: unread_count as i32,
            last_message,
            last_activity_at,
            is_muted,
            is_pinned,
            presence_status,
            can_message: Some(can_message),
            message_blocked_reason,
        };

        let mut message_items = Vec::with_capacity(messages.len());
        for m in messages {
            let sender_info = if let Some(s) = sender_map.get(&m.sender_id) {
                let avatar_url = self
                    .media_service
                    .sign_url_if_path(s.avatar_url.as_deref())
                    .await;
                UserInfo::new(
                    s.user_id,
                    s.nickname
                        .clone()
                        .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                    avatar_url,
                )
            } else {
                UserInfo::new(m.sender_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
            };
            let is_read = if m.sender_id != current_user_id {
                true
            } else if let Some(ref last_ts) = other_last_read_sent_at {
                if m.sent_at < *last_ts {
                    true
                } else if m.sent_at == *last_ts {
                    other_last_read_id
                        .map(|last_id| last_id == m.message_id)
                        .unwrap_or(false)
                } else {
                    false
                }
            } else {
                false
            };

            let is_media_message = matches!(
                m.r#type,
                MessageType::Image | MessageType::Video | MessageType::Audio | MessageType::File
            );

            let media = if is_media_message {
                let storage_path = m.content.clone().unwrap_or_default();
                let signed_url = match self.media_service.sign_url(&storage_path).await {
                    Ok(url) => url,
                    Err(e) => {
                        warn!(
                            error = %e,
                            path = %storage_path,
                            "Failed to sign private chat media URL"
                        );
                        storage_path.clone()
                    }
                };

                Some(MessageMedia {
                    url: signed_url,
                    file_name: m.file_name.clone(),
                    file_size: m.file_size,
                    thumbnail_url: m.thumbnail_url.clone(),
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
                            .and_then(|s| s.nickname.clone())
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

            let item = PrivateMessageItem {
                message_id: m.message_id.to_string(),
                message_type: m.r#type,
                content: if share_code.is_some() || is_media_message {
                    None
                } else {
                    m.content.clone()
                },
                sender: sender_info,
                reply_to,
                media,
                share_code,
                sent_at: m.sent_at.to_rfc3339(),
                edited_at: m.edited_at.map(|t| t.to_rfc3339()),
                recalled_at: m.recalled_at.map(|t| t.to_rfc3339()),
                is_read,
            };
            message_items.push(item);
        }

        Ok(PrivateChatOpenResult {
            chat: chat_item,
            recent_messages: PaginatedData {
                items: message_items,
                next_cursor,
            },
        })
    }
}
