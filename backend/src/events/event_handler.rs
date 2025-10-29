use redis::aio::ConnectionManager;
use sqlx::PgPool;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    events::events::AppEvent,
    models::inbox::InboxItem,
    repositories::inbox_repo::InboxRepository,
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        types::WsMessage,
    },
};

/// 事件处理器
pub struct EventHandler {
    db: PgPool,
    redis: ConnectionManager,
    ws_manager: WsConnectionManager,
    inbox_repo: InboxRepository,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        ws_manager: WsConnectionManager,
    ) -> Self {
        Self {
            inbox_repo: InboxRepository::new(db.clone()),
            db,
            redis,
            ws_manager,
        }
    }

    /// 启动事件处理循环
    pub async fn start(mut self, mut receiver: mpsc::UnboundedReceiver<AppEvent>) {
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
                let item = InboxItem::new(
                    *to_user_id,
                    "notification".to_string(),
                    "friend_request".to_string(),
                    serde_json::json!({
                        "from_user_id": from_user_id,
                        "message": "您收到了一个好友请求"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::FriendRequestAccepted { from_user_id, to_user_id } => {
                let item = InboxItem::new(
                    *from_user_id,
                    "notification".to_string(),
                    "friend_request_accepted".to_string(),
                    serde_json::json!({
                        "user_id": to_user_id,
                        "message": "您的好友请求已被接受"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::GroupInviteSent { group_id, to_user_id, from_user_id } => {
                let item = InboxItem::new(
                    *to_user_id,
                    "notification".to_string(),
                    "group_invite".to_string(),
                    serde_json::json!({
                        "group_id": group_id,
                        "from_user_id": from_user_id,
                        "message": "您收到了一个群组邀请"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::MomentLiked { moment_id, user_id, liker_id } => {
                let item = InboxItem::new(
                    *user_id,
                    "notification".to_string(),
                    "moment_liked".to_string(),
                    serde_json::json!({
                        "moment_id": moment_id,
                        "liker_id": liker_id,
                        "message": "有人点赞了您的瞬间"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::MomentCommented { moment_id, user_id, commenter_id, .. } => {
                let item = InboxItem::new(
                    *user_id,
                    "notification".to_string(),
                    "moment_commented".to_string(),
                    serde_json::json!({
                        "moment_id": moment_id,
                        "commenter_id": commenter_id,
                        "message": "有人评论了您的瞬间"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::CommentLiked { comment_id, user_id, liker_id } => {
                let item = InboxItem::new(
                    *user_id,
                    "notification".to_string(),
                    "comment_liked".to_string(),
                    serde_json::json!({
                        "comment_id": comment_id,
                        "liker_id": liker_id,
                        "message": "有人点赞了您的评论"
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
            }
            AppEvent::SystemNotification { user_id, title, content } => {
                let item = InboxItem::new(
                    *user_id,
                    "system".to_string(),
                    "system_notification".to_string(),
                    serde_json::json!({
                        "title": title,
                        "content": content
                    }),
                );
                self.inbox_repo.create_inbox_item(&item).await?;
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
            AppEvent::MessageSent { sender_id, conversation_id, conversation_type, .. } => {
                let message = WsMessage::NewMessage {
                    sender_id: *sender_id,
                    conversation_id: *conversation_id,
                    conversation_type: conversation_type.clone(),
                };
                
                // 根据会话类型推送给相关用户
                if conversation_type == "private" {
                    // 私聊：推送给对方
                    // 这里需要查询会话的另一方用户ID
                    // 简化实现：推送给所有订阅了private_message的用户
                    self.ws_manager.broadcast_to_channel("private_message", message).await;
                } else {
                    // 群聊：推送给群组成员
                    self.ws_manager.broadcast_to_channel("group_message", message).await;
                }
            }
            AppEvent::FriendRequestSent { to_user_id, from_user_id } => {
                let message = WsMessage::FriendRequest {
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                self.ws_manager.send_to_user(*to_user_id, message).await;
            }
            AppEvent::FriendRequestAccepted { from_user_id, to_user_id } => {
                let message = WsMessage::FriendRequestAccepted {
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                self.ws_manager.send_to_user(*from_user_id, message).await;
            }
            AppEvent::GroupInviteSent { group_id, to_user_id, from_user_id } => {
                let message = WsMessage::GroupInvite {
                    group_id: *group_id,
                    from_user_id: *from_user_id,
                    to_user_id: *to_user_id,
                };
                self.ws_manager.send_to_user(*to_user_id, message).await;
            }
            AppEvent::MomentCreated { moment_id, user_id } => {
                let message = WsMessage::NewMoment {
                    moment_id: *moment_id,
                    user_id: *user_id,
                };
                self.ws_manager.broadcast_to_channel("moment", message).await;
            }
            AppEvent::MomentCommented { moment_id, comment_id, user_id, .. } => {
                let message = WsMessage::MomentComment {
                    moment_id: *moment_id,
                    comment_id: *comment_id,
                    user_id: *user_id,
                };
                self.ws_manager.send_to_user(*user_id, message).await;
            }
            AppEvent::SystemNotification { user_id, title, content } => {
                let message = WsMessage::SystemNotification {
                    title: title.clone(),
                    content: content.clone(),
                };
                self.ws_manager.send_to_user(*user_id, message).await;
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

