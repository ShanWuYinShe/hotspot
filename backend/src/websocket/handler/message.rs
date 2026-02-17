//! WebSocket message handling and dispatch.

use axum::extract::ws::Message;

use crate::{
    error::AppError, state::AppState, types::enums::SubscriptionOperation,
    websocket::subscription::SubscriptionType, websocket::types::WsMessage,
};

use super::subscribe::{
    build_current_subscription_ids, build_failed_ids, clear_subscriptions_by_type, is_rate_limited,
    process_subscribe, process_unsubscribe, subscription_ack_code, validate_subscription_request,
};

/// Main message dispatcher. Delegates to focused handlers.
pub async fn handle_message(
    msg: Message,
    user_id: i64,
    _nickname: &str,
    connection_id: &str,
    state: &AppState,
) -> Result<(), String> {
    match msg {
        Message::Text(text) => {
            let client_msg: WsMessage = serde_json::from_str(text.as_str())
                .map_err(|e| AppError::WebSocketMessageFormatError(e.to_string()).to_string())?;

            match client_msg {
                WsMessage::Ping { .. } => handle_ping(state, user_id, connection_id).await?,

                WsMessage::SetSubscriptions {
                    client_msg_id,
                    operation,
                    r#type,
                    ids,
                } => {
                    handle_set_subscriptions(
                        state,
                        user_id,
                        connection_id,
                        client_msg_id,
                        operation,
                        r#type,
                        ids,
                    )
                    .await?
                }

                WsMessage::PrivateTyping {
                    target_user_id,
                    is_typing,
                } => handle_private_typing(state, user_id, target_user_id, is_typing).await,

                WsMessage::GroupTyping {
                    group_id,
                    is_typing,
                } => handle_group_typing(state, user_id, group_id, is_typing).await,

                WsMessage::GetSubscriptions { client_msg_id } => {
                    handle_get_subscriptions(state, user_id, client_msg_id).await?
                }

                _ => {
                    tracing::debug!("收到未处理的客户端消息类型: user_id={}", user_id);
                }
            }
        }

        Message::Binary(_) => {
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
            let _ = state.ws_manager.update_ping(connection_id).await;
        }

        Message::Close(_) => {
            tracing::debug!("收到关闭消息: user_id={}", user_id);
        }
    }

    Ok(())
}

async fn handle_ping(state: &AppState, user_id: i64, connection_id: &str) -> Result<(), String> {
    state.ws_manager.update_ping(connection_id).await?;

    let pong_msg = WsMessage::Pong {};
    state.ws_manager.send_to_user(user_id, &pong_msg).await?;

    Ok(())
}

async fn handle_set_subscriptions(
    state: &AppState,
    user_id: i64,
    _connection_id: &str,
    client_msg_id: String,
    operation: SubscriptionOperation,
    r#type: String,
    ids: Vec<String>,
) -> Result<(), String> {
    let is_set_op = operation == SubscriptionOperation::Set;

    if let Err(err) = validate_subscription_request(&r#type, &ids, is_set_op) {
        let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
        let ack_msg = WsMessage::SubscribeAck {
            client_msg_id,
            operation,
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
        let failed_ids = build_failed_ids(
            &ids,
            crate::types::enums::SubscriptionFailureReason::InternalError,
        );
        let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
        let ack_msg = WsMessage::SubscribeAck {
            client_msg_id,
            operation,
            r#type: r#type.clone(),
            ids: current_ids,
            code: 20300,
            message: Some("订阅过于频繁，请稍后重试".to_string()),
            failed_ids: Some(failed_ids),
        };
        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
        return Ok(());
    }

    match operation {
        SubscriptionOperation::Subscribe => {
            let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
            let code = subscription_ack_code(&failed_ids);
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
        SubscriptionOperation::Unsubscribe => {
            process_unsubscribe(user_id, &r#type, &ids, state).await;
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
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
        SubscriptionOperation::Set => {
            clear_subscriptions_by_type(user_id, &r#type, state).await;
            let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
            let code = subscription_ack_code(&failed_ids);
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
    }

    Ok(())
}

async fn handle_private_typing(
    state: &AppState,
    user_id: i64,
    target_user_id: i64,
    is_typing: bool,
) {
    let typing_msg = WsMessage::PrivateUserTyping {
        target_user_id,
        user_id,
        is_typing,
    };
    let _ = state
        .ws_manager
        .send_to_user(target_user_id, &typing_msg)
        .await;
}

async fn handle_group_typing(state: &AppState, user_id: i64, group_id: String, is_typing: bool) {
    let typing_msg = WsMessage::GroupUserTyping {
        group_id: group_id.clone(),
        user_id,
        is_typing,
    };

    let subscribers = state
        .ws_subscription
        .get_group_chat_subscribers(&group_id)
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

async fn handle_get_subscriptions(
    state: &AppState,
    user_id: i64,
    client_msg_id: String,
) -> Result<(), String> {
    let subscriptions = state.ws_subscription.get_subscriptions(user_id).await;
    let mut user_ids = Vec::new();
    let mut group_ids = Vec::new();
    for sub in subscriptions {
        match sub {
            SubscriptionType::User(uid) => {
                user_ids.push(uid.to_string());
            }
            SubscriptionType::Group(gid) => {
                group_ids.push(gid);
            }
        }
    }
    let snapshot_msg = WsMessage::SubscriptionsSnapshot {
        client_msg_id,
        user_ids,
        group_ids,
    };
    state
        .ws_manager
        .send_to_user(user_id, &snapshot_msg)
        .await?;

    Ok(())
}
