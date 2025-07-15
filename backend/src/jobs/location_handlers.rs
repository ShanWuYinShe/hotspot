//! 位置相关任务处理器
use std::sync::Arc;
use tracing::{error, info, warn};


use crate::cache::location_cache::LocationCache;

use crate::jobs::handler::{JobHandler, JobHandlerResult, JobHandlerError};
use crate::jobs::location_models::{
    CheckinIndexJob, NearbyCheckinNotificationJob
};
use crate::utils::job_queue::Job;
use crate::ws::message_types::{SystemNotificationData, SystemNotificationType as WsSystemNotificationType};
use uuid::Uuid;
use chrono::Utc;

/// 签到索引任务处理器
pub struct CheckinIndexHandler<C: LocationCache + ?Sized> {
    /// 位置缓存
    location_cache: Option<Arc<C>>,
}

impl<C: LocationCache + ?Sized> CheckinIndexHandler<C> {
    /// 创建新的签到索引任务处理器
    pub fn new(
        location_cache: Option<Arc<C>>,
    ) -> Self {
        Self {
            location_cache,
        }
    }
}

impl<C: LocationCache + ?Sized + Send + Sync> JobHandler<CheckinIndexJob> for CheckinIndexHandler<C> {
    async fn handle(&self, job: &Job<CheckinIndexJob>) -> JobHandlerResult {
        if let Some(cache) = &self.location_cache {
            let job_data = &job.data;
            info!(
                "处理签到索引更新任务: checkin_id={}, user_id={}, position=({}, {})",
                job_data.checkin_id, job_data.user_id, job_data.latitude, job_data.longitude
            );
            // 将签到位置保存到缓存中，设置30天过期时间
            cache.store_checkin_location(
                job_data.checkin_id.clone(),
                job_data.user_id,
                job_data.latitude,
                job_data.longitude,
                30 // 30天过期
            ).await
                .map_err(JobHandlerError::Cache)?;
            Ok(())
        } else {
            warn!("LocationCache not available, skipping CheckinIndexJob for checkin_id: {}", job.data.checkin_id);
            Ok(()) // Or return an error if this job cannot be skipped
        }
    }
}

/// 附近签到通知任务处理器
pub struct NearbyCheckinNotificationHandler {
    /// 通知服务
    notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
}

impl NearbyCheckinNotificationHandler {
    /// 创建新的附近签到通知任务处理器
    pub fn new(
        notification_service: Arc<crate::services::notification::WebSocketNotificationService>,
    ) -> Self {
        Self {
            notification_service,
        }
    }
}

impl JobHandler<NearbyCheckinNotificationJob> for NearbyCheckinNotificationHandler {
    async fn handle(&self, job: &Job<NearbyCheckinNotificationJob>) -> JobHandlerResult {
        let job_data = &job.data;
        info!(
            "处理附近签到通知任务: checkin_id={}, creator_id={}, nearby_users={}",
            job_data.checkin_id, job_data.creator_id, job_data.nearby_user_ids.len()
        );
        
        for user_id_to_notify in &job_data.nearby_user_ids {
            // Construct SystemNotificationData for each user
            let notification_data = SystemNotificationData {
                notification_id: Uuid::new_v4(), // Unique ID for this specific notification instance
                notification_type: WsSystemNotificationType::NewCheckIn, // Or a more specific type like NearbyCheckIn
                title: "附近的签到".to_string(),
                content: format!("{} 在您附近签到了: {}", job_data.creator_nickname, job_data.content.chars().take(50).collect::<String>() ),
                data: Some(serde_json::json!({
                    "checkin_id": job_data.checkin_id,
                    "creator_id": job_data.creator_id,
                    "creator_nickname": job_data.creator_nickname,
                    "latitude": job_data.latitude,
                    "longitude": job_data.longitude,
                    "content_preview": job_data.content.chars().take(50).collect::<String>(), // Preview
                    "original_content": job_data.content, // Full content if needed by client
                    "timestamp": job_data.timestamp // Timestamp from the job data if available, else Utc::now()
                })),
                created_at: Utc::now(), // Timestamp of this notification event
            };

            if let Err(e) = self.notification_service.notify_system_event_to_user(*user_id_to_notify, notification_data.clone()).await { // clone data if needed for retries/logging
                error!(
                    "Failed to send nearby check-in notification to user {}: {:?}. Checkin ID: {}", 
                    user_id_to_notify, e, job_data.checkin_id
                );
                // Decide if to continue to next user or retry/fail job. For now, just log and continue.
            }
        }
        Ok(())
    }
}
