/// 缓存接口定义 - 高性能数据访问抽象
/// 
/// 缓存特征：
/// - 键值存储
/// - 过期时间管理
/// - 批量操作
/// - 统计信息

use async_trait::async_trait;
use std::time::Duration;
use crate::domain::DomainResult;

// === 缓存接口 ===

/// 缓存接口
#[async_trait]
pub trait Cache: Send + Sync {
    /// 获取缓存值
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send;
    
    /// 设置缓存值
    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync;
    
    /// 删除缓存值
    async fn delete(&self, key: &str) -> DomainResult<bool>;
    
    /// 检查键是否存在
    async fn exists(&self, key: &str) -> DomainResult<bool>;
    
    /// 设置过期时间
    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool>;
    
    /// 获取剩余过期时间
    async fn ttl(&self, key: &str) -> DomainResult<Option<Duration>>;
    
    /// 批量获取
    async fn mget<T>(&self, keys: &[String]) -> DomainResult<Vec<Option<T>>>
    where
        T: serde::de::DeserializeOwned + Send;
    
    /// 批量设置
    async fn mset<T>(&self, items: &[(String, T)], ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync;
    
    /// 批量删除
    async fn mdel(&self, keys: &[String]) -> DomainResult<u64>;
    
    /// 清空所有缓存
    async fn flush_all(&self) -> DomainResult<()>;
    
    /// 获取缓存统计信息
    async fn stats(&self) -> DomainResult<CacheStats>;
    
    /// 批量删除（别名方法）
    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()>;
    
    /// 清空所有缓存（别名方法）
    async fn clear_all(&self) -> DomainResult<()>;
    
    /// 获取统计信息（别名方法）
    async fn get_stats(&self) -> DomainResult<CacheStats>;
}

/// 缓存统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct CacheStats {
    pub total_keys: u64,
    pub memory_usage: u64, // 字节
    pub hit_count: u64,
    pub miss_count: u64,
    pub hit_rate: f64,
    pub evicted_keys: u64,
    pub expired_keys: u64,
}

impl CacheStats {
    /// 计算命中率
    pub fn calculate_hit_rate(&mut self) {
        let total = self.hit_count + self.miss_count;
        self.hit_rate = if total > 0 {
            self.hit_count as f64 / total as f64
        } else {
            0.0
        };
    }
}

// === 分布式缓存接口 ===

/// 分布式缓存接口
#[async_trait]
pub trait DistributedCache: Cache {
    /// 分布式锁
    async fn lock(&self, key: &str, ttl: Duration) -> DomainResult<Option<LockGuard>>;
    
    /// 释放锁
    async fn unlock(&self, guard: LockGuard) -> DomainResult<bool>;
    
    /// 原子递增
    async fn incr(&self, key: &str, delta: i64) -> DomainResult<i64>;
    
    /// 原子递减
    async fn decr(&self, key: &str, delta: i64) -> DomainResult<i64>;
    
    /// 发布消息
    async fn publish(&self, channel: &str, message: &str) -> DomainResult<u64>;
    
    /// 订阅频道
    async fn subscribe(&self, channels: &[String]) -> DomainResult<SubscriptionHandle>;
    
    /// 取消订阅
    async fn unsubscribe(&self, handle: SubscriptionHandle) -> DomainResult<()>;
}

/// 锁守卫
#[derive(Debug, Clone)]
pub struct LockGuard {
    pub key: String,
    pub token: String,
    pub acquired_at: chrono::DateTime<chrono::Utc>,
    pub expires_at: chrono::DateTime<chrono::Utc>,
}

/// 订阅句柄
#[derive(Debug, Clone)]
pub struct SubscriptionHandle {
    pub subscription_id: String,
    pub channels: Vec<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

// === 缓存层接口 ===

/// 多层缓存接口
#[async_trait]
pub trait LayeredCache: Send + Sync + Clone {
    /// 从L1缓存获取
    async fn get_l1<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send;
    
    /// 从L2缓存获取
    async fn get_l2<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send;
    
    /// 设置到所有层
    async fn set_all_layers<T>(&self, key: &str, value: &T, l1_ttl: Duration, l2_ttl: Duration) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync;
    
    /// 从所有层删除
    async fn delete_all_layers(&self, key: &str) -> DomainResult<()>;
    
    /// 预热缓存
    async fn warm_up<T>(&self, key: &str, loader: Box<dyn CacheLoader<T>>) -> DomainResult<T>
    where
        T: serde::Serialize + serde::de::DeserializeOwned + Send + Sync;
    
    /// 获取层级统计
    async fn layer_stats(&self) -> DomainResult<LayeredCacheStats>;
}

/// 缓存加载器
#[async_trait]
pub trait CacheLoader<T>: Send + Sync {
    async fn load(&self, key: &str) -> DomainResult<T>;
}

/// 多层缓存统计
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct LayeredCacheStats {
    pub l1_stats: CacheStats,
    pub l2_stats: CacheStats,
    pub l1_hit_rate: f64,
    pub l2_hit_rate: f64,
    pub overall_hit_rate: f64,
}

// === 缓存策略接口 ===

/// 缓存策略接口
pub trait CacheStrategy: Send + Sync + Clone {
    /// 生成缓存键
    fn generate_key(&self, prefix: &str, params: &[&str]) -> String;
    
    /// 获取默认TTL
    fn default_ttl(&self) -> Duration;
    
    /// 获取键的TTL
    fn key_ttl(&self, key: &str) -> Duration;
    
    /// 检查是否应该缓存
    fn should_cache(&self, key: &str, value_size: usize) -> bool;
    
    /// 获取缓存优先级
    fn cache_priority(&self, key: &str) -> CachePriority;
}

/// 缓存优先级
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum CachePriority {
    Low = 1,
    Normal = 2,
    High = 3,
    Critical = 4,
}

// === 缓存装饰器接口 ===

/// 缓存装饰器 - 为方法调用添加缓存
#[async_trait]
pub trait CacheDecorator<T>: Send + Sync + Clone {
    /// 缓存方法调用结果
    async fn cached_call<F, R>(&self, key: &str, ttl: Duration, func: F) -> DomainResult<R>
    where
        F: FnOnce() -> std::pin::Pin<Box<dyn std::future::Future<Output = DomainResult<R>> + Send>> + Send,
        R: serde::Serialize + serde::de::DeserializeOwned + Send + Sync;
    
    /// 缓存方法调用结果（带条件）
    async fn cached_call_if<F, R, P>(&self, key: &str, ttl: Duration, predicate: P, func: F) -> DomainResult<R>
    where
        F: FnOnce() -> std::pin::Pin<Box<dyn std::future::Future<Output = DomainResult<R>> + Send>> + Send,
        P: FnOnce(&R) -> bool + Send,
        R: serde::Serialize + serde::de::DeserializeOwned + Send + Sync;
    
    /// 刷新缓存
    async fn refresh_cache<F, R>(&self, key: &str, ttl: Duration, func: F) -> DomainResult<R>
    where
        F: FnOnce() -> std::pin::Pin<Box<dyn std::future::Future<Output = DomainResult<R>> + Send>> + Send,
        R: serde::Serialize + serde::de::DeserializeOwned + Send + Sync;
}

// === 缓存工厂接口 ===

/// 缓存工厂 - 创建和管理缓存实例
pub trait CacheFactory: Send + Sync + Clone {
    type Cache: Cache;
    type DistributedCache: DistributedCache;
    type LayeredCache: LayeredCache;
    type Strategy: CacheStrategy;
    
    /// 创建基础缓存
    fn cache(&self) -> Self::Cache;
    
    /// 创建分布式缓存
    fn distributed_cache(&self) -> Self::DistributedCache;
    
    /// 创建多层缓存
    fn layered_cache(&self) -> Self::LayeredCache;
    
    /// 创建缓存策略
    fn cache_strategy(&self) -> Self::Strategy;
    
    /// 创建缓存装饰器
    fn cache_decorator<T>(&self) -> Box<dyn CacheDecorator<T>>
    where
        T: Send + Sync + 'static;
}

// === 缓存键生成器 ===

/// 缓存键生成器
pub struct CacheKeyBuilder {
    prefix: String,
    parts: Vec<String>,
}

impl CacheKeyBuilder {
    /// 创建新的键生成器
    pub fn new(prefix: &str) -> Self {
        Self {
            prefix: prefix.to_string(),
            parts: Vec::new(),
        }
    }
    
    /// 添加键部分
    pub fn part(mut self, part: &str) -> Self {
        self.parts.push(part.to_string());
        self
    }
    
    /// 添加用户ID部分
    pub fn user_id(mut self, user_id: crate::domain::UserId) -> Self {
        self.parts.push(user_id.value().to_string());
        self
    }
    
    /// 添加群组ID部分
    pub fn group_id(mut self, group_id: crate::domain::GroupId) -> Self {
        self.parts.push(group_id.value().to_string());
        self
    }
    
    /// 构建最终键
    pub fn build(self) -> String {
        if self.parts.is_empty() {
            self.prefix
        } else {
            format!("{}:{}", self.prefix, self.parts.join(":"))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cache_stats() {
        let mut stats = CacheStats {
            total_keys: 100,
            memory_usage: 1024,
            hit_count: 80,
            miss_count: 20,
            hit_rate: 0.0,
            evicted_keys: 5,
            expired_keys: 10,
        };
        
        stats.calculate_hit_rate();
        assert_eq!(stats.hit_rate, 0.8);
    }
    
    #[test]
    fn test_cache_key_builder() {
        let key = CacheKeyBuilder::new("user")
            .part("profile")
            .user_id(crate::domain::UserId::new(123).unwrap())
            .build();
        
        assert_eq!(key, "user:profile:123");
    }
    
    #[test]
    fn test_cache_priority_ordering() {
        assert!(CachePriority::Critical > CachePriority::High);
        assert!(CachePriority::High > CachePriority::Normal);
        assert!(CachePriority::Normal > CachePriority::Low);
    }
}

// === 类型别名（向后兼容） ===

/// 缓存端口（向后兼容别名）
pub type CachePort = dyn Cache;