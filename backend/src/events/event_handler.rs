use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    dto::response::Location,
    events::event_types::AppEvent,
    repositories::group_member_repo::GroupMemberRepository,
    repositories::notification_repo::NotificationType,
    repositories::user_repo::UserRepository,
    services::notification_service::NotificationService,
    services::subscription_service::SubscriptionService,
    types::enums::{AnnouncementSeverity, FollowRequestAction},
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        subscription::SubscriptionManager,
        types::{GroupUpdatedFields, WsMessage},
    },
    EventBus,
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
    notification_service: NotificationService,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        ws_manager: WsConnectionManager,
        ws_subscription: SubscriptionManager,
        event_bus: Arc<EventBus>,
        media_service: crate::services::media_service::MediaService,
    ) -> Self {
        let user_repo = UserRepository::new(db.clone());
        let subscription_service = SubscriptionService::new(db.clone(), redis.clone());
        let group_member_repo = GroupMemberRepository::new(db.clone(), redis.clone());
        let notification_repo =
            crate::repositories::notification_repo::NotificationRepository::new(db.clone());
        let notification_service = NotificationService::new(
            notification_repo,
            user_repo.clone(),
            event_bus,
            media_service,
        );

        Self {
            user_repo,
            ws_manager,
            ws_subscription,
            subscription_service,
            group_member_repo,
            notification_service,
        }
    }

    /// 获取用户信息
    async fn get_user_info(&self, user_id: i64) -> crate::dto::response::UserInfo {
        match self.user_repo.find_by_id(user_id).await {
            Ok(Some(user)) => crate::dto::response::UserInfo {
                user_id: user.user_id,
                nickname: user.nickname,
                avatar_url: user.avatar_url,
                user_type: user.user_type,
            },
            _ => crate::dto::response::UserInfo {
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

        // 持久化通知 (Inbox)
        if let Err(e) = self.persist_notification(&event).await {
            error!("Failed to persist notification: {}", e);
            // 继续执行推送，不因为持久化失败阻断实时通知
        }

        // 发送WebSocket推送
        self.send_websocket_notifications(&event).await?;

        Ok(())
    }

    /// 持久化通知
    async fn persist_notification(
        &self,
        event: &AppEvent,
    ) -> Result<(), Box<dyn std::error::Error>> {
        match event {
            AppEvent::NewFollower {
                user_id,
                follower_id,
                ..
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::Follow,
                        "新增关注".to_string(),
                        "关注了你".to_string(),
                        Some(*follower_id),
                        Some("user".to_string()),
                        Some(follower_id.to_string()),
                        None,
                        None,
                    )
                    .await?;
            }
            AppEvent::MomentLiked {
                moment_id,
                moment_summary,
                user_id,
                liker_id,
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::Like,
                        "瞬间获赞".to_string(),
                        "点赞了你的瞬间".to_string(),
                        Some(*liker_id),
                        Some("moment".to_string()),
                        Some(moment_id.to_string()),
                        None,
                        Some(
                            moment_summary
                                .cover_url
                                .clone()
                                .unwrap_or(moment_summary.text_preview.clone()),
                        ),
                    )
                    .await?;
            }
            AppEvent::MomentCommented {
                moment_id,
                moment_summary,
                comment_id,
                user_id,
                commenter_id,
                content,
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::Comment,
                        "瞬间评论".to_string(),
                        content.clone(),
                        Some(*commenter_id),
                        Some("moment".to_string()),
                        Some(moment_id.to_string()),
                        Some(comment_id.to_string()),
                        Some(
                            moment_summary
                                .cover_url
                                .clone()
                                .unwrap_or(moment_summary.text_preview.clone()),
                        ),
                    )
                    .await?;
            }
            AppEvent::CommentReplied {
                moment_id,
                comment_id: _,
                reply_to_content,
                reply_id,
                user_id,
                replier_id,
                content,
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::Reply,
                        "收到回复".to_string(),
                        content.clone(),
                        Some(*replier_id),
                        Some("moment".to_string()),
                        Some(moment_id.to_string()), // 跳转到动态详情
                        Some(reply_id.to_string()),
                        Some(reply_to_content.clone()), // 显示原评论摘要
                    )
                    .await?;
            }
            AppEvent::GroupInviteSent {
                invite_id,
                group_id,
                group_name,
                from_user_id,
                to_user_id,
                ..
            } => {
                self.notification_service
                    .create_notification(
                        *to_user_id,
                        NotificationType::GroupInvite,
                        "群组邀请".to_string(),
                        format!("邀请你加入群组: {}", group_name),
                        Some(*from_user_id),
                        Some("group".to_string()),
                        Some(group_id.to_string()),
                        Some(invite_id.to_string()),
                        None,
                    )
                    .await?;
            }
            AppEvent::GroupApplicationReceived {
                application_id,
                group_id,
                applicant_id,
                message,
            } => {
                // 需要通知所有管理员
                let admin_ids = self
                    .group_member_repo
                    .list_admin_user_ids(*group_id)
                    .await?;
                for admin_id in admin_ids {
                    self.notification_service
                        .create_notification(
                            admin_id,
                            NotificationType::GroupApply,
                            "入群申请".to_string(),
                            message
                                .clone()
                                .unwrap_or_else(|| "申请加入群组".to_string()),
                            Some(*applicant_id),
                            Some("group".to_string()),
                            Some(group_id.to_string()),
                            Some(application_id.to_string()),
                            None,
                        )
                        .await?;
                }
            }
            AppEvent::SystemNotification {
                user_id,
                title,
                content,
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::System,
                        title.clone(),
                        content.clone(),
                        None, // System has no sender
                        None,
                        None,
                        None,
                        None,
                    )
                    .await?;
            }
            AppEvent::FollowRequestReceived {
                request_id,
                requester_id,
                target_id,
            } => {
                // 创建关注请求通知
                self.notification_service
                    .create_notification(
                        *target_id,
                        NotificationType::Follow,
                        "新的关注请求".to_string(),
                        "请求关注你".to_string(),
                        Some(*requester_id),
                        Some("follow_request".to_string()),
                        Some(request_id.clone()),
                        None,
                        None,
                    )
                    .await?;
            }
            AppEvent::FollowRequestHandled {
                request_id,
                requester_id,
                target_id,
                accepted,
            } => {
                // 创建关注请求处理通知
                let title = if *accepted {
                    "关注请求已通过"
                } else {
                    "关注请求被拒绝"
                };
                let content = if *accepted {
                    "已同意你的关注请求"
                } else {
                    "拒绝了你的关注请求"
                };
                self.notification_service
                    .create_notification(
                        *requester_id,
                        NotificationType::Follow,
                        title.to_string(),
                        content.to_string(),
                        Some(*target_id),
                        Some("follow_request".to_string()),
                        Some(request_id.clone()),
                        None,
                        None,
                    )
                    .await?;
            }
            _ => {}
        }
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
            AppEvent::UserProfileUpdated {
                user_id,
                nickname,
                avatar_url,
                bio,
            } => {
                let message = WsMessage::UserProfileUpdated {
                    user_id: *user_id,
                    nickname: nickname.clone(),
                    avatar_url: avatar_url.clone(),
                    bio: bio.clone(),
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                // 推送给订阅了该用户状态的用户
                let subscribers = self
                    .ws_subscription
                    .get_user_status_subscribers(*user_id)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::UserLocationUpdated {
                user_id,
                latitude,
                longitude,
                location_name,
            } => {
                let message = WsMessage::UserLocationUpdate {
                    user_id: *user_id,
                    location: Location {
                        latitude: *latitude,
                        longitude: *longitude,
                        location_name: location_name.clone(),
                    },
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                // 推送给订阅了该用户状态的用户
                let subscribers = self
                    .ws_subscription
                    .get_user_status_subscribers(*user_id)
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
                        let _ = self
                            .ws_manager
                            .send_to_user(*user_id, &message_for_user)
                            .await;
                    }
                }

                if self
                    .ws_subscription
                    .has_notification_subscription(*mutual_user_id, "mutual_follow")
                    .await
                {
                    let cfg = self
                        .subscription_service
                        .get_config(*mutual_user_id)
                        .await?;
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
            AppEvent::FollowRequestReceived {
                request_id,
                requester_id,
                target_id,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*target_id, "follow_request")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*target_id).await?;
                if !cfg.follow_request_enabled {
                    return Ok(());
                }
                let requester = self.get_user_info(*requester_id).await;
                let message = WsMessage::FollowRequestReceived {
                    request_id: request_id.clone(),
                    requester,
                    message: None,
                    requested_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*target_id, &message).await;
            }
            AppEvent::FollowRequestHandled {
                request_id,
                requester_id,
                target_id,
                accepted,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*requester_id, "follow_request_handled")
                    .await
                {
                    return Ok(());
                }
                let cfg = self.subscription_service.get_config(*requester_id).await?;
                if !cfg.follow_request_handled_enabled {
                    return Ok(());
                }
                let target_user = self.get_user_info(*target_id).await;
                let message = WsMessage::FollowRequestHandled {
                    request_id: request_id.clone(),
                    target_user,
                    action: if *accepted {
                        FollowRequestAction::Accept
                    } else {
                        FollowRequestAction::Reject
                    },
                    handled_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self
                    .ws_manager
                    .send_to_user(*requester_id, &message)
                    .await;
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
                group_id,
                user_id,
                role,
                member_count,
                ..
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::GroupMemberJoined {
                    group_chat_id: group_id.to_string(),
                    user,
                    role: role.clone(),
                    member_count: *member_count as i32,
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
                group_id,
                user_id,
                member_count,
                reason,
                ..
            } => {
                let message = WsMessage::GroupMemberLeft {
                    group_chat_id: group_id.to_string(),
                    user_id: *user_id,
                    member_count: *member_count as i32,
                    reason: reason.clone(),
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
                group_avatar_url,
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
                    invite_id: invite_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    group_name: group_name.clone(),
                    group_avatar_url: group_avatar_url.clone(),
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
                    invite_id: invite_id.to_string(),
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
                    application_id: application_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    applicant,
                    message: message.clone(),
                    created_at: chrono::Utc::now().to_rfc3339(),
                };

                let admin_ids = self
                    .group_member_repo
                    .list_admin_user_ids(*group_id)
                    .await?;
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

                self.ws_manager
                    .send_to_users(&recipients, &ws_message)
                    .await;
            }
            AppEvent::GroupApplicationResponse {
                application_id,
                group_id,
                group_name,
                applicant_id,
                status,
            } => {
                let message = WsMessage::GroupApplicationResponse {
                    application_id: application_id.to_string(),
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
                media,
                location,
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::GroupMessageNew {
                    message_id: message_id.to_string(),
                    group_chat_id: group_chat_id.to_string(),
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    media: media.clone(),
                    location: location.clone(),
                    reply_to: reply_to
                        .clone()
                        .map(|r| crate::websocket::types::MessageReplyTo {
                            message_id: r.message_id.to_string(),
                            user_id: r.user_id,
                            nickname: r.nickname,
                            content: r.content,
                        }),
                    mentions: mentions.clone(),
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
                let message = WsMessage::GroupMessageEdited {
                    message_id: message_id.to_string(),
                    group_chat_id: group_chat_id.to_string(),
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
                let message = WsMessage::GroupMessageRecalled {
                    message_id: message_id.to_string(),
                    group_chat_id: group_chat_id.to_string(),
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
                // 仅当 for_everyone=true 时推送删除事件（否则仅影响自己，且后端不推送个人删除）
                if *for_everyone {
                    let message = WsMessage::GroupMessageDeleted {
                        message_id: message_id.to_string(),
                        group_chat_id: group_chat_id.to_string(),
                        deleted_by: *deleted_by,
                        deleted_at: chrono::Utc::now().to_rfc3339(),
                    };
                    let group_id_str = group_chat_id.to_string();
                    let subscribers = self
                        .ws_subscription
                        .get_group_chat_subscribers(&group_id_str)
                        .await;
                    self.ws_manager.send_to_users(&subscribers, &message).await;
                }
            }
            AppEvent::GroupMessageRead {
                group_chat_id,
                reader_id,
                up_to,
                read_at,
            } => {
                let message = WsMessage::GroupMessageRead {
                    group_chat_id: group_chat_id.to_string(),
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
                let message = WsMessage::GroupMessageReaction {
                    message_id: message_id.to_string(),
                    group_chat_id: group_chat_id.to_string(),
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
                let message = WsMessage::GroupUserTyping {
                    group_chat_id: group_chat_id.to_string(),
                    user_id: *user_id,
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
                sender_id,
                target_user_id,
                message_type,
                content,
                reply_to,
                media,
                location,
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::PrivateMessageNew {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    media: media.clone(),
                    location: location.clone(),
                    reply_to: reply_to
                        .clone()
                        .map(|r| crate::websocket::types::MessageReplyTo {
                            message_id: r.message_id.to_string(),
                            user_id: r.user_id,
                            nickname: r.nickname,
                            content: r.content,
                        }),
                    sent_at: chrono::Utc::now().to_rfc3339(),
                };
                // 私聊消息直接推送给对方（不需要订阅）
                // 同时也推送给发送者自己（多端同步 + 前端以 WS 推送为准上屏）
                self.ws_manager
                    .send_to_users(&[*target_user_id, *sender_id], &message)
                    .await;
            }
            AppEvent::PrivateMessageEdited {
                message_id,
                sender_id,
                target_user_id,
                content,
                edited_at,
            } => {
                let message = WsMessage::PrivateMessageEdited {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    content: content.clone(),
                    edited_at: edited_at.clone(),
                };
                // 发送给接收者
                let _ = self
                    .ws_manager
                    .send_to_user(*target_user_id, &message)
                    .await;
                // 发送给发送者（多端同步）
                let _ = self.ws_manager.send_to_user(*sender_id, &message).await;
            }
            AppEvent::PrivateMessageRecalled {
                message_id,
                sender_id,
                target_user_id,
                recalled_by,
                recalled_at,
            } => {
                let message = WsMessage::PrivateMessageRecalled {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    recalled_by: *recalled_by,
                    recalled_at: recalled_at.clone(),
                };
                // 发送给接收者
                let _ = self
                    .ws_manager
                    .send_to_user(*target_user_id, &message)
                    .await;
                // 发送给发送者（多端同步）
                let _ = self.ws_manager.send_to_user(*sender_id, &message).await;
            }
            AppEvent::PrivateMessageDeleted {
                message_id,
                target_user_id,
                deleted_by,
            } => {
                let message = WsMessage::PrivateMessageDeleted {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    deleted_by: *deleted_by,
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                // 私聊删除只影响自己，不需要推送给对方
                // 但需要推送给自己的其他设备（多端同步）
                let _ = self.ws_manager.send_to_user(*deleted_by, &message).await;
            }
            AppEvent::PrivateMessageRead {
                target_user_id,
                reader_id,
                up_to,
                read_at,
            } => {
                let message = WsMessage::PrivateMessageRead {
                    target_user_id: *target_user_id,
                    up_to: up_to.to_string(),
                    reader_id: *reader_id,
                    read_at: read_at.clone(),
                };
                // 推送给消息发送者 (target_user_id)，告知已读
                let _ = self
                    .ws_manager
                    .send_to_user(*target_user_id, &message)
                    .await;
                // 推送给读者自己（多端同步未读状态）
                let _ = self.ws_manager.send_to_user(*reader_id, &message).await;
            }
            AppEvent::PrivateMessageReaction {
                message_id,
                target_user_id,
                user_id,
                reaction_type,
                action,
                ..
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::PrivateMessageReaction {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    user,
                    reaction_type: reaction_type.clone(),
                    has_reaction: action == "add",
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self
                    .ws_manager
                    .send_to_user(*target_user_id, &message)
                    .await;
                // 也推送给自己（多端同步）
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::PrivateUserTyping {
                target_user_id,
                user_id,
                is_typing,
                ..
            } => {
                let message = WsMessage::PrivateUserTyping {
                    target_user_id: *target_user_id,
                    user_id: *user_id,
                    is_typing: *is_typing,
                };
                let _ = self
                    .ws_manager
                    .send_to_user(*target_user_id, &message)
                    .await;
            }

            // ========== 多端同步事件 ==========
            // 注：SelfJoinedGroup 和 SelfLeftGroup 通过 GroupMemberJoined/Left 事件处理
            // 客户端通过判断 user_id 是否是自己来识别多端同步场景
            AppEvent::SelfJoinedGroup { .. } => {
                // 由 GroupMemberAdded 事件统一处理
            }
            AppEvent::SelfLeftGroup { .. } => {
                // 由 GroupMemberRemoved 事件统一处理
            }
            AppEvent::PrivateChatDeleted {
                user_id,
                target_user_id,
            } => {
                let message = WsMessage::PrivateChatDeleted {
                    target_user_id: *target_user_id,
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::PrivateChatSettingsUpdated {
                user_id,
                target_user_id,
                is_muted,
                is_pinned,
            } => {
                let message = WsMessage::PrivateChatSettingsUpdated {
                    target_user_id: *target_user_id,
                    is_muted: *is_muted,
                    is_pinned: *is_pinned,
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::NotificationReadStateUpdated {
                user_id,
                total_unread_count,
            } => {
                let message = WsMessage::NotificationReadStateUpdated {
                    total_unread_count: *total_unread_count,
                    read_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }

            // ========== 瞬间相关事件 ==========
            AppEvent::MomentCreated { moment_id, user_id } => {
                tracing::debug!("瞬间已创建: moment_id={}, user_id={}", moment_id, user_id);
            }
            AppEvent::MomentLiked {
                moment_id,
                moment_summary,
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
                    moment_summary: crate::websocket::types::MomentSummary {
                        moment_id: moment_summary.moment_id.to_string(),
                        cover_url: moment_summary.cover_url.clone(),
                        text_preview: moment_summary.text_preview.clone(),
                    },
                    user: liker,
                    liked_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::MomentCommented {
                moment_id,
                moment_summary,
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
                    moment_summary: crate::websocket::types::MomentSummary {
                        moment_id: moment_summary.moment_id.to_string(),
                        cover_url: moment_summary.cover_url.clone(),
                        text_preview: moment_summary.text_preview.clone(),
                    },
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
                reply_to_content,
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
                    reply_to_content: reply_to_content.clone(),
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
                    announcement_id: uuid::Uuid::new_v4().to_string(),
                    title: title.clone(),
                    content: content.clone(),
                    severity: AnnouncementSeverity::Info,
                    action_url: None,
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::SystemAnnouncement {
                announcement_id,
                title,
                content,
                severity,
                action_url,
                created_at,
            } => {
                // 系统公告推送给所有订阅了 system_announcement 的用户
                let message = WsMessage::SystemAnnouncement {
                    announcement_id: announcement_id.clone(),
                    title: title.clone(),
                    content: content.clone(),
                    severity: *severity,
                    action_url: action_url.clone(),
                    created_at: created_at.clone(),
                };
                // 系统公告需要推送给所有在线用户（如果他们订阅了 system_announcement）
                // 这里简化处理：通过 ws_manager 广播给所有连接
                // 实际实现中，ws_manager 应该检查用户的订阅状态
                let _ = self.ws_manager.broadcast(&message).await;
            }

            // ========== 群公告事件 ==========
            AppEvent::GroupAnnouncementCreated {
                announcement_id,
                group_id,
                title,
                content,
                created_by_user_id,
                is_pinned,
            } => {
                let created_by = self.get_user_info(*created_by_user_id).await;
                let message = WsMessage::GroupAnnouncementCreated {
                    announcement_id: announcement_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    title: title.clone(),
                    content: content.clone(),
                    created_by,
                    is_pinned: *is_pinned,
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                for uid in subscribers {
                    let cfg = self.subscription_service.get_config(uid).await?;
                    if cfg.group_announcement_enabled {
                        let _ = self.ws_manager.send_to_user(uid, &message).await;
                    }
                }
            }
            AppEvent::GroupAnnouncementUpdated {
                announcement_id,
                group_id,
                title,
                content,
                is_pinned,
            } => {
                let message = WsMessage::GroupAnnouncementUpdated {
                    announcement_id: announcement_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    title: title.clone(),
                    content: content.clone(),
                    is_pinned: *is_pinned,
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                for uid in subscribers {
                    let cfg = self.subscription_service.get_config(uid).await?;
                    if cfg.group_announcement_enabled {
                        let _ = self.ws_manager.send_to_user(uid, &message).await;
                    }
                }
            }
            AppEvent::GroupAnnouncementDeleted {
                announcement_id,
                group_id,
            } => {
                let message = WsMessage::GroupAnnouncementDeleted {
                    announcement_id: announcement_id.to_string(),
                    group_chat_id: group_id.to_string(),
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                for uid in subscribers {
                    let cfg = self.subscription_service.get_config(uid).await?;
                    if cfg.group_announcement_enabled {
                        let _ = self.ws_manager.send_to_user(uid, &message).await;
                    }
                }
            }

            // ========== @ 提及事件 ==========
            AppEvent::MomentMentioned {
                moment_id,
                mentioned_user_id,
                mentioning_user_id,
                moment_text_preview,
                moment_cover_url,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*mentioned_user_id, "message_mention")
                    .await
                {
                    return Ok(());
                }
                let cfg = self
                    .subscription_service
                    .get_config(*mentioned_user_id)
                    .await?;
                if !cfg.message_mention_enabled {
                    return Ok(());
                }
                let user = self.get_user_info(*mentioning_user_id).await;
                let message = WsMessage::MomentMentioned {
                    moment_id: moment_id.to_string(),
                    moment_summary: crate::websocket::types::MomentSummary {
                        moment_id: moment_id.to_string(),
                        cover_url: moment_cover_url.clone(),
                        text_preview: moment_text_preview.clone(),
                    },
                    user,
                    mentioned_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self
                    .ws_manager
                    .send_to_user(*mentioned_user_id, &message)
                    .await;
            }
            AppEvent::CommentMentioned {
                comment_id,
                mentioned_user_id,
                moment_id,
                mentioning_user_id,
                content_preview,
            } => {
                if !self
                    .ws_subscription
                    .has_notification_subscription(*mentioned_user_id, "message_mention")
                    .await
                {
                    return Ok(());
                }
                let cfg = self
                    .subscription_service
                    .get_config(*mentioned_user_id)
                    .await?;
                if !cfg.message_mention_enabled {
                    return Ok(());
                }
                let user = self.get_user_info(*mentioning_user_id).await;
                let message = WsMessage::CommentMentioned {
                    moment_id: moment_id.to_string(),
                    comment_id: comment_id.to_string(),
                    user,
                    content: content_preview.clone(),
                    mentioned_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self
                    .ws_manager
                    .send_to_user(*mentioned_user_id, &message)
                    .await;
            }
        }

        Ok(())
    }
}
