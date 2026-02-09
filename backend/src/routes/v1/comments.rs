//! 评论路由模块
//!
//! 统一的评论 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（评论列表、回复列表、点赞列表）
//! - token_required_routes: Token 必须的路由（创建、删除、点赞等）

use axum::{routing::post, Router};

use crate::{handlers::comments, state::AppState};

/// Token 可选的评论路由
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        .route("/list", post(comments::get_comments))
        .route("/list_replies", post(comments::get_replies))
        .route("/list_likes", post(comments::get_likes))
}

/// Token 必须的评论路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        .route("/create", post(comments::create))
        .route("/delete", post(comments::delete));

    // /me 子路由
    let me_routes = Router::new()
        .route("/like", post(comments::like))
        .route("/unlike", post(comments::unlike));

    main_routes.nest("/me", me_routes)
}
