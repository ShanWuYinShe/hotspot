//! 认证中间件
//!
//! 提供 JWT 认证功能，支持必须认证和可选认证两种模式。
//!
//! ## 性能优化
//!
//! - **内存缓存**: 使用 moka 缓存已验证的 token，减少 Redis 查询（TTL: 5分钟）
//! - **懒验证**: 可选认证模式下，token 无效时降级为未登录而不是拒绝请求
//! - **并发安全**: 所有缓存操作都是并发安全的
//!
//! ## 使用方式
//!
//! ### 必须认证的路由
//!
//! ```rust
//! Router::new()
//!     .route("/api/v1/user/me/profile", post(handler))
//!     .layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
//! ```
//!
//! ### 可选认证的路由
//!
//! ```rust
//! Router::new()
//!     .route("/api/v1/user/search", post(handler))
//!     .layer(middleware::from_fn_with_state(state.clone(), optional_auth_middleware))
//! ```

use axum::{
    extract::{FromRequestParts, Request, State},
    http::request::Parts,
    middleware::Next,
    response::Response,
};
use std::time::Instant;

use crate::{
    error::AppError,
    state::AppState,
    utils::{jwt::JwtUtil, web_cookie::get_cookie_value},
};

/// 认证失败原因（用于日志和监控）
#[derive(Debug, Clone)]
enum AuthFailureReason {
    /// Token 缺失
    TokenMissing,
    /// Token 格式错误（不是 Bearer token）
    InvalidFormat,
    /// JWT 签名验证失败
    InvalidSignature,
    /// JWT 已过期
    Expired,
    /// 用户 ID 无效
    InvalidUserId,
    /// Session ID 无效
    InvalidSessionId,
    /// Session 不在 Redis 白名单中（已撤销或过期）
    SessionRevoked,
    /// Redis 查询失败
    RedisError,
}

impl AuthFailureReason {
    /// 转换为用户友好的错误消息
    fn to_user_message(&self) -> &'static str {
        match self {
            Self::TokenMissing => "未提供认证令牌",
            Self::InvalidFormat => "认证令牌格式错误",
            Self::InvalidSignature => "认证令牌无效",
            Self::Expired => "认证令牌已过期",
            Self::InvalidUserId => "无效的用户ID",
            Self::InvalidSessionId => "无效的会话",
            Self::SessionRevoked => "会话已失效或过期",
            Self::RedisError => "认证服务暂时不可用",
        }
    }

    /// 是否应该记录为错误日志
    fn should_log_error(&self) -> bool {
        matches!(self, Self::RedisError)
    }
}

/// 从请求头或 Cookie 提取 access token
///
/// 优先级：
/// 1. Authorization Header (Bearer token)
/// 2. Cookie (access_token)
///
/// 这是一个同步函数，避免跨 await 持有 Request 引用导致 Future 非 Send
fn extract_bearer_token(req: &Request) -> Result<Option<String>, AuthFailureReason> {
    // 优先从 Authorization Header 提取
    if let Some(auth_header) = req
        .headers()
        .get("Authorization")
        .and_then(|h| h.to_str().ok())
    {
        let token = auth_header
            .strip_prefix("Bearer ")
            .ok_or(AuthFailureReason::InvalidFormat)?;
        return Ok(Some(token.to_string()));
    }

    // 备选：从 Cookie 提取
    if let Some(cookie_token) = get_cookie_value(req.headers(), "access_token") {
        return Ok(Some(cookie_token));
    }

    Ok(None)
}

/// 验证 token 并返回 (user_id, session_id)
///
/// ## 性能优化
///
/// 1. 首先查询内存缓存（moka），命中则直接返回
/// 2. 缓存未命中才验证 JWT 并查询 Redis
/// 3. 验证成功后更新缓存
///
/// ## 错误处理
///
/// - JWT 验证失败：返回相应的 AuthFailureReason
/// - Redis 查询失败：返回 RedisError，但不阻塞请求（fail-closed for security）
async fn authenticate_token(
    state: &AppState,
    token: &str,
) -> Result<(i64, uuid::Uuid), AuthFailureReason> {
    // 1. 尝试从 L1 本地缓存获取
    // 性能优化：优先查本地缓存，避免 Redis RTT 和 CPU 签名计算
    if let Some((user_id, session_id)) = state.token_cache.get(token).await {
        // 检查是否在本地黑名单中 (处理分布式撤销)
        if state.revoked_sessions.contains_key(&session_id.to_string()) {
            tracing::debug!(
                user_id,
                session_id = %session_id,
                "Token found in local cache but session is in revoked blacklist"
            );
            return Err(AuthFailureReason::SessionRevoked);
        }

        return Ok((user_id, session_id));
    }

    let start = Instant::now();

    // 2. 验证 JWT
    let claims = JwtUtil::verify_token(token, &state.config.jwt_secret).map_err(|e| {
        tracing::debug!(error = %e, "JWT verification failed");
        match e {
            crate::error::AppError::Unauthorized(msg) if msg.contains("expired") => {
                AuthFailureReason::Expired
            }
            _ => AuthFailureReason::InvalidSignature,
        }
    })?;

    let user_id = claims.sub.parse::<i64>().map_err(|e| {
        tracing::warn!(
            error = %e,
            sub = %claims.sub,
            "Failed to parse user_id from JWT claims"
        );
        AuthFailureReason::InvalidUserId
    })?;

    let session_id = uuid::Uuid::parse_str(&claims.sid).map_err(|e| {
        tracing::warn!(
            error = %e,
            sid = %claims.sid,
            "Failed to parse session_id from JWT claims"
        );
        AuthFailureReason::InvalidSessionId
    })?;

    // 3. 检查 Session 白名单（Active Session Validation）
    let whitelist_key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, claims.sid);
    let is_valid: bool = redis::cmd("EXISTS")
        .arg(&whitelist_key)
        .query_async(&mut state.redis.clone())
        .await
        .unwrap_or_else(|e| {
            tracing::error!(
                error = %e,
                session_id = %session_id,
                "Redis query failed during session validation"
            );
            false // Fail closed for security
        });

    if !is_valid {
        tracing::debug!(
            user_id = user_id,
            session_id = %session_id,
            "Session not found in whitelist (revoked or expired)"
        );
        return Err(AuthFailureReason::SessionRevoked);
    }

    // 4. 写入 L1 本地缓存
    state
        .token_cache
        .insert(token.to_string(), (user_id, session_id))
        .await;

    let elapsed = start.elapsed();
    tracing::debug!(
        user_id = user_id,
        session_id = %session_id,
        elapsed_us = elapsed.as_micros(),
        "Authentication succeeded (cache miss)"
    );

    Ok((user_id, session_id))
}

/// JWT 必须认证中间件
///
/// - Token 缺失或无效：返回 401 错误
/// - Token 有效：注入 user_id 和 session_id 到请求扩展
///
/// ## 性能监控
///
/// - 记录认证耗时（包括缓存命中/未命中）
/// - 记录认证失败原因
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let token = extract_bearer_token(&req)
        .map_err(|reason| {
            if reason.should_log_error() {
                tracing::error!(reason = ?reason, "Authentication failed");
            } else {
                tracing::debug!(reason = ?reason, "Authentication failed");
            }
            AppError::Unauthorized(reason.to_user_message().to_string())
        })?
        .ok_or_else(|| {
            tracing::debug!(reason = ?AuthFailureReason::TokenMissing, "Authentication failed");
            AppError::AccessTokenMissing
        })?;

    let (user_id, session_id) = authenticate_token(&state, &token).await.map_err(|reason| {
        if reason.should_log_error() {
            tracing::error!(reason = ?reason, "Authentication failed");
        } else {
            tracing::debug!(reason = ?reason, "Authentication failed");
        }
        AppError::Unauthorized(reason.to_user_message().to_string())
    })?;

    // 将 user_id / session_id 注入到请求扩展中
    req.extensions_mut().insert(user_id);
    req.extensions_mut().insert(Some(session_id));

    Ok(next.run(req).await)
}

/// JWT 可选认证中间件（推荐的容错策略）
///
/// ## 行为说明
///
/// - **未提供 Token**: 请求继续，不注入 user_id（按未登录处理）
/// - **提供有效 Token**: 注入 user_id 和 session_id
/// - **提供无效 Token**: 记录警告日志，但**降级为未登录**继续处理（容错策略）
///
/// ## 设计理念
///
/// 1. **用户体验优先**: token 过期时允许用户继续浏览公开内容
/// 2. **安全监控**: 记录所有无效 token 尝试（用于检测攻击）
/// 3. **渐进式降级**: 不因认证问题阻塞整个请求
///
/// ## 性能监控
///
/// - 记录认证耗时和缓存命中率
/// - 记录无效 token 尝试（用于安全审计）
pub async fn optional_auth_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let token = match extract_bearer_token(&req) {
        Ok(Some(token)) => token,
        Ok(None) => {
            // 未提供 token，正常情况
            tracing::trace!("No token provided (optional auth)");
            return Ok(next.run(req).await);
        }
        Err(reason) => {
            // Token 格式错误，记录警告后降级为未登录
            tracing::warn!(
                reason = ?reason,
                "Invalid token format in optional auth, degrading to unauthenticated"
            );
            return Ok(next.run(req).await);
        }
    };

    // 尝试验证 token
    match authenticate_token(&state, &token).await {
        Ok((user_id, session_id)) => {
            // 验证成功，注入用户信息
            req.extensions_mut().insert(user_id);
            req.extensions_mut().insert(Some(session_id));
        }
        Err(reason) => {
            // 验证失败，记录详细日志但降级为未登录
            if reason.should_log_error() {
                tracing::error!(
                    reason = ?reason,
                    "Token validation failed in optional auth (serious issue)"
                );
            } else {
                tracing::warn!(
                    reason = ?reason,
                    "Token validation failed in optional auth, degrading to unauthenticated"
                );
            }
            // 不注入任何用户信息，继续处理请求
        }
    }

    Ok(next.run(req).await)
}

/// Bearer Token 提取器
///
/// 从 Authorization header 中提取 Bearer token
pub struct BearerToken(pub String);

impl<S> FromRequestParts<S> for BearerToken
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let auth_header = parts
            .headers
            .get("Authorization")
            .and_then(|h| h.to_str().ok())
            .ok_or_else(|| AppError::AccessTokenMissing)?;

        let token = auth_header
            .strip_prefix("Bearer ")
            .ok_or_else(|| AppError::AccessTokenMissing)?;

        Ok(BearerToken(token.to_string()))
    }
}

/// 已认证用户提取器（推荐使用）
///
/// 从请求扩展中提取已认证的用户信息（由 auth_middleware 注入）
///
/// ## 使用示例
///
/// ```rust
/// async fn handler(user: AuthenticatedUser) -> Result<Json<Response>, AppError> {
///     let user_id = user.user_id;
///     let session_id = user.session_id.unwrap();
///     // ...
/// }
/// ```
#[derive(Debug, Clone, Copy)]
pub struct AuthenticatedUser {
    pub user_id: i64,
    /// 当前请求的会话ID
    ///
    /// ## 用途
    ///
    /// - **会话管理**: 标记当前会话（如 list_sessions 中的 is_current）
    /// - **会话撤销**: 通过删除 Redis 中的 `session:{sid}` key 撤销会话
    /// - **审计日志**: 记录操作来源的会话ID
    ///
    /// ## 注意
    ///
    /// 使用 `AuthenticatedUser` 提取器可以同时获取 user_id 和 session_id，
    /// 比分别提取 `Extension<i64>` 和 `Extension<Option<uuid::Uuid>>` 更清晰。
    pub session_id: Option<uuid::Uuid>,
}

impl<S> FromRequestParts<S> for AuthenticatedUser
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let user_id = parts
            .extensions
            .get::<i64>()
            .copied()
            .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))?;

        let session_id = parts
            .extensions
            .get::<Option<uuid::Uuid>>()
            .copied()
            .flatten();

        Ok(AuthenticatedUser {
            user_id,
            session_id,
        })
    }
}

/// 可选登录态提取器（推荐使用）
///
/// 用于 Token 可选的接口：如果请求上下文里已经注入了 user_id，则返回 Some；
/// 否则返回 None（不会拒绝请求）。
///
/// ## 使用示例
///
/// ```rust
/// async fn search_handler(user: OptionalAuthenticatedUser) -> Result<Json<Response>, AppError> {
///     if let Some(user_id) = user.user_id {
///         // 返回带有个性化信息的结果
///     } else {
///         // 返回公开信息
///     }
/// }
/// ```
#[derive(Debug, Clone, Copy)]
pub struct OptionalAuthenticatedUser {
    pub user_id: Option<i64>,
    pub session_id: Option<uuid::Uuid>,
}

impl<S> FromRequestParts<S> for OptionalAuthenticatedUser
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let user_id = parts.extensions.get::<i64>().copied();
        let session_id = parts
            .extensions
            .get::<Option<uuid::Uuid>>()
            .copied()
            .flatten();

        Ok(OptionalAuthenticatedUser {
            user_id,
            session_id,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_auth_failure_reason_messages() {
        assert_eq!(
            AuthFailureReason::TokenMissing.to_user_message(),
            "未提供认证令牌"
        );
        assert_eq!(
            AuthFailureReason::Expired.to_user_message(),
            "认证令牌已过期"
        );
        assert_eq!(
            AuthFailureReason::SessionRevoked.to_user_message(),
            "会话已失效或过期"
        );
    }

    #[test]
    fn test_auth_failure_reason_log_level() {
        assert!(!AuthFailureReason::TokenMissing.should_log_error());
        assert!(!AuthFailureReason::Expired.should_log_error());
        assert!(AuthFailureReason::RedisError.should_log_error());
    }
}
