/// 增强的性能监控系统
/// 
/// 提供详细的业务指标监控、性能分析和告警功能
use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant, SystemTime, UNIX_EPOCH};
use serde::{Serialize, Deserialize};
use tracing::{debug, warn, info, error};

/// 性能监控配置
#[derive(Debug, Clone)]
pub struct EnhancedPerformanceConfig {
    /// 是否启用详细监控
    pub enable_detailed_monitoring: bool,
    /// 慢查询阈值（毫秒）
    pub slow_query_threshold_ms: u64,
    /// 慢API阈值（毫秒）
    pub slow_api_threshold_ms: u64,
    /// 内存使用告警阈值（MB）
    pub memory_warning_threshold_mb: u64,
    /// CPU使用告警阈值（百分比）
    pub cpu_warning_threshold_percent: f64,
    /// 错误率告警阈值（百分比）
    pub error_rate_warning_threshold: f64,
    /// 统计窗口大小（秒）
    pub stats_window_seconds: u64,
    /// 保留历史数据时长（小时）
    pub history_retention_hours: u64,
}

impl Default for EnhancedPerformanceConfig {
    fn default() -> Self {
        Self {
            enable_detailed_monitoring: true,
            slow_query_threshold_ms: 100,
            slow_api_threshold_ms: 1000,
            memory_warning_threshold_mb: 1024,
            cpu_warning_threshold_percent: 80.0,
            error_rate_warning_threshold: 5.0,
            stats_window_seconds: 300, // 5分钟窗口
            history_retention_hours: 24,
        }
    }
}

/// API性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApiMetrics {
    pub endpoint: String,
    pub method: String,
    pub total_requests: u64,
    pub successful_requests: u64,
    pub failed_requests: u64,
    pub average_response_time_ms: f64,
    pub min_response_time_ms: u64,
    pub max_response_time_ms: u64,
    pub p95_response_time_ms: f64,
    pub p99_response_time_ms: f64,
    pub requests_per_minute: f64,
    pub error_rate_percent: f64,
    pub last_updated: u64,
}

impl ApiMetrics {
    fn new(endpoint: String, method: String) -> Self {
        Self {
            endpoint,
            method,
            total_requests: 0,
            successful_requests: 0,
            failed_requests: 0,
            average_response_time_ms: 0.0,
            min_response_time_ms: u64::MAX,
            max_response_time_ms: 0,
            p95_response_time_ms: 0.0,
            p99_response_time_ms: 0.0,
            requests_per_minute: 0.0,
            error_rate_percent: 0.0,
            last_updated: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
        }
    }
}

/// 数据库性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseMetrics {
    pub total_queries: u64,
    pub successful_queries: u64,
    pub failed_queries: u64,
    pub slow_queries: u64,
    pub average_query_time_ms: f64,
    pub geo_queries: u64,
    pub cache_hits: u64,
    pub cache_misses: u64,
    pub cache_hit_rate_percent: f64,
    pub connection_pool_active: usize,
    pub connection_pool_idle: usize,
    pub connection_pool_max: usize,
}

/// WebSocket性能指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebSocketMetrics {
    pub active_connections: usize,
    pub peak_connections: usize,
    pub total_connections_created: u64,
    pub total_connections_closed: u64,
    pub messages_sent: u64,
    pub messages_received: u64,
    pub bytes_sent: u64,
    pub bytes_received: u64,
    pub average_connection_duration_seconds: f64,
    pub connection_errors: u64,
    pub message_errors: u64,
}

/// 系统资源指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    pub cpu_usage_percent: f64,
    pub memory_usage_mb: u64,
    pub memory_usage_percent: f64,
    pub disk_usage_percent: f64,
    pub network_bytes_sent: u64,
    pub network_bytes_received: u64,
    pub uptime_seconds: u64,
    pub load_average: f64,
}

/// 业务指标
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessMetrics {
    pub active_users: u64,
    pub new_users_today: u64,
    pub total_messages_sent: u64,
    pub total_groups_created: u64,
    pub total_moments_created: u64,
    pub daily_active_users: u64,
    pub monthly_active_users: u64,
    pub user_retention_rate_percent: f64,
}

/// 性能告警
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceAlert {
    pub alert_type: String,
    pub severity: AlertSeverity,
    pub message: String,
    pub metric_name: String,
    pub current_value: f64,
    pub threshold_value: f64,
    pub timestamp: u64,
    pub resolved: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertSeverity {
    Info,
    Warning,
    Critical,
}

/// 综合性能报告
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnhancedPerformanceReport {
    pub timestamp: u64,
    pub api_metrics: HashMap<String, ApiMetrics>,
    pub database_metrics: DatabaseMetrics,
    pub websocket_metrics: WebSocketMetrics,
    pub system_metrics: SystemMetrics,
    pub business_metrics: BusinessMetrics,
    pub active_alerts: Vec<PerformanceAlert>,
    pub performance_score: f64, // 0-100的综合性能评分
    pub recommendations: Vec<String>,
}

/// 性能数据点
#[derive(Debug, Clone)]
struct PerformanceDataPoint {
    timestamp: Instant,
    response_time_ms: u64,
    success: bool,
}

/// 增强的性能监控器
pub struct EnhancedPerformanceMonitor {
    config: EnhancedPerformanceConfig,
    api_metrics: Arc<Mutex<HashMap<String, ApiMetrics>>>,
    api_data_points: Arc<Mutex<HashMap<String, VecDeque<PerformanceDataPoint>>>>,
    database_metrics: Arc<Mutex<DatabaseMetrics>>,
    websocket_metrics: Arc<Mutex<WebSocketMetrics>>,
    system_metrics: Arc<Mutex<SystemMetrics>>,
    business_metrics: Arc<Mutex<BusinessMetrics>>,
    alerts: Arc<Mutex<Vec<PerformanceAlert>>>,
    start_time: Instant,
}

impl EnhancedPerformanceMonitor {
    /// 创建新的增强性能监控器
    pub fn new(config: EnhancedPerformanceConfig) -> Self {
        let monitor = Self {
            config,
            api_metrics: Arc::new(Mutex::new(HashMap::new())),
            api_data_points: Arc::new(Mutex::new(HashMap::new())),
            database_metrics: Arc::new(Mutex::new(DatabaseMetrics {
                total_queries: 0,
                successful_queries: 0,
                failed_queries: 0,
                slow_queries: 0,
                average_query_time_ms: 0.0,
                geo_queries: 0,
                cache_hits: 0,
                cache_misses: 0,
                cache_hit_rate_percent: 0.0,
                connection_pool_active: 0,
                connection_pool_idle: 0,
                connection_pool_max: 0,
            })),
            websocket_metrics: Arc::new(Mutex::new(WebSocketMetrics {
                active_connections: 0,
                peak_connections: 0,
                total_connections_created: 0,
                total_connections_closed: 0,
                messages_sent: 0,
                messages_received: 0,
                bytes_sent: 0,
                bytes_received: 0,
                average_connection_duration_seconds: 0.0,
                connection_errors: 0,
                message_errors: 0,
            })),
            system_metrics: Arc::new(Mutex::new(SystemMetrics {
                cpu_usage_percent: 0.0,
                memory_usage_mb: 0,
                memory_usage_percent: 0.0,
                disk_usage_percent: 0.0,
                network_bytes_sent: 0,
                network_bytes_received: 0,
                uptime_seconds: 0,
                load_average: 0.0,
            })),
            business_metrics: Arc::new(Mutex::new(BusinessMetrics {
                active_users: 0,
                new_users_today: 0,
                total_messages_sent: 0,
                total_groups_created: 0,
                total_moments_created: 0,
                daily_active_users: 0,
                monthly_active_users: 0,
                user_retention_rate_percent: 0.0,
            })),
            alerts: Arc::new(Mutex::new(Vec::new())),
            start_time: Instant::now(),
        };

        // 启动系统指标收集任务
        monitor.start_system_metrics_collection();
        
        // 启动告警检查任务
        monitor.start_alert_monitoring();
        
        monitor
    }

    /// 记录API请求性能
    pub fn record_api_request(&self, endpoint: &str, method: &str, response_time_ms: u64, success: bool) {
        if !self.config.enable_detailed_monitoring {
            return;
        }

        let key = format!("{}:{}", method, endpoint);
        let now = Instant::now();
        
        // 更新API指标
        {
            let mut api_metrics = self.api_metrics.lock().unwrap();
            let metrics = api_metrics.entry(key.clone())
                .or_insert_with(|| ApiMetrics::new(endpoint.to_string(), method.to_string()));
            
            metrics.total_requests += 1;
            if success {
                metrics.successful_requests += 1;
            } else {
                metrics.failed_requests += 1;
            }
            
            // 更新响应时间统计
            if response_time_ms < metrics.min_response_time_ms {
                metrics.min_response_time_ms = response_time_ms;
            }
            if response_time_ms > metrics.max_response_time_ms {
                metrics.max_response_time_ms = response_time_ms;
            }
            
            metrics.error_rate_percent = (metrics.failed_requests as f64 / metrics.total_requests as f64) * 100.0;
            metrics.last_updated = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
        }
        
        // 添加数据点
        {
            let mut data_points = self.api_data_points.lock().unwrap();
            let points = data_points.entry(key.clone()).or_insert_with(VecDeque::new);
            
            points.push_back(PerformanceDataPoint {
                timestamp: now,
                response_time_ms,
                success,
            });
            
            // 清理过期数据点
            let window_duration = Duration::from_secs(self.config.stats_window_seconds);
            while let Some(front) = points.front() {
                if now.duration_since(front.timestamp) > window_duration {
                    points.pop_front();
                } else {
                    break;
                }
            }
        }
        
        // 检查慢API告警
        if response_time_ms > self.config.slow_api_threshold_ms {
            self.create_alert(
                "slow_api".to_string(),
                AlertSeverity::Warning,
                format!("API {} 响应时间过慢: {}ms", endpoint, response_time_ms),
                "api_response_time".to_string(),
                response_time_ms as f64,
                self.config.slow_api_threshold_ms as f64,
            );
        }
        
        // 更新计算指标
        self.update_calculated_api_metrics(&key);
        
        debug!("记录API性能: {} {} {}ms {}", method, endpoint, response_time_ms, if success { "成功" } else { "失败" });
    }

    /// 记录数据库查询性能
    pub fn record_database_query(&self, query_type: &str, duration_ms: u64, success: bool, is_geo_query: bool) {
        let mut db_metrics = self.database_metrics.lock().unwrap();
        
        db_metrics.total_queries += 1;
        if success {
            db_metrics.successful_queries += 1;
        } else {
            db_metrics.failed_queries += 1;
        }
        
        if is_geo_query {
            db_metrics.geo_queries += 1;
        }
        
        if duration_ms > self.config.slow_query_threshold_ms {
            db_metrics.slow_queries += 1;
            
            self.create_alert(
                "slow_query".to_string(),
                AlertSeverity::Warning,
                format!("数据库查询过慢: {} {}ms", query_type, duration_ms),
                "query_response_time".to_string(),
                duration_ms as f64,
                self.config.slow_query_threshold_ms as f64,
            );
        }
        
        // 更新平均查询时间
        let total_time = db_metrics.average_query_time_ms * (db_metrics.total_queries - 1) as f64 + duration_ms as f64;
        db_metrics.average_query_time_ms = total_time / db_metrics.total_queries as f64;
        
        debug!("记录数据库查询: {} {}ms {}", query_type, duration_ms, if success { "成功" } else { "失败" });
    }

    /// 记录缓存性能
    pub fn record_cache_operation(&self, hit: bool) {
        let mut db_metrics = self.database_metrics.lock().unwrap();
        
        if hit {
            db_metrics.cache_hits += 1;
        } else {
            db_metrics.cache_misses += 1;
        }
        
        let total_cache_ops = db_metrics.cache_hits + db_metrics.cache_misses;
        if total_cache_ops > 0 {
            db_metrics.cache_hit_rate_percent = (db_metrics.cache_hits as f64 / total_cache_ops as f64) * 100.0;
        }
    }

    /// 更新WebSocket指标
    pub fn update_websocket_metrics(&self, 
        active_connections: usize,
        messages_sent: u64,
        messages_received: u64,
        bytes_sent: u64,
        bytes_received: u64) {
        
        let mut ws_metrics = self.websocket_metrics.lock().unwrap();
        
        ws_metrics.active_connections = active_connections;
        if active_connections > ws_metrics.peak_connections {
            ws_metrics.peak_connections = active_connections;
        }
        
        ws_metrics.messages_sent = messages_sent;
        ws_metrics.messages_received = messages_received;
        ws_metrics.bytes_sent = bytes_sent;
        ws_metrics.bytes_received = bytes_received;
    }

    /// 更新业务指标
    pub fn update_business_metrics(&self, 
        active_users: u64,
        new_users_today: u64,
        total_messages: u64,
        total_groups: u64,
        total_moments: u64) {
        
        let mut business_metrics = self.business_metrics.lock().unwrap();
        
        business_metrics.active_users = active_users;
        business_metrics.new_users_today = new_users_today;
        business_metrics.total_messages_sent = total_messages;
        business_metrics.total_groups_created = total_groups;
        business_metrics.total_moments_created = total_moments;
    }

    /// 获取综合性能报告
    pub fn get_performance_report(&self) -> EnhancedPerformanceReport {
        let api_metrics = self.api_metrics.lock().unwrap().clone();
        let database_metrics = self.database_metrics.lock().unwrap().clone();
        let websocket_metrics = self.websocket_metrics.lock().unwrap().clone();
        let system_metrics = self.system_metrics.lock().unwrap().clone();
        let business_metrics = self.business_metrics.lock().unwrap().clone();
        let active_alerts = self.alerts.lock().unwrap()
            .iter()
            .filter(|alert| !alert.resolved)
            .cloned()
            .collect();
        
        let performance_score = self.calculate_performance_score(&api_metrics, &database_metrics, &websocket_metrics, &system_metrics);
        let recommendations = self.generate_recommendations(&api_metrics, &database_metrics, &system_metrics);
        
        EnhancedPerformanceReport {
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            api_metrics,
            database_metrics,
            websocket_metrics,
            system_metrics,
            business_metrics,
            active_alerts,
            performance_score,
            recommendations,
        }
    }

    /// 计算性能评分
    fn calculate_performance_score(&self, 
        api_metrics: &HashMap<String, ApiMetrics>,
        db_metrics: &DatabaseMetrics,
        ws_metrics: &WebSocketMetrics,
        sys_metrics: &SystemMetrics) -> f64 {
        
        let mut score = 100.0;
        
        // API性能评分（权重30%）
        let avg_api_response_time = api_metrics.values()
            .map(|m| m.average_response_time_ms)
            .fold(0.0, |acc, x| acc + x) / api_metrics.len().max(1) as f64;
        
        if avg_api_response_time > 1000.0 {
            score -= 20.0;
        } else if avg_api_response_time > 500.0 {
            score -= 10.0;
        }
        
        // 数据库性能评分（权重25%）
        if db_metrics.average_query_time_ms > 100.0 {
            score -= 15.0;
        }
        if db_metrics.cache_hit_rate_percent < 80.0 {
            score -= 10.0;
        }
        
        // 系统资源评分（权重25%）
        if sys_metrics.cpu_usage_percent > 80.0 {
            score -= 15.0;
        } else if sys_metrics.cpu_usage_percent > 60.0 {
            score -= 8.0;
        }
        
        if sys_metrics.memory_usage_percent > 85.0 {
            score -= 10.0;
        } else if sys_metrics.memory_usage_percent > 70.0 {
            score -= 5.0;
        }
        
        // WebSocket性能评分（权重20%）
        if ws_metrics.connection_errors > 0 {
            score -= 10.0;
        }
        if ws_metrics.message_errors > 0 {
            score -= 5.0;
        }
        
        score.max(0.0)
    }

    /// 生成性能优化建议
    fn generate_recommendations(&self, 
        api_metrics: &HashMap<String, ApiMetrics>,
        db_metrics: &DatabaseMetrics,
        sys_metrics: &SystemMetrics) -> Vec<String> {
        
        let mut recommendations = Vec::new();
        
        // API优化建议
        for (endpoint, metrics) in api_metrics {
            if metrics.average_response_time_ms > 1000.0 {
                recommendations.push(format!("优化API {} 的响应时间，当前平均 {:.2}ms", endpoint, metrics.average_response_time_ms));
            }
            if metrics.error_rate_percent > 5.0 {
                recommendations.push(format!("降低API {} 的错误率，当前 {:.2}%", endpoint, metrics.error_rate_percent));
            }
        }
        
        // 数据库优化建议
        if db_metrics.average_query_time_ms > 100.0 {
            recommendations.push("考虑优化数据库查询，添加索引或重构慢查询".to_string());
        }
        if db_metrics.cache_hit_rate_percent < 80.0 {
            recommendations.push("提高缓存命中率，考虑调整缓存策略".to_string());
        }
        if db_metrics.slow_queries > 10 {
            recommendations.push("存在较多慢查询，建议进行查询优化".to_string());
        }
        
        // 系统资源建议
        if sys_metrics.cpu_usage_percent > 80.0 {
            recommendations.push("CPU使用率过高，考虑扩容或优化计算密集型操作".to_string());
        }
        if sys_metrics.memory_usage_percent > 85.0 {
            recommendations.push("内存使用率过高，检查内存泄漏或考虑增加内存".to_string());
        }
        
        recommendations
    }

    /// 更新计算的API指标
    fn update_calculated_api_metrics(&self, key: &str) {
        let data_points = self.api_data_points.lock().unwrap();
        if let Some(points) = data_points.get(key) {
            if points.is_empty() {
                return;
            }
            
            let mut api_metrics = self.api_metrics.lock().unwrap();
            if let Some(metrics) = api_metrics.get_mut(key) {
                // 计算平均响应时间
                let total_time: u64 = points.iter().map(|p| p.response_time_ms).sum();
                metrics.average_response_time_ms = total_time as f64 / points.len() as f64;
                
                // 计算百分位数
                let mut response_times: Vec<u64> = points.iter().map(|p| p.response_time_ms).collect();
                response_times.sort_unstable();
                
                let len = response_times.len();
                if len > 0 {
                    let p95_index = (len as f64 * 0.95) as usize;
                    let p99_index = (len as f64 * 0.99) as usize;
                    
                    metrics.p95_response_time_ms = response_times[p95_index.min(len - 1)] as f64;
                    metrics.p99_response_time_ms = response_times[p99_index.min(len - 1)] as f64;
                }
                
                // 计算每分钟请求数
                let window_duration = Duration::from_secs(self.config.stats_window_seconds);
                let requests_in_window = points.len() as f64;
                metrics.requests_per_minute = (requests_in_window / window_duration.as_secs() as f64) * 60.0;
            }
        }
    }

    /// 创建告警
    fn create_alert(&self, alert_type: String, severity: AlertSeverity, message: String, 
                   metric_name: String, current_value: f64, threshold_value: f64) {
        let alert = PerformanceAlert {
            alert_type,
            severity,
            message: message.clone(),
            metric_name,
            current_value,
            threshold_value,
            timestamp: SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs(),
            resolved: false,
        };
        
        let mut alerts = self.alerts.lock().unwrap();
        alerts.push(alert);
        
        // 保持最近100个告警
        if alerts.len() > 100 {
            alerts.remove(0);
        }
        
        match severity {
            AlertSeverity::Critical => error!("性能告警[严重]: {}", message),
            AlertSeverity::Warning => warn!("性能告警[警告]: {}", message),
            AlertSeverity::Info => info!("性能告警[信息]: {}", message),
        }
    }

    /// 启动系统指标收集
    fn start_system_metrics_collection(&self) {
        let system_metrics = self.system_metrics.clone();
        let start_time = self.start_time;
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(30));
            
            loop {
                interval.tick().await;
                
                // 这里应该实现真实的系统指标收集
                // 为了演示，使用模拟数据
                let mut metrics = system_metrics.lock().unwrap();
                metrics.uptime_seconds = start_time.elapsed().as_secs();
                // 实际实现中应该使用系统API获取真实指标
                metrics.cpu_usage_percent = 25.0; // 模拟值
                metrics.memory_usage_mb = 512; // 模拟值
                metrics.memory_usage_percent = 50.0; // 模拟值
            }
        });
    }

    /// 启动告警监控
    fn start_alert_monitoring(&self) {
        let system_metrics = self.system_metrics.clone();
        let config = self.config.clone();
        let monitor = Arc::new(self);
        
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(60));
            
            loop {
                interval.tick().await;
                
                let metrics = system_metrics.lock().unwrap();
                
                // 检查CPU告警
                if metrics.cpu_usage_percent > config.cpu_warning_threshold_percent {
                    monitor.create_alert(
                        "high_cpu".to_string(),
                        AlertSeverity::Warning,
                        format!("CPU使用率过高: {:.2}%", metrics.cpu_usage_percent),
                        "cpu_usage".to_string(),
                        metrics.cpu_usage_percent,
                        config.cpu_warning_threshold_percent,
                    );
                }
                
                // 检查内存告警
                if metrics.memory_usage_mb > config.memory_warning_threshold_mb {
                    monitor.create_alert(
                        "high_memory".to_string(),
                        AlertSeverity::Warning,
                        format!("内存使用过高: {}MB", metrics.memory_usage_mb),
                        "memory_usage".to_string(),
                        metrics.memory_usage_mb as f64,
                        config.memory_warning_threshold_mb as f64,
                    );
                }
            }
        });
    }
}

/// 全局增强性能监控器实例
static mut GLOBAL_ENHANCED_PERFORMANCE_MONITOR: Option<Arc<EnhancedPerformanceMonitor>> = None;
static INIT: std::sync::Once = std::sync::Once::new();

/// 获取全局增强性能监控器
pub fn get_global_enhanced_performance_monitor() -> Arc<EnhancedPerformanceMonitor> {
    unsafe {
        INIT.call_once(|| {
            let config = EnhancedPerformanceConfig::default();
            GLOBAL_ENHANCED_PERFORMANCE_MONITOR = Some(Arc::new(EnhancedPerformanceMonitor::new(config)));
        });
        GLOBAL_ENHANCED_PERFORMANCE_MONITOR.as_ref().unwrap().clone()
    }
}

/// 初始化增强性能监控器
pub fn init_enhanced_performance_monitor(config: EnhancedPerformanceConfig) {
    unsafe {
        GLOBAL_ENHANCED_PERFORMANCE_MONITOR = Some(Arc::new(EnhancedPerformanceMonitor::new(config)));
    }
    info!("增强性能监控器初始化完成");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_api_metrics_creation() {
        let metrics = ApiMetrics::new("/api/test".to_string(), "GET".to_string());
        assert_eq!(metrics.endpoint, "/api/test");
        assert_eq!(metrics.method, "GET");
        assert_eq!(metrics.total_requests, 0);
        assert_eq!(metrics.min_response_time_ms, u64::MAX);
    }

    #[test]
    fn test_performance_monitor_creation() {
        let config = EnhancedPerformanceConfig::default();
        let monitor = EnhancedPerformanceMonitor::new(config);
        
        // 测试记录API请求
        monitor.record_api_request("/api/test", "GET", 150, true);
        monitor.record_api_request("/api/test", "GET", 200, false);
        
        let report = monitor.get_performance_report();
        assert!(report.api_metrics.contains_key("GET:/api/test"));
        
        let api_metric = &report.api_metrics["GET:/api/test"];
        assert_eq!(api_metric.total_requests, 2);
        assert_eq!(api_metric.successful_requests, 1);
        assert_eq!(api_metric.failed_requests, 1);
    }

    #[test]
    fn test_database_metrics() {
        let config = EnhancedPerformanceConfig::default();
        let monitor = EnhancedPerformanceMonitor::new(config);
        
        monitor.record_database_query("SELECT", 50, true, false);
        monitor.record_database_query("INSERT", 120, true, false);
        monitor.record_database_query("GEO_QUERY", 80, true, true);
        
        let report = monitor.get_performance_report();
        assert_eq!(report.database_metrics.total_queries, 3);
        assert_eq!(report.database_metrics.successful_queries, 3);
        assert_eq!(report.database_metrics.geo_queries, 1);
        assert!(report.database_metrics.average_query_time_ms > 0.0);
    }

    #[test]
    fn test_cache_metrics() {
        let config = EnhancedPerformanceConfig::default();
        let monitor = EnhancedPerformanceMonitor::new(config);
        
        monitor.record_cache_operation(true);  // hit
        monitor.record_cache_operation(true);  // hit
        monitor.record_cache_operation(false); // miss
        
        let report = monitor.get_performance_report();
        assert_eq!(report.database_metrics.cache_hits, 2);
        assert_eq!(report.database_metrics.cache_misses, 1);
        assert!((report.database_metrics.cache_hit_rate_percent - 66.67).abs() < 0.1);
    }

    #[test]
    fn test_performance_score_calculation() {
        let config = EnhancedPerformanceConfig::default();
        let monitor = EnhancedPerformanceMonitor::new(config);
        
        // 记录一些性能数据
        monitor.record_api_request("/api/fast", "GET", 50, true);
        monitor.record_database_query("FAST_QUERY", 30, true, false);
        
        let report = monitor.get_performance_report();
        assert!(report.performance_score > 80.0); // 应该有较高的性能评分
    }
}
