//! 举报路由模块
//!
//! 统一的举报 API 路由
//! 所有举报接口都是 Token 必须的

use axum::{routing::post, Router};

use crate::{handlers::reports, state::AppState};

/// Token 必须的举报路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由 - 创建举报
    let main_routes = Router::new().route("/create", post(reports::create));

    // /me 子路由 - 当前用户举报记录
    let me_routes = Router::new().route("/", post(reports::me));

    main_routes.nest("/me", me_routes)
}
