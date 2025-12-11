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
    websocket::subscription::SubscriptionType,
    websocket::types::{WsMessage, SubscriptionItem, FailedSubscriptionItem},
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
    let claims = JwtUtil::verify_token(&query.token, &state.config.jwt_secret)
        .map_err(|_| AppError::Unauthorized("无效的Token".to_string()))?;

    let user_id = claims.sub.parse::<i64>().map_err(|_| AppError::Unauthorized("无效的Token: user_id不是数字".to_string()))?;

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

    // 标记用户上线（Redis + 数据库）
    use crate::services::presence_service::PresenceService;
    use crate::repositories::user_online_status_repo::UserOnlineStatusRepository;
    
    // 1. 更新 Redis（实时状态）
    let mut presence_service = PresenceService::new(state.redis.clone());
    if let Err(e) = presence_service.mark_user_online(user_id).await {
        tracing::error!("标记用户上线失败(Redis): user_id={}, error={}", user_id, e);
    }
    
    // 2. 更新数据库（持久化状态）
    let online_status_repo = UserOnlineStatusRepository::new(state.db.clone());
    if let Err(e) = online_status_repo.set_online(user_id).await {
        tracing::error!("标记用户上线失败(DB): user_id={}, error={}", user_id, e);
    }

    // 发送欢迎消息
    let welcome_msg = WsMessage::ConnectionEstablished {
        connection_id: connection_id.clone(),
        user_id,
        server_time: chrono::Utc::now().to_rfc3339(),
    };
    if let Ok(json) = serde_json::to_string(&welcome_msg) {
        let _ = tx.send(Message::Text(json.into()));
    }

    // 通知订阅了该用户状态的其他用户
    let online_msg = WsMessage::UserOnline {
        user_id,
        nickname: "Unknown".to_string(), // TODO: 查询用户昵称
        avatar_url: None,
        online_at: chrono::Utc::now().to_rfc3339(),
    };
    // 只推送给订阅了该用户状态的用户
    let subscribers = state.ws_subscription.get_user_status_subscribers(&user_id.to_string()).await;
    state.ws_manager.send_to_users(&subscribers, &online_msg).await;

    // 启动心跳任务（使用常量配置的间隔）
    let heartbeat_tx = tx.clone();
    let heartbeat_task = tokio::spawn(async move {
        let mut heartbeat_interval = interval(Duration::from_secs(crate::constants::WS_SERVER_PING_INTERVAL_SECS));
        loop {
            heartbeat_interval.tick().await;
            if heartbeat_tx.send(Message::Ping(vec![].into())).is_err() {
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
    let connection_id_for_handle = connection_id.clone();
    let connection_id_for_return = connection_id.clone();
    let receive_task = tokio::spawn(async move {
        while let Some(result) = receiver.next().await {
            match result {
                Ok(msg) => {
                    if let Err(e) = handle_message(msg, user_id, &connection_id_for_handle, &state_clone).await {
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

    // 等待任务完成
    let connection_id = receive_task.await.unwrap_or(connection_id);

    // 清理资源
    heartbeat_task.abort();
    send_task.abort();

    // 移除连接
    state.ws_manager.remove_connection(&connection_id).await;

    // 取消用户所有订阅
    state.ws_subscription.unsubscribe_all(user_id).await;

    // 标记用户离线（Redis + 数据库，仅当用户没有其他连接时）
    if state.ws_manager.get_user_connection_count(user_id).await == 0 {
        use crate::services::presence_service::PresenceService;
        use crate::repositories::user_online_status_repo::UserOnlineStatusRepository;
        
        // 1. 更新 Redis（实时状态）
        let mut presence_service = PresenceService::new(state.redis.clone());
        if let Err(e) = presence_service.mark_user_offline(user_id).await {
            tracing::error!("标记用户离线失败(Redis): user_id={}, error={}", user_id, e);
        }
        
        // 2. 更新数据库（持久化状态）
        let online_status_repo = UserOnlineStatusRepository::new(state.db.clone());
        if let Err(e) = online_status_repo.set_offline(user_id).await {
            tracing::error!("标记用户离线失败(DB): user_id={}, error={}", user_id, e);
        }

        // 通知订阅了该用户状态的其他用户
        let offline_msg = WsMessage::UserOffline {
            user_id,
            offline_at: chrono::Utc::now().to_rfc3339(),
        };
        let subscribers = state.ws_subscription.get_user_status_subscribers(&user_id.to_string()).await;
        state.ws_manager.send_to_users(&subscribers, &offline_msg).await;
    }

    tracing::info!("WebSocket连接关闭: user_id={}, connection_id={}", user_id, connection_id);
}

/// 处理WebSocket消息
async fn handle_message(msg: Message, user_id: i64, connection_id: &str, state: &AppState) -> Result<(), String> {
    match msg {
        Message::Text(text) => {
            // 解析客户端消息
            let client_msg: WsMessage = serde_json::from_str(text.as_str())
                .map_err(|e| format!("解析消息失败: {}", e))?;

            match client_msg {
                WsMessage::Ping { .. } => {
                    // 更新心跳时间
                    state.ws_manager.update_ping(connection_id).await?;
                    
                    // 回复Pong
                    let pong_msg = WsMessage::Pong { timestamp: chrono::Utc::now().to_rfc3339() };
                    state.ws_manager.send_to_user(user_id, &pong_msg).await?;
                }
                
                WsMessage::Subscribe { items } => {
                    // 处理增量订阅
                    let (subscribed_items, failed_items) = process_subscribe(user_id, &items, state).await;
                    
                    // 发送订阅确认
                    let ack_msg = WsMessage::SubscribeAck { 
                        operation: "subscribe".to_string(),
                        items: subscribed_items,
                        code: 0,
                        message: None,
                        failed_items: if failed_items.is_empty() { None } else { Some(failed_items) },
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }
                
                WsMessage::Unsubscribe { items } => {
                    // 处理取消订阅
                    let unsubscribed_items = process_unsubscribe(user_id, &items, state).await;
                    
                    // 发送取消订阅确认
                    let ack_msg = WsMessage::SubscribeAck { 
                        operation: "unsubscribe".to_string(),
                        items: unsubscribed_items,
                        code: 0,
                        message: None,
                        failed_items: None,
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }
                
                WsMessage::SetSubscriptions { items } => {
                    // 先清空所有订阅
                    state.ws_subscription.unsubscribe_all(user_id).await;
                    
                    // 然后添加新订阅
                    let (subscribed_items, failed_items) = process_subscribe(user_id, &items, state).await;
                    
                    // 发送确认
                    let ack_msg = WsMessage::SubscribeAck { 
                        operation: "set".to_string(),
                        items: subscribed_items,
                        code: 0,
                        message: None,
                        failed_items: if failed_items.is_empty() { None } else { Some(failed_items) },
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }
                
                WsMessage::MarkAsRead { chat_type, chat_id, up_to } => {
                    // 处理标记已读请求
                    tracing::debug!("收到标记已读请求: user_id={}, chat_type={}, chat_id={}, up_to={}", 
                        user_id, chat_type, chat_id, up_to);
                    
                    // 广播已读回执给相关用户
                    let read_msg = WsMessage::ChatMessageRead {
                        chat_type: chat_type.clone(),
                        chat_id: chat_id.clone(),
                        up_to,
                        reader_id: user_id,
                        read_at: chrono::Utc::now().to_rfc3339(),
                    };
                    
                    // 根据类型推送
                    if chat_type == "private_chat" {
                        // 私聊：推送给对方
                        if let Ok(other_user_id) = chat_id.parse::<i64>() {
                            let _ = state.ws_manager.send_to_user(other_user_id, &read_msg).await;
                        }
                    } else if chat_type == "group_chat" {
                        // 群聊：推送给订阅了该群聊的所有用户
                        let subscribers = state.ws_subscription.get_group_chat_subscribers(&chat_id).await;
                        state.ws_manager.send_to_users(&subscribers, &read_msg).await;
                    }
                }
                
                WsMessage::Typing { chat_type, chat_id, is_typing } => {
                    // 处理正在输入事件
                    let typing_msg = WsMessage::UserTyping {
                        chat_type: chat_type.clone(),
                        chat_id: chat_id.clone(),
                        user_id,
                        nickname: "Unknown".to_string(), // TODO: 查询用户昵称
                        is_typing,
                    };
                    
                    // 推送给目标
                    if chat_type == "private_chat" {
                        // 私聊：推送给对方
                        if let Ok(target_user_id) = chat_id.parse::<i64>() {
                            let _ = state.ws_manager.send_to_user(target_user_id, &typing_msg).await;
                        }
                    } else if chat_type == "group_chat" {
                        // 群聊：推送给订阅了该群聊的所有用户（排除自己）
                        let subscribers = state.ws_subscription.get_group_chat_subscribers(&chat_id).await;
                        for subscriber_id in subscribers {
                            if subscriber_id != user_id {
                                let _ = state.ws_manager.send_to_user(subscriber_id, &typing_msg).await;
                            }
                        }
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
            // 心跳响应，更新最后活跃时间
            let _ = state.ws_manager.update_ping(connection_id).await;
        }
        
        Message::Close(_) => {
            tracing::info!("收到关闭消息: user_id={}", user_id);
        }
    }

    Ok(())
}

/// 处理订阅请求
async fn process_subscribe(
    user_id: i64, 
    items: &[SubscriptionItem], 
    state: &AppState
) -> (Vec<SubscriptionItem>, Vec<FailedSubscriptionItem>) {
    let mut subscribed = Vec::new();
    let mut failed = Vec::new();
    
    for item in items {
        let mut success_ids = Vec::new();
        let mut fail_ids = Vec::new();
        
        for source_id in &item.source_ids {
            match item.source_type.as_str() {
                "group_chat" => {
                    let subscription = SubscriptionType::GroupChat(source_id.clone());
                    state.ws_subscription.subscribe(user_id, subscription).await;
                    success_ids.push(source_id.clone());
                }
                "user" => {
                    let subscription = SubscriptionType::User(source_id.clone());
                    state.ws_subscription.subscribe(user_id, subscription).await;
                    success_ids.push(source_id.clone());
                }
                "notification" => {
                    let subscription = SubscriptionType::Notification(source_id.clone());
                    state.ws_subscription.subscribe(user_id, subscription).await;
                    success_ids.push(source_id.clone());
                }
                _ => {
                    fail_ids.push(source_id.clone());
                }
            }
        }
        
        if !success_ids.is_empty() {
            subscribed.push(SubscriptionItem {
                source_type: item.source_type.clone(),
                source_ids: success_ids,
            });
        }
        
        if !fail_ids.is_empty() {
            failed.push(FailedSubscriptionItem {
                source_type: item.source_type.clone(),
                source_ids: fail_ids,
                reason: "不支持的订阅类型".to_string(),
            });
        }
    }
    
    (subscribed, failed)
}

/// 处理取消订阅请求
async fn process_unsubscribe(
    user_id: i64, 
    items: &[SubscriptionItem], 
    state: &AppState
) -> Vec<SubscriptionItem> {
    let mut unsubscribed = Vec::new();
    
    for item in items {
        let mut success_ids = Vec::new();
        
        for source_id in &item.source_ids {
            match item.source_type.as_str() {
                "group_chat" => {
                    let subscription = SubscriptionType::GroupChat(source_id.clone());
                    state.ws_subscription.unsubscribe(user_id, &subscription).await;
                    success_ids.push(source_id.clone());
                }
                "user" => {
                    let subscription = SubscriptionType::User(source_id.clone());
                    state.ws_subscription.unsubscribe(user_id, &subscription).await;
                    success_ids.push(source_id.clone());
                }
                "notification" => {
                    let subscription = SubscriptionType::Notification(source_id.clone());
                    state.ws_subscription.unsubscribe(user_id, &subscription).await;
                    success_ids.push(source_id.clone());
                }
                _ => {
                    // 忽略不支持的类型
                }
            }
        }
        
        if !success_ids.is_empty() {
            unsubscribed.push(SubscriptionItem {
                source_type: item.source_type.clone(),
                source_ids: success_ids,
            });
        }
    }
    
    unsubscribed
}
