/// 幂等性中间件实现
///
/// 服务端控制幂等性设计规范
///
/// 核心原则：
/// - 零信任原则：不依赖客户端生成或提供幂等键
/// - 服务端完全控制：基于请求内容自动生成幂等键
/// - 透明化处理：客户端无需任何修改
use axum::{
    body::Body,
    extract::{Request, State},
    http::{HeaderValue, StatusCode},
    middleware::Next,
    response::Response,
};
use chrono::{DateTime, Timelike, Utc};
use redis::AsyncCommands;
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

use crate::{error::AppError, state::AppState};

/// 幂等键生成器
pub struct IdempotencyKeyGenerator;

impl IdempotencyKeyGenerator {
    /// 生成幂等键
    ///
    /// 参数:
    /// - user_id: 从JWT token中提取的用户ID
    /// - endpoint: API端点路径（如 "/api/v1/protected/chat_messages/send_message"）
    /// - body: 请求体JSON（已标准化）
    /// - time_window: 时间窗口（5分钟为单位）
    pub fn generate(
        user_id: i64,
        endpoint: &str,
        body: &str,
        time_window: DateTime<Utc>,
    ) -> String {
        let mut hasher = Sha256::new();
        hasher.update(user_id.to_string());
        hasher.update(endpoint);
        hasher.update(body);
        hasher.update(time_window.to_rfc3339());

        let result = hasher.finalize();
        format!("{:x}", result)
    }

    /// 计算时间窗口（向下取整到5分钟）
    ///
    /// 例如:
    /// - 2025-11-08T10:27:30Z → 2025-11-08T10:25:00Z
    /// - 2025-11-08T10:33:45Z → 2025-11-08T10:30:00Z
    pub fn calculate_time_window(now: DateTime<Utc>) -> DateTime<Utc> {
        let minutes = now.minute();
        let window_minutes = (minutes / 5) * 5;
        now.with_minute(window_minutes)
            .unwrap()
            .with_second(0)
            .unwrap()
            .with_nanosecond(0)
            .unwrap()
    }
}

/// JSON 标准化器
pub struct JsonNormalizer;

impl JsonNormalizer {
    /// 标准化JSON（确保相同内容生成相同幂等键）
    ///
    /// 规则:
    /// 1. 移除所有空白字符
    /// 2. Key按字母顺序排序
    /// 3. 忽略特殊字段: client_request_id, timestamp, nonce
    pub fn normalize(json: &Value) -> String {
        match json {
            Value::Object(map) => {
                let filtered: BTreeMap<String, Value> = map
                    .iter()
                    .filter(|(key, _)| {
                        !matches!(key.as_str(), "client_request_id" | "timestamp" | "nonce")
                    })
                    .map(|(k, v)| (k.clone(), v.clone()))
                    .collect();

                serde_json::to_string(&filtered).unwrap_or_default()
            }
            _ => serde_json::to_string(json).unwrap_or_default(),
        }
    }
}

/// 幂等性中间件
///
/// 仅对需要幂等处理的端点生效
pub async fn idempotency_middleware(
    State(state): State<AppState>,
    mut req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 1. 检查是否需要幂等处理
    if !should_check_idempotency(req.uri().path()) {
        return Ok(next.run(req).await);
    }

    // 2. 提取上下文
    let endpoint = req.uri().path().to_string();

    // 3. 读取请求体
    let (parts, body) = req.into_parts();

    let user_id = parts
        .extensions
        .get::<i64>()
        .copied()
        .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))?;

    let bytes = axum::body::to_bytes(body, usize::MAX)
        .await
        .map_err(|_| AppError::BadRequest("无法读取请求体".to_string()))?;

    // 4. 解析并标准化JSON
    let json_value: Value = serde_json::from_slice(&bytes)
        .map_err(|_| AppError::BadRequest("无效的JSON格式".to_string()))?;
    let normalized_body = JsonNormalizer::normalize(&json_value);

    // 5. 计算时间窗口
    let time_window = IdempotencyKeyGenerator::calculate_time_window(Utc::now());

    // 6. 生成幂等键
    let idempotency_key =
        IdempotencyKeyGenerator::generate(user_id, &endpoint, &normalized_body, time_window);

    // 7. 检查缓存
    let cache_key = format!("idempotency:{}", idempotency_key);
    let mut redis_conn = state.redis.clone();

    if let Ok(Some(cached_response)) = redis_conn.get::<&str, Option<String>>(&cache_key).await {
        // 返回缓存的响应
        tracing::debug!("幂等键命中缓存: {}", idempotency_key);
        return Ok(build_cached_response(&cached_response, &idempotency_key));
    }

    // 8. 重新构建请求
    req = Request::from_parts(parts, Body::from(bytes));

    // 9. 执行业务逻辑
    let response = next.run(req).await;

    // 10. 如果成功，缓存响应
    if response.status().is_success() {
        // 提取响应体
        let (parts, body) = response.into_parts();
        let bytes = axum::body::to_bytes(body, usize::MAX)
            .await
            .unwrap_or_default();
        let response_body = String::from_utf8_lossy(&bytes).to_string();

        // 缓存到Redis（10分钟 = 2倍时间窗口）
        let _: () = redis::cmd("SETEX")
            .arg(&cache_key)
            .arg(600)
            .arg(&response_body)
            .query_async(&mut redis_conn)
            .await
            .unwrap_or_default();

        // 重新构建响应
        let mut response = Response::from_parts(parts, Body::from(bytes));
        if let Ok(header_value) = HeaderValue::from_str(&idempotency_key) {
            response
                .headers_mut()
                .insert("X-Idempotent-Key", header_value);
        }
        Ok(response)
    } else {
        Ok(response)
    }
}

/// 判断是否需要幂等处理
fn should_check_idempotency(path: &str) -> bool {
    matches!(
        path,
        "/api/v1/protected/chat_messages/send_message"
            | "/api/v1/protected/moments/create"
            | "/api/v1/protected/user/me/follow_user"  // 好友 = 互相关注
            | "/api/v1/protected/group_chats/create"
            | "/api/v1/protected/media/upload_file"
    )
}

/// 构建缓存的响应
fn build_cached_response(cached_body: &str, idempotency_key: &str) -> Response {
    let mut response = Response::new(Body::from(cached_body.to_owned()));
    *response.status_mut() = StatusCode::OK;
    response
        .headers_mut()
        .insert("X-Idempotent-Replayed", HeaderValue::from_static("true"));
    if let Ok(header_value) = HeaderValue::from_str(idempotency_key) {
        response
            .headers_mut()
            .insert("X-Idempotent-Key", header_value);
    }
    response
}
