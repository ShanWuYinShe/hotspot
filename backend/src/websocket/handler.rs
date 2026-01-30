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
    error::AppError,
    repositories::{
        group_member_repo::GroupMemberRepository, group_repo::GroupRepository,
        user_blocks_repo::UserBlocksRepository, user_repo::UserRepository,
    },
    state::AppState,
    types::enums::{SubscriptionFailureReason, SubscriptionOperation},
    utils::{cache::RateLimiter, jwt::JwtUtil},
    websocket::subscription::SubscriptionType,
    websocket::types::{FailedSubscriptionId, SubscriptionSourceId, WsMessage},
};

const MAX_SUBSCRIPTION_SOURCE_IDS: usize = 200;

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
            let err = AppError::WebSocketProtocolIncompatible("仅支持文本协议".to_string());
            let err_msg = WsMessage::Error {
                code: err.error_code(),
                message: err.to_string(),
            };
            if let Ok(json) = serde_json::to_string(&err_msg) {
                let _ = sender.send(Message::Text(json.into())).await;
            }
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
        Ok(Some(Err(_))) | Ok(None) | Err(_) => {
            // 超时/断开/错误 -> 直接断开
            let err = AppError::WebSocketError("鉴权阶段连接异常".to_string());
            tracing::debug!(code = err.error_code(), "websocket auth phase error");
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let auth_event: WsMessage = match serde_json::from_str(auth_msg.as_str()) {
        Ok(m) => m,
        Err(_) => {
            let err = AppError::WebSocketMessageFormatError("鉴权消息格式无效".to_string());
            let err_msg = WsMessage::Error {
                code: err.error_code(),
                message: err.to_string(),
            };
            if let Ok(json) = serde_json::to_string(&err_msg) {
                let _ = sender.send(Message::Text(json.into())).await;
            }
            let _ = sender.send(Message::Close(None)).await;
            return;
        }
    };

    let access_token = match auth_event {
        WsMessage::Auth { access_token, .. } => access_token,
        _ => {
            // 首帧不是 Auth
            let err = AppError::WebSocketAuthFailed("鉴权失败：未提供 Token".to_string());
            let err_msg = WsMessage::Error {
                code: err.error_code(),
                message: err.to_string(),
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
            let err = AppError::WebSocketAuthFailed("鉴权失败：Token 无效或过期".to_string());
            let err_msg = WsMessage::Error {
                code: err.error_code(),
                message: err.to_string(),
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
            let err = AppError::WebSocketAuthFailed("鉴权失败：Token 无效或过期".to_string());
            let err_msg = WsMessage::Error {
                code: err.error_code(),
                message: err.to_string(),
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
        .get_user_status_subscribers(user_id)
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

/// 处理WebSocket消息
async fn handle_message(
    msg: Message,
    user_id: i64,
    _nickname: &str,
    connection_id: &str,
    state: &AppState,
) -> Result<(), String> {
    match msg {
        Message::Text(text) => {
            // 解析客户端消息
            let client_msg: WsMessage = serde_json::from_str(text.as_str())
                .map_err(|e| AppError::WebSocketMessageFormatError(e.to_string()).to_string())?;

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

                WsMessage::Subscribe {
                    client_msg_id,
                    r#type,
                    ids,
                } => {
                    if let Err(err) = validate_subscription_request(&r#type, &ids, false) {
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Subscribe,
                            r#type,
                            ids: current_ids,
                            code: err.error_code(),
                            message: Some(err.to_string()),
                            failed_ids: None,
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }
                    if is_rate_limited(user_id, state).await {
                        let failed_ids =
                            build_failed_ids(&ids, SubscriptionFailureReason::RateLimited);
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Subscribe,
                            r#type,
                            ids: current_ids,
                            code: 20103,
                            message: Some("订阅过于频繁，请稍后重试".to_string()),
                            failed_ids: Some(failed_ids),
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }

                    // 处理增量订阅
                    let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
                    let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
                    let code = subscription_ack_code(&failed_ids);

                    // 发送订阅确认
                    let ack_msg = WsMessage::SubscribeAck {
                        client_msg_id,
                        operation: SubscriptionOperation::Subscribe,
                        r#type,
                        ids: current_ids,
                        code,
                        message: if failed_ids.is_empty() {
                            None
                        } else {
                            Some("订阅部分失败".to_string())
                        },
                        failed_ids: if failed_ids.is_empty() {
                            None
                        } else {
                            Some(failed_ids)
                        },
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }

                WsMessage::Unsubscribe {
                    client_msg_id,
                    r#type,
                    ids,
                } => {
                    if let Err(err) = validate_subscription_request(&r#type, &ids, false) {
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Unsubscribe,
                            r#type,
                            ids: current_ids,
                            code: err.error_code(),
                            message: Some(err.to_string()),
                            failed_ids: None,
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }
                    if is_rate_limited(user_id, state).await {
                        let failed_ids =
                            build_failed_ids(&ids, SubscriptionFailureReason::RateLimited);
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Unsubscribe,
                            r#type,
                            ids: current_ids,
                            code: 20103,
                            message: Some("订阅过于频繁，请稍后重试".to_string()),
                            failed_ids: Some(failed_ids),
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }

                    // 处理取消订阅
                    process_unsubscribe(user_id, &r#type, &ids, state).await;
                    let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;

                    // 发送取消订阅确认
                    let ack_msg = WsMessage::SubscribeAck {
                        client_msg_id,
                        operation: SubscriptionOperation::Unsubscribe,
                        r#type,
                        ids: current_ids,
                        code: 0,
                        message: None,
                        failed_ids: None,
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }

                WsMessage::SetSubscriptions {
                    client_msg_id,
                    r#type,
                    ids,
                } => {
                    if let Err(err) = validate_subscription_request(&r#type, &ids, true) {
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Set,
                            r#type,
                            ids: current_ids,
                            code: err.error_code(),
                            message: Some(err.to_string()),
                            failed_ids: None,
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }
                    if is_rate_limited(user_id, state).await {
                        let failed_ids =
                            build_failed_ids(&ids, SubscriptionFailureReason::RateLimited);
                        let current_ids =
                            build_current_subscription_ids(user_id, state, &r#type).await;
                        let ack_msg = WsMessage::SubscribeAck {
                            client_msg_id,
                            operation: SubscriptionOperation::Set,
                            r#type,
                            ids: current_ids,
                            code: 20103,
                            message: Some("订阅过于频繁，请稍后重试".to_string()),
                            failed_ids: Some(failed_ids),
                        };
                        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                        return Ok(());
                    }

                    // 先清空该类型订阅
                    clear_subscriptions_by_type(user_id, &r#type, state).await;

                    // 然后添加新订阅
                    let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
                    let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
                    let code = subscription_ack_code(&failed_ids);

                    // 发送确认
                    let ack_msg = WsMessage::SubscribeAck {
                        client_msg_id,
                        operation: SubscriptionOperation::Set,
                        r#type,
                        ids: current_ids,
                        code,
                        message: if failed_ids.is_empty() {
                            None
                        } else {
                            Some("订阅部分失败".to_string())
                        },
                        failed_ids: if failed_ids.is_empty() {
                            None
                        } else {
                            Some(failed_ids)
                        },
                    };
                    state.ws_manager.send_to_user(user_id, &ack_msg).await?;
                }

                WsMessage::PrivateTyping {
                    target_user_id,
                    is_typing,
                } => {
                    let typing_msg = WsMessage::PrivateUserTyping {
                        target_user_id,
                        user_id,
                        is_typing,
                    };

                    // 私聊：推给对方
                    let _ = state
                        .ws_manager
                        .send_to_user(target_user_id, &typing_msg)
                        .await;
                }

                WsMessage::GroupTyping {
                    group_chat_id,
                    is_typing,
                } => {
                    let typing_msg = WsMessage::GroupUserTyping {
                        group_chat_id: group_chat_id.clone(),
                        user_id,
                        is_typing,
                    };

                    // 群聊：推送给订阅了该群聊的所有用户（排除自己）
                    let subscribers = state
                        .ws_subscription
                        .get_group_chat_subscribers(&group_chat_id)
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

                _ => {
                    // 客户端发了未识别的事件：更适合 debug（避免被噪音刷屏）
                    tracing::debug!("收到未处理的客户端消息类型: user_id={}", user_id);
                }
            }
        }

        Message::Binary(_) => {
            // 二进制消息通常是客户端误用/探测：这里降到 debug，避免刷屏
            let err = AppError::WebSocketProtocolIncompatible("二进制消息暂不支持".to_string());
            tracing::debug!(
                code = err.error_code(),
                "收到不兼容的二进制消息: user_id={}",
                user_id
            );
            return Err(err.to_string());
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

/// 处理订阅请求（单一类型）
async fn process_subscribe(
    user_id: i64,
    source_type: &str,
    ids: &[SubscriptionSourceId],
    state: &AppState,
) -> Vec<FailedSubscriptionId> {
    let mut failed = Vec::new();
    let mut group_count = state.ws_subscription.count_by_type(user_id, "group").await;
    let mut user_count = state.ws_subscription.count_by_type(user_id, "user").await;

    let group_repo = GroupRepository::new(state.db.clone());
    let group_member_repo = GroupMemberRepository::new(state.db.clone(), state.redis.clone());
    let user_repo = UserRepository::new(state.db.clone());
    let blocks_repo = UserBlocksRepository::new(state.db.clone());

    for source_id in ids {
        match source_type {
            "group" => {
                let group_id = match source_id {
                    SubscriptionSourceId::String(raw) => match Uuid::parse_str(raw) {
                        Ok(id) => id,
                        Err(_) => {
                            failed.push(build_failed_id(
                                source_id,
                                SubscriptionFailureReason::InvalidItem,
                            ));
                            continue;
                        }
                    },
                    SubscriptionSourceId::Number(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InvalidItem,
                        ));
                        continue;
                    }
                };
                let exists = match group_repo.exists(group_id).await {
                    Ok(v) => v,
                    Err(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InternalError,
                        ));
                        continue;
                    }
                };
                if !exists {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::NotFound,
                    ));
                    continue;
                }
                let is_member = match group_member_repo.is_member(group_id, user_id).await {
                    Ok(v) => v,
                    Err(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InternalError,
                        ));
                        continue;
                    }
                };
                if !is_member {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::PermissionDenied,
                    ));
                    continue;
                }
                if group_count >= 100 {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::TooManySubscriptions,
                    ));
                    continue;
                }

                let group_id_str = match source_id {
                    SubscriptionSourceId::String(raw) => raw.clone(),
                    SubscriptionSourceId::Number(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InvalidItem,
                        ));
                        continue;
                    }
                };
                let subscription = SubscriptionType::Group(group_id_str);
                if state
                    .ws_subscription
                    .has_subscription(user_id, &subscription)
                    .await
                {
                    continue;
                }

                state.ws_subscription.subscribe(user_id, subscription).await;
                group_count += 1;
            }
            "user" => {
                let target_user_id = match source_id {
                    SubscriptionSourceId::Number(id) => *id,
                    SubscriptionSourceId::String(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InvalidItem,
                        ));
                        continue;
                    }
                };
                let exists = match user_repo.exists(target_user_id).await {
                    Ok(v) => v,
                    Err(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InternalError,
                        ));
                        continue;
                    }
                };
                if !exists {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::NotFound,
                    ));
                    continue;
                }
                let blocked = match blocks_repo.exists(target_user_id, user_id).await {
                    Ok(v) => v,
                    Err(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InternalError,
                        ));
                        continue;
                    }
                } || match blocks_repo.exists(user_id, target_user_id).await {
                    Ok(v) => v,
                    Err(_) => {
                        failed.push(build_failed_id(
                            source_id,
                            SubscriptionFailureReason::InternalError,
                        ));
                        continue;
                    }
                };
                if blocked {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::PermissionDenied,
                    ));
                    continue;
                }
                if user_count >= 500 {
                    failed.push(build_failed_id(
                        source_id,
                        SubscriptionFailureReason::TooManySubscriptions,
                    ));
                    continue;
                }

                let subscription = SubscriptionType::User(target_user_id);
                if state
                    .ws_subscription
                    .has_subscription(user_id, &subscription)
                    .await
                {
                    continue;
                }

                state.ws_subscription.subscribe(user_id, subscription).await;
                user_count += 1;

                // 订阅用户状态后，立刻推送一次“当前在线状态”
                if state
                    .ws_manager
                    .get_user_connection_count(target_user_id)
                    .await
                    > 0
                {
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
            _ => {
                failed.push(build_failed_id(
                    source_id,
                    SubscriptionFailureReason::InvalidItem,
                ));
            }
        }
    }

    failed
}

/// 处理取消订阅请求（单一类型）
async fn process_unsubscribe(
    user_id: i64,
    source_type: &str,
    ids: &[SubscriptionSourceId],
    state: &AppState,
) {
    for source_id in ids {
        match source_type {
            "group" => {
                let group_id = match source_id {
                    SubscriptionSourceId::String(raw) => raw.clone(),
                    SubscriptionSourceId::Number(_) => continue,
                };
                let subscription = SubscriptionType::Group(group_id);
                state
                    .ws_subscription
                    .unsubscribe(user_id, &subscription)
                    .await;
            }
            "user" => {
                let target_user_id = match source_id {
                    SubscriptionSourceId::Number(id) => *id,
                    SubscriptionSourceId::String(_) => continue,
                };
                let subscription = SubscriptionType::User(target_user_id);
                state
                    .ws_subscription
                    .unsubscribe(user_id, &subscription)
                    .await;
            }
            _ => {
                // 忽略不支持的类型
            }
        }
    }
}

/// 清理指定类型的订阅
async fn clear_subscriptions_by_type(user_id: i64, source_type: &str, state: &AppState) {
    let subscriptions = state.ws_subscription.get_subscriptions(user_id).await;
    for sub in subscriptions {
        match (source_type, &sub) {
            ("group", SubscriptionType::Group(_)) => {
                state.ws_subscription.unsubscribe(user_id, &sub).await;
            }
            ("user", SubscriptionType::User(_)) => {
                state.ws_subscription.unsubscribe(user_id, &sub).await;
            }
            _ => {}
        }
    }
}

async fn is_rate_limited(user_id: i64, state: &AppState) -> bool {
    let limiter = RateLimiter::new(state.redis.clone());
    let key = format!("ws_subscribe:{}", user_id);
    let max_requests = 10;
    let window_secs = 1;
    let limited = limiter
        .is_rate_limited(&key, max_requests, window_secs)
        .await
        .unwrap_or(false);
    if limited {
        let remaining = limiter
            .get_remaining(&key, max_requests, window_secs)
            .await
            .unwrap_or(0);
        tracing::debug!(user_id, remaining, "websocket subscribe rate limited");
    }
    limited
}

fn build_failed_ids(
    ids: &[SubscriptionSourceId],
    reason: SubscriptionFailureReason,
) -> Vec<FailedSubscriptionId> {
    ids.iter().map(|id| build_failed_id(id, reason)).collect()
}

fn build_failed_id(
    id: &SubscriptionSourceId,
    reason: SubscriptionFailureReason,
) -> FailedSubscriptionId {
    match id {
        SubscriptionSourceId::String(raw) => FailedSubscriptionId::String {
            id: raw.clone(),
            reason,
        },
        SubscriptionSourceId::Number(num) => FailedSubscriptionId::Number { id: *num, reason },
    }
}

fn validate_subscription_request(
    source_type: &str,
    ids: &[SubscriptionSourceId],
    allow_empty: bool,
) -> Result<(), AppError> {
    if source_type != "group" && source_type != "user" {
        return Err(AppError::InvalidSubscriptionFormat(
            "订阅类型无效".to_string(),
        ));
    }
    if !allow_empty && ids.is_empty() {
        return Err(AppError::InvalidSubscriptionFormat(
            "订阅ID不能为空".to_string(),
        ));
    }
    if ids.len() > MAX_SUBSCRIPTION_SOURCE_IDS {
        return Err(AppError::BatchLimitExceeded);
    }
    Ok(())
}

fn source_id_sort_key(id: &SubscriptionSourceId) -> String {
    match id {
        SubscriptionSourceId::String(raw) => format!("s:{}", raw),
        SubscriptionSourceId::Number(num) => format!("n:{:020}", num),
    }
}

async fn build_current_subscription_ids(
    user_id: i64,
    state: &AppState,
    source_type: &str,
) -> Vec<SubscriptionSourceId> {
    let subscriptions = state.ws_subscription.get_subscriptions(user_id).await;
    let mut ids: Vec<SubscriptionSourceId> = Vec::new();

    for sub in subscriptions {
        match (source_type, sub) {
            ("group", SubscriptionType::Group(id)) => {
                ids.push(SubscriptionSourceId::String(id));
            }
            ("user", SubscriptionType::User(id)) => {
                ids.push(SubscriptionSourceId::Number(id));
            }
            _ => {}
        }
    }

    ids.sort_by(|a, b| source_id_sort_key(a).cmp(&source_id_sort_key(b)));
    ids
}

fn subscription_ack_code(failed_ids: &[FailedSubscriptionId]) -> i32 {
    if failed_ids.is_empty() {
        return 0;
    }
    if failed_ids.iter().any(|i| {
        matches!(
            failure_reason(i),
            SubscriptionFailureReason::RateLimited
                | SubscriptionFailureReason::TooManySubscriptions
        )
    }) {
        return AppError::SubscriptionLimitExceeded.error_code();
    }
    if failed_ids.iter().any(|i| {
        matches!(
            failure_reason(i),
            SubscriptionFailureReason::PermissionDenied
        )
    }) {
        return AppError::SubscriptionPermissionDenied("权限不足".to_string()).error_code();
    }
    if failed_ids
        .iter()
        .any(|i| matches!(failure_reason(i), SubscriptionFailureReason::NotFound))
    {
        return AppError::SubscriptionChannelNotFound("订阅目标不存在".to_string()).error_code();
    }
    if failed_ids
        .iter()
        .any(|i| matches!(failure_reason(i), SubscriptionFailureReason::InvalidItem))
    {
        return AppError::InvalidSubscriptionFormat("订阅项格式无效".to_string()).error_code();
    }
    AppError::SubscriptionFailed("订阅失败".to_string()).error_code()
}

fn failure_reason(item: &FailedSubscriptionId) -> SubscriptionFailureReason {
    match item {
        FailedSubscriptionId::Number { reason, .. } => *reason,
        FailedSubscriptionId::String { reason, .. } => *reason,
    }
}
