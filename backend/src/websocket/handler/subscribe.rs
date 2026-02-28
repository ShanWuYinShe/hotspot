//! WebSocket subscription handling.

pub const MAX_USER_SUBSCRIPTION_SOURCE_IDS: usize = 500;
pub const MAX_GROUP_SUBSCRIPTION_SOURCE_IDS: usize = 100;

use crate::{
    error::AppError,
    repositories::{
        group::{group_repo::GroupRepository, member_repo::GroupMemberRepository},
        user::{
            blocks_repo::UserBlocksRepository, settings_repo::UserSettingsRepository,
            user_repo::UserRepository,
        },
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
    tracing::debug!(
        user_id,
        source_type,
        ids_count = ids.len(),
        "processing websocket subscribe request"
    );
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
                tracing::warn!(
                    user_id,
                    source_type,
                    source_id,
                    "subscribe ignored: invalid source type"
                );
                failed.push(build_failed_id(
                    source_id,
                    SubscriptionFailureReason::InvalidTopic,
                ));
            }
        }
    }

    if failed.is_empty() {
        tracing::debug!(
            user_id,
            source_type,
            ids_count = ids.len(),
            "websocket subscribe request processed successfully"
        );
    } else {
        let invalid_count = failed
            .iter()
            .filter(|item| matches!(item.reason, SubscriptionFailureReason::InvalidTopic))
            .count();
        let forbidden_count = failed
            .iter()
            .filter(|item| matches!(item.reason, SubscriptionFailureReason::Forbidden))
            .count();
        let limit_count = failed
            .iter()
            .filter(|item| matches!(item.reason, SubscriptionFailureReason::LimitExceeded))
            .count();
        let internal_count = failed
            .iter()
            .filter(|item| matches!(item.reason, SubscriptionFailureReason::InternalError))
            .count();
        tracing::warn!(
            user_id,
            source_type,
            ids_count = ids.len(),
            failed_count = failed.len(),
            invalid_count,
            forbidden_count,
            limit_count,
            internal_count,
            "websocket subscribe request processed with failures"
        );
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
        Err(err) => {
            tracing::debug!(
                user_id,
                source_id,
                error = %err,
                "subscribe group failed: invalid group id format"
            );
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
        Err(err) => {
            tracing::error!(
                user_id,
                group_id,
                error = %err,
                "subscribe group failed: query group exists error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !exists {
        tracing::debug!(user_id, group_id, "subscribe group failed: group not found");
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::InvalidTopic,
        ));
    }

    let is_member = match group_member_repo.is_member(group_id, user_id).await {
        Ok(v) => v,
        Err(err) => {
            tracing::error!(
                user_id,
                group_id,
                error = %err,
                "subscribe group failed: query member relation error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !is_member {
        tracing::warn!(
            user_id,
            group_id,
            "subscribe group denied: user is not group member"
        );
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::Forbidden,
        ));
    }
    if *group_count >= MAX_GROUP_SUBSCRIPTION_SOURCE_IDS {
        tracing::warn!(
            user_id,
            group_id,
            group_count = *group_count,
            limit = MAX_GROUP_SUBSCRIPTION_SOURCE_IDS,
            "subscribe group denied: user subscription limit exceeded"
        );
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
        tracing::debug!(
            user_id,
            group_id,
            "subscribe group skipped: already subscribed"
        );
        return None;
    }

    state.ws_subscription.subscribe(user_id, subscription).await;
    *group_count += 1;
    tracing::debug!(
        user_id,
        group_id,
        group_count = *group_count,
        "subscribe group success"
    );
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
        Err(err) => {
            tracing::debug!(
                user_id,
                source_id,
                error = %err,
                "subscribe user failed: invalid user id format"
            );
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
        Err(err) => {
            tracing::error!(
                user_id,
                target_user_id,
                error = %err,
                "subscribe user failed: query target user exists error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    if !exists {
        tracing::debug!(
            user_id,
            target_user_id,
            "subscribe user failed: target user not found"
        );
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::InvalidTopic,
        ));
    }

    let blocked_by_target = match blocks_repo.is_blocked(target_user_id, user_id).await {
        Ok(v) => v,
        Err(err) => {
            tracing::error!(
                user_id,
                target_user_id,
                error = %err,
                "subscribe user failed: query blocked_by_target error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    let blocked_by_self = match blocks_repo.is_blocked(user_id, target_user_id).await {
        Ok(v) => v,
        Err(err) => {
            tracing::error!(
                user_id,
                target_user_id,
                error = %err,
                "subscribe user failed: query blocked_by_self error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };
    let blocked = blocked_by_target || blocked_by_self;
    if blocked {
        tracing::warn!(
            user_id,
            target_user_id,
            blocked_by_target,
            blocked_by_self,
            "subscribe user denied: blocked relation exists"
        );
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::Forbidden,
        ));
    }
    let settings_repo = UserSettingsRepository::new(state.db.clone());
    let presence_visible = match settings_repo.find_by_user_id(target_user_id).await {
        Ok(settings) => settings
            .and_then(|row| {
                row.privacy_settings
                    .get("presence_visible")
                    .and_then(|value| value.as_bool())
            })
            .unwrap_or(true),
        Err(err) => {
            tracing::error!(
                user_id,
                target_user_id,
                error = %err,
                "subscribe user failed: query privacy settings error"
            );
            return Some(build_failed_id(
                source_id,
                SubscriptionFailureReason::InternalError,
            ));
        }
    };

    if target_user_id != user_id && !presence_visible {
        tracing::warn!(
            user_id,
            target_user_id,
            "subscribe user denied: target presence not visible"
        );
        return Some(build_failed_id(
            source_id,
            SubscriptionFailureReason::Forbidden,
        ));
    }
    if *user_count >= MAX_USER_SUBSCRIPTION_SOURCE_IDS {
        tracing::warn!(
            user_id,
            target_user_id,
            user_count = *user_count,
            limit = MAX_USER_SUBSCRIPTION_SOURCE_IDS,
            "subscribe user denied: user subscription limit exceeded"
        );
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
        tracing::debug!(
            user_id,
            target_user_id,
            "subscribe user skipped: already subscribed"
        );
        return None;
    }

    state.ws_subscription.subscribe(user_id, subscription).await;
    *user_count += 1;
    tracing::debug!(
        user_id,
        target_user_id,
        user_count = *user_count,
        "subscribe user success"
    );

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
            if let Err(err) = state.ws_manager.send_to_user(user_id, &msg).await {
                tracing::warn!(
                    user_id,
                    target_user_id,
                    error = %err,
                    "subscribe user: failed to send initial online snapshot"
                );
            }
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
    tracing::debug!(
        user_id,
        source_type,
        ids_count = ids.len(),
        "processing websocket unsubscribe request"
    );
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
                    Err(err) => {
                        tracing::debug!(
                            user_id,
                            source_id,
                            error = %err,
                            "unsubscribe user skipped: invalid user id format"
                        );
                        continue;
                    }
                };
                let subscription = SubscriptionType::User(target_user_id);
                state
                    .ws_subscription
                    .unsubscribe(user_id, &subscription)
                    .await;
            }
            _ => {
                tracing::warn!(
                    user_id,
                    source_type,
                    source_id,
                    "unsubscribe ignored: invalid source type"
                );
            }
        }
    }
}

/// Clear all subscriptions of a given type for a user.
pub async fn clear_subscriptions_by_type(user_id: i64, source_type: &str, state: &AppState) {
    tracing::debug!(
        user_id,
        source_type,
        "clearing websocket subscriptions by type"
    );
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
    let max_ids = match source_type {
        "group" => MAX_GROUP_SUBSCRIPTION_SOURCE_IDS,
        "user" => MAX_USER_SUBSCRIPTION_SOURCE_IDS,
        _ => {
            tracing::warn!(
                source_type,
                ids_count = ids.len(),
                "subscription request validation failed: invalid source type"
            );
            return Err(AppError::InvalidSubscriptionFormat(
                "订阅类型无效".to_string(),
            ));
        }
    };
    if !allow_empty && ids.is_empty() {
        tracing::warn!(
            source_type,
            allow_empty,
            "subscription request validation failed: ids empty"
        );
        return Err(AppError::InvalidSubscriptionFormat(
            "订阅ID不能为空".to_string(),
        ));
    }
    if ids.len() > max_ids {
        tracing::warn!(
            source_type,
            ids_count = ids.len(),
            limit = max_ids,
            "subscription request validation failed: ids exceed limit"
        );
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
    let limited = match limiter
        .is_rate_limited(&key, max_requests, window_secs)
        .await
    {
        Ok(value) => value,
        Err(err) => {
            tracing::warn!(
                user_id,
                error = %err,
                "websocket subscribe rate limit check failed, fallback not limited"
            );
            false
        }
    };
    if limited {
        let remaining = match limiter.get_remaining(&key, max_requests, window_secs).await {
            Ok(value) => value,
            Err(err) => {
                tracing::warn!(
                    user_id,
                    error = %err,
                    "websocket subscribe rate limit remaining check failed"
                );
                0
            }
        };
        tracing::debug!(user_id, remaining, "websocket subscribe rate limited");
    }
    limited
}
