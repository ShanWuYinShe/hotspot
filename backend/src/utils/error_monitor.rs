/// 错误监控和指标收集模块
/// 
/// 提供错误统计、性能监控和告警功能
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};
use tokio::sync::RwLock;
use tracing::{error, warn, info};
use serde::{Serialize, Deserialize};

use crate::utils::error::{AppError, ErrorContext};

/// 错误统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorStats {
    pub total_errors: u64,
    pub error_by_type: HashMap<String, u64>,
    pub error_by_operation: HashMap<String, u64>,
    pub recent_errors: Vec<ErrorRecord>,
    pub uptime_seconds: u64,
    pub last_reset: chrono::DateTime<chrono::Utc>,
}

/// 错误记录
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorRecord {
    pub error_type: String,
    pub message: String,
    pub operation: String,
    pub user_id: Option<i64>,
    pub request_id: Option<String>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub source_location: Option<String>,
    pub additional_data: HashMap<String, String>,
}

/// 错误监控器
#[derive(Debug)]
pub struct ErrorMonitor {
    /// 总错误计数
    total_errors: AtomicU64,
    /// 按类型分类的错误计数
    error_by_type: Arc<RwLock<HashMap<String, AtomicU64>>>,
    /// 按操作分类的错误计数
    error_by_operation: Arc<RwLock<HashMap<String, AtomicU64>>>,
    /// 最近的错误记录（保留最近100条）
    recent_errors: Arc<RwLock<Vec<ErrorRecord>>>,
    /// 启动时间
    start_time: chrono::DateTime<chrono::Utc>,
    /// 最后重置时间
    last_reset: Arc<RwLock<chrono::DateTime<chrono::Utc>>>,
}

impl Default for ErrorMonitor {
    fn default() -> Self {
        Self::new()
    }
}

impl ErrorMonitor {
    /// 创建新的错误监控器
    pub fn new() -> Self {
        let now = chrono::Utc::now();
        Self {
            total_errors: AtomicU64::new(0),
            error_by_type: Arc::new(RwLock::new(HashMap::new())),
            error_by_operation: Arc::new(RwLock::new(HashMap::new())),
            recent_errors: Arc::new(RwLock::new(Vec::new())),
            start_time: now,
            last_reset: Arc::new(RwLock::new(now)),
        }
    }

    /// 记录错误
    pub async fn record_error(&self, error: &AppError, context: Option<ErrorContext>) {
        // 增加总错误计数
        let total = self.total_errors.fetch_add(1, Ordering::Relaxed) + 1;
        
        let error_type = self.get_error_type_name(error);
        let operation = context.as_ref().map(|c| c.operation.clone()).unwrap_or_else(|| "unknown".to_string());
        
        // 按类型计数
        {
            let mut type_counters = self.error_by_type.write().await;
            let counter = type_counters.entry(error_type.clone()).or_insert_with(|| AtomicU64::new(0));
            counter.fetch_add(1, Ordering::Relaxed);
        }
        
        // 按操作计数
        {
            let mut operation_counters = self.error_by_operation.write().await;
            let counter = operation_counters.entry(operation.clone()).or_insert_with(|| AtomicU64::new(0));
            counter.fetch_add(1, Ordering::Relaxed);
        }
        
        // 记录详细错误信息
        let error_record = ErrorRecord {
            error_type: error_type.clone(),
            message: error.to_string(),
            operation: operation.clone(),
            user_id: context.as_ref().and_then(|c| c.user_id),
            request_id: context.as_ref().and_then(|c| c.request_id.clone()),
            timestamp: chrono::Utc::now(),
            source_location: context.as_ref().and_then(|c| c.source_location.clone()),
            additional_data: context.as_ref().map(|c| c.additional_data.clone()).unwrap_or_default(),
        };
        
        {
            let mut recent = self.recent_errors.write().await;
            recent.push(error_record);
            
            // 只保留最近100条错误记录
            if recent.len() > 100 {
                recent.remove(0);
            }
        }
        
        // 记录日志
        if total % 100 == 0 {
            warn!("错误统计: 总错误数达到 {}, 类型: {}, 操作: {}", total, error_type, operation);
        }
        
        // 检查是否需要告警
        self.check_alert_conditions(&error_type, &operation).await;
    }

    /// 获取错误统计信息
    pub async fn get_stats(&self) -> ErrorStats {
        let type_counters = self.error_by_type.read().await;
        let operation_counters = self.error_by_operation.read().await;
        let recent_errors = self.recent_errors.read().await;
        let last_reset = *self.last_reset.read().await;
        
        let error_by_type = type_counters
            .iter()
            .map(|(k, v)| (k.clone(), v.load(Ordering::Relaxed)))
            .collect();
            
        let error_by_operation = operation_counters
            .iter()
            .map(|(k, v)| (k.clone(), v.load(Ordering::Relaxed)))
            .collect();
        
        ErrorStats {
            total_errors: self.total_errors.load(Ordering::Relaxed),
            error_by_type,
            error_by_operation,
            recent_errors: recent_errors.clone(),
            uptime_seconds: (chrono::Utc::now() - self.start_time).num_seconds() as u64,
            last_reset,
        }
    }

    /// 重置统计信息
    pub async fn reset_stats(&self) {
        self.total_errors.store(0, Ordering::Relaxed);
        
        {
            let mut type_counters = self.error_by_type.write().await;
            type_counters.clear();
        }
        
        {
            let mut operation_counters = self.error_by_operation.write().await;
            operation_counters.clear();
        }
        
        {
            let mut recent_errors = self.recent_errors.write().await;
            recent_errors.clear();
        }
        
        {
            let mut last_reset = self.last_reset.write().await;
            *last_reset = chrono::Utc::now();
        }
        
        info!("错误统计信息已重置");
    }

    /// 获取错误类型名称
    fn get_error_type_name(&self, error: &AppError) -> String {
        match error {
            AppError::NotFound(_) => "NotFound".to_string(),
            AppError::BadRequest(_) => "BadRequest".to_string(),
            AppError::InternalServerError(_) => "InternalServerError".to_string(),
            AppError::Unauthorized(_) => "Unauthorized".to_string(),
            AppError::Forbidden(_) => "Forbidden".to_string(),
            AppError::Conflict(_) => "Conflict".to_string(),
            AppError::DatabaseError(_) => "DatabaseError".to_string(),
            AppError::ExternalServiceError(_) => "ExternalServiceError".to_string(),
            AppError::NotImplemented(_) => "NotImplemented".to_string(),
            AppError::ValidationError(_) => "ValidationError".to_string(),
            AppError::Timeout(_) => "Timeout".to_string(),
        }
    }

    /// 检查告警条件
    async fn check_alert_conditions(&self, error_type: &str, operation: &str) {
        let total_errors = self.total_errors.load(Ordering::Relaxed);
        
        // 总错误数告警
        if total_errors > 0 && total_errors % 1000 == 0 {
            error!("🚨 高错误率告警: 总错误数已达到 {}", total_errors);
        }
        
        // 特定错误类型告警
        if error_type == "DatabaseError" {
            let type_counters = self.error_by_type.read().await;
            if let Some(counter) = type_counters.get(error_type) {
                let count = counter.load(Ordering::Relaxed);
                if count > 0 && count % 50 == 0 {
                    error!("🚨 数据库错误告警: {} 错误已达到 {} 次", error_type, count);
                }
            }
        }
        
        // 特定操作告警
        let operation_counters = self.error_by_operation.read().await;
        if let Some(counter) = operation_counters.get(operation) {
            let count = counter.load(Ordering::Relaxed);
            if count > 0 && count % 100 == 0 {
                warn!("⚠️ 操作错误告警: {} 操作已出现 {} 次错误", operation, count);
            }
        }
    }

    /// 获取错误率（每分钟错误数）
    pub async fn get_error_rate(&self) -> f64 {
        let uptime_minutes = (chrono::Utc::now() - self.start_time).num_minutes() as f64;
        if uptime_minutes > 0.0 {
            self.total_errors.load(Ordering::Relaxed) as f64 / uptime_minutes
        } else {
            0.0
        }
    }

    /// 获取健康状态
    pub async fn get_health_status(&self) -> HealthStatus {
        let error_rate = self.get_error_rate().await;
        let total_errors = self.total_errors.load(Ordering::Relaxed);
        
        if error_rate > 10.0 || total_errors > 5000 {
            HealthStatus::Critical
        } else if error_rate > 5.0 || total_errors > 1000 {
            HealthStatus::Warning
        } else {
            HealthStatus::Healthy
        }
    }
}

/// 健康状态枚举
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum HealthStatus {
    Healthy,
    Warning,
    Critical,
}

/// 全局错误监控器实例
static ERROR_MONITOR: once_cell::sync::Lazy<ErrorMonitor> = once_cell::sync::Lazy::new(ErrorMonitor::new);

/// 获取全局错误监控器
pub fn get_error_monitor() -> &'static ErrorMonitor {
    &ERROR_MONITOR
}

/// 便捷函数：记录错误
pub async fn record_error(error: &AppError, context: Option<ErrorContext>) {
    get_error_monitor().record_error(error, context).await;
}

/// 便捷函数：获取错误统计
pub async fn get_error_stats() -> ErrorStats {
    get_error_monitor().get_stats().await
}

/// 便捷函数：重置统计
pub async fn reset_error_stats() {
    get_error_monitor().reset_stats().await;
}
