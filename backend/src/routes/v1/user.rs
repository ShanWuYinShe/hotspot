//! 用户路由模块
//!
//! 统一的用户 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（搜索、获取用户信息等）
//! - token_required_routes: Token 必须的路由（关系查询、/me/* 路径等）

use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

/// Token 可选的用户路由
/// 带 Token 时返回 @auth-only 字段（如 relation）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 搜索 API
        .route("/search_by_keyword", post(user::search_by_keyword))
        .route("/search_by_tag", post(user::search_by_tag))
        .route("/search_by_location", post(user::search_by_location))
        // 获取指定用户信息（包含在线状态，受隐私设置控制）
        .route("/open_the_door", post(user::open_the_door))
        .route("/stats", post(user::get_stats))
        // 获取指定用户列表（受隐私设置控制）
        .route("/friends", post(user::get_friends))
        .route("/following", post(user::get_following))
        .route("/followers", post(user::get_followers))
        .route("/groups", post(user::get_groups))
        .route("/liked_moments", post(user::get_liked_moments))
        .route("/moments", post(user::get_moments))
}

/// Token 必须的用户路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        // 获取指定用户信息（需要登录）
        .route("/relationship", post(user::get_relationship));

    // /me 子路由 - 当前用户操作
    let me_routes = Router::new()
        // 当前用户资料
        .route("/profile", post(user::get_my_profile))
        .route("/profile/update", post(user::update_my_profile))
        .route("/location", post(user::update_location))
        .route("/upgrade", post(user::upgrade_account))
        .route("/stats", post(user::get_my_stats))
        // 设置
        .route("/settings", post(user::get_my_settings))
        .route("/settings/update", post(user::update_my_settings))
        .route("/delete", post(user::delete_account))
        // 关注系统（互相关注 = 好友）
        .route("/follow", post(user::follow))
        .route("/unfollow", post(user::unfollow))
        .route("/remark", post(user::set_remark))
        // 关注请求（当 allow_follow 设置为需要审批时）
        .route("/follow_requests", post(user::get_follow_requests))
        .route("/follow_requests/handle", post(user::handle_follow_request))
        // 拉黑管理
        .route("/block", post(user::block))
        .route("/unblock", post(user::unblock))
        // 列表
        .route("/friends", post(user::get_my_friends))
        .route("/following", post(user::get_my_following))
        .route("/followers", post(user::get_my_followers))
        .route("/blocked", post(user::get_my_blocked));

    main_routes.nest("/me", me_routes)
}
