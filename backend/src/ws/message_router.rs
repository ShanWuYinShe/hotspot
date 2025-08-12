//! WebSocket消息路由器
//! 
//! 专门负责消息的路由和分发

use std::sync::Arc;
use futures::stream::{FuturesUnordered, StreamExt};
use tracing::{debug, warn};

use crate::ws::{
    message_types::{ServerMessage, RoomId},
    connection_manager::ConnectionManager,
    room_manager::RoomManager,
    errors::WsResult,
};

/// 消息路由器
/// 
/// 职责：
/// - 根据目标类型路由消息
/// - 处理消息分发逻辑
/// - 提供统一的消息发送接口
#[derive(Debug)]
pub struct MessageRouter {
    connection_manager: Arc<ConnectionManager>,
    room_manager: Arc<RoomManager>,
}

impl MessageRouter {
    /// 创建新的消息路由器
    pub fn new(
        connection_manager: Arc<ConnectionManager>,
        room_manager: Arc<RoomManager>,
    ) -> Self {
        Self {
            connection_manager,
            room_manager,
        }
    }

    /// 向特定连接发送消息
    pub async fn send_to_connection(&self, connection_id: uuid::Uuid, message: ServerMessage) -> WsResult<()> {
        self.connection_manager.send_to_connection(connection_id, message).await
    }

    /// 向用户的所有连接发送消息
    #[allow(dead_code)]
    pub async fn send_to_user(&self, user_id: i64, message: ServerMessage) -> WsResult<()> {
        debug!("向用户 {} 发送消息: {:?}", user_id, message);
        self.connection_manager.send_to_user(user_id, message).await
    }

    /// 向房间中的所有连接发送消息
    pub async fn send_to_room(&self, room_id: &RoomId, message: ServerMessage) -> WsResult<()> {
        debug!("向房间 {:?} 发送消息: {:?}", room_id, message);
        
        let subscribers = self.room_manager.get_room_subscribers(room_id).await;
        if subscribers.is_empty() {
            debug!("房间 {:?} 没有订阅者", room_id);
            return Ok(());
        }

        let mut tasks = FuturesUnordered::new();
        for connection_id in subscribers {
            let msg = message.clone();
            let cm = self.connection_manager.clone();
            tasks.push(async move { (connection_id, cm.send_to_connection(connection_id, msg).await) });
        }
        let mut errors = Vec::new();
        while let Some((connection_id, res)) = tasks.next().await {
            if let Err(e) = res { errors.push((connection_id, e)); }
        }

        if !errors.is_empty() {
            warn!("向房间 {:?} 发送消息时部分连接失败: {:?}", room_id, errors);
        }

        Ok(())
    }

    // 全局广播已在项目规范中禁用，如需恢复请在策略层精确选择目标用户

    /// 发送消息给房间内除指定用户外的所有用户
    pub async fn send_to_room_except(&self, room_id: &RoomId, message: ServerMessage, except_user_id: i64) -> WsResult<()> {
        debug!("向房间 {:?} 发送消息，排除用户 {}: {:?}", room_id, except_user_id, message);
        
        let subscribers = self.room_manager.get_room_subscribers(room_id).await;
        if subscribers.is_empty() {
            debug!("房间 {:?} 没有订阅者", room_id);
            return Ok(());
        }

        // 获取要排除的用户的连接
        let except_connections = self.connection_manager.get_user_connections(except_user_id).await;

        let mut tasks = FuturesUnordered::new();
        for connection_id in subscribers {
            if except_connections.contains(&connection_id) { continue; }
            let msg = message.clone();
            let cm = self.connection_manager.clone();
            tasks.push(async move { (connection_id, cm.send_to_connection(connection_id, msg).await) });
        }
        let mut errors = Vec::new();
        while let Some((connection_id, res)) = tasks.next().await {
            if let Err(e) = res { errors.push((connection_id, e)); }
        }

        if !errors.is_empty() {
            warn!("向房间 {:?} 发送消息时部分连接失败: {:?}", room_id, errors);
        }

        Ok(())
    }

    /// 发送消息给多个用户
    #[allow(dead_code)]
    pub async fn send_to_users(&self, user_ids: &[i64], message: ServerMessage) -> WsResult<()> {
        debug!("向 {} 个用户发送消息: {:?}", user_ids.len(), message);

        let mut errors = Vec::new();
        for &user_id in user_ids {
            if let Err(e) = self.connection_manager.send_to_user(user_id, message.clone()).await {
                errors.push((user_id, e));
            }
        }

        if !errors.is_empty() {
            warn!("向多个用户发送消息时部分用户发送失败: {:?}", errors);
        }

        Ok(())
    }
}
