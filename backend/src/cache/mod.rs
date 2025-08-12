//! 缓存接口定义
use uuid::Uuid;
use serde_json::Value;
use std::future::Future;
use std::sync::Arc;

// 导出子模块
pub mod group_activity_cache;
pub mod notification_cache;
pub mod location_cache;

// 从子模块中重新导出具体实现（暂不导出未使用实现，避免未使用警告）
// pub use group_activity_cache::RedisGroupActivityCache;


/// 缓存错误类型
#[derive(Debug, thiserror::Error)]
pub enum CacheError {
    #[error("Redis错误: {0}")]
    Redis(#[from] redis::RedisError),
    #[error("序列化错误: {0}")]
    Serialization(#[from] serde_json::Error),
    #[error("缓存连接不可用")]
    ConnectionUnavailable,
    #[error("缓存键不存在")]
    #[allow(dead_code)]
    KeyNotFound,
    #[error("其他缓存错误: {0}")]
    #[allow(dead_code)]
    Other(String),
}

/// 通用缓存接口
pub trait Cache: Send + Sync {
    /// 获取缓存名称
    #[allow(dead_code)]
    fn name(&self) -> &str;
}

/// 用户通知缓存接口
pub trait NotificationCache: Cache {
    /// 存储离线通知
    #[allow(dead_code)]
    fn store_offline_notification<'a>(
        &'a self, 
        user_id: i64, 
        notification: &'a Value,
        expire_days: u32
    ) -> std::pin::Pin<Box<dyn Future<Output = Result<(), CacheError>> + Send + 'a>>;
    
    /// 获取离线通知列表
    #[allow(dead_code)]
    fn get_offline_notifications<'a>(
        &'a self, 
        user_id: i64
    ) -> std::pin::Pin<Box<dyn Future<Output = Result<Vec<String>, CacheError>> + Send + 'a>>;
    
    /// 清除用户的离线通知
    #[allow(dead_code)]
    fn clear_offline_notifications<'a>(
        &'a self, 
        user_id: i64
    ) -> std::pin::Pin<Box<dyn Future<Output = Result<(), CacheError>> + Send + 'a>>;
}

/// 群组活动缓存接口
pub trait GroupActivityCache: Cache {
    /// 记录用户群组活动
    #[allow(dead_code)]
    fn record_user_group_activity(
        &self, 
        group_id: Uuid, 
        user_id: i64, 
        activity_type: &str,
        timestamp: i64,
        score: i32
    ) -> impl Future<Output = Result<(), CacheError>> + Send;
    
    /// 更新用户在群组中的最后活跃时间
    fn update_user_last_active_in_group(
        &self, 
        user_id: i64, 
        group_id: Uuid, 
        timestamp: i64
    ) -> impl Future<Output = Result<(), CacheError>> + Send;
    
    /// 获取群组活跃度排行
    #[allow(dead_code)]
    #[cfg(feature = "group_digest_scheduler")]
    fn get_group_activity_ranking(
        &self, 
        group_id: Uuid,
        limit: usize
    ) -> impl Future<Output = Result<Vec<(i64, f64)>, CacheError>> + Send;
}

// CacheManager struct
#[derive(Clone)]
pub struct CacheManager {
    redis_client: Option<Arc<redis::Client>>,
    redis_connection_manager: Option<Arc<redis::aio::ConnectionManager>>,
}

impl CacheManager {
    pub fn new(
        redis_client: Option<Arc<redis::Client>>,
        redis_conn_manager: Option<Arc<redis::aio::ConnectionManager>>
    ) -> Self {
        Self { 
            redis_client,
            redis_connection_manager: redis_conn_manager 
        }
    }

    pub fn get_redis_client(&self) -> Option<Arc<redis::Client>> {
        self.redis_client.clone()
    }

    pub fn get_redis_connection_manager(&self) -> Option<Arc<redis::aio::ConnectionManager>> {
        self.redis_connection_manager.clone()
    }

    // Optional: Add methods to get specific cache instances if needed
    // pub fn get_group_activity_cache(&self) -> Option<Arc<RedisGroupActivityCache>> {
    //     self.redis_connection_manager.as_ref().map(|conn_manager| {
    //         Arc::new(RedisGroupActivityCache::new(Some(conn_manager.clone())))
    //     })
    // }

    // pub fn get_location_cache(&self) -> Option<Arc<RedisLocationCache>> {
    //     self.redis_connection_manager.as_ref().map(|conn_manager| {
    //         Arc::new(RedisLocationCache::new(conn_manager.clone())) // Assuming this signature for RedisLocationCache
    //     })
    // }
}
