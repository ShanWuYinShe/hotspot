use redis::aio::ConnectionManager;
use sqlx::PgPool;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    events::event_types::AppEvent,
    services::subscription_service::SubscriptionService,
    repositories::user_repo::UserRepository,
    repositories::{group_member_repo::GroupMemberRepository},
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        subscription::SubscriptionManager,
        types::{GroupUpdatedFields, WsMessage},
    },
};

/// 事件处理器
///
/// 注：通知是实时推送机制，不存储。通过 WebSocket 推送给在线用户。
/// 冷启动通过各业务 API 获取待处理数据。
pub struct EventHandler {
    ws_manager: WsConnectionManager,
    ws_subscription: SubscriptionManager,
    user_repo: UserRepository,
    subscription_service: SubscriptionService,
    group_member_repo: GroupMemberRepository,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        _redis: ConnectionManager,
        ws_manager: WsConnectionManager,
        ws_subscription: SubscriptionManager,
    ) -> Self {
        let user_repo = UserRepository::new(db.clone());
        let subscription_service = SubscriptionService::new(db.clone());
        let group_member_repo = GroupMemberRepository::new(db);

        Self {
            user_repo,
            ws_manager,
            ws_subscription,
            subscription_service,
            group_member_repo,
        }
    }

    /// 获取用户信息
    async fn get_user_info(&self, user_id: i64) -> crate::websocket::types::UserInfo {
        match self.user_repo.find_by_id(user_id).await {
            Ok(Some(user)) => crate::websocket::types::UserInfo {
                user_id: user.user_id,
                nickname: user.nickname,
                avatar_url: user.avatar_url,
                user_type: user.user_type,
            },
            _ => crate::websocket::types::UserInfo {
                user_id,
                nickname: "未知用户".to_string(),
                avatar_url: None,
                user_type: crate::types::enums::UserType::Normal,
            },
        }
    }

    /// 启动事件处理循环
    pub async fn start(self, mut receiver: mpsc::UnboundedReceiver<AppEvent>) {
        info!("Event handler started");

        while let Some(event) = receiver.recv().await {
            if let Err(e) = self.handle_event(event.clone()).await {
                error!("Failed to handle event {:?}: {}", event, e);
            }
        }

        info!("Event handler stopped");
    }

    /// 处理单个事件
    async fn handle_event(&self, event: AppEvent) -> Result<(), Box<dyn std::error::Error>> {
        info!("Handling event: {}", event.event_type());

        // 发送WebSocket推送
        self.send_websocket_notifications(&event).await?;

        Ok(())
    }

    /// 发送WebSocket推送
    async fn send_websocket_notifications(
        &self,
        event: &AppEvent,
    ) -> Result<(), Box<dyn std::error::Error>> {
        match event {
            // ========== 用户相关事件 ==========
            AppEvent::UserRegistered { .. } => {
                // 注册事件不需要 WebSocket 推送
            }
            AppEvent::UserProfileUpdated { .. } => {
                // 资料更新事件，订阅者可以通过 API 获取
            }
            AppEvent::UserLocationUpdated {
                user_id,
                latitude,
                longitude,
                location_name,
            } => {
                let message = WsMessage::UserLocationUpdate {
                    user_id: *user_id,
                    latitude: *latitude,
                    longitude: *longitude,
                    location_name: location_name.clone(),
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                // 推送给订阅了该用户状态的用户
                let user_id_str = user_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_user_status_subscribers(&user_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }

            // ========== 关注关系事件 ==========
            AppEvent::NewFollower {
                user_id,
                follower_id,
                is_mutual,
            } => {
                // 通知推送 = 持久化偏好 AND 实时订阅（严格按订阅推送）
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "new_follower")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.new_follower_enabled {
                    return Ok(());
                }

                let follower_info = self.get_user_info(*follower_id).await;
                let message = WsMessage::NewFollower {
                    user: follower_info,
                    followed_at: chrono::Utc::now().to_rfc3339(),
                    is_mutual: *is_mutual,
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::MutualFollow {
                user_id,
                mutual_user_id,
            } => {
                // mutual_follow：两端都各自判断自己是否订阅/启用
                let user_info = self.get_user_info(*user_id).await;
                let mutual_user_info = self.get_user_info(*mutual_user_id).await;
                let now = chrono::Utc::now().to_rfc3339();

                if self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "mutual_follow")
                    .await
                {
                    let cfg = self.subscription_service.get_config(*user_id).await?;
                    if cfg.mutual_follow_enabled {
                        let message_for_user = WsMessage::MutualFollow {
                            user: mutual_user_info.clone(),
                            mutual_at: now.clone(),
                        };
                        let _ = self.ws_manager.send_to_user(*user_id, &message_for_user).await;
                    }
                }

                if self
                    .ws_subscription
                    .has_notification_subscription(*mutual_user_id, "mutual_follow")
                    .await
                {
                    let cfg = self.subscription_service.get_config(*mutual_user_id).await?;
                    if cfg.mutual_follow_enabled {
                        let message_for_mutual = WsMessage::MutualFollow {
                            user: user_info,
                            mutual_at: now,
                        };
                        let _ = self
                            .ws_manager
                            .send_to_user(*mutual_user_id, &message_for_mutual)
                            .await;
                    }
                }
            }
            AppEvent::Unfollowed {
                follower_id,
                user_id,
                was_mutual,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "unfollowed")
                    .await
                {
                    return Ok(());
                }
                // 取消关注：目前偏好表里没有单独的 unfollowed 开关，复用 mutual_follow_enabled 的语义（好友关系变化）
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.mutual_follow_enabled {
                    return Ok(());
                }

                let message = WsMessage::Unfollowed {
                    user_id: *follower_id,
                    unfollowed_at: chrono::Utc::now().to_rfc3339(),
                    was_mutual: *was_mutual,
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }

            // ========== 群组管理事件 ==========
            AppEvent::GroupCreated {
                group_id,
                creator_id,
            } => {
                tracing::debug!(
                    "群组已创建: group_id={}, creator_id={}",
                    group_id,
                    creator_id
                );
            }
            AppEvent::GroupMemberAdded {
                group_id, user_id, ..
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::GroupMemberJoined {
                    group_chat_id: group_id.to_string(),
                    user,
                    joined_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMemberRemoved {
                group_id, user_id, ..
            } => {
                let message = WsMessage::GroupMemberLeft {
                    group_chat_id: group_id.to_string(),
                    user_id: *user_id,
                    left_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::GroupProfileUpdated {
                group_id,
                name,
                description,
                avatar_url,
            } => {
                let message = WsMessage::GroupProfileUpdated {
                    group_chat_id: group_id.to_string(),
                    updated_fields: GroupUpdatedFields {
                        name: name.clone(),
                        description: description.clone(),
                        avatar_url: avatar_url.clone(),
                        settings: None,
                    },
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMemberRoleChanged {
                group_id,
                user_id,
                old_role,
                new_role,
            } => {
                let message = WsMessage::GroupMemberRoleChanged {
                    group_chat_id: group_id.to_string(),
                    user_id: *user_id,
                    old_role: old_role.clone(),
                    new_role: new_role.clone(),
                    changed_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMemberMuted {
                group_id,
                user_id,
                muted_until,
            } => {
                let message = WsMessage::GroupMemberMuted {
                    group_chat_id: group_id.to_string(),
                    user_id: *user_id,
                    muted_until: muted_until.clone(),
                    muted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMemberUnmuted { group_id, user_id } => {
                let message = WsMessage::GroupMemberUnmuted {
                    group_chat_id: group_id.to_string(),
                    user_id: *user_id,
                    unmuted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupDismissed {
                group_id,
                dismissed_by,
            } => {
                let message = WsMessage::GroupDismissed {
                    group_chat_id: group_id.to_string(),
                    dismissed_by: *dismissed_by,
                    dismissed_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupInviteSent {
                invite_id,
                group_id,
                group_name,
                from_user_id,
                to_user_id,
                message,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*to_user_id, "group_invite")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*to_user_id).await?;
                if !cfg.group_invite_enabled {
                    return Ok(());
                }

                let inviter = self.get_user_info(*from_user_id).await;
                let ws_message = WsMessage::GroupInviteReceived {
                    id: invite_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    group_name: group_name.clone(),
                    inviter,
                    message: message.clone(),
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*to_user_id, &ws_message).await;
            }
            AppEvent::GroupInviteResponse {
                invite_id,
                group_id,
                inviter_id,
                invitee_id,
                status,
            } => {
                // 邀请响应只通知邀请人（通知类事件，按订阅与偏好）
                if !self
                    .ws_subscription
                    .has_notification_subscription(*inviter_id, "group_invite_response")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*inviter_id).await?;
                if !cfg.group_invite_response_enabled {
                    return Ok(());
                }

                let user = self.get_user_info(*invitee_id).await;
                let message = WsMessage::GroupInviteResponse {
                    id: invite_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    user,
                    status: status.clone(),
                    responded_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*inviter_id, &message).await;
            }
            AppEvent::GroupApplicationReceived {
                application_id,
                group_id,
                applicant_id,
                message,
            } => {
                // 群组申请：通知该群组的 Owner/Admin（并且他们自己订阅 + 偏好启用）
                let applicant = self.get_user_info(*applicant_id).await;
                let ws_message = WsMessage::GroupApplicationReceived {
                    id: application_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    applicant,
                    message: message.clone(),
                    created_at: chrono::Utc::now().to_rfc3339(),
                };

                let admin_ids = self.group_member_repo.list_admin_user_ids(*group_id).await?;
                let mut recipients = Vec::with_capacity(admin_ids.len());
                for admin_id in admin_ids {
                    if !self
                        .ws_subscription
                        .has_notification_subscription(admin_id, "group_application")
                        .await
                    {
                        continue;
                    }
                    let cfg = self.subscription_service.get_config(admin_id).await?;
                    if !cfg.group_application_enabled {
                        continue;
                    }
                    recipients.push(admin_id);
                }

                self.ws_manager.send_to_users(&recipients, &ws_message).await;
            }
            AppEvent::GroupApplicationResponse {
                application_id,
                group_id,
                group_name,
                applicant_id,
                status,
            } => {
                let message = WsMessage::GroupApplicationResponse {
                    id: application_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    group_name: group_name.clone(),
                    status: status.clone(),
                    responded_at: chrono::Utc::now().to_rfc3339(),
                };
                if !self
                    .ws_subscription
                    .has_notification_subscription(*applicant_id, "group_application_response")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*applicant_id).await?;
                if !cfg.group_application_response_enabled {
                    return Ok(());
                }
                let _ = self.ws_manager.send_to_user(*applicant_id, &message).await;
            }

            // ========== 群聊消息事件 ==========
            AppEvent::GroupMessageSent {
                message_id,
                group_chat_id,
                sender_id,
                message_type,
                content,
                reply_to,
                mentions,
                media_url,
                thumbnail_url,
                file_name,
                file_size,
                duration,
                width,
                height,
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::ChatMessageNew {
                    id: message_id.to_string(),
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    reply_to: reply_to.map(|id| id.to_string()),
                    mentions: mentions.clone(),
                    media_url: media_url.clone(),
                    thumbnail_url: thumbnail_url.clone(),
                    file_name: file_name.clone(),
                    file_size: *file_size,
                    duration: *duration,
                    width: *width,
                    height: *height,
                    sent_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMessageEdited {
                message_id,
                group_chat_id,
                content,
                edited_at,
            } => {
                let message = WsMessage::ChatMessageEdited {
                    id: message_id.to_string(),
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    content: content.clone(),
                    edited_at: edited_at.clone(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMessageRecalled {
                message_id,
                group_chat_id,
                recalled_by,
                recalled_at,
            } => {
                let message = WsMessage::ChatMessageRecalled {
                    id: message_id.to_string(),
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    recalled_by: *recalled_by,
                    recalled_at: recalled_at.clone(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMessageDeleted {
                message_id,
                group_chat_id,
                deleted_by,
                for_everyone,
            } => {
                let message = WsMessage::ChatMessageDeleted {
                    id: message_id.to_string(),
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    deleted_by: *deleted_by,
                    for_everyone: *for_everyone,
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMessageRead {
                group_chat_id,
                reader_id,
                up_to,
                read_at,
            } => {
                let message = WsMessage::ChatMessageRead {
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    up_to: up_to.to_string(),
                    reader_id: *reader_id,
                    read_at: read_at.clone(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMessageReaction {
                message_id,
                group_chat_id,
                user_id,
                reaction_type,
                action,
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::ChatMessageReaction {
                    id: message_id.to_string(),
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    user,
                    reaction_type: reaction_type.clone(),
                    has_reaction: action == "add",
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupUserTyping {
                group_chat_id,
                user_id,
                is_typing,
            } => {
                let user_info = self.get_user_info(*user_id).await;
                let message = WsMessage::UserTyping {
                    chat_type: "group_chat".to_string(),
                    chat_id: group_chat_id.to_string(),
                    user_id: *user_id,
                    nickname: user_info.nickname,
                    is_typing: *is_typing,
                };
                let group_id_str = group_chat_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                let filtered_subscribers: Vec<i64> = subscribers
                    .into_iter()
                    .filter(|&id| id != *user_id)
                    .collect();
                self.ws_manager
                    .send_to_users(&filtered_subscribers, &message)
                    .await;
            }

            // ========== 私聊消息事件 ==========
            AppEvent::PrivateMessageSent {
                message_id,
                private_chat_id,
                sender_id,
                recipient_id,
                message_type,
                content,
                reply_to,
                media_url,
                thumbnail_url,
                file_name,
                file_size,
                duration,
                width,
                height,
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::ChatMessageNew {
                    id: message_id.to_string(),
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    reply_to: reply_to.map(|id| id.to_string()),
                    mentions: None,
                    media_url: media_url.clone(),
                    thumbnail_url: thumbnail_url.clone(),
                    file_name: file_name.clone(),
                    file_size: *file_size,
                    duration: *duration,
                    width: *width,
                    height: *height,
                    sent_at: chrono::Utc::now().to_rfc3339(),
                };
                // 私聊消息直接推送给对方（不需要订阅）
                // 同时也推送给发送者自己（多端同步 + 前端以 WS 推送为准上屏）
                self.ws_manager
                    .send_to_users(&[*recipient_id, *sender_id], &message)
                    .await;
            }
            AppEvent::PrivateMessageEdited {
                message_id,
                private_chat_id,
                sender_id,
                recipient_id,
                content,
                edited_at,
            } => {
                let message = WsMessage::ChatMessageEdited {
                    id: message_id.to_string(),
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    content: content.clone(),
                    edited_at: edited_at.clone(),
                };
                // 发送给接收者
                let _ = self.ws_manager.send_to_user(*recipient_id, &message).await;
                // 发送给发送者（多端同步）
                let _ = self.ws_manager.send_to_user(*sender_id, &message).await;
            }
            AppEvent::PrivateMessageRecalled {
                message_id,
                private_chat_id,
                sender_id,
                recipient_id,
                recalled_by,
                recalled_at,
            } => {
                let message = WsMessage::ChatMessageRecalled {
                    id: message_id.to_string(),
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    recalled_by: *recalled_by,
                    recalled_at: recalled_at.clone(),
                };
                // 发送给接收者
                let _ = self.ws_manager.send_to_user(*recipient_id, &message).await;
                // 发送给发送者（多端同步）
                let _ = self.ws_manager.send_to_user(*sender_id, &message).await;
            }
            AppEvent::PrivateMessageDeleted {
                message_id,
                private_chat_id,
                deleted_by,
            } => {
                let message = WsMessage::ChatMessageDeleted {
                    id: message_id.to_string(),
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    deleted_by: *deleted_by,
                    for_everyone: false,
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                // 私聊删除只影响自己，不需要推送
                tracing::debug!("私聊消息已删除: message_id={}", message_id);
                let _ = message; // 消除警告
            }
            AppEvent::PrivateMessageRead {
                private_chat_id,
                sender_id,
                reader_id,
                up_to,
                read_at,
            } => {
                let message = WsMessage::ChatMessageRead {
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    up_to: up_to.to_string(),
                    reader_id: *reader_id,
                    read_at: read_at.clone(),
                };
                // 推送给消息发送者，告知已读
                let _ = self.ws_manager.send_to_user(*sender_id, &message).await;
            }
            AppEvent::PrivateMessageReaction {
                message_id,
                private_chat_id,
                other_user_id,
                user_id,
                reaction_type,
                action,
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::ChatMessageReaction {
                    id: message_id.to_string(),
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    user,
                    reaction_type: reaction_type.clone(),
                    has_reaction: action == "add",
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*other_user_id, &message).await;
            }
            AppEvent::PrivateUserTyping {
                private_chat_id,
                other_user_id,
                user_id,
                is_typing,
            } => {
                let user_info = self.get_user_info(*user_id).await;
                let message = WsMessage::UserTyping {
                    chat_type: "private_chat".to_string(),
                    chat_id: private_chat_id.to_string(),
                    user_id: *user_id,
                    nickname: user_info.nickname,
                    is_typing: *is_typing,
                };
                let _ = self.ws_manager.send_to_user(*other_user_id, &message).await;
            }

            // ========== 瞬间相关事件 ==========
            AppEvent::MomentCreated { moment_id, user_id } => {
                tracing::debug!("瞬间已创建: moment_id={}, user_id={}", moment_id, user_id);
            }
            AppEvent::MomentLiked {
                moment_id,
                user_id,
                liker_id,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "moment_like")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.moment_like_enabled {
                    return Ok(());
                }

                let liker = self.get_user_info(*liker_id).await;
                let message = WsMessage::MomentLiked {
                    moment_id: moment_id.to_string(),
                    user: liker,
                    liked_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::MomentCommented {
                moment_id,
                comment_id,
                user_id,
                commenter_id,
                content,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "moment_comment")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.moment_comment_enabled {
                    return Ok(());
                }

                let commenter = self.get_user_info(*commenter_id).await;
                let message = WsMessage::MomentCommented {
                    moment_id: moment_id.to_string(),
                    comment_id: comment_id.to_string(),
                    user: commenter,
                    content: content.clone(),
                    commented_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }

            // ========== 评论相关事件 ==========
            AppEvent::CommentAdded {
                comment_id,
                target_type,
                target_id,
                user_id,
            } => {
                tracing::debug!(
                    "新评论: comment_id={}, target_type={}, target_id={}, user_id={}",
                    comment_id,
                    target_type,
                    target_id,
                    user_id
                );
            }
            AppEvent::CommentLiked {
                comment_id,
                user_id,
                liker_id,
            } => {
                tracing::debug!(
                    "评论被点赞: comment_id={}, user_id={}, liker_id={}",
                    comment_id,
                    user_id,
                    liker_id
                );
            }
            AppEvent::CommentReplied {
                moment_id,
                comment_id,
                reply_id,
                user_id,
                replier_id,
                content,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "comment_reply")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.comment_reply_enabled {
                    return Ok(());
                }

                let replier = self.get_user_info(*replier_id).await;
                let message = WsMessage::CommentReply {
                    moment_id: moment_id.to_string(),
                    comment_id: comment_id.to_string(),
                    reply_id: reply_id.to_string(),
                    user: replier,
                    content: content.clone(),
                    replied_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }

            // ========== 系统事件 ==========
            AppEvent::SystemNotification {
                user_id,
                title,
                content,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*user_id, "system_announcement")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*user_id).await?;
                if !cfg.system_announcement_enabled {
                    return Ok(());
                }

                let message = WsMessage::SystemAnnouncement {
                    id: uuid::Uuid::new_v4().to_string(),
                    title: title.clone(),
                    content: content.clone(),
                    severity: "info".to_string(),
                    action_url: None,
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
        }

        Ok(())
    }
}
