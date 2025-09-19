/// 任务应用服务
/// 
/// 提供高级任务管理功能，基于任务调度端口实现业务相关的异步任务处理
use async_trait::async_trait;
use std::sync::Arc;
use uuid::Uuid;
use serde::{Deserialize, Serialize};

use crate::domain::DomainResult;
use crate::ports::{JobDispatcher, JobId, JobStatus, QueueStats};
use crate::ports::jobs::{
    WelcomeEmailJob, LocationIndexJob,
    GroupMemberNotificationJob, MomentContentIndexJob, 
    MessageNotificationJob, CleanupExpiredSessionsJob
};

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
pub struct JobApplicationServiceImpl<J: JobDispatcher> {
    job_dispatcher: Arc<J>,
}

impl<J: JobDispatcher> JobApplicationServiceImpl<J> {
    /// 创建新的任务应用服务
    pub fn new(job_dispatcher: Arc<J>) -> Self {
        Self { job_dispatcher }
    }
}

#[async_trait]
impl<J: JobDispatcher> JobApplicationService for JobApplicationServiceImpl<J> {
    async fn dispatch_welcome_email(
        &self,
        user_id: i64,
        nickname: String,
        email: Option<String>,
    ) -> DomainResult<JobId> {
        let job = WelcomeEmailJob {
            job_id: JobId::new(),
            user_id: crate::domain::UserId::new(user_id)?,
            email: email.unwrap_or_default(),
            user_name: nickname,
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn dispatch_location_index(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        _location_name: Option<String>,
    ) -> DomainResult<JobId> {
        let job = LocationIndexJob {
            job_id: JobId::new(),
            user_id: crate::domain::UserId::new(user_id)?,
            latitude,
            longitude,
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn dispatch_group_member_notification(
        &self,
        group_id: Uuid,
        user_id: i64,
        notification_type: String,
        message: String,
    ) -> DomainResult<JobId> {
        let job = GroupMemberNotificationJob {
            job_id: JobId::new(),
            group_id,
            user_id,
            notification_type,
            message,
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn dispatch_moment_content_index(
        &self,
        moment_id: i64,
        user_id: i64,
        content: String,
        latitude: f64,
        longitude: f64,
    ) -> DomainResult<JobId> {
        let job = MomentContentIndexJob {
            job_id: JobId::new(),
            moment_id,
            user_id,
            content,
            latitude,
            longitude,
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn dispatch_message_notification(
        &self,
        message_id: i64,
        sender_id: i64,
        recipient_id: i64,
        content: String,
    ) -> DomainResult<JobId> {
        let job = MessageNotificationJob {
            job_id: JobId::new(),
            message_id,
            sender_id,
            recipient_id,
            content,
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn dispatch_cleanup_expired_sessions(&self) -> DomainResult<JobId> {
        let job = CleanupExpiredSessionsJob {
            job_id: JobId::new(),
            max_age_hours: 24, // 默认清理24小时前的会话
        };

        self.job_dispatcher.dispatch(job).await
    }

    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>> {
        // 通过调度器获取任务状态
        match self.job_dispatcher.get_job_status(job_id).await {
            Ok(status) => Ok(status),
            Err(e) => {
                log::error!("获取任务状态失败: {:?} - {:?}", job_id, e);
                Err(e)
            }
        }
    }

    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool> {
        // 通过调度器取消任务
        match self.job_dispatcher.cancel_job(job_id).await {
            Ok(cancelled) => {
                if cancelled {
                    log::info!("任务取消成功: {:?}", job_id);
                } else {
                    log::warn!("任务取消失败或任务不存在: {:?}", job_id);
                }
                Ok(cancelled)
            }
            Err(e) => {
                log::error!("任务取消操作失败: {:?} - {:?}", job_id, e);
                Err(e)
            }
        }
    }

    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool> {
        // 通过调度器重试任务
        match self.job_dispatcher.retry_job(job_id).await {
            Ok(retried) => {
                if retried {
                    log::info!("任务重试成功: {:?}", job_id);
                } else {
                    log::warn!("任务重试失败或任务不符合重试条件: {:?}", job_id);
                }
                Ok(retried)
            }
            Err(e) => {
                log::error!("任务重试操作失败: {:?} - {:?}", job_id, e);
                Err(e)
            }
        }
    }

    async fn get_queue_stats(&self) -> DomainResult<QueueStats> {
        // 获取调度器状态并转换为队列统计
        let dispatcher_status = self.job_dispatcher.get_dispatcher_status().await?;
        
        Ok(QueueStats {
            pending_jobs: 0, // 基础实现暂时返回0，具体实现需要根据调度器类型
            running_jobs: 0,
            completed_jobs: dispatcher_status.dispatched_jobs.max(0) as u64,
            failed_jobs: dispatcher_status.failed_dispatches.max(0) as u64,
            cancelled_jobs: 0,
            total_jobs: dispatcher_status.dispatched_jobs.max(0) as u64,
            active_workers: if dispatcher_status.is_active { 1 } else { 0 },
            queue_size: 0,
            average_processing_time: std::time::Duration::from_secs(0),
            last_processed_at: None,
        })
    }
}

/// 任务管理器
/// 
/// 提供任务相关的管理功能
#[derive(Clone)]
pub struct JobManager<J: JobDispatcher> {
    job_dispatcher: Arc<J>,
    job_service: Arc<JobApplicationServiceImpl<J>>,
}

impl<J: JobDispatcher> JobManager<J> {
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
        // 获取调度器状态并转换为队列统计
        let _dispatcher_status = self.job_dispatcher.get_dispatcher_status().await?;
        Ok(QueueStats {
            pending_jobs: 0,
            running_jobs: 0,
            completed_jobs: 0,
            failed_jobs: 0,
            cancelled_jobs: 0,
            total_jobs: 0,
            active_workers: 0,
            queue_size: 0,
            average_processing_time: std::time::Duration::from_secs(0),
            last_processed_at: None,
        })
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

// 任务工厂已移除，现在直接使用具体的 Job 类型
