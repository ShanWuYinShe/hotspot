//! API v1 路由模块
//!
//! API 路径与 Token 要求规范（与 spec 对齐）：
//! - 所有 API 统一使用 `/api/v1/` 前缀
//! - Token 可选：带 token 则按登录态返回更多字段/更宽权限；不带则按匿名语义返回
//! - Token 必须：缺失或无效直接拒绝
//! - 管理员：需要 Token + 管理员权限

// 统一的路由模块（按功能域组织）
pub mod admin;
pub mod announcements;
pub mod auth;
pub mod comments;
pub mod device_tokens;
pub mod feed;
pub mod global_settings;
pub mod group_chats;
pub mod group_messages;
pub mod map;
pub mod media;
pub mod messages;
pub mod moments;
pub mod notifications;
pub mod private_chats;
pub mod private_messages;
pub mod reports;
pub mod search;
pub mod subscriptions;
pub mod system;
pub mod tags;
pub mod user;

use axum::{middleware, Router};

use crate::{
    middleware::{
        admin_auth::admin_auth_middleware, auth::auth_middleware, auth::optional_auth_middleware,
        idempotency::idempotency_middleware, rate_limit::rate_limit_middleware,
        resource_guard::resource_guard_middleware,
    },
    state::AppState,
};

/// 创建 API v1 的所有路由（Token 可选 + Token 必须 + admin）
/// 返回 Router<AppState>，需要在外层调用 with_state 绑定状态
pub fn v1_routes(state: AppState) -> Router<AppState> {
    // axum 0.8 中，from_fn_with_state 在部分场景下会触发复杂的 trait 约束问题。
    // 这里统一用 from_fn + 闭包捕获 state，保证中间件可组合、类型稳定。
    let state_for_optional_auth = state.clone();
    let optional_auth_layer = middleware::from_fn(move |req, next| {
        let st = state_for_optional_auth.clone();
        async move { optional_auth_middleware(axum::extract::State(st), req, next).await }
    });

    let state_for_auth = state.clone();
    let auth_layer = middleware::from_fn(move |req, next| {
        let st = state_for_auth.clone();
        async move { auth_middleware(axum::extract::State(st), req, next).await }
    });

    let state_for_admin_auth = state.clone();
    let admin_auth_layer = middleware::from_fn(move |req, next| {
        let st = state_for_admin_auth.clone();
        async move { admin_auth_middleware(axum::extract::State(st), req, next).await }
    });

    let state_for_rate = state.clone();
    let rate_limit_layer = middleware::from_fn(move |req, next| {
        let st = state_for_rate.clone();
        async move { rate_limit_middleware(axum::extract::State(st), req, next).await }
    });

    let state_for_resource_guard = state.clone();
    let resource_guard_layer = middleware::from_fn(move |req, next| {
        let st = state_for_resource_guard.clone();
        async move { resource_guard_middleware(axum::extract::State(st), req, next).await }
    });

    let state_for_idempotency = state.clone();
    let idempotency_layer = middleware::from_fn(move |req, next| {
        let st = state_for_idempotency.clone();
        async move { idempotency_middleware(axum::extract::State(st), req, next).await }
    });

    // ==================== Token 可选路由 ====================
    // 可带 token；带 token 会注入 user_id（用于"登录态差异化返回"/限流维度）
    let token_optional_routes = Router::new()
        .nest("/auth", auth::token_optional_routes())
        .nest("/user", user::token_optional_routes())
        .nest("/group_chats", group_chats::token_optional_routes())
        .nest("/moments", moments::token_optional_routes())
        .nest("/comments", comments::token_optional_routes())
        .nest("/map", map::token_optional_routes())
        .nest("/system", system::token_optional_routes())
        .nest("/global_settings", global_settings::token_optional_routes())
        .nest("/feed", feed::token_optional_routes())
        .nest("/search", search::routes())
        .nest("/tags", tags::token_optional_routes())
        .nest("/announcements", announcements::token_optional_routes())
        .with_state(state.clone())
        // 约定：最后添加的 layer 最先执行（outermost）
        // 期望执行顺序：optional_auth -> rate_limit -> resource_guard
        .layer(resource_guard_layer.clone())
        .layer(rate_limit_layer.clone())
        .layer(optional_auth_layer.clone());

    // ==================== Token 必须路由 ====================
    // 需要认证，且有幂等性、速率限制中间件
    let auth_required_routes = Router::new()
        .nest("/auth", auth::token_required_routes())
        .nest("/user", user::token_required_routes())
        .nest("/group_chats", group_chats::token_required_routes())
        .nest("/private_chats", private_chats::token_required_routes())
        .nest(
            "/private_messages",
            private_messages::token_required_routes(),
        )
        .nest("/group_messages", group_messages::token_required_routes())
        .nest("/moments", moments::token_required_routes())
        .nest("/comments", comments::token_required_routes())
        .nest("/reports", reports::token_required_routes())
        .nest("/device_tokens", device_tokens::token_required_routes())
        .nest("/subscriptions", subscriptions::token_required_routes())
        .nest("/system", system::token_required_routes())
        .nest("/feed", feed::token_required_routes())
        .nest("/media", media::token_required_routes())
        .nest("/messages", messages::token_required_routes())
        .nest("/notifications", notifications::token_required_routes())
        .with_state(state.clone())
        // 期望执行顺序：auth -> rate_limit -> resource_guard -> idempotency
        .layer(idempotency_layer.clone())
        .layer(resource_guard_layer.clone())
        .layer(rate_limit_layer.clone())
        .layer(auth_layer.clone());

    // ==================== 管理员路由 ====================
    // 需要管理员权限，应用认证 + 管理员验证 + 其他中间件
    let admin_routes = Router::new()
        .nest("/admin", admin::admin_routes())
        .with_state(state.clone())
        // 期望执行顺序：auth -> admin_auth -> rate_limit -> resource_guard -> idempotency
        .layer(idempotency_layer)
        .layer(resource_guard_layer)
        .layer(rate_limit_layer)
        .layer(admin_auth_layer)
        .layer(auth_layer);

    // ==================== 合并 v1 路由 ====================
    // request_id 中间件已提升到 app 全局层，避免 TraceLayer 日志缺失 request_id
    token_optional_routes
        .merge(auth_required_routes)
        .merge(admin_routes)
}
