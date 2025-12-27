use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
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
