use axum::{extract::Request, extract::State, middleware::Next, response::Response};
use std::net::IpAddr;
use std::str::FromStr;

use crate::{error::AppError, state::AppState, utils::cache::CacheManager};

async fn apply_limit(
    cache: &CacheManager,
    key_prefix: &str,
    identity: &str,
    limit: u32,
    window_secs: u64,
) -> Result<(), AppError> {
    let key = format!("rate_limit:endpoint:{}:{}", key_prefix, identity);
    let count = cache.incr(&key, 1).await.unwrap_or(0) as u32;

    if count == 1 {
        let _ = cache.expire(&key, window_secs).await;
    }

    if count > limit {
        return Err(match window_secs {
            60 => AppError::MinuteRateLimitExceeded,
            3600 => AppError::HourlyRateLimitExceeded,
            86400 => AppError::DailyRateLimitExceeded,
            _ => AppError::RateLimited("请求过于频繁，请稍后再试".to_string()),
        });
    }

    Ok(())
}

fn parse_ip(s: &str) -> Option<IpAddr> {
    let ip = s.trim().trim_matches('"');
    IpAddr::from_str(ip).ok()
}

fn parse_x_forwarded_for(v: &str) -> Option<IpAddr> {
    // X-Forwarded-For: client, proxy1, proxy2
    let first = v.split(',').next()?.trim();
    // 兼容 "ip:port"
    let ip_part = first.split(':').next().unwrap_or(first);
    parse_ip(ip_part)
}

fn parse_forwarded(v: &str) -> Option<IpAddr> {
    // Forwarded: for=192.0.2.60;proto=http;by=203.0.113.43
    // 这里只做最小可用解析：找到 for=... 并取第一个
    for part in v.split(';') {
        let part = part.trim();
        if let Some(rest) = part.strip_prefix("for=") {
            let token = rest.trim();
            // 兼容 for="[2001:db8:cafe::17]:4711"
            let token = token.trim_matches('"');
            if token.starts_with('[') {
                let end = token.find(']')?;
                return parse_ip(&token[1..end]);
            }
            let ip_part = token.split(':').next().unwrap_or(token);
            return parse_ip(ip_part);
        }
    }
    None
}

fn get_client_ip(req: &Request) -> String {
    // 优先使用 Nginx 反代注入的真实 IP（你当前部署模型下更准确）
    // 注意：如果未来服务直接暴露公网，应只在“可信反代”场景下信任这些 header
    if let Some(v) = req.headers().get("x-real-ip").and_then(|v| v.to_str().ok()) {
        if let Some(ip) = parse_ip(v) {
            return ip.to_string();
        }
    }

    if let Some(v) = req
        .headers()
        .get("x-forwarded-for")
        .and_then(|v| v.to_str().ok())
    {
        if let Some(ip) = parse_x_forwarded_for(v) {
            return ip.to_string();
        }
    }

    if let Some(v) = req.headers().get("forwarded").and_then(|v| v.to_str().ok()) {
        if let Some(ip) = parse_forwarded(v) {
            return ip.to_string();
        }
    }

    "unknown".to_string()
}

/// 频率限制中间件（全局）
pub async fn rate_limit_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, AppError> {
    let cache = CacheManager::new(state.redis.clone());
    // 获取客户端IP
    let client_ip = get_client_ip(&req);

    let user_id = req.extensions().get::<i64>().copied();

    // 动态获取限流配置的辅助函数
    let get_limit = |key: &str, default: u32| {
        let state = state.clone();
        let key = key.to_string();
        async move {
            let config_key = format!("rate_limit.{}", key);
            match state
                .system_config_service
                .get_config_value(&config_key)
                .await
            {
                Ok(Some(val)) => val.parse::<u32>().unwrap_or(default),
                _ => default,
            }
        }
    };

    // 检查全局频率限制（按IP）
    let global_key = format!("rate_limit:global:{}", client_ip);
    let global_count = cache.incr(&global_key, 1).await.unwrap_or(0) as u32;

    if global_count == 1 {
        // 设置过期时间（1小时）
        let _ = cache.expire(&global_key, 3600).await;
    }

    let global_limit = get_limit("global_per_hour", state.config.rate_limit_global_per_hour).await;
    if global_count > global_limit {
        return Err(AppError::HourlyRateLimitExceeded);
    }

    // 如果有用户ID，检查用户级别的频率限制
    if let Some(user_id) = user_id {
        let user_key = format!("rate_limit:user:{}", user_id);
        let user_count = cache.incr(&user_key, 1).await.unwrap_or(0) as u32;

        if user_count == 1 {
            let _ = cache.expire(&user_key, 3600).await;
        }

        let user_limit = get_limit("user_per_hour", state.config.rate_limit_user_per_hour).await;
        if user_count > user_limit {
            return Err(AppError::HourlyRateLimitExceeded);
        }
    }

    // 端点级限流策略（更细粒度）
    let path = req.uri().path();
    let method = req.method().as_str();
    match (method, path) {
        ("POST", "/api/v1/auth/send_verification_code") => {
            let limit = get_limit(
                "send_code_per_minute",
                state.config.rate_limit_send_code_per_minute,
            )
            .await;
            apply_limit(&cache, "auth:send_code:ip", &client_ip, limit, 60).await?;
        }
        ("POST", "/api/v1/auth/verify_code") => {
            let limit = get_limit(
                "verify_code_per_10minutes",
                state.config.rate_limit_verify_code_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:verify_code:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/register") => {
            let limit = get_limit(
                "register_per_10minutes",
                state.config.rate_limit_register_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:register:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/login") => {
            let limit = get_limit(
                "login_per_10minutes",
                state.config.rate_limit_login_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:login:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/guest") => {
            let limit = get_limit(
                "guest_per_10minutes",
                state.config.rate_limit_guest_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:guest:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/refresh_token") => {
            let limit = get_limit(
                "refresh_per_10minutes",
                state.config.rate_limit_refresh_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:refresh:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/request_password_reset") => {
            let limit = get_limit(
                "reset_request_per_minute",
                state.config.rate_limit_reset_request_per_minute,
            )
            .await;
            apply_limit(&cache, "auth:reset_request:ip", &client_ip, limit, 60).await?;
        }
        ("POST", "/api/v1/auth/verify_reset_code") => {
            let limit = get_limit(
                "reset_verify_per_10minutes",
                state.config.rate_limit_reset_verify_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:reset_verify:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/auth/reset_password") => {
            let limit = get_limit(
                "reset_per_10minutes",
                state.config.rate_limit_reset_per_10minutes,
            )
            .await;
            apply_limit(&cache, "auth:reset:ip", &client_ip, limit, 600).await?;
        }
        ("POST", "/api/v1/moments/create") => {
            if let Some(uid) = user_id {
                let limit = get_limit(
                    "moments_create_per_minute",
                    state.config.rate_limit_moments_create_per_minute,
                )
                .await;
                apply_limit(&cache, "moments:create:user", &uid.to_string(), limit, 60).await?;
            }
        }
        ("POST", "/api/v1/comments/create") => {
            if let Some(uid) = user_id {
                let limit = get_limit(
                    "comments_create_per_minute",
                    state.config.rate_limit_comments_create_per_minute,
                )
                .await;
                apply_limit(&cache, "comments:create:user", &uid.to_string(), limit, 60).await?;
            }
        }
        ("POST", "/api/v1/reports/create") => {
            let identity = user_id
                .map(|uid| uid.to_string())
                .unwrap_or_else(|| client_ip.clone());
            let limit = get_limit(
                "reports_create_per_hour",
                state.config.rate_limit_reports_create_per_hour,
            )
            .await;
            apply_limit(&cache, "reports:create", &identity, limit, 3600).await?;
        }
        _ => {}
    }

    Ok(next.run(req).await)
}
