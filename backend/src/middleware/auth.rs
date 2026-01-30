use axum::{
    extract::{FromRequestParts, Request, State},
    http::request::Parts,
    middleware::Next,
    response::Response,
};

use crate::{error::AppError, state::AppState, utils::jwt::JwtUtil};

/// 从请求头提取 Bearer token（不跨 await，避免 Future 非 Send）
fn extract_bearer_token(req: &Request) -> Result<Option<String>, AppError> {
    let auth_header = match req
        .headers()
        .get("Authorization")
        .and_then(|h| h.to_str().ok())
    {
        Some(v) => v,
        None => return Ok(None),
    };

    let token = auth_header
        .strip_prefix("Bearer ")
        .ok_or_else(|| AppError::AccessTokenMissing)?;

    Ok(Some(token.to_string()))
}

/// 验证 token 并返回 (user_id, session_id)
async fn authenticate_token(state: &AppState, token: &str) -> Result<(i64, uuid::Uuid), AppError> {
    // 验证 JWT Token
    let claims = JwtUtil::verify_token(token, &state.config.jwt_secret)?;
    let user_id = claims
        .sub
        .parse::<i64>()
        .map_err(|_| AppError::Unauthorized("无效的用户ID".to_string()))?;

    // 解析 session_id（sid）
    let session_id = uuid::Uuid::parse_str(&claims.sid)
        .map_err(|_| AppError::Unauthorized("无效的Session".to_string()))?;

    // 检查 Token 是否在 Session 白名单中 (Active Session Validation)
    let whitelist_key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, claims.sid);
    let is_valid: bool = redis::cmd("EXISTS")
        .arg(&whitelist_key)
        .query_async(&mut state.redis.clone())
        .await
        .unwrap_or(false); // Redis error defaults to false (fail closed for security)

    if !is_valid {
        return Err(AppError::Unauthorized("Session无效或已过期".to_string()));
    }

    Ok((user_id, session_id))
}

/// 从请求扩展中提取用户ID
///
/// 在 auth_middleware 之后使用，用于获取已认证用户的ID
pub fn extract_user_id(req: &Request) -> Result<i64, AppError> {
    req.extensions()
        .get::<i64>()
        .copied()
        .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))
}

/// JWT 认证中间件
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let token = extract_bearer_token(&req)?.ok_or_else(|| AppError::AccessTokenMissing)?;
    let (user_id, session_id) = authenticate_token(&state, &token).await?;

    // 将 user_id / session_id 注入到请求扩展中
    req.extensions_mut().insert(user_id);
    req.extensions_mut().insert(Some(session_id));

    Ok(next.run(req).await)
}

/// JWT 可选认证中间件（Token 可选）
///
/// - 如果请求未带 Authorization，则直接透传（不注入 user_id）
/// - 如果请求带了 Authorization，但 token 无效，则返回对应错误（不会“忽略坏 token”）
pub async fn optional_auth_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    if let Some(token) = extract_bearer_token(&req)? {
        let (user_id, session_id) = authenticate_token(&state, &token).await?;
        req.extensions_mut().insert(user_id);
        req.extensions_mut().insert(Some(session_id));
    }
    Ok(next.run(req).await)
}

/// Bearer Token 提取器
///
/// 从Authorization header中提取Bearer token
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

/// 已认证用户提取器
///
/// 从请求扩展中提取已认证的用户信息（由 auth_middleware 注入）
#[derive(Debug)]
pub struct AuthenticatedUser {
    pub user_id: i64,
    /// 当前请求的会话ID
    ///
    /// 用途：
    /// - 会话管理：用于标记当前会话（如 list_sessions 中的 is_current）
    /// - 会话撤销：通过删除 Redis 中的 `session:{sid}` key 来撤销会话
    /// - 审计日志：记录操作来源的会话ID
    ///
    /// 注意：虽然大多数 handler 使用 `Extension<i64>` 和 `Extension<Option<uuid::Uuid>>` 分别提取，
    /// 但使用 `AuthenticatedUser` 提取器可以同时获取 user_id 和 session_id，代码更清晰。
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

/// 可选登录态提取器
///
/// 用于 Token 可选的接口：如果请求上下文里已经注入了 user_id，则返回 Some；
/// 否则返回 None（不会拒绝请求）。
#[derive(Debug)]
pub struct OptionalAuthenticatedUser {
    pub user_id: Option<i64>,
}

impl<S> FromRequestParts<S> for OptionalAuthenticatedUser
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let user_id = parts.extensions.get::<i64>().copied();
        Ok(OptionalAuthenticatedUser { user_id })
    }
}
