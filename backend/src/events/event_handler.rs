use redis::aio::ConnectionManager;
use sqlx::PgPool;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    events::events::AppEvent,
    repositories::{
        inbox_repo::InboxRepository,
        user_repo::UserRepository,
    },
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        types::WsMessage,
    },
    types::enums::{NotificationCategory, NotificationSeverity},
};

/// 事件处理器
pub struct EventHandler {
    db: PgPool,
    #[allow(dead_code)]
    redis: ConnectionManager,
    ws_manager: WsConnectionManager,
    inbox_repo: InboxRepository,
    user_repo: UserRepository,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        ws_manager: WsConnectionManager,
    ) -> Self {
        Self {
            inbox_repo: InboxRepository::new(db.clone()),
            user_repo: UserRepository::new(db.clone()),
            db,
            redis,
            ws_manager,
        }
    }

    /// 获取用户信息
    async fn get_user_info(&self, user_id: i64) -> crate::websocket::types::UserInfo {
        match self.user_repo.find_by_id(user_id).await {
            Ok(Some(user)) => crate::websocket::types::UserInfo {
                user_id: user.user_id.to_string(),
                nickname: user.nickname,
                avatar_url: user.avatar_url,
            },
            _ => crate::websocket::types::UserInfo {
                user_id: user_id.to_string(),
                nickname: "未知用户".to_string(),
                avatar_url: None,
            },
        }
    }

    /// 启动事件处理循环
    pub async fn start(self, mut receiver: mpsc::UnboundedReceiver<AppEvent>) {
        info!("Event handler started");

        while let Some(event) = receiver.recv().await {
            if let Err(e) = self.handle_event(event.clone()).await {
                error!("Failed to handle event {:?}: {}", event, e);
            }
        }

        info!("Event handler stopped");
    }

    /// 处理单个事件
    async fn handle_event(&self, event: AppEvent) -> Result<(), Box<dyn std::error::Error>> {
        info!("Handling event: {}", event.event_type());

        // 1. 创建收件箱项
        self.create_inbox_items(&event).await?;

        // 2. 发送WebSocket推送
        self.send_websocket_notifications(&event).await?;

        // 3. 更新排行榜（如果需要）
        self.update_leaderboard(&event).await?;

        Ok(())
    }

    /// 创建收件箱项
    async fn create_inbox_items(&self, event: &AppEvent) -> Result<(), Box<dyn std::error::Error>> {
        match event {
            AppEvent::FriendRequestSent { to_user_id, from_user_id } => {
                self.inbox_repo.create(
                    *to_user_id,
                    NotificationCategory::Social,
                    "friend_request".to_string(),
                    None,
                    "您收到了一个好友请求".to_string(),
                    Some(serde_json::json!({
                        "from_user_id": from_user_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::FriendRequestAccepted { from_user_id, to_user_id } => {
                self.inbox_repo.create(
                    *from_user_id,
                    NotificationCategory::Social,
                    "friend_request_accepted".to_string(),
                    None,
                    "您的好友请求已被接受".to_string(),
                    Some(serde_json::json!({
                        "user_id": to_user_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::GroupInviteSent { group_id, to_user_id, from_user_id } => {
                self.inbox_repo.create(
                    *to_user_id,
                    NotificationCategory::Social,
                    "group_invite".to_string(),
                    None,
                    "您收到了一个群组邀请".to_string(),
                    Some(serde_json::json!({
                        "group_id": group_id,
                        "from_user_id": from_user_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::MomentLiked { moment_id, user_id, liker_id } => {
                self.inbox_repo.create(
                    *user_id,
                    NotificationCategory::Content,
                    "moment_liked".to_string(),
                    None,
                    "有人点赞了您的瞬间".to_string(),
                    Some(serde_json::json!({
                        "moment_id": moment_id,
                        "liker_id": liker_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::MomentCommented { moment_id, user_id, commenter_id, .. } => {
                self.inbox_repo.create(
                    *user_id,
                    NotificationCategory::Content,
                    "moment_commented".to_string(),
                    None,
                    "有人评论了您的瞬间".to_string(),
                    Some(serde_json::json!({
                        "moment_id": moment_id,
                        "commenter_id": commenter_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::CommentLiked { comment_id, user_id, liker_id } => {
                self.inbox_repo.create(
                    *user_id,
                    NotificationCategory::Content,
                    "comment_liked".to_string(),
                    None,
                    "有人点赞了您的评论".to_string(),
                    Some(serde_json::json!({
                        "comment_id": comment_id,
                        "liker_id": liker_id,
                    })),
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            AppEvent::SystemNotification { user_id, title, content } => {
                self.inbox_repo.create(
                    *user_id,
                    NotificationCategory::System,
                    "system_notification".to_string(),
                    Some(title.clone()),
                    content.clone(),
                    None,
                    None,
                    Some(NotificationSeverity::Info),
                ).await?;
            }
            _ => {
                // 其他事件不需要创建收件箱项
            }
        }

        Ok(())
    }

    /// 发送WebSocket推送
    async fn send_websocket_notifications(&self, event: &AppEvent) -> Result<(), Box<dyn std::error::Error>> {
        match event {
            AppEvent::MessageSent { message_id, group_chat_id, sender_id, content, message_type } => {
                // 获取发送者信息
                let sender = self.get_user_info(*sender_id).await;
                
                let message = WsMessage::NewMessage {
                    id: message_id.to_string(),
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    sender,
                    message_type: message_type.clone(),
                    content: content.clone(),
                    reply_to: None,
                    mentions: None,
                    sent_at: chrono::Utc::now().to_rfc3339(),
                    latitude: None,
                    longitude: None,
                    location_name: None,
                    file_name: None,
                    file_size: None,
                    thumbnail_url: None,
                    duration: None,
                    system_event: None,
                    affected_users: None,
                };
                
                // 推送给订阅了该群聊的用户
                let channel = format!("group:{}", group_chat_id);
                let _ = self.ws_manager.broadcast_to_channel(&channel, &message).await;
            }
            AppEvent::MessageEdited { message_id, group_chat_id, content, edited_at } => {
                let message = WsMessage::MessageEdited {
                    id: message_id.to_string(),
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    content: content.clone(),
                    edited_at: edited_at.clone(),
                };
                let _ = self.ws_manager.broadcast(&message).await;
            }
            AppEvent::MessageRecalled { message_id, group_chat_id, recalled_by, recalled_at } => {
                let message = WsMessage::MessageRecalled {
                    id: message_id.to_string(),
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    recalled_by: recalled_by.to_string(),
                    recalled_at: recalled_at.clone(),
                };
                let _ = self.ws_manager.broadcast(&message).await;
            }
            AppEvent::MessageDeleted { message_id, group_chat_id, deleted_by, for_everyone } => {
                let message = WsMessage::MessageDeleted {
                    id: message_id.to_string(),
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    deleted_by: deleted_by.to_string(),
                    for_everyone: *for_everyone,
                    deleted_at: chrono::Utc::now().to_rfc3339(),
                };
                let _ = self.ws_manager.broadcast(&message).await;
            }
            AppEvent::MessageRead { group_chat_id, reader_id, up_to, read_at } => {
                let message = WsMessage::MessageRead {
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    up_to: up_to.to_string(),
                    reader_id: reader_id.to_string(),
                    read_at: read_at.clone(),
                };
                let _ = self.ws_manager.broadcast(&message).await;
            }
            AppEvent::MessageReaction { message_id, group_chat_id, user_id, reaction_type, action } => {
                // 获取用户信息
                let user = self.get_user_info(*user_id).await;
                
                let message = WsMessage::MessageReaction {
                    id: message_id.to_string(),
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    user,
                    reaction_type: reaction_type.clone(),
                    action: action.clone(),
                    created_at: chrono::Utc::now().to_rfc3339(),
                };
                let channel = format!("group:{}", group_chat_id);
                let _ = self.ws_manager.broadcast_to_channel(&channel, &message).await;
            }
            AppEvent::UserTyping { group_chat_id, user_id, is_typing } => {
                // 获取用户昵称
                let user_info = self.get_user_info(*user_id).await;
                
                let message = WsMessage::UserTyping {
                    target_id: group_chat_id.to_string(),
                    target_type: "group".to_string(),
                    user_id: user_id.to_string(),
                    nickname: user_info.nickname,
                    is_typing: *is_typing,
                };
                let _ = self.ws_manager.broadcast(&message).await;
            }
            AppEvent::FriendRequestSent { to_user_id, from_user_id } => {
                let message = WsMessage::FriendRequest {
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                let _ = self.ws_manager.send_to_user(*to_user_id, &message).await;
            }
            AppEvent::FriendRequestAccepted { from_user_id, to_user_id } => {
                let message = WsMessage::FriendRequestAccepted {
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                let _ = self.ws_manager.send_to_user(*from_user_id, &message).await;
            }
            AppEvent::GroupInviteSent { group_id, to_user_id, from_user_id } => {
                let message = WsMessage::GroupInvite {
                    group_id: *group_id,
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                let _ = self.ws_manager.send_to_user(*to_user_id, &message).await;
            }
            AppEvent::MomentCreated { moment_id, user_id } => {
                let message = WsMessage::NewMoment {
                    moment_id: *moment_id,
                    user_id: *user_id,
                };
                let _ = self.ws_manager.broadcast_to_channel("moment", &message).await;
            }
            AppEvent::MomentCommented { moment_id, comment_id, user_id, .. } => {
                let message = WsMessage::MomentComment {
                    moment_id: *moment_id,
                    comment_id: *comment_id,
                    user_id: *user_id,
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            AppEvent::SystemNotification { user_id, title, content } => {
                let message = WsMessage::SystemNotification {
                    title: title.clone(),
                    content: content.clone(),
                };
                let _ = self.ws_manager.send_to_user(*user_id, &message).await;
            }
            _ => {
                // 其他事件不需要WebSocket推送
            }
        }

        Ok(())
    }

    /// 更新排行榜
    async fn update_leaderboard(&self, event: &AppEvent) -> Result<(), Box<dyn std::error::Error>> {
        // 使用Redis更新排行榜分数
        match event {
            AppEvent::MessageSent { sender_id, .. } => {
                // 增加活跃度分数
                let mut conn = self.redis.clone();
                let _: () = redis::cmd("ZINCRBY")
                    .arg("leaderboard:active")
                    .arg(1.0)
                    .arg(sender_id.to_string())
                    .query_async(&mut conn)
                    .await?;
            }
            AppEvent::MomentCreated { user_id, .. } => {
                // 增加贡献度分数
                let mut conn = self.redis.clone();
                let _: () = redis::cmd("ZINCRBY")
                    .arg("leaderboard:contributor")
                    .arg(5.0)
                    .arg(user_id.to_string())
                    .query_async(&mut conn)
                    .await?;
            }
            AppEvent::MomentLiked { user_id, .. } => {
                // 增加人气分数
                let mut conn = self.redis.clone();
                let _: () = redis::cmd("ZINCRBY")
                    .arg("leaderboard:popular")
                    .arg(2.0)
                    .arg(user_id.to_string())
                    .query_async(&mut conn)
                    .await?;
            }
            _ => {
                // 其他事件不影响排行榜
            }
        }

        Ok(())
    }
}
