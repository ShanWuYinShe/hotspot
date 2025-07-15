//! 群组事件通知服务
//! 
//! 这个服务专门处理群组相关的事件通知，包括：
//! - 新成员加入群组
//! - 成员离开群组  
//! - 群组消息通知
//! - 群组设置变更通知
//! 
//! 与用户状态通知不同，群组事件通知是基于事件驱动的，
//! 只在特定的群组事件发生时才向群组成员发送通知。

use std::sync::Arc;
use tracing::{info, warn};
use uuid::Uuid;

use crate::repositories::{PgUserRepository, PgGroupRepository, GroupRepository};

use crate::ws::message_router::MessageRouter;
use crate::ws::message_types::{ServerMessage, GroupEventData, GroupEventType, SystemNotificationData, SystemNotificationType, RoomId};
use crate::services::notification::{GroupNotificationPolicy, NotificationError};

/// 群组事件类型
#[derive(Debug, Clone)]
#[allow(dead_code)]
pub enum GroupEvent {
    /// 成员加入群组
    MemberJoined {
        group_id: Uuid,
        user_id: i64,
        user_nickname: String,
    },
    /// 成员离开群组
    MemberLeft {
        group_id: Uuid,
        user_id: i64,
        user_nickname: String,
    },
    /// 群组消息
    GroupMessage {
        group_id: Uuid,
        sender_id: i64,
        message_id: String,
    },
    /// 群组设置变更
    GroupSettingsChanged {
        group_id: Uuid,
        changed_by: i64,
        changes: Vec<String>,
    },
}

/// 群组通知服务
pub struct GroupNotificationService {
    #[allow(dead_code)]
    message_router: Arc<MessageRouter>,
    #[allow(dead_code)]
    user_repository: Arc<PgUserRepository>,
    #[allow(dead_code)]
    group_repository: Arc<PgGroupRepository>,
    #[allow(dead_code)]
    policy: GroupNotificationPolicy,
}

impl GroupNotificationService {
    pub fn new(
        message_router: Arc<MessageRouter>,
        user_repository: Arc<PgUserRepository>,
        group_repository: Arc<PgGroupRepository>,
        policy: Option<GroupNotificationPolicy>,
    ) -> Self {
        Self {
            message_router,
            user_repository,
            group_repository,
            policy: policy.unwrap_or_default(),
        }
    }

    /// 发送群组事件通知
    #[allow(dead_code)]
    pub async fn notify_group_event(&self, event: GroupEvent) -> Result<(), NotificationError> {
        match event {
            GroupEvent::MemberJoined { group_id, user_id, user_nickname } => {
                if self.policy.enable_member_join_leave_notifications {
                    self.notify_member_joined(group_id, user_id, &user_nickname).await?;
                }
            }
            GroupEvent::MemberLeft { group_id, user_id, user_nickname } => {
                if self.policy.enable_member_join_leave_notifications {
                    self.notify_member_left(group_id, user_id, &user_nickname).await?;
                }
            }
            GroupEvent::GroupMessage { group_id, sender_id, message_id } => {
                if self.policy.enable_group_message_notifications {
                    self.notify_group_message(group_id, sender_id, &message_id).await?;
                }
            }
            GroupEvent::GroupSettingsChanged { group_id, changed_by, changes } => {
                if self.policy.enable_group_settings_notifications {
                    self.notify_group_settings_changed(group_id, changed_by, changes).await?;
                }
            }
        }
        Ok(())
    }

    /// 通知群组成员有新成员加入
    #[allow(dead_code)]
    async fn notify_member_joined(
        &self,
        group_id: Uuid,
        new_user_id: i64,
        user_nickname: &str,
    ) -> Result<(), NotificationError> {
        // 暂时使用简化的逻辑，获取群组成员
        let (group_members, _, _) = self.group_repository
            .get_group_members(group_id, 0, 1000) // 获取前1000个成员
            .await
            .map_err(NotificationError::Repository)?;

        let group_event_data = GroupEventData {
            group_id,
            event_type: GroupEventType::MemberJoined,
            actor_id: new_user_id,
            actor_nickname: user_nickname.to_string(),
            target_id: None,
            target_nickname: None,
            created_at: chrono::Utc::now(),
        };

        let message = ServerMessage::GroupEvent(group_event_data);

        let mut notification_count = 0;
        for (_, member_id, _, _, _, _) in group_members {
            // 不需要通知新加入的用户自己
            if member_id == new_user_id {
                continue;
            }

            let user_room = RoomId::from_user(member_id);
            if let Err(e) = self.message_router.send_to_room(&user_room, message.clone()).await {
                warn!("发送群组成员加入通知失败 - 目标用户: {}, 错误: {:?}", member_id, e);
            } else {
                notification_count += 1;
            }
        }

        info!("群组 {} 新成员加入通知已发送给 {} 个成员", group_id, notification_count);
        Ok(())
    }

    /// 通知群组成员有成员离开
    #[allow(dead_code)]
    async fn notify_member_left(
        &self,
        group_id: Uuid,
        left_user_id: i64,
        user_nickname: &str,
    ) -> Result<(), NotificationError> {
        let (group_members, _, _) = self.group_repository
            .get_group_members(group_id, 0, 1000) // 获取前1000个成员
            .await
            .map_err(NotificationError::Repository)?;

        let group_event_data = GroupEventData {
            group_id,
            event_type: GroupEventType::MemberLeft,
            actor_id: left_user_id,
            actor_nickname: user_nickname.to_string(),
            target_id: None,
            target_nickname: None,
            created_at: chrono::Utc::now(),
        };

        let message = ServerMessage::GroupEvent(group_event_data);

        let mut notification_count = 0;
        for (_, member_id, _, _, _, _) in group_members {
            let user_room = RoomId::from_user(member_id);
            if let Err(e) = self.message_router.send_to_room(&user_room, message.clone()).await {
                warn!("发送群组成员离开通知失败 - 目标用户: {}, 错误: {:?}", member_id, e);
            } else {
                notification_count += 1;
            }
        }

        info!("群组 {} 成员离开通知已发送给 {} 个成员", group_id, notification_count);
        Ok(())
    }

    /// 通知群组消息
    #[allow(dead_code)]
    async fn notify_group_message(
        &self,
        group_id: Uuid,
        sender_id: i64,
        message_id: &str,
    ) -> Result<(), NotificationError> {
        let (group_members, _, _) = self.group_repository
            .get_group_members(group_id, 0, 1000) // 获取前1000个成员
            .await
            .map_err(NotificationError::Repository)?;

        let notification_data = SystemNotificationData {
            notification_id: uuid::Uuid::new_v4(),
            notification_type: SystemNotificationType::GroupDigest,
            title: "新群组消息".to_string(),
            content: format!("群组中有新消息"),
            data: Some(serde_json::json!({
                "group_id": group_id,
                "sender_id": sender_id,
                "message_id": message_id
            })),
            created_at: chrono::Utc::now(),
        };

        let message = ServerMessage::SystemNotification(notification_data);

        let mut notification_count = 0;
        for (_, member_id, _, _, _, _) in group_members {
            // 不需要通知发送者自己
            if member_id == sender_id {
                continue;
            }

            let user_room = RoomId::from_user(member_id);
            if let Err(e) = self.message_router.send_to_room(&user_room, message.clone()).await {
                warn!("发送群组消息通知失败 - 目标用户: {}, 错误: {:?}", member_id, e);
            } else {
                notification_count += 1;
            }
        }

        info!("群组 {} 消息通知已发送给 {} 个成员", group_id, notification_count);
        Ok(())
    }

    /// 通知群组设置变更
    #[allow(dead_code)]
    async fn notify_group_settings_changed(
        &self,
        group_id: Uuid,
        changed_by: i64,
        _changes: Vec<String>,
    ) -> Result<(), NotificationError> {
        let (group_members, _, _) = self.group_repository
            .get_group_members(group_id, 0, 1000) // 获取前1000个成员
            .await
            .map_err(NotificationError::Repository)?;

        let group_event_data = GroupEventData {
            group_id,
            event_type: GroupEventType::GroupUpdated,
            actor_id: changed_by,
            actor_nickname: "".to_string(), // 可以根据需要获取用户昵称
            target_id: None,
            target_nickname: None,
            created_at: chrono::Utc::now(),
        };

        let message = ServerMessage::GroupEvent(group_event_data);

        let mut notification_count = 0;
        for (_, member_id, _, _, _, _) in group_members {
            let user_room = RoomId::from_user(member_id);
            if let Err(e) = self.message_router.send_to_room(&user_room, message.clone()).await {
                warn!("发送群组设置变更通知失败 - 目标用户: {}, 错误: {:?}", member_id, e);
            } else {
                notification_count += 1;
            }
        }

        info!("群组 {} 设置变更通知已发送给 {} 个成员", group_id, notification_count);
        Ok(())
    }

    /// 更新群组通知策略
    #[allow(dead_code)]
    pub fn update_policy(&mut self, new_policy: GroupNotificationPolicy) {
        self.policy = new_policy;
        info!("群组通知策略已更新: {:?}", self.policy);
    }

    /// 获取当前策略
    #[allow(dead_code)]
    pub fn get_policy(&self) -> &GroupNotificationPolicy {
        &self.policy
    }
}
