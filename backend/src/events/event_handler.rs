use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{error, info};

use crate::{
    events::{
        event_bus::EventBus,
        event_types::AppEvent,
        handlers::{
            content_persist, content_ws, group_persist, group_ws, message_persist, message_ws,
            system_persist, system_ws, user_persist, user_ws,
        },
    },
    repositories::{
        group_member_repo::GroupMemberRepository, notification_repo::NotificationRepository,
        user_repo::UserRepository,
    },
    services::{
        media_service::MediaService, notification_service::NotificationService,
        subscription_service::SubscriptionService,
    },
    websocket::{
        manager::ConnectionManager as WsConnectionManager, subscription::SubscriptionManager,
    },
};

/// 事件处理器
///
/// 注：通知是实时推送机制，不存储。通过 WebSocket 推送给在线用户。
/// 冷启动通过各业务 API 获取待处理数据。
pub struct EventHandler {
    ws_manager: WsConnectionManager,
    ws_subscription: SubscriptionManager,
    user_repo: UserRepository,
    subscription_service: SubscriptionService,
    group_member_repo: GroupMemberRepository,
    notification_service: NotificationService,
}

impl EventHandler {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        ws_manager: WsConnectionManager,
        ws_subscription: SubscriptionManager,
        _event_bus: Arc<EventBus>,
        _media_service: MediaService,
    ) -> Self {
        let user_repo = UserRepository::new(db.clone());
        let subscription_service = SubscriptionService::new(db.clone(), redis.clone());
        let group_member_repo = GroupMemberRepository::new(db.clone(), redis.clone());
        let notification_repo = NotificationRepository::new(db.clone());
        let notification_service = NotificationService::new(notification_repo, user_repo.clone());

        Self {
            user_repo,
            ws_manager,
            ws_subscription,
            subscription_service,
            group_member_repo,
            notification_service,
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

        // 持久化通知 (Inbox)
        if let Err(e) = self.persist_notification(&event).await {
            error!("Failed to persist notification: {}", e);
            // 继续执行推送，不因为持久化失败阻断实时通知
        }

        // 发送WebSocket推送
        self.send_websocket_notifications(&event).await?;

        Ok(())
    }

    /// 持久化通知
    async fn persist_notification(
        &self,
        event: &AppEvent,
    ) -> Result<(), Box<dyn std::error::Error>> {
        user_persist(event, &self.notification_service).await?;
        group_persist(event, &self.notification_service).await?;
        message_persist(event).await?;
        content_persist(event, &self.notification_service).await?;
        system_persist(event, &self.notification_service, &self.user_repo).await?;
        Ok(())
    }

    /// 发送WebSocket推送
    async fn send_websocket_notifications(
        &self,
        event: &AppEvent,
    ) -> Result<(), Box<dyn std::error::Error>> {
        user_ws(
            event,
            &self.ws_manager,
            &self.ws_subscription,
            &self.user_repo,
            &self.subscription_service,
        )
        .await?;
        group_ws(
            event,
            &self.ws_manager,
            &self.ws_subscription,
            &self.user_repo,
            &self.group_member_repo,
            &self.subscription_service,
        )
        .await?;
        message_ws(
            event,
            &self.ws_manager,
            &self.ws_subscription,
            &self.user_repo,
        )
        .await?;
        content_ws(
            event,
            &self.ws_manager,
            &self.user_repo,
            &self.subscription_service,
        )
        .await?;
        system_ws(event, &self.ws_manager, &self.subscription_service).await?;
        Ok(())
    }
}
