use super::get_user_info;
use crate::{
    dto::response::MomentSummary,
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
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::MomentLiked {
            moment_id,
            moment_summary,
            user_id,
            liker_id,
        } => {
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::MomentLiked,
                    "瞬间获赞".to_string(),
                    "点赞了你的瞬间".to_string(),
                    Some(*liker_id),
                    Some("moment".to_string()),
                    Some(moment_id.to_string()),
                    None,
                    Some(
                        moment_summary
                            .cover_url
                            .clone()
                            .unwrap_or(moment_summary.description_preview.clone()),
                    ),
                    None,
                )
                .await?;
        }
        AppEvent::MomentCommented {
            moment_id,
            moment_summary,
            comment_id,
            user_id,
            commenter_id,
            commenter_nickname: _,
            commenter_avatar_url: _,
            content,
        } => {
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::MomentCommented,
                    "瞬间评论".to_string(),
                    content.clone(),
                    Some(*commenter_id),
                    Some("moment".to_string()),
                    Some(moment_id.to_string()),
                    Some(comment_id.to_string()),
                    Some(
                        moment_summary
                            .cover_url
                            .clone()
                            .unwrap_or(moment_summary.description_preview.clone()),
                    ),
                    None,
                )
                .await?;
        }
        AppEvent::CommentReplied {
            moment_id,
            moment_summary: _,
            comment_id,
            replied_comment_preview,
            reply_id,
            user_id,
            replier_id,
            content,
        } => {
            notification_service
                .create_notification(
                    *user_id,
                    NotificationType::CommentReply,
                    "收到回复".to_string(),
                    content.clone(),
                    Some(*replier_id),
                    Some("moment".to_string()),
                    Some(moment_id.to_string()),
                    Some(reply_id.to_string()),
                    Some(replied_comment_preview.clone()),
                    Some(serde_json::json!({ "comment_id": comment_id.to_string() })),
                )
                .await?;
        }
        AppEvent::CommentLiked {
            comment_id,
            moment_id,
            moment_summary: _,
            user_id,
            liker_id,
            comment_preview,
        } => {
            if *user_id != *liker_id {
                notification_service
                    .create_notification(
                        *user_id,
                        NotificationType::CommentLiked,
                        "评论获赞".to_string(),
                        "点赞了你的评论".to_string(),
                        Some(*liker_id),
                        Some("comment".to_string()),
                        Some(moment_id.to_string()),
                        Some(comment_id.to_string()),
                        Some(comment_preview.clone()),
                        None,
                    )
                    .await?;
            }
        }
        AppEvent::MomentMentioned {
            moment_id,
            mentioned_user_id,
            mentioning_user_id,
            moment_description_preview,
            moment_cover_url,
        } => {
            let preview = moment_cover_url
                .clone()
                .unwrap_or_else(|| moment_description_preview.clone());
            notification_service
                .create_notification(
                    *mentioned_user_id,
                    NotificationType::Mentioned,
                    "被提及".to_string(),
                    "在瞬间中提到了你".to_string(),
                    Some(*mentioning_user_id),
                    Some("moment".to_string()),
                    Some(moment_id.to_string()),
                    None,
                    Some(preview),
                    None,
                )
                .await?;
        }
        AppEvent::CommentMentioned {
            moment_id,
            moment_summary: _,
            comment_id,
            mentioned_user_id,
            mentioning_user_id,
            content_preview,
        } => {
            notification_service
                .create_notification(
                    *mentioned_user_id,
                    NotificationType::Mentioned,
                    "被提及".to_string(),
                    "在评论中提到了你".to_string(),
                    Some(*mentioning_user_id),
                    Some("comment".to_string()),
                    Some(moment_id.to_string()),
                    Some(comment_id.to_string()),
                    Some(content_preview.clone()),
                    None,
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
    user_repo: &UserRepository,
    subscription_service: &SubscriptionService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::MomentCreated { moment_id, user_id } => {
            tracing::debug!("瞬间已创建: moment_id={}, user_id={}", moment_id, user_id);
        }
        AppEvent::MomentLiked {
            moment_id,
            moment_summary,
            user_id,
            liker_id,
        } => {
            let cfg = subscription_service.get_config(*user_id).await?;
            if !cfg.moment_like_enabled {
                return Ok(());
            }
            let liker = get_user_info(user_repo, *liker_id).await;
            let message = WsMessage::MomentLiked {
                moment_id: moment_id.to_string(),
                moment_summary: MomentSummary {
                    moment_id: moment_summary.moment_id.to_string(),
                    cover_url: moment_summary.cover_url.clone(),
                    description_preview: moment_summary.description_preview.clone(),
                },
                actor: liker,
                liked_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::MomentCommented {
            moment_id,
            moment_summary,
            comment_id,
            user_id,
            commenter_id,
            commenter_nickname,
            commenter_avatar_url,
            content,
        } => {
            let cfg = subscription_service.get_config(*user_id).await?;
            if !cfg.moment_comment_enabled {
                return Ok(());
            }
            let commenter = match commenter_nickname.as_deref() {
                Some(nick) => crate::dto::response::UserInfo::new(
                    *commenter_id,
                    nick.to_string(),
                    commenter_avatar_url.clone(),
                ),
                None => get_user_info(user_repo, *commenter_id).await,
            };
            let message = WsMessage::MomentCommented {
                moment_id: moment_id.to_string(),
                moment_summary: MomentSummary {
                    moment_id: moment_summary.moment_id.to_string(),
                    cover_url: moment_summary.cover_url.clone(),
                    description_preview: moment_summary.description_preview.clone(),
                },
                comment_id: comment_id.to_string(),
                actor: commenter,
                content: content.clone(),
                commented_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::CommentAdded {
            comment_id,
            target_type,
            target_id,
            user_id,
        } => {
            tracing::debug!(
                "新评论: comment_id={}, target_type={}, target_id={}, user_id={}",
                comment_id,
                target_type,
                target_id,
                user_id
            );
        }
        AppEvent::CommentLiked {
            comment_id,
            moment_id,
            moment_summary,
            user_id,
            liker_id,
            comment_preview,
        } => {
            if user_id == liker_id {
                return Ok(());
            }
            let liker = get_user_info(user_repo, *liker_id).await;
            let message = WsMessage::CommentLiked {
                comment_id: comment_id.to_string(),
                moment_id: moment_id.to_string(),
                moment_summary: MomentSummary {
                    moment_id: moment_summary.moment_id.to_string(),
                    cover_url: moment_summary.cover_url.clone(),
                    description_preview: moment_summary.description_preview.clone(),
                },
                comment_preview: comment_preview.clone(),
                actor: liker,
                liked_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::CommentReplied {
            moment_id,
            moment_summary,
            comment_id,
            replied_comment_preview,
            reply_id,
            user_id,
            replier_id,
            content,
        } => {
            let cfg = subscription_service.get_config(*user_id).await?;
            if !cfg.comment_reply_enabled {
                return Ok(());
            }
            let replier = get_user_info(user_repo, *replier_id).await;
            let message = WsMessage::CommentReply {
                moment_id: moment_id.to_string(),
                moment_summary: MomentSummary {
                    moment_id: moment_summary.moment_id.to_string(),
                    cover_url: moment_summary.cover_url.clone(),
                    description_preview: moment_summary.description_preview.clone(),
                },
                comment_id: comment_id.to_string(),
                replied_comment_preview: replied_comment_preview.clone(),
                reply_id: reply_id.to_string(),
                actor: replier,
                content: content.clone(),
                replied_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::MomentMentioned {
            moment_id,
            mentioned_user_id,
            mentioning_user_id,
            moment_description_preview,
            moment_cover_url,
        } => {
            let cfg = subscription_service.get_config(*mentioned_user_id).await?;
            if !cfg.message_mention_enabled {
                return Ok(());
            }
            let actor = get_user_info(user_repo, *mentioning_user_id).await;
            let message = WsMessage::MessageMentioned {
                target_type: "moment".to_string(),
                moment_id: moment_id.to_string(),
                comment_id: None,
                moment_summary: MomentSummary {
                    moment_id: moment_id.to_string(),
                    cover_url: moment_cover_url.clone(),
                    description_preview: moment_description_preview.clone(),
                },
                comment_preview: None,
                actor,
                mentioned_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*mentioned_user_id, &message).await;
        }
        AppEvent::CommentMentioned {
            comment_id,
            mentioned_user_id,
            moment_id,
            moment_summary,
            mentioning_user_id,
            content_preview,
        } => {
            let cfg = subscription_service.get_config(*mentioned_user_id).await?;
            if !cfg.message_mention_enabled {
                return Ok(());
            }
            let actor = get_user_info(user_repo, *mentioning_user_id).await;
            let message = WsMessage::MessageMentioned {
                target_type: "comment".to_string(),
                moment_id: moment_id.to_string(),
                comment_id: Some(comment_id.to_string()),
                moment_summary: MomentSummary {
                    moment_id: moment_summary.moment_id.to_string(),
                    cover_url: moment_summary.cover_url.clone(),
                    description_preview: moment_summary.description_preview.clone(),
                },
                comment_preview: Some(content_preview.clone()),
                actor,
                mentioned_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*mentioned_user_id, &message).await;
        }
        _ => {}
    }
    Ok(())
}
