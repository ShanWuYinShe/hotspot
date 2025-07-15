use std::sync::Arc;
use chrono::{DateTime, Utc};
use uuid::Uuid;
use tracing::{error, info, warn};

use crate::ws::message_router::MessageRouter;
// Ensure all necessary types are correctly imported from message_types
use crate::ws::message_types::{    ServerMessage, MessageType as WsMessageType, 
    GroupEventData, 
    SystemNotificationData, SystemNotificationType, // Import SystemNotificationData and SystemNotificationType directly
    MessageNotification, RoomId // Import MessageNotification and RoomId
};
// Assuming NotificationResult and NotificationError are defined in notification/mod.rs
use crate::services::notification::{NotificationResult, NotificationError, NotificationService}; 
use crate::services::notification::policy::{NotificationPolicyService, NotificationPolicy};
use crate::repositories::PgUserRepository;
use crate::cache::NotificationCache;

// pub use crate::models::api::user::UserInfo; // Not directly used here, can be removed if not needed for future extensions

// Use the RoomType from ws::message_types directly
pub use crate::ws::message_types::RoomType;

use futures::FutureExt; // For .map() and .boxed()

#[derive(Clone)]
pub struct WebSocketNotificationService {
    message_router: Arc<MessageRouter>,
    policy_service: NotificationPolicyService,
    notification_cache: Option<Arc<dyn NotificationCache>>,
}

// Remove duplicate SystemNotificationData definition
/*
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemNotificationData {
    pub notification_id: Uuid,
    pub notification_type: SystemNotificationType,
    pub title: String,
    pub content: String,
    pub data: Option<serde_json::Value>,
    pub created_at: DateTime<Utc>,
}
*/

// Import SystemNotificationType from message_types - REMOVED: Imported in {} block
// pub use crate::ws::message_types::SystemNotificationType;


impl WebSocketNotificationService {    pub fn new(
        message_router: Arc<MessageRouter>,
        user_repository: Arc<PgUserRepository>,
        notification_policy: Option<NotificationPolicy>,
    ) -> Self {
        let policy_service = NotificationPolicyService::new(user_repository, notification_policy, None);
        Self { 
            message_router,
            policy_service,
            notification_cache: None,
        }
    }

    /// 创建带缓存的 WebSocket 通知服务
    pub fn with_cache(
        message_router: Arc<MessageRouter>,
        user_repository: Arc<PgUserRepository>,
        notification_policy: Option<NotificationPolicy>,
        notification_cache: Arc<dyn NotificationCache>,
    ) -> Self {
        let policy_service = NotificationPolicyService::new(user_repository, notification_policy, None);
        Self { 
            message_router,
            policy_service,
            notification_cache: Some(notification_cache),
        }
    }/// 更新通知策略
    #[allow(dead_code)]
    pub fn update_notification_policy(&mut self, policy: NotificationPolicy) {
        self.policy_service.update_policy(policy);
    }

    /// 获取当前通知策略
    #[allow(dead_code)]
    pub fn get_notification_policy(&self) -> &NotificationPolicy {
        self.policy_service.get_policy()
    }

    // Helper to get room ID from ServerMessage within the service
    fn get_room_id_helper(&self, message: &ServerMessage) -> Option<Uuid> {
        match message {
            ServerMessage::NewMessage(msg) => Some(msg.room_id),
            ServerMessage::UserStartedTyping { room_id, .. } => Some(*room_id),
            ServerMessage::UserStoppedTyping { room_id, .. } => Some(*room_id),
            ServerMessage::MessageRead { room_id, .. } => Some(*room_id),
            ServerMessage::GroupEvent(event) => Some(event.group_id),
            ServerMessage::MessageDeleted { room_id, .. } => Some(*room_id),
            // ServerMessage variants that do not have a room_id:
            ServerMessage::Pong
            | ServerMessage::Connected { .. }
            | ServerMessage::Error { .. }
            | ServerMessage::Success { .. }            | ServerMessage::SystemNotification { .. } => None,
        }
    }

    pub async fn notify_new_message(
        &self,
        message_id: Uuid,
        room_id: Uuid,
        room_type: RoomType,
        sender_id: i64,
        sender_nickname: String,
        content: String,
        message_type: String,
        created_at: DateTime<Utc>,
        location: Option<(f64, f64, Option<String>)>,    ) -> NotificationResult<()> {
        let ws_message_type = WsMessageType::from_string_or_default(&message_type);

        let (latitude, longitude, _location_name) = match location {
            Some((lat, lon, name)) => (Some(lat), Some(lon), name),
            None => (None, None, None),
        };

        let message_data = MessageNotification {
            message_id,
            room_id,
            room_type: room_type.clone(),
            sender_id,
            sender_nickname,
            content,
            message_type: ws_message_type,
            created_at,
            latitude,
            longitude,
            // location_name, // Removed as per MessageNotification definition
        };

        info!(
            "Attempting to send new message notification: msg_id={}, room_id={}, room_type={:?}",
            message_id, room_id, room_type
        );

        // 所有消息直接发送到相应房间，前端控制通知显示
        self.message_router
            .send_to_room(
                &RoomId::from_room_type(room_id, room_type.clone()),
                ServerMessage::NewMessage(message_data)
            )
            .await
            .map_err(|e| {
                error!("Error sending new message notification: {:?}", e);
                NotificationError::WebSocket(e.to_string())
            })
    }

    #[allow(dead_code)]
    pub async fn notify_user_typing(
        &self,
        room_id: Uuid,
        room_type: RoomType,
        user_id: i64,
        user_nickname: String,
        is_typing: bool,
    ) -> NotificationResult<()> {
        let typing_event = if is_typing {
            ServerMessage::UserStartedTyping { room_id: room_id.into(), user_id, nickname: user_nickname }
        } else {
            ServerMessage::UserStoppedTyping { room_id: room_id.into(), user_id }
        };
        
        // Assuming MessageRouter::send_to_room takes RoomId and ServerMessage
        self.message_router
            .send_to_room(&RoomId::from_room_type(room_id, room_type), typing_event)
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))
    }    pub async fn notify_message_read_to_recipient(
        &self,
        reader_id: i64,
        conversation_id: Uuid,
        last_read_message_id: Option<Uuid>,
        _read_at: DateTime<Utc>,
    ) -> NotificationResult<()> {
        let read_receipt = ServerMessage::MessageRead {
            room_id: conversation_id,
            user_id: reader_id, // Assuming user_id maps to reader_id in MessageRead
            message_id: last_read_message_id,
        };
            
        // This notification is intended for the *other* participant(s) in the conversation.
        // We might need a specific MessageRouter method for sending to a room excluding one user.
        // For now, sending to the room and the router should handle exclusion if needed.
        let room_key = RoomId::from_conversation(conversation_id);
        self.message_router
            .send_to_room(&room_key, read_receipt)
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))
    }    pub async fn notify_group_event(
        &self,
        event_data: GroupEventData,
        actor_id_to_exclude: Option<i64>,
    ) -> NotificationResult<()> {
        info!(
            "Attempting to notify group event: {:?}, exclude actor: {:?}", 
            event_data, actor_id_to_exclude
        );
        let message = ServerMessage::GroupEvent(event_data);
        let room_key = RoomId::from_group(self.get_room_id_helper(&message).expect("GroupEvent should have a room_id"));
        
        match actor_id_to_exclude {
            Some(actor_id) => {
                // Send to room excluding the actor
                self.message_router.send_to_room_except(&room_key, message, actor_id)
                    .await
                    .map_err(|e| NotificationError::WebSocket(e.to_string()))
            }
            None => {
                // Send to entire room
                self.message_router.send_to_room(&room_key, message)
                    .await
                    .map_err(|e| NotificationError::WebSocket(e.to_string()))
            }
        }
    }

    /// Sends a system event notification to a specific user.
    pub async fn notify_system_event_to_user(
        &self,
        user_id: i64,
        event_data: SystemNotificationData,
    ) -> NotificationResult<()> {
        info!(
            "Attempting to send system event to user {}: {:?}", 
            user_id, event_data
        );

        let message = ServerMessage::SystemNotification(event_data);
        let user_room = RoomId::from_user(user_id);
        // 直接发送系统通知，不进行过滤
        self.message_router.send_to_room(&user_room, message)
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))    }

    #[allow(dead_code)]
    pub async fn send_direct_success_confirmation_to_connection(
        &self,
        connection_id: Uuid,
        _action: String,
        _request_id: Option<String>,
        _data: Option<serde_json::Value>,
    ) -> NotificationResult<()> {
        // Based on ServerMessage::Success definition: Success { message: String }
        let confirmation = ServerMessage::Success {
            message: "操作成功".to_string(), // Assuming Success takes a message string
        };
        self.message_router
            .send_to_connection(connection_id, confirmation)
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))
    }
    #[allow(dead_code)]
    pub async fn send_direct_error_message_to_connection(
        &self,
        connection_id: Uuid,
        _action: String,
        _request_id: Option<String>,
        error_message: String,
        error_code: Option<i32>,
    ) -> NotificationResult<()> {
        // Based on ServerMessage::Error definition: Error { code: String, message: String }
        let error_msg = ServerMessage::Error {
            code: error_code.unwrap_or(0).to_string(), // Convert error_code to String
            message: error_message,
        };
        self.message_router
            .send_to_connection(connection_id, error_msg)
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))
    }
    
    pub async fn notify_system_event(
        &self,
        user_id: i64,
        notification_data: SystemNotificationData,
    ) -> NotificationResult<()> {
        let user_room = RoomId::from_user(user_id);
        self.message_router.send_to_room(&user_room, ServerMessage::SystemNotification(notification_data))
            .await
            .map_err(|e| NotificationError::WebSocket(e.to_string()))
    }

    pub async fn notify_message_deleted(        &self,
        message_id: Uuid,
        room_id: Uuid,
        room_type: RoomType,
        _actor_id: Option<i64>,
    ) -> NotificationResult<()> {
        let delete_event = ServerMessage::MessageDeleted {
            message_id,
            room_id,
            room_type: room_type.clone(),
        };
        info!(
            "Attempting to send message deleted notification: msg_id={}, room_id={}, room_type={:?}",
            message_id, room_id, room_type
        );
        self.message_router
            .send_to_room(&RoomId::from_room_type(room_id, room_type), delete_event)
            .await
            .map_err(|e| {
                error!("Error sending message deleted notification: {:?}", e);
                NotificationError::WebSocket(e.to_string())
            })
    }

    /// 获取用户的离线通知
    pub async fn get_offline_notifications(&self, user_id: i64) -> NotificationResult<Vec<String>> {
        match &self.notification_cache {
            Some(cache) => {
                info!("获取用户 {} 的离线通知", user_id);
                cache.get_offline_notifications(user_id)
                    .await
                    .map_err(|e| {
                        error!("获取离线通知失败: {:?}", e);
                        NotificationError::Other(format!("缓存获取失败: {}", e))
                    })
            }
            None => {
                warn!("通知缓存未配置，无法获取离线通知 for user_id: {}", user_id);
                Ok(vec![])
            }
        }
    }

    /// 清除用户的离线通知（用户上线后调用）
    #[allow(dead_code)]    pub async fn clear_offline_notifications(&self, user_id: i64) -> NotificationResult<()> {
        match &self.notification_cache {
            Some(cache) => {
                info!("清除用户 {} 的离线通知", user_id);
                cache.clear_offline_notifications(user_id).await?;
                Ok(())
            }
            None => {
                warn!("通知缓存未配置，无需清除离线通知 for user_id: {}", user_id);
                Ok(())
            }
        }
    }

    /// 发送通知，不进行设置检查
    pub async fn send_notification_with_settings(
        &self,
        user_id: i64,
        event_data: SystemNotificationData,
    ) -> NotificationResult<()> {
        // 直接发送通知，不检查设置
        self.notify_system_event_to_user(user_id, event_data).await
    }

    // ...existing code...
}

// Ensure WsMessageType has from_string_or_default or similar
// This should ideally live in ws/message_types.rs with WsMessageType definition
// For now, adding it here to make the service compile.
// Consider moving this to the WsMessageType impl block in its original file.
impl WsMessageType {
    pub fn from_string_or_default(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "text" => WsMessageType::Text,
            "image" => WsMessageType::Image,
            "voice" => WsMessageType::Voice,
            "video" => WsMessageType::Video,
            "file" => WsMessageType::File,
            "location" => WsMessageType::Location,
            _ => WsMessageType::Text, // Default to Text if unknown
        }
    }
}

impl NotificationService for WebSocketNotificationService {
    fn name(&self) -> &str {
        "WebSocketNotificationService"
    }

    fn send_notification(
        &self, 
        user_id: i64,
        notification: &serde_json::Value
    ) -> impl std::future::Future<Output = NotificationResult<bool>> + Send {
        // This implementation is for the NotificationService trait, not the internal async fn
        // It should parse the notification value and call the appropriate internal method.
        // For simplicity, assuming the notification value is a SystemNotificationData.
        let event_data: SystemNotificationData = serde_json::from_value(notification.clone()).unwrap_or_else(|_| {
            error!("Failed to parse notification value to SystemNotificationData in send_notification");
            SystemNotificationData { 
                notification_id: uuid::Uuid::new_v4(), 
                notification_type: SystemNotificationType::System, // Use imported type
                title: "解析错误".to_string(), 
                content: "无法解析通知数据".to_string(), 
                data: None, 
                created_at: Utc::now() 
            }
        });
        
        // Call the internal async function and map its result
        self.notify_system_event_to_user(user_id, event_data).map(|result| result.map(|_| true)).boxed()
    }      fn store_offline_notification(
        &self, 
        user_id: i64,
        notification: &serde_json::Value,
        expire_days: u32
    ) -> impl std::future::Future<Output = NotificationResult<()>> + Send {
        let notification_cache = self.notification_cache.clone();
        let notification_clone = notification.clone();
        
        async move {
            match notification_cache {
                Some(cache) => {
                    info!("存储离线通知 for user_id: {}, expire_days: {}", user_id, expire_days);
                    cache.store_offline_notification(user_id, &notification_clone, expire_days)
                        .await
                        .map_err(|e| {
                            error!("存储离线通知失败: {:?}", e);
                            NotificationError::Other(format!("缓存存储失败: {}", e))
                        })
                }
                None => {
                    warn!("通知缓存未配置，跳过离线通知存储 for user_id: {}", user_id);
                    Ok(())
                }
            }
        }.boxed()
    }
      fn send_group_digest(
        &self,
        user_id: i64,
        user_nickname: &str,
        group_id: uuid::Uuid,
        group_name: &str,
        digest_content: &str
    ) -> impl std::future::Future<Output = NotificationResult<bool>> + Send {
        // Implement group digest notification as a system notification
        let notification_data = SystemNotificationData {
            notification_id: Uuid::new_v4(),
            notification_type: SystemNotificationType::GroupDigest,
            title: format!("群组摘要 - {}", group_name),
            content: digest_content.to_string(),
            data: Some(serde_json::json!({
                "group_id": group_id,
                "group_name": group_name,
                "sender_nickname": user_nickname
            })),
            created_at: Utc::now(),
        };
        
        // Send as system notification to the specific user
        self.notify_system_event_to_user(user_id, notification_data)
            .map(|result| result.map(|_| true))
            .boxed()
    }
} 

// Helper to create RoomId from Uuid and RoomType
impl RoomId {
    fn from_room_type(id: Uuid, room_type: RoomType) -> Self {
        match room_type {
            RoomType::Group => RoomId::Group(id),
            RoomType::Conversation => RoomId::Conversation(id),
        }
    }
}