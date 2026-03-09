use axum::{extract::Request, extract::State, middleware::Next, response::Response};

use crate::{
    constants::{SECS_10_MINUTES, SECS_5_MINUTES, SECS_PER_DAY, SECS_PER_HOUR, SECS_PER_MINUTE},
    error::ApiError,
    state::AppState,
    utils::cache::CacheManager,
    utils::client_ip::get_client_ip_from_headers,
};

async fn apply_limit(
    cache: &CacheManager,
    key_prefix: &str,
    identity: &str,
    limit: u32,
    window_secs: u64,
    fail_open: bool,
) -> Result<(), ApiError> {
    let key = format!("rate_limit:endpoint:{}:{}", key_prefix, identity);

    // 使用 Lua 脚本保证 INCR 和 EXPIRE 的原子性
    // 避免竞态条件：如果 INCR 后但 EXPIRE 前进程崩溃，计数器会永久存在
    let script = r#"
        local count = redis.call('INCR', KEYS[1])
        if count == 1 then
            redis.call('EXPIRE', KEYS[1], ARGV[1])
        end
        return count
    "#;

    let count: u32 = match cache
        .eval_script(script, &[&key], &[&window_secs.to_string()])
        .await
    {
        Ok(c) => c,
        Err(e) => {
            tracing::error!(
                error = %e,
                key = %key,
                "Failed to execute rate limit script in Redis"
            );

            if !fail_open {
                // fail-closed: 拒绝请求，优先保证安全性
                return Err(ApiError::ServiceUnavailable);
            }

            // fail-open: 允许请求通过，优先保证可用性
            return Ok(());
        }
    };

    if count > limit {
        // 详细记录触发的限制信息
        tracing::warn!(
            limit_type = key_prefix,
            identity = identity,
            current_count = count,
            limit = limit,
            window_seconds = window_secs,
            "Rate limit exceeded"
        );

        return Err(match window_secs {
            crate::constants::SECS_PER_MINUTE => ApiError::MinuteRateLimitExceeded,
            crate::constants::SECS_PER_HOUR => ApiError::HourlyRateLimitExceeded,
            SECS_PER_DAY => ApiError::DailyRateLimitExceeded,
            _ => ApiError::RateLimited("请求过于频繁，请稍后再试".to_string()),
        });
    }

    Ok(())
}

/// 频率限制中间件（全局）
pub async fn rate_limit_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, ApiError> {
    let cache = CacheManager::new(state.redis.clone());
    // 获取客户端IP
    let client_ip = get_client_ip_from_headers(req.headers());

    let user_id = req.extensions().get::<i64>().copied();

    // 动态获取限流配置的辅助函数
    let get_limit = |key: &str, default: u32| {
        let state = state.clone();
        let key = key.to_string();
        async move {
            let config_key = format!("rate_limits.{}", key);
            match state
                .system_config_service
                .get_config_value(&config_key)
                .await
            {
                Ok(Some(val)) => val.parse::<u32>().unwrap_or_else(|e| {
                    tracing::warn!(
                        error = %e,
                        config_key = %config_key,
                        value = %val,
                        default = default,
                        "Failed to parse rate limit config value, using default"
                    );
                    default
                }),
                Ok(None) => default,
                Err(e) => {
                    tracing::error!(
                        error = %e,
                        config_key = %config_key,
                        default = default,
                        "Failed to fetch rate limit config, using default"
                    );
                    default
                }
            }
        }
    };

    // IP级限流（按IP地址限制，考虑多用户共享同一IP的场景，如公司/学校/家庭网络）
    let ip_key = format!("rate_limit:ip:{}", client_ip);
    let ip_count = cache.incr(&ip_key, 1).await.unwrap_or_else(|e| {
        tracing::error!(
            error = %e,
            key = %ip_key,
            client_ip = %client_ip,
            "Failed to increment IP rate limit counter"
        );
        0
    }) as u32;

    if ip_count == 1 {
        // 设置过期时间（1小时）
        let _ = cache.expire(&ip_key, SECS_PER_HOUR).await;
    }

    let ip_limit = get_limit("global_per_hour", state.config.rate_limit_global_per_hour).await;
    if ip_count > ip_limit {
        tracing::warn!(
            limit_type = "global_per_hour",
            client_ip = %client_ip,
            current_count = ip_count,
            limit = ip_limit,
            path = req.uri().path(),
            "IP rate limit exceeded (multiple users may share this IP)"
        );
        return Err(ApiError::HourlyRateLimitExceeded);
    }

    // 用户级限流（已认证用户）- 分层限流策略
    if let Some(user_id) = user_id {
        // 第一层：5 分钟窗口限流（短期防护，防止突发攻击）
        let user_key_5min = format!("rate_limit:user:5min:{}", user_id);
        let user_count_5min = cache.incr(&user_key_5min, 1).await.unwrap_or_else(|e| {
            tracing::error!(
                error = %e,
                user_id = user_id,
                "Failed to increment user 5min rate limit counter"
            );
            0
        }) as u32;

        if user_count_5min == 1 {
            let _ = cache.expire(&user_key_5min, SECS_5_MINUTES).await;
        }

        let user_limit_5min = get_limit(
            "user_per_5minutes",
            state.config.rate_limit_user_per_5minutes,
        )
        .await;
        if user_count_5min > user_limit_5min {
            tracing::warn!(
                limit_type = "user_per_5minutes",
                user_id = user_id,
                client_ip = %client_ip,
                current_count = user_count_5min,
                limit = user_limit_5min,
                window_seconds = SECS_5_MINUTES,
                path = req.uri().path(),
                "User rate limit exceeded (5 minutes window)"
            );
            return Err(ApiError::RateLimited(
                "请求过于频繁，请稍后 5 分钟再试".to_string(),
            ));
        }

        // 第二层：1 小时窗口限流（长期防护，补充保护）
        let user_key_hour = format!("rate_limit:user:hour:{}", user_id);
        let user_count_hour = cache.incr(&user_key_hour, 1).await.unwrap_or_else(|e| {
            tracing::error!(
                error = %e,
                user_id = user_id,
                "Failed to increment user hourly rate limit counter"
            );
            0
        }) as u32;

        if user_count_hour == 1 {
            let _ = cache.expire(&user_key_hour, SECS_PER_HOUR).await;
        }

        let user_limit_hour =
            get_limit("user_per_hour", state.config.rate_limit_user_per_hour).await;
        if user_count_hour > user_limit_hour {
            tracing::warn!(
                limit_type = "user_per_hour",
                user_id = user_id,
                client_ip = %client_ip,
                current_count = user_count_hour,
                limit = user_limit_hour,
                window_seconds = SECS_PER_HOUR,
                path = req.uri().path(),
                "User rate limit exceeded (hourly window)"
            );
            return Err(ApiError::HourlyRateLimitExceeded);
        }
    }

    // 端点级限流策略（更细粒度）
    let path = req.uri().path();
    let method = req.method().as_str();
    match (method, path) {
        ("POST", "/api/v1/auth/register") => {
            let limit = get_limit(
                "auth_register_per_10minutes",
                state.config.rate_limit_register_per_10minutes,
            )
            .await;
            apply_limit(
                &cache,
                "auth:register:ip",
                &client_ip,
                limit,
                SECS_10_MINUTES,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        ("POST", "/api/v1/auth/login") => {
            let limit = get_limit(
                "auth_login_per_10minutes",
                state.config.rate_limit_login_per_10minutes,
            )
            .await;
            apply_limit(
                &cache,
                "auth:login:ip",
                &client_ip,
                limit,
                SECS_10_MINUTES,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        ("POST", "/api/v1/auth/request_password_reset") => {
            let limit = get_limit(
                "auth_reset_per_10minutes",
                state.config.rate_limit_auth_reset_per_10minutes,
            )
            .await;
            apply_limit(
                &cache,
                "auth:reset:ip",
                &client_ip,
                limit,
                SECS_10_MINUTES,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        ("POST", "/api/v1/auth/guest") => {
            let limit = get_limit(
                "guest_per_10minutes",
                state.config.rate_limit_guest_per_10minutes,
            )
            .await;
            apply_limit(
                &cache,
                "auth:guest:ip",
                &client_ip,
                limit,
                SECS_10_MINUTES,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        ("POST", "/api/v1/auth/refresh_token") => {
            let limit = get_limit(
                "refresh_per_10minutes",
                state.config.rate_limit_refresh_per_10minutes,
            )
            .await;
            apply_limit(
                &cache,
                "auth:refresh:ip",
                &client_ip,
                limit,
                SECS_10_MINUTES,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        ("POST", "/api/v1/moments/create") => {
            if let Some(uid) = user_id {
                let limit = get_limit(
                    "moments_create_per_minute",
                    state.config.rate_limit_moments_create_per_minute,
                )
                .await;
                apply_limit(
                    &cache,
                    "moments:create:user",
                    &uid.to_string(),
                    limit,
                    SECS_PER_MINUTE,
                    state.config.rate_limit_fail_open,
                )
                .await?;
            }
        }
        ("POST", "/api/v1/comments/create") => {
            if let Some(uid) = user_id {
                let limit = get_limit(
                    "comments_create_per_minute",
                    state.config.rate_limit_comments_create_per_minute,
                )
                .await;
                apply_limit(
                    &cache,
                    "comments:create:user",
                    &uid.to_string(),
                    limit,
                    SECS_PER_MINUTE,
                    state.config.rate_limit_fail_open,
                )
                .await?;
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
            apply_limit(
                &cache,
                "reports:create",
                &identity,
                limit,
                SECS_PER_HOUR,
                state.config.rate_limit_fail_open,
            )
            .await?;
        }
        _ => {}
    }

    Ok(next.run(req).await)
}
