use crate::{
    events::event_types::AppEvent,
    repositories::{notification_repo::NotificationType, user_repo::UserRepository},
    services::{
        notification_service::NotificationService, subscription_service::SubscriptionService,
    },
    websocket::{manager::ConnectionManager as WsConnectionManager, types::WsMessage},
};

pub async fn persist_notification(
    event: &AppEvent,
    notification_service: &NotificationService,
    user_repo: &UserRepository,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::SystemAnnouncement {
            announcement_id,
            title,
            content,
            severity,
            action_url,
            ..
        } => {
            let extra = Some(serde_json::json!({ "severity": severity.as_str() }));
            let mut last_user_id = 0;
            loop {
                let user_ids = user_repo.list_user_ids_after(last_user_id, 1000).await?;
                if user_ids.is_empty() {
                    break;
                }
                for uid in &user_ids {
                    notification_service
                        .create_notification(
                            *uid,
                            NotificationType::SystemAnnouncement,
                            title.clone(),
                            content.clone(),
                            None,
                            action_url.as_ref().map(|_| "web".to_string()),
                            action_url.clone(),
                            Some(announcement_id.clone()),
                            None,
                            extra.clone(),
                        )
                        .await?;
                }
                if let Some(last) = user_ids.last() {
                    last_user_id = *last;
                } else {
                    break;
                }
            }
        }
        AppEvent::SystemNotification {
            user_id,
            title,
            content,
        } => {
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::SystemAnnouncement,
                    title.clone(),
                    content.clone(),
                    None,
                    None,
                    None,
                    None,
                    None,
                    Some(serde_json::json!({ "severity": "info" })),
                )
                .await?;
        }
        _ => {}
    }
    Ok(())
}

pub async fn send_websocket(
    event: &AppEvent,
    ws_manager: &WsConnectionManager,
    subscription_service: &SubscriptionService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::SystemNotification { .. } => {}
        AppEvent::SystemAnnouncement {
            announcement_id,
            title,
            content,
            severity,
            action_url,
            created_at,
        } => {
            let message = WsMessage::SystemAnnouncement {
                announcement_id: announcement_id.clone(),
                title: title.clone(),
                content: content.clone(),
                severity: *severity,
                action_url: action_url.clone(),
                created_at: created_at.clone(),
            };
            let online_users = ws_manager.get_online_user_ids().await;
            for uid in online_users {
                let cfg = subscription_service.get_config(uid).await?;
                if cfg.system_announcement_enabled {
                    let _ = ws_manager.send_to_user(uid, &message).await;
                }
            }
        }
        AppEvent::AccountBanned {
            user_id,
            reason,
            duration_days,
        } => {
            let message = WsMessage::AccountBanned {
                reason: reason.clone(),
                duration_days: *duration_days,
                banned_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
            ws_manager.disconnect_user(*user_id).await;
        }
        _ => {}
    }
    Ok(())
}
