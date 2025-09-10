/// 监控和指标系统 - 统一的监控、日志和指标收集
/// 
/// 监控特征：
/// - 结构化日志
/// - 指标收集
/// - 分布式追踪
/// - 健康检查

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use serde::{Deserialize, Serialize};
use tokio::sync::RwLock;
use crate::infrastructure::errors::ContextualError;

// === 指标系统 ===

/// 指标收集器
#[derive(Clone)]
pub struct MetricsCollector {
    counters: Arc<RwLock<HashMap<String, u64>>>,
    gauges: Arc<RwLock<HashMap<String, f64>>>,
    histograms: Arc<RwLock<HashMap<String, Vec<f64>>>>,
    timers: Arc<RwLock<HashMap<String, Vec<Duration>>>>,
}

impl MetricsCollector {
    /// 创建新的指标收集器
    pub fn new() -> Self {
        Self {
            counters: Arc::new(RwLock::new(HashMap::new())),
            gauges: Arc::new(RwLock::new(HashMap::new())),
            histograms: Arc::new(RwLock::new(HashMap::new())),
            timers: Arc::new(RwLock::new(HashMap::new())),
        }
    }
    
    /// 递增计数器
    pub async fn increment_counter(&self, name: &str, value: u64) {
        let mut counters = self.counters.write().await;
        *counters.entry(name.to_string()).or_insert(0) += value;
    }
    
    /// 设置仪表值
    pub async fn set_gauge(&self, name: &str, value: f64) {
        let mut gauges = self.gauges.write().await;
        gauges.insert(name.to_string(), value);
    }
    
    /// 记录直方图值
    pub async fn record_histogram(&self, name: &str, value: f64) {
        let mut histograms = self.histograms.write().await;
        histograms.entry(name.to_string()).or_insert_with(Vec::new).push(value);
    }
    
    /// 记录计时器值
    pub async fn record_timer(&self, name: &str, duration: Duration) {
        let mut timers = self.timers.write().await;
        timers.entry(name.to_string()).or_insert_with(Vec::new).push(duration);
    }
    
    /// 获取所有指标
    pub async fn get_metrics(&self) -> MetricsSnapshot {
        let counters = self.counters.read().await.clone();
        let gauges = self.gauges.read().await.clone();
        let histograms = self.histograms.read().await.clone();
        let timers = self.timers.read().await.clone();
        
        MetricsSnapshot {
            counters,
            gauges,
            histograms: histograms.into_iter()
                .map(|(k, v)| (k, HistogramStats::from_values(&v)))
                .collect(),
            timers: timers.into_iter()
                .map(|(k, v)| (k, TimerStats::from_durations(&v)))
                .collect(),
            timestamp: chrono::Utc::now(),
        }
    }
    
    /// 重置所有指标
    pub async fn reset(&self) {
        self.counters.write().await.clear();
        self.gauges.write().await.clear();
        self.histograms.write().await.clear();
        self.timers.write().await.clear();
    }
}

impl Default for MetricsCollector {
    fn default() -> Self {
        Self::new()
    }
}

/// 指标快照
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsSnapshot {
    pub counters: HashMap<String, u64>,
    pub gauges: HashMap<String, f64>,
    pub histograms: HashMap<String, HistogramStats>,
    pub timers: HashMap<String, TimerStats>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

/// 直方图统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HistogramStats {
    pub count: usize,
    pub sum: f64,
    pub min: f64,
    pub max: f64,
    pub mean: f64,
    pub p50: f64,
    pub p95: f64,
    pub p99: f64,
}

impl HistogramStats {
    /// 从值列表计算统计信息
    pub fn from_values(values: &[f64]) -> Self {
        if values.is_empty() {
            return Self {
                count: 0,
                sum: 0.0,
                min: 0.0,
                max: 0.0,
                mean: 0.0,
                p50: 0.0,
                p95: 0.0,
                p99: 0.0,
            };
        }
        
        let mut sorted_values = values.to_vec();
        sorted_values.sort_by(|a, b| a.partial_cmp(b).unwrap());
        
        let count = values.len();
        let sum = values.iter().sum();
        let min = sorted_values[0];
        let max = sorted_values[count - 1];
        let mean = sum / count as f64;
        
        let p50 = Self::percentile(&sorted_values, 0.5);
        let p95 = Self::percentile(&sorted_values, 0.95);
        let p99 = Self::percentile(&sorted_values, 0.99);
        
        Self {
            count,
            sum,
            min,
            max,
            mean,
            p50,
            p95,
            p99,
        }
    }
    
    /// 计算百分位数
    fn percentile(sorted_values: &[f64], p: f64) -> f64 {
        let index = (p * (sorted_values.len() - 1) as f64) as usize;
        sorted_values[index.min(sorted_values.len() - 1)]
    }
}

/// 计时器统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimerStats {
    pub count: usize,
    pub total_ms: f64,
    pub min_ms: f64,
    pub max_ms: f64,
    pub mean_ms: f64,
    pub p50_ms: f64,
    pub p95_ms: f64,
    pub p99_ms: f64,
}

impl TimerStats {
    /// 从持续时间列表计算统计信息
    pub fn from_durations(durations: &[Duration]) -> Self {
        if durations.is_empty() {
            return Self {
                count: 0,
                total_ms: 0.0,
                min_ms: 0.0,
                max_ms: 0.0,
                mean_ms: 0.0,
                p50_ms: 0.0,
                p95_ms: 0.0,
                p99_ms: 0.0,
            };
        }
        
        let values: Vec<f64> = durations.iter()
            .map(|d| d.as_secs_f64() * 1000.0) // 转换为毫秒
            .collect();
        
        let histogram_stats = HistogramStats::from_values(&values);
        
        Self {
            count: histogram_stats.count,
            total_ms: histogram_stats.sum,
            min_ms: histogram_stats.min,
            max_ms: histogram_stats.max,
            mean_ms: histogram_stats.mean,
            p50_ms: histogram_stats.p50,
            p95_ms: histogram_stats.p95,
            p99_ms: histogram_stats.p99,
        }
    }
}

// === 性能监控 ===

/// 性能监控器
pub struct PerformanceMonitor {
    metrics: MetricsCollector,
}

impl PerformanceMonitor {
    /// 创建新的性能监控器
    pub fn new(metrics: MetricsCollector) -> Self {
        Self { metrics }
    }
    
    /// 监控函数执行时间
    pub async fn time_function<F, T>(&self, name: &str, func: F) -> T
    where
        F: std::future::Future<Output = T>,
    {
        let start = Instant::now();
        let result = func.await;
        let duration = start.elapsed();
        
        self.metrics.record_timer(name, duration).await;
        self.metrics.increment_counter(&format!("{}_calls", name), 1).await;
        
        result
    }
    
    /// 监控函数执行时间（带错误计数）
    pub async fn time_function_with_error_tracking<F, T, E>(&self, name: &str, func: F) -> Result<T, E>
    where
        F: std::future::Future<Output = Result<T, E>>,
    {
        let start = Instant::now();
        let result = func.await;
        let duration = start.elapsed();
        
        self.metrics.record_timer(name, duration).await;
        self.metrics.increment_counter(&format!("{}_calls", name), 1).await;
        
        match &result {
            Ok(_) => {
                self.metrics.increment_counter(&format!("{}_success", name), 1).await;
            }
            Err(_) => {
                self.metrics.increment_counter(&format!("{}_errors", name), 1).await;
            }
        }
        
        result
    }
    
    /// 记录数据库查询性能
    pub async fn record_db_query(&self, query_type: &str, duration: Duration, rows_affected: u64) {
        self.metrics.record_timer(&format!("db_query_{}", query_type), duration).await;
        self.metrics.increment_counter(&format!("db_query_{}_calls", query_type), 1).await;
        self.metrics.record_histogram(&format!("db_query_{}_rows", query_type), rows_affected as f64).await;
    }
    
    /// 记录HTTP请求性能
    pub async fn record_http_request(&self, method: &str, path: &str, status: u16, duration: Duration) {
        let metric_name = format!("http_request_{}_{}", method.to_lowercase(), Self::sanitize_path(path));
        
        self.metrics.record_timer(&metric_name, duration).await;
        self.metrics.increment_counter(&format!("{}_calls", metric_name), 1).await;
        self.metrics.increment_counter(&format!("http_status_{}", status), 1).await;
        
        if status >= 400 {
            self.metrics.increment_counter(&format!("{}_errors", metric_name), 1).await;
        }
    }
    
    /// 记录缓存性能
    pub async fn record_cache_operation(&self, operation: &str, hit: bool, duration: Duration) {
        let metric_name = format!("cache_{}", operation);
        
        self.metrics.record_timer(&metric_name, duration).await;
        self.metrics.increment_counter(&format!("{}_calls", metric_name), 1).await;
        
        if hit {
            self.metrics.increment_counter(&format!("{}_hits", metric_name), 1).await;
        } else {
            self.metrics.increment_counter(&format!("{}_misses", metric_name), 1).await;
        }
    }
    
    /// 清理路径用于指标名称
    fn sanitize_path(path: &str) -> String {
        path.replace('/', "_")
            .replace('-', "_")
            .replace('.', "_")
            .trim_start_matches('_')
            .to_string()
    }
}

// === 健康检查 ===

/// 健康检查状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum HealthStatus {
    Healthy,
    Degraded,
    Unhealthy,
}

/// 健康检查结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthCheckResult {
    pub status: HealthStatus,
    pub message: String,
    pub details: Option<serde_json::Value>,
    pub checked_at: chrono::DateTime<chrono::Utc>,
    pub duration_ms: f64,
}

/// 健康检查器特征
#[async_trait::async_trait]
pub trait HealthChecker: Send + Sync {
    /// 执行健康检查
    async fn check_health(&self) -> HealthCheckResult;
    
    /// 获取检查器名称
    fn name(&self) -> &str;
}

/// 数据库健康检查器
pub struct DatabaseHealthChecker {
    pool: sqlx::PgPool,
}

impl DatabaseHealthChecker {
    pub fn new(pool: sqlx::PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait::async_trait]
impl HealthChecker for DatabaseHealthChecker {
    async fn check_health(&self) -> HealthCheckResult {
        let start = Instant::now();
        let checked_at = chrono::Utc::now();
        
        match sqlx::query("SELECT 1").fetch_one(&self.pool).await {
            Ok(_) => HealthCheckResult {
                status: HealthStatus::Healthy,
                message: "Database connection is healthy".to_string(),
                details: Some(serde_json::json!({
                    "pool_size": self.pool.size(),
                    "idle_connections": self.pool.num_idle(),
                })),
                checked_at,
                duration_ms: start.elapsed().as_secs_f64() * 1000.0,
            },
            Err(e) => HealthCheckResult {
                status: HealthStatus::Unhealthy,
                message: format!("Database connection failed: {}", e),
                details: Some(serde_json::json!({
                    "error": e.to_string(),
                })),
                checked_at,
                duration_ms: start.elapsed().as_secs_f64() * 1000.0,
            },
        }
    }
    
    fn name(&self) -> &str {
        "database"
    }
}

/// 健康检查管理器
pub struct HealthCheckManager {
    checkers: Vec<Box<dyn HealthChecker>>,
}

impl HealthCheckManager {
    /// 创建新的健康检查管理器
    pub fn new() -> Self {
        Self {
            checkers: Vec::new(),
        }
    }
    
    /// 添加健康检查器
    pub fn add_checker(&mut self, checker: Box<dyn HealthChecker>) {
        self.checkers.push(checker);
    }
    
    /// 执行所有健康检查
    pub async fn check_all(&self) -> OverallHealthResult {
        let mut results = HashMap::new();
        let mut overall_status = HealthStatus::Healthy;
        
        for checker in &self.checkers {
            let result = checker.check_health().await;
            
            match result.status {
                HealthStatus::Unhealthy => overall_status = HealthStatus::Unhealthy,
                HealthStatus::Degraded if matches!(overall_status, HealthStatus::Healthy) => {
                    overall_status = HealthStatus::Degraded;
                }
                _ => {}
            }
            
            results.insert(checker.name().to_string(), result);
        }
        
        OverallHealthResult {
            status: overall_status,
            checks: results,
            checked_at: chrono::Utc::now(),
        }
    }
}

impl Default for HealthCheckManager {
    fn default() -> Self {
        Self::new()
    }
}

/// 整体健康检查结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OverallHealthResult {
    pub status: HealthStatus,
    pub checks: HashMap<String, HealthCheckResult>,
    pub checked_at: chrono::DateTime<chrono::Utc>,
}

// === 结构化日志 ===

/// 结构化日志记录器
pub struct StructuredLogger;

impl StructuredLogger {
    /// 记录错误日志
    pub fn log_error(error: &ContextualError) {
        tracing::error!(
            error_id = %error.context.error_id,
            error_type = error.error_type(),
            operation = %error.context.operation,
            user_id = ?error.context.user_id,
            request_id = ?error.context.request_id,
            session_id = ?error.context.session_id,
            error_chain = %error.error_chain(),
            metadata = ?error.context.metadata,
            "Operation failed"
        );
    }
    
    /// 记录性能日志
    pub fn log_performance(operation: &str, duration: Duration, metadata: Option<&HashMap<String, String>>) {
        tracing::info!(
            operation = operation,
            duration_ms = duration.as_secs_f64() * 1000.0,
            metadata = ?metadata,
            "Performance measurement"
        );
    }
    
    /// 记录业务事件日志
    pub fn log_business_event(event_type: &str, user_id: Option<&str>, metadata: &HashMap<String, String>) {
        tracing::info!(
            event_type = event_type,
            user_id = user_id,
            metadata = ?metadata,
            "Business event occurred"
        );
    }
    
    /// 记录安全事件日志
    pub fn log_security_event(event_type: &str, user_id: Option<&str>, ip_address: Option<&str>, details: &str) {
        tracing::warn!(
            event_type = event_type,
            user_id = user_id,
            ip_address = ip_address,
            details = details,
            "Security event detected"
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_metrics_collector() {
        let collector = MetricsCollector::new();
        
        // 测试计数器
        collector.increment_counter("test_counter", 5).await;
        collector.increment_counter("test_counter", 3).await;
        
        // 测试仪表
        collector.set_gauge("test_gauge", 42.5).await;
        
        // 测试直方图
        collector.record_histogram("test_histogram", 1.0).await;
        collector.record_histogram("test_histogram", 2.0).await;
        collector.record_histogram("test_histogram", 3.0).await;
        
        // 测试计时器
        collector.record_timer("test_timer", Duration::from_millis(100)).await;
        collector.record_timer("test_timer", Duration::from_millis(200)).await;
        
        let metrics = collector.get_metrics().await;
        
        assert_eq!(metrics.counters.get("test_counter"), Some(&8));
        assert_eq!(metrics.gauges.get("test_gauge"), Some(&42.5));
        assert_eq!(metrics.histograms.get("test_histogram").unwrap().count, 3);
        assert_eq!(metrics.timers.get("test_timer").unwrap().count, 2);
    }
    
    #[tokio::test]
    async fn test_performance_monitor() {
        let metrics = MetricsCollector::new();
        let monitor = PerformanceMonitor::new(metrics.clone());
        
        // 测试函数计时
        let result = monitor.time_function("test_function", async {
            tokio::time::sleep(Duration::from_millis(10)).await;
            "result"
        }).await;
        
        assert_eq!(result, "result");
        
        let snapshot = metrics.get_metrics().await;
        assert_eq!(snapshot.counters.get("test_function_calls"), Some(&1));
        assert!(snapshot.timers.contains_key("test_function"));
    }
    
    #[test]
    fn test_histogram_stats() {
        let values = vec![1.0, 2.0, 3.0, 4.0, 5.0];
        let stats = HistogramStats::from_values(&values);
        
        assert_eq!(stats.count, 5);
        assert_eq!(stats.sum, 15.0);
        assert_eq!(stats.min, 1.0);
        assert_eq!(stats.max, 5.0);
        assert_eq!(stats.mean, 3.0);
    }
}
