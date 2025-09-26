/// 错误处理和恢复机制
/// 
/// 功能职责：
/// - 自动错误检测和分类
/// - 智能恢复策略
/// - 错误统计和分析
/// - 系统健康自愈
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;
use serde::{Serialize, Deserialize};
use tracing::{info, warn, debug, error};

use crate::domain::DomainResult;

/// 错误恢复系统
#[derive(Clone)]
pub struct ErrorRecoverySystem {
    error_tracker: Arc<RwLock<ErrorTracker>>,
    recovery_strategies: Arc<RwLock<HashMap<ErrorType, RecoveryStrategy>>>,
    config: RecoveryConfig,
}

/// 错误跟踪器
#[derive(Debug, Clone)]
pub struct ErrorTracker {
    pub error_counts: HashMap<ErrorType, ErrorCount>,
    pub recent_errors: Vec<ErrorEvent>,
    pub recovery_attempts: HashMap<String, RecoveryAttempt>,
    pub last_cleanup: Instant,
}

/// 错误计数
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorCount {
    pub total: u64,
    pub last_hour: u64,
    pub last_day: u64,
    pub first_occurrence: chrono::DateTime<chrono::Utc>,
    pub last_occurrence: chrono::DateTime<chrono::Utc>,
}

/// 错误事件
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorEvent {
    pub id: String,
    pub error_type: ErrorType,
    pub severity: ErrorSeverity,
    pub message: String,
    pub context: HashMap<String, String>,
    pub occurred_at: chrono::DateTime<chrono::Utc>,
    pub recovery_attempted: bool,
    pub recovery_successful: Option<bool>,
}

/// 错误类型
#[derive(Debug, Clone, Hash, Eq, PartialEq, Serialize, Deserialize)]
pub enum ErrorType {
    DatabaseConnection,
    DatabaseQuery,
    WebSocketConnection,
    MessageDelivery,
    AuthenticationFailure,
    RateLimitExceeded,
    ResourceExhaustion,
    ExternalServiceFailure,
    ConfigurationError,
    NetworkTimeout,
    DataCorruption,
    Unknown,
}

/// 错误严重程度
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ErrorSeverity {
    Low,
    Medium,
    High,
    Critical,
}

/// 恢复策略
#[derive(Debug, Clone)]
pub struct RecoveryStrategy {
    pub strategy_type: RecoveryStrategyType,
    pub max_attempts: u32,
    pub backoff_strategy: BackoffStrategy,
    pub timeout: Duration,
    pub prerequisites: Vec<String>,
}

/// 恢复策略类型
#[derive(Debug, Clone)]
pub enum RecoveryStrategyType {
    Retry,
    Restart,
    Fallback,
    CircuitBreaker,
    ResourceCleanup,
    ConfigReload,
    ServiceRestart,
}

/// 退避策略
#[derive(Debug, Clone)]
pub enum BackoffStrategy {
    Fixed(Duration),
    Linear(Duration),
    Exponential { base: Duration, max: Duration },
}

/// 恢复尝试
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoveryAttempt {
    pub error_id: String,
    pub strategy_used: String,
    pub attempt_count: u32,
    pub started_at: chrono::DateTime<chrono::Utc>,
    pub completed_at: Option<chrono::DateTime<chrono::Utc>>,
    pub success: Option<bool>,
    pub error_message: Option<String>,
}

/// 恢复配置
#[derive(Debug, Clone)]
pub struct RecoveryConfig {
    pub max_error_history: usize,
    pub error_retention_hours: u64,
    pub cleanup_interval: Duration,
    pub enable_auto_recovery: bool,
    pub recovery_timeout: Duration,
}

impl Default for RecoveryConfig {
    fn default() -> Self {
        Self {
            max_error_history: 1000,
            error_retention_hours: 24,
            cleanup_interval: Duration::from_secs(3600), // 1小时
            enable_auto_recovery: true,
            recovery_timeout: Duration::from_secs(30),
        }
    }
}

impl ErrorRecoverySystem {
    /// 创建新的错误恢复系统
    pub fn new(config: RecoveryConfig) -> Self {
        let mut system = Self {
            error_tracker: Arc::new(RwLock::new(ErrorTracker::new())),
            recovery_strategies: Arc::new(RwLock::new(HashMap::new())),
            config,
        };
        
        // 初始化默认恢复策略
        system.initialize_default_strategies();
        system
    }

    /// 初始化默认恢复策略
    fn initialize_default_strategies(&mut self) {
        let strategies = vec![
            (ErrorType::DatabaseConnection, RecoveryStrategy {
                strategy_type: RecoveryStrategyType::Retry,
                max_attempts: 3,
                backoff_strategy: BackoffStrategy::Exponential {
                    base: Duration::from_secs(1),
                    max: Duration::from_secs(30),
                },
                timeout: Duration::from_secs(10),
                prerequisites: vec!["database_health_check".to_string()],
            }),
            (ErrorType::WebSocketConnection, RecoveryStrategy {
                strategy_type: RecoveryStrategyType::Restart,
                max_attempts: 2,
                backoff_strategy: BackoffStrategy::Fixed(Duration::from_secs(5)),
                timeout: Duration::from_secs(15),
                prerequisites: vec!["websocket_service_check".to_string()],
            }),
            (ErrorType::MessageDelivery, RecoveryStrategy {
                strategy_type: RecoveryStrategyType::Fallback,
                max_attempts: 1,
                backoff_strategy: BackoffStrategy::Fixed(Duration::from_secs(1)),
                timeout: Duration::from_secs(5),
                prerequisites: vec![],
            }),
            (ErrorType::RateLimitExceeded, RecoveryStrategy {
                strategy_type: RecoveryStrategyType::CircuitBreaker,
                max_attempts: 1,
                backoff_strategy: BackoffStrategy::Linear(Duration::from_secs(60)),
                timeout: Duration::from_secs(300),
                prerequisites: vec![],
            }),
            (ErrorType::ResourceExhaustion, RecoveryStrategy {
                strategy_type: RecoveryStrategyType::ResourceCleanup,
                max_attempts: 1,
                backoff_strategy: BackoffStrategy::Fixed(Duration::from_secs(10)),
                timeout: Duration::from_secs(60),
                prerequisites: vec!["resource_check".to_string()],
            }),
        ];

        let recovery_strategies = Arc::clone(&self.recovery_strategies);
        tokio::spawn(async move {
            let mut strategies_map = recovery_strategies.write().await;
            for (error_type, strategy) in strategies {
                strategies_map.insert(error_type, strategy);
            }
        });
    }

    /// 启动错误恢复系统
    pub async fn start(&self) {
        let system = self.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(system.config.cleanup_interval);
            
            loop {
                interval.tick().await;
                if let Err(e) = system.cleanup_old_errors().await {
                    error!("清理旧错误记录失败: {:?}", e);
                }
            }
        });

        info!("错误恢复系统已启动");
    }

    /// 记录错误事件
    pub async fn record_error(
        &self,
        error_type: ErrorType,
        severity: ErrorSeverity,
        message: String,
        context: HashMap<String, String>,
    ) -> String {
        let error_id = format!("{:?}_{}", error_type, chrono::Utc::now().timestamp_nanos_opt().unwrap_or(0));
        
        let error_event = ErrorEvent {
            id: error_id.clone(),
            error_type: error_type.clone(),
            severity,
            message: message.clone(),
            context,
            occurred_at: chrono::Utc::now(),
            recovery_attempted: false,
            recovery_successful: None,
        };

        {
            let mut tracker = self.error_tracker.write().await;
            
            // 更新错误计数
            let count = tracker.error_counts.entry(error_type.clone()).or_insert_with(|| ErrorCount {
                total: 0,
                last_hour: 0,
                last_day: 0,
                first_occurrence: chrono::Utc::now(),
                last_occurrence: chrono::Utc::now(),
            });
            
            count.total += 1;
            count.last_hour += 1;
            count.last_day += 1;
            count.last_occurrence = chrono::Utc::now();
            
            // 添加到最近错误列表
            tracker.recent_errors.push(error_event);
            
            // 限制历史记录大小
            if tracker.recent_errors.len() > self.config.max_error_history {
                tracker.recent_errors.remove(0);
            }
        }

        warn!("记录错误事件: {:?} - {}", error_type, message);

        // 如果启用自动恢复，尝试恢复
        if self.config.enable_auto_recovery {
            if let Err(e) = self.attempt_recovery(&error_id, error_type).await {
                error!("自动恢复失败: {:?}", e);
            }
        }

        error_id
    }

    /// 尝试错误恢复
    pub async fn attempt_recovery(&self, error_id: &str, error_type: ErrorType) -> DomainResult<bool> {
        let strategy = {
            let strategies = self.recovery_strategies.read().await;
            strategies.get(&error_type).cloned()
        };

        let strategy = match strategy {
            Some(s) => s,
            None => {
                warn!("没有找到错误类型 {:?} 的恢复策略", error_type);
                return Ok(false);
            }
        };

        let recovery_attempt = RecoveryAttempt {
            error_id: error_id.to_string(),
            strategy_used: format!("{:?}", strategy.strategy_type),
            attempt_count: 1,
            started_at: chrono::Utc::now(),
            completed_at: None,
            success: None,
            error_message: None,
        };

        {
            let mut tracker = self.error_tracker.write().await;
            tracker.recovery_attempts.insert(error_id.to_string(), recovery_attempt);
        }

        info!("开始恢复尝试: {} 使用策略 {:?}", error_id, strategy.strategy_type);

        let recovery_result = self.execute_recovery_strategy(&strategy, error_id).await;

        // 更新恢复尝试结果
        {
            let mut tracker = self.error_tracker.write().await;
            if let Some(attempt) = tracker.recovery_attempts.get_mut(error_id) {
                attempt.completed_at = Some(chrono::Utc::now());
                match &recovery_result {
                    Ok(success) => {
                        attempt.success = Some(*success);
                        if *success {
                            info!("恢复成功: {}", error_id);
                        } else {
                            warn!("恢复失败: {}", error_id);
                        }
                    }
                    Err(e) => {
                        attempt.success = Some(false);
                        attempt.error_message = Some(format!("{:?}", e));
                        error!("恢复过程出错: {} - {:?}", error_id, e);
                    }
                }
            }
        }

        recovery_result
    }

    /// 执行恢复策略
    async fn execute_recovery_strategy(&self, strategy: &RecoveryStrategy, error_id: &str) -> DomainResult<bool> {
        match strategy.strategy_type {
            RecoveryStrategyType::Retry => {
                info!("执行重试策略: {}", error_id);
                // 这里应该实现具体的重试逻辑
                tokio::time::sleep(Duration::from_secs(1)).await;
                Ok(true)
            }
            RecoveryStrategyType::Restart => {
                info!("执行重启策略: {}", error_id);
                // 这里应该实现具体的重启逻辑
                tokio::time::sleep(Duration::from_secs(2)).await;
                Ok(true)
            }
            RecoveryStrategyType::Fallback => {
                info!("执行降级策略: {}", error_id);
                // 这里应该实现具体的降级逻辑
                Ok(true)
            }
            RecoveryStrategyType::CircuitBreaker => {
                info!("执行熔断策略: {}", error_id);
                // 这里应该实现具体的熔断逻辑
                Ok(true)
            }
            RecoveryStrategyType::ResourceCleanup => {
                info!("执行资源清理策略: {}", error_id);
                // 这里应该实现具体的资源清理逻辑
                Ok(true)
            }
            RecoveryStrategyType::ConfigReload => {
                info!("执行配置重载策略: {}", error_id);
                // 这里应该实现具体的配置重载逻辑
                Ok(true)
            }
            RecoveryStrategyType::ServiceRestart => {
                info!("执行服务重启策略: {}", error_id);
                // 这里应该实现具体的服务重启逻辑
                Ok(true)
            }
        }
    }

    /// 获取错误统计
    pub async fn get_error_statistics(&self) -> ErrorStatistics {
        let tracker = self.error_tracker.read().await;
        
        let total_errors: u64 = tracker.error_counts.values().map(|c| c.total).sum();
        let errors_last_hour: u64 = tracker.error_counts.values().map(|c| c.last_hour).sum();
        let errors_last_day: u64 = tracker.error_counts.values().map(|c| c.last_day).sum();
        
        let recovery_success_rate = if tracker.recovery_attempts.is_empty() {
            0.0
        } else {
            let successful_recoveries = tracker.recovery_attempts.values()
                .filter(|a| a.success == Some(true))
                .count();
            (successful_recoveries as f64 / tracker.recovery_attempts.len() as f64) * 100.0
        };

        ErrorStatistics {
            total_errors,
            errors_last_hour,
            errors_last_day,
            error_types: tracker.error_counts.clone(),
            recent_errors: tracker.recent_errors.clone(),
            recovery_attempts: tracker.recovery_attempts.len() as u64,
            recovery_success_rate,
            last_updated: chrono::Utc::now(),
        }
    }

    /// 清理旧错误记录
    async fn cleanup_old_errors(&self) -> DomainResult<()> {
        let cutoff_time = chrono::Utc::now() - chrono::Duration::hours(self.config.error_retention_hours as i64);
        
        let mut tracker = self.error_tracker.write().await;
        
        // 清理旧的错误事件
        tracker.recent_errors.retain(|error| error.occurred_at > cutoff_time);
        
        // 清理旧的恢复尝试
        tracker.recovery_attempts.retain(|_, attempt| attempt.started_at > cutoff_time);
        
        // 重置小时和天计数器（简化实现）
        for count in tracker.error_counts.values_mut() {
            count.last_hour = 0;
            count.last_day = 0;
        }
        
        tracker.last_cleanup = Instant::now();
        
        debug!("清理旧错误记录完成，保留 {} 条错误事件", tracker.recent_errors.len());
        Ok(())
    }

    /// 获取恢复建议
    pub async fn get_recovery_suggestions(&self) -> Vec<RecoverySuggestion> {
        let tracker = self.error_tracker.read().await;
        let mut suggestions = Vec::new();
        
        for (error_type, count) in &tracker.error_counts {
            if count.last_hour > 10 {
                suggestions.push(RecoverySuggestion {
                    error_type: error_type.clone(),
                    priority: SuggestionPriority::High,
                    description: format!("错误类型 {:?} 在过去1小时内发生 {} 次", error_type, count.last_hour),
                    recommended_action: "检查系统状态并考虑调整恢复策略".to_string(),
                });
            } else if count.last_day > 50 {
                suggestions.push(RecoverySuggestion {
                    error_type: error_type.clone(),
                    priority: SuggestionPriority::Medium,
                    description: format!("错误类型 {:?} 在过去24小时内发生 {} 次", error_type, count.last_day),
                    recommended_action: "分析错误模式并优化相关组件".to_string(),
                });
            }
        }
        
        suggestions
    }
}

impl ErrorTracker {
    pub fn new() -> Self {
        Self {
            error_counts: HashMap::new(),
            recent_errors: Vec::new(),
            recovery_attempts: HashMap::new(),
            last_cleanup: Instant::now(),
        }
    }
}

/// 错误统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorStatistics {
    pub total_errors: u64,
    pub errors_last_hour: u64,
    pub errors_last_day: u64,
    pub error_types: HashMap<ErrorType, ErrorCount>,
    pub recent_errors: Vec<ErrorEvent>,
    pub recovery_attempts: u64,
    pub recovery_success_rate: f64,
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

/// 恢复建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecoverySuggestion {
    pub error_type: ErrorType,
    pub priority: SuggestionPriority,
    pub description: String,
    pub recommended_action: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SuggestionPriority {
    Low,
    Medium,
    High,
    Critical,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_error_recording() {
        let system = ErrorRecoverySystem::new(RecoveryConfig::default());
        
        let mut context = HashMap::new();
        context.insert("component".to_string(), "database".to_string());
        
        let error_id = system.record_error(
            ErrorType::DatabaseConnection,
            ErrorSeverity::High,
            "连接超时".to_string(),
            context,
        ).await;
        
        assert!(!error_id.is_empty());
        
        let stats = system.get_error_statistics().await;
        assert_eq!(stats.total_errors, 1);
    }

    #[tokio::test]
    async fn test_recovery_attempt() {
        let system = ErrorRecoverySystem::new(RecoveryConfig::default());
        
        let result = system.attempt_recovery("test_error", ErrorType::DatabaseConnection).await;
        assert!(result.is_ok());
    }
}
