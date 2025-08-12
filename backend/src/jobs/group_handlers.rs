//! 群组相关任务处理器
use std::sync::Arc;
use chrono::Utc;
use tracing::{error, info};
use serde_json::json;

use crate::repositories::GroupRepository;
use crate::cache::GroupActivityCache;
use crate::services::notification::NotificationService;
use crate::jobs::handler::{JobHandler, JobHandlerResult};
use crate::jobs::group_models::{
    GroupMessageDigestJob, GroupMemberActivityJob
};
use crate::utils::job_queue::Job;

/// 群组消息摘要任务处理器
pub struct GroupDigestHandler<R: GroupRepository + ?Sized> {
    group_repository: Arc<R>,
    notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
}

impl<R: GroupRepository + ?Sized> GroupDigestHandler<R> {
    /// 创建新的群组消息摘要任务处理器
    pub fn new(
        group_repository: Arc<R>,
        notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
    ) -> Self {
        Self {
            group_repository,
            notification_service,
        }
    }
}

impl<R: GroupRepository + ?Sized> JobHandler<GroupMessageDigestJob> for GroupDigestHandler<R> {
    async fn handle(&self, job: &Job<GroupMessageDigestJob>) -> JobHandlerResult {
        let job_data = &job.data;
        
        info!(
            "处理群组消息摘要任务: group_id={}, group_name={}",
            job_data.group_id, job_data.group_name
        );
        
        // 查找不活跃的群组成员
        let inactive_members = self.group_repository
            .find_inactive_members(job_data.group_id, job_data.min_inactivity_days)
            .await?;
        
        if inactive_members.is_empty() {
            info!("群组 {} 没有不活跃成员，跳过发送摘要", job_data.group_name);
            return Ok(());
        }
        
        // 生成消息摘要
        let digest = self.group_repository
            .generate_message_digest(job_data.group_id, job_data.last_active_at)
            .await?;
        
        // 为不活跃成员发送通知
        for (user_id, username) in inactive_members {
            info!("为用户 {} 发送群组 {} 的消息摘要", username, job_data.group_name);
            
            let notification_payload = json!({
                "type": "group_digest",
                "group_id": job_data.group_id.to_string(),
                "group_name": job_data.group_name,
                "digest": digest
            });
            
            if let Err(e) = self.notification_service
                .send_notification(user_id, &notification_payload)
                .await
            {
                error!(
                    "向用户 {} 发送群组 {} 的消息摘要失败: {:?}",
                    username, job_data.group_name, e
                );
            }
        }
        
        info!("群组 {} 的消息摘要任务处理完成", job_data.group_name);
        Ok(())
    }
}

/// 群组成员活跃度更新任务处理器
pub struct GroupMemberActivityHandler {
    /// 群组活跃度缓存
    group_activity_cache: Arc<crate::cache::group_activity_cache::RedisGroupActivityCache>,
}

impl GroupMemberActivityHandler {
    /// 创建新的群组成员活跃度更新任务处理器
    pub fn new(
        group_activity_cache: Arc<crate::cache::group_activity_cache::RedisGroupActivityCache>,
    ) -> Self {
        Self {
            group_activity_cache,
        }
    }
}

impl JobHandler<GroupMemberActivityJob> for GroupMemberActivityHandler {
    async fn handle(&self, job: &Job<GroupMemberActivityJob>) -> JobHandlerResult {
        let job_data = &job.data;
        
        info!(
            "处理群组成员活跃度更新任务: group_id={}, user_id={}, activity_type={:?}",
            job_data.group_id, job_data.user_id, job_data.activity_type
        );
        
        // 获取活动分数
        let _score = job_data.activity_type.get_score();
        
        // 更新用户的群组活跃度
        if let Err(e) = self.group_activity_cache
            .update_user_last_active_in_group(job_data.user_id, job_data.group_id, Utc::now().timestamp())
            .await
        {
            error!("更新群组活跃度失败: {:?}", e);
        }
        Ok(())
    }
}