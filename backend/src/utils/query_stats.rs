/// 查询统计收集器
/// 
/// 提供更详细的查询性能统计和分析功能
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use serde::{Serialize, Deserialize};

/// 查询统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryStats {
    /// 操作名称
    pub operation: String,
    /// 总调用次数
    pub total_calls: u64,
    /// 成功次数
    pub success_calls: u64,
    /// 失败次数
    pub error_calls: u64,
    /// 总执行时间（毫秒）
    pub total_duration_ms: u64,
    /// 平均执行时间（毫秒）
    pub avg_duration_ms: f64,
    /// 最小执行时间（毫秒）
    pub min_duration_ms: u64,
    /// 最大执行时间（毫秒）
    pub max_duration_ms: u64,
    /// 慢查询次数（>100ms）
    pub slow_query_count: u64,
    /// 最后更新时间
    pub last_updated: std::time::SystemTime,
}

impl QueryStats {
    fn new(operation: String) -> Self {
        Self {
            operation,
            total_calls: 0,
            success_calls: 0,
            error_calls: 0,
            total_duration_ms: 0,
            avg_duration_ms: 0.0,
            min_duration_ms: u64::MAX,
            max_duration_ms: 0,
            slow_query_count: 0,
            last_updated: std::time::SystemTime::now(),
        }
    }

    fn update(&mut self, duration: Duration, success: bool) {
        let duration_ms = duration.as_millis() as u64;
        
        self.total_calls += 1;
        if success {
            self.success_calls += 1;
        } else {
            self.error_calls += 1;
        }
        
        self.total_duration_ms += duration_ms;
        self.avg_duration_ms = self.total_duration_ms as f64 / self.total_calls as f64;
        
        if duration_ms < self.min_duration_ms {
            self.min_duration_ms = duration_ms;
        }
        if duration_ms > self.max_duration_ms {
            self.max_duration_ms = duration_ms;
        }
        
        if duration_ms > 100 {
            self.slow_query_count += 1;
        }
        
        self.last_updated = std::time::SystemTime::now();
    }

    /// 获取成功率百分比
    pub fn success_rate(&self) -> f64 {
        if self.total_calls == 0 {
            0.0
        } else {
            (self.success_calls as f64 / self.total_calls as f64) * 100.0
        }
    }

    /// 获取慢查询率百分比
    pub fn slow_query_rate(&self) -> f64 {
        if self.total_calls == 0 {
            0.0
        } else {
            (self.slow_query_count as f64 / self.total_calls as f64) * 100.0
        }
    }
}

/// 全局查询统计收集器
#[derive(Debug)]
pub struct QueryStatsCollector {
    stats: Arc<Mutex<HashMap<String, QueryStats>>>,
}

impl QueryStatsCollector {
    pub fn new() -> Self {
        Self {
            stats: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    /// 记录查询执行
    pub fn record_query(&self, operation: &str, duration: Duration, success: bool) {
        let mut stats = self.stats.lock().unwrap();
        let entry = stats.entry(operation.to_string())
            .or_insert_with(|| QueryStats::new(operation.to_string()));
        entry.update(duration, success);
    }

    /// 获取所有统计信息
    pub fn get_all_stats(&self) -> HashMap<String, QueryStats> {
        self.stats.lock().unwrap().clone()
    }

    /// 获取特定操作的统计信息
    pub fn get_stats(&self, operation: &str) -> Option<QueryStats> {
        self.stats.lock().unwrap().get(operation).cloned()
    }

    /// 获取慢查询统计
    pub fn get_slow_queries(&self) -> Vec<QueryStats> {
        self.stats.lock().unwrap()
            .values()
            .filter(|stats| stats.slow_query_count > 0)
            .cloned()
            .collect()
    }

    /// 获取性能最差的查询（按平均执行时间排序）
    pub fn get_slowest_queries(&self, limit: usize) -> Vec<QueryStats> {
        let mut stats: Vec<QueryStats> = self.stats.lock().unwrap()
            .values()
            .cloned()
            .collect();
        
        stats.sort_by(|a, b| b.avg_duration_ms.partial_cmp(&a.avg_duration_ms).unwrap());
        stats.truncate(limit);
        stats
    }

    /// 清理旧的统计数据
    pub fn cleanup_old_stats(&self, max_age: Duration) {
        let cutoff = std::time::SystemTime::now() - max_age;
        let mut stats = self.stats.lock().unwrap();
        stats.retain(|_, stat| stat.last_updated > cutoff);
    }

    /// 重置所有统计信息
    pub fn reset_all(&self) {
        let mut stats = self.stats.lock().unwrap();
        stats.clear();
    }
}

impl Default for QueryStatsCollector {
    fn default() -> Self {
        Self::new()
    }
}

/// 全局查询统计收集器实例
static GLOBAL_QUERY_STATS: std::sync::OnceLock<QueryStatsCollector> = std::sync::OnceLock::new();

/// 获取全局查询统计收集器
pub fn get_query_stats_collector() -> &'static QueryStatsCollector {
    GLOBAL_QUERY_STATS.get_or_init(QueryStatsCollector::new)
}

/// 记录查询统计的便捷函数
pub fn record_query_stats(operation: &str, duration: Duration, success: bool) {
    get_query_stats_collector().record_query(operation, duration, success);
}

/// 获取查询性能报告
pub fn get_performance_report() -> serde_json::Value {
    let collector = get_query_stats_collector();
    let all_stats = collector.get_all_stats();
    let slow_queries = collector.get_slow_queries();
    let slowest_queries = collector.get_slowest_queries(10);

    serde_json::json!({
        "total_operations": all_stats.len(),
        "all_stats": all_stats,
        "slow_queries": slow_queries,
        "slowest_queries": slowest_queries,
        "summary": {
            "total_calls": all_stats.values().map(|s| s.total_calls).sum::<u64>(),
            "total_errors": all_stats.values().map(|s| s.error_calls).sum::<u64>(),
            "total_slow_queries": all_stats.values().map(|s| s.slow_query_count).sum::<u64>(),
            "avg_success_rate": if all_stats.is_empty() { 0.0 } else {
                all_stats.values().map(|s| s.success_rate()).sum::<f64>() / all_stats.len() as f64
            }
        }
    })
}
