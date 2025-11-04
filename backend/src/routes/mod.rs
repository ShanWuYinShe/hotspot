pub mod v1;
pub mod media;  // 保留 media 路由（用于文件上传）

use axum::{middleware, Router};

use crate::{
    middleware::auth::auth_middleware,
    state::AppState,
    websocket::routes::websocket_routes,
};

pub fn create_routes(state: AppState) -> Router {
    // 使用 v1 API 路由
    let api_v1_routes = Router::new()
        .nest("/api/v1", v1::routes())
        .merge(media::media_routes());  // media 路由保持独立

    // WebSocket路由（单独处理，不需要auth中间件，在handler中验证token）
    let ws_routes = websocket_routes();

    // 对受保护的路由应用认证中间件
    // 注意：中间件会检查路径是否包含 /protected/，如果包含则验证JWT
    let routes_with_auth = api_v1_routes
        .layer(middleware::from_fn_with_state(
            state.clone(),
            auth_middleware,
        ));

    Router::new()
        .merge(routes_with_auth)
        .merge(ws_routes)
        .with_state(state)
}

