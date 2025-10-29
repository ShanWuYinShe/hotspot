use axum::{routing::post, Router};

use crate::{handlers::moment, state::AppState};

pub fn moment_routes() -> Router<AppState> {
    Router::new()
        // 公开端点
        .route("/api/v1/public/moment/search_by_location", post(moment::search_by_location))
        .route("/api/v1/public/moment/search_by_tags", post(moment::search_by_tags))
        
        // 受保护端点
        .route("/api/v1/protected/moment/create", post(moment::create))
        .route("/api/v1/protected/moment/update", post(moment::update))
        .route("/api/v1/protected/moment/delete", post(moment::delete))
        .route("/api/v1/protected/moment/my_moments", post(moment::my_moments))
        .route("/api/v1/protected/moment/like", post(moment::like))
        .route("/api/v1/protected/moment/unlike", post(moment::unlike))
        .route("/api/v1/protected/moment/collect", post(moment::collect))
        .route("/api/v1/protected/moment/uncollect", post(moment::uncollect))
        .route("/api/v1/protected/moment/collections", post(moment::collections))
}

