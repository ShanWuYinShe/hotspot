use super::get_user_info;
use crate::{
    events::event_types::AppEvent,
    repositories::{notification_repo::NotificationType, user_repo::UserRepository},
    services::{
        notification_service::NotificationService, subscription_service::SubscriptionService,
    },
    websocket::{
        manager::ConnectionManager as WsConnectionManager, subscription::SubscriptionManager,
        types::WsMessage,
    },
};

pub async fn persist_notification(
    event: &AppEvent,
    notification_service: &NotificationService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::NewFollower {
            user_id,
            follower_id,
            is_mutual,
            ..
        } => {
            if *is_mutual {
                return Ok(());
            }
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::NewFollower,
                    "新增关注".to_string(),
                    "关注了你".to_string(),
                    Some(*follower_id),
                    Some("user".to_string()),
                    Some(follower_id.to_string()),
                    None,
                    None,
                    None,
                )
                .await?;
        }
        AppEvent::MutualFollow {
            user_id,
            mutual_user_id,
        } => {
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::MutualFollow,
                    "互相关注".to_string(),
                    "与你成为好友".to_string(),
                    Some(*mutual_user_id),
                    Some("user".to_string()),
                    Some(mutual_user_id.to_string()),
                    None,
                    None,
                    None,
                )
                .await?;
            notification_service
                .create_notification(
                    *mutual_user_id,
                    NotificationType::MutualFollow,
                    "互相关注".to_string(),
                    "与你成为好友".to_string(),
                    Some(*user_id),
                    Some("user".to_string()),
                    Some(user_id.to_string()),
                    None,
                    None,
                    None,
                )
                .await?;
        }
        AppEvent::FollowRequestReceived { .. } => {
            // spec 已移除 follow_request 通知类型，不再持久化
        }
        AppEvent::FollowRequestHandled { .. } => {
            // spec 已移除 follow_request_handled 通知类型，不再持久化
        }
        _ => {}
    }
    Ok(())
}

pub async fn send_websocket(
    event: &AppEvent,
    ws_manager: &WsConnectionManager,
    ws_subscription: &SubscriptionManager,
    user_repo: &UserRepository,
    subscription_service: &SubscriptionService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::UserRegistered { .. } => {
            // 注册事件不需要 WebSocket 推送
        }
        AppEvent::UserProfileUpdated {
            user_id,
            nickname,
            avatar_url,
            ..
        } => {
            let user = crate::dto::response::UserInfo::new(
                *user_id,
                nickname
                    .clone()
                    .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                avatar_url.clone(),
            );
            let message = WsMessage::UserProfileUpdated {
                user,
                updated_at: chrono::Utc::now().to_rfc3339(),
            };
            let subscribers = ws_subscription.get_user_status_subscribers(*user_id).await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::UserLocationUpdated { .. } => {
            // spec 未定义 UserLocationUpdate WebSocket 事件，不推送
        }
        AppEvent::NewFollower {
            user_id,
            follower_id,
            is_mutual: _,
            source,
            source_moment_id,
        } => {
            let cfg = subscription_service.get_config(*user_id).await?;
            if !cfg.new_follower_enabled {
                return Ok(());
            }
            let follower_info = get_user_info(user_repo, *follower_id).await;
            let message = WsMessage::NewFollower {
                actor: follower_info,
                followed_at: chrono::Utc::now().to_rfc3339(),
                source: *source,
                source_moment_id: source_moment_id.clone(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::MutualFollow {
            user_id,
            mutual_user_id,
        } => {
            let user_info = get_user_info(user_repo, *user_id).await;
            let mutual_user_info = get_user_info(user_repo, *mutual_user_id).await;
            let now = chrono::Utc::now().to_rfc3339();
            let cfg_for_user = subscription_service.get_config(*user_id).await?;
            if cfg_for_user.mutual_follow_enabled {
                let message_for_user = WsMessage::MutualFollow {
                    actor: mutual_user_info.clone(),
                    mutual_at: now.clone(),
                };
                let _ = ws_manager.send_to_user(*user_id, &message_for_user).await;
            }
            let cfg_for_mutual = subscription_service.get_config(*mutual_user_id).await?;
            if cfg_for_mutual.mutual_follow_enabled {
                let message_for_mutual = WsMessage::MutualFollow {
                    actor: user_info,
                    mutual_at: now,
                };
                let _ = ws_manager
                    .send_to_user(*mutual_user_id, &message_for_mutual)
                    .await;
            }
        }
        AppEvent::FollowRequestReceived { .. } => {}
        AppEvent::FollowRequestHandled { .. } => {}
        _ => {}
    }
    Ok(())
}
