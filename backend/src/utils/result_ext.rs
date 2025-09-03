/// 结果扩展工具
/// 
/// 提供Result类型的扩展方法，兼容新架构

use crate::domain::{DomainError, DomainResult};
use crate::ports::repositories::RepoPortError;
use crate::utils::response::AppError;

/// 结果扩展trait
pub trait ResultExt<T> {
    /// 将结果映射为领域结果
    fn to_domain_result(self) -> DomainResult<T>;
    
    /// 将结果映射为领域结果，带自定义错误消息
    fn to_domain_result_with_message(self, message: &str) -> DomainResult<T>;
    
    /// 将仓库错误映射为应用错误（传统服务兼容性）
    fn map_repository_error(self, operation: &str) -> Result<T, AppError>
    where
        Self: Sized;
    
    /// 将仓库错误映射为应用错误，带自定义消息（传统服务兼容性）
    fn map_repository_error_with_msg(self, operation: &str, user_msg: &str) -> Result<T, AppError>
    where
        Self: Sized;
}

// 为 RepoPortError 提供专门的实现
impl<T> ResultExt<T> for Result<T, RepoPortError> {
    fn to_domain_result(self) -> DomainResult<T> {
        self.map_err(|e| match e {
            RepoPortError::NotFound(msg) => DomainError::BusinessRuleViolation {
                rule: format!("Resource not found: {}", msg),
            },
            RepoPortError::Database(db_err) => DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", db_err),
            },
            RepoPortError::InvalidArgument(msg) => DomainError::BusinessRuleViolation {
                rule: format!("Invalid argument: {}", msg),
            },
            RepoPortError::Other(msg) => DomainError::BusinessRuleViolation {
                rule: format!("Repository error: {}", msg),
            },
            RepoPortError::Connection(msg) => DomainError::BusinessRuleViolation {
                rule: format!("Connection error: {}", msg),
            },
            RepoPortError::Timeout(msg) => DomainError::BusinessRuleViolation {
                rule: format!("Timeout error: {}", msg),
            },
        })
    }
    
    fn to_domain_result_with_message(self, message: &str) -> DomainResult<T> {
        self.map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("{}: {}", message, e),
        })
    }
    
    fn map_repository_error(self, operation: &str) -> Result<T, AppError> {
        self.map_err(|e| match e {
            RepoPortError::NotFound(msg) => AppError::NotFound(format!("{}: {}", operation, msg)),
            RepoPortError::Database(db_err) => {
                tracing::error!("{}: 数据库错误: {}", operation, db_err);
                AppError::InternalServerError(format!("{}: 数据库操作失败", operation))
            },
            RepoPortError::InvalidArgument(msg) => AppError::BadRequest(format!("{}: {}", operation, msg)),
            RepoPortError::Other(msg) => {
                tracing::error!("{}: {}", operation, msg);
                AppError::InternalServerError(format!("{}: 操作失败", operation))
            },
            RepoPortError::Connection(msg) => {
                tracing::error!("{}: 连接错误: {}", operation, msg);
                AppError::InternalServerError(format!("{}: 连接失败", operation))
            },
            RepoPortError::Timeout(msg) => {
                tracing::error!("{}: 超时错误: {}", operation, msg);
                AppError::InternalServerError(format!("{}: 请求超时", operation))
            },
        })
    }
    
    fn map_repository_error_with_msg(self, operation: &str, user_msg: &str) -> Result<T, AppError> {
        self.map_err(|e| {
            tracing::error!("{}: {:?}", operation, e);
            match e {
                RepoPortError::NotFound(_) => AppError::NotFound(user_msg.to_string()),
                RepoPortError::InvalidArgument(_) => AppError::BadRequest(user_msg.to_string()),
                _ => AppError::InternalServerError(user_msg.to_string()),
            }
        })
    }
}



/// Option扩展trait
pub trait OptionExt<T> {
    /// 将Option转换为领域结果
    fn to_domain_result_or_not_found(self) -> DomainResult<T>;
    
    /// 将Option转换为领域结果，带自定义错误消息
    fn to_domain_result_or_error(self, message: &str) -> DomainResult<T>;
}

impl<T> OptionExt<T> for Option<T> {
    fn to_domain_result_or_not_found(self) -> DomainResult<T> {
        self.ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Resource not found".to_string(),
        })
    }
    
    fn to_domain_result_or_error(self, message: &str) -> DomainResult<T> {
        self.ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: message.to_string(),
        })
    }
}