//! WebSocket message handling and dispatch.

use axum::extract::ws::Message;
use serde_json::json;

use crate::{
    error::ApiError,
    repositories::group::member_repo::GroupMemberRepository,
    state::AppState,
    types::enums::SubscriptionOperation,
    websocket::subscription::SubscriptionType,
    websocket::types::{FailedSubscriptionId, WsMessage},
};

use super::subscribe::{
    build_current_subscription_ids, build_failed_ids, clear_subscriptions_by_type, is_rate_limited,
    process_subscribe, process_unsubscribe, subscription_ack_code, validate_subscription_request,
};

fn truncate_for_log(input: &str, max_chars: usize) -> String {
    let mut iter = input.chars();
    let truncated: String = iter.by_ref().take(max_chars).collect();
    if iter.next().is_some() {
        format!("{truncated}...")
    } else {
        truncated
    }
}

fn summarize_failed_subscription_ids(
    failed_ids: &[FailedSubscriptionId],
) -> (usize, usize, usize, usize) {
    let invalid_count = failed_ids
        .iter()
        .filter(|item| {
            matches!(
                item.reason,
                crate::types::enums::SubscriptionFailureReason::InvalidTopic
            )
        })
        .count();
    let forbidden_count = failed_ids
        .iter()
        .filter(|item| {
            matches!(
                item.reason,
                crate::types::enums::SubscriptionFailureReason::Forbidden
            )
        })
        .count();
    let limit_count = failed_ids
        .iter()
        .filter(|item| {
            matches!(
                item.reason,
                crate::types::enums::SubscriptionFailureReason::LimitExceeded
            )
        })
        .count();
    let internal_count = failed_ids
        .iter()
        .filter(|item| {
            matches!(
                item.reason,
                crate::types::enums::SubscriptionFailureReason::InternalError
            )
        })
        .count();
    (invalid_count, forbidden_count, limit_count, internal_count)
}

async fn should_skip_duplicate_typing(
    state: &AppState,
    scope: &str,
    actor_id: i64,
    target: &str,
    is_typing: bool,
) -> bool {
    let status = if is_typing { "typing" } else { "stopped" };
    let dedupe_key = format!(
        "ws:typing:dedupe:{}:{}:{}:{}",
        scope, actor_id, target, status
    );
    let mut conn = state.redis.clone();
    let set_result: Option<String> = match redis::cmd("SET")
        .arg(&dedupe_key)
        .arg("1")
        .arg("EX")
        .arg(3)
        .arg("NX")
        .query_async(&mut conn)
        .await
    {
        Ok(v) => v,
        Err(err) => {
            tracing::warn!(
                scope,
                actor_id,
                target,
                is_typing,
                error = %err,
                "typing dedupe check failed, fallback to forward event"
            );
            return false;
        }
    };

    if set_result.is_none() {
        return true;
    }
    false
}

async fn send_ws_error(
    state: &AppState,
    user_id: i64,
    connection_id: &str,
    code: i32,
    message: &str,
    severity: &str,
    retryable: bool,
    retry_after: Option<u32>,
    context: Option<serde_json::Value>,
) -> Result<(), String> {
    let ws_error = WsMessage::Error {
        code,
        message: message.to_string(),
        severity: severity.to_string(),
        retryable,
        retry_after,
        context,
    };
    state
        .ws_manager
        .send_to_user(user_id, &ws_error)
        .await
        .map_err(|err| {
            tracing::warn!(
                user_id,
                connection_id,
                error = %err,
                code,
                "failed to send websocket error event"
            );
            err
        })
}

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
            let client_msg: WsMessage = match serde_json::from_str(text.as_str()) {
                Ok(msg) => msg,
                Err(err) => {
                    tracing::warn!(
                        user_id,
                        connection_id,
                        error = %err,
                        message_preview = %truncate_for_log(text.as_str(), 256),
                        "invalid websocket client message json"
                    );
                    send_ws_error(
                        state,
                        user_id,
                        connection_id,
                        ApiError::WebSocketMessageFormatError(
                            "客户端消息不是合法 JSON".to_string(),
                        )
                        .error_code(),
                        "消息格式错误：请发送合法 JSON",
                        "warning",
                        false,
                        None,
                        Some(json!({ "operation": "parse_client_message" })),
                    )
                    .await?;
                    return Ok(());
                }
            };

            match client_msg {
                WsMessage::Ping { .. } => handle_ping(state, user_id, connection_id).await?,

                WsMessage::SetSubscriptions {
                    client_request_id,
                    operation,
                    r#type,
                    ids,
                } => {
                    handle_set_subscriptions(
                        state,
                        user_id,
                        connection_id,
                        client_request_id,
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

                WsMessage::GetSubscriptions { client_request_id } => {
                    handle_get_subscriptions(state, user_id, connection_id, client_request_id)
                        .await?
                }

                other => {
                    tracing::warn!(
                        user_id,
                        connection_id,
                        message = ?other,
                        "收到不支持的客户端消息类型"
                    );
                    send_ws_error(
                        state,
                        user_id,
                        connection_id,
                        ApiError::WebSocketProtocolIncompatible(
                            "协议不兼容：不支持的消息类型".to_string(),
                        )
                        .error_code(),
                        "协议不兼容：不支持的消息类型",
                        "warning",
                        false,
                        None,
                        Some(json!({ "operation": "unsupported_client_message" })),
                    )
                    .await?;
                }
            }
        }

        Message::Binary(_) => {
            let err = ApiError::WebSocketProtocolIncompatible("二进制消息暂不支持".to_string());
            tracing::warn!(
                user_id,
                connection_id,
                code = err.error_code(),
                "收到不兼容的二进制消息"
            );
            send_ws_error(
                state,
                user_id,
                connection_id,
                err.error_code(),
                "协议不兼容：二进制消息暂不支持",
                "warning",
                false,
                None,
                Some(json!({ "operation": "binary_frame_not_supported" })),
            )
            .await?;
            return Ok(());
        }

        Message::Ping(_) => {
            // Axum会自动回复Pong
        }

        Message::Pong(_) => {
            if let Err(err) = state.ws_manager.update_ping(connection_id).await {
                tracing::warn!(
                    user_id,
                    connection_id,
                    error = %err,
                    "websocket pong ignored: failed to update ping timestamp"
                );
            }
        }

        Message::Close(_) => {
            tracing::debug!(user_id, connection_id, "收到关闭消息");
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
    connection_id: &str,
    client_request_id: String,
    operation: SubscriptionOperation,
    r#type: String,
    ids: Vec<String>,
) -> Result<(), String> {
    let request_id = client_request_id.clone();
    tracing::debug!(
        user_id,
        connection_id,
        client_request_id = %request_id,
        operation = ?operation,
        subscription_type = %r#type,
        ids_count = ids.len(),
        "received websocket subscription request"
    );
    let is_set_op = operation == SubscriptionOperation::Set;

    if let Err(err) = validate_subscription_request(&r#type, &ids, is_set_op) {
        tracing::warn!(
            user_id,
            connection_id,
            client_request_id = %request_id,
            operation = ?operation,
            subscription_type = %r#type,
            ids_count = ids.len(),
            error = %err,
            "subscription request validation failed"
        );
        let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
        let ack_count = current_ids.len();
        let code = err.error_code();
        let message = err.to_string();
        let ack_msg = WsMessage::SubscribeAck {
            client_request_id,
            operation,
            r#type: r#type.clone(),
            ids: current_ids,
            code,
            message: Some(message),
            failed_ids: None,
        };
        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
        tracing::warn!(
            user_id,
            connection_id,
            client_request_id = %request_id,
            operation = ?operation,
            subscription_type = %r#type,
            requested_count = ids.len(),
            ack_count,
            code,
            "subscription request rejected by validation"
        );
        return Ok(());
    }

    if is_rate_limited(user_id, state).await {
        tracing::warn!(
            user_id,
            connection_id,
            client_request_id = %request_id,
            operation = ?operation,
            subscription_type = %r#type,
            ids_count = ids.len(),
            "subscription request rate-limited"
        );
        let failed_ids = build_failed_ids(
            &ids,
            crate::types::enums::SubscriptionFailureReason::InternalError,
        );
        let failed_count = failed_ids.len();
        let (invalid_count, forbidden_count, limit_count, internal_count) =
            summarize_failed_subscription_ids(&failed_ids);
        let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
        let ack_count = current_ids.len();
        let code = subscription_ack_code(&failed_ids);
        let ack_msg = WsMessage::SubscribeAck {
            client_request_id,
            operation,
            r#type: r#type.clone(),
            ids: current_ids,
            code,
            message: Some("订阅请求过于频繁，请稍后重试".to_string()),
            failed_ids: Some(failed_ids),
        };
        state.ws_manager.send_to_user(user_id, &ack_msg).await?;
        tracing::warn!(
            user_id,
            connection_id,
            client_request_id = %request_id,
            operation = ?operation,
            subscription_type = %r#type,
            requested_count = ids.len(),
            ack_count,
            failed_count,
            invalid_count,
            forbidden_count,
            limit_count,
            internal_count,
            code,
            retryable = true,
            "subscription request rejected by rate limit"
        );
        return Ok(());
    }

    match operation {
        SubscriptionOperation::Subscribe => {
            let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
            let failed_count = failed_ids.len();
            let (invalid_count, forbidden_count, limit_count, internal_count) =
                summarize_failed_subscription_ids(&failed_ids);
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
            let ack_count = current_ids.len();
            let code = subscription_ack_code(&failed_ids);
            let ack_msg = WsMessage::SubscribeAck {
                client_request_id,
                operation: SubscriptionOperation::Subscribe,
                r#type: r#type.clone(),
                ids: current_ids,
                code,
                message: if failed_count == 0 {
                    None
                } else {
                    Some("订阅部分失败".to_string())
                },
                failed_ids: if failed_count == 0 {
                    None
                } else {
                    Some(failed_ids)
                },
            };
            state.ws_manager.send_to_user(user_id, &ack_msg).await?;
            tracing::info!(
                user_id,
                connection_id,
                client_request_id = %request_id,
                operation = ?SubscriptionOperation::Subscribe,
                subscription_type = %r#type,
                requested_count = ids.len(),
                ack_count,
                failed_count,
                invalid_count,
                forbidden_count,
                limit_count,
                internal_count,
                code,
                "subscription request handled"
            );
        }
        SubscriptionOperation::Unsubscribe => {
            process_unsubscribe(user_id, &r#type, &ids, state).await;
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
            let ack_count = current_ids.len();
            let code = 0;
            let ack_msg = WsMessage::SubscribeAck {
                client_request_id,
                operation: SubscriptionOperation::Unsubscribe,
                r#type: r#type.clone(),
                ids: current_ids,
                code,
                message: None,
                failed_ids: None,
            };
            state.ws_manager.send_to_user(user_id, &ack_msg).await?;
            tracing::info!(
                user_id,
                connection_id,
                client_request_id = %request_id,
                operation = ?SubscriptionOperation::Unsubscribe,
                subscription_type = %r#type,
                requested_count = ids.len(),
                ack_count,
                failed_count = 0,
                code,
                "subscription request handled"
            );
        }
        SubscriptionOperation::Set => {
            clear_subscriptions_by_type(user_id, &r#type, state).await;
            let failed_ids = process_subscribe(user_id, &r#type, &ids, state).await;
            let failed_count = failed_ids.len();
            let (invalid_count, forbidden_count, limit_count, internal_count) =
                summarize_failed_subscription_ids(&failed_ids);
            let current_ids = build_current_subscription_ids(user_id, state, &r#type).await;
            let ack_count = current_ids.len();
            let code = subscription_ack_code(&failed_ids);
            let ack_msg = WsMessage::SubscribeAck {
                client_request_id,
                operation: SubscriptionOperation::Set,
                r#type: r#type.clone(),
                ids: current_ids,
                code,
                message: if failed_count == 0 {
                    None
                } else {
                    Some("订阅部分失败".to_string())
                },
                failed_ids: if failed_count == 0 {
                    None
                } else {
                    Some(failed_ids)
                },
            };
            state.ws_manager.send_to_user(user_id, &ack_msg).await?;
            tracing::info!(
                user_id,
                connection_id,
                client_request_id = %request_id,
                operation = ?SubscriptionOperation::Set,
                subscription_type = %r#type,
                requested_count = ids.len(),
                ack_count,
                failed_count,
                invalid_count,
                forbidden_count,
                limit_count,
                internal_count,
                code,
                "subscription request handled"
            );
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
    let required_subscription = SubscriptionType::User(user_id);
    let target_is_subscribed = state
        .ws_subscription
        .has_subscription(target_user_id, &required_subscription)
        .await;
    if !target_is_subscribed {
        tracing::debug!(
            user_id,
            target_user_id,
            is_typing,
            "private typing ignored: target user has not subscribed sender presence"
        );
        return;
    }

    if should_skip_duplicate_typing(
        state,
        "private",
        user_id,
        &target_user_id.to_string(),
        is_typing,
    )
    .await
    {
        return;
    }

    let typing_msg = WsMessage::PrivateUserTyping {
        target_user_id,
        user_id,
        is_typing,
    };
    if let Err(err) = state
        .ws_manager
        .send_to_user(target_user_id, &typing_msg)
        .await
    {
        tracing::warn!(
            user_id,
            target_user_id,
            is_typing,
            error = %err,
            "private typing forward failed"
        );
    }
}

async fn handle_group_typing(state: &AppState, user_id: i64, group_id: String, is_typing: bool) {
    let group_id_i64 = match group_id.parse::<i64>() {
        Ok(id) => id,
        Err(err) => {
            tracing::warn!(
                user_id,
                group_id,
                error = %err,
                "group typing ignored: invalid group_id"
            );
            return;
        }
    };

    let group_member_repo = GroupMemberRepository::new(state.db.clone(), state.redis.clone());
    match group_member_repo.is_member(group_id_i64, user_id).await {
        Ok(true) => {}
        Ok(false) => {
            tracing::warn!(
                user_id,
                group_id = group_id_i64,
                "group typing ignored: sender is not a group member"
            );
            return;
        }
        Err(err) => {
            tracing::error!(
                user_id,
                group_id = group_id_i64,
                error = %err,
                "group typing ignored: membership check failed"
            );
            return;
        }
    }

    if should_skip_duplicate_typing(state, "group", user_id, &group_id, is_typing).await {
        return;
    }

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
            if let Err(err) = state
                .ws_manager
                .send_to_user(subscriber_id, &typing_msg)
                .await
            {
                tracing::warn!(
                    user_id,
                    subscriber_id,
                    group_id = %group_id,
                    is_typing,
                    error = %err,
                    "group typing forward failed"
                );
            }
        }
    }
}

async fn handle_get_subscriptions(
    state: &AppState,
    user_id: i64,
    connection_id: &str,
    client_request_id: String,
) -> Result<(), String> {
    let request_id = client_request_id.clone();
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
    let user_ids_count = user_ids.len();
    let group_ids_count = group_ids.len();
    let snapshot_msg = WsMessage::SubscriptionsSnapshot {
        client_request_id,
        user_ids,
        group_ids,
    };
    tracing::debug!(
        user_id,
        connection_id,
        client_request_id = %request_id,
        user_ids_count,
        group_ids_count,
        "sending websocket subscriptions snapshot"
    );
    state
        .ws_manager
        .send_to_user(user_id, &snapshot_msg)
        .await?;

    Ok(())
}
