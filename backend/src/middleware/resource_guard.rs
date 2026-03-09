//! 资源存在性检查中间件
//!
//! 使用布隆过滤器快速过滤掉不存在的资源请求，
//! 防止恶意请求穿透缓存直接查询数据库

use crate::{dto::response::ApiResponse, error::ApiError, state::AppState};
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
use uuid::Uuid;

const MAX_GUARD_BODY_BYTES: usize = 512 * 1024;

/// 布隆过滤器存在性检查（宏封装，避免闭包生命周期问题）
macro_rules! check_bloom_ids {
    ($ids:expr, $single_expr:expr, $batch_expr:expr, $code:expr, $msg:expr) => {
        if !$ids.is_empty() {
            let results = if $ids.len() == 1 {
                match $single_expr.await {
                    Ok(v) => vec![v],
                    Err(e) => {
                        tracing::warn!("Bloom filter check failed: {}", e);
                        vec![true]
                    }
                }
            } else {
                match $batch_expr.await {
                    Ok(v) => v,
                    Err(e) => {
                        tracing::warn!("Bloom filter batch check failed: {}", e);
                        vec![true; $ids.len()]
                    }
                }
            };
            if let Some((idx, _)) = results.iter().enumerate().find(|(_, v)| !**v) {
                tracing::debug!("Bloom filter rejected: {}", &$ids[idx]);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error($code, $msg.to_string())),
                )
                    .into_response();
            }
        }
    };
}

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
                    Json(ApiResponse::<()>::error(10000, "请求体过大".to_string())),
                )
                    .into_response();
            }
        }

        let bytes = match to_bytes(body, MAX_GUARD_BODY_BYTES).await {
            Ok(b) => b,
            Err(_) => {
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(10000, "请求体过大".to_string())),
                )
                    .into_response();
            }
        };
        let value: Value = match serde_json::from_slice(&bytes) {
            Ok(v) => v,
            Err(e) => {
                let err = ApiError::JsonParseError(e.to_string());
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
    check_bloom_ids!(
        user_ids,
        state.resource_checker.user_might_exist(user_ids[0]),
        state.resource_checker.user_might_exist_batch(&user_ids),
        10300,
        "用户不存在"
    );

    let group_ids: Vec<String> = extracted.group_ids.iter().cloned().collect();
    check_bloom_ids!(
        group_ids,
        state.resource_checker.group_might_exist(&group_ids[0]),
        state.resource_checker.group_might_exist_batch(&group_ids),
        10300,
        "群组不存在"
    );

    let moment_ids: Vec<String> = extracted.moment_ids.iter().cloned().collect();
    check_bloom_ids!(
        moment_ids,
        state.resource_checker.moment_might_exist(&moment_ids[0]),
        state.resource_checker.moment_might_exist_batch(&moment_ids),
        10300,
        "瞬间不存在"
    );

    let comment_ids: Vec<String> = extracted.comment_ids.iter().cloned().collect();
    check_bloom_ids!(
        comment_ids,
        state.resource_checker.comment_might_exist(&comment_ids[0]),
        state
            .resource_checker
            .comment_might_exist_batch(&comment_ids),
        10300,
        "评论不存在"
    );

    let message_ids: Vec<String> = extracted.message_ids.iter().cloned().collect();
    check_bloom_ids!(
        message_ids,
        state.resource_checker.message_might_exist(&message_ids[0]),
        state
            .resource_checker
            .message_might_exist_batch(&message_ids),
        10300,
        "消息不存在"
    );

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
                "group_id" => {
                    out.group_ids.insert(normalize_i64_id(value));
                }
                "moment_id" => {
                    out.moment_ids.insert(normalize_uuid_id(value));
                }
                "comment_id" => {
                    out.comment_ids.insert(normalize_uuid_id(value));
                }
                "message_id" => {
                    out.message_ids.insert(normalize_uuid_id(value));
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
                    "group_id" => {
                        if let Some(id) = v.as_str() {
                            out.group_ids.insert(normalize_i64_id(id));
                        }
                    }
                    "moment_id" => {
                        if let Some(id) = v.as_str() {
                            out.moment_ids.insert(normalize_uuid_id(id));
                        }
                    }
                    "comment_id" => {
                        if let Some(id) = v.as_str() {
                            out.comment_ids.insert(normalize_uuid_id(id));
                        }
                    }
                    "message_id" => {
                        if let Some(id) = v.as_str() {
                            out.message_ids.insert(normalize_uuid_id(id));
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

fn normalize_i64_id(raw: &str) -> String {
    raw.parse::<i64>()
        .map(|v| v.to_string())
        .unwrap_or_else(|_| raw.to_string())
}

fn normalize_uuid_id(raw: &str) -> String {
    Uuid::parse_str(raw)
        .map(|v| v.to_string())
        .unwrap_or_else(|_| raw.to_string())
}
