use axum::{
    async_trait,
    extract::{FromRequestParts, Request, State},
    http::request::Parts,
    middleware::Next,
    response::Response,
};

use crate::{error::AppError, state::AppState, utils::jwt::JwtUtil};

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
        .ok_or_else(|| AppError::Unauthorized("缺少认证令牌".to_string()))?;

    // 2. 验证 Bearer 格式
    let token = auth_header
        .strip_prefix("Bearer ")
        .ok_or_else(|| AppError::Unauthorized("令牌格式错误".to_string()))?;

    // 3. 验证 JWT Token
    let claims = JwtUtil::verify_token(token, &state.config.jwt_secret)?;
    let user_id = claims
        .sub
        .parse::<i64>()
        .map_err(|_| AppError::Unauthorized("无效的用户ID".to_string()))?;

    // 4. 检查 Token 是否在黑名单中
    let blacklist_key = format!("token_blacklist:{}", token);
    let is_blacklisted: bool = redis::cmd("EXISTS")
        .arg(&blacklist_key)
        .query_async(&mut state.redis.clone())
        .await
        .unwrap_or(false);

    if is_blacklisted {
        return Err(AppError::Unauthorized("令牌已失效".to_string()));
    }

    // 5. 将 user_id 注入到请求扩展中
    req.extensions_mut().insert(user_id);

    // 6. 继续处理请求
    Ok(next.run(req).await)
}

/// 可选的用户ID提取器（用于公开API）
/// 
/// 如果请求已认证（有token），则返回Some(user_id)
/// 否则返回None
pub struct OptionalUserId(pub Option<i64>);

#[async_trait]
impl<S> FromRequestParts<S> for OptionalUserId
where
    S: Send + Sync,
{
    type Rejection = std::convert::Infallible;

    async fn from_request_parts(parts: &mut Parts, _state: &S) -> Result<Self, Self::Rejection> {
        let user_id = parts.extensions.get::<i64>().copied();
        Ok(OptionalUserId(user_id))
    }
}

/// Bearer Token 提取器
/// 
/// 从Authorization header中提取Bearer token
pub struct BearerToken(pub String);

#[async_trait]
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
            .ok_or_else(|| AppError::Unauthorized("缺少认证令牌".to_string()))?;

        let token = auth_header
            .strip_prefix("Bearer ")
            .ok_or_else(|| AppError::Unauthorized("令牌格式错误".to_string()))?;

        Ok(BearerToken(token.to_string()))
    }
}

