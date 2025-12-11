mod config;
mod constants;
mod dto;
mod error;
mod events;
mod handlers;
mod middleware;
mod repositories;
mod routes;
mod services;
mod state;
mod types;
mod utils;
mod websocket;

use axum::Router;
use std::sync::Arc;
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::{
    config::Config,
    events::{event_bus::EventBus, event_handler::EventHandler},
    middleware::cors::create_cors_layer,
    state::AppState,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化日志
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,hotspot_backend=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    tracing::info!("Starting HotSpot Backend...");

    // 加载配置
    let config = Config::from_env()?;
    tracing::info!("Configuration loaded");

    // 创建事件总线
    let (event_bus, event_receiver) = EventBus::new();
    let event_bus = Arc::new(event_bus);
    tracing::info!("Event bus created");

    // 初始化应用状态
    let state = AppState::new(config.clone(), event_bus.clone()).await?;
    tracing::info!("Application state initialized");

    // 启动事件处理器
    let event_handler = EventHandler::new(
        state.db.clone(),
        state.redis.clone(),
        state.ws_manager.clone(),
        state.ws_subscription.clone(),
    );
    tokio::spawn(async move {
        event_handler.start(event_receiver).await;
    });
    tracing::info!("Event handler started");

    // 启动 WebSocket 超时检查任务
    let ws_manager_clone = state.ws_manager.clone();
    tokio::spawn(async move {
        use tokio::time::{interval, Duration};
        let mut check_interval = interval(Duration::from_secs(30));
        loop {
            check_interval.tick().await;
            let timeout_connections = ws_manager_clone.check_timeout().await;
            for connection_id in timeout_connections {
                tracing::warn!("WebSocket连接超时，移除连接: {}", connection_id);
                ws_manager_clone.remove_connection(&connection_id).await;
            }
        }
    });
    tracing::info!("WebSocket timeout checker started");

    // 创建路由
    // 明确挂载各版本 API，方便后续支持 v2、v3 等版本
    let app: Router = Router::new()
        .nest("/api/v1", routes::v1_routes(state.clone()))
        // 后续版本可在此添加：
        // .nest("/api/v2", routes::v2_routes(state.clone()))
        .merge(routes::ws_routes())
        .with_state(state)
        .layer(create_cors_layer())
        .layer(TraceLayer::new_for_http());

    // 启动服务器
    let addr = config.server_addr();
    tracing::info!("Server listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
