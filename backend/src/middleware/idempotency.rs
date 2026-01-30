/// 幂等性中间件实现（最佳实践版）
///
/// 核心原则：
/// - **避免误去重**：同内容的“重复意图”必须允许（例如用户连续发送两条相同文本）。
/// - **客户端意图唯一**：仅当客户端提供 Idempotency-Key（或 X-Idempotency-Key）时才启用幂等。
/// - **不依赖请求体**：幂等键只基于 (user_id + endpoint + client_key) 计算，兼容 multipart 等非 JSON 请求。
/// - **仅缓存业务成功**：只缓存 ApiResponse.code == 0 的响应，避免把错误响应“钉死”。
use axum::{
    body::Body,
    extract::{Request, State},
    http::{HeaderMap, HeaderValue, StatusCode},
    middleware::Next,
    response::Response,
};
use redis::AsyncCommands;
use sha2::{Digest, Sha256};

use crate::{error::AppError, state::AppState};

const IDEMPOTENCY_TTL_SECONDS: usize = 24 * 60 * 60; // 24小时
const IDEMPOTENCY_LOCK_TTL_SECONDS: usize = 30; // 处理锁TTL

/// 幂等性中间件
///
/// 仅对需要幂等处理的端点生效
pub async fn idempotency_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 1. 检查是否需要幂等处理
    if !should_check_idempotency(req.uri().path()) {
        return Ok(next.run(req).await);
    }

    // 2. 读取客户端幂等键：没有则根据端点规则决定是否强制
    let client_key = match extract_client_idempotency_key(req.headers()) {
        Some(v) => v,
        None => {
            if requires_idempotency_key(req.uri().path()) {
                return Err(AppError::MissingParameter("Idempotency-Key".to_string()));
            }
            return Ok(next.run(req).await);
        }
    };

    // 3. 提取上下文
    let endpoint = req.uri().path().to_string();

    // 4. 提取 user_id（由 auth_middleware 注入 extensions）
    let user_id = req
        .extensions()
        .get::<i64>()
        .copied()
        .ok_or_else(|| AppError::Unauthorized("未认证".to_string()))?;

    // 5. 生成服务端缓存用幂等 hash key
    let hashed_key = hash_idempotency_key(user_id, &endpoint, &client_key);

    // 6. 检查缓存
    let cache_key = format!("idempotency:{}", hashed_key);
    let meta_key = format!("idempotency:meta:{}:{}", user_id, client_key);
    let mut redis_conn = state.redis.clone();

    if let Ok(Some(cached_response)) = redis_conn.get::<&str, Option<String>>(&cache_key).await {
        tracing::debug!("幂等键命中缓存: {}", hashed_key);
        return Ok(build_cached_response(
            &cached_response,
            &client_key,
            &hashed_key,
        ));
    }

    if let Ok(Some(endpoint)) = redis_conn.get::<&str, Option<String>>(&meta_key).await {
        if endpoint != req.uri().path() {
            return Err(AppError::IdempotencyKeyConflict(
                "幂等键已用于其他请求".to_string(),
            ));
        }
        return Err(AppError::IdempotencyCacheExpired);
    }

    // 7. 加锁，防止并发重复执行
    let lock_key = format!("idempotency:lock:{}", hashed_key);
    let lock_result: Option<String> = redis::cmd("SET")
        .arg(&lock_key)
        .arg("1")
        .arg("NX")
        .arg("EX")
        .arg(IDEMPOTENCY_LOCK_TTL_SECONDS)
        .query_async(&mut redis_conn)
        .await
        .unwrap_or(None);
    if lock_result.is_none() {
        return Err(AppError::IdempotencyOperationInProgress);
    }

    // 8. 执行业务逻辑
    let response = next.run(req).await;

    // 9. 只缓存业务成功（ApiResponse.code == 0）
    //    注意：业务成功与否以 ApiResponse.code 为准，不依赖 HTTP status。
    let (parts, body) = response.into_parts();
    let bytes = axum::body::to_bytes(body, usize::MAX)
        .await
        .unwrap_or_default();
    let response_body = String::from_utf8_lossy(&bytes).to_string();

    let should_cache = serde_json::from_slice::<serde_json::Value>(&bytes)
        .ok()
        .and_then(|v| v.get("code").and_then(|c| c.as_i64()))
        .map(|code| code == 0)
        .unwrap_or(false);

    if should_cache {
        let _: () = redis::cmd("SETEX")
            .arg(&cache_key)
            .arg(IDEMPOTENCY_TTL_SECONDS)
            .arg(&response_body)
            .query_async(&mut redis_conn)
            .await
            .unwrap_or_default();

        let _: () = redis::cmd("SETEX")
            .arg(&meta_key)
            .arg(IDEMPOTENCY_TTL_SECONDS * 2)
            .arg(endpoint)
            .query_async(&mut redis_conn)
            .await
            .unwrap_or_default();
    }

    let _: () = redis::cmd("DEL")
        .arg(&lock_key)
        .query_async(&mut redis_conn)
        .await
        .unwrap_or_default();

    // 9. 重新构建响应并写入幂等相关 header
    let mut response = Response::from_parts(parts, Body::from(bytes));
    attach_idempotency_headers(&mut response, &client_key, &hashed_key, false);
    Ok(response)
}

/// 判断是否需要幂等处理
fn should_check_idempotency(path: &str) -> bool {
    matches!(
        path,
        // 发送消息：客户端必须提供 key（但同内容不同 key 允许重复意图）
        "/api/v1/private_messages/send" | "/api/v1/group_messages/send"
            // 创建类
            | "/api/v1/moments/create"
            | "/api/v1/reports/create"
            | "/api/v1/group_chats/create"
            // 群组成员操作
            | "/api/v1/group_chats/members/join"
            | "/api/v1/group_chats/members/apply"
            | "/api/v1/group_chats/applications/approve"
            | "/api/v1/group_chats/applications/reject"
            | "/api/v1/group_chats/me/invites/accept"
            | "/api/v1/group_chats/me/invites/reject"
            // 上传：仅在客户端提供 key 时去重
            | "/api/v1/media/complete_upload"
            // 关注：虽然天然幂等，但可通过 key 去重重试，减少写放大
            | "/api/v1/user/me/follow"
    )
}

fn requires_idempotency_key(path: &str) -> bool {
    matches!(
        path,
        "/api/v1/private_messages/send"
            | "/api/v1/group_messages/send"
            | "/api/v1/moments/create"
            | "/api/v1/reports/create"
            | "/api/v1/group_chats/members/join"
            | "/api/v1/group_chats/applications/approve"
            | "/api/v1/group_chats/applications/reject"
            | "/api/v1/group_chats/me/invites/accept"
            | "/api/v1/group_chats/me/invites/reject"
    )
}

fn extract_client_idempotency_key(headers: &HeaderMap) -> Option<String> {
    let raw = headers
        .get("Idempotency-Key")
        .or_else(|| headers.get("X-Idempotency-Key"))?;
    let s = raw.to_str().ok()?.trim().to_string();
    if s.is_empty() {
        None
    } else {
        Some(s)
    }
}

fn hash_idempotency_key(user_id: i64, endpoint: &str, client_key: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(user_id.to_string());
    hasher.update(":");
    hasher.update(endpoint);
    hasher.update(":");
    hasher.update(client_key);
    format!("{:x}", hasher.finalize())
}

fn attach_idempotency_headers(
    response: &mut Response,
    client_key: &str,
    hashed_key: &str,
    replayed: bool,
) {
    // 回放响应需要显式设置 content-type
    response
        .headers_mut()
        .insert("Content-Type", HeaderValue::from_static("application/json"));

    response.headers_mut().insert(
        "X-Idempotent-Replayed",
        HeaderValue::from_static(if replayed { "true" } else { "false" }),
    );

    if let Ok(v) = HeaderValue::from_str(client_key) {
        response.headers_mut().insert("Idempotency-Key", v.clone());
        response.headers_mut().insert("X-Idempotency-Key", v);
    }
    if let Ok(v) = HeaderValue::from_str(hashed_key) {
        response.headers_mut().insert("X-Idempotency-Hash", v);
    }
}

/// 构建缓存命中的响应
fn build_cached_response(cached_body: &str, client_key: &str, hashed_key: &str) -> Response {
    let mut response = Response::new(Body::from(cached_body.to_owned()));
    *response.status_mut() = StatusCode::OK;
    attach_idempotency_headers(&mut response, client_key, hashed_key, true);
    response
}
