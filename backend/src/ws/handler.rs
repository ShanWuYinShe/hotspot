use crate::{
    middleware::auth::AppState, 
    ws::message_types::{ClientMessage, ServerMessage},
    ws::errors::{WsError, WsResult},
    ws::manager::WebSocketManager,
    utils::jwt::verify_jwt_token,
};
use axum::{
    extract::{
        State, WebSocketUpgrade,
        ws::{Message, WebSocket},
    },
    response::Response,
};
use futures::{sink::SinkExt, stream::StreamExt};
use std::sync::Arc;
use tracing::{error, info, debug};
use serde_json;
use uuid::Uuid;

// WebSocket连接处理函数
pub async fn ws_handler(ws: WebSocketUpgrade, State(state): State<Arc<AppState>>) -> Response {
    ws.on_upgrade(|socket| handle_socket(socket, state))
}

// 处理WebSocket连接
async fn handle_socket(socket: WebSocket, state: Arc<AppState>) {
    let (mut sender, mut receiver) = socket.split();
    let connection_id = Uuid::new_v4();
    
    info!("新的WebSocket连接: {}", connection_id);

    // 等待认证消息
    let auth_message = match receiver.next().await {
        Some(Ok(Message::Text(text))) => text,
        Some(Ok(_)) => {
            let error_msg = ServerMessage::Error { 
                code: "INVALID_MESSAGE_TYPE".to_string(),
                message: "需要文本消息进行认证".to_string() 
            };
            let _ = send_message(&mut sender, &error_msg).await;
            return;
        }
        Some(Err(e)) => {
            error!("接收消息错误: {:?}", e);
            return;
        }
        None => {
            info!("连接关闭，未收到认证消息");
            return;
        }
    };

    // 解析认证消息
    let user_id = match parse_auth_message(&auth_message, &state).await {
        Ok(id) => id,
        Err(err_msg) => {
            let _ = send_message(&mut sender, &err_msg).await;
            return;
        }
    };

    info!("用户 {} 认证成功，连接ID: {}", user_id, connection_id);

    // 创建消息通道用于接收要发送给客户端的消息
    let (msg_tx, msg_rx) = tokio::sync::mpsc::unbounded_channel::<ServerMessage>();

    // 注册连接到WebSocket管理器
    let ws_manager = &state.ws_manager;
    if let Err(e) = ws_manager.add_connection(connection_id, user_id, msg_tx).await {
        error!("添加连接失败: {:?}", e);
        return;
    }

    // 发送连接成功消息
    let connected_msg = ServerMessage::Connected { 
        user_id, 
        session_id: connection_id.to_string()
    };
    if let Err(e) = send_message(&mut sender, &connected_msg).await {
        error!("发送连接成功消息失败: {:?}", e);
        if let Err(e) = ws_manager.remove_connection(connection_id).await {
            error!("移除连接失败: {:?}", e);
        }
        return;
    }

    // 开始处理消息循环
    handle_message_loop(sender, receiver, connection_id, user_id, ws_manager, msg_rx).await;
    
    // 清理连接
    if let Err(e) = ws_manager.remove_connection(connection_id).await {
        error!("清理连接失败: {:?}", e);
    }
    info!("用户 {} 断开连接", user_id);
}

// 解析认证消息
async fn parse_auth_message(
    message: &str, 
    state: &AppState
) -> Result<i64, ServerMessage> {
    let client_msg: ClientMessage = serde_json::from_str(message)
        .map_err(|_| ServerMessage::Error { 
            code: "INVALID_JSON".to_string(),
            message: "无效的JSON格式".to_string() 
        })?;

    match client_msg {
        ClientMessage::Connect { token } => {
            let config = state.config_service.get_config();
            let claims = verify_jwt_token(&token, &config.auth.jwt_secret)
                .map_err(|_| ServerMessage::Error { 
                    code: "INVALID_TOKEN".to_string(),
                    message: "无效的认证令牌".to_string() 
                })?;
            Ok(claims.sub)
        }
        _ => Err(ServerMessage::Error { 
            code: "INVALID_AUTH_MESSAGE".to_string(),
            message: "首条消息必须是连接认证消息".to_string() 
        })
    }
}

// 发送消息到WebSocket
async fn send_message(
    sender: &mut futures::stream::SplitSink<WebSocket, Message>,
    message: &ServerMessage
) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let json_str = serde_json::to_string(message)?;
    sender.send(Message::Text(json_str.into())).await?;
    Ok(())
}

// 消息处理循环
async fn handle_message_loop(
    mut sender: futures::stream::SplitSink<WebSocket, Message>,
    mut receiver: futures::stream::SplitStream<WebSocket>,
    connection_id: Uuid,
    _user_id: i64,
    ws_manager: &WebSocketManager,
    mut message_rx: tokio::sync::mpsc::UnboundedReceiver<ServerMessage>,
) {
    loop {
        tokio::select! {
            // 处理来自客户端的消息
            client_msg = receiver.next() => {
                match client_msg {
                    Some(Ok(Message::Text(text))) => {
                        debug!("收到客户端消息: {}", text);
                        if let Err(e) = handle_client_message(&text, connection_id, ws_manager).await {
                            error!("处理客户端消息失败: {:?}", e);
                        }
                    }
                    Some(Ok(Message::Close(_))) => {
                        info!("客户端主动关闭连接");
                        break;
                    }
                    Some(Err(e)) => {
                        error!("接收消息错误: {:?}", e);
                        break;
                    }
                    None => {
                        info!("客户端连接已关闭");
                        break;
                    }
                    _ => {}
                }
            }
            // 处理来自服务器的消息
            server_msg = message_rx.recv() => {
                match server_msg {
                    Some(message) => {
                        if let Err(e) = send_message(&mut sender, &message).await {
                            error!("发送服务器消息失败: {:?}", e);
                            break;
                        }
                    }
                    None => {
                        info!("消息通道已关闭");
                        break;
                    }
                }
            }
        }
    }
}

// 处理客户端消息
async fn handle_client_message(
    message: &str,
    connection_id: Uuid,
    ws_manager: &WebSocketManager,
) -> WsResult<()> {
    let client_msg: ClientMessage = serde_json::from_str(message)
        .map_err(|_| WsError::invalid_message("无效的JSON格式"))?;

    debug!("处理客户端消息: {:?}", client_msg);

    // 将消息转发给WebSocket管理器处理
    ws_manager.handle_client_message(connection_id, client_msg).await?;

    Ok(())
}
