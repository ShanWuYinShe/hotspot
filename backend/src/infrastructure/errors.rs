/// 统一错误处理架构 - 分层错误转换和上下文传播
/// 
/// 错误处理原则：
/// - 分层错误转换
/// - 上下文信息保留
/// - 错误链追踪
/// - 结构化日志

use std::fmt;
use thiserror::Error;
use uuid::Uuid;

// === 基础设施层错误 ===

/// 基础设施层错误 - 技术实现相关的错误
#[derive(Error, Debug)]
pub enum InfrastructureError {
    // === 数据库错误 ===
    #[error("数据库连接错误: {message}")]
    DatabaseConnection { message: String },
    
    #[error("数据库查询错误: {query} - {message}")]
    DatabaseQuery { query: String, message: String },
    
    #[error("数据库事务错误: {message}")]
    DatabaseTransaction { message: String },
    
    #[error("数据库迁移错误: {message}")]
    DatabaseMigration { message: String },
    
    #[error("数据序列化错误: {message}")]
    Serialization { message: String },
    
    // === 缓存错误 ===
    #[error("缓存连接错误: {message}")]
    CacheConnection { message: String },
    
    #[error("缓存操作错误: {operation} - {message}")]
    CacheOperation { operation: String, message: String },
    
    #[error("缓存序列化错误: {message}")]
    CacheSerialization { message: String },
    
    // === 网络错误 ===
    #[error("HTTP请求错误: {url} - {status} - {message}")]
    HttpRequest { url: String, status: u16, message: String },
    
    #[error("网络超时: {operation} - {timeout_ms}ms")]
    NetworkTimeout { operation: String, timeout_ms: u64 },
    
    #[error("网络连接错误: {message}")]
    NetworkConnection { message: String },
    
    // === 文件系统错误 ===
    #[error("文件操作错误: {path} - {message}")]
    FileOperation { path: String, message: String },
    
    #[error("文件不存在: {path}")]
    FileNotFound { path: String },
    
    #[error("权限不足: {path}")]
    PermissionDenied { path: String },
    
    // === 外部服务错误 ===
    #[error("邮件服务错误: {message}")]
    EmailService { message: String },
    
    #[error("短信服务错误: {message}")]
    SmsService { message: String },
    
    #[error("推送服务错误: {message}")]
    PushService { message: String },
    
    #[error("地理位置服务错误: {message}")]
    GeolocationService { message: String },
    
    #[error("文件存储服务错误: {message}")]
    FileStorageService { message: String },
    
    // === 认证和授权错误 ===
    #[error("JWT令牌错误: {message}")]
    JwtToken { message: String },
    
    #[error("密码哈希错误: {message}")]
    PasswordHash { message: String },
    
    #[error("加密错误: {message}")]
    Encryption { message: String },
    
    // === 配置错误 ===
    #[error("配置错误: {key} - {message}")]
    Configuration { key: String, message: String },
    
    #[error("环境变量错误: {var} - {message}")]
    Environment { var: String, message: String },
    
    // === 任务队列错误 ===
    #[error("任务队列错误: {queue} - {message}")]
    JobQueue { queue: String, message: String },
    
    #[error("任务执行错误: {job_type} - {message}")]
    JobExecution { job_type: String, message: String },
    
    // === WebSocket错误 ===
    #[error("WebSocket连接错误: {message}")]
    WebSocketConnection { message: String },
    
    #[error("WebSocket消息错误: {message}")]
    WebSocketMessage { message: String },
    
    // === 通用错误 ===
    #[error("资源不可用: {resource}")]
    ResourceUnavailable { resource: String },
    
    #[error("操作超时: {operation}")]
    OperationTimeout { operation: String },
    
    #[error("内部错误: {message}")]
    Internal { message: String },
}

// === 错误上下文 ===

/// 错误上下文 - 提供错误发生时的上下文信息
#[derive(Debug, Clone)]
pub struct ErrorContext {
    /// 错误ID - 用于追踪和关联
    pub error_id: String,
    /// 操作名称
    pub operation: String,
    /// 用户ID（如果有）
    pub user_id: Option<String>,
    /// 请求ID（如果有）
    pub request_id: Option<String>,
    /// 会话ID（如果有）
    pub session_id: Option<String>,
    /// 时间戳
    pub timestamp: chrono::DateTime<chrono::Utc>,
    /// 额外的上下文数据
    pub metadata: std::collections::HashMap<String, String>,
}

impl ErrorContext {
    /// 创建新的错误上下文
    pub fn new(operation: &str) -> Self {
        Self {
            error_id: Uuid::new_v4().to_string(),
            operation: operation.to_string(),
            user_id: None,
            request_id: None,
            session_id: None,
            timestamp: chrono::Utc::now(),
            metadata: std::collections::HashMap::new(),
        }
    }
    
    /// 设置用户ID
    pub fn with_user_id(mut self, user_id: &str) -> Self {
        self.user_id = Some(user_id.to_string());
        self
    }
    
    /// 设置请求ID
    pub fn with_request_id(mut self, request_id: &str) -> Self {
        self.request_id = Some(request_id.to_string());
        self
    }
    
    /// 设置会话ID
    pub fn with_session_id(mut self, session_id: &str) -> Self {
        self.session_id = Some(session_id.to_string());
        self
    }
    
    /// 添加元数据
    pub fn with_metadata(mut self, key: &str, value: &str) -> Self {
        self.metadata.insert(key.to_string(), value.to_string());
        self
    }
    
    /// 批量添加元数据
    pub fn with_metadata_map(mut self, metadata: std::collections::HashMap<String, String>) -> Self {
        self.metadata.extend(metadata);
        self
    }
}

// === 带上下文的错误 ===

/// 带上下文的错误 - 包含错误和上下文信息
#[derive(Debug)]
pub struct ContextualError {
    pub error: InfrastructureError,
    pub context: ErrorContext,
    pub source_chain: Vec<String>,
}

impl ContextualError {
    /// 创建新的上下文错误
    pub fn new(error: InfrastructureError, context: ErrorContext) -> Self {
        Self {
            error,
            context,
            source_chain: Vec::new(),
        }
    }
    
    /// 添加源错误到链中
    pub fn with_source(mut self, source: &str) -> Self {
        self.source_chain.push(source.to_string());
        self
    }
    
    /// 获取错误链的字符串表示
    pub fn error_chain(&self) -> String {
        if self.source_chain.is_empty() {
            self.error.to_string()
        } else {
            format!("{} <- {}", self.error, self.source_chain.join(" <- "))
        }
    }
    
    /// 转换为结构化日志
    pub fn to_structured_log(&self) -> serde_json::Value {
        serde_json::json!({
            "error_id": self.context.error_id,
            "error_type": self.error_type(),
            "error_message": self.error.to_string(),
            "error_chain": self.error_chain(),
            "operation": self.context.operation,
            "user_id": self.context.user_id,
            "request_id": self.context.request_id,
            "session_id": self.context.session_id,
            "timestamp": self.context.timestamp.to_rfc3339(),
            "metadata": self.context.metadata,
        })
    }
    
    /// 获取错误类型
    pub fn error_type(&self) -> &'static str {
        match &self.error {
            InfrastructureError::DatabaseConnection { .. } => "database_connection",
            InfrastructureError::DatabaseQuery { .. } => "database_query",
            InfrastructureError::DatabaseTransaction { .. } => "database_transaction",
            InfrastructureError::DatabaseMigration { .. } => "database_migration",
            InfrastructureError::Serialization { .. } => "serialization",
            InfrastructureError::CacheConnection { .. } => "cache_connection",
            InfrastructureError::CacheOperation { .. } => "cache_operation",
            InfrastructureError::CacheSerialization { .. } => "cache_serialization",
            InfrastructureError::HttpRequest { .. } => "http_request",
            InfrastructureError::NetworkTimeout { .. } => "network_timeout",
            InfrastructureError::NetworkConnection { .. } => "network_connection",
            InfrastructureError::FileOperation { .. } => "file_operation",
            InfrastructureError::FileNotFound { .. } => "file_not_found",
            InfrastructureError::PermissionDenied { .. } => "permission_denied",
            InfrastructureError::EmailService { .. } => "email_service",
            InfrastructureError::SmsService { .. } => "sms_service",
            InfrastructureError::PushService { .. } => "push_service",
            InfrastructureError::GeolocationService { .. } => "geolocation_service",
            InfrastructureError::FileStorageService { .. } => "file_storage_service",
            InfrastructureError::JwtToken { .. } => "jwt_token",
            InfrastructureError::PasswordHash { .. } => "password_hash",
            InfrastructureError::Encryption { .. } => "encryption",
            InfrastructureError::Configuration { .. } => "configuration",
            InfrastructureError::Environment { .. } => "environment",
            InfrastructureError::JobQueue { .. } => "job_queue",
            InfrastructureError::JobExecution { .. } => "job_execution",
            InfrastructureError::WebSocketConnection { .. } => "websocket_connection",
            InfrastructureError::WebSocketMessage { .. } => "websocket_message",
            InfrastructureError::ResourceUnavailable { .. } => "resource_unavailable",
            InfrastructureError::OperationTimeout { .. } => "operation_timeout",
            InfrastructureError::Internal { .. } => "internal",
        }
    }
}

impl fmt::Display for ContextualError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "[{}] {}: {}", self.context.error_id, self.context.operation, self.error_chain())
    }
}

impl std::error::Error for ContextualError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        Some(&self.error)
    }
}

// === 错误转换器 ===

/// 错误转换器 - 将各种错误转换为基础设施错误
pub struct ErrorConverter;

impl ErrorConverter {
    /// 转换sqlx错误
    pub fn from_sqlx_error(error: sqlx::Error, query: Option<&str>) -> InfrastructureError {
        match error {
            sqlx::Error::Database(db_err) => {
                InfrastructureError::DatabaseQuery {
                    query: query.unwrap_or("unknown").to_string(),
                    message: db_err.to_string(),
                }
            }
            sqlx::Error::Io(io_err) => {
                InfrastructureError::DatabaseConnection {
                    message: io_err.to_string(),
                }
            }
            sqlx::Error::PoolTimedOut => {
                InfrastructureError::DatabaseConnection {
                    message: "Connection pool timed out".to_string(),
                }
            }
            sqlx::Error::PoolClosed => {
                InfrastructureError::DatabaseConnection {
                    message: "Connection pool is closed".to_string(),
                }
            }
            _ => {
                InfrastructureError::DatabaseQuery {
                    query: query.unwrap_or("unknown").to_string(),
                    message: error.to_string(),
                }
            }
        }
    }
    
    /// 转换serde错误
    pub fn from_serde_error(error: serde_json::Error) -> InfrastructureError {
        InfrastructureError::Serialization {
            message: error.to_string(),
        }
    }
    
    /// 转换reqwest错误
    pub fn from_reqwest_error(error: reqwest::Error, url: Option<&str>) -> InfrastructureError {
        if error.is_timeout() {
            InfrastructureError::NetworkTimeout {
                operation: format!("HTTP request to {}", url.unwrap_or("unknown")),
                timeout_ms: 30000, // 默认30秒
            }
        } else if error.is_connect() {
            InfrastructureError::NetworkConnection {
                message: error.to_string(),
            }
        } else if let Some(status_code) = error.status() {
            InfrastructureError::HttpRequest {
                url: url.unwrap_or("unknown").to_string(),
                status: status_code.as_u16(),
                message: error.to_string(),
            }
        } else {
            InfrastructureError::NetworkConnection {
                message: error.to_string(),
            }
        }
    }
    
    /// 转换IO错误
    pub fn from_io_error(error: std::io::Error, path: Option<&str>) -> InfrastructureError {
        match error.kind() {
            std::io::ErrorKind::NotFound => {
                InfrastructureError::FileNotFound {
                    path: path.unwrap_or("unknown").to_string(),
                }
            }
            std::io::ErrorKind::PermissionDenied => {
                InfrastructureError::PermissionDenied {
                    path: path.unwrap_or("unknown").to_string(),
                }
            }
            _ => {
                InfrastructureError::FileOperation {
                    path: path.unwrap_or("unknown").to_string(),
                    message: error.to_string(),
                }
            }
        }
    }
}

// === 错误处理宏 ===

/// 创建上下文错误的宏
#[macro_export]
macro_rules! contextual_error {
    ($error:expr, $operation:expr) => {
        ContextualError::new($error, ErrorContext::new($operation))
    };
    ($error:expr, $operation:expr, $($key:expr => $value:expr),*) => {
        {
            let mut context = ErrorContext::new($operation);
            $(
                context = context.with_metadata($key, $value);
            )*
            ContextualError::new($error, context)
        }
    };
}

/// 错误链追踪宏
#[macro_export]
macro_rules! error_chain {
    ($result:expr, $operation:expr) => {
        $result.map_err(|e| {
            let infra_error = match e {
                e if e.is::<sqlx::Error>() => {
                    ErrorConverter::from_sqlx_error(e.downcast::<sqlx::Error>().unwrap(), Some($operation))
                }
                e if e.is::<serde_json::Error>() => {
                    ErrorConverter::from_serde_error(e.downcast::<serde_json::Error>().unwrap())
                }
                _ => InfrastructureError::Internal { message: e.to_string() }
            };
            contextual_error!(infra_error, $operation)
        })
    };
}

// === 错误恢复策略 ===

/// 错误恢复策略
#[derive(Debug, Clone)]
pub enum RecoveryStrategy {
    /// 不重试
    NoRetry,
    /// 固定间隔重试
    FixedInterval { max_attempts: u32, interval_ms: u64 },
    /// 指数退避重试
    ExponentialBackoff { max_attempts: u32, base_delay_ms: u64, max_delay_ms: u64 },
    /// 自定义重试策略
    Custom { strategy: String },
}

/// 错误恢复器
pub struct ErrorRecovery;

impl ErrorRecovery {
    /// 检查错误是否可以重试
    pub fn is_retryable(error: &InfrastructureError) -> bool {
        match error {
            // 网络相关错误通常可以重试
            InfrastructureError::NetworkTimeout { .. } |
            InfrastructureError::NetworkConnection { .. } => true,
            InfrastructureError::HttpRequest { status, .. } if *status >= 500 => true,
            
            // 数据库连接错误可以重试
            InfrastructureError::DatabaseConnection { .. } => true,
            
            // 缓存连接错误可以重试
            InfrastructureError::CacheConnection { .. } => true,
            
            // 外部服务错误可以重试
            InfrastructureError::EmailService { .. } |
            InfrastructureError::SmsService { .. } |
            InfrastructureError::PushService { .. } => true,
            
            // 其他错误通常不可重试
            _ => false,
        }
    }
    
    /// 获取推荐的恢复策略
    pub fn recommended_strategy(error: &InfrastructureError) -> RecoveryStrategy {
        match error {
            // 网络错误使用指数退避
            InfrastructureError::NetworkTimeout { .. } |
            InfrastructureError::NetworkConnection { .. } => {
                RecoveryStrategy::ExponentialBackoff {
                    max_attempts: 3,
                    base_delay_ms: 1000,
                    max_delay_ms: 10000,
                }
            }
            
            // 数据库连接错误使用固定间隔
            InfrastructureError::DatabaseConnection { .. } => {
                RecoveryStrategy::FixedInterval {
                    max_attempts: 5,
                    interval_ms: 2000,
                }
            }
            
            // 外部服务错误使用指数退避
            InfrastructureError::EmailService { .. } |
            InfrastructureError::SmsService { .. } |
            InfrastructureError::PushService { .. } => {
                RecoveryStrategy::ExponentialBackoff {
                    max_attempts: 3,
                    base_delay_ms: 500,
                    max_delay_ms: 5000,
                }
            }
            
            // 其他错误不重试
            _ => RecoveryStrategy::NoRetry,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_context_creation() {
        let context = ErrorContext::new("test_operation")
            .with_user_id("user123")
            .with_request_id("req456")
            .with_metadata("key1", "value1");
        
        assert_eq!(context.operation, "test_operation");
        assert_eq!(context.user_id, Some("user123".to_string()));
        assert_eq!(context.request_id, Some("req456".to_string()));
        assert_eq!(context.metadata.get("key1"), Some(&"value1".to_string()));
    }
    
    #[test]
    fn test_contextual_error() {
        let error = InfrastructureError::DatabaseConnection {
            message: "Connection failed".to_string(),
        };
        let context = ErrorContext::new("database_operation");
        let contextual = ContextualError::new(error, context);
        
        assert_eq!(contextual.error_type(), "database_connection");
        assert!(contextual.to_string().contains("database_operation"));
    }
    
    #[test]
    fn test_error_recovery_strategy() {
        let network_error = InfrastructureError::NetworkTimeout {
            operation: "test".to_string(),
            timeout_ms: 5000,
        };
        
        assert!(ErrorRecovery::is_retryable(&network_error));
        
        let strategy = ErrorRecovery::recommended_strategy(&network_error);
        assert!(matches!(strategy, RecoveryStrategy::ExponentialBackoff { .. }));
    }
}
