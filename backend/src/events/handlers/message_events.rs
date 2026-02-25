use super::get_user_info;
use crate::{
    events::event_types::AppEvent,
    repositories::user_repo::UserRepository,
    websocket::{
        manager::ConnectionManager as WsConnectionManager, subscription::SubscriptionManager,
        types::WsMessage,
    },
};

pub async fn persist_notification(_event: &AppEvent) -> Result<(), Box<dyn std::error::Error>> {
    // 消息事件无持久化通知
    Ok(())
}

pub async fn send_websocket(
    event: &AppEvent,
    ws_manager: &WsConnectionManager,
    ws_subscription: &SubscriptionManager,
    user_repo: &UserRepository,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::GroupMessageSent {
            message_id,
            group_id,
            sender_id,
            message_type,
            content,
            share_code,
            reply_to,
            mentions,
            media,
        } => {
            let sender = get_user_info(user_repo, *sender_id).await;
            let message = WsMessage::GroupMessageNew {
                message_id: message_id.to_string(),
                group_id: group_id.to_string(),
                sender,
                message_type: message_type.clone(),
                content: content.clone(),
                share_code: share_code.clone(),
                media: media.clone(),
                reply_to: reply_to
                    .clone()
                    .map(|r| crate::dto::response::MessageReplyTo {
                        message_id: r.message_id.to_string(),
                        user_id: r.user_id,
                        nickname: r.nickname,
                        content: r.content,
                        share_code: r.share_code,
                    }),
                mentions: mentions.clone(),
                sent_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupMessageEdited {
            message_id,
            group_id,
            content,
            edited_at,
        } => {
            let message = WsMessage::GroupMessageEdited {
                message_id: message_id.to_string(),
                group_id: group_id.to_string(),
                content: content.clone(),
                edited_at: edited_at.clone(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupMessageRecalled {
            message_id,
            group_id,
            recalled_by,
            recalled_at,
        } => {
            let recalled_by_user = get_user_info(user_repo, *recalled_by).await;
            let message = WsMessage::GroupMessageRecalled {
                message_id: message_id.to_string(),
                group_id: group_id.to_string(),
                recalled_by: recalled_by_user,
                recalled_at: recalled_at.clone(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupMessageDeleted { .. } => {}
        AppEvent::GroupMessageRead { .. } => {}
        AppEvent::GroupUserTyping {
            group_id,
            user_id,
            is_typing,
        } => {
            let message = WsMessage::GroupUserTyping {
                group_id: group_id.to_string(),
                user_id: *user_id,
                is_typing: *is_typing,
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            let filtered_subscribers: Vec<i64> = subscribers
                .into_iter()
                .filter(|&id| id != *user_id)
                .collect();
            ws_manager
                .send_to_users(&filtered_subscribers, &message)
                .await;
        }
        AppEvent::PrivateMessageSent {
            message_id,
            sender_id,
            target_user_id,
            message_type,
            content,
            share_code,
            reply_to,
            media,
        } => {
            let sender = get_user_info(user_repo, *sender_id).await;
            let message = WsMessage::PrivateMessageNew {
                message_id: message_id.to_string(),
                target_user_id: *target_user_id,
                sender,
                message_type: message_type.clone(),
                content: content.clone(),
                share_code: share_code.clone(),
                media: media.clone(),
                reply_to: reply_to
                    .clone()
                    .map(|r| crate::dto::response::MessageReplyTo {
                        message_id: r.message_id.to_string(),
                        user_id: r.user_id,
                        nickname: r.nickname,
                        content: r.content,
                        share_code: r.share_code,
                    }),
                sent_at: chrono::Utc::now().to_rfc3339(),
            };
            ws_manager
                .send_to_users(&[*target_user_id, *sender_id], &message)
                .await;
        }
        AppEvent::PrivateMessageEdited {
            message_id,
            sender_id,
            target_user_id,
            content,
            edited_at,
        } => {
            let message = WsMessage::PrivateMessageEdited {
                message_id: message_id.to_string(),
                target_user_id: *target_user_id,
                content: content.clone(),
                edited_at: edited_at.clone(),
            };
            let _ = ws_manager.send_to_user(*target_user_id, &message).await;
            let _ = ws_manager.send_to_user(*sender_id, &message).await;
        }
        AppEvent::PrivateMessageRecalled {
            message_id,
            sender_id,
            target_user_id,
            recalled_by,
            recalled_at,
        } => {
            let recalled_by_info = get_user_info(user_repo, *recalled_by).await;
            let message = WsMessage::PrivateMessageRecalled {
                message_id: message_id.to_string(),
                target_user_id: *target_user_id,
                recalled_by: recalled_by_info,
                recalled_at: recalled_at.clone(),
            };
            let _ = ws_manager.send_to_user(*target_user_id, &message).await;
            let _ = ws_manager.send_to_user(*sender_id, &message).await;
        }
        AppEvent::PrivateMessageDeleted { .. } => {}
        AppEvent::PrivateMessageRead {
            target_user_id,
            reader_id,
            up_to_message_id,
            read_at,
        } => {
            let message = WsMessage::PrivateMessageRead {
                target_user_id: *target_user_id,
                up_to_message_id: up_to_message_id.to_string(),
                reader_id: *reader_id,
                read_at: read_at.clone(),
            };
            let _ = ws_manager.send_to_user(*target_user_id, &message).await;
            let _ = ws_manager.send_to_user(*reader_id, &message).await;
        }
        AppEvent::PrivateUserTyping {
            target_user_id,
            user_id,
            is_typing,
            ..
        } => {
            let message = WsMessage::PrivateUserTyping {
                target_user_id: *target_user_id,
                user_id: *user_id,
                is_typing: *is_typing,
            };
            let _ = ws_manager.send_to_user(*target_user_id, &message).await;
        }
        AppEvent::SelfJoinedGroup { .. } => {}
        AppEvent::SelfLeftGroup { .. } => {}
        AppEvent::PrivateChatDeleted { .. } => {}
        AppEvent::PrivateChatSettingsUpdated { .. } => {}
        AppEvent::NotificationReadStateUpdated { .. } => {}
        _ => {}
    }
    Ok(())
}
