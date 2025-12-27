use axum::{routing::post, Router};

use crate::{handlers::moments, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 搜索 API
        .route("/search_by_keyword", post(moments::search_by_keyword))
        .route("/search_by_tag", post(moments::search_by_tag))
        .route("/search_by_location", post(moments::search_by_location))
        // 获取指定瞬间信息
        .route("/open_the_door", post(moments::open_the_door))
        // 列表 API
        .route("/comments", post(moments::get_comments))
        .route("/likes", post(moments::get_likes))
}
