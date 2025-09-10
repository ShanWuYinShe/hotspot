/// 结果扩展工具
/// 
/// 提供Result类型的扩展方法，兼容新架构
use crate::domain::{DomainError, DomainResult};
// RepoPortError 已移除，现在直接使用 DomainError
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

// 为 DomainError 提供专门的实现
impl<T> ResultExt<T> for DomainResult<T> {
    fn to_domain_result(self) -> DomainResult<T> {
        self
    }
    
    fn to_domain_result_with_message(self, message: &str) -> DomainResult<T> {
        self.map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("{}: {}", message, e),
        })
    }
    
    fn map_repository_error(self, operation: &str) -> Result<T, AppError> {
        self.map_err(|e| match e {
            DomainError::BusinessRuleViolation { rule } => {
                if rule.contains("not found") || rule.contains("未找到") {
                    AppError::NotFound(format!("{}: 资源未找到", operation))
                } else if rule.contains("invalid") || rule.contains("无效") {
                    AppError::BadRequest(format!("{}: {}", operation, rule))
                } else {
                    tracing::error!("{}: {}", operation, rule);
                    AppError::InternalServerError(format!("{}: 操作失败", operation))
                }
            },
            DomainError::ValidationError { field, reason } => {
                AppError::BadRequest(format!("{}: 字段 {} 验证失败: {}", operation, field, reason))
            },
            DomainError::NotFound { resource } => {
                AppError::NotFound(format!("{}: {} 未找到", operation, resource))
            },
            DomainError::Unauthorized { reason } => {
                AppError::Unauthorized(format!("{}: {}", operation, reason))
            },
            _ => {
                tracing::error!("{}: 未处理的领域错误: {:?}", operation, e);
                AppError::InternalServerError(format!("{}: 操作失败", operation))
            },
        })
    }
    
    fn map_repository_error_with_msg(self, operation: &str, user_msg: &str) -> Result<T, AppError> {
        self.map_err(|e| {
            tracing::error!("{}: {:?}", operation, e);
            match e {
                DomainError::NotFound { .. } => AppError::NotFound(user_msg.to_string()),
                DomainError::ValidationError { .. } => AppError::BadRequest(user_msg.to_string()),
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