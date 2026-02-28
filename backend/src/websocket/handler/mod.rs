//! WebSocket connection and message handling.

mod auth;
mod message;
mod subscribe;

use axum::{
    extract::{
        ws::{Message, WebSocket},
        State, WebSocketUpgrade,
    },
    http::HeaderMap,
    response::Response,
};
use futures_util::{SinkExt, StreamExt};
use tokio::time::interval;
use uuid::Uuid;

use crate::{state::AppState, websocket::types::WsMessage};

use auth::authenticate_ws_connection;
use message::handle_message;

/// WebSocket升级处理器
pub async fn ws_handler(
    ws: WebSocketUpgrade,
    State(state): State<AppState>,
    headers: HeaderMap,
) -> Response {
    ws.on_upgrade(move |socket| handle_socket(socket, state, headers))
}

/// 处理WebSocket连接
async fn handle_socket(socket: WebSocket, state: AppState, headers: HeaderMap) {
    log_handshake_request(&headers);
    let (mut sender, mut receiver) = socket.split();

    let (user_id, _device_fingerprint, nickname) =
        match authenticate_ws_connection(&mut sender, &mut receiver, &state, &headers).await {
            Ok(triple) => triple,
            Err(()) => return,
        };

    let connection_id = Uuid::new_v4().to_string();

    tracing::debug!(
        "WebSocket连接建立: user_id={}, connection_id={}",
        user_id,
        connection_id
    );

    let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<Message>();

    state
        .ws_manager
        .add_connection(connection_id.clone(), user_id, tx.clone())
        .await;

    use crate::repositories::user::online_status_repo::UserOnlineStatusRepository;

    if let Err(e) = state.presence_service.mark_user_online(user_id).await {
        tracing::error!("标记用户上线失败(Redis): user_id={}, error={}", user_id, e);
    }

    let online_status_repo = UserOnlineStatusRepository::new(state.db.clone());
    if let Err(e) = online_status_repo.set_online(user_id).await {
        tracing::error!("标记用户上线失败(DB): user_id={}, error={}", user_id, e);
    }

    let welcome_msg = WsMessage::ConnectionEstablished {
        connection_id: connection_id.clone(),
        user_id,
        server_time: chrono::Utc::now().to_rfc3339(),
        config: crate::websocket::types::ConnectionConfig {
            ping_interval: 30,
            ping_timeout: 10,
            max_subscriptions: crate::websocket::types::MaxSubscriptions {
                user: 500,
                group: 100,
            },
        },
    };
    if let Ok(json) = serde_json::to_string(&welcome_msg) {
        let _ = tx.send(Message::Text(json.into()));
    }

    let online_msg = WsMessage::UserOnline {
        user_id,
        online_at: chrono::Utc::now().to_rfc3339(),
    };
    let subscribers = state
        .ws_subscription
        .get_user_status_subscribers(user_id)
        .await;
    state
        .ws_manager
        .send_to_users(&subscribers, &online_msg)
        .await;

    let heartbeat_tx = tx.clone();
    let heartbeat_task = tokio::spawn(async move {
        let mut heartbeat_interval = interval(std::time::Duration::from_secs(
            crate::constants::WS_SERVER_PING_INTERVAL_SECS,
        ));
        loop {
            heartbeat_interval.tick().await;
            if heartbeat_tx.send(Message::Ping(vec![].into())).is_err() {
                break;
            }
        }
    });

    let send_task = tokio::spawn(async move {
        while let Some(msg) = rx.recv().await {
            if sender.send(msg).await.is_err() {
                break;
            }
        }
    });

    let state_clone = state.clone();
    let connection_id_for_handle = connection_id.clone();
    let connection_id_for_return = connection_id.clone();
    let nickname_clone = nickname.clone();
    let receive_task = tokio::spawn(async move {
        while let Some(result) = receiver.next().await {
            match result {
                Ok(msg) => {
                    if let Err(e) = handle_message(
                        msg,
                        user_id,
                        &nickname_clone,
                        &connection_id_for_handle,
                        &state_clone,
                    )
                    .await
                    {
                        tracing::error!("处理WebSocket消息失败: user_id={}, error={}", user_id, e);
                    }
                }
                Err(e) => {
                    tracing::error!("WebSocket接收错误: user_id={}, error={}", user_id, e);
                    break;
                }
            }
        }
        connection_id_for_return
    });

    let connection_id = receive_task.await.unwrap_or(connection_id);

    heartbeat_task.abort();
    send_task.abort();

    state.ws_manager.remove_connection(&connection_id).await;

    state.ws_subscription.unsubscribe_all(user_id).await;

    if state.ws_manager.get_user_connection_count(user_id).await == 0 {
        use crate::repositories::user::online_status_repo::UserOnlineStatusRepository;

        if let Err(e) = state.presence_service.mark_user_offline(user_id).await {
            tracing::error!("标记用户离线失败(Redis): user_id={}, error={}", user_id, e);
        }

        let online_status_repo = UserOnlineStatusRepository::new(state.db.clone());
        if let Err(e) = online_status_repo.set_offline(user_id).await {
            tracing::error!("标记用户离线失败(DB): user_id={}, error={}", user_id, e);
        }

        let offline_msg = WsMessage::UserOffline {
            user_id,
            offline_at: chrono::Utc::now().to_rfc3339(),
        };
        let subscribers = state
            .ws_subscription
            .get_user_status_subscribers(user_id)
            .await;
        state
            .ws_manager
            .send_to_users(&subscribers, &offline_msg)
            .await;
    }

    tracing::debug!(
        "WebSocket连接关闭: user_id={}, connection_id={}",
        user_id,
        connection_id
    );
}
fn log_handshake_request(headers: &HeaderMap) {
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
    tracing::info!(
        target = "ws_handshake",
        origin = %origin,
        user_agent = %user_agent,
        "WebSocket handshake requested"
    );
}
