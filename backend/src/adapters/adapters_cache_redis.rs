/// Redis缓存适配器
/// 
/// 基于Redis的缓存实现

use crate::ports::{CachePort, CacheStats};
use crate::domain::DomainResult;
use async_trait::async_trait;
use std::time::Duration;
use redis::{Client, Connection, AsyncCommands, RedisResult};
use redis::aio::ConnectionManager;
use serde::{Serialize, Deserialize};
use tracing::{debug, warn, error};

/// Redis缓存适配器
#[derive(Clone)]
pub struct RedisCache {
    connection_manager: ConnectionManager,
}

impl RedisCache {
    /// 从配置创建Redis缓存适配器
    pub async fn from_config(redis_url: &str) -> DomainResult<Self> {
        debug!("正在连接Redis: {}", redis_url);
        
        let client = Client::open(redis_url)
            .map_err(|e| crate::domain::DomainError::Cache(
                crate::domain::CacheDomainError::ConnectionFailed
            ))?;
            
        let connection_manager = ConnectionManager::new(client).await
            .map_err(|e| {
                error!("Redis连接管理器创建失败: {}", e);
                crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ConnectionFailed
                )
            })?;
            
        debug!("Redis连接建立成功");
        Ok(Self { connection_manager })
    }
    
    /// 序列化值为JSON字符串
    fn serialize_value<T: Serialize>(&self, value: &T) -> DomainResult<String> {
        serde_json::to_string(value)
            .map_err(|e| crate::domain::DomainError::Cache(
                crate::domain::CacheDomainError::SerializationError(e.to_string())
            ))
    }
    
    /// 反序列化JSON字符串为值
    fn deserialize_value<T: for<'de> Deserialize<'de>>(&self, data: &str) -> DomainResult<T> {
        serde_json::from_str(data)
            .map_err(|e| crate::domain::DomainError::Cache(
                crate::domain::CacheDomainError::DeserializationError(e.to_string())
            ))
    }
}

#[async_trait]
impl CachePort for RedisCache {
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send,
    {
        debug!("Redis缓存GET请求: {}", key);
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<Option<String>> = conn.get(key).await;
        
        match result {
            Ok(Some(data)) => {
                debug!("Redis缓存命中: {}", key);
                let value = self.deserialize_value(&data)?;
                Ok(Some(value))
            }
            Ok(None) => {
                debug!("Redis缓存未命中: {}", key);
                Ok(None)
            }
            Err(e) => {
                error!("Redis GET操作失败: {} - {}", key, e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync,
    {
        debug!("Redis缓存SET请求: {} (TTL: {:?})", key, ttl);
        
        let serialized_value = self.serialize_value(value)?;
        let mut conn = self.connection_manager.clone();
        
        let result = match ttl {
            Some(duration) => {
                let seconds = duration.as_secs();
                conn.set_ex(key, serialized_value, seconds).await
            }
            None => {
                conn.set(key, serialized_value).await
            }
        };
        
        match result {
            Ok(_) => {
                debug!("Redis缓存设置成功: {}", key);
                Ok(())
            }
            Err(e) => {
                error!("Redis SET操作失败: {} - {}", key, e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn delete(&self, key: &str) -> DomainResult<()> {
        debug!("Redis缓存DELETE请求: {}", key);
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<i32> = conn.del(key).await;
        
        match result {
            Ok(_) => {
                debug!("Redis缓存删除成功: {}", key);
                Ok(())
            }
            Err(e) => {
                error!("Redis DELETE操作失败: {} - {}", key, e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()> {
        debug!("Redis缓存批量DELETE请求: {} keys", keys.len());
        
        if keys.is_empty() {
            return Ok(());
        }
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<i32> = conn.del(keys).await;
        
        match result {
            Ok(deleted_count) => {
                debug!("Redis缓存批量删除成功: {} keys", deleted_count);
                Ok(())
            }
            Err(e) => {
                error!("Redis批量DELETE操作失败: {}", e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn exists(&self, key: &str) -> DomainResult<bool> {
        debug!("Redis缓存EXISTS请求: {}", key);
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<bool> = conn.exists(key).await;
        
        match result {
            Ok(exists) => {
                debug!("Redis EXISTS结果: {} = {}", key, exists);
                Ok(exists)
            }
            Err(e) => {
                error!("Redis EXISTS操作失败: {} - {}", key, e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool> {
        debug!("Redis缓存EXPIRE请求: {} (TTL: {:?})", key, ttl);
        
        let mut conn = self.connection_manager.clone();
        let seconds = ttl.as_secs();
        let result: RedisResult<bool> = conn.expire(key, seconds as usize).await;
        
        match result {
            Ok(success) => {
                debug!("Redis EXPIRE结果: {} = {}", key, success);
                Ok(success)
            }
            Err(e) => {
                error!("Redis EXPIRE操作失败: {} - {}", key, e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn clear_all(&self) -> DomainResult<()> {
        warn!("Redis缓存CLEAR_ALL请求 - 清空所有缓存数据");
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<String> = conn.flushdb().await;
        
        match result {
            Ok(_) => {
                warn!("Redis缓存已清空");
                Ok(())
            }
            Err(e) => {
                error!("Redis FLUSHDB操作失败: {}", e);
                Err(crate::domain::DomainError::Cache(
                    crate::domain::CacheDomainError::ServiceUnavailable
                ))
            }
        }
    }

    async fn get_stats(&self) -> DomainResult<CacheStats> {
        debug!("Redis缓存统计信息请求");
        
        let mut conn = self.connection_manager.clone();
        let result: RedisResult<String> = redis::cmd("INFO").arg("stats").query_async(&mut conn).await;
        
        match result {
            Ok(info_string) => {
                // 解析Redis INFO stats输出
                let stats = self.parse_redis_stats(&info_string);
                debug!("Redis统计信息: {:?}", stats);
                Ok(stats)
            }
            Err(e) => {
                error!("Redis INFO操作失败: {}", e);
                // 返回默认统计信息而不是错误
                Ok(CacheStats {
                    hits: 0,
                    misses: 0,
                    entries: 0,
                    memory_usage: 0,
                })
            }
        }
    }
}

impl RedisCache {
    /// 解析Redis INFO stats输出
    fn parse_redis_stats(&self, info_string: &str) -> CacheStats {
        let mut hits = 0;
        let mut misses = 0;
        let mut entries = 0;
        let mut memory_usage = 0;
        
        for line in info_string.lines() {
            if let Some((key, value)) = line.split_once(':') {
                match key {
                    "keyspace_hits" => {
                        hits = value.parse().unwrap_or(0);
                    }
                    "keyspace_misses" => {
                        misses = value.parse().unwrap_or(0);
                    }
                    "db0" => {
                        // 解析 "keys=123,expires=45,avg_ttl=67890"
                        if let Some(keys_part) = value.split(',').next() {
                            if let Some(keys_value) = keys_part.strip_prefix("keys=") {
                                entries = keys_value.parse().unwrap_or(0);
                            }
                        }
                    }
                    "used_memory" => {
                        memory_usage = value.parse().unwrap_or(0);
                    }
                    _ => {}
                }
            }
        }
        
        CacheStats {
            hits,
            misses,
            entries,
            memory_usage,
        }
    }
}
