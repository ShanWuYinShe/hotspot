#![allow(async_fn_in_trait)]
#[cfg(any(feature = "audit", feature = "ws_hooks", feature = "jobs"))]
use std::sync::{Arc, RwLock};
use futures::stream::{FuturesUnordered, StreamExt};
use chrono::{DateTime, Utc};
use crate::utils::types::{UserId, GroupId, ConversationId, MessageId, RoomKey};
// use crate::ws::message_types::RoomType;

/// 领域事件定义
#[derive(Debug, Clone)]
pub enum DomainEvent {
    UserRegistered { user_id: UserId },
    TempUserCreated { user_id: UserId },
    MessageSent {
        room: RoomKey,
        message_id: MessageId,
        sender_id: UserId,
        sender_nickname: String,
        content: String,
        message_type: String,
        created_at: DateTime<Utc>,
        latitude: Option<f64>,
        longitude: Option<f64>,
    },
    MessageDeleted { room: RoomKey, message_id: MessageId, actor_id: Option<UserId> },
    DirectMessagesRead { conversation_id: ConversationId, reader_id: UserId, message_id: Option<MessageId> },
    GroupMessagesRead { group_id: GroupId, reader_id: UserId, message_id: Option<MessageId> },
    GroupJoined { group_id: GroupId, user_id: UserId },
    GroupLeft { group_id: GroupId, user_id: UserId },
    GroupKicked { group_id: GroupId, actor_id: UserId, target_id: UserId },
    GroupOwnershipTransferred { group_id: GroupId, from_user_id: UserId, to_user_user_id: UserId },
}

/// 事件总线（支持多处理器并发）
pub struct EventBus {
    #[cfg(feature = "audit")]
    audit: RwLock<Option<Arc<audit_handler::AuditEventsHandler>>>,
    #[cfg(feature = "ws_hooks")]
    ws: RwLock<Option<Arc<ws_handler::WsEventsHandler>>>,
    #[cfg(feature = "jobs")]
    jobs: RwLock<Option<Arc<jobs_handler::JobsEventsHandler>>>,
}

impl EventBus {
    pub fn new() -> Self { Self { 
        #[cfg(feature = "audit")]
        audit: RwLock::new(None),
        #[cfg(feature = "ws_hooks")]
        ws: RwLock::new(None),
        #[cfg(feature = "jobs")]
        jobs: RwLock::new(None),
    } }
}

impl Default for EventBus {
    fn default() -> Self {
        Self::new()
    }
}

impl EventBus {
    #[cfg(feature = "audit")]
    pub fn register_audit_handler(&self, handler: Arc<audit_handler::AuditEventsHandler>) {
        *self.audit.write().unwrap() = Some(handler);
    }

    #[cfg(feature = "ws_hooks")]
    pub fn register_ws_handler(&self, handler: Arc<ws_handler::WsEventsHandler>) {
        *self.ws.write().unwrap() = Some(handler);
    }

    #[cfg(feature = "jobs")]
    pub fn register_jobs_handler(&self, handler: Arc<jobs_handler::JobsEventsHandler>) {
        *self.jobs.write().unwrap() = Some(handler);
    }

    pub async fn publish(&self, event: DomainEvent) {
        // 避免在无处理器特性时的未使用参数警告
        let _ = &event;
        let mut tasks: FuturesUnordered<futures::future::BoxFuture<'static, ()>> = FuturesUnordered::new();
        #[cfg(feature = "audit")]
        if let Some(h) = self.audit.read().unwrap().clone() {
            let ev = event.clone();
            tasks.push(Box::pin(async move { h.handle(&ev).await; }));
        }
        #[cfg(feature = "ws_hooks")]
        if let Some(h) = self.ws.read().unwrap().clone() {
            let ev = event.clone();
            tasks.push(Box::pin(async move { h.handle(&ev).await; }));
        }
        #[cfg(feature = "jobs")]
        if let Some(h) = self.jobs.read().unwrap().clone() {
            let ev = event.clone();
            tasks.push(Box::pin(async move { h.handle(&ev).await; }));
        }
        while tasks.next().await.is_some() {}
    }
}

/// 审计事件处理器（特性开启时启用）
#[cfg(feature = "audit")]
pub mod audit_handler {
    use super::DomainEvent;
    use crate::utils::audit::{AuditEvent, AuditService, NoopAuditSink};
    use std::sync::Arc;
    use uuid::Uuid;

    #[derive(Clone)]
    pub struct AuditEventsHandler {
        audit: Arc<AuditService<NoopAuditSink>>, // 默认 Noop，可在未来替换为具体实现
    }

    impl AuditEventsHandler {
        pub fn new(audit: Arc<AuditService<NoopAuditSink>>) -> Self { Self { audit } }
        pub fn name(&self) -> &'static str { "audit_events" }
    }

    impl AuditEventsHandler {
        pub async fn handle(&self, event: &DomainEvent) {
            let (category, action, actor, target, meta) = match event.clone() {
                DomainEvent::UserRegistered { user_id } => ("user", "registered", Some(user_id.into()), None, None),
                DomainEvent::TempUserCreated { user_id } => ("user", "temp_created", Some(user_id.into()), None, None),
                DomainEvent::MessageSent { room, message_id, sender_id, .. } => (
                    "message", "sent", Some(sender_id.into()), Some(room.to_string()), Some(serde_json::json!({"message_id": Uuid::from(message_id)}))
                ),
                DomainEvent::MessageDeleted { room, message_id, actor_id } => (
                    "message", "deleted", actor_id.map(Into::into), Some(room.to_string()), Some(serde_json::json!({"message_id": Uuid::from(message_id)}))
                ),
                DomainEvent::GroupJoined { group_id, user_id } => (
                    "group", "joined", Some(user_id.into()), Some(group_id.to_string()), None
                ),
                DomainEvent::GroupLeft { group_id, user_id } => (
                    "group", "left", Some(user_id.into()), Some(group_id.to_string()), None
                ),
                DomainEvent::GroupKicked { group_id, actor_id, target_id } => (
                    "group", "kicked", Some(actor_id.into()), Some(group_id.to_string()), Some(serde_json::json!({"target_id": Into::<i64>::into(target_id)}))
                ),
                DomainEvent::GroupOwnershipTransferred { group_id, from_user_id, to_user_user_id } => (
                    "group", "ownership_transferred", Some(from_user_id.into()), Some(group_id.to_string()), Some(serde_json::json!({"to_user_id": Into::<i64>::into(to_user_user_id)}))
                ),
                _ => ("misc", "other", None, None, None),
            };
            let mut ev = AuditEvent::now(category, action);
            if let Some(a) = actor { ev = ev.with_actor(a); }
            if let Some(t) = target { ev = ev.with_target(t); }
            if let Some(m) = meta { ev = ev.with_metadata(m); }
            self.audit.submit(ev).await;
        }
    }
}

/// WebSocket 事件处理器（特性开启时启用）
#[cfg(feature = "ws_hooks")]
pub mod ws_handler {
    use super::DomainEvent;
    use crate::ws::manager::WebSocketNotificationService;
    use crate::ws::message_types::{SystemNotificationData, SystemNotificationType};
    use chrono::Utc;
    use std::sync::Arc;

    #[derive(Clone)]
    pub struct WsEventsHandler {
        ws: Arc<WebSocketNotificationService>,
    }

    impl WsEventsHandler {
        pub fn new(ws: Arc<WebSocketNotificationService>) -> Self { Self { ws } }
        pub fn name(&self) -> &'static str { "ws_events" }

        pub async fn handle(&self, event: &DomainEvent) {
            match event.clone() {
                DomainEvent::UserRegistered { user_id } | DomainEvent::TempUserCreated { user_id } => {
                    let data = SystemNotificationData {
                        notification_id: uuid::Uuid::new_v4(),
                        notification_type: SystemNotificationType::System,
                        title: "欢迎使用 HotSpot".to_string(),
                        content: "账户就绪，您可以开始体验功能。".to_string(),
                        data: None,
                        created_at: Utc::now(),
                    };
                    let _ = self.ws.notify_system_event_to_user(user_id.into(), data).await;
                }
                DomainEvent::MessageSent { room, message_id: _, sender_id, sender_nickname, content, message_type, created_at, latitude, longitude } => {
                    let location = match (latitude, longitude) {
                        (Some(lat), Some(lon)) => Some((lat, lon, None)),
                        _ => None,
                    };
                                    let _ = self.ws
                    .notify_new_message(
                        room, sender_id, sender_nickname,
                        content, message_type, created_at, location,
                    )
                    .await;
                }
                DomainEvent::MessageDeleted { room, actor_id, .. } => {
                    let _ = self.ws.notify_message_deleted(room, actor_id).await;
                }
                DomainEvent::GroupJoined { group_id, user_id } => {
                    let data = crate::ws::message_types::GroupEventData {
                        group_id: group_id.into(),
                        event_type: crate::ws::message_types::GroupEventType::MemberJoined,
                        actor_id: user_id.0.into(),
                        actor_nickname: String::new(),
                        target_id: None,
                        target_nickname: None,
                        created_at: Utc::now(),
                    };
                    let _ = self.ws.notify_group_event(data, None).await;
                }
                DomainEvent::GroupLeft { group_id, user_id } => {
                    let data = crate::ws::message_types::GroupEventData {
                        group_id: group_id.into(),
                        event_type: crate::ws::message_types::GroupEventType::MemberLeft,
                        actor_id: user_id.0.into(),
                        actor_nickname: String::new(),
                        target_id: None,
                        target_nickname: None,
                        created_at: Utc::now(),
                    };
                    let _ = self.ws.notify_group_event(data, None).await;
                }
                DomainEvent::GroupKicked { group_id, actor_id, target_id } => {
                    let data = crate::ws::message_types::GroupEventData {
                        group_id: group_id.into(),
                        event_type: crate::ws::message_types::GroupEventType::MemberKicked,
                        actor_id: actor_id.0.into(),
                        actor_nickname: String::new(),
                        target_id: Some(target_id.0.into()),
                        target_nickname: None,
                        created_at: Utc::now(),
                    };
                    let _ = self.ws.notify_group_event(data, None).await;
                }
                DomainEvent::GroupOwnershipTransferred { group_id, from_user_id, to_user_user_id: to_user_id } => {
                    let data = crate::ws::message_types::GroupEventData {
                        group_id: group_id.into(),
                        event_type: crate::ws::message_types::GroupEventType::OwnerTransferred,
                        actor_id: from_user_id.0.into(),
                        actor_nickname: String::new(),
                        target_id: Some(to_user_id.0.into()),
                        target_nickname: None,
                        created_at: Utc::now(),
                    };
                    let _ = self.ws.notify_group_event(data, None).await;
                }
                DomainEvent::DirectMessagesRead { conversation_id, reader_id, message_id } => {
                    let _ = self.ws
                        .notify_message_read_to_recipient(
                            reader_id,
                            conversation_id,
                            message_id,
                        )
                        .await;
                }
                DomainEvent::GroupMessagesRead { group_id: _gid, reader_id: _reader, message_id: _mid } => {
                    // 预留：如需群组已读通知，可在此实现（当前消息模型未定义群已读 ServerMessage）
                }
            }
        }
    }
}

/// 作业队列事件处理器（特性开启时启用）
#[cfg(feature = "jobs")]
pub mod jobs_handler {
    use super::DomainEvent;
    use crate::ports::JobDispatcherPort;
    use std::sync::Arc;

    #[derive(Clone)]
    pub struct JobsEventsHandler {
        jobs: Arc<dyn JobDispatcherPort>,
    }

    impl JobsEventsHandler {
        pub fn new(jobs: Arc<dyn JobDispatcherPort>) -> Self { Self { jobs } }
        pub fn name(&self) -> &'static str { "jobs_events" }

        pub async fn handle(&self, event: &DomainEvent) {
            if let DomainEvent::TempUserCreated { user_id } = event.clone() {
                // 可派发欢迎邮件或初始化位置等任务（示例：欢迎邮件）
                let _ = self.jobs.dispatch_welcome_email(user_id.into(), "用户".to_string(), None).await;
            }
        }
    }
}

