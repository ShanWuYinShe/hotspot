//! 作业服务模块
//! 作业处理模块 - 使用模块化架构实现

pub mod handler;
pub mod dispatcher;
pub mod group_models;
pub mod group_handlers;
pub mod location_models;
pub mod location_handlers;
pub mod user_models;
pub mod user_handlers;

use std::sync::Arc;

use tracing::{info, warn};

use crate::cache::RedisGroupActivityCache;
use crate::cache::location_cache::RedisLocationCache;
use crate::services::notification::WebSocketNotificationService;

use crate::repositories::PgGroupRepository;
use crate::repositories::PgLocationRepository;
use crate::repositories::user_repository::PgUserRepository;

use crate::utils::job_queue::JobQueue;
use self::group_handlers::{GroupDigestHandler, GroupMemberActivityHandler};
use self::location_handlers::{CheckinIndexHandler, NearbyCheckinNotificationHandler};
use self::user_handlers::{WelcomeEmailHandler, NotificationHandler, LocationIndexHandler};
use self::group_models::GroupActivityType;
use self::dispatcher::JobDispatcher;
use crate::config::db::Database;
use crate::cache::CacheManager;

/// 作业服务，用于管理异步任务
pub struct JobService {
    /// 任务队列
    #[allow(dead_code)]
    pub job_queue: Arc<JobQueue>,
    /// 任务调度器
    dispatcher: Arc<JobDispatcher>,
}

impl JobService {
    /// 创建作业服务实例 - 需要提供必要的依赖
    pub async fn new(
        db: Database,
        cache_manager: Arc<CacheManager>,
        websocket_notification_service: Arc<WebSocketNotificationService>,
        amqp_url: &str,
    ) -> Result<Self, lapin::Error> {
        let job_queue = Arc::new(JobQueue::new(amqp_url).await?);
        
        let pg_pool = db.pg_pool.clone();

        let group_repository = Arc::new(PgGroupRepository::new(pg_pool.clone()));
        let _location_repository = Arc::new(PgLocationRepository::new(pg_pool.clone()));
        let user_repository = Arc::new(PgUserRepository::new(pg_pool.clone()));
        
        let group_activity_cache = Arc::new(RedisGroupActivityCache::new(
            cache_manager.get_redis_connection_manager() // Returns Option<Arc<ConnectionManager>>
        ));
        
        // RedisLocationCache needs Arc<Client>. Handle Option if Redis is disabled.
        let location_cache = if let Some(redis_client_arc) = cache_manager.get_redis_client() {
            Some(Arc::new(RedisLocationCache::new(redis_client_arc)))
        } else {
            warn!("Redis Client not available, RedisLocationCache will not be initialized.");
            None // Or handle appropriately if RedisLocationCache is mandatory for some jobs
        };

        let notification_service = websocket_notification_service;

        let group_digest_handler = Arc::new(GroupDigestHandler::new(
            group_repository.clone(),
            notification_service.clone(),
        ));
        let group_activity_handler = Arc::new(GroupMemberActivityHandler::new(
            group_activity_cache.clone(),
        ));
        let checkin_index_handler = Arc::new(CheckinIndexHandler::new(
            location_cache.clone(), // Pass Option<Arc<RedisLocationCache>>
        ));
        let nearby_checkin_notification_handler = Arc::new(NearbyCheckinNotificationHandler::new(
            notification_service.clone(),
        ));
        let welcome_email_handler = Arc::new(WelcomeEmailHandler::new(
            user_repository.clone(),
            notification_service.clone(),
        ));
        let notification_handler = Arc::new(NotificationHandler::new(
            notification_service.clone(),
        ));
        let location_index_handler = Arc::new(LocationIndexHandler::new(
            user_repository.clone(),
        ));
        let dispatcher = Arc::new(JobDispatcher::new(
            job_queue.clone(),
            Some(group_digest_handler),
            Some(group_activity_handler),
            Some(checkin_index_handler),
            Some(nearby_checkin_notification_handler),
            Some(welcome_email_handler),
            Some(notification_handler),
            Some(location_index_handler),
        ));
        dispatcher.register_handlers().await?;
        info!("作业服务已成功初始化并注册处理器");
        Ok(Self { 
            job_queue,
            dispatcher
        })
    }
    
    /// 分派群组消息摘要任务
    #[allow(dead_code)]
    pub async fn dispatch_group_message_digest(
        &self,
        group_id: uuid::Uuid,
        group_name: String,
        last_active_at: i64,
        min_inactivity_days: i32,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_group_message_digest(
            group_id, 
            group_name, 
            last_active_at, 
            min_inactivity_days
        ).await
    }
    
    /// 分派群组成员活跃度更新任务
    pub async fn dispatch_group_member_activity(
        &self,
        group_id: uuid::Uuid,
        user_id: i64,
        activity_type: &str,
    ) -> Result<(), lapin::Error> {
        let activity = match activity_type {
            "message_sent" => GroupActivityType::MessageSent,
            "message_read" => GroupActivityType::MessageRead,
            "join" => GroupActivityType::Join,
            "leave" => GroupActivityType::Leave,
            "reaction" => GroupActivityType::Reaction,
            "event_create" => GroupActivityType::EventCreate,
            "event_join" => GroupActivityType::EventJoin,
            "media_share" => GroupActivityType::MediaShare,
            _ => GroupActivityType::MessageRead,
        };
        
        self.dispatcher.dispatch_group_member_activity(
            group_id, 
            user_id, 
            activity
        ).await
    }
    
    /// 分派签到索引任务
    pub async fn dispatch_checkin_index(
        &self,
        checkin_id: String,
        user_id: i64,
        latitude: f64,
        longitude: f64,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_checkin_index(
            checkin_id,
            user_id,
            latitude,
            longitude
        ).await
    }
    
    /// 分派附近签到通知任务
    pub async fn dispatch_nearby_checkin_notification(
        &self,
        checkin_id: String,
        creator_id: i64,
        creator_nickname: String,
        nearby_user_ids: Vec<i64>,
        latitude: f64,
        longitude: f64,
        content: String,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_nearby_checkin_notification(
            checkin_id,
            creator_id,
            creator_nickname,
            nearby_user_ids,
            latitude,
            longitude,
            content
        ).await
    }
    
    /// 分派欢迎邮件任务
    pub async fn dispatch_welcome_email(
        &self,
        user_id: i64,
        nickname: String,
        email: Option<String>,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_welcome_email(
            user_id,
            nickname,
            email
        ).await
    }
    
    /// 分派通知推送任务
    pub async fn dispatch_notification(
        &self,
        user_id: i64,
        notification_type: &str,
        payload: String,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_notification(
            user_id,
            notification_type,
            payload
        ).await
    }
    
    /// 获取任务调度器
    pub fn get_dispatcher(&self) -> Arc<JobDispatcher> {
        self.dispatcher.clone()
    }
    
    /// 分派位置索引任务
    pub async fn dispatch_location_index(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: Option<String>,
    ) -> Result<(), lapin::Error> {
        self.dispatcher.dispatch_location_index(
            user_id,
            latitude,
            longitude,
            location_name
        ).await
    }
}
