/// 高级缓存策略
/// 
/// 提供多层缓存、智能失效、预热等高级缓存功能
use std::collections::HashMap;
use std::hash::Hash;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use tracing::{debug, info, warn};
use serde::{Serialize, Deserialize};

use crate::domain::{DomainResult, DomainError};

/// 缓存项元数据
#[derive(Debug, Clone)]
struct CacheItemMeta {
    created_at: Instant,
    last_accessed: Instant,
    access_count: u64,
    ttl: Option<Duration>,
    size_bytes: usize,
}

impl CacheItemMeta {
    fn new(ttl: Option<Duration>, size_bytes: usize) -> Self {
        let now = Instant::now();
        Self {
            created_at: now,
            last_accessed: now,
            access_count: 1,
            ttl,
            size_bytes,
        }
    }

    fn is_expired(&self) -> bool {
        if let Some(ttl) = self.ttl {
            self.created_at.elapsed() > ttl
        } else {
            false
        }
    }

    fn touch(&mut self) {
        self.last_accessed = Instant::now();
        self.access_count += 1;
    }
}

/// 缓存项
#[derive(Debug, Clone)]
struct CacheItem<V> {
    value: V,
    meta: CacheItemMeta,
}

/// 缓存策略配置
#[derive(Debug, Clone)]
pub struct CacheConfig {
    /// 最大缓存项数量
    pub max_items: usize,
    /// 最大内存使用（字节）
    pub max_memory_bytes: usize,
    /// 默认TTL
    pub default_ttl: Option<Duration>,
    /// 清理间隔
    pub cleanup_interval: Duration,
    /// LRU清理阈值
    pub lru_cleanup_threshold: f64,
}

impl Default for CacheConfig {
    fn default() -> Self {
        Self {
            max_items: 10000,
            max_memory_bytes: 100 * 1024 * 1024, // 100MB
            default_ttl: Some(Duration::from_secs(3600)), // 1小时
            cleanup_interval: Duration::from_secs(300), // 5分钟
            lru_cleanup_threshold: 0.8, // 80%
        }
    }
}

/// 高级内存缓存
pub struct AdvancedMemoryCache<K, V> {
    data: Arc<RwLock<HashMap<K, CacheItem<V>>>>,
    config: CacheConfig,
    stats: Arc<RwLock<CacheStats>>,
}

impl<K, V> AdvancedMemoryCache<K, V>
where
    K: Hash + Eq + Clone + Send + Sync + 'static,
    V: Clone + Send + Sync + 'static,
{
    /// 创建新的高级缓存
    pub fn new(config: CacheConfig) -> Self {
        let cache = Self {
            data: Arc::new(RwLock::new(HashMap::new())),
            config: config.clone(),
            stats: Arc::new(RwLock::new(CacheStats::default())),
        };

        // 启动后台清理任务
        cache.start_cleanup_task();
        
        cache
    }

    /// 获取缓存项
    pub async fn get(&self, key: &K) -> Option<V> {
        let mut data = self.data.write().await;
        let mut stats = self.stats.write().await;

        if let Some(item) = data.get_mut(key) {
            if item.meta.is_expired() {
                data.remove(key);
                stats.record_miss();
                None
            } else {
                item.meta.touch();
                stats.record_hit();
                Some(item.value.clone())
            }
        } else {
            stats.record_miss();
            None
        }
    }

    /// 设置缓存项
    pub async fn set(&self, key: K, value: V, ttl: Option<Duration>) -> DomainResult<()> {
        let size_bytes = self.estimate_size(&value);
        let ttl = ttl.or(self.config.default_ttl);
        
        let mut data = self.data.write().await;
        let mut stats = self.stats.write().await;

        // 检查是否需要清理空间
        if data.len() >= self.config.max_items {
            self.cleanup_lru(&mut data, &mut stats).await;
        }

        let item = CacheItem {
            value,
            meta: CacheItemMeta::new(ttl, size_bytes),
        };

        data.insert(key, item);
        stats.record_set();
        
        Ok(())
    }

    /// 删除缓存项
    pub async fn remove(&self, key: &K) -> Option<V> {
        let mut data = self.data.write().await;
        let mut stats = self.stats.write().await;
        
        if let Some(item) = data.remove(key) {
            stats.record_delete();
            Some(item.value)
        } else {
            None
        }
    }

    /// 清空缓存
    pub async fn clear(&self) {
        let mut data = self.data.write().await;
        let mut stats = self.stats.write().await;
        
        let count = data.len();
        data.clear();
        stats.record_clear(count);
    }

    /// 获取缓存统计信息
    pub async fn stats(&self) -> CacheStats {
        self.stats.read().await.clone()
    }

    /// 获取缓存大小
    pub async fn len(&self) -> usize {
        self.data.read().await.len()
    }

    /// 检查缓存是否为空
    pub async fn is_empty(&self) -> bool {
        self.data.read().await.is_empty()
    }

    /// 估算值的大小（简化实现）
    fn estimate_size(&self, _value: &V) -> usize {
        // 简化实现，实际应用中可能需要更精确的大小计算
        std::mem::size_of::<V>()
    }

    /// LRU清理
    async fn cleanup_lru(
        &self,
        data: &mut HashMap<K, CacheItem<V>>,
        stats: &mut CacheStats,
    ) {
        let cleanup_count = (data.len() as f64 * (1.0 - self.config.lru_cleanup_threshold)) as usize;
        
        // 收集访问时间最早的项目
        let mut items: Vec<_> = data.iter().collect();
        items.sort_by_key(|(_, item)| item.meta.last_accessed);
        
        let mut removed_count = 0;
        for (key, _) in items.iter().take(cleanup_count) {
            data.remove(*key);
            removed_count += 1;
        }
        
        stats.record_eviction(removed_count);
        
        if removed_count > 0 {
            debug!("LRU清理完成，移除了 {} 个缓存项", removed_count);
        }
    }

    /// 启动后台清理任务
    fn start_cleanup_task(&self) {
        let data = Arc::clone(&self.data);
        let stats = Arc::clone(&self.stats);
        let cleanup_interval = self.config.cleanup_interval;
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(cleanup_interval);
            
            loop {
                interval.tick().await;
                
                let mut data_guard = data.write().await;
                let mut stats_guard = stats.write().await;
                
                let initial_count = data_guard.len();
                
                // 清理过期项目
                data_guard.retain(|_, item| !item.meta.is_expired());
                
                let expired_count = initial_count - data_guard.len();
                if expired_count > 0 {
                    stats_guard.record_expiration(expired_count);
                    debug!("清理了 {} 个过期缓存项", expired_count);
                }
            }
        });
    }
}

/// 缓存统计信息
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct CacheStats {
    pub hits: u64,
    pub misses: u64,
    pub sets: u64,
    pub deletes: u64,
    pub evictions: u64,
    pub expirations: u64,
    pub clears: u64,
}

impl CacheStats {
    fn record_hit(&mut self) {
        self.hits += 1;
    }

    fn record_miss(&mut self) {
        self.misses += 1;
    }

    fn record_set(&mut self) {
        self.sets += 1;
    }

    fn record_delete(&mut self) {
        self.deletes += 1;
    }

    fn record_eviction(&mut self, count: usize) {
        self.evictions += count as u64;
    }

    fn record_expiration(&mut self, count: usize) {
        self.expirations += count as u64;
    }

    fn record_clear(&mut self, count: usize) {
        self.clears += 1;
        self.evictions += count as u64;
    }

    /// 计算命中率
    pub fn hit_rate(&self) -> f64 {
        let total = self.hits + self.misses;
        if total == 0 {
            0.0
        } else {
            self.hits as f64 / total as f64
        }
    }

    /// 获取总操作数
    pub fn total_operations(&self) -> u64 {
        self.hits + self.misses + self.sets + self.deletes
    }
}

/// 多层缓存管理器
pub struct MultiLayerCache<K, V> {
    l1_cache: AdvancedMemoryCache<K, V>, // L1: 内存缓存
    l2_cache: Option<Arc<dyn CacheLayer<K, V>>>, // L2: 可选的外部缓存
}

/// 缓存层trait
#[async_trait::async_trait]
pub trait CacheLayer<K, V>: Send + Sync {
    async fn get(&self, key: &K) -> DomainResult<Option<V>>;
    async fn set(&self, key: K, value: V, ttl: Option<Duration>) -> DomainResult<()>;
    async fn remove(&self, key: &K) -> DomainResult<Option<V>>;
    async fn clear(&self) -> DomainResult<()>;
}

impl<K, V> MultiLayerCache<K, V>
where
    K: Hash + Eq + Clone + Send + Sync + 'static,
    V: Clone + Send + Sync + 'static,
{
    /// 创建多层缓存
    pub fn new(config: CacheConfig) -> Self {
        Self {
            l1_cache: AdvancedMemoryCache::new(config),
            l2_cache: None,
        }
    }

    /// 设置L2缓存层
    pub fn with_l2_cache(mut self, l2_cache: Arc<dyn CacheLayer<K, V>>) -> Self {
        self.l2_cache = Some(l2_cache);
        self
    }

    /// 获取缓存项（多层查找）
    pub async fn get(&self, key: &K) -> DomainResult<Option<V>> {
        // 先查L1缓存
        if let Some(value) = self.l1_cache.get(key).await {
            return Ok(Some(value));
        }

        // 查L2缓存
        if let Some(ref l2) = self.l2_cache {
            if let Some(value) = l2.get(key).await? {
                // 回写到L1缓存
                let _ = self.l1_cache.set(key.clone(), value.clone(), None).await;
                return Ok(Some(value));
            }
        }

        Ok(None)
    }

    /// 设置缓存项（写入所有层）
    pub async fn set(&self, key: K, value: V, ttl: Option<Duration>) -> DomainResult<()> {
        // 写入L1缓存
        self.l1_cache.set(key.clone(), value.clone(), ttl).await?;

        // 写入L2缓存
        if let Some(ref l2) = self.l2_cache {
            l2.set(key, value, ttl).await?;
        }

        Ok(())
    }

    /// 删除缓存项（从所有层删除）
    pub async fn remove(&self, key: &K) -> DomainResult<Option<V>> {
        let l1_result = self.l1_cache.remove(key).await;

        if let Some(ref l2) = self.l2_cache {
            let _ = l2.remove(key).await;
        }

        Ok(l1_result)
    }

    /// 清空所有缓存层
    pub async fn clear(&self) -> DomainResult<()> {
        self.l1_cache.clear().await;

        if let Some(ref l2) = self.l2_cache {
            l2.clear().await?;
        }

        Ok(())
    }

    /// 获取L1缓存统计
    pub async fn l1_stats(&self) -> CacheStats {
        self.l1_cache.stats().await
    }
}

/// 缓存预热器
pub struct CacheWarmer<K, V> {
    cache: Arc<MultiLayerCache<K, V>>,
    warmup_fn: Arc<dyn Fn() -> Box<dyn std::future::Future<Output = DomainResult<Vec<(K, V)>>> + Send + Unpin> + Send + Sync>,
}

impl<K, V> CacheWarmer<K, V>
where
    K: Hash + Eq + Clone + Send + Sync + 'static,
    V: Clone + Send + Sync + 'static,
{
    /// 创建缓存预热器
    pub fn new<F, Fut>(
        cache: Arc<MultiLayerCache<K, V>>,
        warmup_fn: F,
    ) -> Self
    where
        F: Fn() -> Fut + Send + Sync + 'static,
        Fut: std::future::Future<Output = DomainResult<Vec<(K, V)>>> + Send + 'static,
    {
        let warmup_fn = Arc::new(move || -> Box<dyn std::future::Future<Output = DomainResult<Vec<(K, V)>>> + Send + Unpin> {
            Box::new(warmup_fn())
        });

        Self {
            cache,
            warmup_fn,
        }
    }

    /// 执行缓存预热
    pub async fn warmup(&self) -> DomainResult<usize> {
        info!("开始缓存预热...");
        let start_time = Instant::now();

        let items = (self.warmup_fn)().await?;
        let item_count = items.len();

        for (key, value) in items {
            self.cache.set(key, value, None).await?;
        }

        let duration = start_time.elapsed();
        info!(
            "缓存预热完成，加载了 {} 个项目，耗时: {:?}",
            item_count, duration
        );

        Ok(item_count)
    }

    /// 定期预热任务
    pub fn start_periodic_warmup(&self, interval: Duration) {
        let cache = Arc::clone(&self.cache);
        let warmup_fn = Arc::clone(&self.warmup_fn);

        tokio::spawn(async move {
            let mut interval_timer = tokio::time::interval(interval);

            loop {
                interval_timer.tick().await;

                match warmup_fn().await {
                    Ok(items) => {
                        for (key, value) in items {
                            let _ = cache.set(key, value, None).await;
                        }
                    }
                    Err(error) => {
                        warn!("定期缓存预热失败: {:?}", error);
                    }
                }
            }
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::time::sleep;

    #[tokio::test]
    async fn test_advanced_memory_cache() {
        let config = CacheConfig {
            max_items: 100,
            default_ttl: Some(Duration::from_millis(100)),
            ..Default::default()
        };

        let cache = AdvancedMemoryCache::new(config);

        // 测试基本操作
        cache.set("key1".to_string(), "value1".to_string(), None).await.unwrap();
        assert_eq!(cache.get(&"key1".to_string()).await, Some("value1".to_string()));

        // 测试TTL过期
        sleep(Duration::from_millis(150)).await;
        assert_eq!(cache.get(&"key1".to_string()).await, None);

        // 测试统计信息
        let stats = cache.stats().await;
        assert!(stats.hits > 0);
        assert!(stats.misses > 0);
        assert!(stats.sets > 0);
    }

    #[tokio::test]
    async fn test_multi_layer_cache() {
        let config = CacheConfig::default();
        let cache = MultiLayerCache::new(config);

        // 测试基本操作
        cache.set("key1".to_string(), "value1".to_string(), None).await.unwrap();
        
        let result = cache.get(&"key1".to_string()).await.unwrap();
        assert_eq!(result, Some("value1".to_string()));

        // 测试删除
        let removed = cache.remove(&"key1".to_string()).await.unwrap();
        assert_eq!(removed, Some("value1".to_string()));

        let result = cache.get(&"key1".to_string()).await.unwrap();
        assert_eq!(result, None);
    }
}
