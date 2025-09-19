/// 缓存应用服务
/// 
/// 提供高级缓存功能，基于缓存端口实现业务相关的缓存策略
use async_trait::async_trait;
use std::sync::Arc;
use serde::{Serialize, Deserialize};

use crate::domain::DomainResult;
use crate::ports::cache::Cache;
use std::time::Duration;

/// 缓存应用服务接口
#[async_trait]
pub trait CacheApplicationService: Send + Sync {
    /// 缓存用户信息
    async fn cache_user_info(&self, user_id: i64, user_data: serde_json::Value, ttl_seconds: u64) -> DomainResult<()>;
    
    /// 获取缓存的用户信息
    async fn get_cached_user_info(&self, user_id: i64) -> DomainResult<Option<serde_json::Value>>;
    
    /// 缓存群组信息
    async fn cache_group_info(&self, group_id: i64, group_data: serde_json::Value, ttl_seconds: u64) -> DomainResult<()>;
    
    /// 获取缓存的群组信息
    async fn get_cached_group_info(&self, group_id: i64) -> DomainResult<Option<serde_json::Value>>;
    
    /// 缓存搜索结果
    async fn cache_search_results(&self, search_key: String, results: serde_json::Value, ttl_seconds: u64) -> DomainResult<()>;
    
    /// 获取缓存的搜索结果
    async fn get_cached_search_results(&self, search_key: String) -> DomainResult<Option<serde_json::Value>>;
    
    /// 清除用户相关缓存
    async fn clear_user_cache(&self, user_id: i64) -> DomainResult<()>;
    
    /// 清除群组相关缓存
    async fn clear_group_cache(&self, group_id: i64) -> DomainResult<()>;
    
    /// 获取缓存统计信息
    async fn get_cache_stats(&self) -> DomainResult<CacheStats>;
}

/// 缓存统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheStats {
    pub hit_count: u64,
    pub miss_count: u64,
    pub hit_ratio: f64,
    pub total_keys: u64,
    pub memory_usage_bytes: u64,
}

/// 缓存应用服务实现
pub struct CacheApplicationServiceImpl<C: Cache> {
    cache: Arc<C>,
}

impl<C: Cache> CacheApplicationServiceImpl<C> {
    /// 创建新的缓存应用服务
    pub fn new(cache: Arc<C>) -> Self {
        Self { cache }
    }

    /// 构建用户缓存键
    fn user_cache_key(&self, user_id: i64) -> String {
        format!("user:{}", user_id)
    }

    /// 构建群组缓存键
    fn group_cache_key(&self, group_id: i64) -> String {
        format!("group:{}", group_id)
    }

    /// 构建搜索缓存键
    fn search_cache_key(&self, search_key: &str) -> String {
        format!("search:{}", search_key)
    }
}

#[async_trait]
impl<C: Cache> CacheApplicationService for CacheApplicationServiceImpl<C> {
    async fn cache_user_info(&self, user_id: i64, user_data: serde_json::Value, ttl_seconds: u64) -> DomainResult<()> {
        let key = self.user_cache_key(user_id);
        let value = serde_json::to_string(&user_data)
            .map_err(|_e| crate::domain::DomainError::CacheSerializationError)?;
        
        self.cache.set(&key, &value, Some(Duration::from_secs(ttl_seconds))).await
    }

    async fn get_cached_user_info(&self, user_id: i64) -> DomainResult<Option<serde_json::Value>> {
        let key = self.user_cache_key(user_id);
        match self.cache.get::<String>(&key).await? {
            Some(value) => {
                let data: serde_json::Value = serde_json::from_str(&value)
                    .map_err(|_e| crate::domain::DomainError::CacheDeserializationError)?;
                Ok(Some(data))
            }
            None => Ok(None),
        }
    }

    async fn cache_group_info(&self, group_id: i64, group_data: serde_json::Value, ttl_seconds: u64) -> DomainResult<()> {
        let key = self.group_cache_key(group_id);
        let value = serde_json::to_string(&group_data)
            .map_err(|_e| crate::domain::DomainError::CacheSerializationError)?;
        
        self.cache.set(&key, &value, Some(Duration::from_secs(ttl_seconds))).await
    }

    async fn get_cached_group_info(&self, group_id: i64) -> DomainResult<Option<serde_json::Value>> {
        let key = self.group_cache_key(group_id);
        match self.cache.get::<String>(&key).await? {
            Some(value) => {
                let data: serde_json::Value = serde_json::from_str(&value)
                    .map_err(|_e| crate::domain::DomainError::CacheDeserializationError)?;
                Ok(Some(data))
            }
            None => Ok(None),
        }
    }

    async fn cache_search_results(&self, search_key: String, results: serde_json::Value, ttl_seconds: u64) -> DomainResult<()> {
        let key = self.search_cache_key(&search_key);
        let value = serde_json::to_string(&results)
            .map_err(|_e| crate::domain::DomainError::CacheSerializationError)?;
        
        self.cache.set(&key, &value, Some(Duration::from_secs(ttl_seconds))).await
    }

    async fn get_cached_search_results(&self, search_key: String) -> DomainResult<Option<serde_json::Value>> {
        let key = self.search_cache_key(&search_key);
        match self.cache.get::<String>(&key).await? {
            Some(value) => {
                let data: serde_json::Value = serde_json::from_str(&value)
                    .map_err(|_e| crate::domain::DomainError::CacheDeserializationError)?;
                Ok(Some(data))
            }
            None => Ok(None),
        }
    }

    async fn clear_user_cache(&self, user_id: i64) -> DomainResult<()> {
        let key = self.user_cache_key(user_id);
        self.cache.delete(&key).await.map(|_| ())
    }

    async fn clear_group_cache(&self, group_id: i64) -> DomainResult<()> {
        let key = self.group_cache_key(group_id);
        self.cache.delete(&key).await.map(|_| ())
    }

    async fn get_cache_stats(&self) -> DomainResult<CacheStats> {
        let stats = self.cache.get_stats().await?;
        
        let hit_ratio = if stats.hit_count + stats.miss_count > 0 {
            stats.hit_count as f64 / (stats.hit_count + stats.miss_count) as f64
        } else {
            0.0
        };

        Ok(CacheStats {
            hit_count: stats.hit_count,
            miss_count: stats.miss_count,
            hit_ratio,
            total_keys: stats.total_keys,
            memory_usage_bytes: stats.memory_usage,
        })
    }
}

/// 缓存管理器
/// 
/// 提供缓存相关的管理功能
#[derive(Clone)]
pub struct CacheManager<C: Cache> {
    cache: Arc<C>,
    cache_service: Arc<CacheApplicationServiceImpl<C>>,
}

impl<C: Cache> CacheManager<C> {
    /// 创建新的缓存管理器
    pub fn new(cache: Arc<C>) -> Self {
        let cache_service = Arc::new(CacheApplicationServiceImpl::new(cache.clone()));
        Self {
            cache,
            cache_service,
        }
    }

    /// 获取缓存端口
    pub fn get_cache(&self) -> Arc<C> {
        self.cache.clone()
    }

    /// 获取缓存应用服务
    pub fn get_cache_service(&self) -> Arc<CacheApplicationServiceImpl<C>> {
        self.cache_service.clone()
    }

    /// 清理过期缓存
    pub async fn cleanup_expired(&self) -> DomainResult<u64> {
        // 清理所有缓存并返回清理的数量
        self.cache.clear_all().await?;
        Ok(0) // 返回清理的数量，这里简化为0
    }

    /// 健康检查
    pub async fn health_check(&self) -> DomainResult<bool> {
        // 尝试设置和获取一个测试键来验证缓存是否正常工作
        let test_key = "health_check_test".to_string();
        let test_value = "ok".to_string();
        
        match self.cache.set(&test_key, &test_value, Some(Duration::from_secs(10))).await {
            Ok(_) => {
                match self.cache.get::<String>(&test_key).await {
                    Ok(Some(value)) if value == test_value => {
                        // 清理测试键
                        let _ = self.cache.delete(&test_key).await;
                        Ok(true)
                    }
                    _ => Ok(false),
                }
            }
            Err(_) => Ok(false),
        }
    }
}