mod config;
mod controllers;
mod middleware;
mod models;
mod routes;
mod services;
mod repositories;
#[cfg(feature = "jobs")]
mod jobs;
mod cache;
mod utils;
mod validators;
mod ws;

use axum::{Router, http::Method, routing::{get, post}};
use config::app_config::AppConfig;
// use config::api_doc::ApiDoc;  // 暂时禁用以修复编译问题
use config::db::Database;
use controllers::{health_check, metrics_handler};
use middleware::auth::{AppState, auth_middleware};
use middleware::app_state_builder::AppStateBuilder;
use routes::{admin_routes, auth_routes, moment_routes, group_routes, message_routes, user_routes};
use routes::auth_routes::auth_protected_routes;
use services::{
    MomentService, ConfigService, GroupService, HealthService, MessageService, 
    UserService,
};

#[cfg(feature = "jobs")]
use jobs::JobService;
use repositories::{PgUserRepository, PgGroupRepository, PgHealthRepository, PgMomentRepository};
use std::{net::SocketAddr, sync::Arc};
use tokio::net::TcpListener;
use tower_http::cors::{CorsLayer, Any};
use tower_http::trace::TraceLayer;
use tracing::info;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::cache::CacheManager;
use ws::handler::ws_handler;
use ws::manager::WebSocketManager;
use ws::message_router::MessageRouter;
use ws::connection_manager::ConnectionManager;
use ws::room_manager::RoomManager;
use utils::metrics::AppMetrics;
#[cfg(feature = "audit")]
use utils::audit::{AuditService, NoopAuditSink};
// use utils::events::EventBus; // 保留注释以提示可用导入，但避免未使用警告
#[cfg(feature = "audit")]
use utils::events::audit_handler::AuditEventsHandler;
#[cfg(feature = "ws_hooks")]
use utils::events::ws_handler::WsEventsHandler;
#[cfg(feature = "jobs")]
use utils::events::jobs_handler::JobsEventsHandler;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化日志
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    info!("初始化配置...");
    let app_config = AppConfig::from_env()?;
    app_config.validate_and_print();

    info!("连接数据库...");
    let db = Database::from_config(&app_config).await?;
    
    info!("执行数据库健康检查...");
    db.check_health().await?;

    // 初始化 CacheManager (Redis连接来自db)
    let cache_manager = Arc::new(CacheManager::new(
        db.redis_client.clone(), 
        db.redis_manager.clone().map(Arc::new)
    ));

    // 服务初始化 (ConfigService, Repositories, Business Services)
    let _config_service = Arc::new(ConfigService::new(app_config.clone()));
    let _user_repository = Arc::new(PgUserRepository::new(db.pg_pool.clone()));

    // --- WebSocket组件初始化 (统一管理) ---
    info!("正在初始化WebSocket组件...");
    let connection_manager = Arc::new(ConnectionManager::new());
    let room_manager = Arc::new(RoomManager::new());
    let message_router = Arc::new(MessageRouter::new(
        connection_manager.clone(),
        room_manager.clone()
    ));
    
    // 根据环境选择通知策略
    let (notification_policy, group_notification_policy) = match std::env::var("RUST_ENV").unwrap_or_default().as_str() {
        "production" => (
            crate::services::notification::NotificationManager::production_policy(),
            crate::services::notification::NotificationManager::production_group_policy()
        ),
        "development" => (
            crate::services::notification::NotificationManager::development_policy(),
            crate::services::notification::NotificationManager::development_group_policy()
        ),
        _ => {
            // 默认使用开发策略，但可以通过环境变量覆盖全局广播设置
            let mut policy = crate::services::notification::NotificationManager::development_policy();
            policy.enable_global_broadcast = std::env::var("ENABLE_GLOBAL_NOTIFICATION_BROADCAST")
                .unwrap_or_default()
                .parse()
                .unwrap_or(false);
            (policy, crate::services::notification::NotificationManager::development_group_policy())
        }
    };
    
    // 首先创建数据库repositories
    let config_service = Arc::new(ConfigService::new(app_config.clone()));
    let user_repository = Arc::new(PgUserRepository::new(db.pg_pool.clone()));
    let group_repository = Arc::new(PgGroupRepository::new(db.pg_pool.clone()));
    
    // 创建通知管理器
    // 构建离线通知缓存（如果可用）并注入
    let notification_cache: Option<Arc<dyn crate::cache::NotificationCache>> =
        cache_manager
            .get_redis_connection_manager()
            .map(|conn| {
                let cm_arc: Arc<redis::aio::ConnectionManager> = (*conn).clone().into();
                Arc::new(crate::cache::notification_cache::RedisNotificationCache::new(Some(cm_arc))) as Arc<dyn crate::cache::NotificationCache>
            });

    let notification_manager = Arc::new(crate::services::notification::NotificationManager::new(
        message_router.clone(),
        user_repository.clone(),
        group_repository.clone(),
        Some(notification_policy),
        Some(group_notification_policy),
        notification_cache,
    ));
    
    let websocket_notification_service = notification_manager.websocket_service();
    
    info!("通知策略: {}", notification_manager.get_policy_summary());
    
    // 注意：metrics 尚未创建，先用 None，稍后在 state 创建后可按需替换/注入
    let ws_manager = Arc::new(WebSocketManager::new(
        message_router.clone(),
        connection_manager.clone(),
        room_manager.clone(),
        db.pg_pool.clone(),
        websocket_notification_service.clone(),
        None,
        #[cfg(feature = "audit")]
        Some(AuditService::new(Arc::new(NoopAuditSink::new()))),
    ));
    // --- WebSocket组件初始化完成 ---

    // 初始化作业队列 (JobService)
    #[cfg(feature = "jobs")]
    let job_service = if app_config.job_queue.enabled {
        let amqp_url = app_config.job_queue.amqp_url.as_ref()
            .expect("AMQP URL is required when job queue is enabled");
        match JobService::new(
            db.clone(), 
            cache_manager.clone(), 
            websocket_notification_service.clone(), 
            amqp_url, 
        ).await {
            Ok(service) => Some(Arc::new(service)),
            Err(e) => {
                 tracing::warn!("Failed to initialize JobService: {}. Async jobs will be disabled.", e);
                None
            }
        }
    } else {
        info!("异步任务队列已禁用");
        None
    };
    #[cfg(not(feature = "jobs"))]
    let _job_service = None::<()>;

    // 服务初始化 (Business Services)
    let user_service = {
        let service = UserService::new(config_service.clone(), user_repository.clone());
        #[cfg(feature = "jobs")]
        if let Some(ref job_service) = job_service {
            service = service.with_job_dispatcher(job_service.get_dispatcher());
        }
        Arc::new(service)
    };
    
    let group_service = Arc::new(GroupService::new(db.clone())); 
    let message_service = Arc::new(MessageService::new(db.clone(), group_repository.clone()));
    
    // 创建moment repository和service
    let moment_repository = Arc::new(PgMomentRepository::new(db.pg_pool.clone()));
    let moment_service = Arc::new(MomentService::new(moment_repository));
    
    let health_repository = Arc::new(PgHealthRepository::new(
        db.pg_pool.clone(), 
        db.redis_client.as_ref().map(|client| (**client).clone())
    ));
    let health_service = Arc::new(HealthService::new(health_repository));

    // 创建统一的应用状态（Builder）
    let metrics = Arc::new(AppMetrics::new()?);
    let builder = AppStateBuilder::new(
        config_service.clone(),
        user_service.clone(),
        user_repository.clone(),
        group_service.clone(),
        message_service.clone(),
        moment_service.clone(),
        health_service.clone(),
        websocket_notification_service.clone(),
        ws_manager.clone(),
        metrics.clone(),
    );
    #[cfg(feature = "jobs")]
    let builder = builder.with_job_service(job_service.clone());
    let state: Arc<AppState> = builder.build();

    // 注册事件处理器（按 feature）
    #[cfg(feature = "audit")]
    {
        let audit_handler = Arc::new(AuditEventsHandler::new(Arc::new(AuditService::new(Arc::new(NoopAuditSink::new())))));
        state.event_bus.register_audit_handler(audit_handler);
    }
    #[cfg(feature = "ws_hooks")]
    {
        let ws_handler = Arc::new(WsEventsHandler::new(websocket_notification_service.clone()));
        state.event_bus.register_ws_handler(ws_handler);
    }
    #[cfg(feature = "jobs")]
    if let Some(ref jobs) = job_service {
        let jobs_handler = Arc::new(JobsEventsHandler::new(jobs.clone()));
        state.event_bus.register_jobs_handler(jobs_handler);
    }
    
    // 启动定时任务调度器
    #[cfg(feature = "jobs")]
    if let Some(ref jobs) = job_service { // Changed to ref jobs
        info!("启动定时任务调度器...");
        let redis_opt = cache_manager.get_redis_connection_manager().map(|m| (*m).clone());
        utils::scheduler::start_scheduler(
            jobs.clone(),
            db.pg_pool.clone(),
            redis_opt,
            Some(state.ws_manager.clone()),
            state.config_service.clone(),
        ).await;
    }

    // CORS 配置 - 从ConfigService获取
    let cors = if state.config_service.cors.permissive_mode {
        CorsLayer::very_permissive()
    } else {
        CorsLayer::new()
            .allow_methods(
                state.config_service.cors.allowed_methods
                    .iter()
                    .map(|m| Method::from_bytes(m.as_bytes()).unwrap())
                    .collect::<Vec<_>>()
            )
            .allow_origin(Any)
            .allow_headers(Any)
    };

    // 需要认证的API路由
    let protected_routes = Router::new()
        .nest("/api/v1/user", user_routes())
        .nest("/api/v1/group", group_routes())
        .nest("/api/v1/chat", message_routes())
        .nest("/api/v1/moments", moment_routes())
        .nest("/api/v1/admin", admin_routes())
        .nest("/api/v1/auth", auth_protected_routes())  // 需要认证的auth相关路由
        .layer(axum::middleware::from_fn_with_state(
            state.clone(),
            auth_middleware,
        ));

    // 公开路由（不需要认证的认证相关接口）
    let public_routes = Router::new()
        .nest("/api/v1/auth", auth_routes());

    // 完整应用路由 - WebSocket路由不经过认证中间件
    let app = Router::new()
        .merge(protected_routes)
        .merge(public_routes)
        .route("/ws/", get(ws_handler)) // WebSocket需要保留GET方法
        .route("/health", post(health_check)) // 改为POST方法
        .route("/metrics", post(metrics_handler)) // 改为POST方法
        .layer(cors)
        .layer(TraceLayer::new_for_http())
        .with_state(state.clone());

    // 启动服务器
    let port = app_config.server.port;
    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    info!("启动服务器，监听端口 {}", port);

    let listener = TcpListener::bind(&addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
