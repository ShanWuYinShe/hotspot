//! Redis实现的通知缓存
use std::sync::Arc;
use serde_json::Value;

use crate::cache::{Cache, NotificationCache, CacheError};

/// Redis实现的通知缓存
pub struct RedisNotificationCache {
    /// Redis连接管理器
    redis: Option<Arc<redis::aio::ConnectionManager>>,
}

impl RedisNotificationCache {
    /// 创建新的Redis通知缓存
    #[allow(dead_code)]
    pub fn new(redis: Option<Arc<redis::aio::ConnectionManager>>) -> Self {
        Self { redis }
    }
}

impl Cache for RedisNotificationCache {
    fn name(&self) -> &str {
        "redis_notification_cache"
    }
}

impl NotificationCache for RedisNotificationCache {
    fn store_offline_notification<'a>(
        &'a self, 
        user_id: i64, 
        notification: &'a Value,
        expire_days: u32
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), CacheError>> + Send + 'a>> {
        Box::pin(async move {
            let redis = match &self.redis {
                Some(redis) => redis,
                None => return Err(CacheError::ConnectionUnavailable),
            };
            
            let mut conn = (**redis).clone();
            
            // 存储到Redis的用户离线通知队列
            let _: i64 = redis::cmd("LPUSH")
                .arg(format!("user:{}:offline_notifications", user_id))
                .arg(notification.to_string())
                .query_async(&mut conn)
                .await?;
            
            // 修剪列表保持合理大小
            let _: () = redis::cmd("LTRIM")
                .arg(format!("user:{}:offline_notifications", user_id))
                .arg(0)
                .arg(99) // 保留最新的100条
                .query_async(&mut conn)
                .await?;
            
            // 设置通知过期时间
            let _: () = redis::cmd("EXPIRE")
                .arg(format!("user:{}:offline_notifications", user_id))
                .arg(86400 * expire_days as i64)
                .query_async(&mut conn)
                .await?;
                
            Ok(())
        })
    }
    
    fn get_offline_notifications<'a>(
        &'a self, 
        user_id: i64
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<Vec<String>, CacheError>> + Send + 'a>> {
        Box::pin(async move {
            let redis = match &self.redis {
                Some(redis) => redis,
                None => return Err(CacheError::ConnectionUnavailable),
            };
            
            let mut conn = (**redis).clone();
            
            // 获取离线通知
            let notifications: Vec<String> = redis::cmd("LRANGE")
                .arg(format!("user:{}:offline_notifications", user_id))
                .arg(0)
                .arg(-1)
                .query_async(&mut conn)
                .await?;
                
            Ok(notifications)
        })
    }
    
    fn clear_offline_notifications<'a>(
        &'a self, 
        user_id: i64
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), CacheError>> + Send + 'a>> {
        Box::pin(async move {
            let redis = match &self.redis {
                Some(redis) => redis,
                None => return Err(CacheError::ConnectionUnavailable),
            };
            
            let mut conn = (**redis).clone();
            
            // 删除用户的离线通知队列
            let _: () = redis::cmd("DEL")
                .arg(format!("user:{}:offline_notifications", user_id))
                .query_async(&mut conn)
                .await?;
                
            Ok(())
        })
    }
}