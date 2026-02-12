use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    events::{event_bus::EventBus, event_types::AppEvent},
    repositories::{
        group_member_repo::GroupMemberRepository, notification_repo::NotificationType,
        user_repo::UserRepository,
    },
    services::{
        notification_service::NotificationService, subscription_service::SubscriptionService,
    },
    websocket::{
        manager::ConnectionManager as WsConnectionManager, subscription::SubscriptionManager,
        types::WsMessage,
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
    notification_service: NotificationService,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        ws_manager: WsConnectionManager,
        ws_subscription: SubscriptionManager,
        _event_bus: Arc<EventBus>,
        _media_service: crate::services::media_service::MediaService,
    ) -> Self {
        let user_repo = UserRepository::new(db.clone());
        let subscription_service = SubscriptionService::new(db.clone(), redis.clone());
        let group_member_repo = GroupMemberRepository::new(db.clone(), redis.clone());
        let notification_repo =
            crate::repositories::notification_repo::NotificationRepository::new(db.clone());
        let notification_service = NotificationService::new(notification_repo, user_repo.clone());

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
                nickname: user.nickname.unwrap_or_default(),
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
                is_mutual,
                ..
            } => {
                if *is_mutual {
                    return Ok(());
                }
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::NewFollower,
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
            AppEvent::MutualFollow {
                user_id,
                mutual_user_id,
            } => {
                self.notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::MutualFollow,
                        "互相关注".to_string(),
                        "与你成为好友".to_string(),
                        Some(*mutual_user_id),
                        Some("user".to_string()),
                        Some(mutual_user_id.to_string()),
                        None,
                        None,
                    )
                    .await?;
                self.notification_service
                    .create_notification(
                        *mutual_user_id,
                        NotificationType::MutualFollow,
                        "互相关注".to_string(),
                        "与你成为好友".to_string(),
                        Some(*user_id),
                        Some("user".to_string()),
                        Some(user_id.to_string()),
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
                        NotificationType::MomentLike,
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
                        NotificationType::MomentComment,
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
                        NotificationType::CommentReply,
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
            AppEvent::GroupInviteResponse {
                invite_id,
                group_id,
                inviter_id,
                invitee_id,
                status,
            } => {
                if status.as_str() != "accepted" {
                    return Ok(());
                }
                self.notification_service
                    .create_notification(
                        *inviter_id,
                        NotificationType::GroupInviteResponse,
                        "群组邀请已接受".to_string(),
                        "对方已接受你的群组邀请".to_string(),
                        Some(*invitee_id),
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
                            NotificationType::GroupApplication,
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
            AppEvent::GroupApplicationResponse {
                application_id,
                group_id,
                group_name,
                applicant_id,
                status,
            } => {
                if status.as_str() != "accepted" {
                    return Ok(());
                }
                self.notification_service
                    .create_notification(
                        *applicant_id,
                        NotificationType::GroupApplicationResponse,
                        "入群申请已通过".to_string(),
                        format!("已通过加入群组: {}", group_name),
                        None,
                        Some("group".to_string()),
                        Some(group_id.to_string()),
                        Some(application_id.to_string()),
                        None,
                    )
                    .await?;
            }
            AppEvent::GroupAnnouncementCreated {
                announcement_id,
                group_id,
                title,
                content,
                created_by_user_id,
                ..
            } => {
                let member_ids = self
                    .group_member_repo
                    .list_member_user_ids(*group_id)
                    .await?;
                for member_id in member_ids {
                    if member_id == *created_by_user_id {
                        continue;
                    }
                    self.notification_service
                        .create_notification(
                            member_id,
                            NotificationType::GroupAnnouncement,
                            title.clone(),
                            content.clone(),
                            Some(*created_by_user_id),
                            Some("group".to_string()),
                            Some(group_id.to_string()),
                            Some(announcement_id.to_string()),
                            None,
                        )
                        .await?;
                }
            }
            AppEvent::SystemAnnouncement {
                announcement_id,
                title,
                content,
                action_url,
                ..
            } => {
                let mut last_user_id = 0;
                loop {
                    let user_ids = self
                        .user_repo
                        .list_user_ids_after(last_user_id, 1000)
                        .await?;
                    if user_ids.is_empty() {
                        break;
                    }
                    for uid in &user_ids {
                        self.notification_service
                            .create_notification(
                                *uid,
                                NotificationType::SystemAnnouncement,
                                title.clone(),
                                content.clone(),
                                None,
                                action_url.as_ref().map(|_| "web".to_string()),
                                action_url.clone(),
                                Some(announcement_id.clone()),
                                None,
                            )
                            .await?;
                    }
                    if let Some(last) = user_ids.last() {
                        last_user_id = *last;
                    } else {
                        break;
                    }
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
                        NotificationType::SystemAnnouncement,
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
                        NotificationType::FollowRequest,
                        "新的关注请求".to_string(),
                        "请求关注你".to_string(),
                        Some(*requester_id),
                        None,
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
                if !accepted {
                    return Ok(());
                }
                // 创建关注请求处理通知
                let title = "关注请求已通过";
                let content = "已同意你的关注请求";
                self.notification_service
                    .create_notification(
                        *requester_id,
                        NotificationType::FollowRequestHandled,
                        title.to_string(),
                        content.to_string(),
                        Some(*target_id),
                        None,
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
            AppEvent::UserProfileUpdated { user_id, .. } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::UserProfileUpdated {
                    user: user,
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                // 推送给订阅了该用户状态的用户
                let subscribers = self
                    .ws_subscription
                    .get_user_status_subscribers(*user_id)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::UserLocationUpdated { .. } => {
                // spec 未定义 UserLocationUpdate WebSocket 事件，不推送
            }

            // ========== 关注关系事件 ==========
            AppEvent::NewFollower {
                user_id,
                follower_id,
                is_mutual,
            } => {
                // 通知推送 = 持久化偏好
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

                let cfg_for_user = self.subscription_service.get_config(*user_id).await?;
                if cfg_for_user.mutual_follow_enabled {
                    let message_for_user = WsMessage::MutualFollow {
                        user: mutual_user_info.clone(),
                        mutual_at: now.clone(),
                    };
                    let _ = self
                        .ws_manager
                        .send_to_user(*user_id, &message_for_user)
                        .await;
                }

                let cfg_for_mutual = self
                    .subscription_service
                    .get_config(*mutual_user_id)
                    .await?;
                if cfg_for_mutual.mutual_follow_enabled {
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
            AppEvent::FollowRequestReceived { .. } => {
                // spec 未定义 FollowRequestReceived WebSocket 事件，不推送
                // 通知已在 persist_notification 中处理
            }
            AppEvent::FollowRequestHandled { .. } => {
                // spec 未定义 FollowRequestHandled WebSocket 事件，不推送
                // 通知已在 persist_notification 中处理
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
                    group_id: group_id.to_string(),
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
                removed_by,
                member_count,
                reason,
            } => {
                let user = self.get_user_info(*user_id).await;
                let group_id_str = group_id.to_string();

                if *reason == crate::types::enums::MemberLeaveReason::Quit {
                    // 成员主动离开
                    let message = WsMessage::GroupMemberLeft {
                        group_id: group_id_str.clone(),
                        user,
                        left_at: chrono::Utc::now().to_rfc3339(),
                        member_count: *member_count as i32,
                    };
                    let subscribers = self
                        .ws_subscription
                        .get_group_chat_subscribers(&group_id_str)
                        .await;
                    self.ws_manager.send_to_users(&subscribers, &message).await;
                } else {
                    // 被踢出或封禁
                    let reason_str = match reason {
                        crate::types::enums::MemberLeaveReason::Kick => Some("kicked".to_string()),
                        crate::types::enums::MemberLeaveReason::Dismiss => None,
                        _ => Some("banned".to_string()),
                    };
                    let message = WsMessage::GroupMemberRemoved {
                        group_id: group_id_str.clone(),
                        user,
                        removed_by: removed_by.to_string(),
                        reason: reason_str,
                        removed_at: chrono::Utc::now().to_rfc3339(),
                        member_count: *member_count as i32,
                    };
                    let subscribers = self
                        .ws_subscription
                        .get_group_chat_subscribers(&group_id_str)
                        .await;
                    self.ws_manager.send_to_users(&subscribers, &message).await;
                    // 也推送给被移除的用户自身
                    let _ = self.ws_manager.send_to_user(*user_id, &message).await;
                }
            }
            AppEvent::GroupProfileUpdated {
                group_id,
                updated_by,
                name,
                description,
                avatar_url,
                tags,
                join_method,
                is_searchable,
                max_members,
                has_password,
            } => {
                let mut fields = serde_json::Map::new();
                if let Some(v) = name {
                    fields.insert("name".into(), serde_json::Value::String(v.clone()));
                }
                if let Some(v) = description {
                    fields.insert("description".into(), serde_json::Value::String(v.clone()));
                }
                if let Some(v) = avatar_url {
                    fields.insert("avatar_url".into(), serde_json::Value::String(v.clone()));
                }
                if let Some(v) = tags {
                    fields.insert("tags".into(), serde_json::json!(v));
                }
                if let Some(v) = join_method {
                    fields.insert("join_method".into(), serde_json::Value::String(v.clone()));
                }
                if let Some(v) = is_searchable {
                    fields.insert("is_searchable".into(), serde_json::Value::Bool(*v));
                }
                if let Some(v) = max_members {
                    fields.insert("max_members".into(), serde_json::json!(v));
                }
                if let Some(v) = has_password {
                    fields.insert("has_password".into(), serde_json::Value::Bool(*v));
                }

                let message = WsMessage::GroupUpdated {
                    group_id: group_id.to_string(),
                    updated_fields: serde_json::Value::Object(fields),
                    updated_by: updated_by.to_string(),
                    updated_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
            }
            AppEvent::GroupMemberRoleChanged { .. } => {}
            AppEvent::GroupMemberMuted {
                group_id,
                user_id,
                muted_by,
                muted_until,
                duration_minutes,
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::GroupMemberMuted {
                    group_id: group_id.to_string(),
                    user,
                    muted_by: muted_by.to_string(),
                    muted_until: muted_until.clone(),
                    duration_minutes: *duration_minutes,
                    muted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
                // 也推送给被禁言用户自身
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::GroupMemberUnmuted {
                group_id,
                user_id,
                unmuted_by,
            } => {
                let user = self.get_user_info(*user_id).await;
                let message = WsMessage::GroupMemberUnmuted {
                    group_id: group_id.to_string(),
                    user,
                    unmuted_by: unmuted_by.to_string(),
                    unmuted_at: chrono::Utc::now().to_rfc3339(),
                };
                let group_id_str = group_id.to_string();
                let subscribers = self
                    .ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                self.ws_manager.send_to_users(&subscribers, &message).await;
                // 也推送给被解除禁言用户自身
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::GroupDismissed {
                group_id,
                dismissed_by,
            } => {
                // 判断是群主解散还是管理员解散
                let reason = Some(format!("dismissed_by_{}", dismissed_by));
                let message = WsMessage::GroupDismissed {
                    group_id: group_id.to_string(),
                    dismissed_at: chrono::Utc::now().to_rfc3339(),
                    reason,
                };
                // 推送给所有群订阅者
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
                let cfg = self.subscription_service.get_config(*to_user_id).await?;
                if !cfg.group_invite_enabled {
                    return Ok(());
                }

                let inviter = self.get_user_info(*from_user_id).await;
                let ws_message = WsMessage::GroupInviteReceived {
                    invite_id: invite_id.to_string(),
                    group_id: group_id.to_string(),
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
                if status.as_str() != "accepted" {
                    return Ok(());
                }
                // 邀请响应只通知邀请人（通知类事件，按订阅与偏好）
                let cfg = self.subscription_service.get_config(*inviter_id).await?;
                if !cfg.group_invite_response_enabled {
                    return Ok(());
                }

                let user = self.get_user_info(*invitee_id).await;
                let message = WsMessage::GroupInviteResponse {
                    invite_id: invite_id.to_string(),
                    group_id: group_id.to_string(),
                    user,
                    status: "accepted".to_string(),
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
                    group_id: group_id.to_string(),
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
                if status.as_str() != "accepted" {
                    return Ok(());
                }
                let message = WsMessage::GroupApplicationResponse {
                    application_id: application_id.to_string(),
                    group_id: group_id.to_string(),
                    group_name: group_name.clone(),
                    status: "accepted".to_string(),
                    responded_at: chrono::Utc::now().to_rfc3339(),
                };
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
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::GroupMessageNew {
                    message_id: message_id.to_string(),
                    group_id: group_chat_id.to_string(),
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    media: media.clone(),
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
                    group_id: group_chat_id.to_string(),
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
                    group_id: group_chat_id.to_string(),
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
            AppEvent::GroupMessageDeleted { .. } => {
                // spec 未定义 GroupMessageDeleted WebSocket 事件，不推送
            }
            AppEvent::GroupMessageRead { .. } => {
                // spec 未定义 GroupMessageRead WebSocket 事件，不推送
            }
            AppEvent::GroupMessageReaction { .. } => {
                // spec 未定义 GroupMessageReaction WebSocket 事件，不推送
            }
            AppEvent::GroupUserTyping {
                group_chat_id,
                user_id,
                is_typing,
            } => {
                let message = WsMessage::GroupUserTyping {
                    group_id: group_chat_id.to_string(),
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
            } => {
                let sender = self.get_user_info(*sender_id).await;
                let message = WsMessage::PrivateMessageNew {
                    message_id: message_id.to_string(),
                    target_user_id: *target_user_id,
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    media: media.clone(),
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
            AppEvent::PrivateMessageDeleted { .. } => {
                // spec 未定义 PrivateMessageDeleted WebSocket 事件，不推送
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
            AppEvent::PrivateMessageReaction { .. } => {
                // spec 未定义 PrivateMessageReaction WebSocket 事件，不推送
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
            // 注：SelfJoinedGroup 由 GroupMemberJoined 统一处理；SelfLeftGroup 不做推送
            AppEvent::SelfJoinedGroup { .. } => {}
            AppEvent::SelfLeftGroup { .. } => {}
            AppEvent::PrivateChatDeleted { .. } => {}
            AppEvent::PrivateChatSettingsUpdated { .. } => {}
            AppEvent::NotificationReadStateUpdated { .. } => {}

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
                moment_id,
                user_id,
                liker_id,
                comment_preview,
            } => {
                // 不推送给自己
                if user_id == liker_id {
                    return Ok(());
                }

                let liker = self.get_user_info(*liker_id).await;
                let message = WsMessage::CommentLiked {
                    comment_id: comment_id.to_string(),
                    moment_id: moment_id.to_string(),
                    moment_summary: None, // 简化：不重复查瞬间摘要
                    comment_preview: comment_preview.clone(),
                    user: liker,
                    liked_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
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
            AppEvent::SystemNotification { .. } => {}
            AppEvent::SystemAnnouncement {
                announcement_id,
                title,
                content,
                severity,
                action_url,
                created_at,
            } => {
                let message = WsMessage::SystemAnnouncement {
                    announcement_id: announcement_id.clone(),
                    title: title.clone(),
                    content: content.clone(),
                    severity: *severity,
                    action_url: action_url.clone(),
                    created_at: created_at.clone(),
                };
                // 系统公告推送给所有在线用户（按通知偏好过滤）
                let online_users = self.ws_manager.get_online_user_ids().await;
                for uid in online_users {
                    let cfg = self.subscription_service.get_config(uid).await?;
                    if cfg.system_announcement_enabled {
                        let _ = self.ws_manager.send_to_user(uid, &message).await;
                    }
                }
            }

            AppEvent::AccountBanned {
                user_id,
                reason,
                duration_days,
            } => {
                let message = WsMessage::AccountBanned {
                    reason: reason.clone(),
                    duration_days: *duration_days,
                    banned_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
                // TODO: 推送后断开该用户的 WebSocket 连接
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
                    group_id: group_id.to_string(),
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
            AppEvent::GroupAnnouncementUpdated { .. } => {}
            AppEvent::GroupAnnouncementDeleted { .. } => {}

            // ========== @ 提及事件 ==========
            AppEvent::MomentMentioned {
                moment_id,
                mentioned_user_id,
                mentioning_user_id,
                moment_text_preview,
                moment_cover_url,
            } => {
                let cfg = self
                    .subscription_service
                    .get_config(*mentioned_user_id)
                    .await?;
                if !cfg.message_mention_enabled {
                    return Ok(());
                }
                let user = self.get_user_info(*mentioning_user_id).await;
                let message = WsMessage::MessageMentioned {
                    target_type: "moment".to_string(),
                    moment_id: moment_id.to_string(),
                    comment_id: None,
                    moment_summary: Some(crate::websocket::types::MomentSummary {
                        moment_id: moment_id.to_string(),
                        cover_url: moment_cover_url.clone(),
                        text_preview: moment_text_preview.clone(),
                    }),
                    comment_preview: None,
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
                let cfg = self
                    .subscription_service
                    .get_config(*mentioned_user_id)
                    .await?;
                if !cfg.message_mention_enabled {
                    return Ok(());
                }
                let user = self.get_user_info(*mentioning_user_id).await;
                let message = WsMessage::MessageMentioned {
                    target_type: "comment".to_string(),
                    moment_id: moment_id.to_string(),
                    comment_id: Some(comment_id.to_string()),
                    moment_summary: None,
                    comment_preview: Some(content_preview.clone()),
                    user,
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
