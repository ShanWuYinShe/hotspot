//! 资源存在性检查中间件
//!
//! 使用布隆过滤器快速过滤掉不存在的资源请求，
//! 防止恶意请求穿透缓存直接查询数据库

use axum::{
    body::{to_bytes, Body},
    extract::State,
    http::{header::CONTENT_LENGTH, header::CONTENT_TYPE, Request, StatusCode},
    middleware::Next,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::Value;
use std::collections::HashSet;

use crate::{dto::response::ApiResponse, error::AppError, state::AppState};

const MAX_GUARD_BODY_BYTES: usize = 512 * 1024;

/// 资源存在性保护中间件
///
/// - 使用布隆过滤器快速过滤不存在的 user/group/moment 请求
/// - 仅在布隆过滤器准备完成后启用（避免误判）
pub async fn resource_guard_middleware(
    State(state): State<AppState>,
    request: Request<Body>,
    next: Next,
) -> Response {
    let ready = state.resource_checker.is_ready().await.unwrap_or(false);
    if !ready {
        return next.run(request).await;
    }

    let (parts, body) = request.into_parts();
    let content_type = parts
        .headers
        .get(CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    let mut extracted = ExtractedIds::default();
    if let Some(query) = parts.uri.query() {
        collect_ids_from_query(query, &mut extracted);
    }

    let body = if content_type.starts_with("application/json") {
        if let Some(len) = parts
            .headers
            .get(CONTENT_LENGTH)
            .and_then(|v| v.to_str().ok())
            .and_then(|v| v.parse::<usize>().ok())
        {
            if len > MAX_GUARD_BODY_BYTES {
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(10005, "请求体过大".to_string())),
                )
                    .into_response();
            }
        }

        let bytes = match to_bytes(body, MAX_GUARD_BODY_BYTES).await {
            Ok(b) => b,
            Err(_) => {
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(10005, "请求体过大".to_string())),
                )
                    .into_response();
            }
        };
        let value: Value = match serde_json::from_slice(&bytes) {
            Ok(v) => v,
            Err(e) => {
                let err = AppError::JsonParseError(e.to_string());
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(err.error_code(), err.to_string())),
                )
                    .into_response();
            }
        };
        collect_ids_from_value(&value, &mut extracted);
        Body::from(bytes)
    } else {
        body
    };

    // Bloom 过滤（false => 一定不存在）
    let user_ids: Vec<i64> = extracted.user_ids.iter().copied().collect();
    if !user_ids.is_empty() {
        let results = if user_ids.len() == 1 {
            match state.resource_checker.user_might_exist(user_ids[0]).await {
                Ok(v) => vec![v],
                Err(e) => {
                    tracing::warn!("Bloom filter check failed: {}", e);
                    vec![true]
                }
            }
        } else {
            match state
                .resource_checker
                .user_might_exist_batch(&user_ids)
                .await
            {
                Ok(v) => v,
                Err(e) => {
                    tracing::warn!("Bloom filter batch check failed: {}", e);
                    vec![true; user_ids.len()]
                }
            }
        };
        if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
            let rejected_id = user_ids[idx];
            tracing::debug!("Bloom filter rejected user_id={}", rejected_id);
            return (
                StatusCode::OK,
                Json(ApiResponse::<()>::error(10310, "用户不存在".to_string())),
            )
                .into_response();
        }
    }

    let group_ids: Vec<String> = extracted.group_ids.iter().cloned().collect();
    if !group_ids.is_empty() {
        let results = if group_ids.len() == 1 {
            match state
                .resource_checker
                .group_might_exist(&group_ids[0])
                .await
            {
                Ok(v) => vec![v],
                Err(e) => {
                    tracing::warn!("Bloom filter check failed: {}", e);
                    vec![true]
                }
            }
        } else {
            match state
                .resource_checker
                .group_might_exist_batch(&group_ids)
                .await
            {
                Ok(v) => v,
                Err(e) => {
                    tracing::warn!("Bloom filter batch check failed: {}", e);
                    vec![true; group_ids.len()]
                }
            }
        };
        if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
            let rejected_id = &group_ids[idx];
            tracing::debug!("Bloom filter rejected group_id={}", rejected_id);
            return (
                StatusCode::OK,
                Json(ApiResponse::<()>::error(10320, "群组不存在".to_string())),
            )
                .into_response();
        }
    }

    let moment_ids: Vec<String> = extracted.moment_ids.iter().cloned().collect();
    if !moment_ids.is_empty() {
        let results = if moment_ids.len() == 1 {
            match state
                .resource_checker
                .moment_might_exist(&moment_ids[0])
                .await
            {
                Ok(v) => vec![v],
                Err(e) => {
                    tracing::warn!("Bloom filter check failed: {}", e);
                    vec![true]
                }
            }
        } else {
            match state
                .resource_checker
                .moment_might_exist_batch(&moment_ids)
                .await
            {
                Ok(v) => v,
                Err(e) => {
                    tracing::warn!("Bloom filter batch check failed: {}", e);
                    vec![true; moment_ids.len()]
                }
            }
        };
        if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
            let rejected_id = &moment_ids[idx];
            tracing::debug!("Bloom filter rejected moment_id={}", rejected_id);
            return (
                StatusCode::OK,
                Json(ApiResponse::<()>::error(10330, "瞬间不存在".to_string())),
            )
                .into_response();
        }
    }

    let comment_ids: Vec<String> = extracted.comment_ids.iter().cloned().collect();
    if !comment_ids.is_empty() {
        let results = if comment_ids.len() == 1 {
            match state
                .resource_checker
                .comment_might_exist(&comment_ids[0])
                .await
            {
                Ok(v) => vec![v],
                Err(e) => {
                    tracing::warn!("Bloom filter check failed: {}", e);
                    vec![true]
                }
            }
        } else {
            match state
                .resource_checker
                .comment_might_exist_batch(&comment_ids)
                .await
            {
                Ok(v) => v,
                Err(e) => {
                    tracing::warn!("Bloom filter batch check failed: {}", e);
                    vec![true; comment_ids.len()]
                }
            }
        };
        if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
            let rejected_id = &comment_ids[idx];
            tracing::debug!("Bloom filter rejected comment_id={}", rejected_id);
            return (
                StatusCode::OK,
                Json(ApiResponse::<()>::error(10340, "评论不存在".to_string())),
            )
                .into_response();
        }
    }

    let message_ids: Vec<String> = extracted.message_ids.iter().cloned().collect();
    if !message_ids.is_empty() {
        let results = if message_ids.len() == 1 {
            match state
                .resource_checker
                .message_might_exist(&message_ids[0])
                .await
            {
                Ok(v) => vec![v],
                Err(e) => {
                    tracing::warn!("Bloom filter check failed: {}", e);
                    vec![true]
                }
            }
        } else {
            match state
                .resource_checker
                .message_might_exist_batch(&message_ids)
                .await
            {
                Ok(v) => v,
                Err(e) => {
                    tracing::warn!("Bloom filter batch check failed: {}", e);
                    vec![true; message_ids.len()]
                }
            }
        };
        if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
            let rejected_id = &message_ids[idx];
            tracing::debug!("Bloom filter rejected message_id={}", rejected_id);
            return (
                StatusCode::OK,
                Json(ApiResponse::<()>::error(10350, "消息不存在".to_string())),
            )
                .into_response();
        }
    }

    let request = Request::from_parts(parts, body);
    next.run(request).await
}

#[derive(Default)]
struct ExtractedIds {
    user_ids: HashSet<i64>,
    group_ids: HashSet<String>,
    moment_ids: HashSet<String>,
    comment_ids: HashSet<String>,
    message_ids: HashSet<String>,
}

fn collect_ids_from_query(query: &str, out: &mut ExtractedIds) {
    for pair in query.split('&') {
        if let Some((key, value)) = pair.split_once('=') {
            match key {
                "user_id" => {
                    if let Ok(id) = value.parse::<i64>() {
                        out.user_ids.insert(id);
                    }
                }
                "group_chat_id" | "group_id" => {
                    out.group_ids.insert(value.to_string());
                }
                "moment_id" => {
                    out.moment_ids.insert(value.to_string());
                }
                "comment_id" => {
                    out.comment_ids.insert(value.to_string());
                }
                "message_id" => {
                    out.message_ids.insert(value.to_string());
                }
                _ => {}
            }
        }
    }
}

fn collect_ids_from_value(value: &Value, out: &mut ExtractedIds) {
    match value {
        Value::Object(map) => {
            for (k, v) in map {
                match k.as_str() {
                    "user_id" => {
                        let parsed = if let Some(id) = v.as_i64() {
                            Some(id)
                        } else if let Some(raw) = v.as_str() {
                            raw.parse::<i64>().ok()
                        } else {
                            None
                        };
                        if let Some(id) = parsed {
                            out.user_ids.insert(id);
                        }
                    }
                    "group_chat_id" | "group_id" => {
                        if let Some(id) = v.as_str() {
                            out.group_ids.insert(id.to_string());
                        }
                    }
                    "moment_id" => {
                        if let Some(id) = v.as_str() {
                            out.moment_ids.insert(id.to_string());
                        }
                    }
                    "comment_id" => {
                        if let Some(id) = v.as_str() {
                            out.comment_ids.insert(id.to_string());
                        }
                    }
                    "message_id" => {
                        if let Some(id) = v.as_str() {
                            out.message_ids.insert(id.to_string());
                        }
                    }
                    _ => {}
                }
                collect_ids_from_value(v, out);
            }
        }
        Value::Array(items) => {
            for item in items {
                collect_ids_from_value(item, out);
            }
        }
        _ => {}
    }
}
