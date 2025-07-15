//! 任务处理接口

use serde::{Serialize, de::DeserializeOwned};

use crate::utils::job_queue::Job;

/// 任务处理器错误
#[derive(Debug, thiserror::Error)]
pub enum JobHandlerError {
    #[error("仓库错误: {0}")]
    Repository(#[from] crate::repositories::RepositoryError),
    #[error("缓存错误: {0}")]
    Cache(#[from] crate::cache::CacheError),
    #[error("通知错误: {0}")]
    Notification(#[from] crate::services::notification::NotificationError),
    #[error("RabbitMQ错误: {0}")]
    RabbitMQ(#[from] lapin::Error),
    #[error("任务数据无效: {0}")]
    #[allow(dead_code)]
    InvalidJobData(String),
    #[error("服务不可用")]
    #[allow(dead_code)]
    ServiceUnavailable,
    #[error("其他错误: {0}")]
    Other(String),
}

/// 任务处理器结果
pub type JobHandlerResult = Result<(), JobHandlerError>;

/// 任务处理器接口
pub trait JobHandler<T>: Send + Sync
where
    T: Clone + Serialize + DeserializeOwned + Send + Sync + 'static,
{
    fn handle(&self, job: &Job<T>) -> impl std::future::Future<Output = JobHandlerResult> + Send;
}
