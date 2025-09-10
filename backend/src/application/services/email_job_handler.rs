/// 邮件任务处理器
/// 
/// 处理各种邮件发送任务，包括欢迎邮件、密码重置邮件等
use async_trait::async_trait;
use std::sync::Arc;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, DomainError};
use crate::ports::{
    JobHandlerPort, Job, JobHandlerResult, JobId,
    NotificationTransportPort, Notification, NotificationChannel, NotificationType,
    UserRepositoryPort,
};

/// 欢迎邮件任务数据
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct WelcomeEmailJobData {
    pub user_id: i64,
    pub nickname: String,
    pub email: String,
}

/// 密码重置邮件任务数据
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PasswordResetEmailJobData {
    pub user_id: i64,
    pub email: String,
    pub reset_token: String,
    pub reset_link: String,
}

/// 群组邀请邮件任务数据
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct GroupInviteEmailJobData {
    pub user_id: i64,
    pub email: String,
    pub group_id: i64,
    pub group_name: String,
    pub inviter_name: String,
}

/// 邮件任务处理器
#[derive(Clone)]
pub struct EmailJobHandler<T, U>
where
    T: NotificationTransportPort,
    U: UserRepositoryPort,
{
    /// 通知传输器（支持邮件渠道）
    transport: Arc<T>,
    /// 用户仓库
    user_repository: Arc<U>,
}

impl<T, U> EmailJobHandler<T, U>
where
    T: NotificationTransportPort,
    U: UserRepositoryPort,
{
    pub fn new(transport: Arc<T>, user_repository: Arc<U>) -> Self {
        Self {
            transport,
            user_repository,
        }
    }
    
    /// 处理欢迎邮件任务
    async fn handle_welcome_email(&self, job: &Job<WelcomeEmailJobData>) -> JobHandlerResult {
        let data = &job.data;
        
        info!("处理欢迎邮件任务: 用户{} - {}", data.user_id, data.email);
        
        // 创建欢迎邮件通知
        let mut notification = Notification::new(
            data.user_id,
            NotificationType::System,
            "欢迎加入 HotSpot！".to_string(),
            format!("欢迎您，{}！感谢您加入 HotSpot 社区。", data.nickname),
        );
        
        // 设置邮件渠道
        notification = notification.with_channels(vec![NotificationChannel::Email]);
        
        // 添加用户数据
        notification = notification
            .with_data("user_id".to_string(), serde_json::Value::Number(data.user_id.into()))
            .with_data("nickname".to_string(), serde_json::Value::String(data.nickname.clone()))
            .with_data("email".to_string(), serde_json::Value::String(data.email.clone()));
        
        // 发送邮件
        match self.transport.send_notification(notification).await {
            Ok(_) => {
                info!("欢迎邮件发送成功: 用户{}", data.user_id);
                Ok(())
            },
            Err(e) => {
                error!("欢迎邮件发送失败: 用户{} - {:?}", data.user_id, e);
                Err(format!("欢迎邮件发送失败: {}", e))
            }
        }
    }
    
    /// 处理密码重置邮件任务
    async fn handle_password_reset_email(&self, job: &Job<PasswordResetEmailJobData>) -> JobHandlerResult {
        let data = &job.data;
        
        info!("处理密码重置邮件任务: 用户{} - {}", data.user_id, data.email);
        
        // 获取用户信息
        let user = match self.user_repository.find_by_id(data.user_id).await {
            Ok(Some(user)) => user,
            Ok(None) => {
                warn!("用户{}不存在，跳过密码重置邮件", data.user_id);
                return Err("用户不存在".to_string());
            },
            Err(e) => {
                error!("获取用户信息失败: 用户{} - {:?}", data.user_id, e);
                return Err(format!("获取用户信息失败: {}", e));
            }
        };
        
        // 创建密码重置邮件通知
        let mut notification = Notification::new(
            data.user_id,
            NotificationType::System,
            "重置您的 HotSpot 密码".to_string(),
            "我们收到了您的密码重置请求。请点击邮件中的链接重置您的密码。".to_string(),
        );
        
        // 设置邮件渠道
        notification = notification.with_channels(vec![NotificationChannel::Email]);
        
        // 添加重置数据
        notification = notification
            .with_data("user_id".to_string(), serde_json::Value::Number(data.user_id.into()))
            .with_data("email".to_string(), serde_json::Value::String(data.email.clone()))
            .with_data("reset_token".to_string(), serde_json::Value::String(data.reset_token.clone()))
            .with_data("reset_link".to_string(), serde_json::Value::String(data.reset_link.clone()));
        
        // 发送邮件
        match self.transport.send_notification(notification).await {
            Ok(_) => {
                info!("密码重置邮件发送成功: 用户{}", data.user_id);
                Ok(())
            },
            Err(e) => {
                error!("密码重置邮件发送失败: 用户{} - {:?}", data.user_id, e);
                Err(format!("密码重置邮件发送失败: {}", e))
            }
        }
    }
    
    /// 处理群组邀请邮件任务
    async fn handle_group_invite_email(&self, job: &Job<GroupInviteEmailJobData>) -> JobHandlerResult {
        let data = &job.data;
        
        info!("处理群组邀请邮件任务: 用户{} - 群组{}", data.user_id, data.group_id);
        
        // 创建群组邀请邮件通知
        let mut notification = Notification::new(
            data.user_id,
            NotificationType::GroupInvite,
            format!("您被邀请加入群组：{}", data.group_name),
            format!("{} 邀请您加入群组 \"{}\"。", data.inviter_name, data.group_name),
        );
        
        // 设置邮件渠道
        notification = notification.with_channels(vec![NotificationChannel::Email]);
        
        // 添加群组数据
        notification = notification
            .with_data("user_id".to_string(), serde_json::Value::Number(data.user_id.into()))
            .with_data("email".to_string(), serde_json::Value::String(data.email.clone()))
            .with_data("group_id".to_string(), serde_json::Value::Number(data.group_id.into()))
            .with_data("group_name".to_string(), serde_json::Value::String(data.group_name.clone()))
            .with_data("inviter_name".to_string(), serde_json::Value::String(data.inviter_name.clone()));
        
        // 发送邮件
        match self.transport.send_notification(notification).await {
            Ok(_) => {
                info!("群组邀请邮件发送成功: 用户{} - 群组{}", data.user_id, data.group_id);
                Ok(())
            },
            Err(e) => {
                error!("群组邀请邮件发送失败: 用户{} - 群组{} - {:?}", data.user_id, data.group_id, e);
                Err(format!("群组邀请邮件发送失败: {}", e))
            }
        }
    }
}

#[async_trait]
impl<T, U> JobHandlerPort for EmailJobHandler<T, U>
where
    T: NotificationTransportPort + Send + Sync,
    U: UserRepositoryPort + Send + Sync,
{
    async fn handle(&self, job: &crate::ports::JobEnvelope) -> JobHandlerResult {
        debug!("处理邮件任务: {} - {}", job.job_type, job.id);
        
        match job.job_type.as_str() {
            "welcome_email" => {
                let job_data: Job<WelcomeEmailJobData> = serde_json::from_value(job.data.clone())
                    .map_err(|e| format!("解析欢迎邮件任务数据失败: {}", e))?;
                self.handle_welcome_email(&job_data).await
            },
            "password_reset_email" => {
                let job_data: Job<PasswordResetEmailJobData> = serde_json::from_value(job.data.clone())
                    .map_err(|e| format!("解析密码重置邮件任务数据失败: {}", e))?;
                self.handle_password_reset_email(&job_data).await
            },
            "group_invite_email" => {
                let job_data: Job<GroupInviteEmailJobData> = serde_json::from_value(job.data.clone())
                    .map_err(|e| format!("解析群组邀请邮件任务数据失败: {}", e))?;
                self.handle_group_invite_email(&job_data).await
            },
            _ => {
                warn!("未知的邮件任务类型: {}", job.job_type);
                Err(format!("未知的邮件任务类型: {}", job.job_type))
            }
        }
    }
    
    async fn can_handle(&self, job_type: &str) -> bool {
        matches!(job_type, "welcome_email" | "password_reset_email" | "group_invite_email")
    }
    
    async fn get_handler_info(&self) -> crate::ports::JobHandlerInfo {
        crate::ports::JobHandlerInfo {
            name: "EmailJobHandler".to_string(),
            version: "1.0.0".to_string(),
            supported_job_types: vec![
                "welcome_email".to_string(),
                "password_reset_email".to_string(),
                "group_invite_email".to_string(),
            ],
            max_concurrent_jobs: 10,
            avg_processing_time_ms: 2000, // 邮件发送通常需要1-3秒
        }
    }
}

/// 邮件任务类型常量
pub mod email_job_types {
    pub const WELCOME_EMAIL: &str = "welcome_email";
    pub const PASSWORD_RESET_EMAIL: &str = "password_reset_email";
    pub const GROUP_INVITE_EMAIL: &str = "group_invite_email";
}

/// 邮件任务辅助函数
pub mod email_job_helpers {
    use super::*;
    use crate::ports::{Job, JobDispatcherPort};
    
    /// 分发欢迎邮件任务
    pub async fn dispatch_welcome_email<J: JobDispatcherPort>(
        dispatcher: &J,
        user_id: i64,
        nickname: String,
        email: String,
    ) -> DomainResult<JobId> {
        let data = WelcomeEmailJobData {
            user_id,
            nickname,
            email,
        };
        
        let job = Job::new(email_job_types::WELCOME_EMAIL.to_string(), data)
            .with_retry(3, 60); // 重试3次，每次间隔60秒
        
        dispatcher.dispatch_job(job).await
    }
    
    /// 分发密码重置邮件任务
    pub async fn dispatch_password_reset_email<J: JobDispatcherPort>(
        dispatcher: &J,
        user_id: i64,
        email: String,
        reset_token: String,
        reset_link: String,
    ) -> DomainResult<JobId> {
        let data = PasswordResetEmailJobData {
            user_id,
            email,
            reset_token,
            reset_link,
        };
        
        let job = Job::new(email_job_types::PASSWORD_RESET_EMAIL.to_string(), data)
            .with_retry(3, 60); // 重试3次，每次间隔60秒
        
        dispatcher.dispatch_job(job).await
    }
    
    /// 分发群组邀请邮件任务
    pub async fn dispatch_group_invite_email<J: JobDispatcherPort>(
        dispatcher: &J,
        user_id: i64,
        email: String,
        group_id: i64,
        group_name: String,
        inviter_name: String,
    ) -> DomainResult<JobId> {
        let data = GroupInviteEmailJobData {
            user_id,
            email,
            group_id,
            group_name,
            inviter_name,
        };
        
        let job = Job::new(email_job_types::GROUP_INVITE_EMAIL.to_string(), data)
            .with_retry(2, 30); // 重试2次，每次间隔30秒
        
        dispatcher.dispatch_job(job).await
    }
}
