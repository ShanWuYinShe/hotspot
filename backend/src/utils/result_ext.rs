use crate::utils::response::AppError;
use crate::repositories::RepositoryError;
use tracing::error;

/// 扩展Result类型以提供更优雅的错误处理
pub trait ResultExt<T> {
    /// 将RepositoryError转换为AppError，并记录错误日志
    fn map_repository_error(self, operation: &str) -> Result<T, AppError>;
    
    /// 将RepositoryError转换为AppError，记录错误日志，并自定义错误消息
    fn map_repository_error_with_msg(self, operation: &str, user_msg: &str) -> Result<T, AppError>;
}

impl<T> ResultExt<T> for Result<T, RepositoryError> {
    fn map_repository_error(self, operation: &str) -> Result<T, AppError> {
        self.map_err(|e| {
            error!("{}失败: {:?}", operation, e);
            match e {
                RepositoryError::NotFound => AppError::NotFound(format!("{}失败: 资源不存在", operation)),
                RepositoryError::Database(_) => AppError::InternalServerError(format!("{}失败", operation)),
                RepositoryError::InvalidArgument(msg) => AppError::BadRequest(msg),
                RepositoryError::UuidParse(err) => AppError::BadRequest(format!("UUID格式错误: {}", err)),
                RepositoryError::Other(_) => AppError::InternalServerError(format!("{}失败", operation)),
            }
        })
    }
    
    fn map_repository_error_with_msg(self, operation: &str, user_msg: &str) -> Result<T, AppError> {
        self.map_err(|e| {
            error!("{}失败: {:?}", operation, e);
            match e {
                RepositoryError::NotFound => AppError::NotFound(user_msg.to_string()),
                RepositoryError::Database(_) => AppError::InternalServerError(user_msg.to_string()),
                RepositoryError::InvalidArgument(_) => AppError::BadRequest(user_msg.to_string()),
                RepositoryError::UuidParse(_) => AppError::BadRequest(user_msg.to_string()),
                RepositoryError::Other(_) => AppError::InternalServerError(user_msg.to_string()),
            }
        })
    }
}
