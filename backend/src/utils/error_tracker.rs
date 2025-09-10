/// 错误率跟踪和告警系统
/// 
/// 提供系统错误率监控、分类统计和自动告警功能
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, SystemTime};
use serde_json::Value;

/// 错误类型分类
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum ErrorCategory {
    /// 数据库错误
    Database,
    /// 网络错误
    Network,
    /// 认证错误
    Authentication,
    /// 权限错误
    Authorization,
    /// 业务逻辑错误
    Business,
    /// 系统错误
    System,
    /// 外部服务错误
    ExternalService,
    /// 未知错误
    Unknown,
}

impl ErrorCategory {
    pub fn from_error_message(message: &str) -> Self {
        let message_lower = message.to_lowercase();
        
        if message_lower.contains("database") || message_lower.contains("sql") || message_lower.contains("connection") {
            Self::Database
        } else if message_lower.contains("network") || message_lower.contains("timeout") || message_lower.contains("connection refused") {
            Self::Network
        } else if message_lower.contains("unauthorized") || message_lower.contains("invalid token") || message_lower.contains("authentication") {
            Self::Authentication
        } else if message_lower.contains("forbidden") || message_lower.contains("permission") || message_lower.contains("access denied") {
            Self::Authorization
        } else if message_lower.contains("business") || message_lower.contains("validation") || message_lower.contains("invalid") {
            Self::Business
        } else if message_lower.contains("system") || message_lower.contains("internal") {
            Self::System
        } else if message_lower.contains("external") || message_lower.contains("service") {
            Self::ExternalService
        } else {
            Self::Unknown
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Database => "database",
            Self::Network => "network",
            Self::Authentication => "authentication",
            Self::Authorization => "authorization",
            Self::Business => "business",
            Self::System => "system",
            Self::ExternalService => "external_service",
            Self::Unknown => "unknown",
        }
    }
}

/// 错误统计信息
#[derive(Debug, Clone)]
pub struct ErrorStats {
    /// 错误类型
    pub category: ErrorCategory,
    /// 错误总数
    pub total_count: u64,
    /// 最近1分钟错误数
    pub recent_1min: u64,
    /// 最近5分钟错误数
    pub recent_5min: u64,
    /// 最近15分钟错误数
    pub recent_15min: u64,
    /// 最近1小时错误数
    pub recent_1hour: u64,
    /// 第一次出现时间
    pub first_occurrence: SystemTime,
    /// 最后一次出现时间
    pub last_occurrence: SystemTime,
    /// 错误消息样本（最近的几个）
    pub recent_messages: Vec<String>,
}

impl ErrorStats {
    pub fn new(category: ErrorCategory, message: String) -> Self {
        let now = SystemTime::now();
        Self {
            category,
            total_count: 1,
            recent_1min: 1,
            recent_5min: 1,
            recent_15min: 1,
            recent_1hour: 1,
            first_occurrence: now,
            last_occurrence: now,
            recent_messages: vec![message],
        }
    }

    pub fn record_error(&mut self, message: String) {
        self.total_count += 1;
        self.last_occurrence = SystemTime::now();
        
        // 保留最近的5个错误消息
        self.recent_messages.push(message);
        if self.recent_messages.len() > 5 {
            self.recent_messages.remove(0);
        }
    }

    /// 清理过期的时间窗口计数
    pub fn cleanup_time_windows(&mut self) {
        let now = SystemTime::now();
        
        // 这里简化处理，实际应该基于时间戳精确计算
        // 在生产环境中应该使用更精确的时间窗口管理
        if let Ok(duration) = now.duration_since(self.last_occurrence) {
            if duration > Duration::from_secs(3600) {
                self.recent_1hour = 0;
            }
            if duration > Duration::from_secs(900) {
                self.recent_15min = 0;
            }
            if duration > Duration::from_secs(300) {
                self.recent_5min = 0;
            }
            if duration > Duration::from_secs(60) {
                self.recent_1min = 0;
            }
        }
    }
}

/// 错误率阈值配置
#[derive(Debug, Clone)]
pub struct ErrorThresholds {
    /// 1分钟内错误数阈值
    pub errors_per_minute: u64,
    /// 5分钟内错误数阈值
    pub errors_per_5min: u64,
    /// 15分钟内错误数阈值
    pub errors_per_15min: u64,
    /// 1小时内错误数阈值
    pub errors_per_hour: u64,
    /// 错误率百分比阈值
    pub error_rate_percent: f64,
}

impl Default for ErrorThresholds {
    fn default() -> Self {
        Self {
            errors_per_minute: 10,
            errors_per_5min: 30,
            errors_per_15min: 100,
            errors_per_hour: 200,
            error_rate_percent: 5.0,
        }
    }
}

/// 告警信息
#[derive(Debug, Clone)]
pub struct Alert {
    pub category: ErrorCategory,
    pub message: String,
    pub severity: AlertSeverity,
    pub timestamp: SystemTime,
    pub error_count: u64,
    pub time_window: String,
}

#[derive(Debug, Clone)]
pub enum AlertSeverity {
    Warning,
    Critical,
}

/// 错误跟踪器
pub struct ErrorTracker {
    stats: Arc<Mutex<HashMap<ErrorCategory, ErrorStats>>>,
    thresholds: ErrorThresholds,
    total_requests: Arc<Mutex<u64>>,
    alerts: Arc<Mutex<Vec<Alert>>>,
}

impl ErrorTracker {
    pub fn new() -> Self {
        Self::with_thresholds(ErrorThresholds::default())
    }

    pub fn with_thresholds(thresholds: ErrorThresholds) -> Self {
        Self {
            stats: Arc::new(Mutex::new(HashMap::new())),
            thresholds,
            total_requests: Arc::new(Mutex::new(0)),
            alerts: Arc::new(Mutex::new(Vec::new())),
        }
    }

    /// 记录错误
    pub fn record_error(&self, error_message: &str) {
        let category = ErrorCategory::from_error_message(error_message);
        let mut stats = self.stats.lock().unwrap();
        
        match stats.get_mut(&category) {
            Some(error_stats) => {
                error_stats.record_error(error_message.to_string());
            }
            None => {
                stats.insert(category.clone(), ErrorStats::new(category.clone(), error_message.to_string()));
            }
        }

        // 检查是否需要触发告警
        if let Some(error_stats) = stats.get(&category) {
            self.check_and_create_alerts(&category, error_stats);
        }
    }

    /// 记录成功请求
    pub fn record_success(&self) {
        let mut total = self.total_requests.lock().unwrap();
        *total += 1;
    }

    /// 检查并创建告警
    fn check_and_create_alerts(&self, category: &ErrorCategory, stats: &ErrorStats) {
        let mut alerts = self.alerts.lock().unwrap();
        
        // 检查各种阈值
        if stats.recent_1min >= self.thresholds.errors_per_minute {
            alerts.push(Alert {
                category: category.clone(),
                message: format!("高错误率告警: {} 类型错误在1分钟内发生{}次", category.as_str(), stats.recent_1min),
                severity: AlertSeverity::Critical,
                timestamp: SystemTime::now(),
                error_count: stats.recent_1min,
                time_window: "1分钟".to_string(),
            });
        } else if stats.recent_5min >= self.thresholds.errors_per_5min {
            alerts.push(Alert {
                category: category.clone(),
                message: format!("错误率告警: {} 类型错误在5分钟内发生{}次", category.as_str(), stats.recent_5min),
                severity: AlertSeverity::Warning,
                timestamp: SystemTime::now(),
                error_count: stats.recent_5min,
                time_window: "5分钟".to_string(),
            });
        }

        // 限制告警数量
        if alerts.len() > 100 {
            alerts.drain(0..50);
        }
    }

    /// 获取所有错误统计
    pub fn get_error_stats(&self) -> HashMap<ErrorCategory, ErrorStats> {
        let mut stats = self.stats.lock().unwrap();
        
        // 清理过期的时间窗口计数
        for error_stats in stats.values_mut() {
            error_stats.cleanup_time_windows();
        }
        
        stats.clone()
    }

    /// 获取错误率
    pub fn get_error_rate(&self) -> f64 {
        let stats = self.stats.lock().unwrap();
        let total_requests = *self.total_requests.lock().unwrap();
        let total_errors: u64 = stats.values().map(|s| s.total_count).sum();
        
        if total_requests == 0 {
            0.0
        } else {
            (total_errors as f64 / total_requests as f64) * 100.0
        }
    }

    /// 获取活跃告警
    pub fn get_active_alerts(&self) -> Vec<Alert> {
        let alerts = self.alerts.lock().unwrap();
        
        // 返回最近1小时的告警
        let one_hour_ago = SystemTime::now() - Duration::from_secs(3600);
        alerts.iter()
            .filter(|alert| alert.timestamp > one_hour_ago)
            .cloned()
            .collect()
    }

    /// 获取错误报告
    pub fn get_error_report(&self) -> Value {
        let stats = self.get_error_stats();
        let error_rate = self.get_error_rate();
        let alerts = self.get_active_alerts();
        let total_requests = *self.total_requests.lock().unwrap();
        let total_errors: u64 = stats.values().map(|s| s.total_count).sum();

        serde_json::json!({
            "summary": {
                "total_requests": total_requests,
                "total_errors": total_errors,
                "error_rate_percent": error_rate,
                "active_alerts": alerts.len(),
                "error_categories": stats.len()
            },
            "error_breakdown": stats.iter().map(|(category, error_stats)| {
                serde_json::json!({
                    "category": category.as_str(),
                    "total_count": error_stats.total_count,
                    "recent_1min": error_stats.recent_1min,
                    "recent_5min": error_stats.recent_5min,
                    "recent_15min": error_stats.recent_15min,
                    "recent_1hour": error_stats.recent_1hour,
                    "first_occurrence": error_stats.first_occurrence
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap_or_default()
                        .as_secs(),
                    "last_occurrence": error_stats.last_occurrence
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap_or_default()
                        .as_secs(),
                    "recent_messages": error_stats.recent_messages
                })
            }).collect::<Vec<_>>(),
            "active_alerts": alerts.iter().map(|alert| {
                serde_json::json!({
                    "category": alert.category.as_str(),
                    "message": alert.message,
                    "severity": match alert.severity {
                        AlertSeverity::Warning => "warning",
                        AlertSeverity::Critical => "critical"
                    },
                    "timestamp": alert.timestamp
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap_or_default()
                        .as_secs(),
                    "error_count": alert.error_count,
                    "time_window": alert.time_window
                })
            }).collect::<Vec<_>>(),
            "thresholds": {
                "errors_per_minute": self.thresholds.errors_per_minute,
                "errors_per_5min": self.thresholds.errors_per_5min,
                "errors_per_15min": self.thresholds.errors_per_15min,
                "errors_per_hour": self.thresholds.errors_per_hour,
                "error_rate_percent": self.thresholds.error_rate_percent
            }
        })
    }

    /// 重置统计数据
    pub fn reset_stats(&self) {
        let mut stats = self.stats.lock().unwrap();
        let mut total_requests = self.total_requests.lock().unwrap();
        let mut alerts = self.alerts.lock().unwrap();
        
        stats.clear();
        *total_requests = 0;
        alerts.clear();
    }

    /// 清理旧告警
    pub fn cleanup_old_alerts(&self, max_age: Duration) {
        let mut alerts = self.alerts.lock().unwrap();
        let cutoff = SystemTime::now() - max_age;
        alerts.retain(|alert| alert.timestamp > cutoff);
    }
}

impl Default for ErrorTracker {
    fn default() -> Self {
        Self::new()
    }
}

/// 全局错误跟踪器实例
static GLOBAL_ERROR_TRACKER: std::sync::OnceLock<ErrorTracker> = std::sync::OnceLock::new();

/// 获取全局错误跟踪器
pub fn get_error_tracker() -> &'static ErrorTracker {
    GLOBAL_ERROR_TRACKER.get_or_init(ErrorTracker::new)
}

/// 记录错误的便捷函数
pub fn record_error(error_message: &str) {
    get_error_tracker().record_error(error_message);
}

/// 记录成功请求的便捷函数
pub fn record_success() {
    get_error_tracker().record_success();
}

/// 获取错误报告的便捷函数
pub fn get_error_report() -> Value {
    get_error_tracker().get_error_report()
}

/// 获取活跃告警的便捷函数
pub fn get_active_alerts() -> Vec<Alert> {
    get_error_tracker().get_active_alerts()
}
