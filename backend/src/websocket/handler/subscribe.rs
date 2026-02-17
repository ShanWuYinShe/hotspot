//! WebSocket subscription handling.

pub const MAX_SUBSCRIPTION_SOURCE_IDS: usize = 200;

use crate::{
    error::AppError,
    repositories::{
        group::{group_repo::GroupRepository, member_repo::GroupMemberRepository},
        user::{blocks_repo::UserBlocksRepository, user_repo::UserRepository},
    },
    state::AppState,
    types::enums::SubscriptionFailureReason,
    websocket::subscription::SubscriptionType,
    websocket::types::{FailedSubscriptionId, WsMessage},
};

/// Process subscribe requests for a single type. Returns list of failed subscription IDs.
pub async fn process_subscribe(
    user_id: i64,
    source_type: &str,
    ids: &[String],
    state: &AppState,
) -> Vec<FailedSubscriptionId> {
    let mut failed = Vec::new();
    let mut group_count = state.ws_subscription.count_by_type(user_id, "group").await;
    let mut user_count = state.ws_subscription.count_by_type(user_id, "user").await;

    for source_id in ids {
        match source_type {
            "group" => {
                if let Some(f) =
                    subscribe_to_group(state, user_id, source_id, &mut group_count).await
                {
                    failed.push(f);
                }
            }
            "user" => {
                if let Some(f) = subscribe_to_user(state, user_id, source_id, &mut user_count).await
                {
                    failed.push(f);
                }
            }
            _ => {
                failed.push(build_failed_id(
                    source_id,
                    SubscriptionFailureReason::InvalidTopic,
                ));
            }
        }
    }

    failed
}

async fn subscribe_to_group(
    state: &AppState,
    user_id: i64,
    source_id: &str,
    group_count: &mut usize,
) -> Option<FailedSubscriptionId> {
    let group_id = match source_id.parse::<i64>() {
        Ok(id) => id,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InvalidTopic,
            ));
        }
    };

    let group_repo = GroupRepository::new(state.db.clone());
    let group_member_repo = GroupMemberRepository::new(state.db.clone(), state.redis.clone());

    let exists = match group_repo.exists(group_id).await {
        Ok(v) => v,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !exists {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::InvalidTopic,
        ));
    }

    let is_member = match group_member_repo.is_member(group_id, user_id).await {
        Ok(v) => v,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !is_member {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::Forbidden,
        ));
    }
    if *group_count >= 100 {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::LimitExceeded,
        ));
    }

    let subscription = SubscriptionType::Group(source_id.to_string());
    if state
        .ws_subscription
        .has_subscription(user_id, &subscription)
        .await
    {
        return None;
    }

    state.ws_subscription.subscribe(user_id, subscription).await;
    *group_count += 1;
    None
}

async fn subscribe_to_user(
    state: &AppState,
    user_id: i64,
    source_id: &str,
    user_count: &mut usize,
) -> Option<FailedSubscriptionId> {
    let target_user_id = match source_id.parse::<i64>() {
        Ok(id) => id,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InvalidTopic,
            ));
        }
    };

    let user_repo = UserRepository::new(state.db.clone());
    let blocks_repo = UserBlocksRepository::new(state.db.clone());

    let exists = match user_repo.exists(target_user_id).await {
        Ok(v) => v,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !exists {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::InvalidTopic,
        ));
    }

    let blocked = match blocks_repo.is_blocked(target_user_id, user_id).await {
        Ok(v) => v,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    } || match blocks_repo.is_blocked(user_id, target_user_id).await {
        Ok(v) => v,
        Err(_) => {
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if blocked {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::Forbidden,
        ));
    }
    if *user_count >= 500 {
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::LimitExceeded,
        ));
    }

    let subscription = SubscriptionType::User(target_user_id);
    if state
        .ws_subscription
        .has_subscription(user_id, &subscription)
        .await
    {
        return None;
    }

    state.ws_subscription.subscribe(user_id, subscription).await;
    *user_count += 1;

    // 订阅用户状态后，立刻推送一次"当前在线状态"
    if state
        .ws_manager
        .get_user_connection_count(target_user_id)
        .await
        > 0
    {
        if let Ok(Some(_)) = user_repo.find_by_id(target_user_id).await {
            let msg = WsMessage::UserOnline {
                user_id: target_user_id,
                online_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = state.ws_manager.send_to_user(user_id, &msg).await;
        }
    }

    None
}

/// Process unsubscribe requests for a single type.
pub async fn process_unsubscribe(
    user_id: i64,
    source_type: &str,
    ids: &[String],
    state: &AppState,
) {
    for source_id in ids {
        match source_type {
            "group" => {
                let subscription = SubscriptionType::Group(source_id.clone());
                state
                    .ws_subscription
                    .unsubscribe(user_id, &subscription)
                    .await;
            }
            "user" => {
                let target_user_id = match source_id.parse::<i64>() {
                    Ok(id) => id,
                    Err(_) => continue,
                };
                let subscription = SubscriptionType::User(target_user_id);
                state
                    .ws_subscription
                    .unsubscribe(user_id, &subscription)
                    .await;
            }
            _ => {}
        }
    }
}

/// Clear all subscriptions of a given type for a user.
pub async fn clear_subscriptions_by_type(user_id: i64, source_type: &str, state: &AppState) {
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

pub fn build_failed_ids(
    ids: &[String],
    reason: SubscriptionFailureReason,
) -> Vec<FailedSubscriptionId> {
    ids.iter().map(|id| build_failed_id(id, reason)).collect()
}

pub fn build_failed_id(id: &str, reason: SubscriptionFailureReason) -> FailedSubscriptionId {
    FailedSubscriptionId {
        subscription_id: id.to_string(),
        reason,
    }
}

pub fn subscription_ack_code(failed_ids: &[FailedSubscriptionId]) -> i32 {
    if failed_ids.is_empty() {
        return 0;
    }
    if failed_ids
        .iter()
        .any(|i| matches!(i.reason, SubscriptionFailureReason::InternalError))
    {
        return AppError::SubscriptionFailed("订阅失败".to_string()).error_code();
    }
    if failed_ids
        .iter()
        .any(|i| matches!(i.reason, SubscriptionFailureReason::Forbidden))
    {
        return AppError::SubscriptionPermissionDenied("权限不足".to_string()).error_code();
    }
    if failed_ids
        .iter()
        .any(|i| matches!(i.reason, SubscriptionFailureReason::InvalidTopic))
    {
        return AppError::InvalidSubscriptionFormat("订阅项格式无效".to_string()).error_code();
    }
    AppError::SubscriptionFailed("订阅失败".to_string()).error_code()
}

pub fn validate_subscription_request(
    source_type: &str,
    ids: &[String],
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

pub async fn build_current_subscription_ids(
    user_id: i64,
    state: &AppState,
    source_type: &str,
) -> Vec<String> {
    let subscriptions = state.ws_subscription.get_subscriptions(user_id).await;
    let mut ids: Vec<String> = Vec::new();

    for sub in subscriptions {
        match (source_type, sub) {
            ("group", SubscriptionType::Group(id)) => {
                ids.push(id);
            }
            ("user", SubscriptionType::User(id)) => {
                ids.push(id.to_string());
            }
            _ => {}
        }
    }

    ids.sort();
    ids
}

pub async fn is_rate_limited(user_id: i64, state: &AppState) -> bool {
    let limiter = crate::utils::cache::RateLimiter::new(state.redis.clone());
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
