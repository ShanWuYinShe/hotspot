use axum::{routing::post, Router};

use crate::{handlers::comment, state::AppState};

pub fn comment_routes() -> Router<AppState> {
    Router::new()
        // 公开端点
        .route("/api/v1/public/comment/list", post(comment::list))
        .route("/api/v1/public/comment/replies", post(comment::replies))
        
        // 受保护端点
        .route("/api/v1/protected/comment/add", post(comment::add))
        .route("/api/v1/protected/comment/delete", post(comment::delete))
        .route("/api/v1/protected/comment/like", post(comment::like))
        .route("/api/v1/protected/comment/unlike", post(comment::unlike))
}

