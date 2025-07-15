//! 消息服务模块

// 导出子模块
pub mod service;

// 从子模块中重新导出具体实现
pub use service::MessageService;

/// 消息服务错误类型
#[derive(Debug, thiserror::Error)]
#[allow(dead_code)]
pub enum MessageServiceError {
    #[error("消息不存在")]
    MessageNotFound,
    #[error("会话不存在")]
    ConversationNotFound,
    #[error("群组不存在")]
    GroupNotFound,
    #[error("用户无权限访问此会话")]
    UnauthorizedAccess,
    #[error("消息内容为空")]
    EmptyMessage,
    #[error("消息过长")]
    MessageTooLong,
    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),
    #[error("仓库层错误: {0}")]
    Repository(#[from] crate::repositories::RepositoryError),
    #[error("配置错误: {0}")]
    Config(String),
    #[error("其他消息服务错误: {0}")]
    Other(String),
}

/// 消息服务结果
#[allow(dead_code)]
pub type MessageServiceResult<T> = Result<T, MessageServiceError>;
