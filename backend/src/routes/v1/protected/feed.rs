use axum::{routing::post, Router};

use crate::{handlers::feed, state::AppState};

/// Protected Feed 路由
/// 
/// 信息流 API - 为已登录用户提供个性化内容推荐
/// 路径: /api/v1/protected/feed/*
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/moments", post(feed::get_moments))     // 获取瞬间信息流
        .route("/groups", post(feed::get_groups))       // 获取群组推荐
        .route("/users", post(feed::get_users))         // 获取用户推荐
        .route("/trending", post(feed::get_trending))   // 获取热门内容
        .route("/feedback", post(feed::feedback))       // 反馈信息流内容
}
