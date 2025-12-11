use axum::{routing::post, Router};

use crate::{handlers::reports, state::AppState};

/// Protected 举报路由
pub fn routes() -> Router<AppState> {
    // 主路由 - 创建举报
    let main_routes = Router::new()
        .route("/create", post(reports::create));

    // /me 子路由 - 当前用户举报记录
    let me_routes = Router::new()
        .route("/get_list", post(reports::get_my_reports));

    main_routes.nest("/me", me_routes)
}
