/// 数据库查询优化器
/// 
/// 提供地理位置查询优化、查询缓存和性能监控功能
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use serde::{Serialize, Deserialize};
use tracing::{debug, warn, info};

/// 查询优化器配置
#[derive(Debug, Clone)]
pub struct QueryOptimizerConfig {
    /// 地理查询缓存TTL（秒）
    pub geo_cache_ttl: u64,
    /// 最大缓存条目数
    pub max_cache_entries: usize,
    /// 慢查询阈值（毫秒）
    pub slow_query_threshold_ms: u64,
    /// 是否启用查询缓存
    pub enable_query_cache: bool,
    /// 是否启用查询优化建议
    pub enable_optimization_hints: bool,
}

impl Default for QueryOptimizerConfig {
    fn default() -> Self {
        Self {
            geo_cache_ttl: 300, // 5分钟
            max_cache_entries: 1000,
            slow_query_threshold_ms: 100,
            enable_query_cache: true,
            enable_optimization_hints: true,
        }
    }
}

/// 查询缓存条目
#[derive(Debug, Clone)]
struct CacheEntry {
    data: String,
    created_at: Instant,
    ttl: Duration,
    hit_count: u64,
}

impl CacheEntry {
    fn new(data: String, ttl: Duration) -> Self {
        Self {
            data,
            created_at: Instant::now(),
            ttl,
            hit_count: 0,
        }
    }

    fn is_expired(&self) -> bool {
        self.created_at.elapsed() > self.ttl
    }

    fn hit(&mut self) -> &str {
        self.hit_count += 1;
        &self.data
    }
}

/// 查询性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPerformanceStats {
    pub total_queries: u64,
    pub cached_queries: u64,
    pub slow_queries: u64,
    pub average_query_time_ms: f64,
    pub cache_hit_rate: f64,
    pub geo_queries: u64,
    pub optimization_suggestions: Vec<String>,
}

/// 查询优化建议
#[derive(Debug, Clone)]
pub struct OptimizationHint {
    pub query_type: String,
    pub suggestion: String,
    pub impact: String,
    pub priority: u8, // 1-10, 10为最高优先级
}

/// 数据库查询优化器
pub struct QueryOptimizer {
    config: QueryOptimizerConfig,
    cache: Arc<Mutex<HashMap<String, CacheEntry>>>,
    stats: Arc<Mutex<QueryPerformanceStats>>,
    query_times: Arc<Mutex<Vec<u64>>>,
    optimization_hints: Arc<Mutex<Vec<OptimizationHint>>>,
}

impl QueryOptimizer {
    /// 创建新的查询优化器
    pub fn new(config: QueryOptimizerConfig) -> Self {
        Self {
            config,
            cache: Arc::new(Mutex::new(HashMap::new())),
            stats: Arc::new(Mutex::new(QueryPerformanceStats {
                total_queries: 0,
                cached_queries: 0,
                slow_queries: 0,
                average_query_time_ms: 0.0,
                cache_hit_rate: 0.0,
                geo_queries: 0,
                optimization_suggestions: Vec::new(),
            })),
            query_times: Arc::new(Mutex::new(Vec::new())),
            optimization_hints: Arc::new(Mutex::new(Vec::new())),
        }
    }

    /// 优化地理位置查询参数
    pub fn optimize_geo_query(&self, latitude: f64, longitude: f64, radius: f64) -> (f64, f64, f64) {
        // 限制查询半径，避免过大的查询范围
        let optimized_radius = if radius > 50000.0 {
            warn!("地理查询半径过大: {}米，限制为50km", radius);
            if self.config.enable_optimization_hints {
                self.add_optimization_hint(OptimizationHint {
                    query_type: "geo_query".to_string(),
                    suggestion: "查询半径过大，建议限制在50km以内以提高性能".to_string(),
                    impact: "高".to_string(),
                    priority: 8,
                });
            }
            50000.0
        } else {
            radius
        };

        // 验证坐标范围
        let optimized_lat = latitude.clamp(-90.0, 90.0);
        let optimized_lng = longitude.clamp(-180.0, 180.0);

        if optimized_lat != latitude || optimized_lng != longitude {
            warn!("地理坐标超出有效范围，已调整: ({}, {}) -> ({}, {})", 
                  latitude, longitude, optimized_lat, optimized_lng);
        }

        debug!("地理查询优化: 坐标({}, {}), 半径{}米", optimized_lat, optimized_lng, optimized_radius);
        
        (optimized_lat, optimized_lng, optimized_radius)
    }

    /// 生成查询缓存键
    pub fn generate_cache_key(&self, query_type: &str, params: &[&str]) -> String {
        let mut key = format!("{}:", query_type);
        for param in params {
            key.push_str(param);
            key.push(':');
        }
        key
    }

    /// 获取缓存的查询结果
    pub fn get_cached_result(&self, cache_key: &str) -> Option<String> {
        if !self.config.enable_query_cache {
            return None;
        }

        let mut cache = self.cache.lock().unwrap();
        
        if let Some(entry) = cache.get_mut(cache_key) {
            if entry.is_expired() {
                cache.remove(cache_key);
                debug!("缓存条目已过期: {}", cache_key);
                return None;
            }
            
            let result = entry.hit().to_string();
            debug!("缓存命中: {}, 命中次数: {}", cache_key, entry.hit_count);
            
            // 更新统计
            let mut stats = self.stats.lock().unwrap();
            stats.cached_queries += 1;
            
            return Some(result);
        }

        None
    }

    /// 缓存查询结果
    pub fn cache_result(&self, cache_key: String, result: String, ttl_seconds: Option<u64>) {
        if !self.config.enable_query_cache {
            return;
        }

        let ttl = Duration::from_secs(ttl_seconds.unwrap_or(self.config.geo_cache_ttl));
        let mut cache = self.cache.lock().unwrap();

        // 检查缓存大小限制
        if cache.len() >= self.config.max_cache_entries {
            // 清理过期条目
            cache.retain(|_, entry| !entry.is_expired());
            
            // 如果还是太多，清理命中次数最少的条目
            if cache.len() >= self.config.max_cache_entries {
                let mut entries: Vec<_> = cache.iter().collect();
                entries.sort_by_key(|(_, entry)| entry.hit_count);
                
                let to_remove = entries.len() - self.config.max_cache_entries + 1;
                for (key, _) in entries.into_iter().take(to_remove) {
                    cache.remove(key);
                }
                
                debug!("清理了{}个低命中率缓存条目", to_remove);
            }
        }

        cache.insert(cache_key.clone(), CacheEntry::new(result, ttl));
        debug!("缓存查询结果: {}", cache_key);
    }

    /// 记录查询性能
    pub fn record_query_performance(&self, query_type: &str, duration_ms: u64, is_geo_query: bool) {
        let mut stats = self.stats.lock().unwrap();
        stats.total_queries += 1;
        
        if is_geo_query {
            stats.geo_queries += 1;
        }

        // 记录慢查询
        if duration_ms > self.config.slow_query_threshold_ms {
            stats.slow_queries += 1;
            warn!("慢查询检测: {} 耗时 {}ms", query_type, duration_ms);
            
            if self.config.enable_optimization_hints {
                self.add_optimization_hint(OptimizationHint {
                    query_type: query_type.to_string(),
                    suggestion: format!("查询耗时{}ms，建议优化索引或查询条件", duration_ms),
                    impact: if duration_ms > 1000 { "高" } else { "中" }.to_string(),
                    priority: if duration_ms > 1000 { 9 } else { 6 },
                });
            }
        }

        // 更新平均查询时间
        let mut query_times = self.query_times.lock().unwrap();
        query_times.push(duration_ms);
        
        // 保持最近1000次查询的记录
        if query_times.len() > 1000 {
            query_times.remove(0);
        }
        
        let avg_time = query_times.iter().sum::<u64>() as f64 / query_times.len() as f64;
        stats.average_query_time_ms = avg_time;
        
        // 计算缓存命中率
        if stats.total_queries > 0 {
            stats.cache_hit_rate = (stats.cached_queries as f64 / stats.total_queries as f64) * 100.0;
        }

        debug!("查询性能记录: {} 耗时 {}ms, 平均耗时 {:.2}ms", 
               query_type, duration_ms, avg_time);
    }

    /// 添加优化建议
    fn add_optimization_hint(&self, hint: OptimizationHint) {
        let mut hints = self.optimization_hints.lock().unwrap();
        hints.push(hint);
        
        // 保持最近50个建议
        if hints.len() > 50 {
            hints.remove(0);
        }
    }

    /// 获取性能统计
    pub fn get_performance_stats(&self) -> QueryPerformanceStats {
        let mut stats = self.stats.lock().unwrap();
        
        // 更新优化建议
        let hints = self.optimization_hints.lock().unwrap();
        stats.optimization_suggestions = hints.iter()
            .map(|h| format!("[{}] {}: {}", h.query_type, h.suggestion, h.impact))
            .collect();
        
        stats.clone()
    }

    /// 清理过期缓存
    pub fn cleanup_expired_cache(&self) {
        let mut cache = self.cache.lock().unwrap();
        let initial_size = cache.len();
        
        cache.retain(|_, entry| !entry.is_expired());
        
        let cleaned = initial_size - cache.len();
        if cleaned > 0 {
            info!("清理了{}个过期缓存条目", cleaned);
        }
    }

    /// 获取缓存统计
    pub fn get_cache_stats(&self) -> (usize, usize) {
        let cache = self.cache.lock().unwrap();
        (cache.len(), self.config.max_cache_entries)
    }

    /// 清空所有缓存
    pub fn clear_cache(&self) {
        let mut cache = self.cache.lock().unwrap();
        let cleared = cache.len();
        cache.clear();
        info!("清空了{}个缓存条目", cleared);
    }

    /// 预热常用查询缓存
    pub fn warmup_cache(&self, common_queries: Vec<(String, String)>) {
        info!("开始预热查询缓存，共{}个查询", common_queries.len());
        
        for (cache_key, result) in common_queries {
            self.cache_result(cache_key, result, None);
        }
        
        info!("查询缓存预热完成");
    }
}

/// 全局查询优化器实例
static mut GLOBAL_QUERY_OPTIMIZER: Option<Arc<QueryOptimizer>> = None;
static INIT: std::sync::Once = std::sync::Once::new();

/// 获取全局查询优化器
pub fn get_global_query_optimizer() -> Arc<QueryOptimizer> {
    unsafe {
        INIT.call_once(|| {
            let config = QueryOptimizerConfig::default();
            GLOBAL_QUERY_OPTIMIZER = Some(Arc::new(QueryOptimizer::new(config)));
        });
        GLOBAL_QUERY_OPTIMIZER.as_ref().unwrap().clone()
    }
}

/// 初始化查询优化器
pub fn init_query_optimizer(config: QueryOptimizerConfig) {
    unsafe {
        GLOBAL_QUERY_OPTIMIZER = Some(Arc::new(QueryOptimizer::new(config)));
    }
    info!("查询优化器初始化完成");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_geo_query_optimization() {
        let optimizer = QueryOptimizer::new(QueryOptimizerConfig::default());
        
        // 测试正常坐标
        let (lat, lng, radius) = optimizer.optimize_geo_query(39.9042, 116.4074, 1000.0);
        assert_eq!(lat, 39.9042);
        assert_eq!(lng, 116.4074);
        assert_eq!(radius, 1000.0);
        
        // 测试超出范围的坐标
        let (lat, lng, _) = optimizer.optimize_geo_query(100.0, 200.0, 1000.0);
        assert_eq!(lat, 90.0);
        assert_eq!(lng, 180.0);
        
        // 测试过大的半径
        let (_, _, radius) = optimizer.optimize_geo_query(39.9042, 116.4074, 100000.0);
        assert_eq!(radius, 50000.0);
    }

    #[test]
    fn test_cache_functionality() {
        let optimizer = QueryOptimizer::new(QueryOptimizerConfig::default());
        let cache_key = "test_query:param1:param2";
        let result = "test_result";
        
        // 测试缓存不存在
        assert!(optimizer.get_cached_result(cache_key).is_none());
        
        // 测试缓存存储
        optimizer.cache_result(cache_key.to_string(), result.to_string(), Some(1));
        
        // 测试缓存命中
        let cached = optimizer.get_cached_result(cache_key);
        assert!(cached.is_some());
        assert_eq!(cached.unwrap(), result);
        
        // 测试缓存过期（需要等待）
        std::thread::sleep(Duration::from_secs(2));
        assert!(optimizer.get_cached_result(cache_key).is_none());
    }

    #[test]
    fn test_performance_recording() {
        let optimizer = QueryOptimizer::new(QueryOptimizerConfig::default());
        
        // 记录几次查询
        optimizer.record_query_performance("test_query", 50, false);
        optimizer.record_query_performance("geo_query", 200, true);
        optimizer.record_query_performance("slow_query", 1500, false);
        
        let stats = optimizer.get_performance_stats();
        assert_eq!(stats.total_queries, 3);
        assert_eq!(stats.geo_queries, 1);
        assert_eq!(stats.slow_queries, 1);
        assert!(stats.average_query_time_ms > 0.0);
    }
}
