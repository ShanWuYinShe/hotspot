//! API v1 路由模块
//!
//! API 路径与 Token 要求规范：
//! - /public/* 路径不需要认证（无 auth_middleware）
//! - /protected/* 路径需要认证（应用 auth_middleware）

pub mod protected;
pub mod public;

use axum::{middleware, Router};
use tower::ServiceBuilder;

use crate::{
    middleware::{
        admin_auth::admin_auth_middleware, auth::auth_middleware,
        idempotency::idempotency_middleware, rate_limit::rate_limit_middleware,
    },
    state::AppState,
};

/// 创建 API v1 的所有路由（public + protected）
/// 返回 Router<AppState>，需要在外层调用 with_state 绑定状态
pub fn v1_routes(state: AppState) -> Router<AppState> {
    // ==================== 公开路由 ====================
    // 无需认证，但有速率限制
    let public_routes = Router::new()
        .nest("/public/auth", public::auth::routes())
        .nest("/public/user", public::user::routes())
        .nest("/public/group_chats", public::group_chats::routes())
        .nest("/public/moments", public::moments::routes())
        .nest("/public/comments", public::comments::routes())
        .nest("/public/map", public::map::routes())
        .nest("/public/system", public::system::routes())
        .nest("/public/tags", public::tags::routes())
        .with_state(state.clone())
        .layer(middleware::from_fn_with_state(
            state.clone(),
            rate_limit_middleware,
        ));

    // ==================== 受保护路由 ====================
    // 需要认证，且有幂等性、速率限制中间件
    let protected_routes = Router::new()
        .nest("/protected/auth", protected::auth::routes())
        .nest("/protected/user", protected::user::routes())
        .nest("/protected/group_chats", protected::group_chats::routes())
        .nest(
            "/protected/private_chats",
            protected::private_chats::routes(),
        )
        .nest(
            "/protected/private_messages",
            protected::private_messages::routes(),
        )
        .nest(
            "/protected/group_messages",
            protected::group_messages::routes(),
        )
        .nest("/protected/moments", protected::moments::routes())
        .nest("/protected/comments", protected::comments::routes())
        .nest("/protected/reports", protected::reports::routes())
        .nest(
            "/protected/device_tokens",
            protected::device_tokens::routes(),
        )
        .nest(
            "/protected/subscriptions",
            protected::subscriptions::routes(),
        )
        .nest("/protected/system", protected::system::routes())
        .nest("/protected/feed", protected::feed::routes())
        .nest("/protected/media", protected::media::routes())
        .nest("/protected/messages", protected::messages::routes())
        .with_state(state.clone())
        .layer(
            ServiceBuilder::new()
                // 中间件执行顺序（从下到上）：
                // 1. auth_middleware（认证）
                // 2. idempotency_middleware（幂等性）
                // 3. rate_limit_middleware（速率限制）
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    rate_limit_middleware,
                ))
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    idempotency_middleware,
                ))
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    auth_middleware,
                )),
        );

    // ==================== 管理员路由 ====================
    // 需要管理员权限，应用认证 + 管理员验证 + 其他中间件
    let admin_routes = Router::new()
        .nest("/admin", protected::admin::admin_routes())
        .with_state(state.clone())
        .layer(
            ServiceBuilder::new()
                // 中间件执行顺序（从下到上）：
                // 1. auth_middleware（认证）
                // 2. admin_auth_middleware（管理员验证）
                // 3. idempotency_middleware（幂等性）
                // 4. rate_limit_middleware（速率限制）
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    rate_limit_middleware,
                ))
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    idempotency_middleware,
                ))
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    admin_auth_middleware,
                ))
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    auth_middleware,
                )),
        );

    // ==================== 合并 v1 路由 ====================
    // request_id 中间件已提升到 app 全局层，避免 TraceLayer 日志缺失 request_id
    public_routes.merge(protected_routes).merge(admin_routes)
}
