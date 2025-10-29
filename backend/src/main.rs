mod config;
mod error;
mod state;
mod utils;
mod middleware;
mod dto;
mod models;
mod repositories;
mod services;
mod handlers;
mod routes;
mod websocket;
mod events;

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
    );
    tokio::spawn(async move {
        event_handler.start(event_receiver).await;
    });
    tracing::info!("Event handler started");

    // 创建路由
    let app = Router::new()
        .nest("/api/v1", routes::create_routes(state.clone()))
        .layer(create_cors_layer())
        .layer(TraceLayer::new_for_http());

    // 启动服务器
    let addr = config.server_addr();
    tracing::info!("Server listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

