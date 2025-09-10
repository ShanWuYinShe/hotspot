/// 高性能日志记录系统
/// 
/// 提供异步、批量、结构化的日志记录功能，最小化对业务逻辑的性能影响
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, SystemTime};
use tokio::sync::mpsc;
use tokio::time::interval;
use serde_json::Value;

/// 日志级别
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum LogLevel {
    Trace,
    Debug,
    Info,
    Warn,
    Error,
}

impl LogLevel {
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Trace => "TRACE",
            Self::Debug => "DEBUG",
            Self::Info => "INFO",
            Self::Warn => "WARN",
            Self::Error => "ERROR",
        }
    }

    pub fn should_log(&self, min_level: LogLevel) -> bool {
        (*self as u8) >= (min_level as u8)
    }
}

/// 结构化日志条目
#[derive(Debug, Clone)]
pub struct LogEntry {
    pub level: LogLevel,
    pub message: String,
    pub module: String,
    pub timestamp: SystemTime,
    pub fields: HashMap<String, Value>,
    pub duration_ms: Option<u64>,
}

impl LogEntry {
    pub fn new(level: LogLevel, message: String, module: String) -> Self {
        Self {
            level,
            message,
            module,
            timestamp: SystemTime::now(),
            fields: HashMap::new(),
            duration_ms: None,
        }
    }

    pub fn with_field(mut self, key: &str, value: Value) -> Self {
        self.fields.insert(key.to_string(), value);
        self
    }

    pub fn with_duration(mut self, duration: Duration) -> Self {
        self.duration_ms = Some(duration.as_millis() as u64);
        self
    }

    pub fn with_fields(mut self, fields: HashMap<String, Value>) -> Self {
        self.fields.extend(fields);
        self
    }
}

/// 日志批处理器配置
#[derive(Debug, Clone)]
pub struct LoggerConfig {
    /// 最小日志级别
    pub min_level: LogLevel,
    /// 批处理大小
    pub batch_size: usize,
    /// 批处理超时时间
    pub batch_timeout: Duration,
    /// 缓冲区大小
    pub buffer_size: usize,
    /// 是否启用异步处理
    pub async_processing: bool,
    /// 是否启用性能监控
    pub enable_performance_monitoring: bool,
}

impl Default for LoggerConfig {
    fn default() -> Self {
        Self {
            min_level: LogLevel::Info,
            batch_size: 100,
            batch_timeout: Duration::from_millis(500),
            buffer_size: 1000,
            async_processing: true,
            enable_performance_monitoring: true,
        }
    }
}

/// 日志统计信息
#[derive(Debug, Clone)]
pub struct LogStats {
    pub total_logs: u64,
    pub logs_by_level: HashMap<LogLevel, u64>,
    pub batches_processed: u64,
    pub avg_batch_size: f64,
    pub avg_processing_time_ms: f64,
    pub dropped_logs: u64,
    pub last_reset: SystemTime,
}

impl LogStats {
    pub fn new() -> Self {
        Self {
            total_logs: 0,
            logs_by_level: HashMap::new(),
            batches_processed: 0,
            avg_batch_size: 0.0,
            avg_processing_time_ms: 0.0,
            dropped_logs: 0,
            last_reset: SystemTime::now(),
        }
    }

    pub fn record_log(&mut self, level: LogLevel) {
        self.total_logs += 1;
        *self.logs_by_level.entry(level).or_insert(0) += 1;
    }

    pub fn record_batch(&mut self, batch_size: usize, processing_time: Duration) {
        self.batches_processed += 1;
        
        // 更新平均批处理大小
        self.avg_batch_size = (self.avg_batch_size * (self.batches_processed - 1) as f64 + batch_size as f64) / self.batches_processed as f64;
        
        // 更新平均处理时间
        let processing_time_ms = processing_time.as_millis() as f64;
        self.avg_processing_time_ms = (self.avg_processing_time_ms * (self.batches_processed - 1) as f64 + processing_time_ms) / self.batches_processed as f64;
    }

    pub fn record_dropped_log(&mut self) {
        self.dropped_logs += 1;
    }
}

/// 高性能日志记录器
pub struct HighPerformanceLogger {
    config: LoggerConfig,
    sender: mpsc::UnboundedSender<LogEntry>,
    stats: Arc<tokio::sync::Mutex<LogStats>>,
}

impl HighPerformanceLogger {
    pub fn new(config: LoggerConfig) -> Self {
        let (sender, receiver) = mpsc::unbounded_channel();
        let stats = Arc::new(tokio::sync::Mutex::new(LogStats::new()));
        
        // 启动后台日志处理任务
        if config.async_processing {
            let processor_config = config.clone();
            let processor_stats = stats.clone();
            tokio::spawn(async move {
                Self::log_processor(receiver, processor_config, processor_stats).await;
            });
        }
        
        Self {
            config,
            sender,
            stats,
        }
    }

    /// 记录日志
    pub fn log(&self, entry: LogEntry) {
        if !entry.level.should_log(self.config.min_level) {
            return;
        }

        if let Err(_) = self.sender.send(entry.clone()) {
            // 如果发送失败，记录丢弃的日志数
            if let Ok(mut stats) = self.stats.try_lock() {
                stats.record_dropped_log();
            }
            
            // 降级到同步日志记录
            self.fallback_log(&entry);
        } else {
            // 记录日志统计
            if let Ok(mut stats) = self.stats.try_lock() {
                stats.record_log(entry.level);
            }
        }
    }

    /// 便捷的日志记录方法
    pub fn trace(&self, module: &str, message: &str) {
        self.log(LogEntry::new(LogLevel::Trace, message.to_string(), module.to_string()));
    }

    pub fn debug(&self, module: &str, message: &str) {
        self.log(LogEntry::new(LogLevel::Debug, message.to_string(), module.to_string()));
    }

    pub fn info(&self, module: &str, message: &str) {
        self.log(LogEntry::new(LogLevel::Info, message.to_string(), module.to_string()));
    }

    pub fn warn(&self, module: &str, message: &str) {
        self.log(LogEntry::new(LogLevel::Warn, message.to_string(), module.to_string()));
    }

    pub fn error(&self, module: &str, message: &str) {
        self.log(LogEntry::new(LogLevel::Error, message.to_string(), module.to_string()));
    }

    /// 带字段的日志记录
    pub fn log_with_fields(&self, level: LogLevel, module: &str, message: &str, fields: HashMap<String, Value>) {
        let entry = LogEntry::new(level, message.to_string(), module.to_string())
            .with_fields(fields);
        self.log(entry);
    }

    /// 带持续时间的日志记录
    pub fn log_with_duration(&self, level: LogLevel, module: &str, message: &str, duration: Duration) {
        let entry = LogEntry::new(level, message.to_string(), module.to_string())
            .with_duration(duration);
        self.log(entry);
    }

    /// 性能监控日志记录
    pub fn log_performance(&self, module: &str, operation: &str, duration: Duration, success: bool) {
        if !self.config.enable_performance_monitoring {
            return;
        }

        let mut fields = HashMap::new();
        fields.insert("operation".to_string(), Value::String(operation.to_string()));
        fields.insert("duration_ms".to_string(), Value::Number(serde_json::Number::from(duration.as_millis() as u64)));
        fields.insert("success".to_string(), Value::Bool(success));

        let level = if duration.as_millis() > 100 {
            LogLevel::Warn
        } else {
            LogLevel::Debug
        };

        let message = if success {
            format!("操作 '{}' 完成，耗时 {}ms", operation, duration.as_millis())
        } else {
            format!("操作 '{}' 失败，耗时 {}ms", operation, duration.as_millis())
        };

        self.log_with_fields(level, module, &message, fields);
    }

    /// 获取日志统计
    pub async fn get_stats(&self) -> LogStats {
        self.stats.lock().await.clone()
    }

    /// 重置统计
    pub async fn reset_stats(&self) {
        let mut stats = self.stats.lock().await;
        *stats = LogStats::new();
    }

    /// 降级日志记录（同步）
    fn fallback_log(&self, entry: &LogEntry) {
        let timestamp = entry.timestamp
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        eprintln!(
            "[{}] {} [{}] {}: {}",
            timestamp,
            entry.level.as_str(),
            entry.module,
            entry.message,
            if entry.fields.is_empty() {
                String::new()
            } else {
                format!(" {:?}", entry.fields)
            }
        );
    }

    /// 后台日志处理器
    async fn log_processor(
        mut receiver: mpsc::UnboundedReceiver<LogEntry>,
        config: LoggerConfig,
        stats: Arc<tokio::sync::Mutex<LogStats>>,
    ) {
        let mut batch = Vec::with_capacity(config.batch_size);
        let mut interval = interval(config.batch_timeout);

        loop {
            tokio::select! {
                // 接收日志条目
                entry = receiver.recv() => {
                    match entry {
                        Some(log_entry) => {
                            batch.push(log_entry);
                            
                            // 如果批处理达到大小限制，立即处理
                            if batch.len() >= config.batch_size {
                                Self::process_batch(&mut batch, &stats).await;
                            }
                        }
                        None => {
                            // 通道关闭，处理剩余的日志并退出
                            if !batch.is_empty() {
                                Self::process_batch(&mut batch, &stats).await;
                            }
                            break;
                        }
                    }
                }
                
                // 定时处理批次
                _ = interval.tick() => {
                    if !batch.is_empty() {
                        Self::process_batch(&mut batch, &stats).await;
                    }
                }
            }
        }
    }

    /// 处理日志批次
    async fn process_batch(batch: &mut Vec<LogEntry>, stats: &Arc<tokio::sync::Mutex<LogStats>>) {
        if batch.is_empty() {
            return;
        }

        let start = std::time::Instant::now();
        let batch_size = batch.len();

        // 批量输出日志
        for entry in batch.iter() {
            let timestamp = entry.timestamp
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap_or_default()
                .as_secs();

            let duration_info = if let Some(duration_ms) = entry.duration_ms {
                format!(" [{}ms]", duration_ms)
            } else {
                String::new()
            };

            let fields_info = if entry.fields.is_empty() {
                String::new()
            } else {
                format!(" {:?}", entry.fields)
            };

            println!(
                "[{}] {} [{}]{}: {}{}",
                timestamp,
                entry.level.as_str(),
                entry.module,
                duration_info,
                entry.message,
                fields_info
            );
        }

        let processing_time = start.elapsed();

        // 更新统计信息
        if let Ok(mut stats_guard) = stats.lock().await {
            stats_guard.record_batch(batch_size, processing_time);
        }

        // 清空批次
        batch.clear();
    }
}

/// 全局高性能日志记录器实例
static GLOBAL_LOGGER: std::sync::OnceLock<HighPerformanceLogger> = std::sync::OnceLock::new();

/// 初始化全局日志记录器
pub fn init_logger(config: LoggerConfig) {
    GLOBAL_LOGGER.set(HighPerformanceLogger::new(config)).ok();
}

/// 获取全局日志记录器
pub fn get_logger() -> Option<&'static HighPerformanceLogger> {
    GLOBAL_LOGGER.get()
}

/// 便捷的日志记录宏
#[macro_export]
macro_rules! hp_log {
    ($level:expr, $module:expr, $message:expr) => {
        if let Some(logger) = $crate::utils::high_performance_logger::get_logger() {
            logger.log($crate::utils::high_performance_logger::LogEntry::new(
                $level,
                $message.to_string(),
                $module.to_string(),
            ));
        }
    };
    
    ($level:expr, $module:expr, $message:expr, $($key:expr => $value:expr),+) => {
        if let Some(logger) = $crate::utils::high_performance_logger::get_logger() {
            let mut fields = std::collections::HashMap::new();
            $(
                fields.insert($key.to_string(), serde_json::json!($value));
            )+
            logger.log_with_fields($level, $module, $message, fields);
        }
    };
}

/// 便捷的性能日志记录函数
pub fn log_performance(module: &str, operation: &str, duration: Duration, success: bool) {
    if let Some(logger) = get_logger() {
        logger.log_performance(module, operation, duration, success);
    }
}

/// 便捷的错误日志记录函数
pub fn log_error(module: &str, message: &str) {
    if let Some(logger) = get_logger() {
        logger.error(module, message);
    }
}

/// 便捷的警告日志记录函数
pub fn log_warn(module: &str, message: &str) {
    if let Some(logger) = get_logger() {
        logger.warn(module, message);
    }
}

/// 便捷的信息日志记录函数
pub fn log_info(module: &str, message: &str) {
    if let Some(logger) = get_logger() {
        logger.info(module, message);
    }
}

/// 便捷的调试日志记录函数
pub fn log_debug(module: &str, message: &str) {
    if let Some(logger) = get_logger() {
        logger.debug(module, message);
    }
}
