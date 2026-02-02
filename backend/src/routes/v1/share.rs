//! 分享 API 路由
//!
//! 对应 API 规范: api_spec_share.mdc

use crate::handlers::share;
use crate::state::AppState;
use axum::{routing::post, Router};

/// 分享模块路由
pub fn share_routes() -> Router<AppState> {
    Router::new()
        .route("/create", post(share::create_share))
        .route("/get", post(share::get_share))
        .route("/apply", post(share::apply_share))
}
