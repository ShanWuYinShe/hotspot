/// 复合通知传输适配器
/// 
/// 支持多种通知渠道的复合传输器，可以同时通过WebSocket和邮件发送通知

use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::domain::DomainResult;
use crate::ports::{
    NotificationTransportPort, Notification, NotificationChannel,
    TransportStats,
};

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
        
        for channel in &notification.channels {
            match channel {
                NotificationChannel::WebSocket => {
                    #[cfg(feature = "ws")]
                    if let Some(ws_transport) = &self.websocket_transport {
                        let mut ws_notification = notification.clone();
                        ws_notification.channels = vec![NotificationChannel::WebSocket];
                        
                        match ws_transport.send_notification(ws_notification).await {
                            Ok(_) => {
                                sent_count += 1;
                                debug!("WebSocket通知发送成功: 用户{}", notification.user_id);
                            },
                            Err(e) => {
                                error!("WebSocket通知发送失败: 用户{} - {:?}", notification.user_id, e);
                                errors.push(e);
                            }
                        }
                    } else {
                        warn!("WebSocket传输器不可用，跳过WebSocket通知");
                    }
                    
                    #[cfg(not(feature = "ws"))]
                    warn!("WebSocket功能未启用，跳过WebSocket通知");
                },
                NotificationChannel::Email => {
                    #[cfg(feature = "email_channel")]
                    if let Some(email_transport) = &self.email_transport {
                        let mut email_notification = notification.clone();
                        email_notification.channels = vec![NotificationChannel::Email];
                        
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
                NotificationChannel::Sms => {
                    // TODO: 实现短信通知
                    warn!("短信通知功能尚未实现，跳过短信通知");
                },
                NotificationChannel::Push => {
                    // TODO: 实现推送通知
                    warn!("推送通知功能尚未实现，跳过推送通知");
                },
            }
        }
        
        if sent_count > 0 {
            info!(
                "复合通知发送完成: 用户{} - 成功{}/总共{}个渠道",
                notification.user_id,
                sent_count,
                notification.channels.len()
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
impl NotificationTransportPort for CompositeNotificationTransport {
    async fn send_notification(&self, notification: Notification) -> DomainResult<()> {
        if notification.channels.is_empty() {
            warn!("通知没有指定任何渠道，使用默认WebSocket渠道: {}", notification.title);
            let mut default_notification = notification;
            default_notification.channels = vec![NotificationChannel::WebSocket];
            return self.dispatch_by_channel(default_notification).await;
        }
        
        self.dispatch_by_channel(notification).await
    }
    
    async fn send_notifications(&self, notifications: Vec<Notification>) -> DomainResult<()> {
        let mut errors = Vec::new();
        let mut success_count = 0;
        
        for notification in notifications {
            match self.send_notification(notification).await {
                Ok(_) => success_count += 1,
                Err(e) => errors.push(e),
            }
        }
        
        info!(
            "批量通知发送完成: 成功{}/总共{}个通知",
            success_count,
            success_count + errors.len()
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
    
    async fn broadcast_notification(&self, notification: Notification) -> DomainResult<()> {
        // 广播通知只通过WebSocket发送，不发送邮件
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            let mut ws_notification = notification;
            ws_notification.channels = vec![NotificationChannel::WebSocket];
            return ws_transport.broadcast_notification(ws_notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法广播通知: {}", notification.title);
        Ok(())
    }
    
    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        let mut total_sent = 0;
        let mut total_failed = 0;
        let mut total_time = 0;
        let mut transport_count = 0;
        
        #[cfg(feature = "ws")]
        if let Some(ws_transport) = &self.websocket_transport {
            if let Ok(stats) = ws_transport.get_transport_stats().await {
                total_sent += stats.sent_notifications;
                total_failed += stats.failed_notifications;
                total_time += stats.avg_send_time_ms;
                transport_count += 1;
            }
        }
        
        #[cfg(feature = "email_channel")]
        if let Some(email_transport) = &self.email_transport {
            if let Ok(stats) = email_transport.get_transport_stats().await {
                total_sent += stats.sent_notifications;
                total_failed += stats.failed_notifications;
                total_time += stats.avg_send_time_ms;
                transport_count += 1;
            }
        }
        
        let avg_time = if transport_count > 0 {
            total_time / transport_count
        } else {
            0
        };
        
        Ok(TransportStats {
            sent_notifications: total_sent,
            failed_notifications: total_failed,
            avg_send_time_ms: avg_time,
        })
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
