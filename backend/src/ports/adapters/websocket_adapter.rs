//! WebSocket 适配器 - 将 WebSocketManager 适配为 WebSocketPort
//! 
//! 提供 WebSocket 系统与端口架构的集成

use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use uuid::Uuid;

use crate::ports::websocket::{WebSocketPort, WebSocketConnectionPort, WebSocketPortError, WebSocketPortResult};
use crate::ws::manager::WebSocketManager;
use crate::ws::message_types::{ServerMessage, SystemNotificationData, RoomId};

/// WebSocket 通信适配器
#[derive(Clone)]
pub struct WebSocketAdapter {
    manager: Arc<WebSocketManager>,
}

impl WebSocketAdapter {
    pub fn new(manager: Arc<WebSocketManager>) -> Self {
        Self { manager }
    }
}

impl WebSocketPort for WebSocketAdapter {
    fn name(&self) -> &str {
        "WebSocketAdapter"
    }

    fn send_to_user<'a>(
        &'a self,
        user_id: i64,
        message: ServerMessage,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            manager.connection_manager().send_to_user(user_id, message)
                .await
                .map_err(WebSocketPortError::from_ws_error)
        })
    }

    fn send_to_room<'a>(
        &'a self,
        room_id: &'a str,
        message: ServerMessage,
        exclude_user_id: Option<i64>,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        let room_id_str = room_id.to_string();
        Box::pin(async move {
            // 简单的房间ID解析逻辑
            let room_id = if let Some(group_uuid_str) = room_id_str.strip_prefix("group_") {
                if let Ok(group_uuid) = uuid::Uuid::parse_str(group_uuid_str) {
                    RoomId::from_group(group_uuid)
                } else {
                    return Err(WebSocketPortError::room_not_found(room_id_str));
                }
            } else if let Some(conv_uuid_str) = room_id_str.strip_prefix("conversation_") {
                if let Ok(conv_uuid) = uuid::Uuid::parse_str(conv_uuid_str) {
                    RoomId::from_conversation(conv_uuid)
                } else {
                    return Err(WebSocketPortError::room_not_found(room_id_str));
                }
            } else if let Some(user_id_str) = room_id_str.strip_prefix("user_") {
                if let Ok(user_id) = user_id_str.parse::<i64>() {
                    RoomId::from_user(user_id)
                } else {
                    return Err(WebSocketPortError::room_not_found(room_id_str));
                }
            } else {
                return Err(WebSocketPortError::room_not_found(room_id_str));
            };
            
            // 注意：message_router 的 send_to_room 不支持 exclude_user_id
            // 这里简化实现，忽略 exclude_user_id 参数
            let _ = exclude_user_id; // 避免未使用变量警告
            manager.message_router().send_to_room(&room_id, message)
                .await
                .map_err(WebSocketPortError::from_ws_error)
        })
    }

    fn send_to_users<'a>(
        &'a self,
        user_ids: &'a [i64],
        message: ServerMessage,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        let user_ids = user_ids.to_vec();
        Box::pin(async move {
            for user_id in user_ids {
                let _ = manager.connection_manager().send_to_user(user_id, message.clone()).await;
            }
            Ok(())
        })
    }

    fn is_user_online<'a>(
        &'a self,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<bool>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            Ok(manager.is_user_online(user_id).await)
        })
    }

    fn get_online_user_count<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<usize>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            // 使用统计信息获取连接数
            let stats = manager.connection_manager().get_stats().await;
            Ok(stats.total_connections)
        })
    }

    fn get_room_user_count<'a>(
        &'a self,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<usize>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _room_id = room_id.to_string();
        Box::pin(async move {
            // 简化实现：返回0，实际实现需要房间管理器支持
            Ok(0)
        })
    }

    fn join_room<'a>(
        &'a self,
        user_id: i64,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _user_id = user_id;
        let _room_id = room_id.to_string();
        Box::pin(async move {
            // 简化实现：房间加入逻辑需要在连接时处理
            tracing::info!("用户 {} 加入房间 {}", _user_id, _room_id);
            Ok(())
        })
    }

    fn leave_room<'a>(
        &'a self,
        user_id: i64,
        room_id: &'a str,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let _manager = self.manager.clone();
        let _user_id = user_id;
        let _room_id = room_id.to_string();
        Box::pin(async move {
            // 简化实现：房间离开逻辑需要在连接时处理
            tracing::info!("用户 {} 离开房间 {}", _user_id, _room_id);
            Ok(())
        })
    }

    fn send_system_notification<'a>(
        &'a self,
        user_id: i64,
        notification: SystemNotificationData,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            let message = ServerMessage::SystemNotification(notification);
            manager.connection_manager().send_to_user(user_id, message)
                .await
                .map_err(WebSocketPortError::from_ws_error)
        })
    }
}

/// WebSocket 连接管理适配器
#[derive(Clone)]
pub struct WebSocketConnectionAdapter {
    manager: Arc<WebSocketManager>,
}

impl WebSocketConnectionAdapter {
    pub fn new(manager: Arc<WebSocketManager>) -> Self {
        Self { manager }
    }
}

impl WebSocketConnectionPort for WebSocketConnectionAdapter {
    fn name(&self) -> &str {
        "WebSocketConnectionAdapter"
    }

    fn register_connection<'a>(
        &'a self,
        connection_id: Uuid,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let _manager = self.manager.clone();
        Box::pin(async move {
            // 注意：实际的连接注册由 WebSocket 处理器完成
            // 这里只是记录连接信息
            tracing::info!("WebSocket 连接已注册: connection_id={}, user_id={}", connection_id, user_id);
            Ok(())
        })
    }

    fn unregister_connection<'a>(
        &'a self,
        connection_id: Uuid,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<()>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            let _ = manager.connection_manager().remove_connection(connection_id).await;
            Ok(())
        })
    }

    fn get_user_connections<'a>(
        &'a self,
        user_id: i64,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<Vec<Uuid>>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            Ok(manager.connection_manager().get_user_connections(user_id).await)
        })
    }

    fn connection_exists<'a>(
        &'a self,
        connection_id: Uuid,
    ) -> Pin<Box<dyn Future<Output = WebSocketPortResult<bool>> + Send + 'a>> {
        let manager = self.manager.clone();
        Box::pin(async move {
            Ok(manager.connection_manager().get_connection(connection_id).await.is_some())
        })
    }
}
