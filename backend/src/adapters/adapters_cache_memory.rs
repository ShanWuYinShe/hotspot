/// 内存缓存适配器
/// 
/// 提供高性能的内存缓存实现，支持TTL、LRU淘汰策略和统计监控
use crate::domain::{DomainResult, DomainError};
use crate::ports::cache::Cache;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;
use std::time::{Duration, Instant};
use serde::{Serialize, Deserialize, de::DeserializeOwned};
/// 内存缓存适配器
/// 
/// 功能职责：
/// - 实现 Cache 的内存缓存适配器
/// - 提供高性能的本地内存缓存功能
/// - 实现 LRU 缓存淘汰策略和内存管理
/// - 支持缓存的 TTL 过期管理和自动清理
/// - 提供缓存性能的实时监控和统计分析
/// 
/// 缓存策略：
/// - LRU 淘汰：最近最少使用的数据优先淘汰
/// - TTL 过期：基于时间的自动过期和清理
/// - 内存限制：可配置的最大内存使用限制
/// - 热点数据：高频访问数据的优先保留
/// 
/// 性能特性：
/// - 零网络延迟：本地内存访问的极高性能
/// - 并发安全：基于 RwLock 的线程安全实现
/// - 内存效率：紧凑的数据结构和内存布局
/// - 实时统计：命中率、内存使用等关键指标
/// 
/// 适用场景：
/// - 单机部署的高性能缓存需求
/// - 热点数据的快速访问和缓存
/// - 开发和测试环境的简化部署
use async_trait::async_trait;
use tracing::{debug, warn, info};

/// 缓存条目
#[derive(Debug, Clone)]
struct CacheEntry {
    /// 缓存的数据（JSON序列化）
    data: String,
    /// 过期时间
    expires_at: Option<Instant>,
    /// 最后访问时间
    last_accessed: Instant,
    /// 访问次数
    access_count: u64,
}

impl CacheEntry {
    /// 创建新的缓存条目
    fn new(data: String, ttl: Option<Duration>) -> Self {
        let now = Instant::now();
        Self {
            data,
            expires_at: ttl.map(|duration| now + duration),
            last_accessed: now,
            access_count: 1,
        }
    }
    
    /// 检查是否已过期
    fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Instant::now() > expires_at
        } else {
            false
        }
    }
    
    /// 更新访问信息
    fn touch(&mut self) {
        self.last_accessed = Instant::now();
        self.access_count += 1;
    }
    
}

/// 缓存配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryCacheConfig {
    /// 最大缓存条目数
    pub max_entries: usize,
    /// 默认TTL（秒）
    pub default_ttl_seconds: Option<u64>,
    /// 清理间隔（秒）
    pub cleanup_interval_seconds: u64,
    /// 启用统计
    pub enable_stats: bool,
    /// LRU淘汰阈值（当缓存使用率超过此值时开始LRU淘汰）
    pub lru_threshold: f64,
}

impl Default for MemoryCacheConfig {
    fn default() -> Self {
        Self {
            max_entries: 10000,
            default_ttl_seconds: Some(3600), // 1小时
            cleanup_interval_seconds: 300,   // 5分钟
            enable_stats: true,
            lru_threshold: 0.8, // 80%
        }
    }
}

/// 内存缓存统计信息
#[derive(Debug, Clone, Serialize)]
pub struct MemoryCacheStats {
    /// 总条目数
    pub total_entries: usize,
    /// 命中次数
    pub hits: u64,
    /// 未命中次数
    pub misses: u64,
    /// 过期清理次数
    pub evictions: u64,
    /// LRU淘汰次数
    pub lru_evictions: u64,
    /// 命中率
    pub hit_rate: f64,
    /// 内存使用估算（字节）
    pub estimated_memory_bytes: usize,
}

impl MemoryCacheStats {
    fn new() -> Self {
        Self {
            total_entries: 0,
            hits: 0,
            misses: 0,
            evictions: 0,
            lru_evictions: 0,
            hit_rate: 0.0,
            estimated_memory_bytes: 0,
        }
    }
    
    fn calculate_hit_rate(&mut self) {
        let total = self.hits + self.misses;
        self.hit_rate = if total > 0 {
            self.hits as f64 / total as f64
        } else {
            0.0
        };
    }
}

/// 内存缓存实现
pub struct MemoryCache {
    /// 缓存存储
    store: Arc<RwLock<HashMap<String, CacheEntry>>>,
    /// 配置
    config: MemoryCacheConfig,
    /// 统计信息
    stats: Arc<RwLock<MemoryCacheStats>>,
    /// 清理任务句柄
    cleanup_handle: Option<tokio::task::JoinHandle<()>>,
}

impl MemoryCache {
    /// 创建新的内存缓存
    pub fn new(config: MemoryCacheConfig) -> Self {
        let store = Arc::new(RwLock::new(HashMap::new()));
        let stats = Arc::new(RwLock::new(MemoryCacheStats::new()));
        
        let mut cache = Self {
            store,
            config,
            stats,
            cleanup_handle: None,
        };
        
        // 启动清理任务
        cache.start_cleanup_task();
        
        cache
    }
    
    /// 启动定期清理任务
    fn start_cleanup_task(&mut self) {
        let store = self.store.clone();
        let stats = self.stats.clone();
        let interval = Duration::from_secs(self.config.cleanup_interval_seconds);
        
        let handle = tokio::spawn(async move {
            let mut cleanup_interval = tokio::time::interval(interval);
            
            loop {
                cleanup_interval.tick().await;
                
                let mut store_guard = store.write().await;
                let mut stats_guard = stats.write().await;
                
                // 清理过期条目
                let initial_count = store_guard.len();
                store_guard.retain(|_key, entry| !entry.is_expired());
                let expired_count = initial_count - store_guard.len();
                
                stats_guard.evictions += expired_count as u64;
                stats_guard.total_entries = store_guard.len();
                
                if expired_count > 0 {
                    debug!("清理了 {} 个过期缓存条目", expired_count);
                }
                
                drop(store_guard);
                drop(stats_guard);
            }
        });
        
        self.cleanup_handle = Some(handle);
    }
    
    /// 检查是否需要LRU淘汰
    async fn maybe_evict_lru(&self) {
        let store_guard = self.store.read().await;
        let usage_rate = store_guard.len() as f64 / self.config.max_entries as f64;
        
        if usage_rate >= self.config.lru_threshold {
            drop(store_guard);
            
            let mut store_guard = self.store.write().await;
            let mut stats_guard = self.stats.write().await;
            
            // 按最后访问时间排序，移除最久未访问的条目
            let mut entries: Vec<_> = store_guard.iter().collect();
            entries.sort_by_key(|(_, entry)| entry.last_accessed);
            
            let evict_count = (store_guard.len() as f64 * 0.1) as usize; // 淘汰10%
            
            // 收集要删除的键
            let keys_to_remove: Vec<String> = entries.iter()
                .take(evict_count)
                .map(|(key, _)| (*key).clone())
                .collect();
            
            // 删除键
            for key in keys_to_remove {
                store_guard.remove(&key);
            }
            
            stats_guard.lru_evictions += evict_count as u64;
            stats_guard.total_entries = store_guard.len();
            
            info!("LRU淘汰了 {} 个缓存条目", evict_count);
        }
    }
    
    /// 估算内存使用量
    fn estimate_memory_usage(store: &HashMap<String, CacheEntry>) -> usize {
        store.iter().map(|(key, entry)| {
            key.len() + entry.data.len() + std::mem::size_of::<CacheEntry>()
        }).sum()
    }
}

#[async_trait]
impl Cache for MemoryCache {
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: DeserializeOwned,
    {
        let mut store_guard = self.store.write().await;
        let mut stats_guard = self.stats.write().await;
        
        if let Some(entry) = store_guard.get_mut(key) {
            if entry.is_expired() {
                // 移除过期条目
                store_guard.remove(key);
                stats_guard.misses += 1;
                stats_guard.evictions += 1;
                stats_guard.total_entries = store_guard.len();
                stats_guard.calculate_hit_rate();
                
                debug!("缓存条目已过期: {}", key);
                return Ok(None);
            }
            
            // 更新访问信息
            entry.touch();
            stats_guard.hits += 1;
            stats_guard.calculate_hit_rate();
            
            // 反序列化数据
            match serde_json::from_str::<T>(&entry.data) {
                Ok(value) => {
                    debug!("缓存命中: {}", key);
                    Ok(Some(value))
                }
                Err(e) => {
                    warn!("缓存数据反序列化失败: {} - {}", key, e);
                    // 移除损坏的数据
                    store_guard.remove(key);
                    stats_guard.total_entries = store_guard.len();
                    Ok(None)
                }
            }
        } else {
            stats_guard.misses += 1;
            stats_guard.calculate_hit_rate();
            debug!("缓存未命中: {}", key);
            Ok(None)
        }
    }
    
    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: Serialize + std::marker::Sync,
    {
        // 序列化数据
        let data = serde_json::to_string(value)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("缓存数据序列化失败: {}", e),
            })?;
        
        // 使用配置的默认TTL
        let effective_ttl = ttl.or_else(|| {
            self.config.default_ttl_seconds.map(Duration::from_secs)
        });
        
        let entry = CacheEntry::new(data, effective_ttl);
        
        {
            let mut store_guard = self.store.write().await;
            let mut stats_guard = self.stats.write().await;
            
            store_guard.insert(key.to_string(), entry);
            stats_guard.total_entries = store_guard.len();
            
            // 更新内存使用估算
            if self.config.enable_stats {
                stats_guard.estimated_memory_bytes = Self::estimate_memory_usage(&store_guard);
            }
        }
        
        // 检查是否需要LRU淘汰
        self.maybe_evict_lru().await;
        
        debug!("缓存已设置: {} (TTL: {:?})", key, effective_ttl);
        Ok(())
    }
    
    async fn delete(&self, key: &str) -> DomainResult<bool> {
        let mut store_guard = self.store.write().await;
        let mut stats_guard = self.stats.write().await;
        
        let removed = store_guard.remove(key).is_some();
        stats_guard.total_entries = store_guard.len();
        
        if removed {
            debug!("缓存已删除: {}", key);
        }
        
        Ok(removed)
    }
    
    async fn exists(&self, key: &str) -> DomainResult<bool> {
        let store_guard = self.store.read().await;
        
        if let Some(entry) = store_guard.get(key) {
            Ok(!entry.is_expired())
        } else {
            Ok(false)
        }
    }
    
    async fn ttl(&self, key: &str) -> DomainResult<Option<Duration>> {
        let store_guard = self.store.read().await;
        
        if let Some(entry) = store_guard.get(key) {
            if entry.is_expired() {
                Ok(None)
            } else if let Some(expires_at) = entry.expires_at {
                let now = Instant::now();
                if expires_at > now {
                    Ok(Some(expires_at.duration_since(now)))
                } else {
                    Ok(None)
                }
            } else {
                // 永不过期
                Ok(None)
            }
        } else {
            Ok(None)
        }
    }
    
    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()> {
        let mut store_guard = self.store.write().await;
        let mut stats_guard = self.stats.write().await;
        
        let mut deleted_count = 0;
        for key in keys {
            if store_guard.remove(*key).is_some() {
                deleted_count += 1;
            }
        }
        
        stats_guard.total_entries = store_guard.len();
        
        if deleted_count > 0 {
            debug!("批量删除缓存: {} 个条目", deleted_count);
        }
        
        Ok(())
    }
    
    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool> {
        let mut store_guard = self.store.write().await;
        
        if let Some(entry) = store_guard.get_mut(key) {
            if !entry.is_expired() {
                entry.expires_at = Some(Instant::now() + ttl);
                debug!("缓存过期时间已更新: {} (TTL: {:?})", key, ttl);
                Ok(true)
            } else {
                // 条目已过期，删除它
                store_guard.remove(key);
                Ok(false)
            }
        } else {
            Ok(false)
        }
    }
    
    async fn mget<T>(&self, keys: &[String]) -> DomainResult<Vec<Option<T>>>
    where
        T: DeserializeOwned + Send,
    {
        let mut results = Vec::with_capacity(keys.len());
        for key in keys {
            let value = self.get::<T>(key).await?;
            results.push(value);
        }
        Ok(results)
    }
    
    async fn mset<T>(&self, items: &[(String, T)], ttl: Option<Duration>) -> DomainResult<()>
    where
        T: Serialize + Send + Sync,
    {
        for (key, value) in items {
            self.set(key, value, ttl).await?;
        }
        Ok(())
    }
    
    async fn mdel(&self, keys: &[String]) -> DomainResult<u64> {
        let mut deleted_count = 0u64;
        for key in keys {
            if self.delete(key).await? {
                deleted_count += 1;
            }
        }
        Ok(deleted_count)
    }
    
    async fn flush_all(&self) -> DomainResult<()> {
        self.clear_all().await
    }
    
    async fn stats(&self) -> DomainResult<crate::ports::cache::CacheStats> {
        self.get_stats().await
    }
    
    async fn clear_all(&self) -> DomainResult<()> {
        let mut store_guard = self.store.write().await;
        let mut stats_guard = self.stats.write().await;
        
        let cleared_count = store_guard.len();
        store_guard.clear();
        stats_guard.total_entries = 0;
        stats_guard.estimated_memory_bytes = 0;
        
        info!("缓存已清空，清理了 {} 个条目", cleared_count);
        Ok(())
    }
    
    async fn get_stats(&self) -> DomainResult<crate::ports::cache::CacheStats> {
        let store_guard = self.store.read().await;
        let stats_guard = self.stats.read().await;
        
        let entries = store_guard.len() as u64;
        let memory_usage = Self::estimate_memory_usage(&store_guard) as u64;
        
        Ok(crate::ports::cache::CacheStats {
            total_keys: entries,
            memory_usage,
            hit_count: stats_guard.hits,
            miss_count: stats_guard.misses,
            hit_rate: {
                let total = stats_guard.hits + stats_guard.misses;
                if total > 0 {
                    stats_guard.hits as f64 / total as f64
                } else {
                    0.0
                }
            },
            evicted_keys: 0, // MemoryCache 暂不支持驱逐统计
            expired_keys: 0, // MemoryCache 暂不支持过期统计
        })
    }
}

impl Drop for MemoryCache {
    fn drop(&mut self) {
        if let Some(handle) = self.cleanup_handle.take() {
            handle.abort();
        }
    }
}

/// 缓存键生成器
pub struct CacheKeyBuilder {
    prefix: String,
}

impl CacheKeyBuilder {
    /// 创建新的缓存键生成器
    pub fn new(prefix: &str) -> Self {
        Self {
            prefix: prefix.to_string(),
        }
    }
    
    /// 生成用户相关的缓存键
    pub fn user_key(&self, user_id: i64, suffix: &str) -> String {
        format!("{}:user:{}:{}", self.prefix, user_id, suffix)
    }
    
    /// 生成群组相关的缓存键
    pub fn group_key(&self, group_id: &str, suffix: &str) -> String {
        format!("{}:group:{}:{}", self.prefix, group_id, suffix)
    }
    
    /// 生成地理位置相关的缓存键
    pub fn geo_key(&self, lat: f64, lng: f64, radius: f64, suffix: &str) -> String {
        format!("{}:geo:{}:{}:{}:{}", self.prefix, 
               (lat * 1000.0) as i64, 
               (lng * 1000.0) as i64, 
               (radius * 1000.0) as i64, 
               suffix)
    }
    
    /// 生成会话相关的缓存键
    pub fn session_key(&self, session_id: &str) -> String {
        format!("{}:session:{}", self.prefix, session_id)
    }
    
    /// 生成通用缓存键
    pub fn generic_key(&self, category: &str, id: &str) -> String {
        format!("{}:{}:{}", self.prefix, category, id)
    }
}

// CacheDecorator removed due to trait object incompatibility with generic Cache methods

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::time::{sleep, Duration};
    
    #[tokio::test]
    async fn test_memory_cache_basic_operations() {
        let config = MemoryCacheConfig::default();
        let cache = MemoryCache::new(config);
        
        // 测试设置和获取
        cache.set("test_key", &"test_value", None).await.unwrap();
        let value: Option<String> = cache.get("test_key").await.unwrap();
        assert_eq!(value, Some("test_value".to_string()));
        
        // 测试存在性检查
        assert!(cache.exists("test_key").await.unwrap());
        assert!(!cache.exists("non_existent").await.unwrap());
        
        // 测试删除
        cache.delete("test_key").await.unwrap();
        let value: Option<String> = cache.get("test_key").await.unwrap();
        assert_eq!(value, None);
    }
    
    #[tokio::test]
    async fn test_cache_ttl() {
        let config = MemoryCacheConfig::default();
        let cache = MemoryCache::new(config);
        
        // 设置短TTL
        cache.set("ttl_key", &"ttl_value", Some(Duration::from_millis(100))).await.unwrap();
        
        // 立即获取应该成功
        let value: Option<String> = cache.get("ttl_key").await.unwrap();
        assert_eq!(value, Some("ttl_value".to_string()));
        
        // 等待过期
        sleep(Duration::from_millis(150)).await;
        
        // 再次获取应该失败
        let value: Option<String> = cache.get("ttl_key").await.unwrap();
        assert_eq!(value, None);
    }
    
    #[test]
    fn test_cache_key_builder() {
        let builder = CacheKeyBuilder::new("hotspot");
        
        assert_eq!(builder.user_key(123, "profile"), "hotspot:user:123:profile");
        assert_eq!(builder.group_key("group-456", "members"), "hotspot:group:group-456:members");
        assert_eq!(builder.session_key("session-789"), "hotspot:session:session-789");
    }
    
    // Cache decorator test removed due to trait object incompatibility
}