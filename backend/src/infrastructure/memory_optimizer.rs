/// 内存优化系统
/// 
/// 功能职责：
/// - 监控内存使用情况
/// - 自动内存清理和优化
/// - 缓存管理和淘汰策略
/// - 连接池优化
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug, error};

use crate::domain::DomainResult;

/// 内存优化器
#[derive(Clone)]
pub struct MemoryOptimizer {
    memory_stats: Arc<RwLock<MemoryStats>>,
    cache_managers: Arc<RwLock<HashMap<String, CacheManager>>>,
    config: MemoryOptimizerConfig,
}

/// 内存统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryStats {
    /// 总内存使用量（MB）
    pub total_memory_mb: f64,
    /// 堆内存使用量（MB）
    pub heap_memory_mb: f64,
    /// 缓存内存使用量（MB）
    pub cache_memory_mb: f64,
    /// 连接池内存使用量（MB）
    pub connection_pool_memory_mb: f64,
    /// WebSocket内存使用量（MB）
    pub websocket_memory_mb: f64,
    /// 内存使用率（%）
    pub memory_usage_percent: f64,
    /// 垃圾回收统计
    pub gc_stats: GcStats,
    /// 最后更新时间
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

/// 垃圾回收统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GcStats {
    pub total_collections: u64,
    pub total_gc_time_ms: u64,
    pub average_gc_time_ms: f64,
    pub last_gc_time: Option<chrono::DateTime<chrono::Utc>>,
}

/// 缓存管理器
#[derive(Clone)]
pub struct CacheManager {
    cache_name: String,
    entries: Arc<RwLock<HashMap<String, CacheEntry>>>,
    stats: Arc<RwLock<CacheStats>>,
    config: CacheConfig,
}

/// 缓存条目
#[derive(Debug, Clone)]
pub struct CacheEntry {
    pub key: String,
    pub data: Vec<u8>,
    pub created_at: Instant,
    pub last_accessed: Instant,
    pub access_count: u64,
    pub ttl: Option<Duration>,
}

/// 缓存统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheStats {
    pub total_entries: usize,
    pub total_memory_mb: f64,
    pub hit_count: u64,
    pub miss_count: u64,
    pub hit_rate: f64,
    pub eviction_count: u64,
    pub last_cleanup: chrono::DateTime<chrono::Utc>,
}

/// 缓存配置
#[derive(Debug, Clone)]
pub struct CacheConfig {
    pub max_entries: usize,
    pub max_memory_mb: f64,
    pub default_ttl: Duration,
    pub cleanup_interval: Duration,
    pub eviction_policy: EvictionPolicy,
}

/// 淘汰策略
#[derive(Debug, Clone)]
pub enum EvictionPolicy {
    LRU,  // 最近最少使用
    LFU,  // 最少使用频率
    TTL,  // 基于过期时间
    FIFO, // 先进先出
}

/// 内存优化器配置
#[derive(Debug, Clone)]
pub struct MemoryOptimizerConfig {
    pub memory_threshold_mb: f64,
    pub cleanup_interval: Duration,
    pub enable_auto_cleanup: bool,
    pub gc_threshold_percent: f64,
    pub cache_cleanup_threshold: f64,
}

impl Default for MemoryOptimizerConfig {
    fn default() -> Self {
        Self {
            memory_threshold_mb: 1024.0, // 1GB
            cleanup_interval: Duration::from_secs(300), // 5分钟
            enable_auto_cleanup: true,
            gc_threshold_percent: 80.0,
            cache_cleanup_threshold: 0.8, // 80%
        }
    }
}

impl Default for CacheConfig {
    fn default() -> Self {
        Self {
            max_entries: 10000,
            max_memory_mb: 100.0,
            default_ttl: Duration::from_secs(3600), // 1小时
            cleanup_interval: Duration::from_secs(300), // 5分钟
            eviction_policy: EvictionPolicy::LRU,
        }
    }
}

impl MemoryOptimizer {
    /// 创建新的内存优化器
    pub fn new(config: MemoryOptimizerConfig) -> Self {
        Self {
            memory_stats: Arc::new(RwLock::new(MemoryStats::default())),
            cache_managers: Arc::new(RwLock::new(HashMap::new())),
            config,
        }
    }

    /// 启动内存优化器
    pub async fn start(&self) {
        let optimizer = self.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(optimizer.config.cleanup_interval);
            
            loop {
                interval.tick().await;
                if let Err(e) = optimizer.perform_memory_optimization().await {
                    error!("内存优化失败: {:?}", e);
                }
            }
        });

        info!("内存优化器已启动，清理间隔: {:?}", self.config.cleanup_interval);
    }

    /// 执行内存优化
    async fn perform_memory_optimization(&self) -> DomainResult<()> {
        // 更新内存统计
        self.update_memory_stats().await?;
        
        let stats = self.memory_stats.read().await;
        
        // 检查是否需要清理
        if stats.memory_usage_percent > self.config.gc_threshold_percent {
            warn!("内存使用率 {:.1}% 超过阈值 {:.1}%，开始清理", 
                stats.memory_usage_percent, self.config.gc_threshold_percent);
            
            drop(stats); // 释放读锁
            
            // 执行缓存清理
            self.cleanup_caches().await?;
            
            // 执行垃圾回收（模拟）
            self.trigger_garbage_collection().await?;
            
            info!("内存优化完成");
        }
        
        Ok(())
    }

    /// 更新内存统计
    async fn update_memory_stats(&self) -> DomainResult<()> {
        let mut stats = self.memory_stats.write().await;
        
        // 获取系统内存信息（简化实现）
        let total_memory = self.get_system_memory_usage().await;
        let cache_memory = self.calculate_cache_memory_usage().await;
        
        stats.total_memory_mb = total_memory;
        stats.cache_memory_mb = cache_memory;
        stats.heap_memory_mb = total_memory * 0.7; // 估算
        stats.connection_pool_memory_mb = 50.0; // 估算
        stats.websocket_memory_mb = 30.0; // 估算
        stats.memory_usage_percent = (total_memory / self.config.memory_threshold_mb) * 100.0;
        stats.last_updated = chrono::Utc::now();
        
        debug!("内存统计已更新: 总内存 {:.1}MB, 使用率 {:.1}%", 
            stats.total_memory_mb, stats.memory_usage_percent);
        
        Ok(())
    }

    /// 获取系统内存使用量（简化实现）
    async fn get_system_memory_usage(&self) -> f64 {
        // 在实际实现中，这里应该调用系统API获取真实的内存使用情况
        // 这里使用模拟数据
        use rand::Rng;
        let mut rng = rand::rng();
        rng.random_range(200.0..800.0)
    }

    /// 计算缓存内存使用量
    async fn calculate_cache_memory_usage(&self) -> f64 {
        let cache_managers = self.cache_managers.read().await;
        let mut total_memory = 0.0;
        
        for manager in cache_managers.values() {
            let stats = manager.stats.read().await;
            total_memory += stats.total_memory_mb;
        }
        
        total_memory
    }

    /// 清理缓存
    async fn cleanup_caches(&self) -> DomainResult<()> {
        let cache_managers = self.cache_managers.read().await;
        let mut cleanup_count = 0;
        
        for manager in cache_managers.values() {
            let cleaned = manager.cleanup_expired_entries().await;
            cleanup_count += cleaned;
        }
        
        if cleanup_count > 0 {
            info!("清理了 {} 个过期缓存条目", cleanup_count);
        }
        
        Ok(())
    }

    /// 触发垃圾回收（模拟）
    async fn trigger_garbage_collection(&self) -> DomainResult<()> {
        let start_time = Instant::now();
        
        // 模拟垃圾回收过程
        tokio::time::sleep(Duration::from_millis(50)).await;
        
        let gc_time = start_time.elapsed();
        
        // 更新GC统计
        {
            let mut stats = self.memory_stats.write().await;
            stats.gc_stats.total_collections += 1;
            stats.gc_stats.total_gc_time_ms += gc_time.as_millis() as u64;
            stats.gc_stats.average_gc_time_ms = 
                stats.gc_stats.total_gc_time_ms as f64 / stats.gc_stats.total_collections as f64;
            stats.gc_stats.last_gc_time = Some(chrono::Utc::now());
        }
        
        info!("垃圾回收完成，耗时: {:?}", gc_time);
        Ok(())
    }

    /// 创建缓存管理器
    pub async fn create_cache_manager(&self, name: String, config: CacheConfig) -> CacheManager {
        let manager = CacheManager::new(name.clone(), config);
        
        {
            let mut cache_managers = self.cache_managers.write().await;
            cache_managers.insert(name.clone(), manager.clone());
        }
        
        info!("创建缓存管理器: {}", name);
        manager
    }

    /// 获取内存统计
    pub async fn get_memory_stats(&self) -> MemoryStats {
        self.memory_stats.read().await.clone()
    }

    /// 获取内存优化建议
    pub async fn get_optimization_suggestions(&self) -> Vec<MemoryOptimizationSuggestion> {
        let mut suggestions = Vec::new();
        let stats = self.memory_stats.read().await;
        
        // 检查总内存使用
        if stats.memory_usage_percent > 90.0 {
            suggestions.push(MemoryOptimizationSuggestion {
                category: OptimizationCategory::Memory,
                priority: OptimizationPriority::Critical,
                description: format!("内存使用率 {:.1}% 过高", stats.memory_usage_percent),
                recommended_action: "立即执行内存清理或增加系统内存".to_string(),
                estimated_savings_mb: stats.total_memory_mb * 0.2,
            });
        } else if stats.memory_usage_percent > 80.0 {
            suggestions.push(MemoryOptimizationSuggestion {
                category: OptimizationCategory::Memory,
                priority: OptimizationPriority::High,
                description: format!("内存使用率 {:.1}% 较高", stats.memory_usage_percent),
                recommended_action: "考虑优化缓存策略或清理不必要的数据".to_string(),
                estimated_savings_mb: stats.total_memory_mb * 0.1,
            });
        }
        
        // 检查缓存内存使用
        if stats.cache_memory_mb > 200.0 {
            suggestions.push(MemoryOptimizationSuggestion {
                category: OptimizationCategory::Cache,
                priority: OptimizationPriority::Medium,
                description: format!("缓存内存使用 {:.1}MB 较高", stats.cache_memory_mb),
                recommended_action: "调整缓存大小限制或清理策略".to_string(),
                estimated_savings_mb: stats.cache_memory_mb * 0.3,
            });
        }
        
        // 检查GC频率
        if stats.gc_stats.average_gc_time_ms > 100.0 {
            suggestions.push(MemoryOptimizationSuggestion {
                category: OptimizationCategory::GarbageCollection,
                priority: OptimizationPriority::Medium,
                description: format!("平均GC时间 {:.1}ms 较长", stats.gc_stats.average_gc_time_ms),
                recommended_action: "优化对象生命周期管理，减少GC压力".to_string(),
                estimated_savings_mb: 0.0,
            });
        }
        
        suggestions
    }

    /// 强制执行内存清理
    pub async fn force_cleanup(&self) -> DomainResult<MemoryCleanupResult> {
        let before_stats = self.get_memory_stats().await;
        
        // 执行所有清理操作
        self.cleanup_caches().await?;
        self.trigger_garbage_collection().await?;
        
        // 更新统计信息
        self.update_memory_stats().await?;
        let after_stats = self.get_memory_stats().await;
        
        let memory_freed = before_stats.total_memory_mb - after_stats.total_memory_mb;
        
        Ok(MemoryCleanupResult {
            memory_before_mb: before_stats.total_memory_mb,
            memory_after_mb: after_stats.total_memory_mb,
            memory_freed_mb: memory_freed,
            cache_entries_cleaned: 0, // 需要从实际清理中获取
            gc_triggered: true,
            cleanup_duration: Duration::from_secs(1), // 需要实际测量
        })
    }
}

impl CacheManager {
    /// 创建新的缓存管理器
    pub fn new(name: String, config: CacheConfig) -> Self {
        let manager = Self {
            cache_name: name,
            entries: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(RwLock::new(CacheStats::default())),
            config,
        };
        
        // 启动清理任务
        let manager_clone = manager.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(manager_clone.config.cleanup_interval);
            
            loop {
                interval.tick().await;
                let _ = manager_clone.cleanup_expired_entries().await;
            }
        });
        
        manager
    }

    /// 设置缓存条目
    pub async fn set(&self, key: String, data: Vec<u8>, ttl: Option<Duration>) -> DomainResult<()> {
        let mut entries = self.entries.write().await;
        let mut stats = self.stats.write().await;
        
        // 检查是否需要淘汰
        if entries.len() >= self.config.max_entries {
            self.evict_entries(&mut entries, &mut stats).await;
        }
        
        let entry = CacheEntry {
            key: key.clone(),
            data,
            created_at: Instant::now(),
            last_accessed: Instant::now(),
            access_count: 0,
            ttl: ttl.or(Some(self.config.default_ttl)),
        };
        
        entries.insert(key, entry);
        self.update_stats(&mut stats, &entries).await;
        
        Ok(())
    }

    /// 获取缓存条目
    pub async fn get(&self, key: &str) -> Option<Vec<u8>> {
        let mut entries = self.entries.write().await;
        let mut stats = self.stats.write().await;
        
        if let Some(entry) = entries.get_mut(key) {
            // 检查是否过期
            if let Some(ttl) = entry.ttl {
                if entry.created_at.elapsed() > ttl {
                    entries.remove(key);
                    stats.miss_count += 1;
                    return None;
                }
            }
            
            entry.last_accessed = Instant::now();
            entry.access_count += 1;
            stats.hit_count += 1;
            
            Some(entry.data.clone())
        } else {
            stats.miss_count += 1;
            None
        }
    }

    /// 清理过期条目
    pub async fn cleanup_expired_entries(&self) -> usize {
        let mut entries = self.entries.write().await;
        let mut stats = self.stats.write().await;
        
        let initial_count = entries.len();
        let now = Instant::now();
        
        entries.retain(|_, entry| {
            if let Some(ttl) = entry.ttl {
                now.duration_since(entry.created_at) <= ttl
            } else {
                true
            }
        });
        
        let cleaned_count = initial_count - entries.len();
        stats.eviction_count += cleaned_count as u64;
        stats.last_cleanup = chrono::Utc::now();
        
        self.update_stats(&mut stats, &entries).await;
        
        if cleaned_count > 0 {
            debug!("缓存 {} 清理了 {} 个过期条目", self.cache_name, cleaned_count);
        }
        
        cleaned_count
    }

    /// 淘汰条目
    async fn evict_entries(&self, entries: &mut HashMap<String, CacheEntry>, stats: &mut CacheStats) {
        let evict_count = (entries.len() as f64 * 0.1) as usize; // 淘汰10%
        
        match self.config.eviction_policy {
            EvictionPolicy::LRU => {
                let mut sorted_entries: Vec<_> = entries.iter().map(|(k, v)| (k.clone(), v.last_accessed)).collect();
                sorted_entries.sort_by_key(|(_, last_accessed)| *last_accessed);
                
                for (key, _) in sorted_entries.iter().take(evict_count) {
                    entries.remove(key);
                }
            }
            EvictionPolicy::LFU => {
                let mut sorted_entries: Vec<_> = entries.iter().map(|(k, v)| (k.clone(), v.access_count)).collect();
                sorted_entries.sort_by_key(|(_, access_count)| *access_count);
                
                for (key, _) in sorted_entries.iter().take(evict_count) {
                    entries.remove(key);
                }
            }
            EvictionPolicy::TTL => {
                let mut sorted_entries: Vec<_> = entries.iter().map(|(k, v)| (k.clone(), v.created_at)).collect();
                sorted_entries.sort_by_key(|(_, created_at)| *created_at);
                
                for (key, _) in sorted_entries.iter().take(evict_count) {
                    entries.remove(key);
                }
            }
            EvictionPolicy::FIFO => {
                let mut sorted_entries: Vec<_> = entries.iter().map(|(k, v)| (k.clone(), v.created_at)).collect();
                sorted_entries.sort_by_key(|(_, created_at)| *created_at);
                
                for (key, _) in sorted_entries.iter().take(evict_count) {
                    entries.remove(key);
                }
            }
        }
        
        stats.eviction_count += evict_count as u64;
    }

    /// 更新统计信息
    async fn update_stats(&self, stats: &mut CacheStats, entries: &HashMap<String, CacheEntry>) {
        stats.total_entries = entries.len();
        stats.total_memory_mb = entries.values()
            .map(|entry| entry.data.len() as f64 / 1024.0 / 1024.0)
            .sum();
        
        let total_requests = stats.hit_count + stats.miss_count;
        stats.hit_rate = if total_requests > 0 {
            (stats.hit_count as f64 / total_requests as f64) * 100.0
        } else {
            0.0
        };
    }

    /// 获取缓存统计
    pub async fn get_stats(&self) -> CacheStats {
        self.stats.read().await.clone()
    }
}

impl Default for MemoryStats {
    fn default() -> Self {
        Self {
            total_memory_mb: 0.0,
            heap_memory_mb: 0.0,
            cache_memory_mb: 0.0,
            connection_pool_memory_mb: 0.0,
            websocket_memory_mb: 0.0,
            memory_usage_percent: 0.0,
            gc_stats: GcStats::default(),
            last_updated: chrono::Utc::now(),
        }
    }
}

impl Default for GcStats {
    fn default() -> Self {
        Self {
            total_collections: 0,
            total_gc_time_ms: 0,
            average_gc_time_ms: 0.0,
            last_gc_time: None,
        }
    }
}

impl Default for CacheStats {
    fn default() -> Self {
        Self {
            total_entries: 0,
            total_memory_mb: 0.0,
            hit_count: 0,
            miss_count: 0,
            hit_rate: 0.0,
            eviction_count: 0,
            last_cleanup: chrono::Utc::now(),
        }
    }
}

/// 内存优化建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryOptimizationSuggestion {
    pub category: OptimizationCategory,
    pub priority: OptimizationPriority,
    pub description: String,
    pub recommended_action: String,
    pub estimated_savings_mb: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptimizationCategory {
    Memory,
    Cache,
    GarbageCollection,
    ConnectionPool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptimizationPriority {
    Low,
    Medium,
    High,
    Critical,
}

/// 内存清理结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryCleanupResult {
    pub memory_before_mb: f64,
    pub memory_after_mb: f64,
    pub memory_freed_mb: f64,
    pub cache_entries_cleaned: usize,
    pub gc_triggered: bool,
    pub cleanup_duration: Duration,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_memory_optimizer_creation() {
        let optimizer = MemoryOptimizer::new(MemoryOptimizerConfig::default());
        let stats = optimizer.get_memory_stats().await;
        assert_eq!(stats.total_memory_mb, 0.0);
    }

    #[tokio::test]
    async fn test_cache_manager() {
        let manager = CacheManager::new("test_cache".to_string(), CacheConfig::default());
        
        // 测试设置和获取
        let data = b"test_data".to_vec();
        manager.set("key1".to_string(), data.clone(), None).await.unwrap();
        
        let retrieved = manager.get("key1").await;
        assert_eq!(retrieved, Some(data));
        
        // 测试统计
        let stats = manager.get_stats().await;
        assert_eq!(stats.total_entries, 1);
        assert_eq!(stats.hit_count, 1);
    }
}
