//! 任务调度器
use std::sync::Arc;
use tracing::error;
use lapin::Error as LapinError;


use crate::utils::job_queue::{Job, JobPriority, JobQueue};
use crate::jobs::group_models::{
    GroupMessageDigestJob, GroupMemberActivityJob, GroupActivityType
};
use crate::jobs::location_models::MomentIndexJob;
use crate::jobs::user_models::{
    WelcomeEmailJob, NotificationJob, LocationIndexJob
};
use crate::jobs::handler::JobHandler;

/// 任务调度器
pub struct JobDispatcher {
    /// 任务队列
    job_queue: Arc<JobQueue>,
    /// 群组消息摘要任务处理器
    group_digest_handler: Option<Arc<crate::jobs::group_handlers::GroupDigestHandler<crate::repositories::PgGroupRepository>>>,
    /// 群组成员活跃度更新任务处理器
    group_activity_handler: Option<Arc<crate::jobs::group_handlers::GroupMemberActivityHandler>>,
    /// 动态索引任务处理器
    moment_index_handler: Option<Arc<crate::jobs::location_handlers::MomentIndexHandler<crate::cache::location_cache::RedisLocationCache>>>,
    
    /// 欢迎邮件任务处理器
    welcome_email_handler: Option<Arc<crate::jobs::user_handlers::WelcomeEmailHandler<crate::services::notification::WebSocketNotificationService>>>,
    /// 通知推送任务处理器
    notification_handler: Option<Arc<crate::jobs::user_handlers::NotificationHandler>>,
    /// 位置索引任务处理器
    location_index_handler: Option<Arc<crate::jobs::user_handlers::LocationIndexHandler>>,
}

impl JobDispatcher {
    /// 创建新的任务调度器
    pub fn new(
        job_queue: Arc<JobQueue>,
        group_digest_handler: Option<Arc<crate::jobs::group_handlers::GroupDigestHandler<crate::repositories::PgGroupRepository>>>,
        group_activity_handler: Option<Arc<crate::jobs::group_handlers::GroupMemberActivityHandler>>,
        moment_index_handler: Option<Arc<crate::jobs::location_handlers::MomentIndexHandler<crate::cache::location_cache::RedisLocationCache>>>,
        
        welcome_email_handler: Option<Arc<crate::jobs::user_handlers::WelcomeEmailHandler<crate::services::notification::WebSocketNotificationService>>>,
        notification_handler: Option<Arc<crate::jobs::user_handlers::NotificationHandler>>,
        location_index_handler: Option<Arc<crate::jobs::user_handlers::LocationIndexHandler>>,
    ) -> Self {
        Self {
            job_queue,
            group_digest_handler,
            group_activity_handler,
            moment_index_handler,
            
            welcome_email_handler,
            notification_handler,
            location_index_handler,
        }
    }
    
    /// 注册所有任务处理器
    pub async fn register_handlers(&self) -> Result<(), LapinError> {
        // Group related handlers
        self.register_group_message_digest_handler().await?;
        self.register_group_member_activity_handler().await?;
        
        // Location related handlers
        self.register_moment_index_handler().await?;
        
        
        // User related handlers
        self.register_welcome_email_handler().await?;
        self.register_notification_handler().await?;
        self.register_location_index_handler().await?;
        
        Ok(())
    }
    
    /// 注册群组消息摘要任务处理器
    async fn register_group_message_digest_handler(&self) -> Result<(), LapinError> {
        let handler = self.group_digest_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<GroupMessageDigestJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle group message digest job: {:?}", e);
                    }
                } else {
                    error!("Group message digest job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    /// 注册群组成员活跃度更新任务处理器
    async fn register_group_member_activity_handler(&self) -> Result<(), LapinError> {
        let handler = self.group_activity_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<GroupMemberActivityJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle group member activity job: {:?}", e);
                    }
                } else {
                    error!("Group member activity job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    /// 注册动态索引任务处理器
    async fn register_moment_index_handler(&self) -> Result<(), LapinError> {
        let handler = self.moment_index_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<MomentIndexJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle moment index job: {:?}", e);
                    }
                } else {
                    error!("Moment index job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    
    
    /// 注册欢迎邮件任务处理器
    async fn register_welcome_email_handler(&self) -> Result<(), LapinError> {
        let handler = self.welcome_email_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<WelcomeEmailJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle welcome email job: {:?}", e);
                    }
                } else {
                    error!("Welcome email job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    /// 注册通知推送任务处理器
    async fn register_notification_handler(&self) -> Result<(), LapinError> {
        let handler = self.notification_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<NotificationJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle notification job: {:?}", e);
                    }
                } else {
                    error!("Notification job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    /// 注册位置索引任务处理器
    async fn register_location_index_handler(&self) -> Result<(), LapinError> {
        let handler = self.location_index_handler.clone();
        
        self.job_queue.register_handler(move |job: Job<LocationIndexJob>| {
            let handler = handler.clone();
            async move {
                if let Some(h) = handler {
                    if let Err(e) = h.handle(&job).await {
                        error!("Failed to handle location index job: {:?}", e);
                    }
                } else {
                    error!("Location index job handler not configured.");
                }
                Ok(())
            }
        }).await?;
        
        Ok(())
    }
    
    /// 分派群组消息摘要任务
    #[allow(dead_code)]
    pub async fn dispatch_group_message_digest(
        &self,
        group_id: uuid::Uuid,
        group_name: String,
        last_active_at: i64,
        min_inactivity_days: i32,
    ) -> Result<(), LapinError> {
        let job = Job::new(
            "group_message_digest",
            GroupMessageDigestJob {
                group_id,
                group_name,
                last_active_at,
                min_inactivity_days,
            },
        )
        .with_priority(JobPriority::Normal)
        .with_delay(60000); // 延迟1分钟执行，避免高峰期处理
        
        self.job_queue.enqueue(job).await
    }
    
    /// 分派群组成员活跃度更新任务
    pub async fn dispatch_group_member_activity(
        &self,
        group_id: uuid::Uuid,
        user_id: i64,
        activity_type: GroupActivityType,
    ) -> Result<(), LapinError> {
        let job = Job::new(
            "group_member_activity_update",
            GroupMemberActivityJob {
                group_id,
                user_id,
                activity_type,
                timestamp: chrono::Utc::now().timestamp(),
            },
        )
        .with_priority(JobPriority::Normal);
        
        self.job_queue.enqueue(job).await
    }
    
    /// 分派生活瞬间索引任务
    pub async fn dispatch_moment_index(
        &self,
        moment_id: String,
        user_id: i64,
        latitude: f64,
        longitude: f64,
    ) -> Result<(), lapin::Error> {
        let moment_index_job = MomentIndexJob {
            moment_id,
            user_id,
            latitude,
            longitude,
            timestamp: std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_secs() as i64,
        };

        let job = Job::new("moment.index", moment_index_job);
        self.job_queue.enqueue(job).await
    }
    
    /// 分派欢迎邮件任务
    pub async fn dispatch_welcome_email(
        &self,
        user_id: i64,
        nickname: String,
        email: Option<String>,
    ) -> Result<(), LapinError> {
        let job = Job::new(
            "welcome_email",
            WelcomeEmailJob {
                user_id,
                nickname,
                email,
            },
        )
        .with_priority(JobPriority::Normal)
        .with_delay(5000); // 延迟5秒执行，等待用户完成注册流程
        
        self.job_queue.enqueue(job).await
    }
    
    /// 分派位置索引任务
    #[allow(dead_code)]
    pub async fn dispatch_location_index(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> Result<(), LapinError> {
        let timestamp = chrono::Utc::now().timestamp();
        let job = Job::new(
            "location_index",
            LocationIndexJob {
                user_id,
                latitude,
                longitude,
                location_name,
                timestamp,
            },
        )
        .with_priority(JobPriority::Normal);
        
        self.job_queue.enqueue(job).await
    }
    
    /// 分派通知推送任务
    pub async fn dispatch_notification(
        &self,
        user_id: i64,
        notification_type: &str,
        payload: String,
    ) -> Result<(), LapinError> {
        let timestamp = chrono::Utc::now().timestamp();
        
        // 将字符串转换为枚举
        use crate::jobs::user_models::NotificationJob;
        use crate::jobs::user_models::NotificationType;
        
        let notification_type = match notification_type {
            "message" => NotificationType::Message,
            "system" => NotificationType::System,
            "moment" => NotificationType::Moment,
            "group_invite" => NotificationType::GroupInvite,
            "friend_request" => NotificationType::FriendRequest,
            _ => NotificationType::System, // 默认
        };
        
        let job = Job::new(
            "notification",
            NotificationJob {
                user_id,
                notification_type,
                payload,
                timestamp,
            },
        )
        .with_priority(JobPriority::Normal);
        
        self.job_queue.enqueue(job).await
    }
}

impl std::fmt::Debug for JobDispatcher {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("JobDispatcher")
            .field("job_queue", &"<JobQueue>")
            .field("handlers_configured", &{
                let mut handlers = Vec::new();
                if self.group_digest_handler.is_some() { handlers.push("GroupDigest"); }
                if self.group_activity_handler.is_some() { handlers.push("GroupActivity"); }
                if self.moment_index_handler.is_some() { handlers.push("MomentIndex"); }
                
                if self.welcome_email_handler.is_some() { handlers.push("WelcomeEmail"); }
                if self.notification_handler.is_some() { handlers.push("Notification"); }
                if self.location_index_handler.is_some() { handlers.push("LocationIndex"); }
                handlers
            })
            .finish()
    }
}
