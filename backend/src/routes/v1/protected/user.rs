use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 用户信息
        .route("/me", post(user::me))
        .route("/update_profile", post(user::update_profile))
        .route("/update_location", post(user::update_location))
        .route("/upgrade_account", post(user::upgrade_account))
        
        // 用户设置
        .route("/settings", post(user::settings))
        .route("/update_settings", post(user::update_settings))
        .route("/delete_account", post(user::delete_account))
        
        // 关系管理
        .route("/add_friend", post(user::add_friend))
        .route("/accept_friend", post(user::accept_friend))
        .route("/reject_friend", post(user::reject_friend))
        .route("/remove_friend", post(user::remove_friend))
        .route("/follow", post(user::follow))
        .route("/unfollow", post(user::unfollow))
        .route("/block", post(user::block))
        .route("/unblock", post(user::unblock))
        .route("/set_friend_remark", post(user::set_friend_remark))
        
        // 关系列表
        .route("/list_friends", post(user::list_friends))
        .route("/list_following", post(user::list_following))
        .route("/list_followers", post(user::list_followers))
        .route("/list_blocked", post(user::list_blocked))
}

