//! 搜索路由模块
//!
//! 统一的搜索 API 路由

use axum::{routing::post, Router};

use crate::{handlers::search, state::AppState};

/// 搜索路由 (Token 可选)
pub fn routes() -> Router<AppState> {
    Router::new().route("/suggest", post(search::suggest))
}
