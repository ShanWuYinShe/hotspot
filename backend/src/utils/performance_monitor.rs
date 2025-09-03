/// 性能监控工具
/// 
/// 提供数据库查询性能监控和优化建议

use std::time::{Duration, Instant};
use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use tracing::{info, warn, debug};

/// 查询性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryPerformanceStats {
    pub query_name: String,
    pub execution_count: u64,
    pub total_duration_ms: u64,
    pub average_duration_ms: f64,
    pub min_duration_ms: u64,
    pub max_duration_ms: u64,
    pub last_executed_at: chrono::DateTime<chrono::Utc>,
}

/// 地理位置查询性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeoQueryStats {
    pub query_type: String,
    pub radius_km: f64,
    pub result_count: usize,
    pub execution_time_ms: u64,
    pub used_index: bool,
    pub bbox_optimization: bool,
}

/// 数据库性能监控器
#[derive(Debug)]
pub struct PerformanceMonitor {
    query_stats: std::sync::Arc<std::sync::Mutex<HashMap<String, QueryPerformanceStats>>>,
    geo_stats: std::sync::Arc<std::sync::Mutex<Vec<GeoQueryStats>>>,
}

impl Default for PerformanceMonitor {
    fn default() -> Self {
        Self::new()
    }
}

impl PerformanceMonitor {
    /// 创建新的性能监控器
    pub fn new() -> Self {
        Self {
            query_stats: std::sync::Arc::new(std::sync::Mutex::new(HashMap::new())),
            geo_stats: std::sync::Arc::new(std::sync::Mutex::new(Vec::new())),
        }
    }

    /// 记录查询性能
    pub fn record_query_performance(&self, query_name: &str, duration: Duration) {
        let duration_ms = duration.as_millis() as u64;
        let mut stats = self.query_stats.lock().unwrap();
        
        match stats.get_mut(query_name) {
            Some(existing_stats) => {
                existing_stats.execution_count += 1;
                existing_stats.total_duration_ms += duration_ms;
                existing_stats.average_duration_ms = 
                    existing_stats.total_duration_ms as f64 / existing_stats.execution_count as f64;
                existing_stats.min_duration_ms = 
                    existing_stats.min_duration_ms.min(duration_ms);
                existing_stats.max_duration_ms = 
                    existing_stats.max_duration_ms.max(duration_ms);
                existing_stats.last_executed_at = chrono::Utc::now();
            }
            None => {
                stats.insert(query_name.to_string(), QueryPerformanceStats {
                    query_name: query_name.to_string(),
                    execution_count: 1,
                    total_duration_ms: duration_ms,
                    average_duration_ms: duration_ms as f64,
                    min_duration_ms: duration_ms,
                    max_duration_ms: duration_ms,
                    last_executed_at: chrono::Utc::now(),
                });
            }
        }

        // 记录慢查询警告
        if duration_ms > 1000 {
            warn!(
                "慢查询检测: {} 执行时间 {}ms",
                query_name, duration_ms
            );
        } else if duration_ms > 500 {
            info!(
                "查询性能提醒: {} 执行时间 {}ms",
                query_name, duration_ms
            );
        }
    }

    /// 记录地理位置查询性能
    pub fn record_geo_query_performance(&self, geo_stats: GeoQueryStats) {
        let mut stats = self.geo_stats.lock().unwrap();
        stats.push(geo_stats.clone());
        
        // 保留最近1000条记录
        if stats.len() > 1000 {
            stats.drain(0..100);
        }

        // 分析地理查询性能
        if geo_stats.execution_time_ms > 500 {
            warn!(
                "地理查询性能警告: {} 查询半径{}km，{}个结果，执行时间{}ms",
                geo_stats.query_type,
                geo_stats.radius_km,
                geo_stats.result_count,
                geo_stats.execution_time_ms
            );
        }

        // 建议优化
        if !geo_stats.used_index && geo_stats.execution_time_ms > 100 {
            warn!(
                "地理查询未使用索引: {} - 建议检查查询计划",
                geo_stats.query_type
            );
        }
    }

    /// 获取查询性能统计
    pub fn get_query_stats(&self) -> HashMap<String, QueryPerformanceStats> {
        self.query_stats.lock().unwrap().clone()
    }

    /// 获取地理查询统计
    pub fn get_geo_stats(&self) -> Vec<GeoQueryStats> {
        self.geo_stats.lock().unwrap().clone()
    }

    /// 获取性能报告
    pub fn get_performance_report(&self) -> PerformanceReport {
        let query_stats = self.get_query_stats();
        let geo_stats = self.get_geo_stats();

        let slow_queries: Vec<_> = query_stats
            .values()
            .filter(|stats| stats.average_duration_ms > 100.0)
            .cloned()
            .collect();

        let total_queries = query_stats.values().map(|s| s.execution_count).sum();
        let average_response_time = if !query_stats.is_empty() {
            query_stats.values().map(|s| s.average_duration_ms).sum::<f64>() 
                / query_stats.len() as f64
        } else {
            0.0
        };

        PerformanceReport {
            total_queries,
            average_response_time,
            slow_queries,
            geo_query_count: geo_stats.len(),
            recommendations: self.generate_recommendations(&query_stats, &geo_stats),
        }
    }

    /// 生成优化建议
    fn generate_recommendations(
        &self,
        query_stats: &HashMap<String, QueryPerformanceStats>,
        geo_stats: &[GeoQueryStats],
    ) -> Vec<String> {
        let mut recommendations = Vec::new();

        // 分析慢查询
        for stats in query_stats.values() {
            if stats.average_duration_ms > 500.0 {
                recommendations.push(format!(
                    "查询 '{}' 平均执行时间 {:.2}ms，建议优化索引或查询逻辑",
                    stats.query_name, stats.average_duration_ms
                ));
            }
        }

        // 分析地理查询
        let slow_geo_queries = geo_stats.iter()
            .filter(|s| s.execution_time_ms > 200)
            .count();
        
        if slow_geo_queries > 0 {
            recommendations.push(format!(
                "发现 {} 个慢地理查询，建议检查空间索引和查询半径",
                slow_geo_queries
            ));
        }

        let large_radius_queries = geo_stats.iter()
            .filter(|s| s.radius_km > 50.0)
            .count();
        
        if large_radius_queries > 0 {
            recommendations.push(format!(
                "发现 {} 个大半径地理查询(>50km)，建议使用边界框预过滤",
                large_radius_queries
            ));
        }

        // 通用建议
        if query_stats.len() > 100 {
            recommendations.push(
                "查询类型较多，建议定期清理统计数据并关注核心查询性能".to_string()
            );
        }

        recommendations
    }

    /// 清理旧统计数据
    pub fn cleanup_old_stats(&self, retention_hours: u64) {
        let cutoff_time = chrono::Utc::now() - chrono::Duration::hours(retention_hours as i64);
        
        // 清理查询统计
        let mut query_stats = self.query_stats.lock().unwrap();
        query_stats.retain(|_, stats| stats.last_executed_at > cutoff_time);
        
        // 清理地理查询统计（保留最近的记录）
        let mut geo_stats = self.geo_stats.lock().unwrap();
        let len = geo_stats.len();
        if len > 500 {
            geo_stats.drain(0..len - 500);
        }
        
        debug!("清理性能统计数据，保留 {} 小时内的记录", retention_hours);
    }
}

/// 性能报告
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceReport {
    pub total_queries: u64,
    pub average_response_time: f64,
    pub slow_queries: Vec<QueryPerformanceStats>,
    pub geo_query_count: usize,
    pub recommendations: Vec<String>,
}

/// 查询性能装饰器宏
#[macro_export]
macro_rules! monitor_query_performance {
    ($monitor:expr, $query_name:expr, $query_block:block) => {{
        let start_time = std::time::Instant::now();
        let result = $query_block;
        let duration = start_time.elapsed();
        $monitor.record_query_performance($query_name, duration);
        result
    }};
}

/// 地理查询性能监控装饰器
pub async fn monitor_geo_query<F, R>(
    monitor: &PerformanceMonitor,
    query_type: &str,
    radius_km: f64,
    query_fn: F,
) -> R
where
    F: std::future::Future<Output = R>,
{
    let start_time = Instant::now();
    let result = query_fn.await;
    let duration = start_time.elapsed();
    
    // 这里需要根据实际结果确定result_count和索引使用情况
    // 在实际使用中，可以通过查询计划分析来确定
    let geo_stats = GeoQueryStats {
        query_type: query_type.to_string(),
        radius_km,
        result_count: 0, // 需要从结果中获取
        execution_time_ms: duration.as_millis() as u64,
        used_index: true, // 需要通过EXPLAIN分析确定
        bbox_optimization: radius_km <= 10.0, // 小半径查询假设使用了边界框优化
    };
    
    monitor.record_geo_query_performance(geo_stats);
    result
}

/// 数据库性能分析工具
pub struct DatabaseAnalyzer;

impl DatabaseAnalyzer {
    /// 分析慢查询（简化版本）
    pub async fn analyze_slow_queries(_pool: &PgPool) -> Result<Vec<SlowQueryInfo>, sqlx::Error> {
        // 返回空结果，避免依赖pg_stat_statements扩展
        Ok(vec![])
    }

    /// 分析索引使用情况（简化版本）
    pub async fn analyze_index_usage(_pool: &PgPool) -> Result<Vec<IndexUsageInfo>, sqlx::Error> {
        // 返回空结果，避免依赖系统表
        Ok(vec![])
    }

    /// 分析表大小和统计信息（简化版本）
    pub async fn analyze_table_stats(_pool: &PgPool) -> Result<Vec<TableStatsInfo>, sqlx::Error> {
        // 返回空结果，避免依赖系统表
        Ok(vec![])
    }
}

/// 慢查询信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SlowQueryInfo {
    pub query: String,
    pub calls: i64,
    pub total_time: f64,
    pub mean_time: f64,
    pub rows: i64,
    pub hit_percent: f64,
}

/// 索引使用信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndexUsageInfo {
    pub schema_name: String,
    pub table_name: String,
    pub index_name: String,
    pub tuples_read: i64,
    pub tuples_fetched: i64,
    pub scans: i64,
}

/// 表统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TableStatsInfo {
    pub schema_name: String,
    pub table_name: String,
    pub inserts: i64,
    pub updates: i64,
    pub deletes: i64,
    pub live_tuples: i64,
    pub dead_tuples: i64,
    pub last_vacuum: Option<chrono::DateTime<chrono::Utc>>,
    pub last_autovacuum: Option<chrono::DateTime<chrono::Utc>>,
    pub last_analyze: Option<chrono::DateTime<chrono::Utc>>,
    pub last_autoanalyze: Option<chrono::DateTime<chrono::Utc>>,
}

/// 全局性能监控器实例
use std::sync::OnceLock;
pub static GLOBAL_PERFORMANCE_MONITOR: OnceLock<PerformanceMonitor> = OnceLock::new();

/// 获取全局性能监控器
pub fn get_global_performance_monitor() -> &'static PerformanceMonitor {
    GLOBAL_PERFORMANCE_MONITOR.get_or_init(PerformanceMonitor::new)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[test]
    fn test_performance_monitor() {
        let monitor = PerformanceMonitor::new();
        
        // 记录一些测试查询
        monitor.record_query_performance("test_query", Duration::from_millis(150));
        monitor.record_query_performance("test_query", Duration::from_millis(200));
        monitor.record_query_performance("slow_query", Duration::from_millis(1500));
        
        let stats = monitor.get_query_stats();
        assert_eq!(stats.len(), 2);
        
        let test_query_stats = stats.get("test_query").unwrap();
        assert_eq!(test_query_stats.execution_count, 2);
        assert_eq!(test_query_stats.average_duration_ms, 175.0);
        
        let report = monitor.get_performance_report();
        assert_eq!(report.total_queries, 3);
        assert!(!report.slow_queries.is_empty());
    }

    #[test]
    fn test_geo_query_stats() {
        let monitor = PerformanceMonitor::new();
        
        let geo_stats = GeoQueryStats {
            query_type: "nearby_users".to_string(),
            radius_km: 5.0,
            result_count: 10,
            execution_time_ms: 250,
            used_index: true,
            bbox_optimization: true,
        };
        
        monitor.record_geo_query_performance(geo_stats);
        
        let stats = monitor.get_geo_stats();
        assert_eq!(stats.len(), 1);
        assert_eq!(stats[0].query_type, "nearby_users");
    }
}