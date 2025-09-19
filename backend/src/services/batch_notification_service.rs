/// 批量通知服务
/// 
/// 使用异步批处理器优化通知发送性能
use std::sync::Arc;
use std::time::Duration;
use async_trait::async_trait;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, UserId};
use crate::ports::services::{Notification, TransportStats};
use crate::ports::NotificationTransport;
use crate::utils::async_batch_processor::{AsyncBatchProcessor, BatchConfig};

/// 批量通知项目
#[derive(Debug, Clone)]
pub struct BatchNotificationItem {
    pub user_id: i64,
    pub notification: Notification,
}

/// 批量通知服务
pub struct BatchNotificationService {
    /// 底层通知传输器
    transport: Arc<dyn NotificationTransport>,
    /// 批处理器
    batch_processor: Arc<AsyncBatchProcessor<BatchNotificationItem, ()>>,
}

impl BatchNotificationService {
    /// 创建新的批量通知服务
    pub fn new(transport: Arc<dyn NotificationTransport>) -> Self {
        let batch_config = BatchConfig {
            batch_size: 50,  // 每批最多50个通知
            batch_timeout: Duration::from_millis(100),  // 100ms超时
            max_concurrency: 5,  // 最多5个并发批次
            max_retries: 2,  // 最多重试2次
            retry_delay: Duration::from_millis(50),
        };
        
        // 克隆transport用于闭包
        let transport_clone = Arc::clone(&transport);
        
        // 创建批处理器
        let processor = AsyncBatchProcessor::new(batch_config, move |items: Vec<BatchNotificationItem>| {
            let transport = Arc::clone(&transport_clone);
            async move {
                debug!("开始批量处理 {} 个通知", items.len());
                
                // 将批量通知转换为 (user_id, notification) 元组
                let notifications: Vec<(i64, Notification)> = items.into_iter()
                    .map(|item| (item.user_id, item.notification))
                    .collect();
                
                // 调用底层传输器的批量发送方法
                match transport.send_notifications(&notifications).await {
                    Ok(_) => {
                        info!("批量通知发送成功: {} 个通知", notifications.len());
                        Ok(vec![(); notifications.len()])
                    }
                    Err(e) => {
                        error!("批量通知发送失败: {:?}", e);
                        Err(e)
                    }
                }
            }
        });
        
        Self {
            transport,
            batch_processor: Arc::new(processor),
        }
    }
    
    /// 异步发送单个通知（通过批处理器）
    pub async fn send_notification_batched(&self, user_id: i64, notification: Notification) -> DomainResult<()> {
        let item = BatchNotificationItem {
            user_id,
            notification: notification.clone(),
        };
        
        // 生成唯一ID用于批处理跟踪
        let item_id = format!("notification_{}_{}", user_id, chrono::Utc::now().timestamp_nanos_opt().unwrap_or(0));
        
        // 通过批处理器处理
        match self.batch_processor.process(item_id, item).await {
            Ok(_) => {
                debug!("通知已提交到批处理队列: 用户 {}", user_id);
                Ok(())
            }
            Err(e) => {
                warn!("通知批处理提交失败: 用户 {} - {:?}", user_id, e);
                // 降级到直接发送
                self.transport.send_notification(user_id, &notification).await
            }
        }
    }
    
    /// 批量发送多个通知（直接调用底层传输器）
    pub async fn send_notifications_bulk(&self, notifications: &[(i64, Notification)]) -> DomainResult<()> {
        info!("直接批量发送 {} 个通知", notifications.len());
        self.transport.send_notifications(notifications).await
    }
}

#[async_trait]
impl NotificationTransport for BatchNotificationService {
    async fn send_notification(&self, user_id: i64, notification: &Notification) -> DomainResult<()> {
        // 使用批处理发送
        self.send_notification_batched(user_id, notification.clone()).await
    }
    
    async fn send_notifications(&self, notifications: &[(i64, Notification)]) -> DomainResult<()> {
        // 对于已经是批量的请求，直接调用底层传输器
        self.send_notifications_bulk(notifications).await
    }
    
    async fn send_to_user(&self, user_id: UserId, notification: &Notification) -> DomainResult<()> {
        // 使用批处理发送给单个用户
        self.send_notification_batched(user_id.value(), notification.clone()).await
    }
    
    async fn send_to_users(&self, user_ids: &[UserId], notification: &Notification) -> DomainResult<()> {
        // 将多个用户的通知转换为批量格式
        let notifications: Vec<(i64, Notification)> = user_ids.iter()
            .map(|user_id| (user_id.value(), notification.clone()))
            .collect();
        
        self.send_notifications(&notifications).await
    }
    
    async fn send_to_group(&self, group_id: &str, notification: &Notification) -> DomainResult<()> {
        // 委托给底层传输器
        self.transport.send_to_group(group_id, notification).await
    }
    
    async fn broadcast(&self, notification: &Notification) -> DomainResult<()> {
        // 委托给底层传输器
        self.transport.broadcast(notification).await
    }
    
    async fn broadcast_notification(&self, notification: &Notification) -> DomainResult<()> {
        // 委托给底层传输器
        self.transport.broadcast_notification(notification).await
    }
    
    async fn get_transport_stats(&self) -> DomainResult<TransportStats> {
        // 委托给底层传输器
        self.transport.get_transport_stats().await
    }
}
