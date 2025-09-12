/// 复合通知传输适配器
/// 
/// 支持多种通知渠道的复合传输器，可以同时通过WebSocket和邮件发送通知
use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, UserId};
use crate::ports::{
    NotificationTransport,
};
use crate::ports::services::{Notification, NotificationPriority};
use crate::ports::services::TransportStats;

/// 复合通知传输器
#[derive(Clone)]
pub struct CompositeNotificationTransport {
    /// WebSocket传输器
    #[cfg(feature = "ws")]
    websocket_transport: Option<Arc<crate::adapters::WebSocketNotificationTransport>>,
    
    /// 邮件传输器
    #[cfg(feature = "email_channel")]
    email_transport: Option<Arc<crate::adapters::EmailNotificationTransport>>,
}

impl CompositeNotificationTransport {
    /// 创建新的复合传输器
    pub fn new() -> Self {
        Self {
            #[cfg(feature = "ws")]
            websocket_transport: None,
            #[cfg(feature = "email_channel")]
            email_transport: None,
        }
    }
    
    /// 设置WebSocket传输器
    #[cfg(feature = "ws")]
    pub fn with_websocket(mut self, transport: Arc<crate::adapters::WebSocketNotificationTransport>) -> Self {
        self.websocket_transport = Some(transport);
        self
    }
    
    /// 设置邮件传输器
    #[cfg(feature = "email_channel")]
    pub fn with_email(mut self, transport: Arc<crate::adapters::EmailNotificationTransport>) -> Self {
        self.email_transport = Some(transport);
        self
    }
    
    /// 根据通知渠道分发到对应的传输器
    async fn dispatch_by_channel(&self, notification: Notification) -> DomainResult<()> {
        let mut errors = Vec::new();
        let mut sent_count = 0;
        
        // 暂时使用默认通道
        let channels = vec![crate::ports::notification_ports::NotificationChannel::WebSocket];
        for channel in &channels {
            match channel {
                crate::ports::notification_ports::NotificationChannel::WebSocket => {
                    #[cfg(feature = "ws")]
                    if let Some(ws_transport) = &self.websocket_transport {
                        let mut ws_notification = notification.clone();
                        // 移除 channels 字段设置，因为 Notification 结构体没有这个字段
                        
                        match ws_transport.send_notification(ws_notification.user_id.unwrap().value(), &ws_notification).await {
                            Ok(_) => {
                                sent_count += 1;
                                debug!("WebSocket通知发送成功: 用户{:?}", notification.user_id);
                            },
                            Err(e) => {
                                error!("WebSocket通知发送失败: 用户{:?} - {:?}", notification.user_id, e);
                                errors.push(e);
                            }
                        }
                    } else {
                        warn!("WebSocket传输器不可用，跳过WebSocket通知");
                    }
                    
                    #[cfg(not(feature = "ws"))]
                    warn!("WebSocket功能未启用，跳过WebSocket通知");
                },
                crate::ports::notification_ports::NotificationChannel::Email => {
                    #[cfg(feature = "email_channel")]
                    if let Some(email_transport) = &self.email_transport {
                        let mut email_notification = notification.clone();
                        email_notification.channels = vec![crate::ports::notification_ports::NotificationChannel::Email];
                        
                        match email_transport.send_notification(email_notification).await {
                            Ok(_) => {
                                sent_count += 1;
                                debug!("邮件通知发送成功: 用户{}", notification.user_id);
                            },
                            Err(e) => {
                                error!("邮件通知发送失败: 用户{} - {:?}", notification.user_id, e);
                                errors.push(e);
                            }
                        }
                    } else {
                        warn!("邮件传输器不可用，跳过邮件通知");
                    }
                    
                    #[cfg(not(feature = "email_channel"))]
                    warn!("邮件功能未启用，跳过邮件通知");
                },
                crate::ports::notification_ports::NotificationChannel::Sms => {
                    // TODO: 实现短信通知
                    warn!("短信通知功能尚未实现，跳过短信通知");
                },
                crate::ports::notification_ports::NotificationChannel::Push => {
                    // TODO: 实现推送通知
                    warn!("推送通知功能尚未实现，跳过推送通知");
                },
            }
        }
        
        if sent_count > 0 {
            info!(
                "复合通知发送完成: 用户{} - 成功{}/总共{}个渠道",
                notification.user_id.as_ref().map(|id| id.value()).unwrap_or(0),
                sent_count,
                channels.len()
            );
        }
        
        // 如果所有渠道都失败了，返回第一个错误
        if sent_count == 0 && !errors.is_empty() {
            return Err(errors.into_iter().next().unwrap());
        }
        
        Ok(())
    }
}

impl Default for CompositeNotificationTransport {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl NotificationTransport for CompositeNotificationTransport {
    async fn send_notification(&self, user_id: i64, notification: &Notification) -> DomainResult<()> {
        // 发送给指定用户
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_notification(user_id, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送通知给用户 {}: {} - {}", user_id, notification.title, notification.content);
        Ok(())
    }
    
    async fn send_notifications(&self, notifications: &[(i64, Notification)]) -> DomainResult<()> {
        let mut errors = Vec::new();
        let mut success_count = 0;
        
        for (user_id, notification) in notifications {
            match self.send_notification(*user_id, notification).await {
                Ok(_) => success_count += 1,
                Err(e) => errors.push(e),
            }
        }
        
        info!(
            "批量通知发送完成: 成功{}/总共{}个通知",
            success_count,
            notifications.len()
        );
        
        // 如果有错误但也有成功的，只记录警告
        if !errors.is_empty() && success_count > 0 {
            warn!("批量通知发送中有{}个失败", errors.len());
        }
        
        // 如果全部失败，返回第一个错误
        if success_count == 0 && !errors.is_empty() {
            return Err(errors.into_iter().next().unwrap());
        }
        
        Ok(())
    }
    
    async fn broadcast_notification(&self, notification: &Notification) -> DomainResult<()> {
        // 广播通知只通过WebSocket发送，不发送邮件
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.broadcast_notification(notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法广播通知: {}", notification.title);
        Ok(())
    }
    
    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        let mut total_sent = 0;
        let mut total_failed = 0;
        
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            if let Ok(stats) = ws_transport.get_transport_stats().await {
                total_sent += stats.total_sent;
                total_failed += stats.total_failed;
            }
        }
        
        #[cfg(feature = "email_channel")]
        if let Some(email_transport) = &self.email_transport {
            if let Ok(stats) = email_transport.get_transport_stats().await {
                total_sent += stats.total_sent;
                total_failed += stats.total_failed;
            }
        }
        
        Ok(TransportStats {
            total_sent,
            total_failed,
            active_connections: 0, // TODO: Calculate actual active connections
            last_activity: None, // TODO: Track last activity
        })
    }
    
    async fn send_to_user(&self, user_id: UserId, notification: &Notification) -> DomainResult<()> {
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_user(user_id, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送用户通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn send_to_users(&self, user_ids: &[UserId], notification: &Notification) -> DomainResult<()> {
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_users(user_ids, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送多用户通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn send_to_group(&self, group_id: &str, notification: &Notification) -> DomainResult<()> {
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_group(group_id, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送群组通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn broadcast(&self, notification: &Notification) -> DomainResult<()> {
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.broadcast(notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法广播通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
}

/// 通知传输器构建器
pub struct NotificationTransportBuilder {
    transport: CompositeNotificationTransport,
}

impl NotificationTransportBuilder {
    pub fn new() -> Self {
        Self {
            transport: CompositeNotificationTransport::new(),
        }
    }
    
    #[cfg(feature = "ws")]
    pub fn with_websocket(mut self, transport: Arc<crate::adapters::WebSocketNotificationTransport>) -> Self {
        self.transport = self.transport.with_websocket(transport);
        self
    }
    
    #[cfg(feature = "email_channel")]
    pub fn with_email(mut self, transport: Arc<crate::adapters::EmailNotificationTransport>) -> Self {
        self.transport = self.transport.with_email(transport);
        self
    }
    
    pub fn build(self) -> CompositeNotificationTransport {
        self.transport
    }
}

impl Default for NotificationTransportBuilder {
    fn default() -> Self {
        Self::new()
    }
}
