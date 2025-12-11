use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 搜索 API
        .route("/search_by_keyword", post(user::search_by_keyword))
        .route("/search_by_tag", post(user::search_by_tag))
        .route("/search_by_location", post(user::search_by_location))
        // 获取指定用户信息
        .route("/open_the_door", post(user::open_the_door))
        .route("/get_status", post(user::get_status))
        .route("/get_stats", post(user::get_user_stats))
        // 获取指定用户列表（受隐私设置控制）
        .route("/get_user_friends_list", post(user::get_user_friends_list))
        .route("/get_user_following_list", post(user::get_user_following_list))
        .route("/get_user_followers_list", post(user::get_user_followers_list))
        .route("/get_user_groups_list", post(user::get_user_groups_list))
        .route("/get_user_liked_moments_list", post(user::get_user_liked_moments_list))
}






