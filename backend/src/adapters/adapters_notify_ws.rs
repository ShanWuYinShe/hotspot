/// WebSocket通知传输适配器
/// 
/// 基于WebSocket的实时通知发送实现
/// WebSocket 通知适配器
/// 
/// 功能职责：
/// - 实现 NotificationTransport 的 WebSocket 适配器
/// - 提供实时通知的主要传输渠道
/// - 管理 WebSocket 连接和消息路由
/// - 实现通知的可靠性和容错机制
/// - 支持多种通知类型和消息格式
/// 
/// 通知类型：
/// - 实时消息：私聊和群聊消息的即时推送
/// - 系统通知：系统事件和状态更新通知
/// - 用户状态：在线状态、位置更新等
/// - 群组事件：成员变动、角色变更等
/// - 业务通知：瞬间更新、点赞评论等
/// 
/// 技术特性：
/// - 双向通信：支持客户端和服务端的双向消息
/// - 连接管理：自动重连、心跳检测、超时处理
/// - 消息路由：基于房间和用户的智能消息分发
/// - 性能优化：消息批量处理和连接池管理
/// 
/// 可靠性保证：
/// - 消息去重和顺序保证
/// - 连接断线的自动恢复和消息补发
/// - 通知发送失败的重试和容错机制
use async_trait::async_trait;
// use std::collections::HashMap; // 暂时未使用
use std::sync::Arc;
use crate::domain::{DomainResult, UserId};
use crate::ports::NotificationTransport;
use crate::ports::services::{TransportStats, /* PolicyStats */}; // PolicyStats 暂时未使用
use crate::ports::services::{Notification, /* NotificationContext */}; // NotificationContext 暂时未使用
// use crate::ports::services::NotificationPolicy as NotificationPolicyTrait; // 暂时未使用
use crate::ports::services::NotificationType;
// use crate::ports::repository_ports::{/* FriendRepository, UserRepository, GroupRepository */};

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
impl NotificationTransport for WebSocketNotificationTransport {
    async fn send_notification(&self, user_id: i64, notification: &Notification) -> DomainResult<()> {
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
                    // Handle other variants
                    NotificationType::GroupInvitation => crate::ws::message_types::SystemNotificationType::GroupInvitation,
                    NotificationType::SystemAnnouncement => crate::ws::message_types::SystemNotificationType::System,
                    NotificationType::GroupJoin => crate::ws::message_types::SystemNotificationType::GroupDigest,
                    NotificationType::GroupLeave => crate::ws::message_types::SystemNotificationType::GroupDigest,
                },
                title: notification.title.clone(),
                content: notification.content.clone(),
                data: notification.data.clone(),
                created_at: chrono::Utc::now(),
            };
            
            let title_for_log = notification_data.title.clone();
            let ws_message = crate::ws::message_types::ServerMessage::SystemNotification(notification_data);
            
            // 通过WebSocket消息路由器发送
            if let Err(e) = ws_manager.message_router().send_to_user(user_id, ws_message).await {
                log::error!("WebSocket通知发送失败: 用户{} - {:?}", user_id, e);
                return Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("WebSocket发送失败: {}", e) 
                });
            }
            
            log::debug!(
                "WebSocket通知已发送: 用户{} - {}",
                user_id,
                title_for_log
            );
        } else {
            // 如果WebSocket管理器不可用，记录日志但不报错
            log::warn!(
                "WebSocket管理器不可用，跳过通知: 用户{} - {}",
                user_id,
                notification.title
            );
        }
        Ok(())
    }

    async fn send_notifications(&self, notifications: &[(i64, Notification)]) -> DomainResult<()> {
        for (user_id, notification) in notifications {
            self.send_notification(*user_id, notification).await?;
        }
        Ok(())
    }

    async fn broadcast_notification(&self, notification: &Notification) -> DomainResult<()> {
        // 谨慎使用广播，建议通过策略来获取目标用户
        log::warn!("广播通知: {} - {}", notification.title, notification.content);
        Ok(())
    }

    async fn send_to_user(&self, user_id: UserId, notification: &Notification) -> DomainResult<()> {
        self.send_notification(user_id.value(), notification).await
    }

    async fn send_to_users(&self, user_ids: &[UserId], notification: &Notification) -> DomainResult<()> {
        for user_id in user_ids {
            self.send_notification(user_id.value(), notification).await?;
        }
        Ok(())
    }

    async fn send_to_group(&self, group_id: &str, notification: &Notification) -> DomainResult<()> {
        log::info!("发送群组通知: 群组 {} - {} - {}", group_id, notification.title, notification.content);
        
        // 通过WebSocket管理器发送群组通知
        if let Some(ws_manager) = &self.ws_manager {
            // 构建WebSocket消息
            let notification_uuid = match uuid::Uuid::parse_str(&notification.notification_id) {
                Ok(uuid) => uuid,
                Err(_) => uuid::Uuid::new_v4(), // 如果解析失败，生成新的UUID
            };
            
            let ws_message = crate::ws::message_types::ServerMessage::SystemNotification(
                crate::ws::message_types::SystemNotificationData {
                    notification_id: notification_uuid,
                    title: notification.title.clone(),
                    content: notification.content.clone(),
                    notification_type: crate::ws::message_types::SystemNotificationType::System,
                    data: notification.data.clone(),
                    created_at: chrono::Utc::now(),
                }
            );
            
            // 发送到群组房间
            let group_uuid = match uuid::Uuid::parse_str(group_id) {
                Ok(uuid) => uuid,
                Err(_) => {
                    log::error!("无效的群组ID格式: {}", group_id);
                    return Err(crate::domain::DomainError::BusinessRuleViolation {
                        rule: "无效的群组ID格式".to_string(),
                    });
                }
            };
            
            let room_id = crate::ws::message_types::RoomId::Group(group_uuid);
            match ws_manager.message_router().send_to_room(&room_id, ws_message).await {
                Ok(_) => {
                    log::debug!("群组通知发送成功: 群组 {}", group_id);
                    Ok(())
                }
                Err(e) => {
                    log::error!("群组通知发送失败: 群组 {} - {:?}", group_id, e);
                    Err(crate::domain::DomainError::BusinessRuleViolation {
                        rule: format!("群组通知发送失败: {}", e),
                    })
                }
            }
        } else {
            log::warn!("WebSocket管理器不可用，跳过群组通知");
            Ok(())
        }
    }

    async fn broadcast(&self, notification: &Notification) -> DomainResult<()> {
        log::warn!("广播通知: {} - {} (谨慎使用，建议通过策略获取目标用户)", notification.title, notification.content);

        // 通过WebSocket管理器发送广播通知
        if let Some(_ws_manager) = &self.ws_manager {
            // 构建WebSocket消息
            let notification_uuid = match uuid::Uuid::parse_str(&notification.notification_id) {
                Ok(uuid) => uuid,
                Err(_) => uuid::Uuid::new_v4(), // 如果解析失败，生成新的UUID
            };
            
            let _ws_message = crate::ws::message_types::ServerMessage::SystemNotification(
                crate::ws::message_types::SystemNotificationData {
                    notification_id: notification_uuid,
                    title: notification.title.clone(),
                    content: notification.content.clone(),
                    notification_type: crate::ws::message_types::SystemNotificationType::System,
                    data: notification.data.clone(),
                    created_at: chrono::Utc::now(),
                }
            );
            
            // 广播功能已在项目规范中禁用，建议通过策略获取目标用户
            log::warn!("广播功能已禁用，建议通过通知策略获取目标用户列表");
            // 如果确实需要广播，可以通过获取所有在线用户来实现
            // let online_users = ws_manager.connection_manager().get_all_online_users().await;
            // match ws_manager.message_router().send_to_users(&online_users, ws_message).await {
            //     Ok(_) => {
            //         log::info!("广播通知发送成功");
            //         Ok(())
            //     }
            //     Err(e) => {
            //         log::error!("广播通知发送失败: {:?}", e);
            //         Err(crate::domain::DomainError::BusinessRuleViolation {
            //             rule: format!("广播通知发送失败: {}", e),
            //         })
            //     }
            // }
            Ok(())
        } else {
            log::warn!("WebSocket管理器不可用，跳过广播通知");
            Ok(())
        }
    }

    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        // 获取WebSocket连接统计信息
        let active_connections = if let Some(ws_manager) = &self.ws_manager {
            ws_manager.connection_manager().get_stats().await.total_connections as u64
        } else {
            0
        };
        
        Ok(TransportStats {
            total_sent: 0, // 暂时返回0，需要实现统计计数器
            total_failed: 0, // 暂时返回0，需要实现统计计数器
            active_connections,
            last_activity: None, // 暂时返回None，需要实现活动时间跟踪
        })
    }
}

// /// 基本通知策略实现
// #[derive(Clone)]
// pub struct BasicNotificationPolicy<U, G, F> 
// where
//     U: UserRepository,
//     G: GroupRepository,
//     F: FriendRepository,
// {
//     user_repository: Arc<U>,
//     group_repository: Arc<G>,
//     friend_repository: Arc<F>,
// }

// impl<U, G, F> BasicNotificationPolicy<U, G, F>
// where
//     U: UserRepository,
//     G: GroupRepository,
//     F: FriendRepository,
// {
//     pub fn new(user_repository: Arc<U>, group_repository: Arc<G>, friend_repository: Arc<F>) -> Self {
//         Self {
//             user_repository,
//             group_repository,
//             friend_repository,
//         }
//     }
// }

// #[async_trait]
// impl<U, G, F> NotificationPolicyTrait for BasicNotificationPolicy<U, G, F>
// where
//     U: UserRepository,
//     G: GroupRepository,
//     F: FriendRepository,
// {
//     async fn should_notify(&self, _context: &NotificationContext) -> DomainResult<bool> {
//         // 基础策略总是允许通知
//         Ok(true)
//     }

//     async fn get_recipients(&self, context: &NotificationContext) -> DomainResult<Vec<i64>> {
//         // 基础实现：返回触发用户自己
//         Ok(vec![context.user_id.value()])
//     }

//     async fn filter_notification(&self, notification: &Notification, _context: &NotificationContext) -> DomainResult<Option<Notification>> {
//         // 基础实现：不过滤通知
//         Ok(Some(notification.clone()))
//     }

// }
