use axum::{
    extract::{Request, State},
    middleware::Next,
    response::Response,
};
use std::net::SocketAddr;

use crate::{error::AppError, state::AppState};

/// 频率限制中间件（全局）
pub async fn rate_limit_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 获取客户端IP
    let client_ip = req
        .extensions()
        .get::<SocketAddr>()
        .map(|addr| addr.ip().to_string())
        .unwrap_or_else(|| "unknown".to_string());

    // 检查全局频率限制（按IP）
    let global_key = format!("rate_limit:global:{}", client_ip);
    let global_count: u32 = redis::cmd("INCR")
        .arg(&global_key)
        .query_async(&mut state.redis.clone())
        .await
        .unwrap_or(0);

    if global_count == 1 {
        // 设置过期时间（1小时）
        let _: () = redis::cmd("EXPIRE")
            .arg(&global_key)
            .arg(3600)
            .query_async(&mut state.redis.clone())
            .await
            .unwrap_or(());
    }

    if global_count > state.config.rate_limit_global_per_hour {
        return Err(AppError::RateLimitExceeded(
            "请求过于频繁，请稍后再试".to_string(),
        ));
    }

    // 如果有用户ID，检查用户级别的频率限制
    if let Some(user_id) = req.extensions().get::<i64>() {
        let user_key = format!("rate_limit:user:{}", user_id);
        let user_count: u32 = redis::cmd("INCR")
            .arg(&user_key)
            .query_async(&mut state.redis.clone())
            .await
            .unwrap_or(0);

        if user_count == 1 {
            let _: () = redis::cmd("EXPIRE")
                .arg(&user_key)
                .arg(3600)
                .query_async(&mut state.redis.clone())
                .await
                .unwrap_or(());
        }

        if user_count > state.config.rate_limit_user_per_hour {
            return Err(AppError::RateLimitExceeded(
                "请求过于频繁，请稍后再试".to_string(),
            ));
        }
    }

    Ok(next.run(req).await)
}

