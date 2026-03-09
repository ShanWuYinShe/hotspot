//! WebSocket authentication flow.

use axum::extract::ws::{Message, WebSocket};
use axum::http::HeaderMap;
use futures_util::{SinkExt, StreamExt};
use std::time::Duration;
use tokio::time::timeout;

use crate::{
    error::ApiError,
    repositories::user_repo::UserRepository,
    state::AppState,
    types::enums::ClientType,
    utils::{jwt::JwtUtil, web_cookie::get_cookie_value},
    websocket::types::{ws_error_from_app_error, WsMessage},
};

type SplitSink = futures_util::stream::SplitSink<WebSocket, Message>;
type SplitStream = futures_util::stream::SplitStream<WebSocket>;

fn extract_request_meta(headers: &HeaderMap) -> (String, String) {
    let origin = headers
        .get("origin")
        .and_then(|value| value.to_str().ok())
        .unwrap_or("-")
        .to_string();
    let user_agent = headers
        .get("user-agent")
        .and_then(|value| value.to_str().ok())
        .unwrap_or("-")
        .to_string();
    (origin, user_agent)
}

fn log_auth_failure(reason: &str, client_type: Option<ClientType>, headers: &HeaderMap) {
    let (origin, user_agent) = extract_request_meta(headers);
    let client_type_label = client_type
        .map(|ct| ct.to_string())
        .unwrap_or_else(|| "unknown".to_string());
    tracing::warn!(
        target = "ws_auth",
        reason = %reason,
        client_type = %client_type_label,
        origin = %origin,
        user_agent = %user_agent,
        "WebSocket authentication failed"
    );
}

fn log_auth_success(user_id: i64, client_type: ClientType, auth_source: &str, headers: &HeaderMap) {
    let (origin, user_agent) = extract_request_meta(headers);
    tracing::info!(
        target = "ws_auth",
        user_id,
        client_type = %client_type,
        auth_source = %auth_source,
        origin = %origin,
        user_agent = %user_agent,
        "WebSocket authentication succeeded"
    );
}

/// Authenticates the WebSocket connection by reading the first Auth message.
/// Returns (user_id, device_fingerprint, nickname) on success.
/// On failure, sends error to client and returns Err(()).
pub async fn authenticate_ws_connection(
    sender: &mut SplitSink,
    receiver: &mut SplitStream,
    state: &AppState,
    headers: &HeaderMap,
) -> Result<(i64, String, String), ()> {
    let auth_msg = timeout(Duration::from_secs(5), receiver.next()).await;
    let auth_msg = match auth_msg {
        Ok(Some(Ok(Message::Text(text)))) => text,
        Ok(Some(Ok(_))) => {
            log_auth_failure("鉴权失败：首帧不是文本消息", None, headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketProtocolIncompatible("仅支持文本协议".to_string()),
            )
            .await;
            return Err(());
        }
        Ok(Some(Err(err))) => {
            let reason = format!("鉴权失败：读取首帧消息异常: {err}");
            log_auth_failure(reason.as_str(), None, headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketError("鉴权超时或连接异常".to_string()),
            )
            .await;
            return Err(());
        }
        Ok(None) => {
            log_auth_failure("鉴权失败：连接在鉴权前已关闭", None, headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketError("鉴权超时或连接异常".to_string()),
            )
            .await;
            return Err(());
        }
        Err(err) => {
            let reason = format!("鉴权失败：首帧超时: {err}");
            log_auth_failure(reason.as_str(), None, headers);
            // spec: 鉴权失败或超时，发送 Error(code=20000) 后断开连接
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketError("鉴权超时或连接异常".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let auth_event: WsMessage = match serde_json::from_str(auth_msg.as_str()) {
        Ok(m) => m,
        Err(err) => {
            let reason = format!("鉴权消息格式无效: {err}");
            log_auth_failure(reason.as_str(), None, headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketMessageFormatError("鉴权消息格式无效".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let (client_type, token_from_payload) = match auth_event {
        WsMessage::Auth {
            client_type,
            access_token,
            ..
        } => (client_type, access_token),
        _ => {
            log_auth_failure("协议不兼容：首帧必须是 Auth 事件", None, headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketProtocolIncompatible(
                    "协议不兼容：首帧必须是 Auth 事件".to_string(),
                ),
            )
            .await;
            return Err(());
        }
    };

    let mut auth_source: &'static str = "payload";
    let token_candidate = match token_from_payload {
        Some(token) if !token.is_empty() => Some(token),
        _ if client_type == ClientType::Web => {
            auth_source = "cookie";
            get_cookie_value(headers, "access_token")
        }
        _ => None,
    };

    let access_token = match token_candidate {
        Some(token) if !token.is_empty() => token,
        _ => {
            log_auth_failure("鉴权失败：未提供 Token", Some(client_type), headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketAuthFailed("鉴权失败：未提供 Token".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let claims = match JwtUtil::verify_token(&access_token, &state.config.jwt_secret) {
        Ok(c) => c,
        Err(ApiError::VerifyTokenInvalid(detail)) => {
            let reason = format!("鉴权失败：Token 已过期或无效: {detail}");
            log_auth_failure(reason.as_str(), Some(client_type), headers);
            send_auth_error_and_close(sender, ApiError::WebSocketTokenExpired).await;
            return Err(());
        }
        Err(err) => {
            let reason = format!("鉴权失败：Token 校验异常: {err}");
            log_auth_failure(reason.as_str(), Some(client_type), headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketAuthFailed("鉴权失败：Token 无效".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let user_id = match claims.sub.parse::<i64>() {
        Ok(uid) => uid,
        Err(err) => {
            let reason = format!("鉴权失败：Token subject 不是有效 user_id: {err}");
            log_auth_failure(reason.as_str(), Some(client_type), headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketAuthFailed("鉴权失败：Token 无效或过期".to_string()),
            )
            .await;
            return Err(());
        }
    };

    // Session 白名单/黑名单检查（与 HTTP auth 中间件一致）
    if state.revoked_sessions.contains_key(&claims.sid) {
        log_auth_failure("鉴权失败：会话已被撤销", Some(client_type), headers);
        send_auth_error_and_close(
            sender,
            ApiError::WebSocketAuthFailed("鉴权失败：会话已失效".to_string()),
        )
        .await;
        return Err(());
    }

    let whitelist_key = format!("{}:{}", crate::constants::REDIS_KEY_SESSION, claims.sid);
    let mut redis_conn = state.redis.clone();
    let session_valid: bool = match redis::cmd("EXISTS")
        .arg(&whitelist_key)
        .query_async(&mut redis_conn)
        .await
    {
        Ok(exists) => exists,
        Err(err) => {
            let reason = format!("鉴权失败：会话白名单检查异常: {err}");
            log_auth_failure(reason.as_str(), Some(client_type), headers);
            send_auth_error_and_close(
                sender,
                ApiError::WebSocketAuthFailed("鉴权失败：会话校验失败，请稍后重试".to_string()),
            )
            .await;
            return Err(());
        }
    };

    if !session_valid {
        log_auth_failure("鉴权失败：会话不在白名单中", Some(client_type), headers);
        send_auth_error_and_close(
            sender,
            ApiError::WebSocketAuthFailed("鉴权失败：会话已失效或过期".to_string()),
        )
        .await;
        return Err(());
    }

    let user_repo = UserRepository::new(state.db.clone());
    let nickname = match user_repo.find_by_id(user_id).await {
        Ok(Some(u)) => u.nickname.unwrap_or_else(|| "Unknown".to_string()),
        Ok(None) => {
            tracing::warn!(
                target = "ws_auth",
                user_id,
                "WebSocket auth succeeded but user profile not found, fallback nickname"
            );
            "Unknown".to_string()
        }
        Err(err) => {
            tracing::warn!(
                target = "ws_auth",
                user_id,
                error = %err,
                "WebSocket auth load profile failed, fallback nickname"
            );
            "Unknown".to_string()
        }
    };

    log_auth_success(user_id, client_type, auth_source, headers);

    // device_fingerprint not in Auth message; use empty string
    Ok((user_id, String::new(), nickname))
}

async fn send_auth_error_and_close(sender: &mut SplitSink, err: ApiError) {
    let err_msg = ws_error_from_app_error(&err);
    if let Ok(json) = serde_json::to_string(&err_msg) {
        if let Err(send_err) = sender.send(Message::Text(json.into())).await {
            tracing::warn!(
                target = "ws_auth",
                error = %send_err,
                "failed to send auth error message before close"
            );
        }
    } else {
        tracing::warn!(
            target = "ws_auth",
            "failed to serialize auth error message before close"
        );
    }
    if let Err(close_err) = sender.send(Message::Close(None)).await {
        tracing::warn!(
            target = "ws_auth",
            error = %close_err,
            "failed to close websocket after auth error"
        );
    }
}
