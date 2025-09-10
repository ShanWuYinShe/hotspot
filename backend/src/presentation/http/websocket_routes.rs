/// WebSocket路由定义
/// 
/// 处理WebSocket连接升级和路由
use axum::{
    routing::{get, post},
    Router,
};
use std::sync::Arc;

use crate::ports::UserRepository;
use crate::ws::handler::{ws_handler, WebSocketState};
use crate::ws::manager::WebSocketManager;
use crate::ws::connection_manager::ConnectionManager;
use crate::ws::room_manager::RoomManager;
use crate::ws::message_router::MessageRouter;

/// 创建WebSocket路由
pub fn create_websocket_routes<R>(_user_repo: Arc<R>, db_pool: sqlx::PgPool) -> Router 
where
    R: UserRepository + 'static + ?Sized,
{
    let jwt_secret = std::env::var("JWT_SECRET")
        .unwrap_or_else(|_| "default_jwt_secret_for_development".to_string());
    
    // 创建WebSocket组件
    let connection_manager = Arc::new(ConnectionManager::new());
    let room_manager = Arc::new(RoomManager::new());
    let message_router = Arc::new(MessageRouter::new(
        connection_manager.clone(),
        room_manager.clone(),
    ));
    
    // 使用传入的数据库连接池
    
    // 创建WebSocket管理器
    let ws_manager = Arc::new(WebSocketManager::new(
        message_router,
        connection_manager,
        room_manager,
        db_pool,
        None, // metrics
    ));
    
    // 创建WebSocket状态
    let ws_state = Arc::new(WebSocketState {
        ws_manager,
        jwt_secret,
    });
    
    Router::new()
        .route("/ws", get(ws_handler))
        .route("/ws/", get(ws_handler))
        .with_state(ws_state)
}

// WebSocket路由已完成，使用完整的WebSocket管理器实现

/// WebSocket连接信息响应
#[derive(serde::Serialize)]
pub struct WebSocketInfo {
    pub endpoint: String,
    pub protocol: String,
    pub authentication: String,
    pub example_url: String,
}

/// 获取WebSocket连接信息
pub async fn websocket_info() -> axum::Json<WebSocketInfo> {
    axum::Json(WebSocketInfo {
        endpoint: "/ws".to_string(),
        protocol: "WebSocket".to_string(),
        authentication: "JWT token via query parameter".to_string(),
        example_url: "/ws?token=your_jwt_token".to_string(),
    })
}

/// 创建WebSocket信息路由
pub fn create_websocket_info_routes() -> Router {
    Router::new()
        .route("/ws/info", post(websocket_info))
}
