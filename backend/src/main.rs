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

use axum::body::Body;
use axum::Router;
use std::sync::Arc;
use std::time::Duration;
use tower_http::trace::TraceLayer;
use tracing::Span;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::{
    config::Config,
    events::{event_bus::EventBus, event_handler::EventHandler},
    state::AppState,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化日志
    // 目标：日志“人类可读”，且每条请求链路日志都能看到 request_id，但不输出 span_list / 海量 DEBUG/TRACE 噪声
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| {
                // 默认：业务日志为主；屏蔽 tower_http/axum/tungstenite 的噪音
                // 仍允许用户用 RUST_LOG 覆盖
                "info,sqlx=warn,hotspot_backend=info,tower_http=warn,axum=warn,tokio_tungstenite=warn".into()
            }),
        )
        .with(
            tracing_subscriber::fmt::layer()
                .compact()
                // 输出 target：便于按 target 精细设置 RUST_LOG（例如：hotspot_backend::services::auth_service=debug）
                .with_target(true)
                .with_thread_ids(false)
                .with_thread_names(false), // 事件在 span 内时会带上 span（http_request{request_id=...}: ...）
        )
        .init();

    tracing::info!("Starting HotSpot Backend...");

    // 加载配置
    let config = Config::from_env()?;
    tracing::debug!("Configuration loaded");

    // 创建事件总线
    let (event_bus, event_receiver) = EventBus::new();
    let event_bus = Arc::new(event_bus);
    tracing::debug!("Event bus created");

    // 初始化应用状态
    let state = AppState::new(config.clone(), event_bus.clone()).await?;
    tracing::debug!("Application state initialized");

    // 启动事件处理器
    let event_handler = EventHandler::new(
        state.db.clone(),
        state.redis.clone(),
        state.ws_manager.clone(),
        state.ws_subscription.clone(),
        event_bus.clone(),
        state.media_service.clone(),
    );
    tokio::spawn(async move {
        event_handler.start(event_receiver).await;
    });
    tracing::debug!("Event handler started");

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
    tracing::debug!("WebSocket timeout checker started");

    // 创建路由
    // 明确挂载各版本 API，方便后续支持 v2、v3 等版本
    let app: Router = Router::new()
        .merge(routes::ws_routes())
        .nest("/api/v1", routes::v1_routes(state.clone()))
        // 后续版本可在此添加：
        // .nest("/api/v2", routes::v2_routes(state.clone()))
        .with_state(state)
        .layer(
            TraceLayer::new_for_http()
                // 让 “request span” 本身就包含 request_id，避免额外 span_list
                .make_span_with(|req: &axum::http::Request<Body>| {
                    let request_id = req
                        .headers()
                        .get("x-request-id")
                        .and_then(|v| v.to_str().ok())
                        .unwrap_or("-");

                    // span 只保留 request_id：便于全局 grep，但不会让每条日志都携带 method/uri 等“噪声”
                    tracing::info_span!("http_request", request_id = %request_id)
                })
                // 不使用 DefaultOnRequest/Response：它们的 target 在 tower_http::* 下，容易被 EnvFilter 过滤掉
                // 这里显式用我们自己的日志事件，并挂在 request span 下，保证按 request_id 可检索到完整链路。
                .on_request(|req: &axum::http::Request<Body>, span: &Span| {
                    tracing::info!(parent: span, method = %req.method(), uri = %req.uri(), "request started");
                })
                .on_response(|res: &axum::http::Response<_>, latency: Duration, span: &Span| {
                    tracing::info!(
                        parent: span,
                        status = %res.status(),
                        latency_ms = latency.as_millis(),
                        "request finished"
                    );
                })
                .on_failure(|error: tower_http::classify::ServerErrorsFailureClass, latency: Duration, span: &Span| {
                    tracing::warn!(
                        parent: span,
                        error = %error,
                        latency_ms = latency.as_millis(),
                        "request failed"
                    );
                }),
        )
        // request_id：提升到 app 全局最外层，保证 TraceLayer / 任意 handler 日志都在 request_id span 内
        .layer(axum::middleware::from_fn(
            crate::middleware::request_id::request_id_middleware,
        ));

    // 启动服务器
    let addr = config.server_addr();
    tracing::info!("Server listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await?;
    axum::serve(listener, app.into_make_service()).await?;

    Ok(())
}
