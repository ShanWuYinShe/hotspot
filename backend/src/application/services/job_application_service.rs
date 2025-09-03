/// 任务应用服务
/// 
/// 提供高级任务管理功能，基于任务调度端口实现业务相关的异步任务处理

use async_trait::async_trait;
use std::sync::Arc;
use uuid::Uuid;
use serde::{Deserialize, Serialize};

use crate::domain::DomainResult;
use crate::ports::{JobDispatcherPort, Job, JobId, JobStatus, QueueStats, job_types};

/// 任务应用服务接口
#[async_trait]
pub trait JobApplicationService: Send + Sync {
    /// 分派欢迎邮件任务
    async fn dispatch_welcome_email(
        &self,
        user_id: i64,
        nickname: String,
        email: Option<String>,
    ) -> DomainResult<JobId>;

    /// 分派位置索引任务
    async fn dispatch_location_index(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> DomainResult<JobId>;

    /// 分派群组成员通知任务
    async fn dispatch_group_member_notification(
        &self,
        group_id: Uuid,
        user_id: i64,
        notification_type: String,
        message: String,
    ) -> DomainResult<JobId>;

    /// 分派瞬间内容索引任务
    async fn dispatch_moment_content_index(
        &self,
        moment_id: i64,
        user_id: i64,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<JobId>;

    /// 分派消息通知任务
    async fn dispatch_message_notification(
        &self,
        message_id: i64,
        sender_id: i64,
        recipient_id: i64,
        content: String,
    ) -> DomainResult<JobId>;

    /// 分派清理过期会话任务
    async fn dispatch_cleanup_expired_sessions(&self) -> DomainResult<JobId>;

    /// 获取任务状态
    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>>;

    /// 取消任务
    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool>;

    /// 重试失败的任务
    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool>;

    /// 获取队列统计信息
    async fn get_queue_stats(&self) -> DomainResult<QueueStats>;
}

/// 任务载荷定义

/// 欢迎邮件任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WelcomeEmailPayload {
    pub user_id: i64,
    pub nickname: String,
    pub email: Option<String>,
}

/// 位置索引任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationIndexPayload {
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
}

/// 群组成员通知任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupMemberNotificationPayload {
    pub group_id: Uuid,
    pub user_id: i64,
    pub notification_type: String,
    pub message: String,
}

/// 瞬间内容索引任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentContentIndexPayload {
    pub moment_id: i64,
    pub user_id: i64,
    pub content: String,
    pub latitude: f64,
    pub longitude: f64,
}

/// 消息通知任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageNotificationPayload {
    pub message_id: i64,
    pub sender_id: i64,
    pub recipient_id: i64,
    pub content: String,
}

/// 清理过期会话任务载荷
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CleanupExpiredSessionsPayload {
    pub max_age_hours: u32,
}

/// 任务应用服务实现
pub struct JobApplicationServiceImpl<J: JobDispatcherPort> {
    job_dispatcher: Arc<J>,
}

impl<J: JobDispatcherPort> JobApplicationServiceImpl<J> {
    /// 创建新的任务应用服务
    pub fn new(job_dispatcher: Arc<J>) -> Self {
        Self { job_dispatcher }
    }
}

#[async_trait]
impl<J: JobDispatcherPort> JobApplicationService for JobApplicationServiceImpl<J> {
    async fn dispatch_welcome_email(
        &self,
        user_id: i64,
        nickname: String,
        email: Option<String>,
    ) -> DomainResult<JobId> {
        let payload = WelcomeEmailPayload {
            user_id,
            nickname,
            email,
        };

        let job = Job::new(job_types::USER_WELCOME_EMAIL.to_string(), payload)
            .with_retry(3, 60); // 重试3次，每次间隔60秒

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn dispatch_location_index(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> DomainResult<JobId> {
        let payload = LocationIndexPayload {
            user_id,
            latitude,
            longitude,
            location_name,
        };

        let job = Job::new(job_types::USER_LOCATION_INDEX.to_string(), payload)
            .with_retry(2, 30); // 重试2次，每次间隔30秒

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn dispatch_group_member_notification(
        &self,
        group_id: Uuid,
        user_id: i64,
        notification_type: String,
        message: String,
    ) -> DomainResult<JobId> {
        let payload = GroupMemberNotificationPayload {
            group_id,
            user_id,
            notification_type,
            message,
        };

        let job = Job::new(job_types::GROUP_MEMBER_NOTIFICATION.to_string(), payload)
            .with_retry(2, 15); // 重试2次，每次间隔15秒

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn dispatch_moment_content_index(
        &self,
        moment_id: i64,
        user_id: i64,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<JobId> {
        let payload = MomentContentIndexPayload {
            moment_id,
            user_id,
            content,
            latitude,
            longitude,
        };

        let job = Job::new(job_types::MOMENT_CONTENT_INDEX.to_string(), payload)
            .with_retry(2, 30); // 重试2次，每次间隔30秒

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn dispatch_message_notification(
        &self,
        message_id: i64,
        sender_id: i64,
        recipient_id: i64,
        content: String,
    ) -> DomainResult<JobId> {
        let payload = MessageNotificationPayload {
            message_id,
            sender_id,
            recipient_id,
            content,
        };

        let job = Job::new(job_types::MESSAGE_NOTIFICATION.to_string(), payload)
            .with_retry(3, 10); // 重试3次，每次间隔10秒

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn dispatch_cleanup_expired_sessions(&self) -> DomainResult<JobId> {
        let payload = CleanupExpiredSessionsPayload {
            max_age_hours: 24, // 默认清理24小时前的会话
        };

        let job = Job::new(job_types::CLEANUP_EXPIRED_SESSIONS.to_string(), payload)
            .with_retry(1, 300); // 重试1次，间隔5分钟

        self.job_dispatcher.dispatch_job(job).await
    }

    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>> {
        self.job_dispatcher.get_job_status(job_id).await
    }

    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool> {
        self.job_dispatcher.cancel_job(job_id).await
    }

    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool> {
        self.job_dispatcher.retry_job(job_id).await
    }

    async fn get_queue_stats(&self) -> DomainResult<QueueStats> {
        self.job_dispatcher.get_queue_stats().await
    }
}

/// 任务管理器
/// 
/// 提供任务相关的管理功能
#[derive(Clone)]
pub struct JobManager<J: JobDispatcherPort> {
    job_dispatcher: Arc<J>,
    job_service: Arc<JobApplicationServiceImpl<J>>,
}

impl<J: JobDispatcherPort> JobManager<J> {
    /// 创建新的任务管理器
    pub fn new(job_dispatcher: Arc<J>) -> Self {
        let job_service = Arc::new(JobApplicationServiceImpl::new(job_dispatcher.clone()));
        Self {
            job_dispatcher,
            job_service,
        }
    }

    /// 获取任务调度器
    pub fn get_dispatcher(&self) -> Arc<J> {
        self.job_dispatcher.clone()
    }

    /// 获取任务应用服务
    pub fn get_job_service(&self) -> Arc<JobApplicationServiceImpl<J>> {
        self.job_service.clone()
    }

    /// 获取队列统计信息
    pub async fn get_stats(&self) -> DomainResult<QueueStats> {
        self.job_dispatcher.get_queue_stats().await
    }

    /// 健康检查
    pub async fn health_check(&self) -> DomainResult<bool> {
        // 尝试获取队列统计信息来验证任务系统是否正常工作
        match self.get_stats().await {
            Ok(_) => Ok(true),
            Err(_) => Ok(false),
        }
    }
}

/// 任务工厂
/// 
/// 提供便捷的任务创建方法
pub struct JobFactory;

impl JobFactory {
    /// 创建用户相关任务
    pub fn create_user_task<T>(task_type: &str, payload: T) -> Job<T> 
    where
        T: serde::Serialize + for<'de> serde::Deserialize<'de> + Send + Sync + std::fmt::Debug,
    {
        Job::new(task_type.to_string(), payload)
            .with_retry(3, 60)
            .with_metadata("category".to_string(), "user".to_string())
    }

    /// 创建群组相关任务
    pub fn create_group_task<T>(task_type: &str, payload: T) -> Job<T>
    where
        T: serde::Serialize + for<'de> serde::Deserialize<'de> + Send + Sync + std::fmt::Debug,
    {
        Job::new(task_type.to_string(), payload)
            .with_retry(2, 30)
            .with_metadata("category".to_string(), "group".to_string())
    }

    /// 创建消息相关任务
    pub fn create_message_task<T>(task_type: &str, payload: T) -> Job<T>
    where
        T: serde::Serialize + for<'de> serde::Deserialize<'de> + Send + Sync + std::fmt::Debug,
    {
        Job::new(task_type.to_string(), payload)
            .with_retry(3, 10)
            .with_metadata("category".to_string(), "message".to_string())
    }

    /// 创建系统维护任务
    pub fn create_maintenance_task<T>(task_type: &str, payload: T) -> Job<T>
    where
        T: serde::Serialize + for<'de> serde::Deserialize<'de> + Send + Sync + std::fmt::Debug,
    {
        Job::new(task_type.to_string(), payload)
            .with_retry(1, 300)
            .with_metadata("category".to_string(), "maintenance".to_string())
    }
}
