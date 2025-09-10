/// 任务队列接口定义 - 异步任务处理抽象
/// 
/// 任务特征：
/// - 异步执行
/// - 重试机制
/// - 优先级队列
/// - 状态跟踪

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::time::Duration;
use uuid::Uuid;
use crate::domain::DomainResult;

// === 任务接口 ===

/// 任务接口
pub trait Job: Send + Sync + Clone + Serialize + for<'de> Deserialize<'de> {
    /// 获取任务类型
    fn job_type(&self) -> &'static str;
    
    /// 获取任务ID
    fn job_id(&self) -> JobId;
    
    /// 获取任务优先级
    fn priority(&self) -> JobPriority {
        JobPriority::Normal
    }
    
    /// 获取最大重试次数
    fn max_retries(&self) -> u32 {
        3
    }
    
    /// 获取任务超时时间
    fn timeout(&self) -> Duration {
        Duration::from_secs(300) // 5分钟默认超时
    }
    
    /// 获取延迟执行时间
    fn delay(&self) -> Option<Duration> {
        None
    }
}

/// 任务ID
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct JobId(pub Uuid);

impl JobId {
    /// 创建新的任务ID
    pub fn new() -> Self {
        Self(Uuid::new_v4())
    }
    
    /// 从UUID创建
    pub fn from_uuid(uuid: Uuid) -> Self {
        Self(uuid)
    }
    
    /// 获取UUID值
    pub fn value(&self) -> Uuid {
        self.0
    }
}

impl Default for JobId {
    fn default() -> Self {
        Self::new()
    }
}

impl std::fmt::Display for JobId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 任务优先级
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
pub enum JobPriority {
    Low = 1,
    Normal = 2,
    High = 3,
    Critical = 4,
}

/// 任务状态
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum JobStatus {
    Pending,    // 等待执行
    Running,    // 正在执行
    Completed,  // 执行完成
    Failed,     // 执行失败
    Cancelled,  // 已取消
    Retrying,   // 重试中
}

/// 任务元数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JobMetadata {
    pub job_id: JobId,
    pub job_type: String,
    pub priority: JobPriority,
    pub status: JobStatus,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub scheduled_at: chrono::DateTime<chrono::Utc>,
    pub started_at: Option<chrono::DateTime<chrono::Utc>>,
    pub completed_at: Option<chrono::DateTime<chrono::Utc>>,
    pub retry_count: u32,
    pub max_retries: u32,
    pub timeout: Duration,
    pub error_message: Option<String>,
    pub worker_id: Option<String>,
}

// === 任务处理器接口 ===

/// 任务处理器接口
#[async_trait]
pub trait JobHandler<T>: Send + Sync
where
    T: Job,
{
    /// 处理任务
    async fn handle(&self, job: &T) -> DomainResult<JobResult>;
    
    /// 获取处理器名称
    fn handler_name(&self) -> &'static str;
    
    /// 检查是否可以处理该任务类型
    fn can_handle(&self, job_type: &str) -> bool;
    
    /// 任务执行前的钩子
    async fn before_handle(&self, _job: &T) -> DomainResult<()> {
        Ok(())
    }
    
    /// 任务执行后的钩子
    async fn after_handle(&self, _job: &T, _result: &JobResult) -> DomainResult<()> {
        Ok(())
    }
    
    /// 任务失败时的钩子
    async fn on_failure(&self, _job: &T, _error: &str) -> DomainResult<()> {
        Ok(())
    }
}

/// 任务执行结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum JobResult {
    /// 成功完成
    Success {
        message: Option<String>,
        data: Option<serde_json::Value>,
    },
    /// 失败，不重试
    Failure {
        error: String,
        permanent: bool,
    },
    /// 需要重试
    Retry {
        error: String,
        delay: Option<Duration>,
    },
}

impl JobResult {
    /// 创建成功结果
    pub fn success() -> Self {
        Self::Success {
            message: None,
            data: None,
        }
    }
    
    /// 创建带消息的成功结果
    pub fn success_with_message(message: &str) -> Self {
        Self::Success {
            message: Some(message.to_string()),
            data: None,
        }
    }
    
    /// 创建带数据的成功结果
    pub fn success_with_data(data: serde_json::Value) -> Self {
        Self::Success {
            message: None,
            data: Some(data),
        }
    }
    
    /// 创建永久失败结果
    pub fn permanent_failure(error: &str) -> Self {
        Self::Failure {
            error: error.to_string(),
            permanent: true,
        }
    }
    
    /// 创建临时失败结果
    pub fn temporary_failure(error: &str) -> Self {
        Self::Failure {
            error: error.to_string(),
            permanent: false,
        }
    }
    
    /// 创建重试结果
    pub fn retry(error: &str) -> Self {
        Self::Retry {
            error: error.to_string(),
            delay: None,
        }
    }
    
    /// 创建延迟重试结果
    pub fn retry_with_delay(error: &str, delay: Duration) -> Self {
        Self::Retry {
            error: error.to_string(),
            delay: Some(delay),
        }
    }
}

// === 任务队列接口 ===

/// 任务队列接口
#[async_trait]
pub trait JobQueue: Send + Sync + Clone {
    /// 提交任务
    async fn enqueue<T>(&self, job: T) -> DomainResult<JobId>
    where
        T: Job;
    
    /// 批量提交任务
    async fn enqueue_batch<T>(&self, jobs: Vec<T>) -> DomainResult<Vec<JobId>>
    where
        T: Job;
    
    /// 延迟提交任务
    async fn enqueue_delayed<T>(&self, job: T, delay: Duration) -> DomainResult<JobId>
    where
        T: Job;
    
    /// 定时提交任务
    async fn enqueue_at<T>(&self, job: T, scheduled_at: chrono::DateTime<chrono::Utc>) -> DomainResult<JobId>
    where
        T: Job;
    
    /// 取消任务
    async fn cancel_job(&self, job_id: JobId) -> DomainResult<bool>;
    
    /// 获取任务状态
    async fn get_job_status(&self, job_id: JobId) -> DomainResult<Option<JobMetadata>>;
    
    /// 获取队列统计
    async fn get_queue_stats(&self) -> DomainResult<QueueStats>;
    
    /// 清理已完成的任务
    async fn cleanup_completed_jobs(&self, older_than: Duration) -> DomainResult<u64>;
}

/// 队列统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueueStats {
    pub pending_jobs: u64,
    pub running_jobs: u64,
    pub completed_jobs: u64,
    pub failed_jobs: u64,
    pub cancelled_jobs: u64,
    pub total_jobs: u64,
    pub active_workers: u32,
    pub queue_size: u64,
    pub average_processing_time: Duration,
    pub last_processed_at: Option<chrono::DateTime<chrono::Utc>>,
}

// === 任务工作器接口 ===

/// 任务工作器接口
#[async_trait]
pub trait JobWorker: Send + Sync {
    /// 启动工作器
    async fn start(&self) -> DomainResult<()>;
    
    /// 停止工作器
    async fn stop(&self) -> DomainResult<()>;
    
    /// 获取工作器状态
    async fn get_status(&self) -> DomainResult<WorkerStatus>;
    
    /// 注册任务处理器
    async fn register_handler<T>(&self, handler: Box<dyn JobHandler<T>>) -> DomainResult<()>
    where
        T: Job + 'static;
    
    /// 处理下一个任务
    async fn process_next_job(&self) -> DomainResult<Option<JobMetadata>>;
    
    /// 获取工作器统计
    async fn get_worker_stats(&self) -> DomainResult<WorkerStats>;
}

/// 工作器状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkerStatus {
    pub worker_id: String,
    pub is_running: bool,
    pub current_job: Option<JobId>,
    pub started_at: Option<chrono::DateTime<chrono::Utc>>,
    pub last_heartbeat: chrono::DateTime<chrono::Utc>,
    pub registered_handlers: Vec<String>,
}

/// 工作器统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkerStats {
    pub worker_id: String,
    pub processed_jobs: u64,
    pub successful_jobs: u64,
    pub failed_jobs: u64,
    pub average_job_time: Duration,
    pub uptime: Duration,
    pub memory_usage: u64,
    pub cpu_usage: f64,
}

// === 任务调度器接口 ===

/// 任务调度器接口
#[async_trait]
pub trait JobScheduler: Send + Sync + Clone {
    /// 添加定时任务
    async fn schedule_recurring<T>(&self, job: T, cron_expression: &str) -> DomainResult<ScheduleId>
    where
        T: Job;
    
    /// 添加间隔任务
    async fn schedule_interval<T>(&self, job: T, interval: Duration) -> DomainResult<ScheduleId>
    where
        T: Job;
    
    /// 取消定时任务
    async fn cancel_schedule(&self, schedule_id: ScheduleId) -> DomainResult<bool>;
    
    /// 获取调度状态
    async fn get_schedule_status(&self, schedule_id: ScheduleId) -> DomainResult<Option<ScheduleStatus>>;
    
    /// 获取所有调度
    async fn list_schedules(&self) -> DomainResult<Vec<ScheduleStatus>>;
    
    /// 启动调度器
    async fn start(&self) -> DomainResult<()>;
    
    /// 停止调度器
    async fn stop(&self) -> DomainResult<()>;
}

/// 调度ID
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct ScheduleId(pub Uuid);

impl ScheduleId {
    /// 创建新的调度ID
    pub fn new() -> Self {
        Self(Uuid::new_v4())
    }
    
    /// 获取UUID值
    pub fn value(&self) -> Uuid {
        self.0
    }
}

impl Default for ScheduleId {
    fn default() -> Self {
        Self::new()
    }
}

/// 调度状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScheduleStatus {
    pub schedule_id: ScheduleId,
    pub job_type: String,
    pub schedule_type: ScheduleType,
    pub is_active: bool,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub last_run_at: Option<chrono::DateTime<chrono::Utc>>,
    pub next_run_at: Option<chrono::DateTime<chrono::Utc>>,
    pub run_count: u64,
    pub success_count: u64,
    pub failure_count: u64,
}

/// 调度类型
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ScheduleType {
    Cron { expression: String },
    Interval { duration: Duration },
    OneTime { scheduled_at: chrono::DateTime<chrono::Utc> },
}

// === 任务工厂接口 ===

/// 任务工厂 - 创建和管理任务相关服务
pub trait JobFactory: Send + Sync + Clone {
    type Queue: JobQueue;
    type Worker: JobWorker;
    type Scheduler: JobScheduler;
    
    /// 创建任务队列
    fn job_queue(&self) -> Self::Queue;
    
    /// 创建任务工作器
    fn job_worker(&self, worker_id: &str) -> Self::Worker;
    
    /// 创建任务调度器
    fn job_scheduler(&self) -> Self::Scheduler;
}

// === 预定义任务类型 ===

/// 欢迎邮件任务
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WelcomeEmailJob {
    pub job_id: JobId,
    pub user_id: crate::domain::UserId,
    pub email: String,
    pub user_name: String,
}

impl Job for WelcomeEmailJob {
    fn job_type(&self) -> &'static str {
        "welcome_email"
    }
    
    fn job_id(&self) -> JobId {
        self.job_id
    }
    
    fn priority(&self) -> JobPriority {
        JobPriority::Normal
    }
}

/// 位置索引任务
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationIndexJob {
    pub job_id: JobId,
    pub user_id: crate::domain::UserId,
    pub latitude: f64,
    pub longitude: f64,
}

impl Job for LocationIndexJob {
    fn job_type(&self) -> &'static str {
        "location_index"
    }
    
    fn job_id(&self) -> JobId {
        self.job_id
    }
    
    fn priority(&self) -> JobPriority {
        JobPriority::High
    }
}

/// 通知推送任务
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationJob {
    pub job_id: JobId,
    pub user_ids: Vec<crate::domain::UserId>,
    pub notification_type: String,
    pub title: String,
    pub content: String,
    pub data: Option<serde_json::Value>,
}

impl Job for NotificationJob {
    fn job_type(&self) -> &'static str {
        "notification"
    }
    
    fn job_id(&self) -> JobId {
        self.job_id
    }
    
    fn priority(&self) -> JobPriority {
        JobPriority::High
    }
    
    fn timeout(&self) -> Duration {
        Duration::from_secs(60) // 1分钟超时
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_job_id() {
        let job_id = JobId::new();
        assert!(!job_id.value().is_nil());
        
        let job_id2 = JobId::new();
        assert_ne!(job_id, job_id2);
    }
    
    #[test]
    fn test_job_priority_ordering() {
        assert!(JobPriority::Critical > JobPriority::High);
        assert!(JobPriority::High > JobPriority::Normal);
        assert!(JobPriority::Normal > JobPriority::Low);
    }
    
    #[test]
    fn test_job_result() {
        let success = JobResult::success();
        assert!(matches!(success, JobResult::Success { .. }));
        
        let failure = JobResult::permanent_failure("test error");
        assert!(matches!(failure, JobResult::Failure { permanent: true, .. }));
        
        let retry = JobResult::retry("retry error");
        assert!(matches!(retry, JobResult::Retry { .. }));
    }
    
    #[test]
    fn test_welcome_email_job() {
        let job = WelcomeEmailJob {
            job_id: JobId::new(),
            user_id: crate::domain::UserId::new(123).unwrap(),
            email: "test@example.com".to_string(),
            user_name: "Test User".to_string(),
        };
        
        assert_eq!(job.job_type(), "welcome_email");
        assert_eq!(job.priority(), JobPriority::Normal);
    }
}

// === 任务调度器接口 ===

/// 任务调度器接口 - 统一的任务分发接口
#[async_trait]
pub trait JobDispatcher: Send + Sync {
    /// 分发任务到队列
    async fn dispatch<T>(&self, job: T) -> DomainResult<JobId>
    where
        T: Job;
    
    /// 批量分发任务
    async fn dispatch_batch<T>(&self, jobs: Vec<T>) -> DomainResult<Vec<JobId>>
    where
        T: Job;
    
    /// 延迟分发任务
    async fn dispatch_delayed<T>(&self, job: T, delay: Duration) -> DomainResult<JobId>
    where
        T: Job;
    
    /// 获取调度器状态
    async fn get_dispatcher_status(&self) -> DomainResult<DispatcherStatus>;
}

/// 调度器状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DispatcherStatus {
    pub is_active: bool,
    pub dispatched_jobs: u64,
    pub failed_dispatches: u64,
    pub queue_health: QueueHealth,
}

/// 队列健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum QueueHealth {
    Healthy,
    Degraded,
    Unhealthy,
}

// === 类型别名（向后兼容） ===

/// 任务调度器端口（向后兼容别名）
pub type JobDispatcherPort = dyn JobDispatcher;
