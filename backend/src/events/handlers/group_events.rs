use super::get_user_info;
use crate::{
    dto::response::GroupInfo,
    events::event_types::AppEvent,
    repositories::{
        group_member_repo::GroupMemberRepository, notification_repo::NotificationType,
        user_repo::UserRepository,
    },
    services::{
        notification_service::NotificationService, subscription_service::SubscriptionService,
    },
    types::enums::{GroupDismissReason, MemberLeaveReason, MemberRemoveReason},
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        subscription::SubscriptionManager,
        types::{GroupUpdatedFields, WsMessage},
    },
};

pub async fn persist_notification(
    event: &AppEvent,
    notification_service: &NotificationService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::GroupInviteSent {
            invite_id,
            group_id,
            group_name,
            from_user_id,
            to_user_id,
            ..
        } => {
            notification_service
                .create_notification(
                    *to_user_id,
                    NotificationType::GroupInviteReceived,
                    "群组邀请".to_string(),
                    format!("邀请你加入群组: {}", group_name),
                    Some(*from_user_id),
                    Some("group".to_string()),
                    Some(group_id.to_string()),
                    Some(invite_id.to_string()),
                    None,
                    None,
                )
                .await?;
        }
        AppEvent::GroupInviteResponse { .. } => {
            // spec 已移除 group_invite_response 通知类型，不再持久化
        }
        AppEvent::GroupApplicationReceived { .. } => {
            // spec 已移除 group_application 通知类型，不再持久化
        }
        AppEvent::GroupApplicationResponse {
            application_id,
            group_id,
            group_name,
            group_avatar_url: _,
            members_count: _,
            applicant_id,
            status,
        } => {
            if status.as_str() != "accepted" {
                return Ok(());
            }
            notification_service
                .create_notification(
                    *applicant_id,
                    NotificationType::GroupApplicationAccepted,
                    "入群申请已通过".to_string(),
                    format!("已通过加入群组: {}", group_name),
                    None,
                    Some("group".to_string()),
                    Some(group_id.to_string()),
                    Some(application_id.to_string()),
                    None,
                    None,
                )
                .await?;
        }
        AppEvent::GroupAnnouncementCreated { .. } => {
            // spec 已移除 group_announcement 通知类型，不再持久化
        }
        _ => {}
    }
    Ok(())
}

#[allow(clippy::too_many_arguments)]
pub async fn send_websocket(
    event: &AppEvent,
    ws_manager: &WsConnectionManager,
    ws_subscription: &SubscriptionManager,
    user_repo: &UserRepository,
    group_member_repo: &GroupMemberRepository,
    subscription_service: &SubscriptionService,
) -> Result<(), Box<dyn std::error::Error>> {
    match event {
        AppEvent::GroupCreated {
            group_id,
            creator_id,
        } => {
            tracing::debug!(
                "群组已创建: group_id={}, creator_id={}",
                group_id,
                creator_id
            );
        }
        AppEvent::GroupMemberAdded {
            group_id,
            user_id,
            role,
            member_count,
            ..
        } => {
            let user = get_user_info(user_repo, *user_id).await;
            let message = WsMessage::GroupMemberJoined {
                group_id: group_id.to_string(),
                user,
                role: role.clone(),
                member_count: *member_count as i32,
                joined_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupMemberRemoved {
            group_id,
            user_id,
            removed_by,
            member_count,
            reason,
        } => {
            let user = get_user_info(user_repo, *user_id).await;
            let group_id_str = group_id.to_string();
            if *reason == MemberLeaveReason::Quit {
                let message = WsMessage::GroupMemberLeft {
                    group_id: group_id_str.clone(),
                    user,
                    left_at: chrono::Utc::now().to_rfc3339(),
                    member_count: *member_count as i32,
                };
                let subscribers = ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                ws_manager.send_to_users(&subscribers, &message).await;
            } else {
                let remove_reason = match reason {
                    MemberLeaveReason::Kick => Some(MemberRemoveReason::Kicked),
                    MemberLeaveReason::Dismiss => None,
                    _ => Some(MemberRemoveReason::Banned),
                };
                let removed_by_info = get_user_info(user_repo, *removed_by).await;
                let message = WsMessage::GroupMemberRemoved {
                    group_id: group_id_str.clone(),
                    user,
                    removed_by: removed_by_info,
                    reason: remove_reason,
                    removed_at: chrono::Utc::now().to_rfc3339(),
                    member_count: *member_count as i32,
                };
                let subscribers = ws_subscription
                    .get_group_chat_subscribers(&group_id_str)
                    .await;
                ws_manager.send_to_users(&subscribers, &message).await;
                let _ = ws_manager.send_to_user(*user_id, &message).await;
            }
        }
        AppEvent::GroupProfileUpdated {
            group_id,
            updated_by,
            name,
            description,
            avatar_url,
            tags,
            join_method,
            is_searchable,
            max_members,
            has_password,
        } => {
            let updated_fields = GroupUpdatedFields {
                name: name.clone(),
                description: description.clone(),
                avatar_url: avatar_url.clone(),
                tags: tags.clone(),
                join_method: join_method
                    .as_deref()
                    .and_then(|s| s.parse().ok()),
                is_searchable: *is_searchable,
                max_members: *max_members,
                has_password: *has_password,
                ..Default::default()
            };
            let message = WsMessage::GroupUpdated {
                group_id: group_id.to_string(),
                updated_fields,
                updated_by: updated_by.to_string(),
                updated_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupMemberRoleChanged { .. } => {}
        AppEvent::GroupMemberMuted {
            group_id,
            user_id,
            user_nickname,
            user_avatar_url,
            muted_by,
            muted_until,
            duration_minutes,
        } => {
            let user = match user_nickname.as_deref() {
                Some(nick) => crate::dto::response::UserInfo::new(
                    *user_id,
                    nick.to_string(),
                    user_avatar_url.clone(),
                ),
                None => get_user_info(user_repo, *user_id).await,
            };
            let muted_by_info = get_user_info(user_repo, *muted_by).await;
            let message = WsMessage::GroupMemberMuted {
                group_id: group_id.to_string(),
                user,
                muted_by: muted_by_info,
                muted_until: muted_until.clone(),
                duration_minutes: *duration_minutes,
                muted_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::GroupMemberUnmuted {
            group_id,
            user_id,
            user_nickname,
            user_avatar_url,
            unmuted_by,
        } => {
            let user = match user_nickname.as_deref() {
                Some(nick) => crate::dto::response::UserInfo::new(
                    *user_id,
                    nick.to_string(),
                    user_avatar_url.clone(),
                ),
                None => get_user_info(user_repo, *user_id).await,
            };
            let unmuted_by_info = get_user_info(user_repo, *unmuted_by).await;
            let message = WsMessage::GroupMemberUnmuted {
                group_id: group_id.to_string(),
                user,
                unmuted_by: unmuted_by_info,
                unmuted_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
            let _ = ws_manager.send_to_user(*user_id, &message).await;
        }
        AppEvent::GroupDismissed {
            group_id,
            dismissed_by_owner,
            ..
        } => {
            let reason = Some(if *dismissed_by_owner {
                GroupDismissReason::OwnerDismissed
            } else {
                GroupDismissReason::PolicyViolation
            });
            let message = WsMessage::GroupDismissed {
                group_id: group_id.to_string(),
                dismissed_at: chrono::Utc::now().to_rfc3339(),
                reason,
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            ws_manager.send_to_users(&subscribers, &message).await;
        }
        AppEvent::GroupInviteSent {
            invite_id,
            group_id,
            group_name,
            group_avatar_url,
            members_count,
            from_user_id,
            to_user_id,
            message,
        } => {
            let cfg = subscription_service.get_config(*to_user_id).await?;
            if !cfg.group_invite_enabled {
                return Ok(());
            }
            let inviter = get_user_info(user_repo, *from_user_id).await;
            let ws_message = WsMessage::GroupInviteReceived {
                invite_id: invite_id.to_string(),
                group: GroupInfo {
                    group_id: group_id.to_string(),
                    name: group_name.clone(),
                    avatar_url: group_avatar_url.clone(),
                    members_count: *members_count,
                },
                inviter,
                message: message.clone(),
                created_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*to_user_id, &ws_message).await;
        }
        AppEvent::GroupInviteResponse {
            invite_id,
            group_id,
            inviter_id,
            invitee_id,
            status,
        } => {
            if status.as_str() != "accepted" {
                return Ok(());
            }
            let cfg = subscription_service.get_config(*inviter_id).await?;
            if !cfg.group_invite_response_enabled {
                return Ok(());
            }
            let user = get_user_info(user_repo, *invitee_id).await;
            let message = WsMessage::GroupInviteResponse {
                invite_id: invite_id.to_string(),
                group_id: group_id.to_string(),
                user,
                status: "accepted".to_string(),
                responded_at: chrono::Utc::now().to_rfc3339(),
            };
            let _ = ws_manager.send_to_user(*inviter_id, &message).await;
        }
        AppEvent::GroupApplicationReceived {
            application_id,
            group_id,
            applicant_id,
            message,
        } => {
            let applicant = get_user_info(user_repo, *applicant_id).await;
            let ws_message = WsMessage::GroupApplicationReceived {
                application_id: application_id.to_string(),
                group_id: group_id.to_string(),
                applicant,
                message: message.clone(),
                created_at: chrono::Utc::now().to_rfc3339(),
            };
            let admin_ids = group_member_repo.list_admin_user_ids(*group_id).await?;
            let mut recipients = Vec::with_capacity(admin_ids.len());
            for admin_id in admin_ids {
                let cfg = subscription_service.get_config(admin_id).await?;
                if !cfg.group_application_enabled {
                    continue;
                }
                recipients.push(admin_id);
            }
            ws_manager.send_to_users(&recipients, &ws_message).await;
        }
        AppEvent::GroupApplicationResponse {
            application_id,
            group_id,
            group_name,
            group_avatar_url,
            members_count,
            applicant_id,
            status,
        } => {
            if status.as_str() != "accepted" {
                return Ok(());
            }
            let message = WsMessage::GroupApplicationResponse {
                application_id: application_id.to_string(),
                group: GroupInfo {
                    group_id: group_id.to_string(),
                    name: group_name.clone(),
                    avatar_url: group_avatar_url.clone(),
                    members_count: *members_count,
                },
                status: "accepted".to_string(),
                responded_at: chrono::Utc::now().to_rfc3339(),
            };
            let cfg = subscription_service.get_config(*applicant_id).await?;
            if !cfg.group_application_response_enabled {
                return Ok(());
            }
            let _ = ws_manager.send_to_user(*applicant_id, &message).await;
        }
        AppEvent::GroupAnnouncementCreated {
            announcement_id,
            group_id,
            title,
            content,
            created_by_user_id,
            created_by_nickname,
            created_by_avatar_url,
            is_pinned,
        } => {
            let created_by = match created_by_nickname.as_deref() {
                Some(nick) => crate::dto::response::UserInfo::new(
                    *created_by_user_id,
                    nick.to_string(),
                    created_by_avatar_url.clone(),
                ),
                None => get_user_info(user_repo, *created_by_user_id).await,
            };
            let message = WsMessage::GroupAnnouncementCreated {
                announcement_id: announcement_id.to_string(),
                group_id: group_id.to_string(),
                title: title.clone(),
                content: content.clone(),
                created_by,
                is_pinned: *is_pinned,
                created_at: chrono::Utc::now().to_rfc3339(),
            };
            let group_id_str = group_id.to_string();
            let subscribers = ws_subscription
                .get_group_chat_subscribers(&group_id_str)
                .await;
            for uid in subscribers {
                let cfg = subscription_service.get_config(uid).await?;
                if cfg.group_announcement_enabled {
                    let _ = ws_manager.send_to_user(uid, &message).await;
                }
            }
        }
        AppEvent::GroupAnnouncementUpdated { .. } => {}
        AppEvent::GroupAnnouncementDeleted { .. } => {}
        _ => {}
    }
    Ok(())
}
