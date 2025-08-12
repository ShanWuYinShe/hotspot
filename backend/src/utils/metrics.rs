//! 应用监控指标模块

use std::sync::Arc;
use tracing::info;
use prometheus::{
    Encoder, TextEncoder, 
    Gauge, 
    Counter, CounterVec, 
    Histogram, HistogramVec,
    opts, register_counter_vec, register_histogram_vec,
    register_counter, register_gauge, register_histogram,
};

use crate::config::db::Database;

/// 应用监控指标
#[derive(Debug, Clone)]
pub struct AppMetrics {
    // API请求计数
    #[allow(dead_code)]
    pub http_requests_total: CounterVec,
    // API请求延迟
    #[allow(dead_code)]
    pub http_request_duration_seconds: HistogramVec,
    // 活跃用户数
    #[allow(dead_code)]
    pub active_users: Gauge,
    // 数据库连接数
    pub db_connections: Gauge,
    // 数据库连接池容量
    pub db_pool_capacity: Gauge,
    // Redis连接状态
    pub redis_connected: Gauge,
    // WebSocket连接数
    #[allow(dead_code)]
    pub ws_connections: Gauge,
    // WebSocket消息发送计数
    #[allow(dead_code)]
    pub ws_messages_sent: Counter,
    // WebSocket消息接收计数
    #[allow(dead_code)]
    pub ws_messages_received: Counter,
    // WebSocket订阅计数
    #[allow(dead_code)]
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
    #[allow(dead_code)]
    pub location_query_count: Counter,
    // 位置查询延迟
    #[allow(dead_code)]
    pub location_query_duration_seconds: Histogram,
    // 缓存命中率
    #[allow(dead_code)]
    pub cache_hit_ratio: Gauge,
    // 系统内存使用
    pub memory_usage_bytes: Gauge,
    // 系统CPU使用
    pub cpu_usage_percent: Gauge,
    // JVM运行时间
    pub uptime_seconds: Gauge,
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
        
        // JVM运行时间
        let uptime_seconds = register_gauge!(
            opts!("uptime_seconds", "应用运行时间（秒）")
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
            uptime_seconds,
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
    #[allow(dead_code)]
    pub fn record_http_request(&self, method: &str, path: &str, status: u16, duration: f64) {
        self.http_requests_total
            .with_label_values(&[method, path, &status.to_string()])
            .inc();
            
        self.http_request_duration_seconds
            .with_label_values(&[method, path])
            .observe(duration);
    }
    
    /// 记录WebSocket消息
    #[allow(dead_code)]
    pub fn record_ws_message_sent(&self) {
        self.ws_messages_sent.inc();
    }
    
    #[allow(dead_code)]
    pub fn record_ws_message_received(&self) {
        self.ws_messages_received.inc();
    }

    #[allow(dead_code)]
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
    #[allow(dead_code)]
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
pub fn metrics_middleware() -> tower_http::trace::TraceLayer<
    tower_http::classify::SharedClassifier<tower_http::classify::ServerErrorsAsFailures>
> {
    use tower_http::trace::TraceLayer;
    
    TraceLayer::new_for_http()
}
