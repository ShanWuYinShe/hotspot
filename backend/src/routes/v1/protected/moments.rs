use axum::{routing::post, Router};

use crate::{handlers::moments, state::AppState};

pub fn routes() -> Router<AppState> {
    // 主路由 - 瞬间操作
    let main_routes = Router::new()
        .route("/create", post(moments::create))
        .route("/update", post(moments::update))
        .route("/delete", post(moments::delete))
        // 点赞
        .route("/like_moment", post(moments::like_moment))
        .route("/unlike_moment", post(moments::unlike_moment))
        // 收藏
        .route("/favorite_moment", post(moments::favorite_moment))
        .route("/unfavorite_moment", post(moments::unfavorite_moment));

    // /me 子路由 - 当前用户的瞬间相关操作
    let me_routes = Router::new()
        .route("/get_list", post(moments::get_my_list))
        .route("/get_favorites_list", post(moments::get_favorites_list))
        .route("/check_liked", post(moments::check_liked))
        .route("/check_favorited", post(moments::check_favorited));

    main_routes.nest("/me", me_routes)
}
