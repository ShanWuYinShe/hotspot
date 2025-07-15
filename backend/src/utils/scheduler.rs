//! 定时任务调度器
use std::sync::Arc;
use tokio::time;
use tracing::{info, error};

use crate::jobs::JobService;
use crate::repositories::scheduler_repository::{PgSchedulerRepository, SchedulerRepository};

/// 启动定时任务调度器
pub async fn start_scheduler(
    job_service: Arc<JobService>,
    pg_pool: sqlx::PgPool,
    _redis_manager: Option<redis::aio::ConnectionManager>,
) {
    // 创建scheduler仓库 - 使用具体类型而不是 trait object
    let scheduler_repository = PgSchedulerRepository::new(pg_pool.clone());
    
    // 启动各个定时任务
    tokio::spawn(schedule_group_message_digest(job_service.clone(), scheduler_repository));
    
    info!("定时任务调度器已启动");
}

/// 定时任务：群组消息摘要
async fn schedule_group_message_digest(
    job_service: Arc<JobService>, 
    scheduler_repository: PgSchedulerRepository
) {
    // 每3小时执行一次
    let mut interval = time::interval(time::Duration::from_secs(3600 * 3));
    
    loop {
        interval.tick().await;
        info!("执行群组消息摘要定时任务");
        
        // 查找需要发送摘要的群组
        // 标准：活跃的群组，但有不活跃超过3天的成员
        match scheduler_repository.find_inactive_groups().await {
            Ok(groups) => {
                for (group_id, group_name, last_active_at) in groups {
                    if let Err(e) = job_service.dispatch_group_message_digest(
                        group_id,
                        group_name,
                        last_active_at.timestamp(),
                        3,  // 三天不活跃就发送摘要
                    )
                    .await
                    {
                        error!("创建群组消息摘要任务失败: {:?}", e);
                    }
                }
            }
            Err(e) => {
                error!("查询需要发送摘要的群组失败: {:?}", e);
            }
        }
    }
}

#[allow(dead_code)]
pub struct Scheduler<R: SchedulerRepository + ?Sized> {
    #[allow(dead_code)]
    repository: Arc<R>,
}

impl<R: SchedulerRepository + ?Sized> Scheduler<R> {
    #[allow(dead_code)]
    pub fn new(repository: Arc<R>) -> Self {
        Self { repository }
    }
    // ...existing code...
}
