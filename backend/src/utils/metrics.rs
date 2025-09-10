//! 应用监控指标模块

use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH, Instant};
use tracing::{info, warn, error};
use prometheus::{
    Encoder, TextEncoder, 
    Gauge, 
    Counter, CounterVec, 
    Histogram, HistogramVec,
    opts, register_counter_vec, register_histogram_vec,
    register_counter, register_gauge, register_histogram,
};
use serde::{Serialize, Deserialize};
use tokio::time::{interval, Duration};

use crate::bootstrap::config_adapter::Database;

/// 应用监控指标
#[derive(Debug, Clone)]
pub struct AppMetrics {
    // API请求计数
    pub http_requests_total: CounterVec,
    // API请求延迟
    pub http_request_duration_seconds: HistogramVec,
    // 活跃用户数
    pub active_users: Gauge,
    // 数据库连接数
    pub db_connections: Gauge,
    // 数据库连接池容量
    pub db_pool_capacity: Gauge,
    // Redis连接状态
    pub redis_connected: Gauge,
    // WebSocket连接数
    pub ws_connections: Gauge,
    // WebSocket消息发送计数
    pub ws_messages_sent: Counter,
    // WebSocket消息接收计数
    pub ws_messages_received: Counter,
    // WebSocket订阅计数
    pub ws_subscriptions_total: CounterVec,
    // 离线通知计数
    #[allow(dead_code)]
    pub ws_offline_notifications_total: CounterVec,
    // 连接清理移除计数
    #[allow(dead_code)]
    pub ws_cleanup_removed_total: Counter,
    // 消息队列任务计数
    #[allow(dead_code)]
    pub job_queue_tasks: CounterVec,
    // 位置查询计数
    pub location_query_count: Counter,
    // 位置查询延迟
    pub location_query_duration_seconds: Histogram,
    // 缓存命中率
    #[allow(dead_code)]
    pub cache_hit_ratio: Gauge,
    // 系统内存使用
    pub memory_usage_bytes: Gauge,
    // 系统CPU使用
    pub cpu_usage_percent: Gauge,
    // 应用运行时间
    pub uptime: Gauge,
    // JVM运行时间
    pub uptime_seconds: Gauge,
    
    // 业务指标
    // 用户注册数
    pub user_registrations_total: Counter,
    // 临时用户创建数
    pub temp_users_created_total: Counter,
    // 群组创建数
    pub groups_created_total: Counter,
    // 消息发送数
    pub messages_sent_total: CounterVec,
    // 生活瞬间创建数
    pub moments_created_total: Counter,
    // 生活瞬间点赞数
    pub moment_likes_total: Counter,
    // 用户登录数
    pub user_logins_total: CounterVec,
    // API错误率
    pub api_errors_total: CounterVec,
    // 数据库查询延迟
    pub db_query_duration_seconds: HistogramVec,
    // 地理位置查询数
    pub geo_queries_total: CounterVec,
    // 缓存操作计数
    pub cache_operations_total: CounterVec,
    // 任务队列指标
    pub job_queue_size: Gauge,
    pub job_processing_duration_seconds: HistogramVec,
}

impl AppMetrics {
    /// 创建应用监控指标
    pub fn new() -> Result<Self, prometheus::Error> {
        // API请求计数
        let http_requests_total = register_counter_vec!(
            opts!("http_requests_total", "API请求总数"),
            &["method", "path", "status"]
        )?;
        
        // API请求延迟
        let http_request_duration_seconds = register_histogram_vec!(
            "http_request_duration_seconds",
            "API请求延迟（秒）",
            &["method", "path"],
            vec![0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
        )?;
        
        // 活跃用户数
        let active_users = register_gauge!(
            opts!("active_users", "当前活跃用户数")
        )?;
        
        // 数据库连接数
        let db_connections = register_gauge!(
            opts!("db_connections", "当前数据库连接数")
        )?;
        
        // 数据库连接池容量
        let db_pool_capacity = register_gauge!(
            opts!("db_pool_capacity", "数据库连接池容量")
        )?;
        
        // Redis连接状态
        let redis_connected = register_gauge!(
            opts!("redis_connected", "Redis连接状态 (1=已连接, 0=未连接)")
        )?;
        
        // WebSocket连接数
        let ws_connections = register_gauge!(
            opts!("ws_connections", "当前WebSocket连接数")
        )?;
        
        // WebSocket消息发送计数
        let ws_messages_sent = register_counter!(
            opts!("ws_messages_sent", "WebSocket消息发送总数")
        )?;
        
        // WebSocket消息接收计数
        let ws_messages_received = register_counter!(
            opts!("ws_messages_received", "WebSocket消息接收总数")
        )?;
        // WebSocket订阅统计
        let ws_subscriptions_total = register_counter_vec!(
            opts!("ws_subscriptions_total", "WebSocket订阅事件计数"),
            &["type", "result"]
        )?;
        // 离线通知统计
        let ws_offline_notifications_total = register_counter_vec!(
            opts!("ws_offline_notifications_total", "WebSocket离线通知计数"),
            &["result"]
        )?;
        // 连接清理移除计数
        let ws_cleanup_removed_total = register_counter!(
            opts!("ws_cleanup_removed_total", "WebSocket清理移除连接计数")
        )?;
        
        // 消息队列任务计数
        let job_queue_tasks = register_counter_vec!(
            opts!("job_queue_tasks", "消息队列任务计数"),
            &["type", "status"]
        )?;
        
        // 位置查询计数
        let location_query_count = register_counter!(
            opts!("location_query_count", "位置查询总数")
        )?;
        
        // 位置查询延迟
        let location_query_duration_seconds = register_histogram!(
            "location_query_duration_seconds",
            "位置查询延迟（秒）",
            vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5]
        )?;
        
        // 缓存命中率
        let cache_hit_ratio = register_gauge!(
            opts!("cache_hit_ratio", "缓存命中率")
        )?;
        
        // 系统内存使用
        let memory_usage_bytes = register_gauge!(
            opts!("memory_usage_bytes", "系统内存使用（字节）")
        )?;
        
        // 系统CPU使用
        let cpu_usage_percent = register_gauge!(
            opts!("cpu_usage_percent", "系统CPU使用率（百分比）")
        )?;
        
        // 应用运行时间
        let uptime = register_gauge!(
            opts!("app_uptime", "应用运行时间（秒）")
        )?;
        
        // JVM运行时间
        let uptime_seconds = register_gauge!(
            opts!("uptime_seconds", "应用运行时间（秒）")
        )?;
        
        // 业务指标
        let user_registrations_total = register_counter!(
            opts!("user_registrations_total", "用户注册总数")
        )?;
        
        let temp_users_created_total = register_counter!(
            opts!("temp_users_created_total", "临时用户创建总数")
        )?;
        
        let groups_created_total = register_counter!(
            opts!("groups_created_total", "群组创建总数")
        )?;
        
        let messages_sent_total = register_counter_vec!(
            opts!("messages_sent_total", "消息发送总数"),
            &["message_type", "channel"]
        )?;
        
        let moments_created_total = register_counter!(
            opts!("moments_created_total", "生活瞬间创建总数")
        )?;
        
        let moment_likes_total = register_counter!(
            opts!("moment_likes_total", "生活瞬间点赞总数")
        )?;
        
        let user_logins_total = register_counter_vec!(
            opts!("user_logins_total", "用户登录总数"),
            &["user_type", "result"]
        )?;
        
        let api_errors_total = register_counter_vec!(
            opts!("api_errors_total", "API错误总数"),
            &["method", "path", "error_type"]
        )?;
        
        let db_query_duration_seconds = register_histogram_vec!(
            "db_query_duration_seconds",
            "数据库查询延迟（秒）",
            &["operation", "table"],
            vec![0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
        )?;
        
        let geo_queries_total = register_counter_vec!(
            opts!("geo_queries_total", "地理位置查询总数"),
            &["query_type", "result"]
        )?;
        
        let cache_operations_total = register_counter_vec!(
            opts!("cache_operations_total", "缓存操作总数"),
            &["operation", "result"]
        )?;
        
        let job_queue_size = register_gauge!(
            opts!("job_queue_size", "任务队列大小")
        )?;
        
        let job_processing_duration_seconds = register_histogram_vec!(
            "job_processing_duration_seconds",
            "任务处理延迟（秒）",
            &["job_type", "result"],
            vec![0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 300.0]
        )?;
        
        Ok(Self {
            http_requests_total,
            http_request_duration_seconds,
            active_users,
            db_connections,
            db_pool_capacity,
            redis_connected,
            ws_connections,
            ws_messages_sent,
            ws_messages_received,
            ws_subscriptions_total,
            ws_offline_notifications_total,
            ws_cleanup_removed_total,
            job_queue_tasks,
            location_query_count,
            location_query_duration_seconds,
            cache_hit_ratio,
            memory_usage_bytes,
            cpu_usage_percent,
            uptime,
            uptime_seconds,
            // 业务指标
            user_registrations_total,
            temp_users_created_total,
            groups_created_total,
            messages_sent_total,
            moments_created_total,
            moment_likes_total,
            user_logins_total,
            api_errors_total,
            db_query_duration_seconds,
            geo_queries_total,
            cache_operations_total,
            job_queue_size,
            job_processing_duration_seconds,
        })
    }
    
    /// 导出Prometheus指标
    pub fn export_metrics(&self) -> String {
        let encoder = TextEncoder::new();
        let metric_families = prometheus::gather();
        let mut buffer = Vec::new();
        encoder.encode(&metric_families, &mut buffer).unwrap();
        String::from_utf8(buffer).unwrap()
    }
    
    /// 更新数据库指标
    pub fn update_db_metrics(&self, db: &Database) {
        let metrics = db.get_db_metrics();
        self.db_connections.set(metrics.pg_pool_size as f64);
        self.db_pool_capacity.set(metrics.pg_pool_max_size as f64);
        self.redis_connected.set(if metrics.redis_available { 1.0 } else { 0.0 });
    }
    
    /// 更新WebSocket指标
    #[allow(dead_code)]
    pub fn update_ws_metrics(&self, sessions: usize) {
        self.ws_connections.set(sessions as f64);
    }
    
    /// 更新系统指标
    pub fn update_system_metrics(&self) {
        use sysinfo::{System, RefreshKind, CpuRefreshKind, MemoryRefreshKind};
        
        // 创建系统信息实例，指定需要刷新的组件
        let mut sys = System::new_with_specifics(
            RefreshKind::new()
                .with_cpu(CpuRefreshKind::everything())
                .with_memory(MemoryRefreshKind::everything())
        );
        
        // 刷新系统信息
        sys.refresh_specifics(
            RefreshKind::new()
                .with_cpu(CpuRefreshKind::everything())
                .with_memory(MemoryRefreshKind::everything())
        );
        
        // 获取总内存和已使用内存
        let total_memory = sys.total_memory();
        let used_memory = sys.used_memory();
        
        // 设置内存使用量（字节）
        self.memory_usage_bytes.set(used_memory as f64);
        
        // 获取CPU使用率（平均值）
        let cpu_usage = if sys.cpus().is_empty() {
            0.0
        } else {
            let total_usage: f32 = sys.cpus().iter().map(|cpu| cpu.cpu_usage()).sum();
            (total_usage / sys.cpus().len() as f32) as f64
        };
        
        // 设置CPU使用率
        self.cpu_usage_percent.set(cpu_usage);
        
        // 记录调试信息
        tracing::debug!(
            "系统指标更新: 内存使用 {:.2} MB / {:.2} MB ({:.1}%), CPU使用率 {:.1}%",
            used_memory as f64 / 1024.0 / 1024.0,
            total_memory as f64 / 1024.0 / 1024.0,
            (used_memory as f64 / total_memory as f64) * 100.0,
            cpu_usage
        );
    }
    
    /// 记录HTTP请求
    pub fn record_http_request(&self, method: &str, path: &str, status: u16, duration: f64) {
        self.http_requests_total
            .with_label_values(&[method, path, &status.to_string()])
            .inc();
            
        self.http_request_duration_seconds
            .with_label_values(&[method, path])
            .observe(duration);
    }
    
    /// 记录WebSocket消息
    pub fn record_ws_message_sent(&self) {
        self.ws_messages_sent.inc();
    }
    
    pub fn record_ws_message_received(&self) {
        self.ws_messages_received.inc();
    }

    pub fn record_ws_subscription(&self, room_type: &str, result: &str) {
        self.ws_subscriptions_total
            .with_label_values(&[room_type, result])
            .inc();
    }

    #[allow(dead_code)]
    pub fn record_offline_notification(&self, result: &str) {
        self.ws_offline_notifications_total
            .with_label_values(&[result])
            .inc();
    }

    #[allow(dead_code)]
    pub fn record_ws_cleanup_removed(&self, removed_count: usize) {
        if removed_count > 0 {
            self.ws_cleanup_removed_total.inc_by(removed_count as f64);
        }
    }
    
    /// 记录任务队列任务
    #[allow(dead_code)]
    pub fn record_job_queue_task(&self, task_type: &str, status: &str) {
        self.job_queue_tasks
            .with_label_values(&[task_type, status])
            .inc();
    }
    
    /// 记录位置查询
    pub fn record_location_query(&self, duration: f64) {
        self.location_query_count.inc();
        self.location_query_duration_seconds.observe(duration);
    }
    
    /// 设置应用运行时间
    pub fn set_uptime(&self, seconds: f64) {
        self.uptime_seconds.set(seconds);
    }
    
    /// 设置活跃用户数
    #[allow(dead_code)]
    pub fn set_active_users(&self, count: i64) {
        self.active_users.set(count as f64);
    }
    
    /// 设置缓存命中率
    #[allow(dead_code)]
    pub fn set_cache_hit_ratio(&self, ratio: f64) {
        self.cache_hit_ratio.set(ratio);
    }
    
    // 业务指标记录方法
    
    /// 记录用户注册
    pub fn record_user_registration(&self) {
        self.user_registrations_total.inc();
    }
    
    /// 记录临时用户创建
    pub fn record_temp_user_creation(&self) {
        self.temp_users_created_total.inc();
    }
    
    /// 记录群组创建
    pub fn record_group_creation(&self) {
        self.groups_created_total.inc();
    }
    
    /// 记录消息发送
    pub fn record_message_sent(&self, message_type: &str, channel: &str) {
        self.messages_sent_total
            .with_label_values(&[message_type, channel])
            .inc();
    }
    
    /// 记录生活瞬间创建
    pub fn record_moment_creation(&self) {
        self.moments_created_total.inc();
    }
    
    /// 记录生活瞬间点赞
    pub fn record_moment_like(&self) {
        self.moment_likes_total.inc();
    }
    
    /// 记录用户登录
    pub fn record_user_login(&self, user_type: &str, success: bool) {
        let result = if success { "success" } else { "failed" };
        self.user_logins_total
            .with_label_values(&[user_type, result])
            .inc();
    }
    
    /// 记录API错误
    pub fn record_api_error(&self, method: &str, path: &str, error_type: &str) {
        self.api_errors_total
            .with_label_values(&[method, path, error_type])
            .inc();
    }
    
    /// 记录数据库查询
    pub fn record_db_query(&self, operation: &str, table: &str, duration: f64) {
        self.db_query_duration_seconds
            .with_label_values(&[operation, table])
            .observe(duration);
    }
    
    /// 记录地理位置查询
    pub fn record_geo_query(&self, query_type: &str, success: bool) {
        let result = if success { "success" } else { "failed" };
        self.geo_queries_total
            .with_label_values(&[query_type, result])
            .inc();
    }
    
    /// 记录缓存操作
    pub fn record_cache_operation(&self, operation: &str, success: bool) {
        let result = if success { "hit" } else { "miss" };
        self.cache_operations_total
            .with_label_values(&[operation, result])
            .inc();
    }
    
    /// 设置任务队列大小
    pub fn set_job_queue_size(&self, size: u64) {
        self.job_queue_size.set(size as f64);
    }
    
    /// 记录任务处理
    pub fn record_job_processing(&self, job_type: &str, duration: f64, success: bool) {
        let result = if success { "success" } else { "failed" };
        self.job_processing_duration_seconds
            .with_label_values(&[job_type, result])
            .observe(duration);
    }
}

/// 指标服务
pub struct MetricsService {
    pub metrics: AppMetrics,
    db: Arc<Database>,
    start_time: std::time::Instant,
}

impl MetricsService {
    /// 创建指标服务
    #[allow(dead_code)]
    pub fn new(db: Arc<Database>) -> Result<Self, prometheus::Error> {
        let metrics = AppMetrics::new()?;
        let start_time = std::time::Instant::now();
        
        info!("指标服务初始化成功");
        
        Ok(Self {
            metrics,
            db,
            start_time,
        })
    }
    
    /// 导出Prometheus指标
    #[allow(dead_code)]
    pub fn export_metrics(&self) -> String {
        self.metrics.export_metrics()
    }
    
    /// 更新所有指标
    #[allow(dead_code)]
    pub fn update_all_metrics(&self) {
        // 更新数据库指标
        self.metrics.update_db_metrics(&self.db);
        
        // 更新系统指标
        self.metrics.update_system_metrics();
        
        // 更新运行时间
        let uptime = self.start_time.elapsed().as_secs_f64();
        self.metrics.set_uptime(uptime);
    }
}

/// 创建指标中间件
#[allow(dead_code)]
/// 全局应用指标实例
use std::sync::OnceLock;
static GLOBAL_APP_METRICS: OnceLock<AppMetrics> = OnceLock::new();

/// 获取全局应用指标实例
pub fn get_app_metrics() -> &'static AppMetrics {
    GLOBAL_APP_METRICS.get_or_init(|| {
        AppMetrics::new().unwrap_or_else(|_| {
            // 如果创建失败，创建一个默认实例
            AppMetrics::new().expect("Failed to create AppMetrics")
        })
    })
}

/// 告警规则
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertRule {
    pub name: String,
    pub metric_name: String,
    pub threshold: f64,
    pub comparison: AlertComparison,
    pub duration_seconds: u64,
    pub severity: AlertSeverity,
    pub enabled: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertComparison {
    GreaterThan,
    LessThan,
    Equal,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertSeverity {
    Info,
    Warning,
    Critical,
}

/// 告警事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertEvent {
    pub rule_name: String,
    pub metric_name: String,
    pub current_value: f64,
    pub threshold: f64,
    pub severity: AlertSeverity,
    pub message: String,
    pub timestamp: SystemTime,
}

/// 健康检查状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthStatus {
    pub status: HealthState,
    pub checks: Vec<HealthCheck>,
    pub timestamp: SystemTime,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum HealthState {
    Healthy,
    Degraded,
    Unhealthy,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthCheck {
    pub name: String,
    pub status: HealthState,
    pub message: Option<String>,
    pub response_time_ms: Option<u64>,
}

/// 告警管理器
#[derive(Debug)]
pub struct AlertManager {
    rules: Vec<AlertRule>,
    alert_history: Vec<AlertEvent>,
    max_history_size: usize,
}

impl AlertManager {
    pub fn new() -> Self {
        Self {
            rules: Self::default_rules(),
            alert_history: Vec::new(),
            max_history_size: 1000,
        }
    }
    
    /// 默认告警规则
    fn default_rules() -> Vec<AlertRule> {
        vec![
            AlertRule {
                name: "高CPU使用率".to_string(),
                metric_name: "cpu_usage_percent".to_string(),
                threshold: 80.0,
                comparison: AlertComparison::GreaterThan,
                duration_seconds: 300,
                severity: AlertSeverity::Warning,
                enabled: true,
            },
            AlertRule {
                name: "高内存使用率".to_string(),
                metric_name: "memory_usage_bytes".to_string(),
                threshold: 0.85, // 85%
                comparison: AlertComparison::GreaterThan,
                duration_seconds: 300,
                severity: AlertSeverity::Warning,
                enabled: true,
            },
            AlertRule {
                name: "API错误率过高".to_string(),
                metric_name: "api_error_rate".to_string(),
                threshold: 0.05, // 5%
                comparison: AlertComparison::GreaterThan,
                duration_seconds: 60,
                severity: AlertSeverity::Critical,
                enabled: true,
            },
            AlertRule {
                name: "数据库连接池耗尽".to_string(),
                metric_name: "db_pool_utilization".to_string(),
                threshold: 0.9, // 90%
                comparison: AlertComparison::GreaterThan,
                duration_seconds: 60,
                severity: AlertSeverity::Critical,
                enabled: true,
            },
        ]
    }
    
    /// 检查告警规则
    pub fn check_alerts(&mut self, metrics: &AppMetrics) -> Vec<AlertEvent> {
        let mut triggered_alerts = Vec::new();
        let mut alerts_to_add = Vec::new();
        
        for rule in &self.rules {
            if !rule.enabled {
                continue;
            }
            
            let current_value = Self::get_metric_value_static(metrics, &rule.metric_name);
            let should_alert = match rule.comparison {
                AlertComparison::GreaterThan => current_value > rule.threshold,
                AlertComparison::LessThan => current_value < rule.threshold,
                AlertComparison::Equal => (current_value - rule.threshold).abs() < f64::EPSILON,
            };
            
            if should_alert {
                let alert = AlertEvent {
                    rule_name: rule.name.clone(),
                    metric_name: rule.metric_name.clone(),
                    current_value,
                    threshold: rule.threshold,
                    severity: rule.severity.clone(),
                    message: format!(
                        "指标 {} 当前值 {:.2} 超过阈值 {:.2}",
                        rule.metric_name, current_value, rule.threshold
                    ),
                    timestamp: SystemTime::now(),
                };
                
                triggered_alerts.push(alert.clone());
                alerts_to_add.push(alert);
            }
        }
        
        // 批量添加告警到历史记录
        for alert in alerts_to_add {
            self.add_alert_to_history(alert);
        }
        
        triggered_alerts
    }
    
    /// 获取指标值（静态方法）
    fn get_metric_value_static(metrics: &AppMetrics, metric_name: &str) -> f64 {
        match metric_name {
            "cpu_usage_percent" => {
                // 从Prometheus获取当前值
                // 这里简化为直接从gauge获取
                0.0 // 实际实现需要从Prometheus查询
            }
            "memory_usage_bytes" => 0.0,
            "api_error_rate" => 0.0,
            "db_pool_utilization" => {
                let current = metrics.db_connections.get();
                let max = metrics.db_pool_capacity.get();
                if max > 0.0 { current / max } else { 0.0 }
            }
            _ => 0.0,
        }
    }
    
    /// 添加告警到历史记录
    fn add_alert_to_history(&mut self, alert: AlertEvent) {
        self.alert_history.push(alert);
        
        // 限制历史记录大小
        if self.alert_history.len() > self.max_history_size {
            self.alert_history.remove(0);
        }
    }
    
    /// 获取告警历史
    pub fn get_alert_history(&self) -> &[AlertEvent] {
        &self.alert_history
    }
}

/// 健康检查管理器
pub struct HealthCheckManager {
    checks: Vec<DatabaseHealthChecker>,
}

#[async_trait::async_trait]
pub trait HealthChecker: Send + Sync {
    fn name(&self) -> &str;
    async fn check(&self) -> HealthCheck;
}

impl HealthCheckManager {
    pub fn new() -> Self {
        Self {
            checks: Vec::new(),
        }
    }
    
    pub fn add_checker(&mut self, checker: DatabaseHealthChecker) {
        self.checks.push(checker);
    }
    
    /// 执行所有健康检查
    pub async fn check_health(&self) -> HealthStatus {
        let mut checks = Vec::new();
        let mut overall_status = HealthState::Healthy;
        
        for checker in &self.checks {
            let check = checker.check().await;
            
            match check.status {
                HealthState::Degraded if matches!(overall_status, HealthState::Healthy) => {
                    overall_status = HealthState::Degraded;
                }
                HealthState::Unhealthy => {
                    overall_status = HealthState::Unhealthy;
                }
                _ => {}
            }
            
            checks.push(check);
        }
        
        HealthStatus {
            status: overall_status,
            checks,
            timestamp: SystemTime::now(),
        }
    }
}

/// 数据库健康检查器
pub struct DatabaseHealthChecker {
    db: Arc<Database>,
}

impl DatabaseHealthChecker {
    pub fn new(db: Arc<Database>) -> Self {
        Self { db }
    }
}

#[async_trait::async_trait]
impl HealthChecker for DatabaseHealthChecker {
    fn name(&self) -> &str {
        "database"
    }
    
    async fn check(&self) -> HealthCheck {
        let start = Instant::now();
        
        // 简单的数据库连接检查
        let metrics = self.db.get_db_metrics();
        let response_time = start.elapsed().as_millis() as u64;
        
        let status = if metrics.pg_pool_size > 0 {
            HealthState::Healthy
        } else {
            HealthState::Unhealthy
        };
        
        HealthCheck {
            name: "database".to_string(),
            status,
            message: Some(format!("连接池大小: {}/{}", metrics.pg_pool_size, metrics.pg_pool_max_size)),
            response_time_ms: Some(response_time),
        }
    }
}

/// 增强的指标服务
impl MetricsService {
    /// 创建带告警的指标服务
    pub fn with_alerting(db: Arc<Database>) -> Result<(Self, AlertManager, HealthCheckManager), prometheus::Error> {
        let metrics_service = Self::new(db.clone())?;
        let alert_manager = AlertManager::new();
        
        let mut health_manager = HealthCheckManager::new();
        health_manager.add_checker(DatabaseHealthChecker::new(db));
        
        Ok((metrics_service, alert_manager, health_manager))
    }
    
    /// 启动指标收集和告警检查
    pub async fn start_monitoring(&self, mut alert_manager: AlertManager, health_manager: HealthCheckManager) {
        let mut interval = interval(Duration::from_secs(30));
        
        loop {
            interval.tick().await;
            
            // 更新所有指标
            self.update_all_metrics();
            
            // 检查告警
            let alerts = alert_manager.check_alerts(&self.metrics);
            for alert in alerts {
                match alert.severity {
                    AlertSeverity::Info => {
                        info!("告警: {} - {}", alert.rule_name, alert.message);
                    }
                    AlertSeverity::Warning => {
                        warn!("告警: {} - {}", alert.rule_name, alert.message);
                    }
                    AlertSeverity::Critical => {
                        error!("严重告警: {} - {}", alert.rule_name, alert.message);
                    }
                }
            }
            
            // 执行健康检查（每5分钟一次）
            if SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs() % 300 == 0 
            {
                let health_status = health_manager.check_health().await;
                match health_status.status {
                    HealthState::Healthy => {
                        info!("健康检查: 系统状态良好");
                    }
                    HealthState::Degraded => {
                        warn!("健康检查: 系统性能下降");
                    }
                    HealthState::Unhealthy => {
                        error!("健康检查: 系统状态异常");
                    }
                }
            }
        }
    }
}

pub fn metrics_middleware() -> tower_http::trace::TraceLayer<
    tower_http::classify::SharedClassifier<tower_http::classify::ServerErrorsAsFailures>
> {
    use tower_http::trace::TraceLayer;
    
    TraceLayer::new_for_http()
}
