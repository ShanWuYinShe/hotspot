/// 任务调度器端口定义
/// 
/// 定义异步任务处理相关的抽象接口

use async_trait::async_trait;
use crate::domain::DomainResult;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// 任务调度器端口
#[async_trait]
pub trait JobDispatcherPort: Send + Sync {
    /// 提交异步任务
    async fn dispatch_job<T>(&self, job: Job<T>) -> DomainResult<JobId>
    where
        T: JobPayload + Send + Sync;
    
    /// 获取任务状态
    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>>;
    
    /// 取消任务
    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool>;
    
    /// 获取队列统计信息
    async fn get_queue_stats(&self) -> DomainResult<QueueStats>;
    
    /// 重试失败的任务
    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool>;
}

/// 任务ID
pub type JobId = String;

/// 任务定义
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Job<T> {
    /// 任务ID
    pub id: JobId,
    /// 任务类型
    pub job_type: String,
    /// 任务载荷
    pub payload: T,
    /// 最大重试次数
    pub max_retries: u32,
    /// 重试延迟（秒）
    pub retry_delay_seconds: u64,
    /// 任务优先级
    pub priority: JobPriority,
    /// 任务元数据
    pub metadata: HashMap<String, String>,
}

impl<T> Job<T> {
    /// 创建新任务
    pub fn new(job_type: String, payload: T) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            job_type,
            payload,
            max_retries: 3,
            retry_delay_seconds: 5,
            priority: JobPriority::Normal,
            metadata: HashMap::new(),
        }
    }
    
    /// 设置重试配置
    pub fn with_retry(mut self, max_retries: u32, retry_delay_seconds: u64) -> Self {
        self.max_retries = max_retries;
        self.retry_delay_seconds = retry_delay_seconds;
        self
    }
    
    /// 设置优先级
    pub fn with_priority(mut self, priority: JobPriority) -> Self {
        self.priority = priority;
        self
    }
    
    /// 添加元数据
    pub fn with_metadata(mut self, key: String, value: String) -> Self {
        self.metadata.insert(key, value);
        self
    }
}

/// 任务载荷基础trait
pub trait JobPayload: Serialize + for<'de> Deserialize<'de> + Send + Sync + std::fmt::Debug {}

// 为所有满足条件的类型自动实现JobPayload
impl<T> JobPayload for T where T: Serialize + for<'de> Deserialize<'de> + Send + Sync + std::fmt::Debug {}

/// 任务优先级
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
pub enum JobPriority {
    Low = 1,
    Normal = 2,
    High = 3,
    Critical = 4,
}

/// 任务状态
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum JobStatus {
    /// 排队中
    Queued,
    /// 运行中
    Running,
    /// 已完成
    Completed { 
        completed_at: chrono::DateTime<chrono::Utc>,
        result: Option<String>,
    },
    /// 已失败
    Failed { 
        failed_at: chrono::DateTime<chrono::Utc>,
        error: String,
        retry_count: u32,
    },
    /// 已取消
    Cancelled {
        cancelled_at: chrono::DateTime<chrono::Utc>,
    },
}

/// 队列统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueueStats {
    /// 排队任务数量
    pub queued_jobs: u64,
    /// 运行中任务数量
    pub running_jobs: u64,
    /// 已完成任务数量
    pub completed_jobs: u64,
    /// 已失败任务数量
    pub failed_jobs: u64,
    /// 平均处理时间（毫秒）
    pub avg_processing_time_ms: u64,
}

/// 常用任务类型常量
pub mod job_types {
    pub const USER_WELCOME_EMAIL: &str = "user_welcome_email";
    pub const USER_LOCATION_INDEX: &str = "user_location_index";
    pub const GROUP_MEMBER_NOTIFICATION: &str = "group_member_notification";
    pub const MOMENT_CONTENT_INDEX: &str = "moment_content_index";
    pub const MESSAGE_NOTIFICATION: &str = "message_notification";
    pub const CLEANUP_EXPIRED_SESSIONS: &str = "cleanup_expired_sessions";
    pub const BACKUP_USER_DATA: &str = "backup_user_data";
}

