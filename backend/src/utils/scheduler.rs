//! 定时任务调度器
use std::sync::Arc;
use tokio::time;
use tracing::{info, error};

use crate::jobs::JobService;
use crate::repositories::scheduler_repository::{PgSchedulerRepository, SchedulerRepository};
use crate::ws::manager::WebSocketManager;
use crate::services::config::service::ConfigService;

/// 启动定时任务调度器
pub async fn start_scheduler(
    job_service: Arc<JobService>,
    pg_pool: sqlx::PgPool,
    _redis_manager: Option<redis::aio::ConnectionManager>,
    ws_manager: Option<Arc<WebSocketManager>>,
    config: Arc<ConfigService>,
) {
    // 创建scheduler仓库 - 使用具体类型而不是 trait object
    let scheduler_repository = PgSchedulerRepository::new(pg_pool.clone());
    
    // 启动各个定时任务
    #[cfg(feature = "group_digest_scheduler")]
    tokio::spawn(schedule_group_message_digest(job_service.clone(), scheduler_repository));
    #[cfg(not(feature = "group_digest_scheduler"))]
    let _ = (job_service, scheduler_repository);
    // 清理不活跃的 WebSocket 连接和空房间
    #[cfg(feature = "ws_scheduler")]
    if let Some(ws) = ws_manager.clone() {
        tokio::spawn(schedule_ws_cleanup(ws.clone(), config.clone()));
        tokio::spawn(schedule_ws_empty_room_cleanup(ws, config.clone()));
    }
    #[cfg(not(feature = "ws_scheduler"))]
    let _ = (ws_manager, config);
    
    info!("定时任务调度器已启动");
}
#[allow(dead_code)]
async fn schedule_ws_cleanup(ws_manager: Arc<WebSocketManager>, config: Arc<ConfigService>) {
    let mut interval = time::interval(time::Duration::from_secs(
        config.config.websocket.cleanup_interval_secs,
    ));
    loop {
        interval.tick().await;
        let removed = ws_manager
            .cleanup_inactive_connections(config.config.websocket.inactive_timeout_secs * 1000)
            .await;
        // 记录清理数量指标
        // 无法直接拿到 metrics，此处仅记录日志；如需指标，请注入 metrics 或通过回调上报
        if removed > 0 {
            info!("清理不活跃连接数量: {}", removed);
        }
    }
}

#[allow(dead_code)]
async fn schedule_ws_empty_room_cleanup(ws_manager: Arc<WebSocketManager>, _config: Arc<ConfigService>) {
    // 复用与连接清理一致的频率；如需单独频率，可扩展配置
    let mut interval = time::interval(time::Duration::from_secs(
        _config.config.websocket.cleanup_interval_secs,
    ));
    loop {
        interval.tick().await;
        let removed = ws_manager.cleanup_empty_rooms().await;
        if removed > 0 {
            info!("清理空房间数量: {}", removed);
        }
    }
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
