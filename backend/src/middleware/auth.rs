use axum::{
    extract::{FromRequestParts, Request, State},
    http::request::Parts,
    middleware::Next,
    response::Response,
};

use crate::{
    error::AppError,
    state::AppState,
    utils::jwt::JwtUtil,
};

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
    // 1. 提取 Authorization 头
    let auth_header = req
        .headers()
        .get("Authorization")
        .and_then(|h| h.to_str().ok())
        .ok_or_else(|| AppError::TokenMissing("缺少认证令牌".to_string()))?;

    // 2. 验证 Bearer 格式
    let token = auth_header
        .strip_prefix("Bearer ")
        .ok_or_else(|| AppError::TokenMissing("令牌格式错误".to_string()))?;

    // 3. 验证 JWT Token
    let claims = JwtUtil::verify_token(token, &state.config.jwt_secret)?;
    let user_id = claims
        .sub
        .parse::<i64>()
        .map_err(|_| AppError::Unauthorized("无效的用户ID".to_string()))?;

    // 4. 解析 session_id（sid）
    let session_id = uuid::Uuid::parse_str(&claims.sid)
        .map_err(|_| AppError::Unauthorized("无效的Session".to_string()))?;

    // 5. 检查 Token 是否在 Session 白名单中 (Active Session Validation)
    // Key: session:{sid}
    // 白名单机制：Session 必须存在于 Redis 中才有效。
    // 这允许服务端通过删除 Redis Key 来立即废除特定的 Access Token (即使它还没过期)。
    // 这比黑名单更安全 (Fail Closed)，因为如果 Redis 挂了或 Key 丢失，Token 会被视为无效，
    // 客户端将通过 refresh_token 流程重新获取 Token 并重建白名单。
    let whitelist_key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, claims.sid);
    let is_valid: bool = redis::cmd("EXISTS")
        .arg(&whitelist_key)
        .query_async(&mut state.redis.clone())
        .await
        .unwrap_or(false); // Redis error defaults to false (fail closed for security)

    if !is_valid {
        return Err(AppError::Unauthorized("Session无效或已过期".to_string()));
    }

    // 6. 将 user_id / session_id 注入到请求扩展中
    req.extensions_mut().insert(user_id);
    req.extensions_mut().insert(Some(session_id));

    // 7. 继续处理请求
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
            .ok_or_else(|| AppError::TokenMissing("缺少认证令牌".to_string()))?;

        let token = auth_header
            .strip_prefix("Bearer ")
            .ok_or_else(|| AppError::TokenMissing("令牌格式错误".to_string()))?;

        Ok(BearerToken(token.to_string()))
    }
}
