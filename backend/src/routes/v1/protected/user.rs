use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
    // 主路由
    let main_routes = Router::new()
        // 获取指定用户信息（需要登录）
        .route("/get_relationship", post(user::get_relationship));

    // /me 子路由 - 当前用户操作
    let me_routes = Router::new()
        // 当前用户资料
        .route("/", post(user::me))
        .route("/update_profile", post(user::update_profile))
        .route("/update_location", post(user::update_location))
        .route("/upgrade_account", post(user::upgrade_account))
        .route("/get_stats", post(user::get_my_stats))
        
        // 设置
        .route("/get_settings", post(user::get_settings))
        .route("/update_settings", post(user::update_settings))
        .route("/delete_account", post(user::delete_account))
        
        // 关注系统（互相关注 = 好友）
        .route("/follow_user", post(user::follow_user))
        .route("/unfollow_user", post(user::unfollow_user))
        .route("/set_friend_remark", post(user::set_friend_remark))
        
        // 拉黑管理
        .route("/block_user", post(user::block_user))
        .route("/unblock_user", post(user::unblock_user))
        
        // 列表
        .route("/get_friends_list", post(user::get_friends_list))
        .route("/get_following_list", post(user::get_following_list))
        .route("/get_followers_list", post(user::get_followers_list))
        .route("/get_blocked_list", post(user::get_blocked_list));

    main_routes.nest("/me", me_routes)
}
