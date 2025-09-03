// 更结构化的错误处理模块

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::{Serialize, Deserialize};
use thiserror::Error;
use tracing::{error, warn};

// use crate::utils::response::ApiResponse;

/// 错误追踪信息
#[derive(Debug, Clone)]
pub struct ErrorContext {
    pub operation: String,
    pub user_id: Option<i64>,
    pub request_id: Option<String>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub source_location: Option<String>,
    pub additional_data: std::collections::HashMap<String, String>,
}

impl ErrorContext {
    pub fn new(operation: impl Into<String>) -> Self {
        Self {
            operation: operation.into(),
            user_id: None,
            request_id: None,
            timestamp: chrono::Utc::now(),
            source_location: None,
            additional_data: std::collections::HashMap::new(),
        }
    }
    
    pub fn with_user_id(mut self, user_id: i64) -> Self {
        self.user_id = Some(user_id);
        self
    }
    
    pub fn with_request_id(mut self, request_id: impl Into<String>) -> Self {
        self.request_id = Some(request_id.into());
        self
    }
    
    pub fn with_source_location(mut self, location: impl Into<String>) -> Self {
        self.source_location = Some(location.into());
        self
    }
    
    pub fn add_data(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.additional_data.insert(key.into(), value.into());
        self
    }
}

// 使用thiserror简化错误定义
#[derive(Error, Debug)]
pub enum AppError {
    #[error("资源不存在: {0}")]
    NotFound(String),
    
    #[error("请求参数错误: {0}")]
    BadRequest(String),
    
    #[error("服务器内部错误: {0}")]
    InternalServerError(String),
    
    #[error("未授权: {0}")]
    Unauthorized(String),
    
    #[error("权限不足: {0}")]
    Forbidden(String),
    
    #[error("资源冲突: {0}")]
    Conflict(String),
    
    #[error("数据库错误: {0}")]
    DatabaseError(String),
    
    #[error("第三方服务错误: {0}")]
    ExternalServiceError(String),
    
    #[error("功能未实现: {0}")]
    #[allow(dead_code)]
    NotImplemented(String),
    
    #[error("验证失败: {0}")]
    ValidationError(String),
    
    #[error("超时: {0}")]
    Timeout(String),
}

// 将具体的错误映射到AppError
impl From<sqlx::Error> for AppError {
    fn from(err: sqlx::Error) -> Self {
        error!("数据库错误: {:?}", err);
        match err {
            sqlx::Error::RowNotFound => Self::NotFound("未找到请求的资源".into()),
            _ => Self::DatabaseError(format!("数据库操作失败: {}", err)),
        }
    }
}

impl From<crate::ports::repositories::RepoPortError> for AppError {
    fn from(err: crate::ports::repositories::RepoPortError) -> Self {
        error!("仓库错误: {:?}", err);
        match err {
            crate::ports::repositories::RepoPortError::NotFound(msg) => Self::NotFound(msg),
            crate::ports::repositories::RepoPortError::Database(db_err) => Self::DatabaseError(format!("数据库操作失败: {}", db_err)),
            crate::ports::repositories::RepoPortError::InvalidArgument(msg) => Self::BadRequest(msg),
            crate::ports::repositories::RepoPortError::Connection(msg) => Self::InternalServerError(format!("连接错误: {}", msg)),
            crate::ports::repositories::RepoPortError::Timeout(msg) => Self::Timeout(format!("超时: {}", msg)),
            crate::ports::repositories::RepoPortError::Other(msg) => Self::InternalServerError(msg),
        }
    }
}

impl From<redis::RedisError> for AppError {
    fn from(err: redis::RedisError) -> Self {
        error!("Redis错误: {:?}", err);
        Self::ExternalServiceError(format!("Redis操作失败: {}", err))
    }
}

impl From<std::io::Error> for AppError {
    fn from(err: std::io::Error) -> Self {
        error!("IO错误: {:?}", err);
        Self::InternalServerError(format!("IO操作失败: {}", err))
    }
}

impl From<crate::presentation::http::validators::common_validator::ValidationError> for AppError {
    fn from(err: crate::presentation::http::validators::common_validator::ValidationError) -> Self {
        Self::BadRequest(err.to_string())
    }
}

// 注释掉已迁移的仓库错误转换
// impl From<crate::repositories::RepositoryError> for AppError {
//     fn from(err: crate::repositories::RepositoryError) -> Self {
//         error!("仓库错误: {:?}", err);
//         match err {
//             crate::repositories::RepositoryError::NotFound => Self::NotFound("请求的资源不存在".into()),
//             crate::repositories::RepositoryError::Database(e) => Self::DatabaseError(format!("数据库操作失败: {}", e)),
//             crate::repositories::RepositoryError::InvalidArgument(msg) => Self::BadRequest(msg),
//             crate::repositories::RepositoryError::UuidParse(err) => Self::BadRequest(format!("UUID格式错误: {}", err)),
//             crate::repositories::RepositoryError::Other(msg) => Self::InternalServerError(msg),
//         }
//     }
// }

// 用于在错误响应中返回的结构
#[derive(Debug, Serialize, Deserialize)]
pub struct ErrorDetail {
    pub code: i32,
    pub message: String,
    pub request_id: Option<String>,
    pub details: Option<Vec<String>>,
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        // 为每个错误记录追踪信息
        let (status_code, error_code, error_message) = match &self {
            AppError::NotFound(message) => {
                warn!("资源不存在: {}", message);
                (StatusCode::OK, 1004, message.clone())
            },
            AppError::BadRequest(message) => {
                warn!("请求参数错误: {}", message);
                (StatusCode::OK, 1000, message.clone())
            },
            AppError::InternalServerError(message) => {
                error!("服务器内部错误: {}", message);
                (StatusCode::OK, 5000, message.clone())
            },
            AppError::Unauthorized(message) => {
                warn!("未授权: {}", message);
                (StatusCode::OK, 1002, message.clone())
            },
            AppError::Forbidden(message) => {
                warn!("权限不足: {}", message);
                (StatusCode::OK, 1003, message.clone())
            },
            AppError::Conflict(message) => {
                warn!("资源冲突: {}", message);
                (StatusCode::OK, 1005, message.clone())
            },
            AppError::DatabaseError(message) => {
                error!("数据库错误: {}", message);
                (StatusCode::OK, 5001, "数据库操作失败".into())
            },
            AppError::ExternalServiceError(message) => {
                error!("第三方服务错误: {}", message);
                (StatusCode::OK, 5002, "外部服务调用失败".into())
            },
            AppError::NotImplemented(message) => {
                warn!("功能未实现: {}", message);
                (StatusCode::OK, 1006, message.clone())
            },
            AppError::ValidationError(message) => {
                warn!("验证失败: {}", message);
                (StatusCode::OK, 1001, message.clone())
            },
            AppError::Timeout(message) => {
                warn!("请求超时: {}", message);
                (StatusCode::OK, 1007, message.clone())
            },
        };

        // 创建结构化的错误响应
        let body = Json(serde_json::json!({
            "code": error_code,
            "msg": error_message,
            "data": null
        }));

        // 统一返回HTTP状态码200，将实际错误码放在body中
        (status_code, body).into_response()
    }
}

impl AppError {
    /// 创建带上下文的错误
    pub fn with_context(self, operation: &str, user_id: Option<i64>) -> Self {
        let context = ErrorContext {
            operation: operation.to_string(),
            user_id,
            request_id: None, // 可以从请求头中获取
            timestamp: chrono::Utc::now(),
            source_location: None,
            additional_data: std::collections::HashMap::new(),
        };
        
        // 记录错误上下文
        match &self {
            AppError::InternalServerError(_) | AppError::DatabaseError(_) => {
                error!("错误上下文: operation={}, user_id={:?}, timestamp={}", 
                       context.operation, context.user_id, context.timestamp);
            },
            _ => {
                warn!("错误上下文: operation={}, user_id={:?}, timestamp={}", 
                      context.operation, context.user_id, context.timestamp);
            }
        }
        
        self
    }
    
    /// 从验证错误创建
    pub fn validation_error(field: &str, message: &str) -> Self {
        Self::ValidationError(format!("{}: {}", field, message))
    }
    
    /// 从超时创建
    pub fn timeout_error(operation: &str, duration_ms: u64) -> Self {
        Self::Timeout(format!("{}超时，耗时{}ms", operation, duration_ms))
    }
}
