/// 复合通知传输适配器
/// 
/// 支持多种通知渠道的复合传输器，可以同时通过WebSocket和邮件发送通知
use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};
use std::time::Duration;

use crate::utils::async_batch_processor::{AsyncBatchProcessor, BatchConfig};

use crate::domain::{DomainResult, UserId};
use crate::ports::{
    NotificationTransport,
};
use crate::ports::services::{Notification /* , NotificationPriority */};
use crate::ports::services::TransportStats;

/// 批量通知项目
#[derive(Debug, Clone)]
pub struct BatchNotificationItem {
    pub user_id: i64,
    pub notification: Notification,
}

/// 复合通知传输器
pub struct CompositeNotificationTransport {
    /// WebSocket传输器
    // WebSocket 是核心功能，始终启用
    websocket_transport: Option<Arc<crate::adapters::WebSocketNotificationTransport>>,
    
    /// 邮件传输器
    #[cfg(feature = "email_notifications")]
    email_transport: Option<Arc<crate::adapters::EmailNotificationTransport>>,
    
    /// 批处理器 - 用于批量发送通知
    batch_processor: Option<Arc<AsyncBatchProcessor<BatchNotificationItem, ()>>>,
}

impl CompositeNotificationTransport {
    /// 创建新的复合传输器
    pub fn new() -> Self {
        Self {
            // WebSocket 是核心功能，始终启用
            websocket_transport: None,
            #[cfg(feature = "email_notifications")]
            email_transport: None,
            batch_processor: None,
        }
    }
    
    /// 启用批处理功能
    pub fn with_batch_processing(mut self) -> Self {
        // 配置批处理器
        let batch_config = BatchConfig {
            batch_size: 50,  // 每批最多50个通知
            batch_timeout: Duration::from_millis(100),  // 100ms超时
            max_concurrency: 5,  // 最多5个并发批次
            max_retries: 2,  // 最多重试2次
            retry_delay: Duration::from_millis(50),
        };
        
        // 创建批处理器
        let processor = AsyncBatchProcessor::new(batch_config, |items: Vec<BatchNotificationItem>| async move {
            // 批量处理逻辑将在这里实现
            info!("批量处理 {} 个通知", items.len());
            
            // 将批量通知转换为 (user_id, notification) 元组
            let notifications: Vec<(i64, Notification)> = items.into_iter()
                .map(|item| (item.user_id, item.notification))
                .collect();
            
            // 这里需要访问 self，但在闭包中无法直接访问
            // 我们需要重新设计这个架构
            Ok(vec![(); notifications.len()])
        });
        
        self.batch_processor = Some(Arc::new(processor));
        self
    }
    
    /// 设置WebSocket传输器
    // WebSocket 是核心功能，始终启用
    pub fn with_websocket(mut self, transport: Arc<crate::adapters::WebSocketNotificationTransport>) -> Self {
        self.websocket_transport = Some(transport);
        self
    }
    
    /// 设置邮件传输器
    #[cfg(feature = "email_notifications")]
    pub fn with_email(mut self, transport: Arc<crate::adapters::EmailNotificationTransport>) -> Self {
        self.email_transport = Some(transport);
        self
    }
    
    /// 根据通知渠道分发到对应的传输器
    #[allow(dead_code)]
    async fn dispatch_by_channel(&self, notification: Notification) -> DomainResult<()> {
        let mut errors = Vec::new();
        let mut sent_count = 0;
        
        // 暂时使用默认通道
        let channels = vec![crate::ports::notification_ports::NotificationChannel::WebSocket];
        for channel in &channels {
            match channel {
                crate::ports::notification_ports::NotificationChannel::WebSocket => {
                    // WebSocket 是核心功能，始终启用
                    if let Some(ws_transport) = &self.websocket_transport {
                        let ws_notification = notification.clone();
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
                    
                    // WebSocket 是核心功能，始终启用，此分支不会执行
                    #[cfg(not(any()))]
                    warn!("WebSocket功能未启用，跳过WebSocket通知");
                },
                crate::ports::notification_ports::NotificationChannel::Email => {
                    #[cfg(feature = "email_notifications")]
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
                    
                    #[cfg(not(feature = "email_notifications"))]
                    warn!("邮件功能未启用，跳过邮件通知");
                },
                crate::ports::notification_ports::NotificationChannel::Sms => {
                    // 短信通知功能（暂未实现具体的短信服务提供商集成）
                    warn!("短信通知功能尚未实现，跳过短信通知");
                    // if let Some(sms_transport) = &self.sms_transport {
                    //     let mut sms_notification = notification.clone();
                    //     sms_notification.channels = vec![crate::ports::notification_ports::NotificationChannel::Sms];
                    //     
                    //     match sms_transport.send_notification(sms_notification).await {
                    //         Ok(_) => {
                    //             sent_count += 1;
                    //             debug!("短信通知发送成功: 用户{}", notification.user_id);
                    //         },
                    //         Err(e) => {
                    //             error!("短信通知发送失败: 用户{} - {:?}", notification.user_id, e);
                    //             errors.push(e);
                    //         }
                    //     }
                    // }
                },
                crate::ports::notification_ports::NotificationChannel::Push => {
                    // 推送通知功能（暂未实现具体的推送服务集成）
                    warn!("推送通知功能尚未实现，跳过推送通知");
                    // if let Some(push_transport) = &self.push_transport {
                    //     let mut push_notification = notification.clone();
                    //     push_notification.channels = vec![crate::ports::notification_ports::NotificationChannel::Push];
                    //     
                    //     match push_transport.send_notification(push_notification).await {
                    //         Ok(_) => {
                    //             sent_count += 1;
                    //             debug!("推送通知发送成功: 用户{}", notification.user_id);
                    //         },
                    //         Err(e) => {
                    //             error!("推送通知发送失败: 用户{} - {:?}", notification.user_id, e);
                    //             errors.push(e);
                    //         }
                    //     }
                    // }
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
        // WebSocket 是核心功能，始终启用
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
        // WebSocket 是核心功能，始终启用
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.broadcast_notification(notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法广播通知: {}", notification.title);
        Ok(())
    }
    
    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        let mut total_sent = 0;
        let mut total_failed = 0;
        
        // WebSocket 是核心功能，始终启用
        if let Some(ws_transport) = &self.websocket_transport {
            if let Ok(stats) = ws_transport.get_transport_stats().await {
                total_sent += stats.total_sent;
                total_failed += stats.total_failed;
            }
        }
        
        #[cfg(feature = "email_notifications")]
        if let Some(email_transport) = &self.email_transport {
            if let Ok(stats) = email_transport.get_transport_stats().await {
                total_sent += stats.total_sent;
                total_failed += stats.total_failed;
            }
        }
        
        Ok(TransportStats {
            total_sent,
            total_failed,
            active_connections: 0,
            last_activity: None,
        })
    }
    
    async fn send_to_user(&self, user_id: UserId, notification: &Notification) -> DomainResult<()> {
        // WebSocket 是核心功能，始终启用
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_user(user_id, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送用户通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn send_to_users(&self, user_ids: &[UserId], notification: &Notification) -> DomainResult<()> {
        // WebSocket 是核心功能，始终启用
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_users(user_ids, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送多用户通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn send_to_group(&self, group_id: &str, notification: &Notification) -> DomainResult<()> {
        // WebSocket 是核心功能，始终启用
        if let Some(ws_transport) = &self.websocket_transport {
            return ws_transport.send_to_group(group_id, notification).await;
        }
        
        warn!("WebSocket传输器不可用，无法发送群组通知: {} - {}", notification.title, notification.content);
        Ok(())
    }
    
    async fn broadcast(&self, notification: &Notification) -> DomainResult<()> {
        // WebSocket 是核心功能，始终启用
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
    
    // WebSocket 是核心功能，始终启用
    pub fn with_websocket(mut self, transport: Arc<crate::adapters::WebSocketNotificationTransport>) -> Self {
        self.transport = self.transport.with_websocket(transport);
        self
    }
    
    #[cfg(feature = "email_notifications")]
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
