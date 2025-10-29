use axum::{
    routing::post,
    Router,
};

use crate::{handlers::user, state::AppState};

pub fn user_routes() -> Router<AppState> {
    Router::new()
        // 公开端点
        .route("/api/v1/public/user/search_by_name", post(user::search_by_name))
        .route("/api/v1/public/user/search_by_id", post(user::search_by_id))
        .route("/api/v1/public/user/search_by_location", post(user::search_by_location))
        
        // 受保护端点
        .route("/api/v1/protected/user/me", post(user::me))
        .route("/api/v1/protected/user/update_location", post(user::update_location))
        .route("/api/v1/protected/user/update_profile", post(user::update_profile))
        .route("/api/v1/protected/user/add_friend", post(user::add_friend))
        .route("/api/v1/protected/user/accept_friend", post(user::accept_friend))
        .route("/api/v1/protected/user/reject_friend", post(user::reject_friend))
        .route("/api/v1/protected/user/remove_friend", post(user::remove_friend))
        .route("/api/v1/protected/user/follow", post(user::follow))
        .route("/api/v1/protected/user/unfollow", post(user::unfollow))
        .route("/api/v1/protected/user/block", post(user::block))
        .route("/api/v1/protected/user/unblock", post(user::unblock))
        .route("/api/v1/protected/user/set_friend_remark", post(user::set_friend_remark))
        // TODO: 添加列表查询路由
        // - list_friends
        // - list_following
        // - list_followers
        // - list_blocked
        // - upgrade_account
        // - settings
        // - update_settings
        // - delete_account
}
