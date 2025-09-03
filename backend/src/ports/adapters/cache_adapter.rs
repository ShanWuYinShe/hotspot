//! 缓存适配器 - 将 CacheManager 适配为 CachePort
//! 
//! 提供缓存系统与端口架构的集成

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use std::time::Duration;

use crate::ports::cache::{CachePort, CachePortError, CachePortResult, CacheStats};
use crate::cache::CacheManager;
use redis::AsyncCommands;

/// 缓存适配器
#[derive(Clone)]
pub struct CacheAdapter {
    manager: Arc<CacheManager>,
}

impl CacheAdapter {
    pub fn new(manager: Arc<CacheManager>) -> Self {
        Self { manager }
    }
}

impl CachePort for CacheAdapter {
    fn name(&self) -> &str {
        "CacheAdapter"
    }

    fn get<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<Option<String>>> + Send + 'a>> {
        let manager = self.manager.clone();
        let key = key.to_string();
        Box::pin(async move {
            if let Some(conn_manager) = manager.get_redis_connection_manager() {
                let mut conn = conn_manager.as_ref().clone();
                conn.get(&key)
                    .await
                    .map_err(|e| CachePortError::Connection(e.to_string()))
            } else {
                Err(CachePortError::Connection("Redis not available".to_string()))
            }
        })
    }

    fn set<'a>(
        &'a self,
        key: &'a str,
        value: &'a str,
        ttl: Option<Duration>,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        let key = key.to_string();
        let value = value.to_string();
        Box::pin(async move {
            if let Some(conn_manager) = manager.get_redis_connection_manager() {
                let mut conn = conn_manager.as_ref().clone();
                if let Some(ttl) = ttl {
                    conn.set_ex(&key, &value, ttl.as_secs())
                        .await
                        .map_err(|e| CachePortError::Connection(e.to_string()))
                } else {
                    conn.set(&key, &value)
                        .await
                        .map_err(|e| CachePortError::Connection(e.to_string()))
                }
            } else {
                Err(CachePortError::Connection("Redis not available".to_string()))
            }
        })
    }

    fn delete<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>> {
        let manager = self.manager.clone();
        let key = key.to_string();
        Box::pin(async move {
            if let Some(conn_manager) = manager.get_redis_connection_manager() {
                let mut conn = conn_manager.as_ref().clone();
                let result: i32 = conn.del(&key)
                    .await
                    .map_err(|e| CachePortError::Connection(e.to_string()))?;
                Ok(result > 0)
            } else {
                Err(CachePortError::Connection("Redis not available".to_string()))
            }
        })
    }

    fn exists<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _key = key.to_string();
        Box::pin(async move {
            // 简化实现：暂时返回 false
            Ok(false)
        })
    }

    fn expire<'a>(
        &'a self,
        key: &'a str,
        ttl: Duration,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _key = key.to_string();
        let _ttl = ttl;
        Box::pin(async move {
            // 简化实现：暂时返回 false
            Ok(false)
        })
    }

    fn ttl<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<Option<Duration>>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _key = key.to_string();
        Box::pin(async move {
            // 简化实现：暂时返回 None
            Ok(None)
        })
    }

    fn mget<'a>(
        &'a self,
        keys: &'a [String],
    ) -> Pin<Box<dyn Future<Output = CachePortResult<Vec<Option<String>>>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let keys = keys.to_vec();
        Box::pin(async move {
            // 简化实现：返回空结果
            Ok(vec![None; keys.len()])
        })
    }

    fn mset<'a>(
        &'a self,
        pairs: &'a [(String, String)],
        ttl: Option<Duration>,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _pairs = pairs.to_vec();
        let _ttl = ttl;
        Box::pin(async move {
            // 简化实现：暂时返回成功
            Ok(())
        })
    }

    fn incr<'a>(
        &'a self,
        key: &'a str,
        delta: i64,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<i64>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _key = key.to_string();
        Box::pin(async move {
            // 简化实现：返回 delta 值
            Ok(delta)
        })
    }

    fn decr<'a>(
        &'a self,
        key: &'a str,
        delta: i64,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<i64>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _key = key.to_string();
        Box::pin(async move {
            // 简化实现：返回 -delta 值
            Ok(-delta)
        })
    }

    fn flush_all<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>> {
        let _manager = self.manager.clone();
        Box::pin(async move {
            // 简化实现：暂时返回成功
            Ok(())
        })
    }

    fn stats<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<CacheStats>> + Send + 'a>> {
        let _manager = self.manager.clone();
        Box::pin(async move {
            // 简化实现：返回默认统计信息
            Ok(CacheStats::default())
        })
    }
}
