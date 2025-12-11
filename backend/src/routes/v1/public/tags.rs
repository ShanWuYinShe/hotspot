use axum::{routing::post, Router};

use crate::{handlers::tags, state::AppState};

/// Public 标签路由
///
/// 这些接口不需要认证，任何人都可以访问
/// 严格按照 api_spec.mdc 定义
pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/get_trending", post(tags::get_trending)) // ✅ 获取热门标签
        .route("/suggest", post(tags::suggest)) // ✅ 标签搜索建议
}


