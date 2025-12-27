use axum::{routing::post, Router};

use crate::{handlers::moments, state::AppState};

pub fn routes() -> Router<AppState> {
    // 主路由 - 瞬间操作
    let main_routes = Router::new()
        .route("/create", post(moments::create))
        .route("/update", post(moments::update))
        .route("/delete", post(moments::delete))
        // 点赞
        .route("/like", post(moments::like))
        .route("/unlike", post(moments::unlike))
        // 收藏
        .route("/favorite", post(moments::favorite))
        .route("/unfavorite", post(moments::unfavorite));

    // /me 子路由 - 当前用户的瞬间相关操作
    let me_routes = Router::new()
        .route("/", post(moments::get_my_moments))
        .route("/favorites", post(moments::get_my_favorites))
        .route("/liked", post(moments::check_liked))
        .route("/favorited", post(moments::check_favorited));

    main_routes.nest("/me", me_routes)
}
