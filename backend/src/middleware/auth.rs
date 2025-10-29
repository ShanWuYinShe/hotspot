use axum::{
    extract::{Request, State},
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

/// 从请求中提取当前用户ID
pub fn extract_user_id(req: &Request) -> Result<i64, AppError> {
    req.extensions()
        .get::<i64>()
        .copied()
        .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))
}

