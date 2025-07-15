//! 群组服务模块

// 导出子模块
pub mod service;

// 从子模块中重新导出具体实现
pub use service::{GroupService, GroupSearchResult};

/// 群组服务错误类型
#[derive(Debug, thiserror::Error)]
#[allow(dead_code)]
pub enum GroupServiceError {
    #[error("群组不存在")]
    GroupNotFound,
    #[error("群组已存在")]
    GroupAlreadyExists,
    #[error("用户不是群组成员")]
    UserNotMember,
    #[error("权限不足")]
    InsufficientPermissions,
    #[error("群组已满")]
    GroupFull,
    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),
    #[error("仓库层错误: {0}")]
    Repository(#[from] crate::repositories::RepositoryError),
    #[error("配置错误: {0}")]
    Config(String),
    #[error("其他群组服务错误: {0}")]
    Other(String),
}

/// 群组服务结果
#[allow(dead_code)]
pub type GroupServiceResult<T> = Result<T, GroupServiceError>;
