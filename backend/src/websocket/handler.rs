use axum::{
    extract::{
        ws::{Message, WebSocket},
        Query, State, WebSocketUpgrade,
    },
    response::Response,
};
use futures_util::{SinkExt, StreamExt};
use serde::Deserialize;
use std::time::Duration;
use tokio::time::interval;
use uuid::Uuid;

use crate::{
    error::AppError,
    state::AppState,
    utils::jwt::JwtUtil,
    websocket::{
        subscription::SubscriptionType,
        types::WsMessage,
    },
};

#[derive(Debug, Deserialize)]
pub struct WsQuery {
    token: String,
}

/// WebSocket升级处理器
pub async fn ws_handler(
    ws: WebSocketUpgrade,
    Query(query): Query<WsQuery>,
    State(state): State<AppState>,
) -> Result<Response, AppError> {
    // 验证JWT Token
    let claims = JwtUtil::verify_access_token(&query.token, &state.config.jwt_secret)
        .map_err(|_| AppError::Unauthorized("无效的Token".to_string()))?;

    let user_id = claims.user_id;

    // 升级到WebSocket连接
    Ok(ws.on_upgrade(move |socket| handle_socket(socket, user_id, state)))
}

/// 处理WebSocket连接
async fn handle_socket(socket: WebSocket, user_id: i64, state: AppState) {
    let connection_id = Uuid::new_v4().to_string();
    
    tracing::info!("WebSocket连接建立: user_id={}, connection_id={}", user_id, connection_id);

    // 分离发送和接收通道
    let (mut sender, mut receiver) = socket.split();

    // 创建消息通道
    let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<Message>();

    // 将连接添加到管理器
    state.ws_manager.add_connection(connection_id.clone(), user_id, tx.clone()).await;

    // 发送欢迎消息
    let welcome_msg = WsMessage::Connected {
        connection_id: connection_id.clone(),
        user_id: user_id.to_string(),
    };
    if let Ok(json) = serde_json::to_string(&welcome_msg) {
        let _ = tx.send(Message::Text(json));
    }

    // 通知其他用户该用户上线
    let online_msg = WsMessage::UserOnline {
        user_id: user_id.to_string(),
        timestamp: chrono::Utc::now().timestamp(),
    };
    // TODO: 只推送给订阅了该用户在线状态的用户
    let _ = state.ws_manager.broadcast(&online_msg).await;

    // 启动心跳任务
    let heartbeat_tx = tx.clone();
    let heartbeat_task = tokio::spawn(async move {
        let mut heartbeat_interval = interval(Duration::from_secs(30));
        loop {
            heartbeat_interval.tick().await;
            if heartbeat_tx.send(Message::Ping(vec![])).is_err() {
                break;
            }
        }
    });

    // 启动发送任务
    let send_task = tokio::spawn(async move {
        while let Some(msg) = rx.recv().await {
            if sender.send(msg).await.is_err() {
                break;
            }
        }
    });

    // 处理接收消息
    let state_clone = state.clone();
    let connection_id_clone = connection_id.clone();
    let receive_task = tokio::spawn(async move {
        while let Some(result) = receiver.next().await {
            match result {
                Ok(msg) => {
                    if let Err(e) = handle_message(msg, user_id, &state_clone).await {
                        tracing::error!("处理WebSocket消息失败: user_id={}, error={}", user_id, e);
                    }
                }
                Err(e) => {
                    tracing::error!("WebSocket接收错误: user_id={}, error={}", user_id, e);
                    break;
                }
            }
        }
        connection_id_clone
    });

    // 等待任务完成
    let connection_id = receive_task.await.unwrap_or(connection_id);

    // 清理资源
    heartbeat_task.abort();
    send_task.abort();

    // 移除连接
    state.ws_manager.remove_connection(&connection_id).await;

    // 取消所有订阅
    state.ws_subscription.unsubscribe_all(user_id).await;

    // 通知其他用户该用户下线
    let offline_msg = WsMessage::UserOffline {
        user_id: user_id.to_string(),
        timestamp: chrono::Utc::now().timestamp(),
    };
    let _ = state.ws_manager.broadcast(&offline_msg).await;

    tracing::info!("WebSocket连接关闭: user_id={}, connection_id={}", user_id, connection_id);
}

/// 处理WebSocket消息
async fn handle_message(msg: Message, user_id: i64, state: &AppState) -> Result<(), String> {
    match msg {
        Message::Text(text) => {
            // 解析客户端消息
            let client_msg: WsMessage = serde_json::from_str(&text)
                .map_err(|e| format!("解析消息失败: {}", e))?;

            match client_msg {
                WsMessage::Ping => {
                    // 回复Pong
                    let pong_msg = WsMessage::Pong;
                    state.ws_manager.send_to_user(user_id, &pong_msg).await?;
                }
                
                WsMessage::Subscribe { channels } => {
                    // 处理订阅请求
                    for channel in channels {
                        let subscription = parse_channel(&channel)?;
                        state.ws_subscription.subscribe(user_id, subscription).await;
                    }
                    
                    // 发送订阅确认
                    let ack_msg = WsMessage::SubscribeAck { channels };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }
                
                WsMessage::Unsubscribe { channels } => {
                    // 处理取消订阅请求
                    for channel in channels {
                        let subscription = parse_channel(&channel)?;
                        state.ws_subscription.unsubscribe(user_id, &subscription).await;
                    }
                    
                    // 发送取消订阅确认
                    let ack_msg = WsMessage::UnsubscribeAck { channels };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }
                
                WsMessage::MarkAsRead { message_ids } => {
                    // 处理标记已读请求
                    // TODO: 调用消息服务标记已读
                    tracing::debug!("收到标记已读请求: user_id={}, message_ids={:?}", user_id, message_ids);
                }
                
                WsMessage::Typing { target_id, target_type } => {
                    // 处理正在输入事件
                    let typing_msg = WsMessage::Typing {
                        target_id: target_id.clone(),
                        target_type: target_type.clone(),
                    };
                    
                    // 推送给目标用户
                    if target_type == "user" {
                        if let Ok(target_user_id) = target_id.parse::<i64>() {
                            let _ = state.ws_manager.send_to_user(target_user_id, &typing_msg).await;
                        }
                    } else if target_type == "group" {
                        // 推送给群组所有成员
                        let subscribers = state.ws_subscription
                            .get_group_message_subscribers(&target_id)
                            .await;
                        state.ws_manager.send_to_users(&subscribers, &typing_msg).await;
                    }
                }
                
                _ => {
                    tracing::warn!("收到未处理的客户端消息类型: user_id={}", user_id);
                }
            }
        }
        
        Message::Binary(_) => {
            tracing::warn!("收到二进制消息，暂不支持: user_id={}", user_id);
        }
        
        Message::Ping(_) => {
            // Axum会自动回复Pong
        }
        
        Message::Pong(_) => {
            // 心跳响应
        }
        
        Message::Close(_) => {
            tracing::info!("收到关闭消息: user_id={}", user_id);
        }
    }

    Ok(())
}

/// 解析频道字符串为订阅类型
fn parse_channel(channel: &str) -> Result<SubscriptionType, String> {
    let parts: Vec<&str> = channel.split(':').collect();
    
    match parts.as_slice() {
        ["user_message", user_id] => {
            let id = user_id.parse::<i64>()
                .map_err(|_| format!("无效的用户ID: {}", user_id))?;
            Ok(SubscriptionType::UserMessage(id))
        }
        ["group_message", group_id] => {
            Ok(SubscriptionType::GroupMessage(group_id.to_string()))
        }
        ["inbox"] => {
            Ok(SubscriptionType::Inbox)
        }
        ["user_online", user_id] => {
            let id = user_id.parse::<i64>()
                .map_err(|_| format!("无效的用户ID: {}", user_id))?;
            Ok(SubscriptionType::UserOnlineStatus(id))
        }
        ["nearby_users"] => {
            Ok(SubscriptionType::NearbyUsers)
        }
        ["group_events", group_id] => {
            Ok(SubscriptionType::GroupEvents(group_id.to_string()))
        }
        ["moment_updates"] => {
            Ok(SubscriptionType::MomentUpdates)
        }
        _ => Err(format!("未知的频道类型: {}", channel)),
    }
}
