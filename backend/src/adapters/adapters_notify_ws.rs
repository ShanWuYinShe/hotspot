/// WebSocket通知传输适配器
/// 
/// 基于WebSocket的实时通知发送实现

use async_trait::async_trait;
use std::collections::HashMap;
use std::sync::Arc;
use crate::domain::DomainResult;
use crate::ports::{
    NotificationTransportPort, NotificationPolicyPort, 
    Notification, NotificationPolicy, NotificationContext, NotificationType,
    TransportStats, PolicyStats
};
use crate::ports::repository_ports::{FriendRepositoryPort, UserRepositoryPort, GroupRepositoryPort};

/// WebSocket通知传输实现
#[derive(Clone)]
pub struct WebSocketNotificationTransport {
    ws_manager: Option<Arc<crate::ws::manager::WebSocketManager>>,
}

impl WebSocketNotificationTransport {
    pub fn new() -> Self {
        Self {
            ws_manager: None,
        }
    }
    
    pub fn with_ws_manager(ws_manager: Arc<crate::ws::manager::WebSocketManager>) -> Self {
        Self {
            ws_manager: Some(ws_manager),
        }
    }
}

impl Default for WebSocketNotificationTransport {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl NotificationTransportPort for WebSocketNotificationTransport {
    async fn send_notification(&self, notification: Notification) -> DomainResult<()> {
        if let Some(ws_manager) = &self.ws_manager {
            // 构造WebSocket消息
            let notification_data = crate::ws::message_types::SystemNotificationData {
                notification_id: uuid::Uuid::new_v4(),
                notification_type: match notification.notification_type {
                    NotificationType::System => crate::ws::message_types::SystemNotificationType::System,
                    NotificationType::Message => crate::ws::message_types::SystemNotificationType::NewMessage,
                    NotificationType::GroupInvite => crate::ws::message_types::SystemNotificationType::GroupInvitation,
                    NotificationType::GroupMessage => crate::ws::message_types::SystemNotificationType::GroupDigest,
                    NotificationType::FriendRequest => crate::ws::message_types::SystemNotificationType::FriendRequest,
                    NotificationType::MomentLike => crate::ws::message_types::SystemNotificationType::NewMoment,
                    NotificationType::MomentComment => crate::ws::message_types::SystemNotificationType::NewMoment,
                    NotificationType::LocationUpdate => crate::ws::message_types::SystemNotificationType::System,
                },
                title: notification.title,
                content: notification.content,
                data: Some(serde_json::Value::Object(notification.data.into_iter().collect())),
                created_at: chrono::Utc::now(),
            };
            
            let title_for_log = notification_data.title.clone();
            let ws_message = crate::ws::message_types::ServerMessage::SystemNotification(notification_data);
            
            // 通过WebSocket消息路由器发送
            if let Err(e) = ws_manager.message_router().send_to_user(notification.user_id, ws_message).await {
                log::error!("WebSocket通知发送失败: 用户{} - {:?}", notification.user_id, e);
                return Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("WebSocket发送失败: {}", e) 
                });
            }
            
            log::debug!(
                "WebSocket通知已发送: 用户{} - {}",
                notification.user_id,
                title_for_log
            );
        } else {
            // 如果WebSocket管理器不可用，记录日志但不报错
            log::warn!(
                "WebSocket管理器不可用，跳过通知: 用户{} - {}",
                notification.user_id,
                notification.title
            );
        }
        Ok(())
    }

    async fn send_notifications(&self, notifications: Vec<Notification>) -> DomainResult<()> {
        for notification in notifications {
            self.send_notification(notification).await?;
        }
        Ok(())
    }

    async fn broadcast_notification(&self, notification: Notification) -> DomainResult<()> {
        // 谨慎使用广播，建议通过策略来获取目标用户
        log::warn!("广播通知: {} - {}", notification.title, notification.content);
        Ok(())
    }

    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        Ok(TransportStats {
            sent_notifications: 0,
            failed_notifications: 0,
            avg_send_time_ms: 0,
        })
    }
}

/// 基本通知策略实现
#[derive(Clone)]
pub struct BasicNotificationPolicy<U, G, F> 
where
    U: UserRepositoryPort,
    G: GroupRepositoryPort,
    F: FriendRepositoryPort,
{
    user_repository: Arc<U>,
    group_repository: Arc<G>,
    friend_repository: Arc<F>,
}

impl<U, G, F> BasicNotificationPolicy<U, G, F>
where
    U: UserRepositoryPort,
    G: GroupRepositoryPort,
    F: FriendRepositoryPort,
{
    pub fn new(user_repository: Arc<U>, group_repository: Arc<G>, friend_repository: Arc<F>) -> Self {
        Self {
            user_repository,
            group_repository,
            friend_repository,
        }
    }
}

#[async_trait]
impl<U, G, F> NotificationPolicyPort for BasicNotificationPolicy<U, G, F>
where
    U: UserRepositoryPort,
    G: GroupRepositoryPort,
    F: FriendRepositoryPort,
{
    async fn get_target_users(
        &self,
        policy: NotificationPolicy,
        _context: NotificationContext,
    ) -> DomainResult<Vec<i64>> {
        match policy {
            NotificationPolicy::Specific { user_ids } => Ok(user_ids),
            NotificationPolicy::Friends => {
                // 实现好友关系查询
                log::debug!("通知策略: 查询好友关系");
                let user_id = _context.trigger_user_id;
                match self.friend_repository.get_friends(user_id, 0, 1000).await {
                    Ok(friends) => {
                        log::debug!("用户{}找到{}个好友", user_id, friends.len());
                        Ok(friends)
                    }
                    Err(e) => {
                        log::error!("查询用户{}好友失败: {:?}", user_id, e);
                        Ok(vec![])
                    }
                }
            }
            NotificationPolicy::GroupMembers { group_id } => {
                // 查询群组成员
                log::debug!("通知策略: 查询群组{}的成员", group_id);
                match self.group_repository.get_group_members(group_id, 0, 1000).await {
                    Ok((members, _has_more, _next_cursor)) => {
                        let user_ids: Vec<i64> = members.into_iter().map(|member| member.user_id).collect();
                        log::debug!("群组{}找到{}个成员", group_id, user_ids.len());
                        Ok(user_ids)
                    }
                    Err(e) => {
                        log::error!("查询群组{}成员失败: {:?}", group_id, e);
                        Ok(vec![])
                    }
                }
            }
            NotificationPolicy::Nearby { latitude, longitude, radius_km } => {
                // 查询附近用户
                log::debug!(
                    "通知策略: 查询附近用户 ({}, {}, {}km)",
                    latitude, longitude, radius_km
                );
                match self.user_repository.find_nearby_users(latitude, longitude, radius_km, 0, 100).await {
                    Ok(users) => {
                        let user_ids: Vec<i64> = users.into_iter().map(|user| user.id).collect();
                        log::debug!("附近找到{}个用户", user_ids.len());
                        Ok(user_ids)
                    }
                    Err(e) => {
                        log::error!("查询附近用户失败: {:?}", e);
                        Ok(vec![])
                    }
                }
            }
            NotificationPolicy::Combined { policies } => {
                let mut all_users = Vec::new();
                for sub_policy in policies {
                    let mut users = self.get_target_users(sub_policy, _context.clone()).await?;
                    all_users.append(&mut users);
                }
                // 去重
                all_users.sort();
                all_users.dedup();
                Ok(all_users)
            }
        }
    }

    async fn should_notify_user(
        &self,
        _user_id: i64,
        _notification_type: NotificationType,
        _context: &NotificationContext,
    ) -> DomainResult<bool> {
        // 基本实现：总是通知
        Ok(true)
    }

    async fn get_policy_stats(&self) -> DomainResult<PolicyStats> {
        Ok(PolicyStats {
            policy_usage: HashMap::new(),
            avg_target_users: 0.0,
            avg_evaluation_time_ms: 0,
        })
    }
}

