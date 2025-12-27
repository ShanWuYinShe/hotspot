pub mod v1;

use axum::Router;

use crate::{state::AppState, websocket::routes::websocket_routes};

/// 创建 v1 版本 API 路由（不含 /api/v1 前缀）
/// 由调用方决定挂载路径，方便后续支持 v2、v3 等版本
pub fn v1_routes(state: AppState) -> Router<AppState> {
    v1::v1_routes(state)
}

/// 创建 WebSocket 路由（不含前缀）
/// WebSocket 不需要 auth 中间件，在 handler 中验证 token
pub fn ws_routes() -> Router<AppState> {
    websocket_routes()
}
