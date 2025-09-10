/// 缓存监控系统
/// 
/// 提供缓存命中率、性能统计和监控功能
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

/// 缓存操作类型
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum CacheOperation {
    Get,
    Set,
    Delete,
    Clear,
}

/// 缓存统计信息
#[derive(Debug, Clone)]
pub struct CacheStats {
    /// 缓存名称
    pub cache_name: String,
    /// 总请求次数
    pub total_requests: u64,
    /// 命中次数
    pub hits: u64,
    /// 未命中次数
    pub misses: u64,
    /// 设置操作次数
    pub sets: u64,
    /// 删除操作次数
    pub deletes: u64,
    /// 清空操作次数
    pub clears: u64,
    /// 总执行时间（毫秒）
    pub total_duration_ms: u64,
    /// 平均执行时间（毫秒）
    pub avg_duration_ms: f64,
    /// 最后更新时间
    pub last_updated: std::time::SystemTime,
}

impl CacheStats {
    fn new(cache_name: String) -> Self {
        Self {
            cache_name,
            total_requests: 0,
            hits: 0,
            misses: 0,
            sets: 0,
            deletes: 0,
            clears: 0,
            total_duration_ms: 0,
            avg_duration_ms: 0.0,
            last_updated: std::time::SystemTime::now(),
        }
    }

    /// 记录缓存操作
    fn record_operation(&mut self, operation: CacheOperation, hit: bool, duration: Duration) {
        let duration_ms = duration.as_millis() as u64;
        
        match operation {
            CacheOperation::Get => {
                self.total_requests += 1;
                if hit {
                    self.hits += 1;
                } else {
                    self.misses += 1;
                }
            }
            CacheOperation::Set => {
                self.sets += 1;
            }
            CacheOperation::Delete => {
                self.deletes += 1;
            }
            CacheOperation::Clear => {
                self.clears += 1;
            }
        }
        
        self.total_duration_ms += duration_ms;
        let total_ops = self.total_requests + self.sets + self.deletes + self.clears;
        if total_ops > 0 {
            self.avg_duration_ms = self.total_duration_ms as f64 / total_ops as f64;
        }
        
        self.last_updated = std::time::SystemTime::now();
    }

    /// 获取命中率百分比
    pub fn hit_rate(&self) -> f64 {
        if self.total_requests == 0 {
            0.0
        } else {
            (self.hits as f64 / self.total_requests as f64) * 100.0
        }
    }

    /// 获取未命中率百分比
    pub fn miss_rate(&self) -> f64 {
        if self.total_requests == 0 {
            0.0
        } else {
            (self.misses as f64 / self.total_requests as f64) * 100.0
        }
    }

    /// 获取总操作数
    pub fn total_operations(&self) -> u64 {
        self.total_requests + self.sets + self.deletes + self.clears
    }
}

/// 缓存监控器
#[derive(Debug)]
pub struct CacheMonitor {
    stats: Arc<Mutex<HashMap<String, CacheStats>>>,
}

impl CacheMonitor {
    pub fn new() -> Self {
        Self {
            stats: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// 记录缓存获取操作
    pub fn record_get(&self, cache_name: &str, hit: bool, duration: Duration) {
        let mut stats = self.stats.lock().unwrap();
        let entry = stats.entry(cache_name.to_string())
            .or_insert_with(|| CacheStats::new(cache_name.to_string()));
        entry.record_operation(CacheOperation::Get, hit, duration);
    }

    /// 记录缓存设置操作
    pub fn record_set(&self, cache_name: &str, duration: Duration) {
        let mut stats = self.stats.lock().unwrap();
        let entry = stats.entry(cache_name.to_string())
            .or_insert_with(|| CacheStats::new(cache_name.to_string()));
        entry.record_operation(CacheOperation::Set, false, duration);
    }

    /// 记录缓存删除操作
    pub fn record_delete(&self, cache_name: &str, duration: Duration) {
        let mut stats = self.stats.lock().unwrap();
        let entry = stats.entry(cache_name.to_string())
            .or_insert_with(|| CacheStats::new(cache_name.to_string()));
        entry.record_operation(CacheOperation::Delete, false, duration);
    }

    /// 记录缓存清空操作
    pub fn record_clear(&self, cache_name: &str, duration: Duration) {
        let mut stats = self.stats.lock().unwrap();
        let entry = stats.entry(cache_name.to_string())
            .or_insert_with(|| CacheStats::new(cache_name.to_string()));
        entry.record_operation(CacheOperation::Clear, false, duration);
    }

    /// 获取所有缓存统计
    pub fn get_all_stats(&self) -> HashMap<String, CacheStats> {
        self.stats.lock().unwrap().clone()
    }

    /// 获取特定缓存的统计
    pub fn get_cache_stats(&self, cache_name: &str) -> Option<CacheStats> {
        self.stats.lock().unwrap().get(cache_name).cloned()
    }

    /// 获取缓存性能报告
    pub fn get_performance_report(&self) -> serde_json::Value {
        let all_stats = self.get_all_stats();
        
        let total_requests: u64 = all_stats.values().map(|s| s.total_requests).sum();
        let total_hits: u64 = all_stats.values().map(|s| s.hits).sum();
        let total_misses: u64 = all_stats.values().map(|s| s.misses).sum();
        
        let overall_hit_rate = if total_requests > 0 {
            (total_hits as f64 / total_requests as f64) * 100.0
        } else {
            0.0
        };

        serde_json::json!({
            "summary": {
                "total_caches": all_stats.len(),
                "total_requests": total_requests,
                "total_hits": total_hits,
                "total_misses": total_misses,
                "overall_hit_rate": overall_hit_rate,
                "overall_miss_rate": 100.0 - overall_hit_rate
            },
            "cache_details": all_stats.values().map(|stats| {
                serde_json::json!({
                    "cache_name": stats.cache_name,
                    "total_requests": stats.total_requests,
                    "hits": stats.hits,
                    "misses": stats.misses,
                    "hit_rate": stats.hit_rate(),
                    "miss_rate": stats.miss_rate(),
                    "sets": stats.sets,
                    "deletes": stats.deletes,
                    "clears": stats.clears,
                    "total_operations": stats.total_operations(),
                    "avg_duration_ms": stats.avg_duration_ms,
                    "last_updated": stats.last_updated
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap_or_default()
                        .as_secs()
                })
            }).collect::<Vec<_>>()
        })
    }

    /// 重置所有统计
    pub fn reset_all(&self) {
        let mut stats = self.stats.lock().unwrap();
        stats.clear();
    }

    /// 清理旧的统计数据
    pub fn cleanup_old_stats(&self, max_age: Duration) {
        let cutoff = std::time::SystemTime::now() - max_age;
        let mut stats = self.stats.lock().unwrap();
        stats.retain(|_, stat| stat.last_updated > cutoff);
    }
}

impl Default for CacheMonitor {
    fn default() -> Self {
        Self::new()
    }
}

/// 全局缓存监控器实例
static GLOBAL_CACHE_MONITOR: std::sync::OnceLock<CacheMonitor> = std::sync::OnceLock::new();

/// 获取全局缓存监控器
pub fn get_cache_monitor() -> &'static CacheMonitor {
    GLOBAL_CACHE_MONITOR.get_or_init(CacheMonitor::new)
}

/// 记录缓存获取操作的便捷函数
pub fn record_cache_get(cache_name: &str, hit: bool, duration: Duration) {
    get_cache_monitor().record_get(cache_name, hit, duration);
}

/// 记录缓存设置操作的便捷函数
pub fn record_cache_set(cache_name: &str, duration: Duration) {
    get_cache_monitor().record_set(cache_name, duration);
}

/// 记录缓存删除操作的便捷函数
pub fn record_cache_delete(cache_name: &str, duration: Duration) {
    get_cache_monitor().record_delete(cache_name, duration);
}

/// 记录缓存清空操作的便捷函数
pub fn record_cache_clear(cache_name: &str, duration: Duration) {
    get_cache_monitor().record_clear(cache_name, duration);
}

/// 缓存监控装饰器 trait
pub trait CacheMonitorExt<T> {
    /// 监控缓存获取操作
    fn with_get_monitoring(self, cache_name: &str) -> T;
    
    /// 监控缓存设置操作
    fn with_set_monitoring(self, cache_name: &str) -> T;
}

/// 为 Future 实现缓存监控装饰器
impl<F, T> CacheMonitorExt<impl std::future::Future<Output = (T, bool)>> for F
where
    F: std::future::Future<Output = (T, bool)>,
{
    fn with_get_monitoring(self, cache_name: &str) -> impl std::future::Future<Output = (T, bool)> {
        let cache_name = cache_name.to_string();
        async move {
            let start = Instant::now();
            let (result, hit) = self.await;
            let duration = start.elapsed();
            record_cache_get(&cache_name, hit, duration);
            (result, hit)
        }
    }

    fn with_set_monitoring(self, cache_name: &str) -> impl std::future::Future<Output = (T, bool)> {
        let cache_name = cache_name.to_string();
        async move {
            let start = Instant::now();
            let (result, _) = self.await;
            let duration = start.elapsed();
            record_cache_set(&cache_name, duration);
            (result, true)
        }
    }
}
