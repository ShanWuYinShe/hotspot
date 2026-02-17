//! 用户路由模块
//!
//! 统一的用户 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（搜索、获取用户信息等）
//! - token_required_routes: Token 必须的路由（/me/* 路径等）
//!
//! ## 有意延期的端点（spec 已定义，实现延期）
//!
//! - `users/me/email/request_change` / `users/me/email/confirm_change`
//! - `users/me/phone/request_change` / `users/me/phone/confirm_change`

use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

/// Token 可选的用户路由
/// 带 Token 时返回 @auth-only 字段（如 relation）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 搜索 API（统一入口，内部根据参数分发）
        .route("/search", post(user::search))
        // 获取指定用户信息（包含在线状态，受隐私设置控制）
        .route("/open_the_door", post(user::open_the_door))
        // 获取指定用户列表（受隐私设置控制）
        .route("/list_friends", post(user::get_friends))
        .route("/list_following", post(user::get_following))
        .route("/list_followers", post(user::get_followers))
        .route("/list_groups", post(user::get_groups))
        .route("/list_moments", post(user::get_moments))
        .route("/list_liked_moments", post(user::get_liked_moments))
}

/// Token 必须的用户路由
pub fn token_required_routes() -> Router<AppState> {
    // /me 子路由 - 当前用户操作
    let me_routes = Router::new()
        // 当前用户资料
        .route("/profile/get", post(user::get_my_profile))
        .route("/profile/update", post(user::update_my_profile))
        .route("/location/update", post(user::update_location))
        .route("/upgrade", post(user::upgrade_account))
        // 设置
        .route("/settings/get", post(user::get_my_settings))
        .route("/settings/update", post(user::update_my_settings))
        .route("/delete", post(user::delete_account))
        // 关注系统（互相关注 = 好友）
        .route("/follow", post(user::follow))
        .route("/unfollow", post(user::unfollow))
        // 拉黑管理
        .route("/block", post(user::block))
        .route("/unblock", post(user::unblock))
        // 列表（list_friends/following/followers 已合并到 token_optional 的公开端点，user_id 可选）
        .route("/list_blocked", post(user::get_my_blocked))
        // 备注
        .route("/set_remark", post(user::set_remark));

    Router::new().nest("/me", me_routes)
}
