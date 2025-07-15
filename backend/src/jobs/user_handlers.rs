//! 用户相关任务处理器
use std::sync::Arc;
use tracing::{error, info, warn};
use serde_json::json;

use crate::repositories::user_repository::UserRepository;
use crate::services::notification::NotificationService;
use crate::jobs::handler::{JobHandler, JobHandlerResult, JobHandlerError};
use crate::jobs::user_models::{
    WelcomeEmailJob, LocationIndexJob, NotificationJob, UserDigestJob
};
use crate::utils::job_queue::Job;

/// 欢迎邮件任务处理器
pub struct WelcomeEmailHandler<N: NotificationService + ?Sized> {
    _phantom: std::marker::PhantomData<N>,
}

impl<N: NotificationService + ?Sized> WelcomeEmailHandler<N> {
    /// 创建新的欢迎邮件任务处理器
    pub fn new(
        _user_repository: Arc<crate::repositories::PgUserRepository>,
        _notification_service: Arc<N>,
    ) -> Self {
        Self {
            _phantom: std::marker::PhantomData,
        }
    }
    
    /// 发送欢迎邮件
    async fn send_email(&self, recipient: &str, nickname: &str) -> Result<(), String> {
        // 在实际应用中，这里会调用邮件发送服务
        // 这里仅作示例
        info!("发送欢迎邮件给 {} <{}>", nickname, recipient);
        Ok(())
    }
}

impl<N: NotificationService + ?Sized> JobHandler<WelcomeEmailJob> for WelcomeEmailHandler<N> {
    async fn handle(&self, job: &Job<WelcomeEmailJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!("处理欢迎邮件任务: user_id={}, nickname={}", job_data.user_id, job_data.nickname);
        if job_data.email.is_none() {
            return Ok(());
        }
        let email = job_data.email.as_ref().unwrap();
        if let Err(e) = self.send_email(email, &job_data.nickname).await {
            error!("发送欢迎邮件失败: {}", e);
            return Err(JobHandlerError::Other(format!("发送邮件失败: {}", e)));
        }
        Ok(())
    }
}

/// 通知推送任务处理器
pub struct NotificationHandler {
    /// 通知服务
    notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
}

impl NotificationHandler {
    /// 创建新的通知推送任务处理器
    pub fn new(
        notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
    ) -> Self {
        Self {
            notification_service,
        }
    }
}

impl JobHandler<NotificationJob> for NotificationHandler {
    async fn handle(&self, job: &Job<NotificationJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!("处理通知推送任务: user_id={}, type={:?}", job_data.user_id, job_data.notification_type);
        let notification = json!({
            "type": format!("{:?}", job_data.notification_type).to_lowercase(),
            "payload": job_data.payload,
            "timestamp": job_data.timestamp
        });
        
        // 只通过WebSocket发送实时通知，不保存到数据库
        match self.notification_service.send_notification(job_data.user_id, &notification).await {
            Ok(_) => Ok(()),
            Err(e) => Err(JobHandlerError::Notification(e)),
        }
    }
}

/// 用户摘要任务处理器
pub struct UserDigestJobHandler {
    _user_repository: Arc<crate::repositories::PgUserRepository>,
}

impl UserDigestJobHandler {
    /// 创建新的用户摘要任务处理器
    #[allow(dead_code)]
    pub fn new(user_repository: Arc<crate::repositories::PgUserRepository>) -> Self {
        Self { _user_repository: user_repository }
    }
    
    /// 生成周活跃度摘要
    async fn generate_weekly_digest(&self, user_id: i64) -> Result<String, JobHandlerError> {
        let user_nickname = self._user_repository.get_user_nickname(user_id).await
            .map_err(JobHandlerError::Repository)?;
        
        // 这里可以添加从Redis或数据库获取用户一周活动数据的逻辑
        // 暂时返回一个示例摘要
        let digest = format!(
            "嗨 {}！这是你本周的活动摘要：\n\n📍 位置更新：本周你更新了位置信息\n💬 互动参与：参与了社区讨论\n🎯 活跃度：保持良好的互动频率\n\n继续保持活跃，发现更多精彩内容！",
            user_nickname
        );
        
        Ok(digest)
    }
    
    /// 生成月活跃度摘要
    async fn generate_monthly_digest(&self, user_id: i64) -> Result<String, JobHandlerError> {
        let user_nickname = self._user_repository.get_user_nickname(user_id).await
            .map_err(JobHandlerError::Repository)?;
        
        let digest = format!(
            "{}，你的月度报告来了！📊\n\n本月亮点：\n🌟 累计活跃天数：为你点赞\n📱 互动次数：感谢你的参与\n🗺️ 探索足迹：记录了你的精彩旅程\n\n下个月继续加油哦！",
            user_nickname
        );
        
        Ok(digest)
    }
    
    /// 生成活动摘要
    async fn generate_activity_summary(&self, user_id: i64) -> Result<String, JobHandlerError> {
        let user_nickname = self._user_repository.get_user_nickname(user_id).await
            .map_err(JobHandlerError::Repository)?;
        
        let digest = format!(
            "Hi {}！🎉\n\n你的最近活动：\n• 参与了社区互动\n• 更新了个人信息\n• 探索了新的地点\n\n感谢你让社区更加活跃！",
            user_nickname
        );
        
        Ok(digest)
    }
    
    /// 生成位置摘要
    async fn generate_location_summary(&self, user_id: i64) -> Result<String, JobHandlerError> {
        let user_nickname = self._user_repository.get_user_nickname(user_id).await
            .map_err(JobHandlerError::Repository)?;
        
        let digest = format!(
            "{}，这是你的位置活动摘要 🗺️\n\n📍 最近访问的地点：记录了你的足迹\n🚶 活动范围：探索了不同区域\n👥 附近互动：与周边用户有过交流\n\n继续探索，发现更多有趣的地方！",
            user_nickname
        );
        
        Ok(digest)
    }
}

impl JobHandler<UserDigestJob> for UserDigestJobHandler {
    async fn handle(&self, job: &Job<UserDigestJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!(
            "处理用户摘要任务: user_id={}, digest_type={}",
            job_data.user_id, job_data.digest_type
        );
        
        // 根据摘要类型生成不同的用户摘要
        let digest_content = match job_data.digest_type.as_str() {
            "weekly" => self.generate_weekly_digest(job_data.user_id).await?,
            "monthly" => self.generate_monthly_digest(job_data.user_id).await?,
            "activity_summary" => self.generate_activity_summary(job_data.user_id).await?,
            "location_summary" => self.generate_location_summary(job_data.user_id).await?,
            _ => {
                warn!("未知的摘要类型: {}", job_data.digest_type);
                return Ok(());
            }
        };
        
        // 用户摘要应该通过其他方式处理（如邮件、推送），而不是保存到数据库
        if !digest_content.is_empty() {
            // TODO: 实现用户摘要的实际处理逻辑（邮件、推送等）
            info!("用户摘要生成完成: user_id={}, type={}, content_length={}", 
                  job_data.user_id, job_data.digest_type, digest_content.len());
        } else {
            info!("用户摘要为空，跳过处理: user_id={}, type={}", job_data.user_id, job_data.digest_type);
        }
        
        Ok(())
    }
}

/// 用户通知任务处理器
pub struct UserNotificationJobHandler {
    notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
    user_repository: Arc<crate::repositories::PgUserRepository>,
}

impl UserNotificationJobHandler {
    /// 创建新的用户通知任务处理器
    #[allow(dead_code)]
    pub fn new(
        notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
        user_repository: Arc<crate::repositories::PgUserRepository>,
    ) -> Self {
        Self {
            notification_service,
            user_repository,
        }
    }
}

impl JobHandler<NotificationJob> for UserNotificationJobHandler {
    async fn handle(&self, job: &Job<NotificationJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!(
            "处理用户通知任务: user_id={}, type={:?}",
            job_data.user_id, job_data.notification_type
        );
        
        // 检查用户是否存在
        let user = match self.user_repository.get_user_info(job_data.user_id).await {
            Ok(user) => user,
            Err(e) => {
                match e {
                    crate::repositories::RepositoryError::NotFound => {
                        warn!("用户不存在: user_id={}", job_data.user_id);
                        return Ok(());
                    }
                    _ => {
                        error!("查找用户失败: user_id={}, error={:?}", job_data.user_id, e);
                        return Err(JobHandlerError::Repository(e));
                    }
                }
            }
        };

        // 根据通知类型处理不同的通知逻辑
        match job_data.notification_type {
            crate::jobs::user_models::NotificationType::Welcome => {
                let message = format!("欢迎 {} 加入 GeoTalk！", user.nickname);
                let notification = serde_json::json!({
                    "title": "欢迎",
                    "content": message,
                    "type": "welcome"
                });
                if let Err(e) = self.notification_service.send_notification(
                    job_data.user_id,
                    &notification
                ).await {
                    error!("发送欢迎通知失败: user_id={}, error={:?}", job_data.user_id, e);
                    return Err(JobHandlerError::Notification(e));
                }
            }
            crate::jobs::user_models::NotificationType::LocationUpdate => {
                let notification = serde_json::json!({
                    "title": "位置更新",
                    "content": "您的位置信息已更新",
                    "type": "location_update"
                });
                if let Err(e) = self.notification_service.send_notification(
                    job_data.user_id,
                    &notification
                ).await {
                    error!("发送位置更新通知失败: user_id={}, error={:?}", job_data.user_id, e);
                    return Err(JobHandlerError::Notification(e));
                }
            }
            _ => {
                // 对于其他类型的通知，使用通用的处理逻辑
                let notification = serde_json::json!({
                    "title": "通知",
                    "content": job_data.payload,
                    "type": format!("{:?}", job_data.notification_type).to_lowercase()
                });
                if let Err(e) = self.notification_service.send_notification(
                    job_data.user_id,
                    &notification
                ).await {
                    error!("发送通知失败: user_id={}, type={:?}, error={:?}", 
                           job_data.user_id, job_data.notification_type, e);
                    return Err(JobHandlerError::Notification(e));
                }
            }
        }

        info!("用户通知任务处理完成: user_id={}", job_data.user_id);
        Ok(())
    }
}

/// 位置索引任务处理器
pub struct LocationIndexHandler {
    user_repository: Arc<crate::repositories::PgUserRepository>,
}

impl LocationIndexHandler {
    #[allow(dead_code)]
    pub fn new(user_repository: Arc<crate::repositories::PgUserRepository>) -> Self {
        Self { user_repository }
    }
}

impl JobHandler<LocationIndexJob> for LocationIndexHandler {
    async fn handle(&self, job: &Job<LocationIndexJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!(
            "处理位置索引任务: user_id={}, latitude={}, longitude={}",
            job_data.user_id, job_data.latitude, job_data.longitude
        );
        
        // 设置用户初始位置信息
        let location_name = job_data.location_name.as_deref().unwrap_or("Unknown");
        match self.user_repository.set_initial_user_location(
            job_data.user_id,
            job_data.latitude,
            job_data.longitude,
            location_name,
        ).await {
            Ok(_) => {
                info!(
                    "位置初始化成功: user_id={}, latitude={}, longitude={}",
                    job_data.user_id, job_data.latitude, job_data.longitude
                );
            }
            Err(e) => {
                error!(
                    "位置初始化失败: user_id={}, error={:?}",
                    job_data.user_id, e
                );
                return Err(JobHandlerError::Repository(e));
            }
        }
        
        Ok(())
    }
}
