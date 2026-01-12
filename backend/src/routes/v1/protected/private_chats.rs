use axum::{routing::post, Router};

use crate::{handlers::private_chats, state::AppState};

/// Protected 私聊管理路由
pub fn routes() -> Router<AppState> {
    // 主路由 - 私聊操作
    let main_routes = Router::new()
        .route("/open_the_door", post(private_chats::open_the_door))
        .route("/update_settings", post(private_chats::update_settings))
        .route("/delete", post(private_chats::delete));

    // /me 子路由 - 当前用户私聊列表
    let me_routes = Router::new().route("/", post(private_chats::me));

    main_routes.nest("/me", me_routes)
}
