//! 生活瞬间路由模块
//!
//! 统一的瞬间 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（搜索、详情、点赞列表等）
//! - token_required_routes: Token 必须的路由（创建、点赞、收藏等）

use axum::{routing::post, Router};

use crate::{handlers::moments, state::AppState};

/// Token 可选的瞬间路由
/// 带 Token 时返回 @auth-only 字段（如 my_state）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 搜索 API（统一入口）
        .route("/search", post(moments::search))
        // 获取指定瞬间信息
        .route("/open_the_door", post(moments::open_the_door))
        // 列表 API
        .route("/list_likes", post(moments::get_likes))
}

/// Token 必须的瞬间路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由 - 瞬间操作
    let main_routes = Router::new()
        .route("/create", post(moments::create))
        .route("/update", post(moments::update))
        .route("/delete", post(moments::delete));

    // /me 子路由 - 当前用户的瞬间相关操作
    let me_routes = Router::new()
        .route("/list", post(moments::get_my_moments))
        .route("/list_favorites", post(moments::get_my_favorites))
        .route("/like", post(moments::like))
        .route("/unlike", post(moments::unlike))
        .route("/favorite", post(moments::favorite))
        .route("/unfavorite", post(moments::unfavorite));

    main_routes.nest("/me", me_routes)
}
