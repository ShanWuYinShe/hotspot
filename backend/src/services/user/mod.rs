//! 用户服务模块

// 导出子模块
pub mod service;

// 从子模块中重新导出具体实现
pub use service::{UserService, UserRow, AuthUserRow};

/// 用户服务错误类型
#[derive(Debug, thiserror::Error)]
#[allow(dead_code)]
pub enum UserServiceError {
    #[error("用户不存在")]
    UserNotFound,
    #[error("用户已存在")]
    UserAlreadyExists,
    #[error("密码错误")]
    InvalidPassword,
    #[error("令牌无效")]
    InvalidToken,
    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),
    #[error("仓库层错误: {0}")]
    Repository(#[from] crate::repositories::RepositoryError),
    #[error("配置错误: {0}")]
    Config(String),
    #[error("其他用户服务错误: {0}")]
    Other(String),
}

/// 用户服务结果
#[allow(dead_code)]
pub type UserServiceResult<T> = Result<T, UserServiceError>;
