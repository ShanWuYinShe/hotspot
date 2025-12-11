use axum::{routing::post, Router};

use crate::{handlers::group_chats, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 搜索 API
        .route("/search_by_keyword", post(group_chats::search_by_keyword))
        .route("/search_by_tag", post(group_chats::search_by_tag))
        .route("/search_by_location", post(group_chats::search_by_location))
        // 获取指定群组信息
        .route("/open_the_door", post(group_chats::open_the_door))
        .route("/get_settings", post(group_chats::get_settings))
        // 成员列表
        .route("/members/get_list", post(group_chats::get_members))
}
