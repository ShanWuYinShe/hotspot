/// 缓存服务
/// 
/// 提供统一的缓存管理接口，支持多种缓存策略和自动失效
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::Duration;
use tracing::debug;

use crate::domain::{DomainResult, UserProfile};
// use crate::adapters::adapters_cache_memory::{MemoryCache, MemoryCacheConfig};  // 暂时禁用
use crate::ports::cache_ports::CachePort;
use crate::ports::cache::Cache;

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

    /// 用户搜索缓存键
    pub fn user_search_key(query: &str, cursor: i64, limit: i32) -> String {
        format!("search:users:{}:{}:{}", query, cursor, limit)
    }

    /// 群组搜索缓存键
    pub fn group_search_key(query: &str, cursor: i64, limit: i32) -> String {
        format!("search:groups:{}:{}:{}", query, cursor, limit)
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

/// 缓存服务接口
#[async_trait]
pub trait CacheService: Send + Sync {
    /// 获取用户缓存
    async fn get_user(&self, user_id: i64) -> DomainResult<Option<UserProfile>>;
    
    /// 设置用户缓存
    async fn set_user(&self, user: &UserProfile) -> DomainResult<()>;
    
    /// 删除用户缓存
    async fn invalidate_user(&self, user_id: i64) -> DomainResult<()>;
    
    /// 获取群组缓存
    async fn get_group(&self, group_id: &str) -> DomainResult<Option<crate::domain::GroupProfile>>;
    
    /// 设置群组缓存
    async fn set_group(&self, group: &crate::domain::GroupProfile) -> DomainResult<()>;
    
    /// 删除群组缓存
    async fn invalidate_group(&self, group_id: &str) -> DomainResult<()>;
    
    /// 获取附近用户缓存
    async fn get_nearby_users(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>>;
    
    /// 设置附近用户缓存
    async fn set_nearby_users(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32, 
        users: &[UserProfile]
    ) -> DomainResult<()>;
    
    /// 获取附近群组缓存
    async fn get_nearby_groups(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<crate::domain::GroupProfile>>>;
    
    /// 设置附近群组缓存
    async fn set_nearby_groups(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32, 
        groups: &[crate::domain::GroupProfile]
    ) -> DomainResult<()>;
    
    /// 获取用户搜索结果缓存
    async fn get_user_search(
        &self, 
        query: &str, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>>;
    
    /// 设置用户搜索结果缓存
    async fn set_user_search(
        &self, 
        query: &str, 
        cursor: i64, 
        limit: i32, 
        users: &[UserProfile]
    ) -> DomainResult<()>;
    
    /// 清理所有缓存
    async fn clear_all(&self) -> DomainResult<()>;
    
    /// 获取缓存统计信息
    async fn get_cache_stats(&self) -> DomainResult<CacheStats>;
}

/// 缓存统计信息
#[derive(Debug, Serialize, Deserialize)]
pub struct CacheStats {
    pub total_keys: u64,
    pub memory_usage: String,
    pub hit_rate: f64,
    pub miss_rate: f64,
}

// /// 基于内存缓存的缓存服务实现 - 暂时禁用
// pub struct InMemoryCacheService {
//     cache: Arc<MemoryCache>,
//     strategy: CacheStrategy,
// }

// impl InMemoryCacheService {
    pub fn new(strategy: CacheStrategy) -> Self {
        let config = MemoryCacheConfig::default();
        Self { 
            cache: Arc::new(MemoryCache::new(config)),
            strategy,
        }
    }
}

#[async_trait]
impl CacheService for InMemoryCacheService {
    async fn get_user(&self, user_id: i64) -> DomainResult<Option<UserProfile>> {
        let key = CacheKeyGenerator::user_key(user_id);
        self.cache.get(&key).await
    }
    
    async fn set_user(&self, user: &UserProfile) -> DomainResult<()> {
        let key = CacheKeyGenerator::user_key(user.user_id);
        let ttl = Duration::from_secs(self.strategy.user_cache_ttl);
        self.cache.set(&key, user, Some(ttl)).await
    }
    
    async fn invalidate_user(&self, user_id: i64) -> DomainResult<()> {
        let key = CacheKeyGenerator::user_key(user_id);
        self.cache.delete(&key).await?;
        
        // 同时清理用户相关的其他缓存
        let profile_key = CacheKeyGenerator::user_profile_key(user_id);
        self.cache.delete(&profile_key).await?;
        
        debug!(user_id = user_id, "User cache invalidated");
        Ok(())
    }
    
    async fn get_group(&self, group_id: &str) -> DomainResult<Option<crate::domain::GroupProfile>> {
        let key = CacheKeyGenerator::group_key(group_id);
        self.cache.get(&key).await
    }
    
    async fn set_group(&self, group: &crate::domain::GroupProfile) -> DomainResult<()> {
        let key = CacheKeyGenerator::group_key(&group.group_id);
        let ttl = Duration::from_secs(self.strategy.group_cache_ttl);
        self.cache.set(&key, group, Some(ttl)).await
    }
    
    async fn invalidate_group(&self, group_id: &str) -> DomainResult<()> {
        let key = CacheKeyGenerator::group_key(group_id);
        self.cache.delete(&key).await?;
        
        debug!(group_id = group_id, "Group cache invalidated");
        Ok(())
    }
    
    async fn get_nearby_users(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>> {
        let key = CacheKeyGenerator::nearby_users_key(lat, lng, radius, cursor, limit);
        self.cache.get(&key).await
    }
    
    async fn set_nearby_users(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32, 
        users: &[UserProfile]
    ) -> DomainResult<()> {
        let key = CacheKeyGenerator::nearby_users_key(lat, lng, radius, cursor, limit);
        let ttl = Duration::from_secs(self.strategy.geo_cache_ttl);
        self.cache.set(&key, &users.to_vec(), Some(ttl)).await
    }
    
    async fn get_nearby_groups(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<crate::domain::GroupProfile>>> {
        let key = CacheKeyGenerator::nearby_groups_key(lat, lng, radius, cursor, limit);
        self.cache.get(&key).await
    }
    
    async fn set_nearby_groups(
        &self, 
        lat: f64, 
        lng: f64, 
        radius: f64, 
        cursor: i64, 
        limit: i32, 
        groups: &[crate::domain::GroupProfile]
    ) -> DomainResult<()> {
        let key = CacheKeyGenerator::nearby_groups_key(lat, lng, radius, cursor, limit);
        let ttl = Duration::from_secs(self.strategy.geo_cache_ttl);
        self.cache.set(&key, &groups.to_vec(), Some(ttl)).await
    }
    
    async fn get_user_search(
        &self, 
        query: &str, 
        cursor: i64, 
        limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>> {
        let key = CacheKeyGenerator::user_search_key(query, cursor, limit);
        self.cache.get(&key).await
    }
    
    async fn set_user_search(
        &self, 
        query: &str, 
        cursor: i64, 
        limit: i32, 
        users: &[UserProfile]
    ) -> DomainResult<()> {
        let key = CacheKeyGenerator::user_search_key(query, cursor, limit);
        let ttl = Duration::from_secs(self.strategy.user_cache_ttl);
        self.cache.set(&key, &users.to_vec(), Some(ttl)).await
    }
    
    async fn clear_all(&self) -> DomainResult<()> {
        self.cache.clear_all().await?;
        debug!("All cache cleared");
        Ok(())
    }
    
    async fn get_cache_stats(&self) -> DomainResult<CacheStats> {
        let stats = self.cache.get_stats().await?;
        
        Ok(CacheStats {
            total_keys: stats.total_keys,
            memory_usage: format!("{} bytes", stats.memory_usage),
            hit_rate: if stats.hit_count + stats.miss_count > 0 { 
                stats.hit_count as f64 / (stats.hit_count + stats.miss_count) as f64 
            } else { 
                0.0 
            },
            miss_rate: if stats.hit_count + stats.miss_count > 0 { 
                stats.miss_count as f64 / (stats.hit_count + stats.miss_count) as f64 
            } else { 
                0.0 
            },
        })
    }
}

/// 空缓存服务（用于禁用缓存的场景）
pub struct NullCacheService;

#[async_trait]
impl CacheService for NullCacheService {
    async fn get_user(&self, _user_id: i64) -> DomainResult<Option<UserProfile>> {
        Ok(None)
    }
    
    async fn set_user(&self, _user: &UserProfile) -> DomainResult<()> {
        Ok(())
    }
    
    async fn invalidate_user(&self, _user_id: i64) -> DomainResult<()> {
        Ok(())
    }
    
    async fn get_group(&self, _group_id: &str) -> DomainResult<Option<crate::domain::GroupProfile>> {
        Ok(None)
    }
    
    async fn set_group(&self, _group: &crate::domain::GroupProfile) -> DomainResult<()> {
        Ok(())
    }
    
    async fn invalidate_group(&self, _group_id: &str) -> DomainResult<()> {
        Ok(())
    }
    
    async fn get_nearby_users(
        &self, 
        _lat: f64, 
        _lng: f64, 
        _radius: f64, 
        _cursor: i64, 
        _limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>> {
        Ok(None)
    }
    
    async fn set_nearby_users(
        &self, 
        _lat: f64, 
        _lng: f64, 
        _radius: f64, 
        _cursor: i64, 
        _limit: i32, 
        _users: &[crate::domain::UserProfile]
    ) -> DomainResult<()> {
        Ok(())
    }
    
    async fn get_nearby_groups(
        &self, 
        _lat: f64, 
        _lng: f64, 
        _radius: f64, 
        _cursor: i64, 
        _limit: i32
    ) -> DomainResult<Option<Vec<crate::domain::GroupProfile>>> {
        Ok(None)
    }
    
    async fn set_nearby_groups(
        &self, 
        _lat: f64, 
        _lng: f64, 
        _radius: f64, 
        _cursor: i64, 
        _limit: i32, 
        _groups: &[crate::domain::GroupProfile]
    ) -> DomainResult<()> {
        Ok(())
    }
    
    async fn get_user_search(
        &self, 
        _query: &str, 
        _cursor: i64, 
        _limit: i32
    ) -> DomainResult<Option<Vec<UserProfile>>> {
        Ok(None)
    }
    
    async fn set_user_search(
        &self, 
        _query: &str, 
        _cursor: i64, 
        _limit: i32, 
        _users: &[crate::domain::UserProfile]
    ) -> DomainResult<()> {
        Ok(())
    }
    
    async fn clear_all(&self) -> DomainResult<()> {
        Ok(())
    }
    
    async fn get_cache_stats(&self) -> DomainResult<CacheStats> {
        Ok(CacheStats {
            total_keys: 0,
            memory_usage: "Disabled".to_string(),
            hit_rate: 0.0,
            miss_rate: 0.0,
        })
    }
}

/// 缓存装饰器，为repository添加缓存功能
pub struct CachedRepository<T> {
    repository: T,
    cache_service: Arc<dyn CacheService>,
}

impl<T> CachedRepository<T> {
    pub fn new(repository: T, cache_service: Arc<dyn CacheService>) -> Self {
        Self {
            repository,
            cache_service,
        }
    }
    
    pub fn into_inner(self) -> T {
        self.repository
    }
    
    pub fn cache_service(&self) -> &Arc<dyn CacheService> {
        &self.cache_service
    }
}

impl<T: Clone> Clone for CachedRepository<T> {
    fn clone(&self) -> Self {
        Self {
            repository: self.repository.clone(),
            cache_service: self.cache_service.clone(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_cache_service_user_operations() {
        let strategy = CacheStrategy::default();
        let cache_service = InMemoryCacheService::new(strategy);
        
        // 测试用户缓存
        let user = User {
            user_id: 123,
            login_id: Some("test_user".to_string()),
            nickname: "Test User".to_string(),
            avatar_url: None,
            email: None,
            bio: None,
            is_temp: false,
            created_at: chrono::Utc::now(),
            last_active_at: chrono::Utc::now(),
            latitude: Some(39.916527),
            longitude: Some(116.397128),
            location_name: Some("Beijing".to_string()),
            distance: None,
        };
        
        // 设置缓存
        cache_service.set_user(&user).await.unwrap();
        
        // 获取缓存
        let cached_user = cache_service.get_user(123).await.unwrap();
        assert!(cached_user.is_some());
        assert_eq!(cached_user.unwrap().nickname, "Test User");
        
        // 删除缓存
        cache_service.invalidate_user(123).await.unwrap();
        let cached_user = cache_service.get_user(123).await.unwrap();
        assert!(cached_user.is_none());
    }
    
    #[tokio::test]
    async fn test_null_cache_service() {
        let cache_service = NullCacheService;
        
        // 所有操作都应该返回None或成功
        let result = cache_service.get_user(123).await.unwrap();
        assert!(result.is_none());
        
        let user = User {
            user_id: 123,
            login_id: Some("test".to_string()),
            nickname: "Test".to_string(),
            avatar_url: None,
            email: None,
            bio: None,
            is_temp: false,
            created_at: chrono::Utc::now(),
            last_active_at: chrono::Utc::now(),
            latitude: None,
            longitude: None,
            location_name: None,
            distance: None,
        };
        
        cache_service.set_user(&user).await.unwrap();
        cache_service.invalidate_user(123).await.unwrap();
        cache_service.clear_all().await.unwrap();
    }
    
    #[test]
    fn test_cache_key_generation() {
        assert_eq!(CacheKeyGenerator::user_key(123), "user:123");
        assert_eq!(CacheKeyGenerator::user_profile_key(456), "user:profile:456");
        assert_eq!(CacheKeyGenerator::group_key("group-uuid"), "group:group-uuid");
        
        let nearby_key = CacheKeyGenerator::nearby_users_key(39.916527, 116.397128, 1000.0, 0, 20);
        assert_eq!(nearby_key, "nearby:users:39.916527:116.397128:1000:0:20");
    }
    
    #[test]
    fn test_cache_strategy_default() {
        let strategy = CacheStrategy::default();
        assert_eq!(strategy.user_cache_ttl, 600);
        assert_eq!(strategy.group_cache_ttl, 300);
        assert_eq!(strategy.geo_cache_ttl, 180);
    }
}