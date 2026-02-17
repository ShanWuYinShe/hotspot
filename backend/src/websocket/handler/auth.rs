//! WebSocket authentication flow.

use axum::extract::ws::{Message, WebSocket};
use axum::http::HeaderMap;
use futures_util::{SinkExt, StreamExt};
use std::time::Duration;
use tokio::time::timeout;

use crate::{
    error::AppError,
    repositories::user_repo::UserRepository,
    state::AppState,
    types::enums::ClientType,
    utils::{jwt::JwtUtil, web_cookie::get_cookie_value},
    websocket::types::{ws_error_from_app_error, WsMessage},
};

type SplitSink = futures_util::stream::SplitSink<WebSocket, Message>;
type SplitStream = futures_util::stream::SplitStream<WebSocket>;

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
            send_auth_error_and_close(
                sender,
                AppError::WebSocketProtocolIncompatible("仅支持文本协议".to_string()),
            )
            .await;
            return Err(());
        }
        Ok(Some(Err(_))) | Ok(None) | Err(_) => {
            // spec: 鉴权失败或超时，发送 Error(code=20000) 后断开连接
            send_auth_error_and_close(
                sender,
                AppError::WebSocketError("鉴权超时或连接异常".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let auth_event: WsMessage = match serde_json::from_str(auth_msg.as_str()) {
        Ok(m) => m,
        Err(_) => {
            send_auth_error_and_close(
                sender,
                AppError::WebSocketMessageFormatError("鉴权消息格式无效".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let access_token = match auth_event {
        WsMessage::Auth {
            client_type,
            access_token,
            ..
        } => access_token.or_else(|| {
            let is_web = client_type.unwrap_or(ClientType::App) == ClientType::Web;
            if is_web {
                get_cookie_value(headers, "access_token")
            } else {
                None
            }
        }),
        _ => {
            send_auth_error_and_close(
                sender,
                AppError::WebSocketAuthFailed("鉴权失败：未提供 Token".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let access_token = match access_token {
        Some(t) if !t.is_empty() => t,
        _ => {
            send_auth_error_and_close(
                sender,
                AppError::WebSocketAuthFailed("鉴权失败：未提供 Token".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let claims = match JwtUtil::verify_token(&access_token, &state.config.jwt_secret) {
        Ok(c) => c,
        Err(_) => {
            send_auth_error_and_close(
                sender,
                AppError::WebSocketAuthFailed("鉴权失败：Token 无效或过期".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let user_id = match claims.sub.parse::<i64>() {
        Ok(uid) => uid,
        Err(_) => {
            send_auth_error_and_close(
                sender,
                AppError::WebSocketAuthFailed("鉴权失败：Token 无效或过期".to_string()),
            )
            .await;
            return Err(());
        }
    };

    let user_repo = UserRepository::new(state.db.clone());
    let nickname = match user_repo.find_by_id(user_id).await {
        Ok(Some(u)) => u.nickname.unwrap_or_else(|| "Unknown".to_string()),
        _ => "Unknown".to_string(),
    };

    // device_fingerprint not in Auth message; use empty string
    Ok((user_id, String::new(), nickname))
}

async fn send_auth_error_and_close(sender: &mut SplitSink, err: AppError) {
    let err_msg = ws_error_from_app_error(&err);
    if let Ok(json) = serde_json::to_string(&err_msg) {
        let _ = sender.send(Message::Text(json.into())).await;
    }
    let _ = sender.send(Message::Close(None)).await;
}
