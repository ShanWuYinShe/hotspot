use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    repositories::{
        private_chat_repo::{PrivateChatRepository, PrivateChatRow},
        user_blocks_repo::UserBlocksRepository,
        user_follows_repo::UserFollowsRepository,
        user_settings_repo::UserSettingsRepository,
    },
};

/// 私聊服务
#[derive(Clone)]
pub struct PrivateChatService {
    repo: PrivateChatRepository,
    settings_repo: UserSettingsRepository,
    blocks_repo: UserBlocksRepository,
    follows_repo: UserFollowsRepository,
    user_repo: crate::repositories::user_repo::UserRepository,
    message_repo: crate::repositories::private_message_repo::PrivateMessageRepository,
    unread_repo: crate::repositories::chat_unread_repo::ChatUnreadRepository,
}

impl PrivateChatService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            repo: PrivateChatRepository::new(pool.clone()),
            settings_repo: UserSettingsRepository::new(pool.clone()),
            blocks_repo: UserBlocksRepository::new(pool.clone()),
            follows_repo: UserFollowsRepository::new(pool.clone()),
            user_repo: crate::repositories::user_repo::UserRepository::new(pool.clone()),
            message_repo: crate::repositories::private_message_repo::PrivateMessageRepository::new(
                pool.clone(),
            ),
            unread_repo: crate::repositories::chat_unread_repo::ChatUnreadRepository::new(pool),
        }
    }

    /// 创建或获取私聊
    ///
    /// 权限验证逻辑（根据 API 规范 3.5.1）：
    /// 1. 目标用户的 allow_private_chat = true（否则返回错误码 1309）
    /// 2. 目标用户没有将当前用户拉黑（否则返回错误码 1310）
    pub async fn create_or_get(
        &self,
        current_user_id: i64,
        target_user_id: i64,
        initial_message: Option<String>,
    ) -> Result<PrivateChatRow> {
        if current_user_id == target_user_id {
            return Err(AppError::BadRequest("不能与自己创建私聊".to_string()));
        }

        // 尝试查找现有私聊（如果已存在则直接返回，无需权限验证）
        if let Some(chat) = self
            .repo
            .find_by_users(current_user_id, target_user_id)
            .await?
        {
            return Ok(chat);
        }

        // 创建新私聊前进行权限验证

        // 1. 检查目标用户是否允许接收私聊
        let target_settings = self.settings_repo.find_by_user_id(target_user_id).await?;
        if let Some(settings) = target_settings {
            match settings.allow_private_chat.as_str() {
                "nobody" => {
                    return Err(AppError::PrivateChatNotAllowed(
                        "对方不允许接收私聊".to_string(),
                    ));
                }
                "friends" => {
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
                    let is_mutual_follow = is_following && is_followed_by;

                    if !is_mutual_follow {
                        return Err(AppError::PrivateChatNotAllowed(
                            "对方仅允许好友发起私聊".to_string(),
                        ));
                    }
                }
                _ => {
                    // "everyone" 或其他值，允许私聊
                }
            }
        }
        // 如果目标用户没有设置记录，使用默认值（allow_private_chat = 'everyone'）

        // 2. 检查当前用户是否被目标用户拉黑
        if self
            .blocks_repo
            .is_blocked(target_user_id, current_user_id)
            .await?
        {
            return Err(AppError::BlockedByUser("已被对方拉黑".to_string()));
        }

        // 创建新私聊
        let chat = self.repo.create(current_user_id, target_user_id).await?;

        // 可选：创建时发送一条初始文本消息
        if let Some(content) = initial_message {
            let content = content.trim().to_string();
            if !content.is_empty() {
                // 复用消息仓库直接插入一条 text 消息
                let _ = self
                    .message_repo
                    .create(
                        chat.private_chat_id,
                        current_user_id,
                        crate::types::enums::MessageType::Text,
                        &content,
                        None,
                        None,
                        None,
                        None,
                        None,
                    )
                    .await?;
            }
        }
        Ok(chat)
    }

    /// 根据 private_chat_id 查找私聊
    pub async fn find_by_id(&self, private_chat_id: Uuid) -> Result<Option<PrivateChatRow>> {
        self.repo.find_by_id(private_chat_id).await
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

    /// 删除私聊
    pub async fn delete(&self, private_chat_id: Uuid, user_id: i64) -> Result<()> {
        // 检查私聊是否存在
        if !self.repo.exists(private_chat_id).await? {
            return Err(AppError::NotFound("私聊不存在".to_string()));
        }

        // 检查用户是否是私聊成员
        let is_member = self.repo.is_user_in_chat(private_chat_id, user_id).await?;
        if !is_member {
            return Err(AppError::Forbidden("无权删除此私聊".to_string()));
        }

        // 软删除私聊
        self.repo.soft_delete(private_chat_id).await?;

        Ok(())
    }

    /// 更新私聊设置
    pub async fn update_settings(
        &self,
        private_chat_id: Uuid,
        user_id: i64,
        is_muted: Option<bool>,
        is_pinned: Option<bool>,
        is_archived: Option<bool>,
    ) -> Result<()> {
        // 检查私聊是否存在
        if !self.repo.exists(private_chat_id).await? {
            return Err(AppError::NotFound("私聊不存在".to_string()));
        }

        // 检查用户是否是私聊成员
        let is_member = self.repo.is_user_in_chat(private_chat_id, user_id).await?;
        if !is_member {
            return Err(AppError::Forbidden("无权修改此私聊设置".to_string()));
        }

        // 更新设置
        self.repo
            .upsert_settings(private_chat_id, user_id, is_muted, is_pinned, is_archived)
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
    ) -> Result<(
        Vec<crate::dto::response::PrivateChatListItem>,
        bool,
        Option<String>,
    )> {
        use crate::dto::response::{PrivateChatLastMessage, PrivateChatListItem, UserInfo};

        // 1. 获取私聊列表基础数据
        let (chats, next_cursor) = self
            .list_by_user(current_user_id, limit, cursor, include_archived)
            .await?;

        let has_more = next_cursor.is_some();
        if chats.is_empty() {
            return Ok((Vec::new(), has_more, next_cursor));
        }

        // 2. 收集 ID
        let mut chat_ids = Vec::with_capacity(chats.len());
        let mut other_user_ids = Vec::with_capacity(chats.len());

        for chat in &chats {
            chat_ids.push(chat.private_chat_id);
            let other_id = if chat.user_a_id == current_user_id {
                chat.user_b_id
            } else {
                chat.user_a_id
            };
            other_user_ids.push(other_id);
        }

        // 3. 批量查询（并行执行）
        let (users, unread_counts, last_messages) = tokio::try_join!(
            self.user_repo.find_profiles(&other_user_ids),
            self.unread_repo
                .get_unread_count_batch("private_chat", &chat_ids, current_user_id),
            self.message_repo.get_last_message_batch(&chat_ids)
        )?;

        // 4. 构建映射
        let user_map: std::collections::HashMap<i64, _> =
            users.into_iter().map(|u| (u.user_id, u)).collect();

        // 5. 组装结果
        let mut items = Vec::with_capacity(chats.len());
        for chat in chats {
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
                    user_type: crate::types::enums::UserType::Normal,
                });

            let unread_count = unread_counts
                .get(&chat.private_chat_id)
                .copied()
                .unwrap_or(0);

            let last_message_row = last_messages.get(&chat.private_chat_id);
            let last_message = last_message_row.map(|m| PrivateChatLastMessage {
                id: m.message_id.to_string(),
                content: Some(m.content.clone()),
                r#type: m.r#type.as_str().to_string(),
                sender_id: m.sender_id,
                sent_at: m.sent_at.to_rfc3339(),
            });

            let last_activity_at = last_message
                .as_ref()
                .map(|m| m.sent_at.clone())
                .unwrap_or_else(|| chat.created_at.to_rfc3339());

            items.push(PrivateChatListItem {
                id: chat.private_chat_id.to_string(),
                other_user: other_user_info,
                unread_count: unread_count as i32,
                last_message,
                last_activity_at,
            });
        }

        Ok((items, has_more, next_cursor))
    }

    /// 获取私聊详情（包含用户信息、消息限制、未读数等）
    pub async fn get_details(
        &self,
        private_chat_id: uuid::Uuid,
        current_user_id: i64,
    ) -> Result<crate::dto::response::PrivateChatDetailData> {
        use crate::dto::response::{
            MessageLimit, PrivateChatDetailData, PrivateChatLastMessage, UserInfo,
        };

        let chat = self
            .find_by_id(private_chat_id)
            .await?
            .ok_or_else(|| AppError::NotFound("私聊不存在".to_string()))?;

        // 获取对方用户信息
        let other_user_id = if chat.user_a_id == current_user_id {
            chat.user_b_id
        } else {
            chat.user_a_id
        };

        let other_user = self.user_repo.find_by_id(other_user_id).await?;
        let other_user_info = other_user
            .map(|u| UserInfo {
                user_id: u.user_id,
                nickname: u.nickname,
                avatar_url: u.avatar_url,
                user_type: u.user_type,
            })
            .unwrap_or_else(|| UserInfo {
                user_id: other_user_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
                user_type: crate::types::enums::UserType::Normal,
            });

        // 获取未读计数
        let unread_count = self
            .unread_repo
            .get_unread_count(
                "private_chat",
                &chat.private_chat_id.to_string(),
                current_user_id,
            )
            .await
            .unwrap_or(0);

        // 获取最后一条消息
        let last_message = self
            .message_repo
            .get_last_message(chat.private_chat_id)
            .await?
            .map(|m| PrivateChatLastMessage {
                id: m.message_id.to_string(),
                content: Some(m.content),
                r#type: m.r#type.as_str().to_string(),
                sender_id: m.sender_id,
                sent_at: m.sent_at.to_rfc3339(),
            });

        let last_activity_at = last_message
            .as_ref()
            .map(|m| m.sent_at.clone())
            .unwrap_or_else(|| chat.created_at.to_rfc3339());

        // 检查消息限制（陌生人消息限制）
        let is_friend = self
            .follows_repo
            .are_mutual_follows(current_user_id, other_user_id)
            .await
            .unwrap_or(false);
        let message_limit = if !is_friend {
            // 从 chat row 中获取已发送消息数和是否可以自由发送
            let (can_message_freely, sent_count) = if chat.user_a_id == current_user_id {
                (chat.user_a_can_message_freely, chat.user_a_sent_count)
            } else {
                (chat.user_b_can_message_freely, chat.user_b_sent_count)
            };

            if !can_message_freely {
                let total = if self
                    .follows_repo
                    .exists(current_user_id, other_user_id)
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
        let settings = self
            .repo
            .get_settings(private_chat_id, current_user_id)
            .await?;

        Ok(PrivateChatDetailData {
            id: chat.private_chat_id.to_string(),
            other_user: other_user_info,
            message_limit,
            unread_count: unread_count as i32,
            is_muted: settings.as_ref().map(|s| s.is_muted).unwrap_or(false),
            is_pinned: settings.as_ref().map(|s| s.is_pinned).unwrap_or(false),
            is_archived: settings.as_ref().map(|s| s.is_archived).unwrap_or(false),
            last_message,
            last_activity_at,
            created_at: chat.created_at.to_rfc3339(),
        })
    }
}
