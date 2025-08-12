//! 位置相关任务处理器
use std::sync::Arc;
use tracing::{info, warn};


use crate::cache::location_cache::LocationCache;

use crate::jobs::handler::{JobHandler, JobHandlerResult, JobHandlerError};
use super::location_models::MomentIndexJob;
use crate::utils::job_queue::Job;
// 无需通知/UUID/时间导入

/// 签到索引任务处理器
pub struct MomentIndexHandler<C: LocationCache + ?Sized> {
    /// 位置缓存
    location_cache: Option<Arc<C>>,
}

impl<C: LocationCache + ?Sized> MomentIndexHandler<C> {
    /// 创建新的签到索引任务处理器
    pub fn new(
        location_cache: Option<Arc<C>>,
    ) -> Self {
        Self {
            location_cache,
        }
    }
}

impl<C: LocationCache + ?Sized + Send + Sync> JobHandler<MomentIndexJob> for MomentIndexHandler<C> {
    async fn handle(&self, job: &Job<MomentIndexJob>) -> JobHandlerResult {
        if let Some(cache) = &self.location_cache {
            let job_data = &job.data;
            info!(
                "处理动态索引更新任务: moment_id={}, user_id={}, position=({}, {})",
                job_data.moment_id, job_data.user_id, job_data.latitude, job_data.longitude
            );
            // 将动态位置保存到缓存中，设置30天过期时间
            cache.store_user_location(
                job_data.user_id,
                job_data.latitude,
                job_data.longitude,
                30 * 24 * 60 * 60 // 30天过期（秒）
            ).await
                .map_err(JobHandlerError::Cache)?;
            Ok(())
        } else {
            warn!("LocationCache not available, skipping MomentIndexJob for moment_id: {}", job.data.moment_id);
            Ok(()) // Or return an error if this job cannot be skipped
        }
    }
}

//
