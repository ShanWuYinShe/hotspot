use axum::{
    extract::{
        ws::{Message, WebSocket},
        State, WebSocketUpgrade,
    },
    response::Response,
};
use futures_util::{SinkExt, StreamExt};
use std::time::Duration;
use tokio::time::{interval, timeout};
use uuid::Uuid;

use crate::{
    repositories::{private_chat_repo::PrivateChatRepository, user_repo::UserRepository},
    state::AppState,
    utils::jwt::JwtUtil,
    websocket::subscription::SubscriptionType,
    websocket::types::{FailedSubscriptionItem, SubscriptionItem, WsMessage},
};

/// WebSocket升级处理器
pub async fn ws_handler(ws: WebSocketUpgrade, State(state): State<AppState>) -> Response {
    // 升级到WebSocket连接
    ws.on_upgrade(move |socket| handle_socket(socket, state))
}

/// 处理WebSocket连接
async fn handle_socket(socket: WebSocket, state: AppState) {
    // 分离发送和接收通道（鉴权阶段先不注册连接）
    let (mut sender, mut receiver) = socket.split();

    // 等待首帧 Auth（短超时，避免未鉴权连接占用资源）
    let auth_msg = timeout(Duration::from_secs(5), receiver.next()).await;
    let auth_msg = match auth_msg {
        Ok(Some(Ok(Message::Text(text)))) => text,
        Ok(Some(Ok(_))) => {
            // 非文本首帧 -> 直接断开
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
        Ok(Some(Err(_))) | Ok(None) | Err(_) => {
            // 超时/断开/错误 -> 直接断开
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let auth_event: WsMessage = match serde_json::from_str(auth_msg.as_str()) {
        Ok(m) => m,
        Err(_) => {
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let access_token = match auth_event {
        WsMessage::Auth { access_token, .. } => access_token,
        _ => {
            // 首帧不是 Auth
            let err_msg = WsMessage::Error {
                code: 1102,
                message: "Token 缺失".to_string(),
            };
            if let Ok(json) = serde_json::to_string(&err_msg) {
                let _ = sender.send(Message::Text(json.into())).await;
            }
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    // 验证 JWT
    let claims = match JwtUtil::verify_token(&access_token, &state.config.jwt_secret) {
        Ok(c) => c,
        Err(_) => {
            let err_msg = WsMessage::Error {
                code: 1101,
                message: "Token 无效或过期".to_string(),
            };
            if let Ok(json) = serde_json::to_string(&err_msg) {
                let _ = sender.send(Message::Text(json.into())).await;
            }
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let user_id = match claims.sub.parse::<i64>() {
        Ok(uid) => uid,
        Err(_) => {
            let err_msg = WsMessage::Error {
                code: 1101,
                message: "Token 无效或过期".to_string(),
            };
            if let Ok(json) = serde_json::to_string(&err_msg) {
                let _ = sender.send(Message::Text(json.into())).await;
            }
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let connection_id = Uuid::new_v4().to_string();

    tracing::debug!(
        "WebSocket连接建立: user_id={}, connection_id={}",
        user_id,
        connection_id
    );

    // 创建消息通道
    let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<Message>();

    // 将连接添加到管理器
    state
        .ws_manager
        .add_connection(connection_id.clone(), user_id, tx.clone())
        .await;

    // 标记用户上线（Redis + 数据库）
    use crate::repositories::user_online_status_repo::UserOnlineStatusRepository;

    // 1. 更新 Redis（实时状态）
    if let Err(e) = state.presence_service.mark_user_online(user_id).await {
        tracing::error!("标记用户上线失败(Redis): user_id={}, error={}", user_id, e);
    }

    // 2. 更新数据库（持久化状态）
    let online_status_repo = UserOnlineStatusRepository::new(state.db.clone());
    if let Err(e) = online_status_repo.set_online(user_id).await {
        tracing::error!("标记用户上线失败(DB): user_id={}, error={}", user_id, e);
    }

    // 获取用户信息
    let user_repo = crate::repositories::user_repo::UserRepository::new(state.db.clone());
    let (nickname, avatar_url) = match user_repo.find_by_id(user_id).await {
        Ok(Some(u)) => (u.nickname, u.avatar_url),
        _ => ("Unknown".to_string(), None),
    };

    // 鉴权通过后发送 ConnectionEstablished
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
        nickname: nickname.clone(),
        avatar_url: avatar_url.clone(),
        online_at: chrono::Utc::now().to_rfc3339(),
    };
    // 只推送给订阅了该用户状态的用户
    let subscribers = state
        .ws_subscription
        .get_user_status_subscribers(&user_id.to_string())
        .await;
    state
        .ws_manager
        .send_to_users(&subscribers, &online_msg)
        .await;

    // 启动心跳任务（使用常量配置的间隔）
    let heartbeat_tx = tx.clone();
    let heartbeat_task = tokio::spawn(async move {
        let mut heartbeat_interval = interval(Duration::from_secs(
            crate::constants::WS_SERVER_PING_INTERVAL_SECS,
        ));
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
        use crate::repositories::user_online_status_repo::UserOnlineStatusRepository;

        // 1. 更新 Redis（实时状态）
        if let Err(e) = state.presence_service.mark_user_offline(user_id).await {
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
        let subscribers = state
            .ws_subscription
            .get_user_status_subscribers(&user_id.to_string())
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

/// 处理WebSocket消息
async fn handle_message(
    msg: Message,
    user_id: i64,
    nickname: &str,
    connection_id: &str,
    state: &AppState,
) -> Result<(), String> {
    match msg {
        Message::Text(text) => {
            // 解析客户端消息
            let client_msg: WsMessage =
                serde_json::from_str(text.as_str()).map_err(|e| format!("解析消息失败: {}", e))?;

            match client_msg {
                WsMessage::Ping { .. } => {
                    // 更新心跳时间
                    state.ws_manager.update_ping(connection_id).await?;

                    // 回复Pong
                    let pong_msg = WsMessage::Pong {
                        timestamp: chrono::Utc::now().to_rfc3339(),
                    };
                    state.ws_manager.send_to_user(user_id, &pong_msg).await?;
                }

                WsMessage::Subscribe { items } => {
                    // 处理增量订阅
                    let (subscribed_items, failed_items) =
                        process_subscribe(user_id, &items, state).await;

                    // 发送订阅确认
                    let ack_msg = WsMessage::SubscribeAck {
                        operation: "subscribe".to_string(),
                        items: subscribed_items,
                        code: 0,
                        message: None,
                        failed_items: if failed_items.is_empty() {
                            None
                        } else {
                            Some(failed_items)
                        },
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
                    let (subscribed_items, failed_items) =
                        process_subscribe(user_id, &items, state).await;

                    // 发送确认
                    let ack_msg = WsMessage::SubscribeAck {
                        operation: "set".to_string(),
                        items: subscribed_items,
                        code: 0,
                        message: None,
                        failed_items: if failed_items.is_empty() {
                            None
                        } else {
                            Some(failed_items)
                        },
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }

                WsMessage::MarkAsRead {
                    chat_type,
                    chat_id,
                    up_to,
                } => {
                    // 处理标记已读请求
                    tracing::debug!(
                        "收到标记已读请求: user_id={}, chat_type={}, chat_id={}, up_to={}",
                        user_id,
                        chat_type,
                        chat_id,
                        up_to
                    );

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
                        // 私聊：chat_id 是 private_chat_id（UUID），需要查库确定对方 user_id
                        if let Ok(private_chat_id) = Uuid::parse_str(&chat_id) {
                            let chat_repo = PrivateChatRepository::new(state.db.clone());
                            if let Ok(Some(chat)) = chat_repo.find_by_id(private_chat_id).await {
                                let other_user_id = if chat.user_a_id == user_id {
                                    chat.user_b_id
                                } else if chat.user_b_id == user_id {
                                    chat.user_a_id
                                } else {
                                    // 非成员，不转发
                                    return Ok(());
                                };
                                let _ = state
                                    .ws_manager
                                    .send_to_user(other_user_id, &read_msg)
                                    .await;
                            }
                        }
                    } else if chat_type == "group_chat" {
                        // 群聊：推送给订阅了该群聊的所有用户
                        let subscribers = state
                            .ws_subscription
                            .get_group_chat_subscribers(&chat_id)
                            .await;
                        state
                            .ws_manager
                            .send_to_users(&subscribers, &read_msg)
                            .await;
                    }
                }

                WsMessage::Typing {
                    chat_type,
                    chat_id,
                    is_typing,
                } => {
                    // 处理正在输入事件
                    let typing_msg = WsMessage::UserTyping {
                        chat_type: chat_type.clone(),
                        chat_id: chat_id.clone(),
                        user_id,
                        nickname: nickname.to_string(),
                        is_typing,
                    };

                    // 推送给目标
                    if chat_type == "private_chat" {
                        // 私聊：chat_id 是 private_chat_id（UUID），需要查库确定对方 user_id
                        if let Ok(private_chat_id) = Uuid::parse_str(&chat_id) {
                            let chat_repo = PrivateChatRepository::new(state.db.clone());
                            if let Ok(Some(chat)) = chat_repo.find_by_id(private_chat_id).await {
                                let other_user_id = if chat.user_a_id == user_id {
                                    chat.user_b_id
                                } else if chat.user_b_id == user_id {
                                    chat.user_a_id
                                } else {
                                    // 非成员，不转发
                                    return Ok(());
                                };
                                let _ = state
                                    .ws_manager
                                    .send_to_user(other_user_id, &typing_msg)
                                    .await;
                            }
                        }
                    } else if chat_type == "group_chat" {
                        // 群聊：推送给订阅了该群聊的所有用户（排除自己）
                        let subscribers = state
                            .ws_subscription
                            .get_group_chat_subscribers(&chat_id)
                            .await;
                        for subscriber_id in subscribers {
                            if subscriber_id != user_id {
                                let _ = state
                                    .ws_manager
                                    .send_to_user(subscriber_id, &typing_msg)
                                    .await;
                            }
                        }
                    }
                }

                _ => {
                    // 客户端发了未识别的事件：更适合 debug（避免被噪音刷屏）
                    tracing::debug!("收到未处理的客户端消息类型: user_id={}", user_id);
                }
            }
        }

        Message::Binary(_) => {
            // 二进制消息通常是客户端误用/探测：这里降到 debug，避免刷屏
            tracing::debug!("收到二进制消息，暂不支持: user_id={}", user_id);
        }

        Message::Ping(_) => {
            // Axum会自动回复Pong
        }

        Message::Pong(_) => {
            // 心跳响应，更新最后活跃时间
            let _ = state.ws_manager.update_ping(connection_id).await;
        }

        Message::Close(_) => {
            tracing::debug!("收到关闭消息: user_id={}", user_id);
        }
    }

    Ok(())
}

/// 处理订阅请求
async fn process_subscribe(
    user_id: i64,
    items: &[SubscriptionItem],
    state: &AppState,
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

                    // 订阅用户状态后，立刻推送一次“当前在线状态”，避免前端只收到“变化事件”导致初始状态错误
                    if let Ok(target_user_id) = source_id.parse::<i64>() {
                        if state.ws_manager.get_user_connection_count(target_user_id).await > 0 {
                            let user_repo = UserRepository::new(state.db.clone());
                            if let Ok(Some(u)) = user_repo.find_by_id(target_user_id).await {
                                let msg = WsMessage::UserOnline {
                                    user_id: target_user_id,
                                    nickname: u.nickname,
                                    avatar_url: u.avatar_url,
                                    online_at: chrono::Utc::now().to_rfc3339(),
                                };
                                let _ = state.ws_manager.send_to_user(user_id, &msg).await;
                            }
                        }
                    }
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
    state: &AppState,
) -> Vec<SubscriptionItem> {
    let mut unsubscribed = Vec::new();

    for item in items {
        let mut success_ids = Vec::new();

        for source_id in &item.source_ids {
            match item.source_type.as_str() {
                "group_chat" => {
                    let subscription = SubscriptionType::GroupChat(source_id.clone());
                    state
                        .ws_subscription
                        .unsubscribe(user_id, &subscription)
                        .await;
                    success_ids.push(source_id.clone());
                }
                "user" => {
                    let subscription = SubscriptionType::User(source_id.clone());
                    state
                        .ws_subscription
                        .unsubscribe(user_id, &subscription)
                        .await;
                    success_ids.push(source_id.clone());
                }
                "notification" => {
                    let subscription = SubscriptionType::Notification(source_id.clone());
                    state
                        .ws_subscription
                        .unsubscribe(user_id, &subscription)
                        .await;
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
