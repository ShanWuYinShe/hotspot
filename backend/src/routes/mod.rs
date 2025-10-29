pub mod auth;
pub mod user;
pub mod group;
pub mod message;
pub mod conversation;
pub mod conversation_settings;
pub mod system;
pub mod moment;
pub mod comment;
pub mod media;
pub mod inbox;
pub mod report;
pub mod subscription;
pub mod leaderboard;
pub mod feed;

use axum::{middleware, Router};

use crate::{
    middleware::auth::auth_middleware,
    state::AppState,
    websocket::routes::websocket_routes,
};

pub fn create_routes(state: AppState) -> Router {
    // 所有路由合并（包含公开和受保护的端点）
    let all_routes = Router::new()
        .merge(auth::auth_routes())
        .merge(user::user_routes())
        .merge(group::group_routes())
        .merge(message::message_routes())
        .merge(conversation::conversation_routes())
        .merge(conversation_settings::conversation_settings_routes())
        .merge(system::system_routes())
        .merge(moment::moment_routes())
        .merge(comment::comment_routes())
        .merge(media::media_routes())
        .merge(inbox::inbox_routes())
        .merge(report::report_routes())
        .merge(subscription::subscription_routes())
        .merge(leaderboard::leaderboard_routes())
        .merge(feed::feed_routes());

    // WebSocket路由（单独处理，不需要auth中间件，在handler中验证token）
    let ws_routes = websocket_routes();

    // 对受保护的路由应用认证中间件
    // 注意：中间件会检查路径是否包含 /protected/，如果包含则验证JWT
    let routes_with_auth = all_routes
        .layer(middleware::from_fn_with_state(
            state.clone(),
            auth_middleware,
        ));

    Router::new()
        .merge(routes_with_auth)
        .merge(ws_routes)
        .with_state(state)
}

