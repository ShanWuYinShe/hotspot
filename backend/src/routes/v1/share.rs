//! 分享 API 路由
//!
//! 对应 API 规范: api_spec_share.md
//!
//! Token 要求：
//! - token_optional_routes: share/get（认证可选）
//! - token_required_routes: share/create、share/revoke、share/me/list（认证必须）

use crate::handlers::share;
use crate::state::AppState;
use axum::{routing::post, Router};

/// Token 可选的分享路由（spec: share/get 认证可选）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new().route("/get", post(share::get_share))
}

/// Token 必须的分享路由
pub fn token_required_routes() -> Router<AppState> {
    let main_routes = Router::new()
        .route("/create", post(share::create_share))
        .route("/revoke", post(share::revoke_share));

    let me_routes = Router::new().route("/list", post(share::list_my_shares));

    main_routes.nest("/me", me_routes)
}
