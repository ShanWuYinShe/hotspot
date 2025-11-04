use axum::{routing::post, Router};

use crate::{handlers::chat, state::AppState};

/// Protected 私聊路由
/// 
/// 提供私聊管理相关接口（使用统一的 chat_rooms 表）
pub fn routes() -> Router<AppState> {
    Router::new()
        // 私聊管理
        .route("/start", post(chat::start))
        .route("/list", post(chat::list))
        .route("/settings", post(chat::settings))
        .route("/update_settings", post(chat::update_settings))
        .route("/delete", post(chat::delete))
        .route("/upgrade_to_group", post(chat::upgrade_to_group))
}










