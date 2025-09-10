/// Redis缓存适配器实现
/// 
/// 提供高性能的Redis缓存服务，支持热点数据缓存和分布式缓存
use async_trait::async_trait;
use redis::{AsyncCommands, Client, RedisResult};
use serde::{Deserialize, Serialize};
use std::time::Duration;
use tracing::{debug, error, warn};

use crate::domain::{DomainError, DomainResult};
use crate::ports::CachePort;

/// Redis缓存配置
#[derive(Debug, Clone)]
pub struct RedisCacheConfig {
    /// Redis连接URL
    pub redis_url: String,
    /// 默认TTL（秒）
    pub default_ttl: u64,
    /// 连接池大小
    pub pool_size: u32,
    /// 连接超时时间（毫秒）
    pub connection_timeout_ms: u64,
}

impl Default for RedisCacheConfig {
    fn default() -> Self {
        Self {
            redis_url: "redis://localhost:6379".to_string(),
            default_ttl: 300, // 5分钟
            pool_size: 10,
            connection_timeout_ms: 5000,
        }
    }
}

/// Redis缓存适配器
#[derive(Clone)]
pub struct RedisCacheAdapter {
    client: Client,
    config: RedisCacheConfig,
}

impl RedisCacheAdapter {
    /// 创建新的Redis缓存适配器
    pub async fn new(config: RedisCacheConfig) -> DomainResult<Self> {
        let client = Client::open(config.redis_url.clone())
            .map_err(|e| DomainError::ExternalServiceError {
                service: "Redis".to_string(),
                error: format!("Failed to create Redis client: {}", e),
            })?;

        // 测试连接
        let mut conn = client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| DomainError::ExternalServiceError {
                service: "Redis".to_string(),
                error: format!("Failed to connect to Redis: {}", e),
            })?;

        // 执行ping测试
        let _: String = conn
            .ping()
            .await
            .map_err(|e| DomainError::ExternalServiceError {
                service: "Redis".to_string(),
                error: format!("Redis ping failed: {}", e),
            })?;

        debug!("Redis cache adapter initialized successfully");

        Ok(Self { client, config })
    }

    /// 获取Redis连接
    async fn get_connection(&self) -> DomainResult<redis::aio::MultiplexedConnection> {
        self.client
            .get_multiplexed_async_connection()
            .await
            .map_err(|e| DomainError::ExternalServiceError {
                service: "Redis".to_string(),
                error: format!("Failed to get Redis connection: {}", e),
            })
    }

    /// 构建缓存键
    fn build_key(&self, namespace: &str, key: &str) -> String {
        format!("hotspot:{}:{}", namespace, key)
    }
}

#[async_trait]
impl CachePort for RedisCacheAdapter {
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send,
    {
        let cache_key = format!("hotspot:{}", key);
        let mut conn = self.get_connection().await?;

        let result: RedisResult<String> = conn.get(&cache_key).await;

        match result {
            Ok(value) => {
                match serde_json::from_str::<T>(&value) {
                    Ok(deserialized) => {
                        debug!(key = %cache_key, "Cache hit");
                        Ok(Some(deserialized))
                    }
                    Err(e) => {
                        warn!(key = %cache_key, error = %e, "Failed to deserialize cached value");
                        // 删除损坏的缓存项
                        let _: RedisResult<()> = conn.del(&cache_key).await;
                        Ok(None)
                    }
                }
            }
            Err(redis::RedisError { kind: redis::ErrorKind::TypeError, .. }) => {
                debug!(key = %cache_key, "Cache miss");
                Ok(None)
            }
            Err(e) => {
                error!(key = %cache_key, error = %e, "Redis get operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Get operation failed: {}", e),
                })
            }
        }
    }

    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync,
    {
        let cache_key = format!("hotspot:{}", key);
        let mut conn = self.get_connection().await?;

        let serialized = serde_json::to_string(value)
            .map_err(|e| DomainError::ValidationError {
                field: "cache_value".to_string(),
                message: format!("Failed to serialize value: {}", e),
            })?;

        let ttl_seconds = ttl
            .map(|d| d.as_secs())
            .unwrap_or(self.config.default_ttl);

        let result: RedisResult<()> = conn.set_ex(&cache_key, &serialized, ttl_seconds).await;

        match result {
            Ok(_) => {
                debug!(key = %cache_key, ttl = ttl_seconds, "Cache set successfully");
                Ok(())
            }
            Err(e) => {
                error!(key = %cache_key, error = %e, "Redis set operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Set operation failed: {}", e),
                })
            }
        }
    }

    async fn delete(&self, key: &str) -> DomainResult<()> {
        let cache_key = format!("hotspot:{}", key);
        let mut conn = self.get_connection().await?;

        let result: RedisResult<i32> = conn.del(&cache_key).await;

        match result {
            Ok(deleted_count) => {
                debug!(key = %cache_key, deleted = deleted_count, "Cache delete completed");
                Ok(())
            }
            Err(e) => {
                error!(key = %cache_key, error = %e, "Redis delete operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Delete operation failed: {}", e),
                })
            }
        }
    }

    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()> {
        if keys.is_empty() {
            return Ok(());
        }

        let cache_keys: Vec<String> = keys.iter()
            .map(|key| format!("hotspot:{}", key))
            .collect();

        let mut conn = self.get_connection().await?;
        let result: RedisResult<i32> = conn.del(&cache_keys).await;

        match result {
            Ok(deleted_count) => {
                debug!(deleted = deleted_count, "Batch delete completed");
                Ok(())
            }
            Err(e) => {
                error!(error = %e, "Redis batch delete operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Batch delete operation failed: {}", e),
                })
            }
        }
    }

    async fn exists(&self, key: &str) -> DomainResult<bool> {
        let cache_key = format!("hotspot:{}", key);
        let mut conn = self.get_connection().await?;

        let result: RedisResult<i32> = conn.exists(&cache_key).await;

        match result {
            Ok(exists) => {
                debug!(key = %cache_key, exists = exists > 0, "Cache exists check completed");
                Ok(exists > 0)
            }
            Err(e) => {
                error!(key = %cache_key, error = %e, "Redis exists operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Exists operation failed: {}", e),
                })
            }
        }
    }

    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool> {
        let cache_key = format!("hotspot:{}", key);
        let mut conn = self.get_connection().await?;

        let result: RedisResult<i32> = conn.expire(&cache_key, ttl.as_secs() as i64).await;

        match result {
            Ok(success) => {
                debug!(key = %cache_key, ttl = ?ttl, success = success == 1, "Cache expire completed");
                Ok(success == 1)
            }
            Err(e) => {
                error!(key = %cache_key, error = %e, "Redis expire operation failed");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Expire operation failed: {}", e),
                })
            }
        }
    }

    async fn clear_all(&self) -> DomainResult<()> {
        let pattern = "hotspot:*";
        let mut conn = self.get_connection().await?;

        // 使用SCAN命令获取所有匹配的键
        let mut cursor = 0;
        let mut total_deleted = 0;

        loop {
            let result: RedisResult<(u64, Vec<String>)> = redis::cmd("SCAN")
                .arg(cursor)
                .arg("MATCH")
                .arg(pattern)
                .arg("COUNT")
                .arg(100)
                .query_async(&mut conn)
                .await;

            match result {
                Ok((new_cursor, keys)) => {
                    if !keys.is_empty() {
                        let deleted: RedisResult<i32> = conn.del(&keys).await;
                        match deleted {
                            Ok(count) => total_deleted += count,
                            Err(e) => {
                                error!(pattern = %pattern, error = %e, "Failed to delete keys");
                            }
                        }
                    }

                    cursor = new_cursor;
                    if cursor == 0 {
                        break;
                    }
                }
                Err(e) => {
                    error!(pattern = %pattern, error = %e, "SCAN operation failed");
                    return Err(DomainError::ExternalServiceError {
                        service: "Redis".to_string(),
                        error: format!("Clear all failed: {}", e),
                    });
                }
            }
        }

        debug!(deleted = total_deleted, "All cache cleared");
        Ok(())
    }

    async fn get_stats(&self) -> DomainResult<crate::ports::cache_ports::CacheStats> {
        let mut conn = self.get_connection().await?;

        // 获取基本统计信息
        let info_result: RedisResult<redis::InfoDict> = redis::cmd("INFO")
            .arg("stats")
            .query_async(&mut conn)
            .await;

        let memory_result: RedisResult<redis::InfoDict> = redis::cmd("INFO")
            .arg("memory")
            .query_async(&mut conn)
            .await;

        // 获取键数量
        let dbsize_result: RedisResult<u64> = redis::cmd("DBSIZE")
            .query_async(&mut conn)
            .await;

        match (info_result, memory_result, dbsize_result) {
            (Ok(stats_info), Ok(memory_info), Ok(entries)) => {
                // 从Redis INFO中提取统计信息
                let hits = stats_info.get::<u64>("keyspace_hits").unwrap_or(0);
                let misses = stats_info.get::<u64>("keyspace_misses").unwrap_or(0);
                let memory_usage = memory_info.get::<u64>("used_memory").unwrap_or(0);

                let cache_stats = crate::ports::cache_ports::CacheStats {
                    hits,
                    misses,
                    entries,
                    memory_usage,
                };

                debug!("Retrieved Redis cache stats: {:?}", cache_stats);
                Ok(cache_stats)
            }
            (Err(e), _, _) | (_, Err(e), _) | (_, _, Err(e)) => {
                error!(error = %e, "Failed to get Redis stats");
                Err(DomainError::ExternalServiceError {
                    service: "Redis".to_string(),
                    error: format!("Get stats failed: {}", e),
                })
            }
        }
    }
}

/// 缓存策略配置
#[derive(Debug, Clone)]
pub struct CacheStrategy {
    /// 用户数据缓存TTL（秒）
    pub user_cache_ttl: u64,
    /// 群组数据缓存TTL（秒）
    pub group_cache_ttl: u64,
    /// 地理位置查询缓存TTL（秒）
    pub geo_cache_ttl: u64,
    /// 消息缓存TTL（秒）
    pub message_cache_ttl: u64,
    /// 瞬间缓存TTL（秒）
    pub moment_cache_ttl: u64,
}

impl Default for CacheStrategy {
    fn default() -> Self {
        Self {
            user_cache_ttl: 600,     // 10分钟
            group_cache_ttl: 300,    // 5分钟
            geo_cache_ttl: 180,      // 3分钟
            message_cache_ttl: 1800, // 30分钟
            moment_cache_ttl: 600,   // 10分钟
        }
    }
}

/// 缓存键生成器
pub struct CacheKeyGenerator;

impl CacheKeyGenerator {
    /// 用户缓存键
    pub fn user_key(user_id: i64) -> String {
        format!("user:{}", user_id)
    }

    /// 用户资料缓存键
    pub fn user_profile_key(user_id: i64) -> String {
        format!("user:profile:{}", user_id)
    }

    /// 群组缓存键
    pub fn group_key(group_id: &str) -> String {
        format!("group:{}", group_id)
    }

    /// 群组成员缓存键
    pub fn group_members_key(group_id: &str, cursor: i64, limit: i32) -> String {
        format!("group:members:{}:{}:{}", group_id, cursor, limit)
    }

    /// 附近用户缓存键
    pub fn nearby_users_key(lat: f64, lng: f64, radius: f64, cursor: i64, limit: i32) -> String {
        format!("nearby:users:{:.6}:{:.6}:{}:{}:{}", lat, lng, radius, cursor, limit)
    }

    /// 附近群组缓存键
    pub fn nearby_groups_key(lat: f64, lng: f64, radius: f64, cursor: i64, limit: i32) -> String {
        format!("nearby:groups:{:.6}:{:.6}:{}:{}:{}", lat, lng, radius, cursor, limit)
    }

    /// 附近瞬间缓存键
    pub fn nearby_moments_key(lat: f64, lng: f64, radius: f64, cursor: i64, limit: i32) -> String {
        format!("nearby:moments:{:.6}:{:.6}:{}:{}:{}", lat, lng, radius, cursor, limit)
    }

    /// 用户搜索缓存键
    pub fn user_search_key(query: &str, cursor: i64, limit: i32) -> String {
        format!("search:users:{}:{}:{}", query, cursor, limit)
    }

    /// 群组搜索缓存键
    pub fn group_search_key(query: &str, cursor: i64, limit: i32) -> String {
        format!("search:groups:{}:{}:{}", query, cursor, limit)
    }

    /// 消息历史缓存键
    pub fn message_history_key(room_id: &str, room_type: &str, cursor: i64, limit: i32) -> String {
        format!("messages:{}:{}:{}:{}", room_type, room_id, cursor, limit)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cache_key_generation() {
        assert_eq!(CacheKeyGenerator::user_key(123), "user:123");
        assert_eq!(CacheKeyGenerator::user_profile_key(456), "user:profile:456");
        assert_eq!(CacheKeyGenerator::group_key("group-uuid"), "group:group-uuid");
        
        let nearby_key = CacheKeyGenerator::nearby_users_key(39.916527, 116.397128, 1000.0, 0, 20);
        assert_eq!(nearby_key, "nearby:users:39.916527:116.397128:1000:0:20");
    }

    #[test]
    fn test_redis_cache_config_default() {
        let config = RedisCacheConfig::default();
        assert_eq!(config.redis_url, "redis://localhost:6379");
        assert_eq!(config.default_ttl, 300);
        assert_eq!(config.pool_size, 10);
    }

    #[test]
    fn test_cache_strategy_default() {
        let strategy = CacheStrategy::default();
        assert_eq!(strategy.user_cache_ttl, 600);
        assert_eq!(strategy.group_cache_ttl, 300);
        assert_eq!(strategy.geo_cache_ttl, 180);
    }
}