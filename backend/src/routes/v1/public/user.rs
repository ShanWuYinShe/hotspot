use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
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
