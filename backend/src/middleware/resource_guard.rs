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

use crate::{dto::response::ApiResponse, state::AppState};

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
                    Json(ApiResponse::<()>::error(
                        10005,
                        "请求体过大".to_string(),
                    )),
                )
                    .into_response();
            }
        }

        let bytes = match to_bytes(body, MAX_GUARD_BODY_BYTES).await {
            Ok(b) => b,
            Err(_) => {
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10005,
                        "请求体过大".to_string(),
                    )),
                )
                    .into_response();
            }
        };
        if let Ok(value) = serde_json::from_slice::<Value>(&bytes) {
            collect_ids_from_value(&value, &mut extracted);
        }
        Body::from(bytes)
    } else {
        body
    };

    // Bloom 过滤（false => 一定不存在）
    for user_id in &extracted.user_ids {
        match state.resource_checker.user_might_exist(*user_id).await {
            Ok(false) => {
                tracing::debug!("Bloom filter rejected user_id={}", user_id);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10310,
                        "用户不存在".to_string(),
                    )),
                )
                    .into_response();
            }
            Ok(true) => {}
            Err(e) => {
                tracing::warn!("Bloom filter check failed: {}", e);
            }
        }
    }
    for group_id in &extracted.group_ids {
        match state.resource_checker.group_might_exist(group_id).await {
            Ok(false) => {
                tracing::debug!("Bloom filter rejected group_id={}", group_id);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10320,
                        "群组不存在".to_string(),
                    )),
                )
                    .into_response();
            }
            Ok(true) => {}
            Err(e) => {
                tracing::warn!("Bloom filter check failed: {}", e);
            }
        }
    }
    for moment_id in &extracted.moment_ids {
        match state.resource_checker.moment_might_exist(moment_id).await {
            Ok(false) => {
                tracing::debug!("Bloom filter rejected moment_id={}", moment_id);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10330,
                        "瞬间不存在".to_string(),
                    )),
                )
                    .into_response();
            }
            Ok(true) => {}
            Err(e) => {
                tracing::warn!("Bloom filter check failed: {}", e);
            }
        }
    }
    for comment_id in &extracted.comment_ids {
        match state.resource_checker.comment_might_exist(comment_id).await {
            Ok(false) => {
                tracing::debug!("Bloom filter rejected comment_id={}", comment_id);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10340,
                        "评论不存在".to_string(),
                    )),
                )
                    .into_response();
            }
            Ok(true) => {}
            Err(e) => {
                tracing::warn!("Bloom filter check failed: {}", e);
            }
        }
    }
    for message_id in &extracted.message_ids {
        match state.resource_checker.message_might_exist(message_id).await {
            Ok(false) => {
                tracing::debug!("Bloom filter rejected message_id={}", message_id);
                return (
                    StatusCode::OK,
                    Json(ApiResponse::<()>::error(
                        10350,
                        "消息不存在".to_string(),
                    )),
                )
                    .into_response();
            }
            Ok(true) => {}
            Err(e) => {
                tracing::warn!("Bloom filter check failed: {}", e);
            }
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

// ============================================================
// 布隆过滤器数据同步
// 这些函数用于在创建资源时同步更新布隆过滤器
// ============================================================

use crate::utils::bloom_filter::ResourceExistenceChecker;

impl ResourceExistenceChecker {
    /// 批量同步用户到布隆过滤器
    pub async fn sync_users(&self, user_ids: &[i64]) -> Result<(), redis::RedisError> {
        for uid in user_ids {
            self.record_user(*uid).await?;
        }
        Ok(())
    }

    /// 批量同步群组到布隆过滤器
    pub async fn sync_groups(&self, group_ids: &[String]) -> Result<(), redis::RedisError> {
        for gid in group_ids {
            self.record_group(gid).await?;
        }
        Ok(())
    }

    /// 批量同步瞬间到布隆过滤器
    pub async fn sync_moments(&self, moment_ids: &[String]) -> Result<(), redis::RedisError> {
        for mid in moment_ids {
            self.record_moment(mid).await?;
        }
        Ok(())
    }

    /// 批量同步评论到布隆过滤器
    pub async fn sync_comments(&self, comment_ids: &[String]) -> Result<(), redis::RedisError> {
        for cid in comment_ids {
            self.record_comment(cid).await?;
        }
        Ok(())
    }

    /// 批量同步消息到布隆过滤器
    pub async fn sync_messages(&self, message_ids: &[String]) -> Result<(), redis::RedisError> {
        for mid in message_ids {
            self.record_message(mid).await?;
        }
        Ok(())
    }
}
