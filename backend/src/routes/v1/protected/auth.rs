use axum::{routing::post, Router};

use crate::{handlers::auth, state::AppState};

pub fn routes() -> Router<AppState> {
    // 主路由 - 基础认证操作
    let main_routes = Router::new()
        .route("/logout", post(auth::logout))
        .route("/check_status", post(auth::check_status));

    // /me 子路由 - 当前用户认证相关操作
    let me_routes = Router::new()
        .route("/change_password", post(auth::change_password))
        .route("/sessions", post(auth::list_sessions))
        .route("/sessions/revoke", post(auth::revoke_session));

    main_routes.nest("/me", me_routes)
}
