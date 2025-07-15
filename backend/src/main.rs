mod config;
mod controllers;
mod middleware;
mod models;
mod routes;
mod services;
mod repositories;
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
use routes::{admin_routes, checkin_routes, group_routes, message_routes, user_routes};
use services::{
    CheckinService, ConfigService, GroupService, HealthService, MessageService, 
    UserService,
};

use jobs::JobService;
use repositories::{PgUserRepository, PgGroupRepository, PgHealthRepository};
use std::{net::SocketAddr, sync::Arc};
use tokio::net::TcpListener;
use tower_http::cors::{CorsLayer, Any};
use tower_http::trace::TraceLayer;
use tracing::{info, warn};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::cache::CacheManager;
use ws::handler::ws_handler;
use ws::manager::WebSocketManager;
use ws::message_router::MessageRouter;
use ws::connection_manager::ConnectionManager;
use ws::room_manager::RoomManager;
// use crate::services::notification::events::GroupEvent; // Removed EventBus related import
// use crate::services::event_bus::EventBus; // Removed EventBus related import
// use ws::session::SessionManager; // Commented out if not used

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
    let notification_manager = Arc::new(crate::services::notification::NotificationManager::new(
        message_router.clone(),
        user_repository.clone(),
        group_repository.clone(),
        Some(notification_policy),
        Some(group_notification_policy),
    ));
    
    let websocket_notification_service = notification_manager.websocket_service();
    
    info!("通知策略: {}", notification_manager.get_policy_summary());
    
    let ws_manager = Arc::new(WebSocketManager::new(
        message_router.clone(),
        connection_manager.clone(),
        room_manager.clone(),
    ));
    // --- WebSocket组件初始化完成 ---

    // 初始化作业队列 (JobService)
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
                warn!("Failed to initialize JobService: {}. Async jobs will be disabled.", e);
                None
            }
        }
    } else {
        info!("异步任务队列已禁用");
        None
    };

    // 服务初始化 (Business Services)
    let mut user_service = UserService::new(config_service.clone(), user_repository.clone());
    if let Some(ref job_service) = job_service {
        user_service = user_service.with_job_dispatcher(job_service.get_dispatcher());
    }
    let user_service = Arc::new(user_service);
    
    let group_service = Arc::new(GroupService::new(db.clone())); 
    let message_service = Arc::new(MessageService::new(db.clone(), group_repository.clone()));
    let checkin_service = Arc::new(CheckinService::new(db.clone()));
    
    let health_repository = Arc::new(PgHealthRepository::new(
        db.pg_pool.clone(), 
        db.redis_client.as_ref().map(|client| (**client).clone())
    ));
    let health_service = Arc::new(HealthService::new(health_repository));

    // 创建统一的应用状态
    let state = Arc::new(AppState {
        config_service,
        user_service,
        user_repository,
        group_service,
        message_service,
        checkin_service,
        health_service,
        websocket_notification_service,
        ws_manager,
        job_service: job_service.clone(),
    });
    
    // 启动定时任务调度器
    if let Some(ref jobs) = job_service { // Changed to ref jobs
        info!("启动定时任务调度器...");
        if let Some(redis_conn_manager) = cache_manager.get_redis_connection_manager() { // Use CacheManager
            utils::scheduler::start_scheduler(
                jobs.clone(), // Pass the entire JobService
                db.pg_pool.clone(),
                Some((*redis_conn_manager).clone()),
            ).await;
        } else {
            warn!("Redis Connection Manager 不可用，跳过定时任务调度器");
        }
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
        .nest("/api/v1/checkin", checkin_routes())
        .nest("/api/v1/admin", admin_routes())
        .layer(axum::middleware::from_fn_with_state(
            state.clone(),
            auth_middleware,
        ));

    // 完整应用路由 - WebSocket路由不经过认证中间件
    let app = Router::new()
        .merge(protected_routes)
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
