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
mod sql_macros;
mod state;
mod types;
mod utils;
mod websocket;

use axum::body::Body;
use axum::Router;
use chrono::SecondsFormat;
use chrono::Utc;
use std::fmt as StdFmt;
use std::sync::Arc;
use std::time::Duration;
use tower_http::trace::TraceLayer;

use crate::error::AppError;
use tracing::field::{Field, Visit};
use tracing::Span;
use tracing::{Event, Level, Subscriber};
use tracing_subscriber::fmt::{format, FmtContext, FormatEvent, FormatFields};
use tracing_subscriber::layer::{Context, Layer};
use tracing_subscriber::registry::LookupSpan;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::{
    config::Config,
    events::{event_bus::EventBus, event_handler::EventHandler},
    state::AppState,
};

#[derive(Clone)]
struct RequestIdFormatter {
    enable_ansi: bool,
}

impl RequestIdFormatter {
    fn new() -> Self {
        let enable_ansi = std::env::var("NO_COLOR").map_or(true, |v| v.is_empty());
        Self { enable_ansi }
    }
}

impl<S, N> FormatEvent<S, N> for RequestIdFormatter
where
    S: Subscriber + for<'span> LookupSpan<'span>,
    N: for<'writer> FormatFields<'writer> + 'static,
{
    fn format_event(
        &self,
        ctx: &FmtContext<'_, S, N>,
        mut writer: format::Writer<'_>,
        event: &Event<'_>,
    ) -> StdFmt::Result {
        let timestamp = Utc::now().to_rfc3339_opts(SecondsFormat::Millis, true);
        writer.write_fmt(format_args!("{} ", timestamp))?;

        let metadata = event.metadata();
        self.write_level(&mut writer, metadata.level())?;

        if self.enable_ansi {
            writer.write_fmt(format_args!("\x1b[2m{}:\x1b[0m ", metadata.target()))?;
        } else {
            writer.write_fmt(format_args!("{}: ", metadata.target()))?;
        }

        if let Some(span) = ctx.lookup_current() {
            if let Some(request_id) = span.extensions().get::<RequestId>() {
                writer.write_fmt(format_args!("[request_id={}] ", request_id.0))?;
            }
        }

        ctx.format_fields(writer.by_ref(), event)?;
        writer.write_char('\n')
    }
}

impl RequestIdFormatter {
    fn write_level(&self, writer: &mut format::Writer<'_>, level: &Level) -> StdFmt::Result {
        if self.enable_ansi {
            let (color, reset) = match *level {
                Level::ERROR => ("\x1b[31m", "\x1b[0m"),
                Level::WARN => ("\x1b[33m", "\x1b[0m"),
                Level::INFO => ("\x1b[32m", "\x1b[0m"),
                Level::DEBUG => ("\x1b[34m", "\x1b[0m"),
                Level::TRACE => ("\x1b[90m", "\x1b[0m"),
            };
            writer.write_fmt(format_args!("{}{:>5}{} ", color, level, reset))
        } else {
            writer.write_fmt(format_args!("{:>5} ", level))
        }
    }
}

struct RequestIdLayer;

impl<S> Layer<S> for RequestIdLayer
where
    S: Subscriber + for<'span> LookupSpan<'span>,
{
    fn on_new_span(
        &self,
        attrs: &tracing::span::Attributes<'_>,
        id: &tracing::span::Id,
        ctx: Context<'_, S>,
    ) {
        struct Visitor {
            request_id: Option<String>,
        }

        impl Visit for Visitor {
            fn record_str(&mut self, field: &Field, value: &str) {
                if field.name() == "request_id" {
                    self.request_id = Some(value.to_string());
                }
            }

            fn record_debug(&mut self, field: &Field, value: &dyn StdFmt::Debug) {
                if field.name() == "request_id" {
                    self.request_id = Some(format!("{:?}", value));
                }
            }
        }

        let mut visitor = Visitor { request_id: None };
        attrs.record(&mut visitor);

        if let Some(request_id) = visitor.request_id {
            if let Some(span) = ctx.span(id) {
                span.extensions_mut().insert(RequestId(request_id));
            }
        }
    }
}

#[derive(Clone, Debug)]
struct RequestId(String);

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化日志
    // 目标：日志“人类可读”，且每条请求链路日志都能看到 request_id，但不输出 span_list / 海量 DEBUG/TRACE 噪声
    let env_filter = tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| {
        // 默认：业务日志为主；屏蔽 tower_http/axum/tungstenite 的噪音
        // 仍允许用户用 RUST_LOG 覆盖
        "info,sqlx=warn,hotspot_backend=info,tower_http=warn,axum=warn,tokio_tungstenite=warn"
            .into()
    });

    let event_format = RequestIdFormatter::new();

    let fmt_layer = tracing_subscriber::fmt::layer()
        .with_target(true)
        .with_thread_ids(false)
        .with_thread_names(false)
        .event_format(event_format);

    tracing_subscriber::registry()
        .with(env_filter)
        .with(RequestIdLayer)
        .with(fmt_layer)
        .init();

    tracing::info!("Starting HotSpot Backend...");

    // 加载配置
    let config = Config::from_env()?;
    tracing::debug!("Configuration loaded");
    if let Some(node_id) = crate::utils::id_gen::init_node_id_with_redis(&config.redis_url)
        .await
        .map_err(|msg| std::io::Error::new(std::io::ErrorKind::Other, msg))?
    {
        tracing::info!(node_id, "ID 生成器已启用 Redis 租约分配");
    }
    error::warmup_error_catalog();

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
                let err = AppError::WebSocketHeartbeatTimeout;
                tracing::warn!(
                    code = err.error_code(),
                    "WebSocket连接超时，移除连接: {}",
                    connection_id
                );
                // spec: 心跳超时先发送 Error(code=20000) 再断开
                let error_msg = crate::websocket::types::ws_error_from_app_error(&err);
                ws_manager_clone
                    .send_to_connection(&connection_id, &error_msg)
                    .await;
                ws_manager_clone.remove_connection(&connection_id).await;
            }
        }
    });
    tracing::debug!("WebSocket timeout checker started");

    // 启动监控指标输出任务
    let state_clone = state.clone();
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(60));
        loop {
            interval.tick().await;

            // 输出缓存指标
            let hit_rate = state_clone.cache_manager.metrics().get_hit_rate();
            let hits = state_clone
                .cache_manager
                .metrics()
                .hits
                .load(std::sync::atomic::Ordering::Relaxed);
            let misses = state_clone
                .cache_manager
                .metrics()
                .misses
                .load(std::sync::atomic::Ordering::Relaxed);
            let errors = state_clone
                .cache_manager
                .metrics()
                .errors
                .load(std::sync::atomic::Ordering::Relaxed);

            tracing::info!(
                cache_hit_rate = %format!("{:.2}%", hit_rate * 100.0),
                cache_hits = hits,
                cache_misses = misses,
                cache_errors = errors,
                "缓存性能指标"
            );

            // 输出数据库连接池指标
            tracing::info!(
                pool_size = state_clone.db.size(),
                pool_idle = state_clone.db.num_idle(),
                "数据库连接池状态"
            );
        }
    });
    tracing::debug!("Monitoring metrics task started");

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

                    // span 挂载 request_id + 可延迟填充的 user_id/session_id，便于跨层日志关联
                    tracing::info_span!(
                        "http_request",
                        request_id = %request_id,
                        user_id = tracing::field::Empty,
                        session_id = tracing::field::Empty
                    )
                })
                // 不使用 DefaultOnRequest/Response：它们的 target 在 tower_http::* 下，容易被 EnvFilter 过滤掉
                // 这里显式用我们自己的日志事件，并挂在 request span 下，保证按 request_id 可检索到完整链路。
                .on_request(|req: &axum::http::Request<Body>, span: &Span| {
                    let user_agent = req
                        .headers()
                        .get(axum::http::header::USER_AGENT)
                        .and_then(|v| v.to_str().ok())
                        .unwrap_or("-");
                    let content_length = req
                        .headers()
                        .get(axum::http::header::CONTENT_LENGTH)
                        .and_then(|v| v.to_str().ok())
                        .and_then(|v| v.parse::<u64>().ok());
                    let client_ip = req
                        .headers()
                        .get("x-forwarded-for")
                        .or_else(|| req.headers().get("x-real-ip"))
                        .and_then(|v| v.to_str().ok())
                        .unwrap_or("-");

                    tracing::info!(
                        parent: span,
                        method = %req.method(),
                        uri = %req.uri(),
                        user_agent = %user_agent,
                        client_ip = %client_ip,
                        content_length = ?content_length,
                        has_authorization = req.headers().contains_key(axum::http::header::AUTHORIZATION),
                        has_cookie = req.headers().contains_key(axum::http::header::COOKIE),
                        "request started"
                    );
                })
                .on_response(|res: &axum::http::Response<_>, latency: Duration, span: &Span| {
                    let response_content_type = res
                        .headers()
                        .get(axum::http::header::CONTENT_TYPE)
                        .and_then(|v| v.to_str().ok());
                    tracing::info!(
                        parent: span,
                        status = %res.status(),
                        content_type = ?response_content_type,
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
        .layer(axum::middleware::from_fn(
            crate::middleware::response_logger::response_logger,
        ))
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
